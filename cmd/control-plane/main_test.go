package main

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/service"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		checkVal func(t *testing.T, cfg *Config)
	}{
		{
			name:    "load valid config",
			path:    "config.yaml",
			wantErr: false,
			checkVal: func(t *testing.T, cfg *Config) {
				if cfg.Server.Port != 8080 {
					t.Errorf("expected port 8080, got %d", cfg.Server.Port)
				}
				if cfg.Server.Mode != "debug" {
					t.Errorf("expected mode debug, got %s", cfg.Server.Mode)
				}
				if cfg.Database.Host != "localhost" {
					t.Errorf("expected database host localhost, got %s", cfg.Database.Host)
				}
			},
		},
		{
			name:    "load non-existent config",
			path:    "nonexistent.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkVal != nil {
				tt.checkVal(t, cfg)
			}
		})
	}
}

func TestConfigToDatabaseConfig(t *testing.T) {
	cfg := &Config{}
	cfg.Database.Host = "testhost"
	cfg.Database.Port = 3307
	cfg.Database.Name = "testdb"
	cfg.Database.User = "testuser"
	cfg.Database.Password = "testpass"

	dbCfg := cfg.ToDatabaseConfig()

	if dbCfg.Host != "testhost" {
		t.Errorf("expected host testhost, got %s", dbCfg.Host)
	}
	if dbCfg.Port != 3307 {
		t.Errorf("expected port 3307, got %d", dbCfg.Port)
	}
	if dbCfg.Database != "testdb" {
		t.Errorf("expected database testdb, got %s", dbCfg.Database)
	}
	if dbCfg.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", dbCfg.Username)
	}
	if dbCfg.Password != "testpass" {
		t.Errorf("expected password testpass, got %s", dbCfg.Password)
	}
}

func TestMockTenantService(t *testing.T) {
	mock := &mockTenantService{}
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		req := &service.CreateTenantRequest{
			Name: "test-tenant",
		}
		tenant, err := mock.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if tenant.Name != "test-tenant" {
			t.Errorf("expected name test-tenant, got %s", tenant.Name)
		}
	})

	t.Run("get by id returns error", func(t *testing.T) {
		_, err := mock.GetByID(ctx, "123")
		if err == nil {
			t.Error("expected error for GetByID, got nil")
		}
	})

	t.Run("list returns empty slice", func(t *testing.T) {
		tenants, total, err := mock.List(ctx, &service.TenantFilter{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
		if len(tenants) != 0 {
			t.Errorf("expected empty slice, got %d items", len(tenants))
		}
	})

	t.Run("update returns error", func(t *testing.T) {
		err := mock.Update(ctx, "123", &service.UpdateTenantRequest{})
		if err == nil {
			t.Error("expected error for Update, got nil")
		}
	})

	t.Run("delete returns error", func(t *testing.T) {
		err := mock.Delete(ctx, "123")
		if err == nil {
			t.Error("expected error for Delete, got nil")
		}
	})
}

func TestMockTemplateService(t *testing.T) {
	mock := &mockTemplateService{}
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		req := &service.CreateTemplateRequest{
			Name:     "test-template",
			TenantID: "tenant-123",
		}
		template, err := mock.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if template.Name != "test-template" {
			t.Errorf("expected name test-template, got %s", template.Name)
		}
	})

	t.Run("get by id returns error", func(t *testing.T) {
		_, err := mock.GetByID(ctx, "123")
		if err == nil {
			t.Error("expected error for GetByID, got nil")
		}
	})

	t.Run("list returns empty slice", func(t *testing.T) {
		templates, total, err := mock.List(ctx, &service.TemplateFilter{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
		if len(templates) != 0 {
			t.Errorf("expected empty slice, got %d items", len(templates))
		}
	})

	t.Run("update returns error", func(t *testing.T) {
		err := mock.Update(ctx, "123", &service.UpdateTemplateRequest{})
		if err == nil {
			t.Error("expected error for Update, got nil")
		}
	})

	t.Run("delete returns error", func(t *testing.T) {
		err := mock.Delete(ctx, "123")
		if err == nil {
			t.Error("expected error for Delete, got nil")
		}
	})
}

func TestMockTaskService(t *testing.T) {
	mock := &mockTaskService{}
	ctx := context.Background()

	t.Run("create", func(t *testing.T) {
		templateID := "template-123"
		req := &service.CreateTaskRequest{
			Name:       "test-task",
			TenantID:   "tenant-123",
			TemplateID: &templateID,
			CreatorID:  "creator-123",
			ProviderID: "provider-123",
		}
		task, err := mock.Create(ctx, req)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if task.Name != "test-task" {
			t.Errorf("expected name test-task, got %s", task.Name)
		}
	})

	t.Run("get by id returns error", func(t *testing.T) {
		_, err := mock.GetByID(ctx, "123")
		if err == nil {
			t.Error("expected error for GetByID, got nil")
		}
	})

	t.Run("list returns empty slice", func(t *testing.T) {
		tasks, total, err := mock.List(ctx, &service.TaskFilter{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0, got %d", total)
		}
		if len(tasks) != 0 {
			t.Errorf("expected empty slice, got %d items", len(tasks))
		}
	})

	t.Run("update returns error", func(t *testing.T) {
		err := mock.Update(ctx, "123", &service.UpdateTaskRequest{})
		if err == nil {
			t.Error("expected error for Update, got nil")
		}
	})

	t.Run("delete returns error", func(t *testing.T) {
		err := mock.Delete(ctx, "123")
		if err == nil {
			t.Error("expected error for Delete, got nil")
		}
	})
}

func TestMockServicesImplementInterfaces(t *testing.T) {
	// Ensure mock services implement their respective interfaces
	var _ service.TenantService = &mockTenantService{}
	var _ service.TemplateService = &mockTemplateService{}
	var _ service.TaskService = &mockTaskService{}
}
