package repository

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockDB simulates GORM DB for testing purposes
type mockDB struct {
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error
	tenants    map[uuid.UUID]*model.Tenant
	totalCount int64
}

// mockTenantRepository is a mock implementation for testing
type mockTenantRepository struct {
	db *mockDB
}

func newMockTenantRepository(db *mockDB) *mockTenantRepository {
	return &mockTenantRepository{db: db}
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.tenants == nil {
		m.db.tenants = make(map[uuid.UUID]*model.Tenant)
	}
	tenant.ID = uuid.New()
	m.db.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.tenants == nil {
		return nil, errors.NewNotFoundError("tenant not found")
	}
	tenant, ok := m.db.tenants[id]
	if !ok {
		return nil, errors.NewNotFoundError("tenant not found")
	}
	return tenant, nil
}

func (m *mockTenantRepository) List(ctx context.Context, filter TenantFilter) ([]*model.Tenant, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.tenants == nil {
		return []*model.Tenant{}, 0, nil
	}

	var result []*model.Tenant
	for _, t := range m.db.tenants {
		result = append(result, t)
	}
	return result, m.db.totalCount, nil
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.tenants == nil {
		return errors.NewNotFoundError("tenant not found")
	}
	if _, ok := m.db.tenants[tenant.ID]; !ok {
		return errors.NewNotFoundError("tenant not found")
	}
	m.db.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.db.deleteErr != nil {
		return m.db.deleteErr
	}
	if m.db.tenants == nil {
		return errors.NewNotFoundError("tenant not found")
	}
	if _, ok := m.db.tenants[id]; !ok {
		return errors.NewNotFoundError("tenant not found")
	}
	delete(m.db.tenants, id)
	return nil
}

// Tests

func TestTenantRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockDB{}
		repo := newMockTenantRepository(mockDB)

		tenant := &model.Tenant{
			Name:    "Test Tenant",
			Status:  model.TenantStatusActive,
			QuotaCPU: 4,
		}

		err := repo.Create(ctx, tenant)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if tenant.ID == uuid.Nil {
			t.Error("Tenant ID should be set after creation")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockDB{createErr: expectedErr}
		repo := newMockTenantRepository(mockDB)

		tenant := &model.Tenant{Name: "Test Tenant"}
		err := repo.Create(ctx, tenant)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTenantRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockDB{
			tenants: map[uuid.UUID]*model.Tenant{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Existing Tenant"},
			},
		}
		repo := newMockTenantRepository(mockDB)

		tenant, err := repo.GetByID(ctx, existingID)
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if tenant == nil {
			t.Error("Expected tenant, got nil")
		}
		if tenant.Name != "Existing Tenant" {
			t.Errorf("Expected name 'Existing Tenant', got '%s'", tenant.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockDB{tenants: make(map[uuid.UUID]*model.Tenant)}
		repo := newMockTenantRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantRepository_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockDB{
			tenants: map[uuid.UUID]*model.Tenant{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Tenant 1"},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Tenant 2"},
			},
			totalCount: 2,
		}
		repo := newMockTenantRepository(mockDB)

		filter := TenantFilter{Page: 1, PageSize: 10}
		tenants, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tenants) != 2 {
			t.Errorf("Expected 2 tenants, got %d", len(tenants))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockDB{tenants: nil}
		repo := newMockTenantRepository(mockDB)

		filter := TenantFilter{Page: 1, PageSize: 10}
		tenants, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tenants) != 0 {
			t.Errorf("Expected 0 tenants, got %d", len(tenants))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockDB{listErr: expectedErr}
		repo := newMockTenantRepository(mockDB)

		filter := TenantFilter{Page: 1, PageSize: 10}
		_, _, err := repo.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTenantRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockDB{
			tenants: map[uuid.UUID]*model.Tenant{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Old Name"},
			},
		}
		repo := newMockTenantRepository(mockDB)

		tenant := &model.Tenant{BaseModel: model.BaseModel{ID: existingID}, Name: "New Name"}
		err := repo.Update(ctx, tenant)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID)
		if updated.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockDB{tenants: make(map[uuid.UUID]*model.Tenant)}
		repo := newMockTenantRepository(mockDB)

		tenant := &model.Tenant{BaseModel: model.BaseModel{ID: uuid.New()}, Name: "New Name"}
		err := repo.Update(ctx, tenant)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantRepository_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockDB{
			tenants: map[uuid.UUID]*model.Tenant{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Tenant to Delete"},
			},
		}
		repo := newMockTenantRepository(mockDB)

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
		mockDB := &mockDB{tenants: make(map[uuid.UUID]*model.Tenant)}
		repo := newMockTenantRepository(mockDB)

		err := repo.Delete(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantFilter_Defaults(t *testing.T) {
	filter := TenantFilter{}

	// Verify filter fields exist
	_ = filter.Page
	_ = filter.PageSize
	_ = filter.Status
	_ = filter.Search
}

func TestTenantFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := TenantFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := TenantFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := TenantFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		filter := TenantFilter{Page: -1, PageSize: -5}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})
}

func TestTenantFilter_Offset(t *testing.T) {
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
		filter := TenantFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestTenantRepository_Interface(t *testing.T) {
	// Verify that mockTenantRepository implements TenantRepository interface
	var _ TenantRepository = (*mockTenantRepository)(nil)
}

// Verify error type checking works
func TestTenantRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("tenant not found")
	if !stderrors.Is(err, errors.ErrNotFound) {
		t.Error("Error should match ErrNotFound")
	}

	internalErr := errors.NewInternalError("something went wrong")
	if !stderrors.Is(internalErr, errors.ErrInternal) {
		t.Error("Error should match ErrInternal")
	}
}
