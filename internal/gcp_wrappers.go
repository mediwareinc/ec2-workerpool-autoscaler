package internal

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/spacelift-io/awsautoscalr/internal/ifaces"
)

// GCPInstancesWrapper wraps the real GCP Instances client and implements the GCPInstances interface.
type GCPInstancesWrapper struct {
	client *compute.InstancesClient
}

// NewGCPInstancesWrapper creates a new wrapper around a real GCP Instances client.
func NewGCPInstancesWrapper(client *compute.InstancesClient) *GCPInstancesWrapper {
	return &GCPInstancesWrapper{client: client}
}

func (w *GCPInstancesWrapper) Get(ctx context.Context, req *computepb.GetInstanceRequest) (*computepb.Instance, error) {
	return w.client.Get(ctx, req)
}

func (w *GCPInstancesWrapper) ListAll(ctx context.Context, req *computepb.ListInstancesRequest) ([]*computepb.Instance, error) {
	var instances []*computepb.Instance
	it := w.client.List(ctx, req)
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

func (w *GCPInstancesWrapper) Delete(ctx context.Context, req *computepb.DeleteInstanceRequest) (ifaces.GCPOperation, error) {
	return w.client.Delete(ctx, req)
}

func (w *GCPInstancesWrapper) Close() error {
	return w.client.Close()
}

// GCPInstanceGroupManagersWrapper wraps the real GCP Instance Group Managers client and implements the GCPInstanceGroupManagers interface.
type GCPInstanceGroupManagersWrapper struct {
	client *compute.InstanceGroupManagersClient
}

// NewGCPInstanceGroupManagersWrapper creates a new wrapper around a real GCP Instance Group Managers client.
func NewGCPInstanceGroupManagersWrapper(client *compute.InstanceGroupManagersClient) *GCPInstanceGroupManagersWrapper {
	return &GCPInstanceGroupManagersWrapper{client: client}
}

func (w *GCPInstanceGroupManagersWrapper) Get(ctx context.Context, req *computepb.GetInstanceGroupManagerRequest) (*computepb.InstanceGroupManager, error) {
	return w.client.Get(ctx, req)
}

func (w *GCPInstanceGroupManagersWrapper) ListManagedInstancesAll(ctx context.Context, req *computepb.ListManagedInstancesInstanceGroupManagersRequest) ([]*computepb.ManagedInstance, error) {
	var instances []*computepb.ManagedInstance
	it := w.client.ListManagedInstances(ctx, req)
	for {
		instance, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, nil
}

func (w *GCPInstanceGroupManagersWrapper) DeleteInstances(ctx context.Context, req *computepb.DeleteInstancesInstanceGroupManagerRequest) (ifaces.GCPOperation, error) {
	return w.client.DeleteInstances(ctx, req)
}

func (w *GCPInstanceGroupManagersWrapper) Resize(ctx context.Context, req *computepb.ResizeInstanceGroupManagerRequest) (ifaces.GCPOperation, error) {
	return w.client.Resize(ctx, req)
}

func (w *GCPInstanceGroupManagersWrapper) Close() error {
	return w.client.Close()
}
