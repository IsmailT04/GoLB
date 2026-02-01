package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type BackendConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type Config struct {
	LBPort   int             `yaml:"lb_port"`
	Strategy string          `yaml:"strategy"` // round-robin, weighted-round-robin, least-connections
	Backends []BackendConfig `yaml:"backends"`

	EnableAuth bool   `yaml:"enable_auth"`
	AuthToken  string `yaml:"auth_token"`

	EnableRateLimit bool `yaml:"enable_ratelimit"`
	RateLimitPerMin int  `yaml:"rate_limit_per_min"`

	EnableCache bool `yaml:"enable_cache"`
}

// LoadConfig reads the file at path and unmarshals it into the Config struct
func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
