// Package service provides business logic implementations for the application.
package service

import (
	"context"
	"strings"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// Valid scene types for templates.
var validSceneTypes = map[string]bool{
	model.TemplateSceneTypeCoding:   true,
	model.TemplateSceneTypeOps:      true,
	model.TemplateSceneTypeAnalysis: true,
	model.TemplateSceneTypeContent:  true,
	model.TemplateSceneTypeCustom:   true,
}

// Valid status values for templates.
var validTemplateStatuses = map[string]bool{
	model.TemplateStatusDraft:      true,
	model.TemplateStatusPublished:  true,
	model.TemplateStatusDeprecated: true,
}

// CreateTemplateRequest represents the request to create a new template.
type CreateTemplateRequest struct {
	TenantID   string  `json:"tenant_id" binding:"required"`
	Name       string  `json:"name" binding:"required"`
	Version    string  `json:"version"`
	Spec       string  `json:"spec"`
	SceneType  string  `json:"scene_type"`
	ProviderID *string `json:"provider_id,omitempty"`
}

// UpdateTemplateRequest represents the request to update an existing template.
type UpdateTemplateRequest struct {
	Name       *string `json:"name"`
	Version    *string `json:"version"`
	Spec       *string `json:"spec"`
	SceneType  *string `json:"scene_type"`
	Status     *string `json:"status"`
	ProviderID *string `json:"provider_id,omitempty"`
}

// TemplateFilter represents filtering options for listing templates.
type TemplateFilter struct {
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
	TenantID  string `form:"tenant_id"`
	Status    string `form:"status"`
	SceneType string `form:"scene_type"`
	Search    string `form:"search"`
}

// TemplateService defines the interface for template business operations.
type TemplateService interface {
	Create(ctx context.Context, req *CreateTemplateRequest) (*model.Template, error)
	GetByID(ctx context.Context, id string) (*model.Template, error)
	List(ctx context.Context, filter *TemplateFilter) ([]*model.Template, int64, error)
	Update(ctx context.Context, id string, req *UpdateTemplateRequest) error
	Delete(ctx context.Context, id string) error
}

// templateService implements TemplateService.
type templateService struct {
	repo repository.TemplateRepository
}

// NewTemplateService creates a new TemplateService instance.
func NewTemplateService(repo repository.TemplateRepository) TemplateService {
	return &templateService{repo: repo}
}

// Create creates a new template with validation.
func (s *templateService) Create(ctx context.Context, req *CreateTemplateRequest) (*model.Template, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewBadRequestError("template name is required")
	}

	if req.TenantID == "" {
		return nil, errors.NewBadRequestError("tenant_id is required")
	}

	// Validate tenant_id format
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid tenant_id format")
	}

	// Validate scene_type
	sceneType := req.SceneType
	if sceneType == "" {
		sceneType = model.TemplateSceneTypeCustom
	}
	if !validSceneTypes[sceneType] {
		return nil, errors.NewBadRequestError("invalid scene_type, must be one of: coding, ops, analysis, content, custom")
	}

	// Validate YAML spec format (basic validation)
	if req.Spec != "" {
		if err := validateYAMLSpec(req.Spec); err != nil {
			return nil, errors.NewBadRequestError("invalid YAML spec format: " + err.Error())
		}
	}

	// Set default version
	version := req.Version
	if version == "" {
		version = "1.0.0"
	}

	// Create template model
	template := &model.Template{
		TenantID:   tenantID.String(),
		Name:       req.Name,
		Version:    version,
		Spec:       req.Spec,
		SceneType:  sceneType,
		Status:     model.TemplateStatusDraft,
		ProviderID: req.ProviderID,
	}

	// Call repository
	if err := s.repo.Create(ctx, template); err != nil {
		return nil, err
	}

	return template, nil
}

// GetByID retrieves a template by its ID.
func (s *templateService) GetByID(ctx context.Context, id string) (*model.Template, error) {
	// Parse and validate ID
	templateID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid template ID format")
	}

	// Call repository
	template, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	return template, nil
}

// List retrieves templates based on filter criteria.
func (s *templateService) List(ctx context.Context, filter *TemplateFilter) ([]*model.Template, int64, error) {
	// Convert service filter to repository filter
	repoFilter := repository.TemplateFilter{
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		TenantID:  filter.TenantID,
		Status:    filter.Status,
		SceneType: filter.SceneType,
		Search:    filter.Search,
	}

	// Call repository
	templates, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, err
	}

	return templates, total, nil
}

// Update updates an existing template with partial update support.
func (s *templateService) Update(ctx context.Context, id string, req *UpdateTemplateRequest) error {
	// Parse and validate ID
	templateID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid template ID format")
	}

	// Get existing template
	template, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}

	// Validate and apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return errors.NewBadRequestError("template name cannot be empty")
		}
		template.Name = *req.Name
	}

	if req.Version != nil {
		template.Version = *req.Version
	}

	if req.Spec != nil {
		if err := validateYAMLSpec(*req.Spec); err != nil {
			return errors.NewBadRequestError("invalid YAML spec format: " + err.Error())
		}
		template.Spec = *req.Spec
	}

	if req.SceneType != nil {
		if !validSceneTypes[*req.SceneType] {
			return errors.NewBadRequestError("invalid scene_type, must be one of: coding, ops, analysis, content, custom")
		}
		template.SceneType = *req.SceneType
	}

	if req.Status != nil {
		if !validTemplateStatuses[*req.Status] {
			return errors.NewBadRequestError("invalid status, must be one of: draft, published, deprecated")
		}
		template.Status = *req.Status
	}

	if req.ProviderID != nil {
		template.ProviderID = req.ProviderID
	}

	// Call repository
	return s.repo.Update(ctx, template)
}

// Delete performs a soft delete on a template.
// Business rule: only draft templates can be deleted.
func (s *templateService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	templateID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid template ID format")
	}

	// Get existing template to check status
	template, err := s.repo.GetByID(ctx, templateID)
	if err != nil {
		return err
	}

	// Business rule: only draft templates can be deleted
	if template.Status != model.TemplateStatusDraft {
		return errors.NewBadRequestError("only draft templates can be deleted")
	}

	// Call repository
	return s.repo.Delete(ctx, templateID)
}

// validateYAMLSpec performs basic YAML validation on the spec.
func validateYAMLSpec(spec string) error {
	// Trim whitespace
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}

	// Try to parse as YAML
	var result interface{}
	return yaml.Unmarshal([]byte(spec), &result)
}
