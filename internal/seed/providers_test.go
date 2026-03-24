package seed

import (
	"testing"

	"github.com/example/agent-infra/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedProviders(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Create providers table manually with SQLite-compatible schema
	// SQLite doesn't support ENUM, so we use TEXT instead
	err = db.Exec(`
		CREATE TABLE providers (
			id VARCHAR(36) PRIMARY KEY,
			scope TEXT NOT NULL DEFAULT 'system',
			tenant_id VARCHAR(36),
			user_id VARCHAR(36),
			name VARCHAR(64) NOT NULL,
			type TEXT NOT NULL,
			description TEXT,
			api_endpoint VARCHAR(512),
			api_key_ref VARCHAR(256),
			model_mapping JSON,
			runtime_type TEXT DEFAULT 'cli',
			runtime_image VARCHAR(256),
			runtime_command JSON,
			env_vars JSON,
			permissions JSON,
			enabled_plugins JSON,
			extra_params JSON,
			status TEXT DEFAULT 'active',
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("Failed to create providers table: %v", err)
	}

	// Create indexes for the unique constraint
	err = db.Exec(`CREATE UNIQUE INDEX uk_scope_name ON providers(scope, name, COALESCE(tenant_id, ''), COALESCE(user_id, ''))`).Error
	if err != nil {
		t.Fatalf("Failed to create unique index: %v", err)
	}

	// Run seed
	err = SeedProviders(db)
	if err != nil {
		t.Fatalf("SeedProviders failed: %v", err)
	}

	// Verify providers were created
	var count int64
	db.Model(&model.Provider{}).Where("scope = ?", model.ProviderScopeSystem).Count(&count)
	if count != int64(len(SystemProviders)) {
		t.Errorf("Expected %d system providers, got %d", len(SystemProviders), count)
	}

	// Test idempotency
	err = SeedProviders(db)
	if err != nil {
		t.Fatalf("Second SeedProviders failed: %v", err)
	}

	db.Model(&model.Provider{}).Where("scope = ?", model.ProviderScopeSystem).Count(&count)
	if count != int64(len(SystemProviders)) {
		t.Errorf("Expected %d system providers after re-seed, got %d", len(SystemProviders), count)
	}
}
