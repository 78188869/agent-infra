// Package migration provides database migration utilities.
package migration

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/example/agent-infra/internal/model"
)

// Migrator handles database migrations.
type Migrator struct {
	db *gorm.DB
}

// NewMigrator creates a new Migrator instance.
func NewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{db: db}
}

// AutoMigrate runs auto migration for all models.
// This is suitable for development environment.
func (m *Migrator) AutoMigrate() error {
	models := model.AllModels()

	log.Printf("Starting auto migration for %d models...", len(models))

	for _, mdl := range models {
		if err := m.db.AutoMigrate(mdl); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", mdl, err)
		}
		log.Printf("Migrated: %T", mdl)
	}

	log.Println("Auto migration completed successfully")
	return nil
}

// DropAll drops all tables. Use with caution!
// This is only for development/testing purposes.
func (m *Migrator) DropAll() error {
	models := model.AllModels()

	log.Printf("Dropping all tables (%d models)...", len(models))

	// Drop in reverse order to handle foreign key constraints
	for i := len(models) - 1; i >= 0; i-- {
		mdl := models[i]
		if err := m.db.Migrator().DropTable(mdl); err != nil {
			log.Printf("Warning: failed to drop %T: %v", mdl, err)
		} else {
			log.Printf("Dropped: %T", mdl)
		}
	}

	log.Println("All tables dropped")
	return nil
}

// Reset drops all tables and re-creates them.
// This is only for development/testing purposes.
func (m *Migrator) Reset() error {
	if err := m.DropAll(); err != nil {
		return err
	}
	return m.AutoMigrate()
}

// MigrateOptions holds options for migration.
type MigrateOptions struct {
	// SeedData indicates whether to seed initial data.
	SeedData bool
}

// MigrateWithOptions runs migration with options.
func (m *Migrator) MigrateWithOptions(opts MigrateOptions) error {
	if err := m.AutoMigrate(); err != nil {
		return err
	}

	if opts.SeedData {
		if err := m.seedData(); err != nil {
			return fmt.Errorf("failed to seed data: %w", err)
		}
	}

	return nil
}

// seedData seeds initial data for development/testing.
func (m *Migrator) seedData() error {
	log.Println("Seeding initial data...")

	// Create default system provider (Claude Code)
	provider := &model.Provider{
		ID:          "system-claude-code",
		Scope:       model.ProviderScopeSystem,
		Name:        "Claude Code",
		Type:        model.ProviderTypeClaudeCode,
		Description: "Official Claude Code CLI runtime",
		RuntimeType: model.RuntimeTypeCLI,
		Status:      model.ProviderStatusActive,
	}

	if err := m.db.FirstOrCreate(provider, "scope = ? AND name = ?", model.ProviderScopeSystem, "Claude Code").Error; err != nil {
		return fmt.Errorf("failed to create default provider: %w", err)
	}

	log.Println("Initial data seeded successfully")
	return nil
}

// CheckConnection verifies the database connection.
func (m *Migrator) CheckConnection() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}
