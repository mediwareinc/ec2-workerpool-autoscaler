package internal

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/exp/slog"
)

// Instance represents a generic compute instance from a cloud provider.
// This struct contains only the fields used by the Autoscaler.
// LaunchTime is only populated when retrieving full instance details (zero value otherwise).
// LifecycleState is only populated for instances within an autoscaling group (empty string otherwise).
type Instance struct {
	InstanceID     string
	LaunchTime     time.Time
	LifecycleState string
}

// AutoScalingGroup represents a generic autoscaling group from a cloud provider.
// This struct contains only the fields used by the Autoscaler.
type AutoScalingGroup struct {
	Name            string
	MinSize         int32
	MaxSize         int32
	DesiredCapacity int32
	Instances       []Instance
}

//go:generate mockery --output ./ --name ControllerInterface --filename mock_controller_test.go --outpkg internal_test --structname MockController
type ControllerInterface interface {
	DescribeInstances(ctx context.Context, instanceIDs []string) (instances []Instance, err error)
	GetAutoscalingGroup(ctx context.Context) (out *AutoScalingGroup, err error)
	GetWorkerPool(ctx context.Context) (out *WorkerPool, err error)
	DrainWorker(ctx context.Context, workerID string) (drained bool, err error)
	KillInstance(ctx context.Context, instanceID string) (err error)
	ScaleUpASG(ctx context.Context, desiredCapacity int32) (err error)
}

type AutoScaler struct {
	controller ControllerInterface
	logger     *slog.Logger
}

func NewAutoScaler(controller ControllerInterface, logger *slog.Logger) *AutoScaler {
	return &AutoScaler{controller: controller, logger: logger}
}

func (s AutoScaler) Scale(ctx context.Context, cfg RuntimeConfig) error {
	error_count := 0

	groupKey, groupID := cfg.GroupKeyAndID()
	logger := s.logger.With(
		groupKey, groupID,
		"worker_pool_id", cfg.SpaceliftWorkerPoolID,
	)

	workerPool, err := s.controller.GetWorkerPool(ctx)
	if err != nil {
		return fmt.Errorf("could not get worker pool: %w", err)
	}

	asg, err := s.controller.GetAutoscalingGroup(ctx)
	if err != nil {
		return fmt.Errorf("could not get autoscaling group: %w", err)
	}

	state, err := NewState(workerPool, asg, cfg)
	if err != nil {
		return fmt.Errorf("could not create state: %w", err)
	}

	// Let's make sure that for each of the in-service instances we have a
	// corresponding worker in Spacelift, or that we have "stray" machines.
	if strayInstances := state.StrayInstances(); len(strayInstances) > 0 {
		// There's a question of what to do with the "stray" machines. The
		// decision will be made based on the creation timestamp.
		instances, err := s.controller.DescribeInstances(ctx, strayInstances)
		if err != nil {
			return fmt.Errorf("could not list instances: %w", err)
		}

		for _, instance := range instances {
			logger := logger.With("instance_id", instance.InstanceID)
			instanceAge := time.Since(instance.LaunchTime)

			logger = logger.With(
				"launch_timestamp", instance.LaunchTime.Unix(),
				"instance_age", instanceAge,
			)

			// If the machine was only created recently (say a generous window of 10
			// minutes), it is possible that it hasn't managed to register itself with
			// Spacelift yet. But if it's been around for a while we will want to kill
			// it and remove it from the ASG.
			if instanceAge > 10*time.Minute {
				logger.Warn("instance has no corresponding worker in Spacelift, removing from the " + cfg.GroupResource())

				if err := s.controller.KillInstance(ctx, instance.InstanceID); err != nil {
					logger.Error("could not kill stray instance", "error", err)
					error_count++
					continue
				}

				logger.Info(fmt.Sprintf("instance successfully removed from the %s and terminated", cfg.GroupResource()))
			}
		}
	}

	decision := state.Decide(cfg.AutoscalingMaxCreate, cfg.AutoscalingMaxKill)

	logger = logger.With(
		cfg.GroupPrefix()+"_instances", len(asg.Instances),
		cfg.GroupPrefix()+"_desired_capacity", asg.DesiredCapacity,
		"scaling_decision_comments", decision.Comments,
		"spacelift_workers", len(workerPool.Workers),
		"spacelift_pending_runs", workerPool.PendingRuns,
	)

	if decision.ScalingDirection == ScalingDirectionNone {
		logger.Info("not scaling the " + cfg.GroupResource())
		return nil
	}

	if decision.ScalingDirection == ScalingDirectionUp {
		logger.With("instances", decision.ScalingSize).Info("scaling up the " + cfg.GroupResource())

		if err := s.controller.ScaleUpASG(ctx, asg.DesiredCapacity+int32(decision.ScalingSize)); err != nil {
			return fmt.Errorf("could not scale up %s: %w", cfg.GroupResource(), err)
		}

		return nil
	}

	// If we got this far, we're scaling down.
	logger.With("instances", decision.ScalingSize).Info("scaling down " + cfg.GroupResource())

	scalableWorkers := state.ScalableWorkers()

	for i := 0; i < decision.ScalingSize; i++ {
		worker := scalableWorkers[i]

		_, instanceID, _ := worker.InstanceIdentity()

		logger := logger.With(
			"worker_id", worker.ID,
			"instance_id", instanceID,
		)
		logger.Info(fmt.Sprintf("scaling down %s and killing worker", cfg.GroupResource()))

		drained, err := s.controller.DrainWorker(ctx, worker.ID)
		if err != nil {
			logger.Error("could not drain worker", "error", err)
			error_count++
			continue
		}

		if !drained {
			logger.Warn("worker was busy; skipping termination")
			continue
		}

		if err := s.controller.KillInstance(ctx, string(instanceID)); err != nil {
			logger.Error("could not kill instance", "error", err)
			error_count++
			continue
		}
	}

	if error_count > 0 {
		return fmt.Errorf("encountered %d errors during scale-down", error_count)
	}

	return nil
}
