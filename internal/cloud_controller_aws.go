package internal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"

	"github.com/spacelift-io/awsautoscalr/internal/ifaces"
)

// AWSCloudController implements CloudController for AWS.
type AWSCloudController struct {
	Autoscaling ifaces.Autoscaling
	EC2         ifaces.EC2
	SSM         *ssm.Client
	ASGName     string
	Tracer      Tracer
}

// NewAWSCloudController creates a new AWS cloud controller instance.
func NewAWSCloudController(ctx context.Context, region, asgARN string, tracer Tracer) (*AWSCloudController, error) {
	// Load AWS configuration
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}

	// Instrument with X-Ray
	awsv2.AWSV2Instrumentor(&awsConfig.APIOptions)

	// Parse ASG name from ARN
	arnParts := strings.Split(asgARN, "/")
	if len(arnParts) != 2 {
		return nil, fmt.Errorf("could not parse autoscaling group ARN")
	}
	asgName := arnParts[1]

	return &AWSCloudController{
		Autoscaling: autoscaling.NewFromConfig(awsConfig),
		EC2:         ec2.NewFromConfig(awsConfig),
		SSM:         ssm.NewFromConfig(awsConfig),
		ASGName:     asgName,
		Tracer:      tracer,
	}, nil
}

// DescribeInstances returns the details of the given instances from AWS,
// making sure that the instances are valid for further processing.
func (c *AWSCloudController) DescribeInstances(ctx context.Context, instanceIDs []string) (instances []Instance, err error) {
	c.Tracer.Capture(ctx, "aws.ec2.describeInstances", func(ctx context.Context) error {
		var output *ec2.DescribeInstancesOutput

		output, err = c.EC2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: instanceIDs,
		})

		if err != nil {
			err = fmt.Errorf("could not describe instances: %w", err)
			return err
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				if instance.InstanceId == nil {
					err = errors.New("could not find instance ID")
					return err
				}

				if instance.LaunchTime == nil {
					err = fmt.Errorf("could not find launch time for instance %s", *instance.InstanceId)
					return err
				}

				instances = append(instances, Instance{
					InstanceID: *instance.InstanceId,
					LaunchTime: *instance.LaunchTime,
				})
			}
		}

		return nil
	})

	return instances, err
}

// GetAutoscalingGroup returns the autoscaling group details from AWS.
//
// It makes sure that the autoscaling group exists and that there is only
// one autoscaling group with the given name.
func (c *AWSCloudController) GetAutoscalingGroup(ctx context.Context) (out *AutoScalingGroup, err error) {
	c.Tracer.Capture(ctx, "aws.asg.get", func(ctx context.Context) error {
		var output *autoscaling.DescribeAutoScalingGroupsOutput

		output, err = c.Autoscaling.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []string{c.ASGName},
		})

		if err != nil {
			err = fmt.Errorf("could not get autoscaling group details: %w", err)
			return err
		}

		if len(output.AutoScalingGroups) == 0 {
			err = fmt.Errorf("could not find autoscaling group %s", c.ASGName)
			return err
		} else if len(output.AutoScalingGroups) > 1 {
			err = fmt.Errorf("found more than one autoscaling group with name %s", c.ASGName)
			return err
		}

		asg := &output.AutoScalingGroups[0]

		// Convert AWS-specific ASG to generic ASG
		out = &AutoScalingGroup{
			Name:            *asg.AutoScalingGroupName,
			MinSize:         *asg.MinSize,
			MaxSize:         *asg.MaxSize,
			DesiredCapacity: *asg.DesiredCapacity,
			Instances:       make([]Instance, 0, len(asg.Instances)),
		}

		for _, instance := range asg.Instances {
			out.Instances = append(out.Instances, Instance{
				InstanceID:     *instance.InstanceId,
				LifecycleState: string(instance.LifecycleState),
			})
		}

		return nil
	})

	return
}

// GetSecret retrieves a secret value from AWS Systems Manager Parameter Store.
func (c *AWSCloudController) GetSecret(ctx context.Context, secretName string) (string, error) {
	var secret string
	var err error

	c.Tracer.Capture(ctx, "aws.ssm.secret", func(ctx context.Context) error {
		var output *ssm.GetParameterOutput

		output, err = c.SSM.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           aws.String(secretName),
			WithDecryption: aws.Bool(true),
		})

		if err != nil {
			return fmt.Errorf("could not get secret from SSM: %w", err)
		}

		if output.Parameter == nil {
			return errors.New("secret not found in SSM")
		}

		if output.Parameter.Value == nil {
			return errors.New("secret value is nil in SSM")
		}

		secret = *output.Parameter.Value
		return nil
	})

	return secret, err
}

// KillInstance detaches an instance from the autoscaling group and terminates it.
func (c *AWSCloudController) KillInstance(ctx context.Context, instanceID string) (err error) {
	c.Tracer.Capture(ctx, "aws.killinstance", func(ctx context.Context) error {
		c.Tracer.AddAnnotation(ctx, "instance_id", instanceID)

		_, err = c.Autoscaling.DetachInstances(ctx, &autoscaling.DetachInstancesInput{
			AutoScalingGroupName:           aws.String(c.ASGName),
			InstanceIds:                    []string{instanceID},
			ShouldDecrementDesiredCapacity: aws.Bool(true),
		})

		// A special instance of the error is when the instance is not part of
		// the autoscaling group. This can happen when the instance successfully
		// detached but for some reason the termination request failed.
		//
		// This will fix one-off errors and machines manually connected to the
		// worker pool (as long as they terminate upon request), but if there
		// are multiple ASGs connected to the same worker pool, this will be a
		// common occurrence and will break the entire autoscaling logic.
		if err != nil && !strings.Contains(err.Error(), "is not part of Auto Scaling group") {
			err = fmt.Errorf("could not detach instance from autoscaling group: %v", err)
			return err
		}

		// Now that the instance is detached from the ASG (or was never part of
		// the ASG), we can terminate it.
		_, err = c.EC2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})

		if err != nil {
			err = fmt.Errorf("could not terminate detached instance: %v", err)
			return err
		}

		return nil
	})

	return
}

// ScaleUpASG scales up the autoscaling group to the desired capacity.
func (c *AWSCloudController) ScaleUpASG(ctx context.Context, desiredCapacity int32) (err error) {
	c.Tracer.Capture(ctx, "aws.asg.scaleup", func(ctx context.Context) error {
		c.Tracer.AddMetadata(ctx, "desired_capacity", desiredCapacity)

		_, err = c.Autoscaling.SetDesiredCapacity(ctx, &autoscaling.SetDesiredCapacityInput{
			AutoScalingGroupName: aws.String(c.ASGName),
			DesiredCapacity:      aws.Int32(int32(desiredCapacity)),
		})

		if err != nil {
			err = fmt.Errorf("could not set desired capacity: %v", err)
			return err
		}

		return nil
	})

	return
}
