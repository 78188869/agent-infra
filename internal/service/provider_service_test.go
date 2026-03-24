package service

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockProviderRepository implements repository.ProviderRepository for testing
type mockProviderRepository struct {
	createFunc              func(ctx context.Context, provider *model.Provider) error
	getByIDFunc             func(ctx context.Context, id uuid.UUID) (*model.Provider, error)
	listFunc                func(ctx context.Context, filter repository.ProviderFilter) ([]*model.Provider, int64, error)
	updateFunc              func(ctx context.Context, provider *model.Provider) error
	deleteFunc              func(ctx context.Context, id uuid.UUID) error
	getByScopeAndNameFunc   func(ctx context.Context, scope model.ProviderScope, tenantID, userID *string, name string) (*model.Provider, error)
	getDefaultProviderFunc  func(ctx context.Context, scope model.ProviderScope, tenantID, userID *string) (*model.Provider, error)
	setUserDefaultFunc      func(ctx context.Context, userID, providerID string) error
	getUserDefaultFunc      func(ctx context.Context, userID string) (*model.Provider, error)
}

func (m *mockProviderRepository) Create(ctx context.Context, provider *model.Provider) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, provider)
	}
	return nil
}

func (m *mockProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockProviderRepository) List(ctx context.Context, filter repository.ProviderFilter) ([]*model.Provider, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockProviderRepository) Update(ctx context.Context, provider *model.Provider) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, provider)
	}
	return nil
}

func (m *mockProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockProviderRepository) GetByScopeAndName(ctx context.Context, scope model.ProviderScope, tenantID, userID *string, name string) (*model.Provider, error) {
	if m.getByScopeAndNameFunc != nil {
		return m.getByScopeAndNameFunc(ctx, scope, tenantID, userID, name)
	}
	return nil, nil
}

func (m *mockProviderRepository) GetDefaultProvider(ctx context.Context, scope model.ProviderScope, tenantID, userID *string) (*model.Provider, error) {
	if m.getDefaultProviderFunc != nil {
		return m.getDefaultProviderFunc(ctx, scope, tenantID, userID)
	}
	return nil, nil
}

func (m *mockProviderRepository) SetUserDefaultProvider(ctx context.Context, userID, providerID string) error {
	if m.setUserDefaultFunc != nil {
		return m.setUserDefaultFunc(ctx, userID, providerID)
	}
	return nil
}

func (m *mockProviderRepository) GetUserDefaultProvider(ctx context.Context, userID string) (*model.Provider, error) {
	if m.getUserDefaultFunc != nil {
		return m.getUserDefaultFunc(ctx, userID)
	}
	return nil, nil
}

