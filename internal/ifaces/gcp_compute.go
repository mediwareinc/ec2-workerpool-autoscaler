package ifaces

import (
	"context"

	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2"
)

// GCPOperation is an interface for GCP compute operations that support waiting.
//
//go:generate mockery --inpackage --name GCPOperation --filename mock_gcp_operation.go
type GCPOperation interface {
	Wait(ctx context.Context, opts ...gax.CallOption) error
}

// GCPInstances is an interface which mocks the subset of the GCP Instances client that we use.
// This wrapper abstracts away the iterator complexity.
//
//go:generate mockery --inpackage --name GCPInstances --filename mock_gcp_instances.go
type GCPInstances interface {
	Get(context.Context, *computepb.GetInstanceRequest) (*computepb.Instance, error)
	ListAll(context.Context, *computepb.ListInstancesRequest) ([]*computepb.Instance, error)
	Delete(context.Context, *computepb.DeleteInstanceRequest) (GCPOperation, error)
	Close() error
}

// GCPInstanceGroupManagers is an interface which mocks the subset of the GCP Instance Group Managers client that we use.
// This wrapper abstracts away the iterator complexity.
//
//go:generate mockery --inpackage --name GCPInstanceGroupManagers --filename mock_gcp_instance_group_managers.go
type GCPInstanceGroupManagers interface {
	Get(context.Context, *computepb.GetInstanceGroupManagerRequest) (*computepb.InstanceGroupManager, error)
	ListManagedInstancesAll(context.Context, *computepb.ListManagedInstancesInstanceGroupManagersRequest) ([]*computepb.ManagedInstance, error)
	DeleteInstances(context.Context, *computepb.DeleteInstancesInstanceGroupManagerRequest) (GCPOperation, error)
	Resize(context.Context, *computepb.ResizeInstanceGroupManagerRequest) (GCPOperation, error)
	Close() error
}
