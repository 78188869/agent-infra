// Package config provides unified application configuration loading
// with environment variable expansion and YAML parsing.
package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// envPattern matches ${VAR} or ${VAR:default} syntax.
var envPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?\}`)

// ExpandEnv replaces ${VAR} and ${VAR:default} patterns in the input string
// with the corresponding environment variable values. If the variable is not
// set and a default is provided, the default is used. If no default is provided
// and the variable is not set, an empty string is used.
func ExpandEnv(s string) string {
	return envPattern.ReplaceAllStringFunc(s, func(match string) string {
		sub := envPattern.FindStringSubmatch(match)
		name := sub[1]
		defaultVal := sub[2]

		val, ok := os.LookupEnv(name)
		if ok {
			return val
		}
		return defaultVal
	})
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`
}

// LogFileConfig holds local file logging configuration.
type LogFileConfig struct {
	Dir        string `yaml:"dir"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level   string        `yaml:"level"`
	Format  string        `yaml:"format"`
	Outputs string        `yaml:"outputs"` // stdout | file | both
	File    LogFileConfig `yaml:"file"`
}

// K8sNamespaceConfig holds Kubernetes namespace configuration.
type K8sNamespaceConfig struct {
	ControlPlane string `yaml:"control_plane"`
	Sandbox      string `yaml:"sandbox"`
}

// K8sConfig holds Kubernetes connection configuration.
type K8sConfig struct {
	Kubeconfig string             `yaml:"kubeconfig"`
	Namespace  K8sNamespaceConfig `yaml:"namespace"`
}

// SLSConfig holds Aliyun SLS (Simple Log Service) configuration.
type SLSConfig struct {
	Endpoint     string `yaml:"endpoint"`
	Project      string `yaml:"project"`
	Logstore     string `yaml:"logstore"`
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
}

// RedisYAMLConfig holds Redis configuration as parsed from YAML (host/port style).
type RedisYAMLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// ToRedisConfig converts RedisYAMLConfig to RedisConfig by combining Host:Port into Addr.
func (c RedisYAMLConfig) ToRedisConfig() RedisConfig {
	host := c.Host
	port := c.Port
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 6379
	}

	return RedisConfig{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: c.Password,
		DB:       c.DB,
	}
}

// AppConfig holds the complete application configuration.
type AppConfig struct {
	Env      string         `yaml:"env"`
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisYAMLConfig `yaml:"redis"`
	Log      LogConfig      `yaml:"log"`
	K8s      K8sConfig      `yaml:"k8s"`
	SLS      SLSConfig      `yaml:"sls"`
}

// GetEnvironment returns the application environment.
// Priority: APP_ENV env var > config file env field > "production" default.
func (c *AppConfig) GetEnvironment() string {
	if env := os.Getenv("APP_ENV"); env != "" {
		return env
	}
	if c.Env != "" {
		return c.Env
	}
	return "production"
}

// IsLocal returns true if the application is running in a local or development environment.
func (c *AppConfig) IsLocal() bool {
	env := c.GetEnvironment()
	return env == "local" || env == "development"
}

// ApplyDefaults sets environment-specific defaults for configuration fields.
// Local environments default to debug mode with text log format.
// Production environments default to release mode with JSON log format.
func (c *AppConfig) ApplyDefaults() {
	// Sync env from GetEnvironment (respects APP_ENV override)
	c.Env = c.GetEnvironment()

	if c.IsLocal() {
		if c.Server.Port == 0 {
			c.Server.Port = 8080
		}
		if c.Server.Mode == "" {
			c.Server.Mode = "debug"
		}
		if c.Log.Level == "" {
			c.Log.Level = "debug"
		}
		if c.Log.Format == "" {
			c.Log.Format = "text"
		}
		if c.Log.Outputs == "" {
			c.Log.Outputs = "both"
		}
		if c.Log.File.Dir == "" {
			c.Log.File.Dir = "logs"
		}
		if c.Log.File.MaxAgeDays == 0 {
			c.Log.File.MaxAgeDays = 30
		}
	} else {
		if c.Server.Port == 0 {
			c.Server.Port = 8080
		}
		if c.Server.Mode == "" {
			c.Server.Mode = "release"
		}
		if c.Log.Level == "" {
			c.Log.Level = "info"
		}
		if c.Log.Format == "" {
			c.Log.Format = "json"
		}
		if c.Log.Outputs == "" {
			c.Log.Outputs = "stdout"
		}
	}
}

// ResolveConfigPath determines the config file path based on environment.
// Priority: CONFIG_PATH env var > APP_ENV-based selection > default config.yaml.
func ResolveConfigPath() string {
	// Explicit override takes highest priority
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// Local/development environments use config.local.yaml
	env := os.Getenv("APP_ENV")
	if env == "local" || env == "development" {
		return "configs/config.local.yaml"
	}

	return "configs/config.yaml"
}

// Load reads a YAML configuration file, expands environment variables,
// and unmarshals it into an AppConfig.
func Load(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Expand environment variables in the raw YAML content
	expanded := ExpandEnv(string(data))

	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	// Override env with APP_ENV if set
	cfg.Env = cfg.GetEnvironment()

	cfg.ApplyDefaults()

	return &cfg, nil
}
