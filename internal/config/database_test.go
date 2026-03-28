package config

import (
	"testing"
	"time"
)

func TestDatabaseConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   DatabaseConfig
		expected string
	}{
		{
			name: "basic DSN",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				Database: "testdb",
			},
			expected: "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "DSN with custom port",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "db.example.com",
				Port:     3307,
				Username: "admin",
				Password: "secret123",
				Database: "production",
			},
			expected: "admin:secret123@tcp(db.example.com:3307)/production?charset=utf8mb4&parseTime=True&loc=Local",
		},
		{
			name: "DSN with empty password",
			config: DatabaseConfig{
				Driver:   "mysql",
				Host:     "127.0.0.1",
				Port:     3306,
				Username: "user",
				Password: "",
				Database: "mydb",
			},
			expected: "user:@tcp(127.0.0.1:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.DSN()
			if got != tt.expected {
				t.Errorf("DSN() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultDatabaseConfig(t *testing.T) {
	cfg := DefaultDatabaseConfig()

	if cfg.Host != "localhost" {
		t.Errorf("Default host = %v, want localhost", cfg.Host)
	}
	if cfg.Port != 3306 {
		t.Errorf("Default port = %v, want 3306", cfg.Port)
	}
	if cfg.MaxIdleConns != 10 {
		t.Errorf("Default MaxIdleConns = %v, want 10", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 100 {
		t.Errorf("Default MaxOpenConns = %v, want 100", cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime != time.Hour {
		t.Errorf("Default ConnMaxLifetime = %v, want 1 hour", cfg.ConnMaxLifetime)
	}
}

func TestDatabaseConfig_ConnectionPoolSettings(t *testing.T) {
	cfg := DatabaseConfig{
		Host:            "localhost",
		Port:            3306,
		Username:        "root",
		Password:        "password",
		Database:        "testdb",
		MaxIdleConns:    5,
		MaxOpenConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
	}

	if cfg.MaxIdleConns != 5 {
		t.Errorf("MaxIdleConns = %v, want 5", cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns != 50 {
		t.Errorf("MaxOpenConns = %v, want 50", cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30 minutes", cfg.ConnMaxLifetime)
	}
}

func TestDatabaseConfig_Driver(t *testing.T) {
	cfg := DatabaseConfig{Driver: "sqlite"}
	if cfg.Driver != "sqlite" {
		t.Errorf("Driver = %v, want sqlite", cfg.Driver)
	}
}

func TestDatabaseConfig_SQLiteDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:   "sqlite",
		Database: "test.db",
	}
	got := cfg.DSN()
	if got != "test.db" {
		t.Errorf("SQLite DSN() = %v, want test.db", got)
	}
}

func TestDatabaseConfig_MySQLDSN(t *testing.T) {
	cfg := DatabaseConfig{
		Driver:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "password",
		Database: "testdb",
	}
	got := cfg.DSN()
	expected := "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"
	if got != expected {
		t.Errorf("MySQL DSN() = %v, want %v", got, expected)
	}
}

func TestDatabaseConfig_DefaultDriver(t *testing.T) {
	cfg := DatabaseConfig{}
	if cfg.Driver != "" {
		t.Errorf("Default Driver = %v, want empty (defaults to mysql)", cfg.Driver)
	}
}

func TestDatabaseConfig_IsSQLite(t *testing.T) {
	tests := []struct {
		driver string
		want   bool
	}{
		{"sqlite", true},
		{"mysql", false},
		{"", false},
	}
	for _, tt := range tests {
		cfg := DatabaseConfig{Driver: tt.driver}
		if got := cfg.IsSQLite(); got != tt.want {
			t.Errorf("IsSQLite(%q) = %v, want %v", tt.driver, got, tt.want)
		}
	}
}
