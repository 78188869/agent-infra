// Package service provides business logic implementations for the application.
package service

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// mockCapabilityRepository implements repository.CapabilityRepository for testing
type mockCapabilityRepository struct {
	createFunc  func(ctx context.Context, capability *model.Capability) error
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*model.Capability, error)
	listFunc    func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error)
	updateFunc  func(ctx context.Context, capability *model.Capability) error
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockCapabilityRepository) Create(ctx context.Context, capability *model.Capability) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, capability)
	}
	return nil
}

func (m *mockCapabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockCapabilityRepository) List(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockCapabilityRepository) Update(ctx context.Context, capability *model.Capability) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, capability)
	}
	return nil
}

func (m *mockCapabilityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestCapabilityService_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("successful create tool capability", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:             model.CapabilityTypeTool,
			Name:             "Test Tool",
			Description:      "A test tool",
			Version:          "1.0.0",
			PermissionLevel:  model.PermissionLevelPublic,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability == nil {
			t.Fatal("Expected capability, got nil")
		}
		if capability.Name != "Test Tool" {
			t.Errorf("Expected name 'Test Tool', got '%s'", capability.Name)
		}
		if capability.Type != model.CapabilityTypeTool {
			t.Errorf("Expected type 'tool', got '%s'", capability.Type)
		}
		if capability.Status != model.CapabilityStatusActive {
			t.Errorf("Expected status 'active', got '%s'", capability.Status)
		}
		if capability.PermissionLevel != model.PermissionLevelPublic {
			t.Errorf("Expected permission level 'public', got '%s'", capability.PermissionLevel)
		}
	})

	t.Run("successful create skill capability", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeSkill,
			Name:            "Test Skill",
			Description:     "A test skill",
			PermissionLevel: model.PermissionLevelRestricted,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.Type != model.CapabilityTypeSkill {
			t.Errorf("Expected type 'skill', got '%s'", capability.Type)
		}
		if capability.PermissionLevel != model.PermissionLevelRestricted {
			t.Errorf("Expected permission level 'restricted', got '%s'", capability.PermissionLevel)
		}
	})

	t.Run("successful create agent_runtime capability", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeAgentRuntime,
			Name:            "Test Runtime",
			Description:     "A test runtime",
			PermissionLevel: model.PermissionLevelAdminOnly,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.Type != model.CapabilityTypeAgentRuntime {
			t.Errorf("Expected type 'agent_runtime', got '%s'", capability.Type)
		}
		if capability.PermissionLevel != model.PermissionLevelAdminOnly {
			t.Errorf("Expected permission level 'admin_only', got '%s'", capability.PermissionLevel)
		}
	})

	t.Run("successful create with tenant_id", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Tenant Tool",
			TenantID:        tenantID.String(),
			PermissionLevel: model.PermissionLevelPublic,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.TenantID == nil {
			t.Error("Expected TenantID to be set, got nil")
		}
		if *capability.TenantID != tenantID.String() {
			t.Errorf("Expected TenantID '%s', got '%s'", tenantID.String(), *capability.TenantID)
		}
	})

	t.Run("successful create global capability (no tenant)", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Global Tool",
			PermissionLevel: model.PermissionLevelPublic,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.TenantID != nil {
			t.Error("Expected nil TenantID for global capability, got value")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "",
			PermissionLevel: model.PermissionLevelPublic,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityType("invalid_type"),
			Name:            "Test Tool",
			PermissionLevel: model.PermissionLevelPublic,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid type, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid permission level", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Test Tool",
			PermissionLevel: model.PermissionLevel("invalid_permission"),
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid permission level, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid tenant_id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Test Tool",
			TenantID:        "invalid-uuid",
			PermissionLevel: model.PermissionLevelPublic,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid tenant_id, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				return errors.NewInternalError("database error")
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Test Tool",
			PermissionLevel: model.PermissionLevelPublic,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("default version is 1.0.0", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			createFunc: func(ctx context.Context, capability *model.Capability) error {
				capability.ID = uuid.New().String()
				return nil
			},
		}
		service := NewCapabilityService(repo)

		req := &CreateCapabilityRequest{
			Type:            model.CapabilityTypeTool,
			Name:            "Test Tool",
			PermissionLevel: model.PermissionLevelPublic,
		}

		capability, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.Version != "1.0.0" {
			t.Errorf("Expected default version '1.0.0', got '%s'", capability.Version)
		}
	})
}

