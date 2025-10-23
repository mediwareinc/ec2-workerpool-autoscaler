package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"

	"github.com/spacelift-io/awsautoscalr/internal/ifaces"
)

// GCPCloudController implements CloudController for GCP.
type GCPCloudController struct {
	InstancesClient          ifaces.GCPInstances
	InstanceGroupManagers    ifaces.GCPInstanceGroupManagers
	SecretManagerClient      ifaces.GCPSecretManager
	Project                  string
	Zone                     string
	ManagedInstanceGroupName string
	MinSize                  int32
	MaxSize                  int32
	Tracer                   Tracer
}

// NewGCPCloudController creates a new GCP cloud controller instance.
func NewGCPCloudController(ctx context.Context, project, zone, migName string, minSize, maxSize int32, serviceVersion string) (*GCPCloudController, error) {
	// Create and configure OTEL tracer
	tracer := NewOtelTracer("autoscaler")
	if err := tracer.Configure(TracerConfig{ServiceVersion: serviceVersion, Enabled: true}); err != nil {
		return nil, fmt.Errorf("could not configure OpenTelemetry: %w", err)
	}

	instancesClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create GCP instances client: %w", err)
	}

	instanceGroupManagersClient, err := compute.NewInstanceGroupManagersRESTClient(ctx)
	if err != nil {
		instancesClient.Close()
		return nil, fmt.Errorf("could not create GCP instance group managers client: %w", err)
	}

	secretManagerClient, err := secretmanager.NewClient(ctx)
	if err != nil {
		instancesClient.Close()
		instanceGroupManagersClient.Close()
		return nil, fmt.Errorf("could not create GCP secret manager client: %w", err)
	}

	return &GCPCloudController{
		InstancesClient:          NewGCPInstancesWrapper(instancesClient),
		InstanceGroupManagers:    NewGCPInstanceGroupManagersWrapper(instanceGroupManagersClient),
		SecretManagerClient:      secretManagerClient,
		Project:                  project,
		Zone:                     zone,
		ManagedInstanceGroupName: migName,
		MinSize:                  minSize,
		MaxSize:                  maxSize,
		Tracer:                   tracer,
	}, nil
}

