package internal

import (
	"fmt"

	"github.com/caarlos0/env/v9"
)

type RuntimeConfig struct {
	SpaceliftAPIKeyID      string `env:"SPACELIFT_API_KEY_ID,notEmpty"`
	SpaceliftAPISecretName string `env:"SPACELIFT_API_KEY_SECRET_NAME,notEmpty"`
	SpaceliftAPIEndpoint   string `env:"SPACELIFT_API_KEY_ENDPOINT,notEmpty"`
	SpaceliftWorkerPoolID  string `env:"SPACELIFT_WORKER_POOL_ID,notEmpty"`

	AutoscalingPlatform       string `env:"AUTOSCALING_PLATFORM" envDefault:"aws"`
	AutoscalingScaleDownDelay int    `env:"AUTOSCALING_SCALE_DOWN_DELAY" envDefault:"0"`
	AutoscalingRegion         string `env:"AUTOSCALING_REGION,notEmpty"`
	AutoscalingMaxKill        int    `env:"AUTOSCALING_MAX_KILL" envDefault:"1"`
	AutoscalingMaxCreate      int    `env:"AUTOSCALING_MAX_CREATE" envDefault:"1"`

	// AWS specific settings
	AutoscalingGroupARN string `env:"AUTOSCALING_GROUP_ARN"`

	// GCP specific settings
	AutoscalingMIGProjectID string `env:"AUTOSCALING_MIG_PROJECT_ID"`
	AutoscalingMIGZone      string `env:"AUTOSCALING_MIG_ZONE"` // Set if a zonal MIG is used, empty for regional MIGs
	AutoscalingMIGName      string `env:"AUTOSCALING_MIG_NAME"`
}

// Parse parses environment variables into the RuntimeConfig.
// If envVars is nil, it uses the actual system environment variables.
// If envVars is provided, it uses those values instead.
func (c *RuntimeConfig) Parse(envVars map[string]string) error {
	opts := env.Options{
		Environment: envVars,
	}
	if err := env.ParseWithOptions(c, opts); err != nil {
		return err
	}
	return c.Validate()
}

// Validate checks that the configuration is valid based on platform-specific requirements.
func (c *RuntimeConfig) Validate() error {
	// When platform is "aws", AutoscalingGroupARN must be set
	if c.AutoscalingPlatform == "aws" && c.AutoscalingGroupARN == "" {
		return fmt.Errorf("AUTOSCALING_GROUP_ARN is required when AUTOSCALING_PLATFORM is 'aws'")
	}

	// When platform is "gcp", GCP-specific fields must be set (except MIGZone which is optional)
	if c.AutoscalingPlatform == "gcp" {
		if c.AutoscalingMIGProjectID == "" {
			return fmt.Errorf("AUTOSCALING_MIG_PROJECT_ID is required when AUTOSCALING_PLATFORM is 'gcp'")
		}
		if c.AutoscalingMIGName == "" {
			return fmt.Errorf("AUTOSCALING_MIG_NAME is required when AUTOSCALING_PLATFORM is 'gcp'")
		}
	}

	return nil
}

func (c *RuntimeConfig) GroupKeyAndID() (string, string) {
	if c.AutoscalingPlatform == "gcp" {
		return gcpMigKey, c.AutoscalingMIGName
	}
	return asgKey, c.AutoscalingGroupARN
}

func (c *RuntimeConfig) GroupPrefix() string {
	if c.AutoscalingPlatform == "gcp" {
		return "mig"
	}
	return "asg"
}

func (c *RuntimeConfig) GroupResource() string {
	if c.AutoscalingPlatform == "gcp" {
		return "MIG"
	}
	return "ASG"
}