func TestCapabilityService_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		tenantIDStr := tenantID.String()
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				if id == existingID {
					return &model.Capability{
						ID:             existingID.String(),
						Type:           model.CapabilityTypeTool,
						Name:           "Test Tool",
						Description:    "A test tool",
						TenantID:       &tenantIDStr,
						PermissionLevel: model.PermissionLevelPublic,
						Status:         model.CapabilityStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		capability, err := service.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if capability == nil {
			t.Fatal("Expected capability, got nil")
		}
		if capability.Name != "Test Tool" {
			t.Errorf("Expected name 'Test Tool', got '%s'", capability.Name)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		_, err := service.GetByID(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		_, err := service.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityService_List(t *testing.T) {
	ctx := context.Background()
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	t.Run("successful list", func(t *testing.T) {
		tenantIDStr := tenantID.String()
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				return []*model.Capability{
					{ID: id1.String(), Type: model.CapabilityTypeTool, Name: "Tool 1", TenantID: &tenantIDStr},
					{ID: id2.String(), Type: model.CapabilityTypeSkill, Name: "Skill 1", TenantID: &tenantIDStr},
				}, 2, nil
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10}
		capabilities, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(capabilities) != 2 {
			t.Errorf("Expected 2 capabilities, got %d", len(capabilities))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("list with tenant_id filter", func(t *testing.T) {
		tenantIDStr := tenantID.String()
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				if filter.TenantID != tenantIDStr {
					t.Errorf("Expected tenant_id '%s', got '%s'", tenantIDStr, filter.TenantID)
				}
				return []*model.Capability{}, 0, nil
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10, TenantID: tenantIDStr}
		_, _, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
	})

	t.Run("list with type filter", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				if filter.Type != string(model.CapabilityTypeTool) {
					t.Errorf("Expected type 'tool', got '%s'", filter.Type)
				}
				return []*model.Capability{}, 0, nil
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10, Type: string(model.CapabilityTypeTool)}
		_, _, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				if filter.Status != string(model.CapabilityStatusActive) {
					t.Errorf("Expected status 'active', got '%s'", filter.Status)
				}
				return []*model.Capability{}, 0, nil
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10, Status: string(model.CapabilityStatusActive)}
		_, _, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				return []*model.Capability{}, 0, nil
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10}
		capabilities, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(capabilities) != 0 {
			t.Errorf("Expected 0 capabilities, got %d", len(capabilities))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			listFunc: func(ctx context.Context, filter repository.CapabilityFilter) ([]*model.Capability, int64, error) {
				return nil, 0, errors.NewInternalError("database error")
			},
		}
		service := NewCapabilityService(repo)

		filter := &CapabilityFilter{Page: 1, PageSize: 10}
		_, _, err := service.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestCapabilityService_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()

	t.Run("successful update with name", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Old Name",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newName := "New Name"
		req := &UpdateCapabilityRequest{
			Name: &newName,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with description", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					Description:    "Old description",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newDesc := "New description"
		req := &UpdateCapabilityRequest{
			Description: &newDesc,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with version", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					Version:        "1.0.0",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newVersion := "2.0.0"
		req := &UpdateCapabilityRequest{
			Version: &newVersion,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with permission level", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newPerm := model.PermissionLevelAdminOnly
		req := &UpdateCapabilityRequest{
			PermissionLevel: &newPerm,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with config", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newConfig := datatypes.JSON([]byte(`{"key": "value"}`))
		req := &UpdateCapabilityRequest{
			Config: &newConfig,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with schema", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		newSchema := datatypes.JSON([]byte(`{"type": "object"}`))
		req := &UpdateCapabilityRequest{
			Schema: &newSchema,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		req := &UpdateCapabilityRequest{}
		err := service.Update(ctx, "invalid-uuid", req)
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
		}
		service := NewCapabilityService(repo)

		emptyName := ""
		req := &UpdateCapabilityRequest{
			Name: &emptyName,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid permission level", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
		}
		service := NewCapabilityService(repo)

		invalidPerm := model.PermissionLevel("invalid_permission")
		req := &UpdateCapabilityRequest{
			PermissionLevel: &invalidPerm,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for invalid permission level, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		req := &UpdateCapabilityRequest{}
		err := service.Update(ctx, uuid.New().String(), req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityService_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()

	t.Run("successful delete", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		err := service.Delete(ctx, existingID.String())
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		err := service.Delete(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		err := service.Delete(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityService_Activate(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()

	t.Run("successful activate", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusInactive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		err := service.Activate(ctx, existingID.String())
		if err != nil {
			t.Errorf("Activate returned error: %v", err)
		}
	})

	t.Run("activate already active capability", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		err := service.Activate(ctx, existingID.String())
		if err != nil {
			t.Errorf("Activate returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		err := service.Activate(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		err := service.Activate(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityService_Deactivate(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()
	tenantIDStr := tenantID.String()

	t.Run("successful deactivate", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		err := service.Deactivate(ctx, existingID.String())
		if err != nil {
			t.Errorf("Deactivate returned error: %v", err)
		}
	})

	t.Run("deactivate already inactive capability", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return &model.Capability{
					ID:             existingID.String(),
					Type:           model.CapabilityTypeTool,
					Name:           "Test Tool",
					TenantID:       &tenantIDStr,
					PermissionLevel: model.PermissionLevelPublic,
					Status:         model.CapabilityStatusInactive,
				}, nil
			},
			updateFunc: func(ctx context.Context, capability *model.Capability) error {
				return nil
			},
		}
		service := NewCapabilityService(repo)

		err := service.Deactivate(ctx, existingID.String())
		if err != nil {
			t.Errorf("Deactivate returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockCapabilityRepository{}
		service := NewCapabilityService(repo)

		err := service.Deactivate(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockCapabilityRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
				return nil, errors.NewNotFoundError("capability not found")
			},
		}
		service := NewCapabilityService(repo)

		err := service.Deactivate(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityService_Interface(t *testing.T) {
	// Verify that capabilityService implements CapabilityService interface
	var _ CapabilityService = (*capabilityService)(nil)
}
