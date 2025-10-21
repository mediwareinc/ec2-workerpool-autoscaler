package internal

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/shurcooL/graphql"
	spacelift "github.com/spacelift-io/spacectl/client"
	"github.com/spacelift-io/spacectl/client/session"

	"github.com/spacelift-io/awsautoscalr/internal/ifaces"
)

// Controller is responsible for handling interactions with external systems
// (Spacelift API as well as cloud provider APIs) so that the main package
// can focus on the core logic.
type Controller struct {
	// Cloud controller for cloud-specific operations.
	Cloud CloudController

	// Spacelift client.
	Spacelift ifaces.Spacelift

	// Configuration.
	SpaceliftWorkerPoolID string

	// Tracer for distributed tracing.
	Tracer Tracer
}

// NewController creates a new controller instance.
func NewController(ctx context.Context, cfg *RuntimeConfig, cloudController CloudController, tracer Tracer) (*Controller, error) {
	// Get Spacelift API key secret from the cloud controller
	apiSecret, err := cloudController.GetSecret(ctx, cfg.SpaceliftAPISecretName)
	if err != nil {
		return nil, fmt.Errorf("could not get Spacelift API key secret: %w", err)
	}

	var slSession session.Session
	httpClient := tracer.Client(nil)

	tracer.Capture(ctx, "spacelift.session.get", func(ctx context.Context) error {
		slSession, err = session.FromAPIKey(ctx, httpClient)(
			cfg.SpaceliftAPIEndpoint,
			cfg.SpaceliftAPIKeyID,
			apiSecret,
		)

		return err
	})

	if err != nil {
		return nil, fmt.Errorf("could not create Spacelift session: %w", err)
	}

	return &Controller{
		Cloud:                 cloudController,
		Spacelift:             spacelift.New(httpClient, slSession),
		SpaceliftWorkerPoolID: cfg.SpaceliftWorkerPoolID,
		Tracer:                tracer,
	}, nil
}

// DescribeInstances returns the details of the given instances from the cloud provider,
// making sure that the instances are valid for further processing.
func (c *Controller) DescribeInstances(ctx context.Context, instanceIDs []string) (instances []Instance, err error) {
	return c.Cloud.DescribeInstances(ctx, instanceIDs)
}

// GetAutoscalingGroup returns the autoscaling group details from the cloud provider.
func (c *Controller) GetAutoscalingGroup(ctx context.Context) (out *AutoScalingGroup, err error) {
	return c.Cloud.GetAutoscalingGroup(ctx)
}

// GetWorkerPool returns the worker pool details from Spacelift.
func (c *Controller) GetWorkerPool(ctx context.Context) (out *WorkerPool, err error) {
	c.Tracer.Capture(ctx, "spacelift.workerpool.get", func(ctx context.Context) error {
		var wpDetails WorkerPoolDetails

		if err = c.Spacelift.Query(ctx, &wpDetails, map[string]any{"workerPool": c.SpaceliftWorkerPoolID}); err != nil {
			err = fmt.Errorf("could not get Spacelift worker pool details: %w", err)
			return err
		}

		if wpDetails.Pool == nil {
			err = errors.New("worker pool not found or not accessible")
			return err
		}

		// Remove any drained workers from the list of workers in the pool.
		// These don't count towards the "available workers", so they shouldn't
		// be included when making a scaling decision.
		worker_index := 0
		for _, worker := range wpDetails.Pool.Workers {
			if !worker.Drained {
				wpDetails.Pool.Workers[worker_index] = worker
				worker_index++
			}
		}
		wpDetails.Pool.Workers = wpDetails.Pool.Workers[:worker_index]

		// Let's sort the workers by their creation time. This is important
		// because Spacelift will always prioritize the newest workers for new runs,
		// so operating on the oldest ones first is going to be the safest.
		//
		// The backend should already return the workers in the order of their
		// creation, but let's be extra safe and not rely on that.
		sort.Slice(wpDetails.Pool.Workers, func(i, j int) bool {
			return wpDetails.Pool.Workers[i].CreatedAt < wpDetails.Pool.Workers[j].CreatedAt
		})

		c.Tracer.AddMetadata(ctx, "workers", len(wpDetails.Pool.Workers))
		c.Tracer.AddMetadata(ctx, "pending_runs", wpDetails.Pool.PendingRuns)

		out = wpDetails.Pool

		return nil
	})

	return
}

// Drain worker drains a worker in the Spacelift worker pool.
func (c *Controller) DrainWorker(ctx context.Context, workerID string) (drained bool, err error) {
	c.Tracer.Capture(ctx, "spacelift.worker.drain", func(ctx context.Context) error {
		c.Tracer.AddAnnotation(ctx, "worker_id", workerID)

		var worker *Worker

		if worker, err = c.workerDrainSet(ctx, workerID, true); err != nil {
			err = fmt.Errorf("could not drain worker: %w", err)
			return err
		}

		c.Tracer.AddMetadata(ctx, "worker_id", worker.ID)
		c.Tracer.AddMetadata(ctx, "worker_busy", worker.Busy)
		c.Tracer.AddMetadata(ctx, "worker_drained", worker.Drained)

		// If the worker is not busy, our job here is done.
		if !worker.Busy {
			drained = true
			return nil
		}

		if _, err = c.workerDrainSet(ctx, workerID, false); err != nil {
			err = fmt.Errorf("could not undrain a busy worker: %w", err)
			return err
		}

		return nil
	})

	return
}

func (c *Controller) KillInstance(ctx context.Context, instanceID string) (err error) {
	return c.Cloud.KillInstance(ctx, instanceID)
}

func (c *Controller) ScaleUpASG(ctx context.Context, desiredCapacity int32) (err error) {
	return c.Cloud.ScaleUpASG(ctx, desiredCapacity)
}

func (c *Controller) workerDrainSet(ctx context.Context, workerID string, drain bool) (worker *Worker, err error) {
	c.Tracer.Capture(ctx, fmt.Sprintf("spacelift.worker.setdrain.%t", drain), func(ctx context.Context) error {
		c.Tracer.AddAnnotation(ctx, "worker_id", workerID)
		c.Tracer.AddAnnotation(ctx, "worker_pool_id", c.SpaceliftWorkerPoolID)
		c.Tracer.AddAnnotation(ctx, "drain", drain)
		var mutation WorkerDrainSet

		variables := map[string]any{
			"workerPoolId": graphql.ID(c.SpaceliftWorkerPoolID),
			"workerId":     graphql.ID(workerID),
			"drain":        graphql.Boolean(drain),
		}

		if err = c.Spacelift.Mutate(ctx, &mutation, variables); err != nil {
			err = fmt.Errorf("could not set worker drain to %t: %w", drain, err)
			return err
		}

		worker = &mutation.Worker

		return nil
	})

	return
}
