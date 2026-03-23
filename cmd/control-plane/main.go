package main

import (
	"fmt"
	"log"
	"os"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Mode string `yaml:"mode"`
	} `yaml:"server"`
}

func main() {
	// Load configuration
	cfg, err := loadConfig("cmd/control-plane/config.yaml")
	if err != nil {
		log.Printf("Warning: failed to load config, using defaults: %v", err)
		cfg = &Config{}
		cfg.Server.Port = 8080
		cfg.Server.Mode = "debug"
	}

	// Set gin mode
	gin.SetMode(cfg.Server.Mode)

	// Setup router
	r := router.Setup()

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting control-plane server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
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