// Close closes the GCP clients.
func (c *GCPCloudController) Close() error {
	var errs []error
	if c.InstancesClient != nil {
		if err := c.InstancesClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.InstanceGroupManagers != nil {
		if err := c.InstanceGroupManagers.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if c.SecretManagerClient != nil {
		if err := c.SecretManagerClient.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing GCP clients: %v", errs)
	}
	return nil
}

// DescribeInstances returns the details of the given instances from GCP,
// making sure that the instances are valid for further processing.
// The instanceIDs parameter contains numeric instance IDs.
func (c *GCPCloudController) DescribeInstances(ctx context.Context, instanceIDs []string) (instances []Instance, err error) {
	c.Tracer.Capture(ctx, "gcp.compute.describeInstances", func(ctx context.Context) error {
		for _, instanceID := range instanceIDs {

			req := &computepb.GetInstanceRequest{
				Project:  c.Project,
				Zone:     c.Zone,
				Instance: instanceID,
			}

			instance, getErr := c.InstancesClient.Get(ctx, req)
			if getErr != nil {
				err = fmt.Errorf("could not get instance %s: %w", instanceID, getErr)
				return err
			}

			if instance.CreationTimestamp == nil {
				err = fmt.Errorf("could not find creation timestamp for instance %s", instanceID)
				return err
			}

			// Parse GCP timestamp format (RFC3339)
			launchTime, parseErr := time.Parse(time.RFC3339, *instance.CreationTimestamp)
			if parseErr != nil {
				err = fmt.Errorf("could not parse creation timestamp for instance %s: %w", instanceID, parseErr)
				return err
			}

			instances = append(instances, Instance{
				InstanceID: strconv.FormatUint(*instance.Id, 10),
				LaunchTime: launchTime,
			})
		}

		return nil
	})

	return instances, err
}

// GetAutoscalingGroup returns the managed instance group details from GCP.
//
// It makes sure that the managed instance group exists and returns its details.
func (c *GCPCloudController) GetAutoscalingGroup(ctx context.Context) (out *AutoScalingGroup, err error) {
	c.Tracer.Capture(ctx, "gcp.mig.get", func(ctx context.Context) error {
		req := &computepb.GetInstanceGroupManagerRequest{
			Project:              c.Project,
			Zone:                 c.Zone,
			InstanceGroupManager: c.ManagedInstanceGroupName,
		}

		mig, getErr := c.InstanceGroupManagers.Get(ctx, req)
		if getErr != nil {
			err = fmt.Errorf("could not get managed instance group details: %w", getErr)
			return err
		}

		if mig.Name == nil {
			err = fmt.Errorf("could not find managed instance group name")
			return err
		}

		// Convert GCP MIG to generic AutoScalingGroup
		out = &AutoScalingGroup{
			Name:            *mig.Name,
			MinSize:         c.MinSize,
			MaxSize:         c.MaxSize,
			DesiredCapacity: int32(0),
			Instances:       []Instance{},
		}

		// Get target size (desired capacity)
		if mig.TargetSize != nil {
			out.DesiredCapacity = int32(*mig.TargetSize)
		}

		// List managed instances
		listReq := &computepb.ListManagedInstancesInstanceGroupManagersRequest{
			Project:              c.Project,
			Zone:                 c.Zone,
			InstanceGroupManager: c.ManagedInstanceGroupName,
		}

		managedInstances, listErr := c.InstanceGroupManagers.ListManagedInstancesAll(ctx, listReq)
		if listErr != nil {
			out = nil
			err = fmt.Errorf("could not list managed instances: %w", listErr)
			return err
		}

		for _, managedInstance := range managedInstances {
			if managedInstance.Instance == nil || managedInstance.Id == nil {
				continue
			}

			// Map GCP instance status to AWS-compatible lifecycle state
			lifecycleState := MapGCPStatusToAWSLifecycleState(managedInstance.InstanceStatus, managedInstance.CurrentAction)

			out.Instances = append(out.Instances, Instance{
				InstanceID:     strconv.FormatUint(*managedInstance.Id, 10),
				LifecycleState: lifecycleState,
			})
		}

		return nil
	})

	return
}

// GetSecret retrieves a secret value from GCP Secret Manager.
func (c *GCPCloudController) GetSecret(ctx context.Context, secretName string) (string, error) {
	var secret string
	var err error

	c.Tracer.Capture(ctx, "gcp.secretmanager.secret", func(ctx context.Context) error {
		// Build the resource name for the secret version
		// Format: projects/*/secrets/*/versions/latest
		name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", c.Project, secretName)

		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: name,
		}

		result, accessErr := c.SecretManagerClient.AccessSecretVersion(ctx, req)
		if accessErr != nil {
			err = fmt.Errorf("could not access secret from Secret Manager: %w", accessErr)
			return err
		}

		if result.Payload == nil || result.Payload.Data == nil {
			err = fmt.Errorf("secret payload is empty")
			return err
		}

		secret = string(result.Payload.Data)
		return nil
	})

	return secret, err
}

// KillInstance removes an instance from the managed instance group and deletes it.
// The instanceID parameter is a numeric instance ID.
func (c *GCPCloudController) KillInstance(ctx context.Context, instanceID string) (err error) {
	c.Tracer.Capture(ctx, "gcp.killinstance", func(ctx context.Context) error {
		c.Tracer.AddAnnotation(ctx, "instance_id", instanceID)

		req := &computepb.GetInstanceRequest{
			Project:  c.Project,
			Zone:     c.Zone,
			Instance: instanceID,
		}

		instance, getErr := c.InstancesClient.Get(ctx, req)
		if getErr != nil {
			err = fmt.Errorf("could not get instance %s: %w", instanceID, getErr)
			return err
		}

		// Build the full instance URL that GCP expects
		instanceURL := fmt.Sprintf("zones/%s/instances/%s", c.Zone, *instance.Name)

		// Delete the instance from the managed instance group
		// This will decrease the target size
		deleteReq := &computepb.DeleteInstancesInstanceGroupManagerRequest{
			Project:              c.Project,
			Zone:                 c.Zone,
			InstanceGroupManager: c.ManagedInstanceGroupName,
			InstanceGroupManagersDeleteInstancesRequestResource: &computepb.InstanceGroupManagersDeleteInstancesRequest{
				Instances: []string{instanceURL},
			},
		}

		op, deleteErr := c.InstanceGroupManagers.DeleteInstances(ctx, deleteReq)
		if deleteErr != nil {
			// Check if the instance is not part of the MIG
			if strings.Contains(deleteErr.Error(), "is not a member") ||
				strings.Contains(deleteErr.Error(), "not found") {
				// Instance is not part of the MIG, try to delete it directly
				directDeleteReq := &computepb.DeleteInstanceRequest{
					Project:  c.Project,
					Zone:     c.Zone,
					Instance: instanceID,
				}

				directOp, directErr := c.InstancesClient.Delete(ctx, directDeleteReq)
				if directErr != nil {
					err = fmt.Errorf("could not delete instance directly: %w", directErr)
					return err
				}

				// Wait for the direct delete operation to complete
				if waitErr := directOp.Wait(ctx); waitErr != nil {
					err = fmt.Errorf("error waiting for direct delete operation: %w", waitErr)
					return err
				}

				return nil
			}

			err = fmt.Errorf("could not delete instance from managed instance group: %w", deleteErr)
			return err
		}

		// Wait for the operation to complete
		if waitErr := op.Wait(ctx); waitErr != nil {
			err = fmt.Errorf("error waiting for delete operation: %w", waitErr)
			return err
		}

		return nil
	})

	return
}

// MapGCPStatusToAWSLifecycleState converts GCP instance status to AWS-compatible lifecycle state.
// This ensures compatibility with the existing autoscaling logic that expects AWS lifecycle states.
func MapGCPStatusToAWSLifecycleState(instanceStatus *string, currentAction *string) string {
	// If we have a current action (like CREATING, DELETING, etc.), prioritize that
	if currentAction != nil {
		action := *currentAction
		switch action {
		case "CREATING", "RECREATING", "CREATING_WITHOUT_RETRIES":
			return "Pending"
		case "DELETING", "ABANDONING":
			return "Terminating"
		case "RESTARTING":
			return "Pending"
		case "NONE", "REFRESHING", "VERIFYING":
			// Fall through to check instance status
		default:
			// Unknown action, fall through to instance status
		}
	}

	// Check instance status
	if instanceStatus != nil {
		status := *instanceStatus
		switch status {
		case "RUNNING":
			return "InService"
		case "PROVISIONING", "STAGING", "REPAIRING":
			return "Pending"
		case "STOPPING", "STOPPED", "SUSPENDING", "SUSPENDED", "TERMINATED":
			return "Terminating"
		default:
			// Unknown status
			return ""
		}
	}

	// Default to empty string if we can't determine the state
	return ""
}

// ScaleUpASG scales up the managed instance group to the desired capacity.
func (c *GCPCloudController) ScaleUpASG(ctx context.Context, desiredCapacity int32) (err error) {
	c.Tracer.Capture(ctx, "gcp.mig.scaleup", func(ctx context.Context) error {
		c.Tracer.AddMetadata(ctx, "desired_capacity", desiredCapacity)

		req := &computepb.ResizeInstanceGroupManagerRequest{
			Project:              c.Project,
			Zone:                 c.Zone,
			InstanceGroupManager: c.ManagedInstanceGroupName,
			Size:                 int32(desiredCapacity),
		}

		op, resizeErr := c.InstanceGroupManagers.Resize(ctx, req)
		if resizeErr != nil {
			err = fmt.Errorf("could not resize managed instance group: %w", resizeErr)
			return err
		}

		// Wait for the operation to complete
		if waitErr := op.Wait(ctx); waitErr != nil {
			err = fmt.Errorf("error waiting for resize operation: %w", waitErr)
			return err
		}

		return nil
	})

	return
}

// GetTracer returns the tracer instance for this cloud controller.
func (c *GCPCloudController) GetTracer() Tracer {
	return c.Tracer
}
