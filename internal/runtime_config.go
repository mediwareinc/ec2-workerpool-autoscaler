package internal

import "github.com/caarlos0/env/v9"

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

	AutoscalingGroupARN string `env:"AUTOSCALING_GROUP_ARN,notEmpty"`
}

// Parse parses environment variables into the RuntimeConfig.
// If envVars is nil, it uses the actual system environment variables.
// If envVars is provided, it uses those values instead.
func (c *RuntimeConfig) Parse(envVars map[string]string) error {
	opts := env.Options{
		Environment: envVars,
	}
	return env.ParseWithOptions(c, opts)
}
