// Package service provides business logic implementations for the application.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// CreateProviderRequest represents the request to create a new provider.
type CreateProviderRequest struct {
	Name           string                `json:"name" binding:"required"`
	Type           model.ProviderType    `json:"type" binding:"required"`
	Scope          model.ProviderScope   `json:"scope" binding:"required"`
	TenantID       *string               `json:"tenant_id"`
	UserID         *string               `json:"user_id"`
	Description    string                `json:"description"`
	APIEndpoint    string                `json:"api_endpoint"`
	APIKeyRef      string                `json:"api_key_ref"`
	ModelMapping   interface{}           `json:"model_mapping"`
	RuntimeType    model.RuntimeType     `json:"runtime_type"`
	RuntimeImage   string                `json:"runtime_image"`
	RuntimeCommand interface{}           `json:"runtime_command"`
	EnvVars        interface{}           `json:"env_vars"`
	Permissions    interface{}           `json:"permissions"`
	EnabledPlugins interface{}           `json:"enabled_plugins"`
	ExtraParams    interface{}           `json:"extra_params"`
}

// UpdateProviderRequest represents the request to update an existing provider.
type UpdateProviderRequest struct {
	Name           *string             `json:"name"`
	Description    *string             `json:"description"`
	APIEndpoint    *string             `json:"api_endpoint"`
	APIKeyRef      *string             `json:"api_key_ref"`
	ModelMapping   interface{}         `json:"model_mapping"`
	RuntimeType    *model.RuntimeType  `json:"runtime_type"`
	RuntimeImage   *string             `json:"runtime_image"`
	RuntimeCommand interface{}         `json:"runtime_command"`
	EnvVars        interface{}         `json:"env_vars"`
	Permissions    interface{}         `json:"permissions"`
	EnabledPlugins interface{}         `json:"enabled_plugins"`
	ExtraParams    interface{}         `json:"extra_params"`
	Status         *model.ProviderStatus `json:"status"`
}

// ConnectionTestResult represents the result of a provider connection test.
type ConnectionTestResult struct {
	Success      bool  `json:"success"`
	Message      string `json:"message"`
	ResponseTime int64 `json:"response_time_ms"`
}

// ProviderService defines the interface for provider business operations.
type ProviderService interface {
	Create(ctx context.Context, req *CreateProviderRequest) (*model.Provider, error)
	GetByID(ctx context.Context, id string) (*model.Provider, error)
	List(ctx context.Context, filter *repository.ProviderFilter) ([]*model.Provider, int64, error)
	Update(ctx context.Context, id string, req *UpdateProviderRequest) error
	Delete(ctx context.Context, id string) error

	// Provider-specific operations
	TestConnection(ctx context.Context, id string) (*ConnectionTestResult, error)
	GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error)
	ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error)
	SetDefaultProvider(ctx context.Context, userID, providerID string) error
}

// providerService implements ProviderService.
type providerService struct {
	repo repository.ProviderRepository
}

// NewProviderService creates a new ProviderService instance.
func NewProviderService(repo repository.ProviderRepository) ProviderService {
	return &providerService{repo: repo}
}

// Create creates a new provider with validation.
func (s *providerService) Create(ctx context.Context, req *CreateProviderRequest) (*model.Provider, error) {
	// Validate required fields
	if req.Name == "" {
		return nil, errors.NewBadRequestError("provider name is required")
	}

	// Validate scope
	validScopes := map[model.ProviderScope]bool{
		model.ProviderScopeSystem: true,
		model.ProviderScopeTenant: true,
		model.ProviderScopeUser:   true,
	}
	if !validScopes[req.Scope] {
		return nil, errors.NewBadRequestError("invalid scope, must be 'system', 'tenant', or 'user'")
	}

	// Validate scope-specific requirements
	if req.Scope == model.ProviderScopeTenant && (req.TenantID == nil || *req.TenantID == "") {
		return nil, errors.NewBadRequestError("tenant_id is required for tenant scope")
	}
	if req.Scope == model.ProviderScopeUser && (req.UserID == nil || *req.UserID == "") {
		return nil, errors.NewBadRequestError("user_id is required for user scope")
	}

	// Set default runtime type if not specified
	runtimeType := req.RuntimeType
	if runtimeType == "" {
		runtimeType = model.RuntimeTypeCLI
	}

	// Create provider model
	provider := &model.Provider{
		Name:           req.Name,
		Type:           req.Type,
		Scope:          req.Scope,
		TenantID:       req.TenantID,
		UserID:         req.UserID,
		Description:    req.Description,
		APIEndpoint:    req.APIEndpoint,
		APIKeyRef:      req.APIKeyRef,
		RuntimeType:    runtimeType,
		RuntimeImage:   req.RuntimeImage,
		Status:         model.ProviderStatusActive,
	}

	// Handle JSON fields
	if req.ModelMapping != nil {
		provider.ModelMapping = marshalToJSON(req.ModelMapping)
	}
	if req.RuntimeCommand != nil {
		provider.RuntimeCommand = marshalToJSON(req.RuntimeCommand)
	}
	if req.EnvVars != nil {
		provider.EnvVars = marshalToJSON(req.EnvVars)
	}
	if req.Permissions != nil {
		provider.Permissions = marshalToJSON(req.Permissions)
	}
	if req.EnabledPlugins != nil {
		provider.EnabledPlugins = marshalToJSON(req.EnabledPlugins)
	}
	if req.ExtraParams != nil {
		provider.ExtraParams = marshalToJSON(req.ExtraParams)
	}

	// Call repository
	if err := s.repo.Create(ctx, provider); err != nil {
		return nil, err
	}

	return provider, nil
}

