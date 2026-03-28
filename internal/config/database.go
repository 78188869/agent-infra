// Package config provides database configuration and connection management.
package config

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Username        string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"name"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	MaxOpenConns    int           `yaml:"max_connections"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// Database wraps GORM DB with configuration.
type Database struct {
	*gorm.DB
	Config DatabaseConfig
}

// DefaultDatabaseConfig returns the default database configuration.
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Host:            "localhost",
		Port:            3306,
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxLifetime: time.Hour,
	}
}

// DSN returns the MySQL-compatible DSN string (works with OceanBase).
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}

// NewDatabase creates a new database connection with the given configuration.
func NewDatabase(cfg DatabaseConfig) (*Database, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return &Database{
		DB:     db,
		Config: cfg,
	}, nil
}

// Close closes the database connection.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping verifies the database connection is still alive.
func (d *Database) Ping() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
