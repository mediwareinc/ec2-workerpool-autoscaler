package internal

import (
	"context"
)

// CloudController is an interface for cloud-specific operations.
// This allows the autoscaler to work with different cloud providers
// by implementing this interface for each provider.
type CloudController interface {
	// DescribeInstances returns the details of the given instances from the cloud provider,
	// making sure that the instances are valid for further processing.
	DescribeInstances(ctx context.Context, instanceIDs []string) (instances []Instance, err error)

	// GetAutoscalingGroup returns the autoscaling group details from the cloud provider.
	GetAutoscalingGroup(ctx context.Context) (out *AutoScalingGroup, err error)

	// KillInstance detaches an instance from the autoscaling group and terminates it.
	KillInstance(ctx context.Context, instanceID string) error

	// ScaleUpASG scales up the autoscaling group to the desired capacity.
	ScaleUpASG(ctx context.Context, desiredCapacity int32) error

	// GetSecret retrieves a secret value from the cloud provider's secret management service.
	GetSecret(ctx context.Context, secretName string) (string, error)

	// GetTracer returns the tracer instance for this cloud controller.
	GetTracer() Tracer
}