// GetByID retrieves a provider by its ID.
func (s *providerService) GetByID(ctx context.Context, id string) (*model.Provider, error) {
	// Parse and validate ID
	providerID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid provider ID format")
	}

	// Call repository
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

// List retrieves providers based on filter criteria.
func (s *providerService) List(ctx context.Context, filter *repository.ProviderFilter) ([]*model.Provider, int64, error) {
	// Call repository
	providers, total, err := s.repo.List(ctx, *filter)
	if err != nil {
		return nil, 0, err
	}

	return providers, total, nil
}

// Update updates an existing provider with partial update support.
func (s *providerService) Update(ctx context.Context, id string, req *UpdateProviderRequest) error {
	// Parse and validate ID
	providerID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid provider ID format")
	}

	// Get existing provider
	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}

	// Validate and apply updates
	if req.Name != nil {
		if *req.Name == "" {
			return errors.NewBadRequestError("provider name cannot be empty")
		}
		provider.Name = *req.Name
	}

	if req.Description != nil {
		provider.Description = *req.Description
	}

	if req.APIEndpoint != nil {
		provider.APIEndpoint = *req.APIEndpoint
	}

	if req.APIKeyRef != nil {
		provider.APIKeyRef = *req.APIKeyRef
	}

	if req.RuntimeType != nil {
		provider.RuntimeType = *req.RuntimeType
	}

	if req.RuntimeImage != nil {
		provider.RuntimeImage = *req.RuntimeImage
	}

	if req.Status != nil {
		validStatuses := map[model.ProviderStatus]bool{
			model.ProviderStatusActive:     true,
			model.ProviderStatusInactive:   true,
			model.ProviderStatusDeprecated: true,
		}
		if !validStatuses[*req.Status] {
			return errors.NewBadRequestError("invalid status value, must be 'active', 'inactive', or 'deprecated'")
		}
		provider.Status = *req.Status
	}

	// Handle JSON fields
	if req.ModelMapping != nil {
		provider.ModelMapping = marshalToJSON(req.ModelMapping)
	}
	if req.RuntimeCommand != nil {
		provider.RuntimeCommand = marshalToJSON(req.RuntimeCommand)
	}
	if req.EnvVars != nil {
		provider.EnvVars = marshalToJSON(req.EnvVars)
	}
	if req.Permissions != nil {
		provider.Permissions = marshalToJSON(req.Permissions)
	}
	if req.EnabledPlugins != nil {
		provider.EnabledPlugins = marshalToJSON(req.EnabledPlugins)
	}
	if req.ExtraParams != nil {
		provider.ExtraParams = marshalToJSON(req.ExtraParams)
	}

	// Call repository
	return s.repo.Update(ctx, provider)
}

// Delete performs a soft delete on a provider.
func (s *providerService) Delete(ctx context.Context, id string) error {
	// Parse and validate ID
	providerID, err := uuid.Parse(id)
	if err != nil {
		return errors.NewBadRequestError("invalid provider ID format")
	}

	// Call repository
	return s.repo.Delete(ctx, providerID)
}

