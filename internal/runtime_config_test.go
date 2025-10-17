package internal_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spacelift-io/awsautoscalr/internal"
)

func TestRuntimeConfig_Parse(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
		validate    func(t *testing.T, cfg internal.RuntimeConfig)
	}{
		{
			name: "parses all required and optional fields",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_PLATFORM":          "custom",
				"AUTOSCALING_SCALE_DOWN_DELAY":  "300",
				"AUTOSCALING_REGION":            "us-west-2",
				"AUTOSCALING_MAX_KILL":          "5",
				"AUTOSCALING_MAX_CREATE":        "10",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-west-2:123456789012:autoScalingGroup:test",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "test-key-id", cfg.SpaceliftAPIKeyID)
				assert.Equal(t, "test-secret-name", cfg.SpaceliftAPISecretName)
				assert.Equal(t, "https://test.app.spacelift.io", cfg.SpaceliftAPIEndpoint)
				assert.Equal(t, "test-pool-id", cfg.SpaceliftWorkerPoolID)
				assert.Equal(t, "custom", cfg.AutoscalingPlatform)
				assert.Equal(t, 300, cfg.AutoscalingScaleDownDelay)
				assert.Equal(t, "us-west-2", cfg.AutoscalingRegion)
				assert.Equal(t, 5, cfg.AutoscalingMaxKill)
				assert.Equal(t, 10, cfg.AutoscalingMaxCreate)
				assert.Equal(t, "arn:aws:autoscaling:us-west-2:123456789012:autoScalingGroup:test", cfg.AutoscalingGroupARN)
			},
		},
		{
			name: "applies default values for optional fields",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "aws", cfg.AutoscalingPlatform, "should default to 'aws'")
				assert.Equal(t, 0, cfg.AutoscalingScaleDownDelay, "should default to 0")
				assert.Equal(t, 1, cfg.AutoscalingMaxKill, "should default to 1")
				assert.Equal(t, 1, cfg.AutoscalingMaxCreate, "should default to 1")
			},
		},
		{
			name: "fails when SPACELIFT_API_KEY_ID is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when SPACELIFT_API_KEY_SECRET_NAME is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":       "test-key-id",
				"SPACELIFT_API_KEY_ENDPOINT": "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":   "test-pool-id",
				"AUTOSCALING_REGION":         "us-east-1",
				"AUTOSCALING_GROUP_ARN":      "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when SPACELIFT_API_KEY_ENDPOINT is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when SPACELIFT_WORKER_POOL_ID is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_REGION is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_GROUP_ARN is missing",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
			},
			expectError: true,
		},
		{
			name: "fails when SPACELIFT_API_KEY_ID is empty string",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_SCALE_DOWN_DELAY is not an integer",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
				"AUTOSCALING_SCALE_DOWN_DELAY":  "not-a-number",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_MAX_KILL is not an integer",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
				"AUTOSCALING_MAX_KILL":          "invalid",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_MAX_CREATE is not an integer",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
				"AUTOSCALING_MAX_CREATE":        "3.14",
			},
			expectError: true,
		},
		{
			name: "parses with minimum required fields",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "min-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "min-secret",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://min.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "min-pool",
				"AUTOSCALING_REGION":            "eu-west-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:eu-west-1:999999999999:autoScalingGroup:min",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "min-key-id", cfg.SpaceliftAPIKeyID)
				assert.Equal(t, "min-secret", cfg.SpaceliftAPISecretName)
				assert.Equal(t, "https://min.spacelift.io", cfg.SpaceliftAPIEndpoint)
				assert.Equal(t, "min-pool", cfg.SpaceliftWorkerPoolID)
				assert.Equal(t, "eu-west-1", cfg.AutoscalingRegion)
				assert.Equal(t, "arn:aws:autoscaling:eu-west-1:999999999999:autoScalingGroup:min", cfg.AutoscalingGroupARN)
			},
		},
		{
			name: "parses zero values for numeric fields when explicitly set",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
				"AUTOSCALING_SCALE_DOWN_DELAY":  "0",
				"AUTOSCALING_MAX_KILL":          "0",
				"AUTOSCALING_MAX_CREATE":        "0",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, 0, cfg.AutoscalingScaleDownDelay)
				assert.Equal(t, 0, cfg.AutoscalingMaxKill)
				assert.Equal(t, 0, cfg.AutoscalingMaxCreate)
			},
		},
		{
			name: "parses large integer values",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-east-1",
				"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
				"AUTOSCALING_SCALE_DOWN_DELAY":  "999999",
				"AUTOSCALING_MAX_KILL":          "100",
				"AUTOSCALING_MAX_CREATE":        "200",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, 999999, cfg.AutoscalingScaleDownDelay)
				assert.Equal(t, 100, cfg.AutoscalingMaxKill)
				assert.Equal(t, 200, cfg.AutoscalingMaxCreate)
			},
		},
		{
			name: "does not require AUTOSCALING_GROUP_ARN when platform is not aws",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-central1",
				"AUTOSCALING_PLATFORM":          "gcp",
				"AUTOSCALING_MIG_PROJECT_ID":    "my-project",
				"AUTOSCALING_MIG_NAME":          "my-mig",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "gcp", cfg.AutoscalingPlatform)
				assert.Equal(t, "", cfg.AutoscalingGroupARN, "AutoscalingGroupARN should be empty for GCP")
				assert.Equal(t, "my-project", cfg.AutoscalingMIGProjectID)
				assert.Equal(t, "my-mig", cfg.AutoscalingMIGName)
			},
		},
		{
			name: "fails when AUTOSCALING_MIG_PROJECT_ID is missing for gcp platform",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-central1",
				"AUTOSCALING_PLATFORM":          "gcp",
				"AUTOSCALING_MIG_NAME":          "my-mig",
			},
			expectError: true,
		},
		{
			name: "fails when AUTOSCALING_MIG_NAME is missing for gcp platform",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-central1",
				"AUTOSCALING_PLATFORM":          "gcp",
				"AUTOSCALING_MIG_PROJECT_ID":    "my-project",
			},
			expectError: true,
		},
		{
			name: "does not require AUTOSCALING_MIG_ZONE for gcp platform (regional MIG)",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-central1",
				"AUTOSCALING_PLATFORM":          "gcp",
				"AUTOSCALING_MIG_PROJECT_ID":    "my-project",
				"AUTOSCALING_MIG_NAME":          "my-regional-mig",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "gcp", cfg.AutoscalingPlatform)
				assert.Equal(t, "my-project", cfg.AutoscalingMIGProjectID)
				assert.Equal(t, "my-regional-mig", cfg.AutoscalingMIGName)
				assert.Equal(t, "", cfg.AutoscalingMIGZone, "AutoscalingMIGZone should be empty for regional MIG")
			},
		},
		{
			name: "parses AUTOSCALING_MIG_ZONE when provided for gcp platform (zonal MIG)",
			envVars: map[string]string{
				"SPACELIFT_API_KEY_ID":          "test-key-id",
				"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
				"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
				"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
				"AUTOSCALING_REGION":            "us-central1",
				"AUTOSCALING_PLATFORM":          "gcp",
				"AUTOSCALING_MIG_PROJECT_ID":    "my-project",
				"AUTOSCALING_MIG_NAME":          "my-zonal-mig",
				"AUTOSCALING_MIG_ZONE":          "us-central1-a",
			},
			expectError: false,
			validate: func(t *testing.T, cfg internal.RuntimeConfig) {
				assert.Equal(t, "gcp", cfg.AutoscalingPlatform)
				assert.Equal(t, "my-project", cfg.AutoscalingMIGProjectID)
				assert.Equal(t, "my-zonal-mig", cfg.AutoscalingMIGName)
				assert.Equal(t, "us-central1-a", cfg.AutoscalingMIGZone)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg internal.RuntimeConfig

			err := cfg.Parse(tt.envVars)

			if tt.expectError {
				assert.Error(t, err, "expected parsing to fail")
			} else {
				require.NoError(t, err, "expected parsing to succeed")
				if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestRuntimeConfig_FieldDefaults(t *testing.T) {
	// Test that defaults are correctly set when env vars are not provided
	envVars := map[string]string{
		"SPACELIFT_API_KEY_ID":          "test-key-id",
		"SPACELIFT_API_KEY_SECRET_NAME": "test-secret-name",
		"SPACELIFT_API_KEY_ENDPOINT":    "https://test.app.spacelift.io",
		"SPACELIFT_WORKER_POOL_ID":      "test-pool-id",
		"AUTOSCALING_REGION":            "us-east-1",
		"AUTOSCALING_GROUP_ARN":         "arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:test",
	}

	var cfg internal.RuntimeConfig

	err := cfg.Parse(envVars)
	require.NoError(t, err)

	// Verify default values
	assert.Equal(t, "aws", cfg.AutoscalingPlatform, "AutoscalingPlatform should default to 'aws'")
	assert.Equal(t, 0, cfg.AutoscalingScaleDownDelay, "AutoscalingScaleDownDelay should default to 0")
	assert.Equal(t, 1, cfg.AutoscalingMaxKill, "AutoscalingMaxKill should default to 1")
	assert.Equal(t, 1, cfg.AutoscalingMaxCreate, "AutoscalingMaxCreate should default to 1")
}

func TestRuntimeConfig_ParseWithNil(t *testing.T) {
	// Test that Parse works with nil, using actual system environment variables
	t.Setenv("SPACELIFT_API_KEY_ID", "system-key-id")
	t.Setenv("SPACELIFT_API_KEY_SECRET_NAME", "system-secret-name")
	t.Setenv("SPACELIFT_API_KEY_ENDPOINT", "https://system.spacelift.io")
	t.Setenv("SPACELIFT_WORKER_POOL_ID", "system-pool-id")
	t.Setenv("AUTOSCALING_REGION", "ap-south-1")
	t.Setenv("AUTOSCALING_GROUP_ARN", "arn:aws:autoscaling:ap-south-1:111111111111:autoScalingGroup:system")
	t.Setenv("AUTOSCALING_PLATFORM", "custom-platform")
	t.Setenv("AUTOSCALING_SCALE_DOWN_DELAY", "600")
	t.Setenv("AUTOSCALING_MAX_KILL", "3")
	t.Setenv("AUTOSCALING_MAX_CREATE", "7")

	var cfg internal.RuntimeConfig

	err := cfg.Parse(nil)
	require.NoError(t, err)

	// Verify values from system environment
	assert.Equal(t, "system-key-id", cfg.SpaceliftAPIKeyID)
	assert.Equal(t, "system-secret-name", cfg.SpaceliftAPISecretName)
	assert.Equal(t, "https://system.spacelift.io", cfg.SpaceliftAPIEndpoint)
	assert.Equal(t, "system-pool-id", cfg.SpaceliftWorkerPoolID)
	assert.Equal(t, "ap-south-1", cfg.AutoscalingRegion)
	assert.Equal(t, "arn:aws:autoscaling:ap-south-1:111111111111:autoScalingGroup:system", cfg.AutoscalingGroupARN)
	assert.Equal(t, "custom-platform", cfg.AutoscalingPlatform)
	assert.Equal(t, 600, cfg.AutoscalingScaleDownDelay)
	assert.Equal(t, 3, cfg.AutoscalingMaxKill)
	assert.Equal(t, 7, cfg.AutoscalingMaxCreate)
}
