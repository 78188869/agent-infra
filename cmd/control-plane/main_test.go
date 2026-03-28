package main

import (
	"context"
	"testing"

	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/service"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantErr  bool
		checkVal func(t *testing.T, cfg *config.AppConfig)
	}{
		{
			name:    "load valid config",
			path:    "../../configs/config.yaml",
			wantErr: false,
			checkVal: func(t *testing.T, cfg *config.AppConfig) {
				if cfg.Server.Port != 8080 {
					t.Errorf("expected port 8080, got %d", cfg.Server.Port)
				}
				if cfg.Database.Host == "" {
					t.Error("expected non-empty database host")
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
			cfg, err := config.Load(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("config.Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkVal != nil {
				tt.checkVal(t, cfg)
			}
		})
	}
}

func TestConfigDatabaseFields(t *testing.T) {
	cfg := &config.AppConfig{}
	cfg.Database.Host = "testhost"
	cfg.Database.Port = 3307
	cfg.Database.Database = "testdb"
	cfg.Database.Username = "testuser"
	cfg.Database.Password = "testpass"

	if cfg.Database.Host != "testhost" {
		t.Errorf("expected host testhost, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("expected port 3307, got %d", cfg.Database.Port)
	}
	if cfg.Database.Database != "testdb" {
		t.Errorf("expected database testdb, got %s", cfg.Database.Database)
	}
	if cfg.Database.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", cfg.Database.Username)
	}
	if cfg.Database.Password != "testpass" {
		t.Errorf("expected password testpass, got %s", cfg.Database.Password)
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