func TestProviderService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create system provider", func(t *testing.T) {
		repo := &mockProviderRepository{
			createFunc: func(ctx context.Context, provider *model.Provider) error {
				provider.ID = uuid.New().String()
				return nil
			},
		}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:        "Claude Code",
			Type:        model.ProviderTypeClaudeCode,
			Scope:       model.ProviderScopeSystem,
			APIEndpoint: "https://api.anthropic.com",
		}

		provider, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.Name != "Claude Code" {
			t.Errorf("Expected name 'Claude Code', got '%s'", provider.Name)
		}
		if provider.Type != model.ProviderTypeClaudeCode {
			t.Errorf("Expected type '%s', got '%s'", model.ProviderTypeClaudeCode, provider.Type)
		}
		if provider.Status != model.ProviderStatusActive {
			t.Errorf("Expected status '%s', got '%s'", model.ProviderStatusActive, provider.Status)
		}
	})

	t.Run("successful create tenant provider", func(t *testing.T) {
		tenantID := uuid.New().String()
		repo := &mockProviderRepository{
			createFunc: func(ctx context.Context, provider *model.Provider) error {
				provider.ID = uuid.New().String()
				return nil
			},
		}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:        "Tenant Provider",
			Type:        model.ProviderTypeOpenAICompat,
			Scope:       model.ProviderScopeTenant,
			TenantID:    &tenantID,
			APIEndpoint: "https://api.openai.com",
		}

		provider, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if *provider.TenantID != tenantID {
			t.Errorf("Expected tenant_id '%s', got '%s'", tenantID, *provider.TenantID)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:  "",
			Type:  model.ProviderTypeClaudeCode,
			Scope: model.ProviderScopeSystem,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid scope", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:  "Test Provider",
			Type:  model.ProviderTypeClaudeCode,
			Scope: model.ProviderScope("invalid"),
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid scope, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("tenant scope without tenant_id", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:  "Test Provider",
			Type:  model.ProviderTypeClaudeCode,
			Scope: model.ProviderScopeTenant,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for tenant scope without tenant_id, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("user scope without user_id", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:  "Test Provider",
			Type:  model.ProviderTypeClaudeCode,
			Scope: model.ProviderScopeUser,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for user scope without user_id, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockProviderRepository{
			createFunc: func(ctx context.Context, provider *model.Provider) error {
				return errors.NewInternalError("database error")
			},
		}
		service := NewProviderService(repo)

		req := &CreateProviderRequest{
			Name:  "Test Provider",
			Type:  model.ProviderTypeClaudeCode,
			Scope: model.ProviderScopeSystem,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestProviderService_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				if id == existingID {
					return &model.Provider{
						ID:     existingID.String(),
						Name:   "Test Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeSystem,
						Status: model.ProviderStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		provider, err := service.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.Name != "Test Provider" {
			t.Errorf("Expected name 'Test Provider', got '%s'", provider.Name)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		_, err := service.GetByID(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		_, err := service.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		repo := &mockProviderRepository{
			listFunc: func(ctx context.Context, filter repository.ProviderFilter) ([]*model.Provider, int64, error) {
				return []*model.Provider{
					{ID: id1.String(), Name: "Provider 1", Type: model.ProviderTypeClaudeCode, Scope: model.ProviderScopeSystem, Status: model.ProviderStatusActive},
					{ID: id2.String(), Name: "Provider 2", Type: model.ProviderTypeOpenAICompat, Scope: model.ProviderScopeSystem, Status: model.ProviderStatusActive},
				}, 2, nil
			},
		}
		service := NewProviderService(repo)

		filter := &ProviderFilter{Page: 1, PageSize: 10}
		providers, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		repo := &mockProviderRepository{
			listFunc: func(ctx context.Context, filter repository.ProviderFilter) ([]*model.Provider, int64, error) {
				return []*model.Provider{}, 0, nil
			},
		}
		service := NewProviderService(repo)

		filter := &ProviderFilter{Page: 1, PageSize: 10}
		providers, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 0 {
			t.Errorf("Expected 0 providers, got %d", len(providers))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})
}

func TestProviderService_ResolveProvider(t *testing.T) {
	ctx := context.Background()
	specifiedID := uuid.New()
	userDefaultID := uuid.New()
	tenantDefaultID := uuid.New()
	systemDefaultID := uuid.New()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	t.Run("use specified provider when provided", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				if id == specifiedID {
					return &model.Provider{
						ID:     specifiedID.String(),
						Name:   "Specified Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeSystem,
						Status: model.ProviderStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		provider, err := service.ResolveProvider(ctx, specifiedID.String(), tenantID, userID)
		if err != nil {
			t.Errorf("ResolveProvider returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.ID != specifiedID.String() {
			t.Errorf("Expected specified provider, got %s", provider.ID)
		}
	})

	t.Run("fall back to user default when specified not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
			getUserDefaultFunc: func(ctx context.Context, uid string) (*model.Provider, error) {
				if uid == userID {
					return &model.Provider{
						ID:     userDefaultID.String(),
						Name:   "User Default Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeUser,
						Status: model.ProviderStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("user default not found")
			},
		}
		service := NewProviderService(repo)

		provider, err := service.ResolveProvider(ctx, "nonexistent", tenantID, userID)
		if err != nil {
			t.Errorf("ResolveProvider returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.ID != userDefaultID.String() {
			t.Errorf("Expected user default provider, got %s", provider.ID)
		}
	})

	t.Run("fall back to tenant default when user default not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
			getUserDefaultFunc: func(ctx context.Context, uid string) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("user default not found")
			},
			getDefaultProviderFunc: func(ctx context.Context, scope model.ProviderScope, tid, uid *string) (*model.Provider, error) {
				if scope == model.ProviderScopeTenant && tid != nil && *tid == tenantID {
					return &model.Provider{
						ID:     tenantDefaultID.String(),
						Name:   "Tenant Default Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeTenant,
						Status: model.ProviderStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		provider, err := service.ResolveProvider(ctx, "", tenantID, userID)
		if err != nil {
			t.Errorf("ResolveProvider returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.ID != tenantDefaultID.String() {
			t.Errorf("Expected tenant default provider, got %s", provider.ID)
		}
	})

	t.Run("fall back to system default when tenant default not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
			getUserDefaultFunc: func(ctx context.Context, uid string) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("user default not found")
			},
			getDefaultProviderFunc: func(ctx context.Context, scope model.ProviderScope, tid, uid *string) (*model.Provider, error) {
				if scope == model.ProviderScopeSystem {
					return &model.Provider{
						ID:     systemDefaultID.String(),
						Name:   "System Default Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeSystem,
						Status: model.ProviderStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		provider, err := service.ResolveProvider(ctx, "", tenantID, userID)
		if err != nil {
			t.Errorf("ResolveProvider returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		if provider.ID != systemDefaultID.String() {
			t.Errorf("Expected system default provider, got %s", provider.ID)
		}
	})

	t.Run("return error when no provider found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
			getUserDefaultFunc: func(ctx context.Context, uid string) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("user default not found")
			},
			getDefaultProviderFunc: func(ctx context.Context, scope model.ProviderScope, tid, uid *string) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		_, err := service.ResolveProvider(ctx, "", tenantID, userID)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})

	t.Run("skip inactive specified provider", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				if id == specifiedID {
					return &model.Provider{
						ID:     specifiedID.String(),
						Name:   "Inactive Provider",
						Type:   model.ProviderTypeClaudeCode,
						Scope:  model.ProviderScopeSystem,
						Status: model.ProviderStatusInactive,
					}, nil
				}
				return nil, errors.NewNotFoundError("provider not found")
			},
			getUserDefaultFunc: func(ctx context.Context, uid string) (*model.Provider, error) {
				return &model.Provider{
					ID:     userDefaultID.String(),
					Name:   "User Default Provider",
					Type:   model.ProviderTypeClaudeCode,
					Scope:  model.ProviderScopeUser,
					Status: model.ProviderStatusActive,
				}, nil
			},
		}
		service := NewProviderService(repo)

		provider, err := service.ResolveProvider(ctx, specifiedID.String(), tenantID, userID)
		if err != nil {
			t.Errorf("ResolveProvider returned error: %v", err)
		}
		if provider == nil {
			t.Fatal("Expected provider, got nil")
		}
		// Should fall back to user default since specified is inactive
		if provider.ID != userDefaultID.String() {
			t.Errorf("Expected user default provider (since specified is inactive), got %s", provider.ID)
		}
	})
}

func TestProviderService_SetDefaultProvider(t *testing.T) {
	ctx := context.Background()
	providerID := uuid.New().String()
	userID := uuid.New().String()

	t.Run("successful set default", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return &model.Provider{
					ID:     providerID,
					Name:   "Test Provider",
					Type:   model.ProviderTypeClaudeCode,
					Scope:  model.ProviderScopeUser,
					Status: model.ProviderStatusActive,
				}, nil
			},
			setUserDefaultFunc: func(ctx context.Context, uid, pid string) error {
				return nil
			},
		}
		service := NewProviderService(repo)

		err := service.SetDefaultProvider(ctx, userID, providerID)
		if err != nil {
			t.Errorf("SetDefaultProvider returned error: %v", err)
		}
	})

	t.Run("invalid provider id format", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		err := service.SetDefaultProvider(ctx, userID, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		err := service.SetDefaultProvider(ctx, userID, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderService_TestConnection(t *testing.T) {
	ctx := context.Background()
	providerID := uuid.New()

	t.Run("no api endpoint configured", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return &model.Provider{
					ID:          providerID.String(),
					Name:        "Test Provider",
					Type:        model.ProviderTypeClaudeCode,
					Scope:       model.ProviderScopeSystem,
					Status:      model.ProviderStatusActive,
					APIEndpoint: "",
				}, nil
			},
		}
		service := NewProviderService(repo)

		result, err := service.TestConnection(ctx, providerID.String())
		if err != nil {
			t.Errorf("TestConnection returned error: %v", err)
		}
		if result.Success {
			t.Error("Expected failure for no API endpoint")
		}
		if result.Message != "API endpoint not configured" {
			t.Errorf("Expected 'API endpoint not configured', got '%s'", result.Message)
		}
	})

	t.Run("invalid provider id format", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		_, err := service.TestConnection(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return nil, errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		_, err := service.TestConnection(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderService_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update with name", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return &model.Provider{
					ID:     existingID.String(),
					Name:   "Old Name",
					Type:   model.ProviderTypeClaudeCode,
					Scope:  model.ProviderScopeSystem,
					Status: model.ProviderStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, provider *model.Provider) error {
				return nil
			},
		}
		service := NewProviderService(repo)

		newName := "New Name"
		req := &UpdateProviderRequest{
			Name: &newName,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		req := &UpdateProviderRequest{}
		err := service.Update(ctx, "invalid-uuid", req)
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		repo := &mockProviderRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
				return &model.Provider{
					ID:     existingID.String(),
					Name:   "Test Provider",
					Scope:  model.ProviderScopeSystem,
					Status: model.ProviderStatusActive,
				}, nil
			},
		}
		service := NewProviderService(repo)

		status := model.ProviderStatus("invalid-status")
		req := &UpdateProviderRequest{
			Status: &status,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for invalid status, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})
}

func TestProviderService_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		repo := &mockProviderRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		service := NewProviderService(repo)

		err := service.Delete(ctx, existingID.String())
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockProviderRepository{}
		service := NewProviderService(repo)

		err := service.Delete(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockProviderRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return errors.NewNotFoundError("provider not found")
			},
		}
		service := NewProviderService(repo)

		err := service.Delete(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderService_GetAvailableProviders(t *testing.T) {
	ctx := context.Background()
	systemID := uuid.New()
	tenantID := uuid.New().String()
	userID := uuid.New().String()

	t.Run("get available providers for user", func(t *testing.T) {
		repo := &mockProviderRepository{
			listFunc: func(ctx context.Context, filter repository.ProviderFilter) ([]*model.Provider, int64, error) {
				// Return providers based on filter
				if filter.Scope == string(model.ProviderScopeSystem) {
					return []*model.Provider{
						{ID: systemID.String(), Name: "System Provider", Scope: model.ProviderScopeSystem, Status: model.ProviderStatusActive},
					}, 1, nil
				}
				return []*model.Provider{}, 0, nil
			},
		}
		service := NewProviderService(repo)

		providers, err := service.GetAvailableProviders(ctx, tenantID, userID)
		if err != nil {
			t.Errorf("GetAvailableProviders returned error: %v", err)
		}
		if len(providers) == 0 {
			t.Error("Expected at least one provider")
		}
	})
}

func TestProviderService_Interface(t *testing.T) {
	// Verify that providerService implements ProviderService interface
	var _ ProviderService = (*providerService)(nil)
}
