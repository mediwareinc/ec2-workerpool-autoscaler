package ifaces

import (
	"context"

	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/googleapis/gax-go/v2"
)

// GCPSecretManager is an interface which mocks the subset of the GCP Secret Manager client that we use.
//
//go:generate mockery --inpackage --name GCPSecretManager --filename mock_gcp_secret_manager.go
type GCPSecretManager interface {
	AccessSecretVersion(context.Context, *secretmanagerpb.AccessSecretVersionRequest, ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}
