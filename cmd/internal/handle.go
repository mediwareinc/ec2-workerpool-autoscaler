package internal

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"

	"github.com/spacelift-io/awsautoscalr/internal"
)

func Handle(ctx context.Context, logger *slog.Logger, tracer internal.Tracer) error {
	var cfg internal.RuntimeConfig
	if err := cfg.Parse(nil); err != nil {
		return fmt.Errorf("could not parse environment variables: %w", err)
	}

	// Create cloud controller
	cloudController, err := internal.NewAWSCloudController(ctx, cfg.AutoscalingRegion, cfg.AutoscalingGroupARN, tracer)
	if err != nil {
		return fmt.Errorf("could not create cloud controller: %w", err)
	}

	controller, err := internal.NewController(ctx, &cfg, cloudController, tracer)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}
	return internal.NewAutoScaler(controller, logger).Scale(ctx, cfg)
}
