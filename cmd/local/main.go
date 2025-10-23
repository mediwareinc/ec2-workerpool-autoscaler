package main

import (
	"context"
	"os"

	"golang.org/x/exp/slog"

	cmdinternal "github.com/spacelift-io/awsautoscalr/cmd/internal"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx := context.Background()

	if err := cmdinternal.Handle(ctx, logger, true); err != nil {
		logger.With("msg", err.Error()).Error("could not handle request")
		os.Exit(1)
	}
}
