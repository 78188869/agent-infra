package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/example/agent-infra/internal/api/router"
	"github.com/example/agent-infra/internal/config"
	"github.com/example/agent-infra/internal/monitoring"
	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/internal/service"
	"github.com/example/agent-infra/pkg/aliyun/sls"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	env := cfg.GetEnvironment()
	// Initialize structured logger (file output in local env, stdout in production)
	logger := monitoring.NewLogger(cfg)
	slog.SetDefault(logger)
	slog.Info("starting control-plane", "env", env)

	gin.SetMode(cfg.Server.Mode)

	var tenantSvc service.TenantService
	var templateSvc service.TemplateService
	var taskSvc service.TaskService
	var providerSvc service.ProviderService
	var capabilitySvc service.CapabilityService
	var interventionSvc service.InterventionService
	var monitoringSvc service.MonitoringService
	var db *config.Database
	db, err = config.NewDatabase(cfg.Database)
	if err != nil {
		slog.Warn("failed to connect to database, using mock service", "error", err)
		tenantSvc = &mockTenantService{}
		templateSvc = &mockTemplateService{}
		taskSvc = &mockTaskService{}
		providerSvc = &mockProviderService{}
		capabilitySvc = &mockCapabilityService{}
		interventionSvc = &mockInterventionService{}
		monitoringSvc = &mockMonitoringService{}
	} else {
		if err := db.AutoMigrate(&model.Tenant{}, &model.Template{}, &model.Task{}, &model.Provider{}, &model.Capability{}, &model.Intervention{}); err != nil {
			slog.Warn("failed to auto-migrate", "error", err)
		}
		tenantRepo := repository.NewTenantRepository(db.DB)
		tenantSvc = service.NewTenantService(tenantRepo)

		templateRepo := repository.NewTemplateRepository(db.DB)
		templateSvc = service.NewTemplateService(templateRepo)

		taskRepo := repository.NewTaskRepository(db.DB)
		taskSvc = service.NewTaskService(taskRepo)

		providerRepo := repository.NewProviderRepository(db.DB)
		providerSvc = service.NewProviderService(providerRepo)

		capabilityRepo := repository.NewCapabilityRepository(db.DB)
		capabilitySvc = service.NewCapabilityService(capabilityRepo)

		interventionRepo := repository.NewInterventionRepository(db.DB)
		interventionSvc = service.NewInterventionService(taskRepo, interventionRepo)
	}

	monitoringHub := monitoring.NewHub()
	slsClient := monitoring.NewSLSClient(sls.Config{
		Endpoint:        cfg.SLS.Endpoint,
		AccessKeyID:     cfg.SLS.AccessKey,
		AccessKeySecret: cfg.SLS.AccessSecret,
		Project:         cfg.SLS.Project,
		LogStore:        cfg.SLS.Logstore,
	})
	monitoringSvc = service.NewMonitoringService(monitoringHub, slsClient)

	r := router.Setup(tenantSvc, templateSvc, taskSvc, providerSvc, capabilitySvc, monitoringSvc, monitoringHub, interventionSvc, db)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	slog.Info("starting server", "addr", addr, "env", env)
	if err := r.Run(addr); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
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

// mockMonitoringService is a fallback monitoring service when components are not initialized.
type mockMonitoringService struct{}

func (m *mockMonitoringService) RecordTaskStatusChange(ctx context.Context, taskID, tenantID, oldStatus, newStatus string) error {
	return nil
}

func (m *mockMonitoringService) RecordLogEntry(ctx context.Context, taskID, tenantID string, eventType model.EventType, eventName string, content interface{}) error {
	return nil
}

func (m *mockMonitoringService) RecordTaskProgress(ctx context.Context, taskID, tenantID string, progress int64, tokensUsed int64, elapsedSecs int64) error {
	return nil
}

func (m *mockMonitoringService) BroadcastTaskCompletion(ctx context.Context, taskID, tenantID string) error {
	return nil
}
