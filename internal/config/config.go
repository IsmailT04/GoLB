package config

import (
	"os"
	"strconv"

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

	// TLS: if both set, server uses ListenAndServeTLS
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// LoadConfig reads the file at path, unmarshals it, then applies env var overrides (12-factor).
func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(file, &c); err != nil {
		return nil, err
	}

	// Env overrides (secrets and deployment-specific values)
	if v := os.Getenv("LB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.LBPort = p
		}
	}
	if v := os.Getenv("AUTH_TOKEN"); v != "" {
		c.AuthToken = v
	}
	if v := os.Getenv("CERT_FILE"); v != "" {
		c.CertFile = v
	}
	if v := os.Getenv("KEY_FILE"); v != "" {
		c.KeyFile = v
	}
	if v := os.Getenv("GOLB_STRATEGY"); v != "" {
		c.Strategy = v
	}

	return &c, nil
}
