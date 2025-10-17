package internal

import (
	"context"
	"fmt"

	"golang.org/x/exp/slog"

	"github.com/spacelift-io/awsautoscalr/internal"
)

func Handle(ctx context.Context, logger *slog.Logger) error {
	var cfg internal.RuntimeConfig
	if err := cfg.Parse(nil); err != nil {
		return fmt.Errorf("could not parse environment variables: %w", err)
	}

	controller, err := internal.NewController(ctx, &cfg)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}
	return internal.NewAutoScaler(controller, logger).Scale(ctx, cfg)
}
