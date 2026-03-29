package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/internal/service"
)

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

// mockTemplateService is a fallback service when database is not available.
type mockTemplateService struct{}

func (m *mockTemplateService) Create(ctx context.Context, req *service.CreateTemplateRequest) (*model.Template, error) {
	return &model.Template{Name: req.Name, Status: model.TemplateStatusDraft}, nil
}

func (m *mockTemplateService) GetByID(ctx context.Context, id string) (*model.Template, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockTemplateService) List(ctx context.Context, filter *service.TemplateFilter) ([]*model.Template, int64, error) {
	return []*model.Template{}, 0, nil
}

func (m *mockTemplateService) Update(ctx context.Context, id string, req *service.UpdateTemplateRequest) error {
	return fmt.Errorf("database not available")
}

func (m *mockTemplateService) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

// mockTaskService is a fallback service when database is not available.
type mockTaskService struct{}

func (m *mockTaskService) Create(ctx context.Context, req *service.CreateTaskRequest) (*model.Task, error) {
	return &model.Task{Name: req.Name, Status: model.TaskStatusPending}, nil
}

func (m *mockTaskService) GetByID(ctx context.Context, id string) (*model.Task, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockTaskService) List(ctx context.Context, filter *service.TaskFilter) ([]*model.Task, int64, error) {
	return []*model.Task{}, 0, nil
}

func (m *mockTaskService) Update(ctx context.Context, id string, req *service.UpdateTaskRequest) error {
	return fmt.Errorf("database not available")
}

func (m *mockTaskService) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

func (m *mockTaskService) UpdateStatus(ctx context.Context, id string, status string, message string) error {
	return fmt.Errorf("database not available")
}

// mockProviderService is a fallback service when database is not available.
type mockProviderService struct{}

func (m *mockProviderService) Create(ctx context.Context, req *service.CreateProviderRequest) (*model.Provider, error) {
	return &model.Provider{Name: req.Name, Status: model.ProviderStatusActive}, nil
}

func (m *mockProviderService) GetByID(ctx context.Context, id string) (*model.Provider, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockProviderService) List(ctx context.Context, filter *repository.ProviderFilter) ([]*model.Provider, int64, error) {
	return []*model.Provider{}, 0, nil
}

func (m *mockProviderService) Update(ctx context.Context, id string, req *service.UpdateProviderRequest) error {
	return fmt.Errorf("database not available")
}

func (m *mockProviderService) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

func (m *mockProviderService) TestConnection(ctx context.Context, id string) (*service.ConnectionTestResult, error) {
	return &service.ConnectionTestResult{Success: false, Message: "database not available"}, nil
}

func (m *mockProviderService) GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error) {
	return []*model.Provider{}, nil
}

func (m *mockProviderService) ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockProviderService) SetDefaultProvider(ctx context.Context, userID, providerID string) error {
	return fmt.Errorf("database not available")
}

// mockCapabilityService is a fallback service when database is not available.
type mockCapabilityService struct{}

func (m *mockCapabilityService) Create(ctx context.Context, req *service.CreateCapabilityRequest) (*model.Capability, error) {
	return &model.Capability{Name: req.Name, Status: model.CapabilityStatusActive}, nil
}

func (m *mockCapabilityService) GetByID(ctx context.Context, id string) (*model.Capability, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockCapabilityService) List(ctx context.Context, filter *service.CapabilityFilter) ([]*model.Capability, int64, error) {
	return []*model.Capability{}, 0, nil
}

func (m *mockCapabilityService) Update(ctx context.Context, id string, req *service.UpdateCapabilityRequest) error {
	return fmt.Errorf("database not available")
}

func (m *mockCapabilityService) Delete(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

func (m *mockCapabilityService) Activate(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

func (m *mockCapabilityService) Deactivate(ctx context.Context, id string) error {
	return fmt.Errorf("database not available")
}

// mockInterventionService is a fallback service when database is not available.
type mockInterventionService struct{}

func (m *mockInterventionService) Pause(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockInterventionService) Resume(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockInterventionService) Cancel(ctx context.Context, taskID, operatorID, reason string) (*model.Intervention, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockInterventionService) Inject(ctx context.Context, req *service.InjectInterventionRequest) (*model.Intervention, error) {
	return nil, fmt.Errorf("database not available")
}

func (m *mockInterventionService) ListInterventions(ctx context.Context, taskID string, filter *service.InterventionFilter) ([]*model.Intervention, int64, error) {
	return []*model.Intervention{}, 0, nil
}

func (m *mockInterventionService) HandleWrapperEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	return nil
}

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

	t.Run("update status returns error", func(t *testing.T) {
		err := mock.UpdateStatus(ctx, "123", "running", "test")
		if err == nil {
			t.Error("expected error for UpdateStatus, got nil")
		}
	})
}

func TestMockServicesImplementInterfaces(t *testing.T) {
	// Ensure mock services implement their respective interfaces
	var _ service.TenantService = &mockTenantService{}
	var _ service.TemplateService = &mockTemplateService{}
	var _ service.TaskService = &mockTaskService{}
	var _ service.ProviderService = &mockProviderService{}
	var _ service.CapabilityService = &mockCapabilityService{}
	var _ service.InterventionService = &mockInterventionService{}
}
