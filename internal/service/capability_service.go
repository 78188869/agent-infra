// Package service provides business logic implementations for the application.
package service

import (
	"context"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Valid capability types.
var validCapabilityTypes = map[model.CapabilityType]bool{
	model.CapabilityTypeTool:         true,
	model.CapabilityTypeSkill:        true,
	model.CapabilityTypeAgentRuntime: true,
}

// Valid permission levels.
var validPermissionLevels = map[model.PermissionLevel]bool{
	model.PermissionLevelPublic:     true,
	model.PermissionLevelRestricted: true,
	model.PermissionLevelAdminOnly:  true,
}

// CreateCapabilityRequest represents the request to create a new capability.
type CreateCapabilityRequest struct {
	Type            model.CapabilityType `json:"type" binding:"required"`
	Name            string               `json:"name" binding:"required"`
	Description     string               `json:"description"`
	Version         string               `json:"version"`
	TenantID        string               `json:"tenant_id,omitempty"`
	PermissionLevel model.PermissionLevel `json:"permission_level" binding:"required"`
	Config          datatypes.JSON       `json:"config"`
	Schema          datatypes.JSON       `json:"schema"`
}

// UpdateCapabilityRequest represents the request to update an existing capability.
type UpdateCapabilityRequest struct {
	Name            *string                `json:"name"`
	Description     *string                `json:"description"`
	Version         *string                `json:"version"`
	PermissionLevel *model.PermissionLevel `json:"permission_level"`
	Config          *datatypes.JSON        `json:"config"`
	Schema          *datatypes.JSON        `json:"schema"`
}

// CapabilityFilter represents filtering options for listing capabilities.
type CapabilityFilter struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TenantID  string `form:"tenant_id"`
	Type      string `form:"type"`
	Status    string `form:"status"`
	Search    string `form:"search"`
}

// CapabilityService defines the interface for capability business operations.
type CapabilityService interface {
	Create(ctx context.Context, req *CreateCapabilityRequest) (*model.Capability, error)
	GetByID(ctx context.Context, id string) (*model.Capability, error)
	List(ctx context.Context, filter *CapabilityFilter) ([]*model.Capability, int64, error)
	Update(ctx context.Context, id string, req *UpdateCapabilityRequest) error
	Delete(ctx context.Context, id string) error
	Activate(ctx context.Context, id string) error
	Deactivate(ctx context.Context, id string) error
}

// capabilityService implements CapabilityService.
type capabilityService struct {
	repo repository.CapabilityRepository
}

// NewCapabilityService creates a new CapabilityService instance.
func NewCapabilityService(repo repository.CapabilityRepository) CapabilityService {
	return &capabilityService{repo: repo}
}

// Create creates a new capability with validation.
func (s *capabilityService) Create(ctx context.Context, req *CreateCapabilityRequest) (*model.Capability, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewBadRequestError("capability name is required")
	}

	// Validate capability type
	if !validCapabilityTypes[req.Type] {
		return nil, errors.NewBadRequestError("invalid type, must be one of: tool, skill, agent_runtime")
	}

	// Validate permission level
	if !validPermissionLevels[req.PermissionLevel] {
		return nil, errors.NewBadRequestError("invalid permission_level, must be one of: public, restricted, admin_only")
	}

	// Validate tenant_id format if provided
	var tenantID *string
	if req.TenantID != "" {
		parsedTenantID, err := uuid.Parse(req.TenantID)
		if err != nil {
			return nil, errors.NewBadRequestError("invalid tenant_id format")
		}
		tenantIDStr := parsedTenantID.String()
		tenantID = &tenantIDStr
	}

	// Set default version
	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	// Create capability model
	capability := &model.Capability{
		Type:            req.Type,
		Name:            req.Name,
		Description:     req.Description,
		Version:         version,
		TenantID:        tenantID,
		PermissionLevel: req.PermissionLevel,
		Config:          req.Config,
		Schema:          req.Schema,
		Status:          model.CapabilityStatusActive,
	}

	// Call repository
	if err := s.repo.Create(ctx, capability); err != nil {
		return nil, err
	}

	return capability, nil
}

// GetByID retrieves a capability by its ID.
func (s *capabilityService) GetByID(ctx context.Context, id string) (*model.Capability, error) {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid capability ID format")
	}

	// Call repository
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return nil, err
	}

	return capability, nil
}

// List retrieves capabilities based on filter criteria.
func (s *capabilityService) List(ctx context.Context, filter *CapabilityFilter) ([]*model.Capability, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.CapabilityFilter{
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		TenantID:  filter.TenantID,
		Type:      filter.Type,
		Status:    filter.Status,
		Search:    filter.Search,
	}

	// Call repository
	capabilities, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return capabilities, total, nil
}

// Update updates an existing capability with partial update support.
func (s *capabilityService) Update(ctx context.Context, id string, req *UpdateCapabilityRequest) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	// Validate and apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return errors.NewBadRequestError("capability name cannot be empty")
		}
		capability.Name = *req.Name
	}

	if req.Description != nil {
		capability.Description = *req.Description
	}

	if req.Version != nil {
		capability.Version = *req.Version
	}

	if req.PermissionLevel != nil {
		if !validPermissionLevels[*req.PermissionLevel] {
			return errors.NewBadRequestError("invalid permission_level, must be one of: public, restricted, admin_only")
		}
		capability.PermissionLevel = *req.PermissionLevel
	}

	if req.Config != nil {
		capability.Config = *req.Config
	}

	if req.Schema != nil {
		capability.Schema = *req.Schema
	}

	// Call repository
	return s.repo.Update(ctx, capability)
}

// Delete performs a soft delete on a capability.
func (s *capabilityService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability to ensure it exists
	_, err = s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	// Call repository
	return s.repo.Delete(ctx, capabilityID)
}

// Activate activates a capability.
func (s *capabilityService) Activate(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	// Set status to active
	capability.Status = model.CapabilityStatusActive

	// Call repository
	return s.repo.Update(ctx, capability)
}

// Deactivate deactivates a capability.
func (s *capabilityService) Deactivate(ctx context.Context, id string) error {
	// Parse and validate ID
	capabilityID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid capability ID format")
	}

	// Get existing capability
	capability, err := s.repo.GetByID(ctx, capabilityID)
	if err != nil {
		return err
	}

	// Set status to inactive
	capability.Status = model.CapabilityStatusInactive

	// Call repository
	return s.repo.Update(ctx, capability)
}
