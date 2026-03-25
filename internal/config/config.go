package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Auth         AuthConfig         `yaml:"auth"`
	APIKeys      map[string]string  `yaml:"api_keys"`
	DefaultRoute string             `yaml:"default_backend"`
	Routes       []RouteConfig      `yaml:"routes"`
	Backends     map[string]Backend `yaml:"backends"`
	Logging      LoggingConfig      `yaml:"logging"`
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	Enabled bool         `yaml:"enabled"`
	Users   []UserConfig `yaml:"users"`
}

// UserConfig represents a user credential
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// RouteConfig defines routing rules
type RouteConfig struct {
	ModelPattern string `yaml:"model_pattern"`
	Backend      string `yaml:"backend"`
}

// Backend represents a backend service configuration
type Backend struct {
	Type         string            `yaml:"type"`
	BaseURL      string            `yaml:"base_url"`
	APIKey       string            `yaml:"api_key"`
	ExtraHeaders map[string]string `yaml:"extra_headers"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level        string `yaml:"level"`
	Format       string `yaml:"format"`
	File         string `yaml:"file"`
	LogRequests  bool   `yaml:"log_requests"`
	LogResponses bool   `yaml:"log_responses"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 120 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 120 * time.Second
	}
	if c.DefaultRoute == "" {
		c.DefaultRoute = "openrouter"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
}

func (c *Config) validate() error {
	// Validate auth configuration
	if c.Auth.Enabled && len(c.Auth.Users) == 0 {
		return fmt.Errorf("auth is enabled but no users are configured")
	}

	// Validate that default backend exists
	if _, ok := c.Backends[c.DefaultRoute]; !ok {
		return fmt.Errorf("default backend '%s' not found in backends configuration", c.DefaultRoute)
	}

	// Validate each route's backend exists
	for _, route := range c.Routes {
		if _, ok := c.Backends[route.Backend]; !ok {
			return fmt.Errorf("backend '%s' in route not found in backends configuration", route.Backend)
		}
	}

	return nil
}

// Addr returns the server address string
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
