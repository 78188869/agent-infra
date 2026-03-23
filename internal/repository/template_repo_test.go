package repository

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockTemplateDB simulates database for testing purposes
type mockTemplateDB struct {
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error
	templates  map[uuid.UUID]*model.Template
	totalCount int64
}

// mockTemplateRepository is a mock implementation for testing
type mockTemplateRepository struct {
	db *mockTemplateDB
}

func newMockTemplateRepository(db *mockTemplateDB) *mockTemplateRepository {
	return &mockTemplateRepository{db: db}
}

func (m *mockTemplateRepository) Create(ctx context.Context, template *model.Template) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.templates == nil {
		m.db.templates = make(map[uuid.UUID]*model.Template)
	}
	template.ID = uuid.New()
	m.db.templates[template.ID] = template
	return nil
}

func (m *mockTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.templates == nil {
		return nil, errors.NewNotFoundError("template not found")
	}
	template, ok := m.db.templates[id]
	if !ok {
		return nil, errors.NewNotFoundError("template not found")
	}
	return template, nil
}

func (m *mockTemplateRepository) List(ctx context.Context, filter TemplateFilter) ([]*model.Template, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.templates == nil {
		return []*model.Template{}, 0, nil
	}

	var result []*model.Template
	for _, t := range m.db.templates {
		// Apply filters
		if filter.TenantID != "" && t.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.SceneType != "" && t.SceneType != filter.SceneType {
			continue
		}
		result = append(result, t)
	}
	return result, m.db.totalCount, nil
}

func (m *mockTemplateRepository) Update(ctx context.Context, template *model.Template) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.templates == nil {
		return errors.NewNotFoundError("template not found")
	}
	if _, ok := m.db.templates[template.ID]; !ok {
		return errors.NewNotFoundError("template not found")
	}
	m.db.templates[template.ID] = template
	return nil
}

func (m *mockTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.db.deleteErr != nil {
		return m.db.deleteErr
	}
	if m.db.templates == nil {
		return errors.NewNotFoundError("template not found")
	}
	if _, ok := m.db.templates[id]; !ok {
		return errors.NewNotFoundError("template not found")
	}
	delete(m.db.templates, id)
	return nil
}

// Tests

func TestTemplateRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockTemplateDB{}
		repo := newMockTemplateRepository(mockDB)

		template := &model.Template{
			Name:      "Test Template",
			TenantID:  uuid.New().String(),
			SceneType: model.TemplateSceneTypeCoding,
			Status:    model.TemplateStatusDraft,
		}

		err := repo.Create(ctx, template)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if template.ID == uuid.Nil {
			t.Error("Template ID should be set after creation")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockTemplateDB{createErr: expectedErr}
		repo := newMockTemplateRepository(mockDB)

		template := &model.Template{Name: "Test Template"}
		err := repo.Create(ctx, template)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTemplateRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockTemplateDB{
			templates: map[uuid.UUID]*model.Template{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Existing Template"},
			},
		}
		repo := newMockTemplateRepository(mockDB)

		template, err := repo.GetByID(ctx, existingID)
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if template == nil {
			t.Error("Expected template, got nil")
		}
		if template.Name != "Existing Template" {
			t.Errorf("Expected name 'Existing Template', got '%s'", template.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockTemplateDB{templates: make(map[uuid.UUID]*model.Template)}
		repo := newMockTemplateRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateRepository_List(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New().String()

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockTemplateDB{
			templates: map[uuid.UUID]*model.Template{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Template 1", TenantID: tenantID},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Template 2", TenantID: tenantID},
			},
			totalCount: 2,
		}
		repo := newMockTemplateRepository(mockDB)

		filter := TemplateFilter{Page: 1, PageSize: 10, TenantID: tenantID}
		templates, total, err := repo.List(ctx, filter)
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

	t.Run("list with scene type filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockTemplateDB{
			templates: map[uuid.UUID]*model.Template{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Coding Template", SceneType: model.TemplateSceneTypeCoding},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Ops Template", SceneType: model.TemplateSceneTypeOps},
			},
			totalCount: 1,
		}
		repo := newMockTemplateRepository(mockDB)

		filter := TemplateFilter{Page: 1, PageSize: 10, SceneType: model.TemplateSceneTypeCoding}
		templates, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(templates) != 1 {
			t.Errorf("Expected 1 template, got %d", len(templates))
		}
		if templates[0].SceneType != model.TemplateSceneTypeCoding {
			t.Errorf("Expected scene type 'coding', got '%s'", templates[0].SceneType)
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockTemplateDB{templates: nil}
		repo := newMockTemplateRepository(mockDB)

		filter := TemplateFilter{Page: 1, PageSize: 10}
		templates, total, err := repo.List(ctx, filter)
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

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockTemplateDB{listErr: expectedErr}
		repo := newMockTemplateRepository(mockDB)

		filter := TemplateFilter{Page: 1, PageSize: 10}
		_, _, err := repo.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTemplateRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockTemplateDB{
			templates: map[uuid.UUID]*model.Template{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Old Name"},
			},
		}
		repo := newMockTemplateRepository(mockDB)

		template := &model.Template{BaseModel: model.BaseModel{ID: existingID}, Name: "New Name"}
		err := repo.Update(ctx, template)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID)
		if updated.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockTemplateDB{templates: make(map[uuid.UUID]*model.Template)}
		repo := newMockTemplateRepository(mockDB)

		template := &model.Template{BaseModel: model.BaseModel{ID: uuid.New()}, Name: "New Name"}
		err := repo.Update(ctx, template)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateRepository_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockTemplateDB{
			templates: map[uuid.UUID]*model.Template{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Template to Delete"},
			},
		}
		repo := newMockTemplateRepository(mockDB)

		err := repo.Delete(ctx, existingID)
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}

		_, err = repo.GetByID(ctx, existingID)
		if err == nil {
			t.Error("Expected error after delete, got nil")
		}
	})

	t.Run("delete non-existent", func(t *testing.T) {
		mockDB := &mockTemplateDB{templates: make(map[uuid.UUID]*model.Template)}
		repo := newMockTemplateRepository(mockDB)

		err := repo.Delete(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTemplateFilter_Fields(t *testing.T) {
	// Verify filter fields exist
	filter := TemplateFilter{}

	_ = filter.Page
	_ = filter.PageSize
	_ = filter.TenantID
	_ = filter.Status
	_ = filter.SceneType
	_ = filter.Search
}

func TestTemplateFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := TemplateFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := TemplateFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := TemplateFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})
}

func TestTemplateFilter_Offset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		expected int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 10, 20},
		{1, 20, 0},
		{5, 10, 40},
	}

	for _, tt := range tests {
		filter := TemplateFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestTemplateRepository_Interface(t *testing.T) {
	// Verify that mockTemplateRepository implements TemplateRepository interface
	var _ TemplateRepository = (*mockTemplateRepository)(nil)
}

func TestNewTemplateRepository(t *testing.T) {
	// Test that NewTemplateRepository returns a non-nil implementation
	repo := NewTemplateRepository(nil)
	if repo == nil {
		t.Error("NewTemplateRepository should return non-nil interface value")
	}
}

// Verify error type checking works
func TestTemplateRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("template not found")
	if !stderrors.Is(err, errors.ErrNotFound) {
		t.Error("Error should match ErrNotFound")
	}

	internalErr := errors.NewInternalError("something went wrong")
	if !stderrors.Is(internalErr, errors.ErrInternal) {
		t.Error("Error should match ErrInternal")
	}
}
