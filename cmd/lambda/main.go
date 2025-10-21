package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"
	"golang.org/x/exp/slog"

	"github.com/spacelift-io/awsautoscalr/cmd/internal"
	appinternal "github.com/spacelift-io/awsautoscalr/internal"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	lambda.Start(func(ctx context.Context) error {
		tracer := appinternal.NewXRayTracer()
		if err := tracer.Configure(appinternal.TracerConfig{ServiceVersion: "1.2.3"}); err != nil {
			return fmt.Errorf("could not configure X-Ray: %w", err)
		}

		logger := logger
		if lc, ok := lambdacontext.FromContext(ctx); ok {
			logger = logger.With("aws_request_id", lc.AwsRequestID)
		}

		return internal.Handle(ctx, logger, tracer)
	})
}
