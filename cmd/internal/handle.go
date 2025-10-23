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

	// Create cloud controller based on platform
	var cloudController internal.CloudController
	var err error

	switch cfg.AutoscalingPlatform {
	case "aws":
		cloudController, err = internal.NewAWSCloudController(ctx, cfg.AutoscalingRegion, cfg.AutoscalingGroupARN, "1.2.3")
		if err != nil {
			return fmt.Errorf("could not create AWS cloud controller: %w", err)
		}
	case "gcp":
		cloudController, err = internal.NewGCPCloudController(ctx, cfg.AutoscalingMIGProjectID, cfg.AutoscalingMIGZone, cfg.AutoscalingMIGName, cfg.AutoscalingMIGMinSize, cfg.AutoscalingMIGMaxSize, "1.2.3")
		if err != nil {
			return fmt.Errorf("could not create GCP cloud controller: %w", err)
		}
	default:
		return fmt.Errorf("unsupported autoscaling platform: %s", cfg.AutoscalingPlatform)
	}

	if isLocal {
		var closeSegment func(error)
		ctx, closeSegment = cloudController.GetTracer().Begin(ctx, "autoscaling")
		defer closeSegment(err)
	}

	controller, err := internal.NewController(ctx, &cfg, cloudController, cloudController.GetTracer())
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	err = internal.NewAutoScaler(controller, logger).Scale(ctx, cfg)
	return err
}
