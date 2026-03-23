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

// mockTemplateRepository implements repository.TemplateRepository for testing
type mockTemplateRepository struct {
	createFunc  func(ctx context.Context, template *model.Template) error
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*model.Template, error)
	listFunc    func(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error)
	updateFunc  func(ctx context.Context, template *model.Template) error
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockTemplateRepository) Create(ctx context.Context, template *model.Template) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, template)
	}
	return nil
}

func (m *mockTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTemplateRepository) List(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTemplateRepository) Update(ctx context.Context, template *model.Template) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, template)
	}
	return nil
}

func (m *mockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestTemplateService_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	t.Run("successful create", func(t *testing.T) {
		repo := &mockTemplateRepository{
			createFunc: func(ctx context.Context, template *model.Template) error {
				template.ID = uuid.New()
				return nil
			},
		}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  tenantID.String(),
			Name:      "Test Template",
			Version:   "1.0.0",
			Spec:      "name: test\nversion: '1.0'",
			SceneType: model.TemplateSceneTypeCoding,
		}

		template, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if template == nil {
			t.Fatal("Expected template, got nil")
		}
		if template.Name != "Test Template" {
			t.Errorf("Expected name 'Test Template', got '%s'", template.Name)
		}
		if template.SceneType != model.TemplateSceneTypeCoding {
			t.Errorf("Expected scene_type '%s', got '%s'", model.TemplateSceneTypeCoding, template.SceneType)
		}
		if template.Status != model.TemplateStatusDraft {
			t.Errorf("Expected status '%s', got '%s'", model.TemplateStatusDraft, template.Status)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  tenantID.String(),
			Name:      "",
			SceneType: model.TemplateSceneTypeCoding,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("empty tenant_id", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  "",
			Name:      "Test Template",
			SceneType: model.TemplateSceneTypeCoding,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for empty tenant_id, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid tenant_id format", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  "invalid-uuid",
			Name:      "Test Template",
			SceneType: model.TemplateSceneTypeCoding,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid tenant_id, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid scene_type", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  tenantID.String(),
			Name:      "Test Template",
			SceneType: "invalid-type",
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid scene_type, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid yaml spec", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  tenantID.String(),
			Name:      "Test Template",
			Spec:      "invalid: [yaml: content",
			SceneType: model.TemplateSceneTypeCoding,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for invalid yaml spec, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTemplateRepository{
			createFunc: func(ctx context.Context, template *model.Template) error {
				return errors.NewInternalError("database error")
			},
		}
		service := NewTemplateService(repo)

		req := &CreateTemplateRequest{
			TenantID:  tenantID.String(),
			Name:      "Test Template",
			SceneType: model.TemplateSceneTypeCoding,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTemplateService_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				if id == existingID {
					return &model.Template{
						BaseModel: model.BaseModel{ID: existingID},
						TenantID:  tenantID.String(),
						Name:      "Test Template",
						Status:    model.TemplateStatusDraft,
					}, nil
				}
				return nil, errors.NewNotFoundError("template not found")
			},
		}
		service := NewTemplateService(repo)

		template, err := service.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if template == nil {
			t.Fatal("Expected template, got nil")
		}
		if template.Name != "Test Template" {
			t.Errorf("Expected name 'Test Template', got '%s'", template.Name)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		_, err := service.GetByID(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return nil, errors.NewNotFoundError("template not found")
			},
		}
		service := NewTemplateService(repo)

		_, err := service.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateService_List(t *testing.T) {
	ctx := context.Background()
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	t.Run("successful list", func(t *testing.T) {
		repo := &mockTemplateRepository{
			listFunc: func(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error) {
				return []*model.Template{
					{BaseModel: model.BaseModel{ID: id1}, TenantID: tenantID.String(), Name: "Template 1"},
					{BaseModel: model.BaseModel{ID: id2}, TenantID: tenantID.String(), Name: "Template 2"},
				}, 2, nil
			},
		}
		service := NewTemplateService(repo)

		filter := &TemplateFilter{Page: 1, PageSize: 10}
		templates, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(templates) != 2 {
			t.Errorf("Expected 2 templates, got %d", len(templates))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("list with tenant_id filter", func(t *testing.T) {
		repo := &mockTemplateRepository{
			listFunc: func(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error) {
				if filter.TenantID != tenantID.String() {
					t.Errorf("Expected tenant_id '%s', got '%s'", tenantID.String(), filter.TenantID)
				}
				return []*model.Template{}, 0, nil
			},
		}
		service := NewTemplateService(repo)

		filter := &TemplateFilter{Page: 1, PageSize: 10, TenantID: tenantID.String()}
		_, _, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		repo := &mockTemplateRepository{
			listFunc: func(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error) {
				return []*model.Template{}, 0, nil
			},
		}
		service := NewTemplateService(repo)

		filter := &TemplateFilter{Page: 1, PageSize: 10}
		templates, total, err := service.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(templates) != 0 {
			t.Errorf("Expected 0 templates, got %d", len(templates))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTemplateRepository{
			listFunc: func(ctx context.Context, filter repository.TemplateFilter) ([]*model.Template, int64, error) {
				return nil, 0, errors.NewInternalError("database error")
			},
		}
		service := NewTemplateService(repo)

		filter := &TemplateFilter{Page: 1, PageSize: 10}
		_, _, err := service.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTemplateService_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	t.Run("successful update with name", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Old Name",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
			updateFunc: func(ctx context.Context, template *model.Template) error {
				return nil
			},
		}
		service := NewTemplateService(repo)

		newName := "New Name"
		req := &UpdateTemplateRequest{
			Name: &newName,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with scene_type", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					SceneType: model.TemplateSceneTypeCoding,
					Status:    model.TemplateStatusDraft,
				}, nil
			},
			updateFunc: func(ctx context.Context, template *model.Template) error {
				return nil
			},
		}
		service := NewTemplateService(repo)

		sceneType := model.TemplateSceneTypeOps
		req := &UpdateTemplateRequest{
			SceneType: &sceneType,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with status", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
			updateFunc: func(ctx context.Context, template *model.Template) error {
				return nil
			},
		}
		service := NewTemplateService(repo)

		status := model.TemplateStatusPublished
		req := &UpdateTemplateRequest{
			Status: &status,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		req := &UpdateTemplateRequest{}
		err := service.Update(ctx, "invalid-uuid", req)
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid scene_type", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
		}
		service := NewTemplateService(repo)

		sceneType := "invalid-type"
		req := &UpdateTemplateRequest{
			SceneType: &sceneType,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for invalid scene_type, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
		}
		service := NewTemplateService(repo)

		status := "invalid-status"
		req := &UpdateTemplateRequest{
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

	t.Run("invalid yaml spec", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
		}
		service := NewTemplateService(repo)

		spec := "invalid: [yaml: content"
		req := &UpdateTemplateRequest{
			Spec: &spec,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for invalid yaml spec, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return nil, errors.NewNotFoundError("template not found")
			},
		}
		service := NewTemplateService(repo)

		req := &UpdateTemplateRequest{}
		err := service.Update(ctx, uuid.New().String(), req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateService_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	t.Run("successful delete draft template", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDraft,
				}, nil
			},
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		service := NewTemplateService(repo)

		err := service.Delete(ctx, existingID.String())
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}
	})

	t.Run("cannot delete published template", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusPublished,
				}, nil
			},
		}
		service := NewTemplateService(repo)

		err := service.Delete(ctx, existingID.String())
		if err == nil {
			t.Error("Expected error for deleting published template, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("cannot delete deprecated template", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return &model.Template{
					BaseModel: model.BaseModel{ID: existingID},
					TenantID:  tenantID.String(),
					Name:      "Test Template",
					Status:    model.TemplateStatusDeprecated,
				}, nil
			},
		}
		service := NewTemplateService(repo)

		err := service.Delete(ctx, existingID.String())
		if err == nil {
			t.Error("Expected error for deleting deprecated template, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTemplateRepository{}
		service := NewTemplateService(repo)

		err := service.Delete(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTemplateRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Template, error) {
				return nil, errors.NewNotFoundError("template not found")
			},
		}
		service := NewTemplateService(repo)

		err := service.Delete(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateService_Interface(t *testing.T) {
	// Verify that templateService implements TemplateService interface
	var _ TemplateService = (*templateService)(nil)
}
