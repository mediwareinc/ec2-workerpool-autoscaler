package internal

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"

	"github.com/spacelift-io/awsautoscalr/internal"
)

func Handle(ctx context.Context, logger *slog.Logger, isLocal bool) error {
	var cfg internal.RuntimeConfig
	if err := cfg.Parse(nil); err != nil {
		return fmt.Errorf("could not parse environment variables: %w", err)
	}

	// Create cloud controller
	cloudController, err := internal.NewAWSCloudController(ctx, cfg.AutoscalingRegion, cfg.AutoscalingGroupARN, "1.2.3")
	if err != nil {
		return fmt.Errorf("could not create cloud controller: %w", err)
	}

	if isLocal {
		var closeSegment func(error)
		ctx, closeSegment = cloudController.Tracer.Begin(ctx, "autoscaling")
		defer closeSegment(err)
	}

	controller, err := internal.NewController(ctx, &cfg, cloudController, cloudController.Tracer)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	err = internal.NewAutoScaler(controller, logger).Scale(ctx, cfg)
	return err
}
