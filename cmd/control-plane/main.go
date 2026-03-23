package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/internal/service"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Name     string `yaml:"name"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"database"`
}

// ToDatabaseConfig converts Config database settings to config.DatabaseConfig.
func (c *Config) ToDatabaseConfig() config.DatabaseConfig {
	return config.DatabaseConfig{
		Host:     c.Database.Host,
		Port:     c.Database.Port,
		Database: c.Database.Name,
		Username: c.Database.User,
		Password: c.Database.Password,
	}
}

func main() {
	// Load configuration
	cfg, err := loadConfig("cmd/control-plane/config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config, using defaults: %v", err)
		cfg = &Config{}
		cfg.Server.Port = 8080
		cfg.Server.Mode = "debug"
		cfg.Database.Host = "localhost"
		cfg.Database.Port = 3306
		cfg.Database.Name = "agent_infra"
		cfg.Database.User = "root"
	}

	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize database (optional - will use mock if not available)
	var tenantSvc service.TenantService
	db, err := config.NewDatabase(cfg.ToDatabaseConfig())
	if err != nil {
		log.Printf("Warning: failed to connect to database, using mock service: %v", err)
		tenantSvc = &mockTenantService{}
	} else {
		// Auto-migrate models
		if err := db.AutoMigrate(&model.Tenant{}); err != nil {
			log.Printf("Warning: failed to auto-migrate: %v", err)
		}
		// Create real service with repository
		tenantRepo := repository.NewTenantRepository(db.DB)
		tenantSvc = service.NewTenantService(tenantRepo)
	}

	// Setup router
	r := router.Setup(tenantSvc)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting control-plane server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// mockTenantService is a fallback service when database is not available.
type mockTenantService struct{}

func (m *mockTenantService) Create(ctx context.Context, req *service.CreateTenantRequest) (*model.Tenant, error) {
	return &model.Tenant{Name: req.Name, Status: model.TenantStatusActive}, nil
}

func (m *mockTenantService) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockTenantService) List(ctx context.Context, filter *service.TenantFilter) ([]*model.Tenant, int64, error) {
	return []*model.Tenant{}, 0, nil
}

func (m *mockTenantService) Update(ctx context.Context, id string, req *service.UpdateTenantRequest) error {
	return fmt.Errorf("database not available")
}

func (m *mockTenantService) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
