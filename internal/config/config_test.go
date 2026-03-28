package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "simple variable with default",
			input:    "${DB_HOST:localhost}",
			envVars:  map[string]string{},
			expected: "localhost",
		},
		{
			name:     "variable overridden by env",
			input:    "${DB_HOST:localhost}",
			envVars:  map[string]string{"DB_HOST": "prod-db.example.com"},
			expected: "prod-db.example.com",
		},
		{
			name:     "variable without default and not set",
			input:    "${MISSING_VAR}",
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name:     "variable without default but set",
			input:    "${MY_VAR}",
			envVars:  map[string]string{"MY_VAR": "hello"},
			expected: "hello",
		},
		{
			name:     "mixed content",
			input:    "host=${DB_HOST:localhost} port=${DB_PORT:3306}",
			envVars:  map[string]string{"DB_HOST": "remote"},
			expected: "host=remote port=3306",
		},
		{
			name:     "empty default",
			input:    "${KUBECONFIG:}",
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name:     "no variables",
			input:    "plain text",
			envVars:  map[string]string{},
			expected: "plain text",
		},
		{
			name:     "empty string",
			input:    "",
			envVars:  map[string]string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			got := ExpandEnv(tt.input)
			if got != tt.expected {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLoad_ValidYAML(t *testing.T) {
	yamlContent := `
env: production
server:
  port: 9090
  mode: release
database:
  host: db.example.com
  port: 3306
  user: admin
  password: secret
  name: myapp
  max_connections: 50
redis:
  host: redis.example.com
  port: 6380
log:
  level: warn
  format: json
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify top-level env
	if cfg.Env != "production" {
		t.Errorf("Env = %q, want %q", cfg.Env, "production")
	}

	// Verify server
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	// Verify database
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("Database.Host = %q, want %q", cfg.Database.Host, "db.example.com")
	}
	if cfg.Database.Username != "admin" {
		t.Errorf("Database.Username = %q, want %q", cfg.Database.Username, "admin")
	}
	if cfg.Database.Database != "myapp" {
		t.Errorf("Database.Database = %q, want %q", cfg.Database.Database, "myapp")
	}

	// Verify redis
	if cfg.Redis.Host != "redis.example.com" {
		t.Errorf("Redis.Host = %q, want %q", cfg.Redis.Host, "redis.example.com")
	}
	if cfg.Redis.Port != 6380 {
		t.Errorf("Redis.Port = %d, want 6380", cfg.Redis.Port)
	}

	// Verify log
	if cfg.Log.Level != "warn" {
		t.Errorf("Log.Level = %q, want %q", cfg.Log.Level, "warn")
	}
}

func TestLoad_EnvVarExpansion(t *testing.T) {
	t.Setenv("TEST_DB_HOST", "env-db.example.com")
	t.Setenv("TEST_REDIS_HOST", "env-redis.example.com")

	yamlContent := `
env: production
server:
  port: 8080
database:
  host: ${TEST_DB_HOST:localhost}
  port: 3306
  user: root
  password: secret
  name: testdb
redis:
  host: ${TEST_REDIS_HOST:localhost}
  port: 6379
log:
  level: info
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Database.Host != "env-db.example.com" {
		t.Errorf("Database.Host = %q, want %q (expanded from env)", cfg.Database.Host, "env-db.example.com")
	}
	if cfg.Redis.Host != "env-redis.example.com" {
		t.Errorf("Redis.Host = %q, want %q (expanded from env)", cfg.Redis.Host, "env-redis.example.com")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Load() expected error for missing file, got nil")
	}
}

func TestLoad_DefaultsEnvFromEnvVar(t *testing.T) {
	t.Setenv("APP_ENV", "staging")

	yamlContent := `
env: production
server:
  port: 8080
database:
  host: localhost
  port: 3306
  user: root
  password: secret
  name: testdb
redis:
  host: localhost
  port: 6379
log:
  level: info
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// APP_ENV env var should override the config file env field
	if cfg.Env != "staging" {
		t.Errorf("Env = %q, want %q (overridden by APP_ENV)", cfg.Env, "staging")
	}
}

func TestLoad_DatabaseFieldMapping(t *testing.T) {
	yamlContent := `
server:
  port: 8080
database:
  host: dbhost
  port: 3306
  user: dbuser
  password: dbpass
  name: dbname
  max_connections: 200
  max_idle_conns: 20
redis:
  host: localhost
  port: 6379
log:
  level: info
`
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify YAML field mappings
	if cfg.Database.Database != "dbname" {
		t.Errorf("Database.Database = %q, want %q (mapped from 'name')", cfg.Database.Database, "dbname")
	}
	if cfg.Database.Username != "dbuser" {
		t.Errorf("Database.Username = %q, want %q (mapped from 'user')", cfg.Database.Username, "dbuser")
	}
	if cfg.Database.MaxOpenConns != 200 {
		t.Errorf("Database.MaxOpenConns = %d, want %d (mapped from 'max_connections')", cfg.Database.MaxOpenConns, 200)
	}
	if cfg.Database.MaxIdleConns != 20 {
		t.Errorf("Database.MaxIdleConns = %d, want %d (mapped from 'max_idle_conns')", cfg.Database.MaxIdleConns, 20)
	}
}

func TestRedisYAMLConfig_ToRedisConfig(t *testing.T) {
	yamlCfg := RedisYAMLConfig{
		Host:     "redis.example.com",
		Port:     6380,
		Password: "secret",
		DB:       2,
	}

	result := yamlCfg.ToRedisConfig()

	if result.Addr != "redis.example.com:6380" {
		t.Errorf("Addr = %q, want %q", result.Addr, "redis.example.com:6380")
	}
	if result.Password != "secret" {
		t.Errorf("Password = %q, want %q", result.Password, "secret")
	}
	if result.DB != 2 {
		t.Errorf("DB = %d, want %d", result.DB, 2)
	}
}

func TestRedisYAMLConfig_ToRedisConfig_Empty(t *testing.T) {
	yamlCfg := RedisYAMLConfig{}

	result := yamlCfg.ToRedisConfig()

	if result.Addr != "localhost:6379" {
		t.Errorf("Addr = %q, want %q (default)", result.Addr, "localhost:6379")
	}
}

func TestAppConfig_IsLocal(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{
			name:     "local environment",
			env:      "local",
			expected: true,
		},
		{
			name:     "development environment",
			env:      "development",
			expected: true,
		},
		{
			name:     "production environment",
			env:      "production",
			expected: false,
		},
		{
			name:     "staging environment",
			env:      "staging",
			expected: false,
		},
		{
			name:     "empty environment",
			env:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &AppConfig{Env: tt.env}
			got := cfg.IsLocal()
			if got != tt.expected {
				t.Errorf("IsLocal() = %v, want %v for env=%q", got, tt.expected, tt.env)
			}
		})
	}
}

func TestAppConfig_ApplyDefaults_Local(t *testing.T) {
	cfg := &AppConfig{Env: "local"}
	cfg.ApplyDefaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080 for local", cfg.Server.Port)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode = %q, want %q for local", cfg.Server.Mode, "debug")
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want %q for local", cfg.Log.Level, "debug")
	}
	if cfg.Log.Format != "text" {
		t.Errorf("Log.Format = %q, want %q for local", cfg.Log.Format, "text")
	}
}

func TestAppConfig_ApplyDefaults_Production(t *testing.T) {
	cfg := &AppConfig{Env: "production"}
	cfg.ApplyDefaults()

	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode = %q, want %q for production", cfg.Server.Mode, "release")
	}
	if cfg.Log.Format != "json" {
		t.Errorf("Log.Format = %q, want %q for production", cfg.Log.Format, "json")
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want %q for production", cfg.Log.Level, "info")
	}
}
