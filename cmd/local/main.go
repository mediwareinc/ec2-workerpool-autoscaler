package main

import (
	"context"
	"os"

	"golang.org/x/exp/slog"

	cmdinternal "github.com/spacelift-io/awsautoscalr/cmd/internal"
	"github.com/spacelift-io/awsautoscalr/internal"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tracer := internal.NewXRayTracer()
	if err := tracer.Configure(internal.TracerConfig{ServiceVersion: "1.2.3"}); err != nil {
		logger.With("msg", err.Error()).Error("could not configure X-Ray")
		os.Exit(1)
	}

	ctx, closeSegment := tracer.Begin(context.Background(), "autoscaling")

	if err := cmdinternal.Handle(ctx, logger, tracer); err != nil {
		logger.With("msg", err.Error()).Error("could not handle request")
		closeSegment(err)
		os.Exit(1)
	}

	closeSegment(nil)
}
