package repository

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockCapabilityDB simulates database for testing purposes
type mockCapabilityDB struct {
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error
	capabilities map[string]*model.Capability
	totalCount  int64
}

// mockCapabilityRepository is a mock implementation for testing
type mockCapabilityRepository struct {
	db *mockCapabilityDB
}

func newMockCapabilityRepository(db *mockCapabilityDB) *mockCapabilityRepository {
	return &mockCapabilityRepository{db: db}
}

func (m *mockCapabilityRepository) Create(ctx context.Context, capability *model.Capability) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.capabilities == nil {
		m.db.capabilities = make(map[string]*model.Capability)
	}
	capability.ID = uuid.New().String()
	m.db.capabilities[capability.ID] = capability
	return nil
}

func (m *mockCapabilityRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Capability, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.capabilities == nil {
		return nil, errors.NewNotFoundError("capability not found")
	}
	capability, ok := m.db.capabilities[id.String()]
	if !ok {
		return nil, errors.NewNotFoundError("capability not found")
	}
	return capability, nil
}

func (m *mockCapabilityRepository) List(ctx context.Context, filter CapabilityFilter) ([]*model.Capability, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.capabilities == nil {
		return []*model.Capability{}, 0, nil
	}

	var result []*model.Capability
	for _, c := range m.db.capabilities {
		// Apply filters
		if filter.TenantID != "" {
			if filter.TenantID == "global" {
				if c.TenantID != nil {
					continue
				}
			} else {
				if c.TenantID == nil || *c.TenantID != filter.TenantID {
					continue
				}
			}
		}
		if filter.Type != "" && string(c.Type) != filter.Type {
			continue
		}
		if filter.Status != "" && string(c.Status) != filter.Status {
			continue
		}
		if filter.Search != "" {
			searchPattern := filter.Search
			found := false
			if len(c.Name) >= len(searchPattern) && containsSubstring(c.Name, searchPattern) {
				found = true
			}
			if !found && len(c.Description) >= len(searchPattern) && containsSubstring(c.Description, searchPattern) {
				found = true
			}
			if !found {
				continue
			}
		}
		result = append(result, c)
	}
	return result, m.db.totalCount, nil
}

func (m *mockCapabilityRepository) Update(ctx context.Context, capability *model.Capability) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.capabilities == nil {
		return errors.NewNotFoundError("capability not found")
	}
	if _, ok := m.db.capabilities[capability.ID]; !ok {
		return errors.NewNotFoundError("capability not found")
	}
	m.db.capabilities[capability.ID] = capability
	return nil
}

func (m *mockCapabilityRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.db.deleteErr != nil {
		return m.db.deleteErr
	}
	if m.db.capabilities == nil {
		return errors.NewNotFoundError("capability not found")
	}
	if _, ok := m.db.capabilities[id.String()]; !ok {
		return errors.NewNotFoundError("capability not found")
	}
	delete(m.db.capabilities, id.String())
	return nil
}

// Helper function for substring matching in tests
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (containsSubstring(s[:len(s)-1], substr) || containsSubstring(s[1:], substr)) || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)
}

// Tests

func TestCapabilityRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockCapabilityDB{}
		repo := newMockCapabilityRepository(mockDB)

		tenantID := uuid.New().String()
		capability := &model.Capability{
			Type:        model.CapabilityTypeTool,
			Name:        "Test Tool",
			Description: "A test tool",
			TenantID:    &tenantID,
			Status:      model.CapabilityStatusActive,
		}

		err := repo.Create(ctx, capability)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.ID == "" {
			t.Error("Capability ID should be set after creation")
		}
	})

	t.Run("successful create global capability", func(t *testing.T) {
		mockDB := &mockCapabilityDB{}
		repo := newMockCapabilityRepository(mockDB)

		capability := &model.Capability{
			Type:        model.CapabilityTypeSkill,
			Name:        "Global Skill",
			Description: "A global skill",
			TenantID:    nil,
			Status:      model.CapabilityStatusActive,
		}

		err := repo.Create(ctx, capability)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if capability.TenantID != nil {
			t.Error("Global capability should have nil TenantID")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockCapabilityDB{createErr: expectedErr}
		repo := newMockCapabilityRepository(mockDB)

		capability := &model.Capability{
			Type:   model.CapabilityTypeTool,
			Name:   "Test Tool",
			Status: model.CapabilityStatusActive,
		}
		err := repo.Create(ctx, capability)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestCapabilityRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		tenantID := uuid.New().String()
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				existingID.String(): {
					ID:          existingID.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Existing Tool",
					Description: "An existing tool",
					TenantID:    &tenantID,
					Status:      model.CapabilityStatusActive,
				},
			},
		}
		repo := newMockCapabilityRepository(mockDB)

		capability, err := repo.GetByID(ctx, existingID)
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if capability == nil {
			t.Error("Expected capability, got nil")
		}
		if capability.Name != "Existing Tool" {
			t.Errorf("Expected name 'Existing Tool', got '%s'", capability.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockCapabilityDB{capabilities: make(map[string]*model.Capability)}
		repo := newMockCapabilityRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityRepository_List(t *testing.T) {
	ctx := context.Background()
	tenantID1 := uuid.New().String()
	_ = uuid.New().String() // tenantID2 reserved for future filter tests

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				id1.String(): {
					ID:          id1.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Tool 1",
					Description: "First tool",
					TenantID:    &tenantID1,
					Status:      model.CapabilityStatusActive,
				},
				id2.String(): {
					ID:          id2.String(),
					Type:        model.CapabilityTypeSkill,
					Name:        "Skill 1",
					Description: "First skill",
					TenantID:    &tenantID1,
					Status:      model.CapabilityStatusActive,
				},
			},
			totalCount: 2,
		}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10, TenantID: tenantID1}
		capabilities, total, err := repo.List(ctx, filter)
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

	t.Run("list with type filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				id1.String(): {
					ID:          id1.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Coding Tool",
					Description: "A tool for coding",
					TenantID:    &tenantID1,
					Status:      model.CapabilityStatusActive,
				},
				id2.String(): {
					ID:          id2.String(),
					Type:        model.CapabilityTypeSkill,
					Name:        "Ops Skill",
					Description: "A skill for ops",
					TenantID:    &tenantID1,
					Status:      model.CapabilityStatusActive,
				},
			},
			totalCount: 1,
		}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10, Type: string(model.CapabilityTypeTool)}
		capabilities, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(capabilities) != 1 {
			t.Errorf("Expected 1 capability, got %d", len(capabilities))
		}
		if capabilities[0].Type != model.CapabilityTypeTool {
			t.Errorf("Expected type 'tool', got '%s'", capabilities[0].Type)
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				id1.String(): {
					ID:          id1.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Active Tool",
					Status:      model.CapabilityStatusActive,
				},
				id2.String(): {
					ID:          id2.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Inactive Tool",
					Status:      model.CapabilityStatusInactive,
				},
			},
			totalCount: 1,
		}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10, Status: string(model.CapabilityStatusActive)}
		capabilities, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(capabilities) != 1 {
			t.Errorf("Expected 1 capability, got %d", len(capabilities))
		}
		if capabilities[0].Status != model.CapabilityStatusActive {
			t.Errorf("Expected status 'active', got '%s'", capabilities[0].Status)
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
	})

	t.Run("list with global tenant filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				id1.String(): {
					ID:          id1.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Global Tool",
					Description: "A global tool",
					TenantID:    nil,
					Status:      model.CapabilityStatusActive,
				},
				id2.String(): {
					ID:          id2.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Tenant Tool",
					TenantID:    &tenantID1,
					Status:      model.CapabilityStatusActive,
				},
			},
			totalCount: 1,
		}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10, TenantID: "global"}
		capabilities, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(capabilities) != 1 {
			t.Errorf("Expected 1 capability, got %d", len(capabilities))
		}
		if capabilities[0].TenantID != nil {
			t.Error("Expected global capability with nil TenantID")
		}
		if total != 1 {
			t.Errorf("Expected total 1, got %d", total)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockCapabilityDB{capabilities: nil}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10}
		capabilities, total, err := repo.List(ctx, filter)
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

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockCapabilityDB{listErr: expectedErr}
		repo := newMockCapabilityRepository(mockDB)

		filter := CapabilityFilter{Page: 1, PageSize: 10}
		_, _, err := repo.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestCapabilityRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New().String()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				existingID.String(): {
					ID:          existingID.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Old Name",
					Description: "Old description",
					TenantID:    &tenantID,
					Status:      model.CapabilityStatusActive,
				},
			},
		}
		repo := newMockCapabilityRepository(mockDB)

		capability := &model.Capability{
			ID:          existingID.String(),
			Type:        model.CapabilityTypeTool,
			Name:        "New Name",
			Description: "New description",
			TenantID:    &tenantID,
			Status:      model.CapabilityStatusInactive,
		}
		err := repo.Update(ctx, capability)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID)
		if updated.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
		}
		if updated.Description != "New description" {
			t.Errorf("Expected description 'New description', got '%s'", updated.Description)
		}
		if updated.Status != model.CapabilityStatusInactive {
			t.Errorf("Expected status 'inactive', got '%s'", updated.Status)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockCapabilityDB{capabilities: make(map[string]*model.Capability)}
		repo := newMockCapabilityRepository(mockDB)

		capability := &model.Capability{
			ID:     uuid.New().String(),
			Type:   model.CapabilityTypeTool,
			Name:   "New Name",
			Status: model.CapabilityStatusActive,
		}
		err := repo.Update(ctx, capability)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityRepository_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New().String()

	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockCapabilityDB{
			capabilities: map[string]*model.Capability{
				existingID.String(): {
					ID:          existingID.String(),
					Type:        model.CapabilityTypeTool,
					Name:        "Tool to Delete",
					TenantID:    &tenantID,
					Status:      model.CapabilityStatusActive,
				},
			},
		}
		repo := newMockCapabilityRepository(mockDB)

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
		mockDB := &mockCapabilityDB{capabilities: make(map[string]*model.Capability)}
		repo := newMockCapabilityRepository(mockDB)

		err := repo.Delete(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestCapabilityFilter_Fields(t *testing.T) {
	// Verify filter fields exist
	filter := CapabilityFilter{}

	_ = filter.Page
	_ = filter.PageSize
	_ = filter.TenantID
	_ = filter.Type
	_ = filter.Status
	_ = filter.Search
}

func TestCapabilityFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := CapabilityFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := CapabilityFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := CapabilityFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})
}

func TestCapabilityFilter_Offset(t *testing.T) {
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
		filter := CapabilityFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestCapabilityRepository_Interface(t *testing.T) {
	// Verify that mockCapabilityRepository implements CapabilityRepository interface
	var _ CapabilityRepository = (*mockCapabilityRepository)(nil)
}

func TestNewCapabilityRepository(t *testing.T) {
	// Test that NewCapabilityRepository returns a non-nil implementation
	repo := NewCapabilityRepository(nil)
	if repo == nil {
		t.Error("NewCapabilityRepository should return non-nil interface value")
	}
}

// Verify error type checking works
func TestCapabilityRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("capability not found")
	if !stderrors.Is(err, errors.ErrNotFound) {
		t.Error("Error should match ErrNotFound")
	}

	internalErr := errors.NewInternalError("something went wrong")
	if !stderrors.Is(internalErr, errors.ErrInternal) {
		t.Error("Error should match ErrInternal")
	}
}