// TestConnection tests the provider connection by making a simple API call.
func (s *providerService) TestConnection(ctx context.Context, id string) (*ConnectionTestResult, error) {
	providerID, err := uuid.Parse(id)
	if err != nil {
		return nil, errors.NewBadRequestError("invalid provider ID format")
	}

	provider, err := s.repo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	if provider.APIEndpoint == "" {
		return &ConnectionTestResult{
			Success: false,
			Message: "API endpoint not configured",
		}, nil
	}

	// Make HTTP request to test connection
	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", provider.APIEndpoint+"/v1/models", nil)
	if err != nil {
		return &ConnectionTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}, nil
	}

	resp, err := client.Do(req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return &ConnectionTestResult{
			Success:      false,
			Message:      fmt.Sprintf("Connection failed: %v", err),
			ResponseTime: elapsed,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &ConnectionTestResult{
			Success:      true,
			Message:      "Connection successful",
			ResponseTime: elapsed,
		}, nil
	}

	return &ConnectionTestResult{
		Success:      false,
		Message:      fmt.Sprintf("API returned status %d", resp.StatusCode),
		ResponseTime: elapsed,
	}, nil
}

// GetAvailableProviders retrieves all providers available to a user.
// This includes system-level providers, tenant-level providers, and user-level providers.
func (s *providerService) GetAvailableProviders(ctx context.Context, tenantID, userID string) ([]*model.Provider, error) {
	var allProviders []*model.Provider

	// Get system-level providers (available to all)
	systemProviders, _, err := s.repo.List(ctx, repository.ProviderFilter{
		Scope:   string(model.ProviderScopeSystem),
		Status:  string(model.ProviderStatusActive),
		Page:    1,
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}
	allProviders = append(allProviders, systemProviders...)

	// Get tenant-level providers (if tenantID provided)
	if tenantID != "" {
		tenantProviders, _, err := s.repo.List(ctx, repository.ProviderFilter{
			Scope:    string(model.ProviderScopeTenant),
			TenantID: tenantID,
			Status:   string(model.ProviderStatusActive),
			Page:     1,
			PageSize: 100,
		})
		if err != nil {
			return nil, err
		}
		allProviders = append(allProviders, tenantProviders...)
	}

	// Get user-level providers (if userID provided)
	if userID != "" {
		userProviders, _, err := s.repo.List(ctx, repository.ProviderFilter{
			Scope:   string(model.ProviderScopeUser),
			UserID:  userID,
			Status:  string(model.ProviderStatusActive),
			Page:    1,
			PageSize: 100,
		})
		if err != nil {
			return nil, err
		}
		allProviders = append(allProviders, userProviders...)
	}

	return allProviders, nil
}

// ResolveProvider resolves the provider based on priority chain.
// Priority: specified > user_default > tenant_default > system_default
func (s *providerService) ResolveProvider(ctx context.Context, specifiedProviderID, tenantID, userID string) (*model.Provider, error) {
	// 1. If specified, use it (if active)
	if specifiedProviderID != "" {
		providerID, err := uuid.Parse(specifiedProviderID)
		if err == nil {
			provider, err := s.repo.GetByID(ctx, providerID)
			if err == nil && provider.IsActive() {
				return provider, nil
			}
		}
	}

	// 2. Try user default
	if userID != "" {
		provider, err := s.repo.GetUserDefaultProvider(ctx, userID)
		if err == nil && provider != nil && provider.IsActive() {
			return provider, nil
		}
	}

	// 3. Try tenant default
	if tenantID != "" {
		tid := tenantID
		provider, err := s.repo.GetDefaultProvider(ctx, model.ProviderScopeTenant, &tid, nil)
		if err == nil && provider != nil && provider.IsActive() {
			return provider, nil
		}
	}

	// 4. Fall back to system default
	provider, err := s.repo.GetDefaultProvider(ctx, model.ProviderScopeSystem, nil, nil)
	if err != nil {
		return nil, errors.NewNotFoundError("no available provider found")
	}
	return provider, nil
}

// SetDefaultProvider sets or updates the default provider for a user.
func (s *providerService) SetDefaultProvider(ctx context.Context, userID, providerID string) error {
	// Validate provider ID
	pid, err := uuid.Parse(providerID)
	if err != nil {
		return errors.NewBadRequestError("invalid provider ID format")
	}

	// Verify provider exists
	_, err = s.repo.GetByID(ctx, pid)
	if err != nil {
		return err
	}

	// Set user default
	return s.repo.SetUserDefaultProvider(ctx, userID, providerID)
}

// marshalToJSON converts an interface to datatypes.JSON.
// Returns nil if the input is nil or cannot be marshaled.
func marshalToJSON(v interface{}) datatypes.JSON {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case datatypes.JSON:
		return val
	case json.RawMessage:
		return datatypes.JSON(val)
	case []byte:
		return datatypes.JSON(val)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return datatypes.JSON(data)
	}
}
