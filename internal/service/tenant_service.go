// Package service provides business logic implementations for the application.
package service

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// CreateTenantRequest represents the request to create a new tenant.
type CreateTenantRequest struct {
	Name             string `json:"name" binding:"required"`
	QuotaCPU         int    `json:"quota_cpu"`
	QuotaMemory      int64  `json:"quota_memory"`
	QuotaConcurrency int    `json:"quota_concurrency"`
	QuotaDailyTasks  int    `json:"quota_daily_tasks"`
}

// UpdateTenantRequest represents the request to update an existing tenant.
type UpdateTenantRequest struct {
	Name             *string `json:"name"`
	QuotaCPU         *int    `json:"quota_cpu"`
	QuotaMemory      *int64  `json:"quota_memory"`
	QuotaConcurrency *int    `json:"quota_concurrency"`
	QuotaDailyTasks  *int    `json:"quota_daily_tasks"`
	Status           *string `json:"status"`
}

// TenantFilter represents filtering options for listing tenants.
type TenantFilter struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Status   string `form:"status"`
	Search   string `form:"search"`
}

// TenantService defines the interface for tenant business operations.
type TenantService interface {
	Create(ctx context.Context, req *CreateTenantRequest) (*model.Tenant, error)
	GetByID(ctx context.Context, id string) (*model.Tenant, error)
	List(ctx context.Context, filter *TenantFilter) ([]*model.Tenant, int64, error)
	Update(ctx context.Context, id string, req *UpdateTenantRequest) error
	Delete(ctx context.Context, id string) error
}

// tenantService implements TenantService.
type tenantService struct {
	repo repository.TenantRepository
}

// NewTenantService creates a new TenantService instance.
func NewTenantService(repo repository.TenantRepository) TenantService {
	return &tenantService{repo: repo}
}

// Create creates a new tenant with validation.
func (s *tenantService) Create(ctx context.Context, req *CreateTenantRequest) (*model.Tenant, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewBadRequestError("tenant name is required")
	}

	// Validate quota limits (must be positive values)
	if req.QuotaCPU < 0 {
		return nil, errors.NewBadRequestError("quota_cpu must be a positive value")
	}
	if req.QuotaMemory < 0 {
		return nil, errors.NewBadRequestError("quota_memory must be a positive value")
	}
	if req.QuotaConcurrency < 0 {
		return nil, errors.NewBadRequestError("quota_concurrency must be a positive value")
	}
	if req.QuotaDailyTasks < 0 {
		return nil, errors.NewBadRequestError("quota_daily_tasks must be a positive value")
	}

	// Create tenant model
	tenant := &model.Tenant{
		Name:             req.Name,
		QuotaCPU:         req.QuotaCPU,
		QuotaMemory:      req.QuotaMemory,
		QuotaConcurrency: req.QuotaConcurrency,
		QuotaDailyTasks:  req.QuotaDailyTasks,
		Status:           model.TenantStatusActive,
	}

	// Call repository
	if err := s.repo.Create(ctx, tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

// GetByID retrieves a tenant by its ID.
func (s *tenantService) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	// Parse and validate ID
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid tenant ID format")
	}

	// Call repository
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

// List retrieves tenants based on filter criteria.
func (s *tenantService) List(ctx context.Context, filter *TenantFilter) ([]*model.Tenant, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.TenantFilter{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Status:   filter.Status,
		Search:   filter.Search,
	}

	// Call repository
	tenants, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

// Update updates an existing tenant with partial update support.
func (s *tenantService) Update(ctx context.Context, id string, req *UpdateTenantRequest) error {
	// Parse and validate ID
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid tenant ID format")
	}

	// Get existing tenant
	tenant, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}

	// Validate and apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return errors.NewBadRequestError("tenant name cannot be empty")
		}
		tenant.Name = *req.Name
	}

	if req.QuotaCPU != nil {
		if *req.QuotaCPU < 0 {
			return errors.NewBadRequestError("quota_cpu must be a positive value")
		}
		tenant.QuotaCPU = *req.QuotaCPU
	}

	if req.QuotaMemory != nil {
		if *req.QuotaMemory < 0 {
			return errors.NewBadRequestError("quota_memory must be a positive value")
		}
		tenant.QuotaMemory = *req.QuotaMemory
	}

	if req.QuotaConcurrency != nil {
		if *req.QuotaConcurrency < 0 {
			return errors.NewBadRequestError("quota_concurrency must be a positive value")
		}
		tenant.QuotaConcurrency = *req.QuotaConcurrency
	}

	if req.QuotaDailyTasks != nil {
		if *req.QuotaDailyTasks < 0 {
			return errors.NewBadRequestError("quota_daily_tasks must be a positive value")
		}
		tenant.QuotaDailyTasks = *req.QuotaDailyTasks
	}

	if req.Status != nil {
		if *req.Status != model.TenantStatusActive && *req.Status != model.TenantStatusSuspended {
			return errors.NewBadRequestError("invalid status value, must be 'active' or 'suspended'")
		}
		tenant.Status = *req.Status
	}

	// Call repository
	return s.repo.Update(ctx, tenant)
}

// Delete performs a soft delete on a tenant.
func (s *tenantService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	tenantID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid tenant ID format")
	}

	// Call repository
	return s.repo.Delete(ctx, tenantID)
}
