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

// mockTenantRepository implements repository.TenantRepository for testing
type mockTenantRepository struct {
	createFunc  func(ctx context.Context, tenant *model.Tenant) error
	getByIDFunc func(ctx context.Context, id uuid.UUID) (*model.Tenant, error)
	listFunc    func(ctx context.Context, filter repository.TenantFilter) ([]*model.Tenant, int64, error)
	updateFunc  func(ctx context.Context, tenant *model.Tenant) error
	deleteFunc  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockTenantRepository) Create(ctx context.Context, tenant *model.Tenant) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTenantRepository) List(ctx context.Context, filter repository.TenantFilter) ([]*model.Tenant, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTenantRepository) Update(ctx context.Context, tenant *model.Tenant) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, tenant)
	}
	return nil
}

func (m *mockTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func TestTenantService_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		repo := &mockTenantRepository{
			createFunc: func(ctx context.Context, tenant *model.Tenant) error {
				tenant.ID = uuid.New()
				return nil
			},
		}
		service := NewTenantService(repo)

		req := &CreateTenantRequest{
			Name:             "Test Tenant",
			QuotaCPU:         4,
			QuotaMemory:      16,
			QuotaConcurrency: 10,
			QuotaDailyTasks:  100,
		}

		tenant, err := service.Create(ctx, req)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if tenant == nil {
			t.Fatal("Expected tenant, got nil")
		}
		if tenant.Name != "Test Tenant" {
			t.Errorf("Expected name 'Test Tenant', got '%s'", tenant.Name)
		}
		if tenant.QuotaCPU != 4 {
			t.Errorf("Expected QuotaCPU 4, got %d", tenant.QuotaCPU)
		}
		if tenant.Status != model.TenantStatusActive {
			t.Errorf("Expected status '%s', got '%s'", model.TenantStatusActive, tenant.Status)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		repo := &mockTenantRepository{}
		service := NewTenantService(repo)

		req := &CreateTenantRequest{
			Name: "",
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("negative quota values", func(t *testing.T) {
		repo := &mockTenantRepository{}
		service := NewTenantService(repo)

		req := &CreateTenantRequest{
			Name:        "Test Tenant",
			QuotaCPU:    -1,
			QuotaMemory: -10,
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error for negative quota, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTenantRepository{
			createFunc: func(ctx context.Context, tenant *model.Tenant) error {
				return errors.NewInternalError("database error")
			},
		}
		service := NewTenantService(repo)

		req := &CreateTenantRequest{
			Name: "Test Tenant",
		}

		_, err := service.Create(ctx, req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTenantService_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				if id == existingID {
					return &model.Tenant{
						BaseModel: model.BaseModel{ID: existingID},
						Name:      "Test Tenant",
						Status:    model.TenantStatusActive,
					}, nil
				}
				return nil, errors.NewNotFoundError("tenant not found")
			},
		}
		service := NewTenantService(repo)

		tenant, err := service.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if tenant == nil {
			t.Fatal("Expected tenant, got nil")
		}
		if tenant.Name != "Test Tenant" {
			t.Errorf("Expected name 'Test Tenant', got '%s'", tenant.Name)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTenantRepository{}
		service := NewTenantService(repo)

		_, err := service.GetByID(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return nil, errors.NewNotFoundError("tenant not found")
			},
		}
		service := NewTenantService(repo)

		_, err := service.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		repo := &mockTenantRepository{
			listFunc: func(ctx context.Context, filter repository.TenantFilter) ([]*model.Tenant, int64, error) {
				return []*model.Tenant{
					{BaseModel: model.BaseModel{ID: id1}, Name: "Tenant 1"},
					{BaseModel: model.BaseModel{ID: id2}, Name: "Tenant 2"},
				}, 2, nil
			},
		}
		service := NewTenantService(repo)

		filter := &TenantFilter{Page: 1, PageSize: 10}
		tenants, total, err := service.List(ctx, filter)
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
		repo := &mockTenantRepository{
			listFunc: func(ctx context.Context, filter repository.TenantFilter) ([]*model.Tenant, int64, error) {
				return []*model.Tenant{}, 0, nil
			},
		}
		service := NewTenantService(repo)

		filter := &TenantFilter{Page: 1, PageSize: 10}
		tenants, total, err := service.List(ctx, filter)
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

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTenantRepository{
			listFunc: func(ctx context.Context, filter repository.TenantFilter) ([]*model.Tenant, int64, error) {
				return nil, 0, errors.NewInternalError("database error")
			},
		}
		service := NewTenantService(repo)

		filter := &TenantFilter{Page: 1, PageSize: 10}
		_, _, err := service.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTenantService_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update with name", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return &model.Tenant{
					BaseModel: model.BaseModel{ID: existingID},
					Name:      "Old Name",
					Status:    model.TenantStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, tenant *model.Tenant) error {
				return nil
			},
		}
		service := NewTenantService(repo)

		newName := "New Name"
		req := &UpdateTenantRequest{
			Name: &newName,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with quotas", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return &model.Tenant{
					BaseModel:        model.BaseModel{ID: existingID},
					Name:             "Test Tenant",
					QuotaCPU:         4,
					QuotaMemory:      16,
					QuotaConcurrency: 10,
					QuotaDailyTasks:  100,
					Status:           model.TenantStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, tenant *model.Tenant) error {
				return nil
			},
		}
		service := NewTenantService(repo)

		quotaCPU := 8
		quotaMemory := int64(32)
		req := &UpdateTenantRequest{
			QuotaCPU:    &quotaCPU,
			QuotaMemory: &quotaMemory,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("successful update with status", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return &model.Tenant{
					BaseModel: model.BaseModel{ID: existingID},
					Name:      "Test Tenant",
					Status:    model.TenantStatusActive,
				}, nil
			},
			updateFunc: func(ctx context.Context, tenant *model.Tenant) error {
				return nil
			},
		}
		service := NewTenantService(repo)

		status := model.TenantStatusSuspended
		req := &UpdateTenantRequest{
			Status: &status,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTenantRepository{}
		service := NewTenantService(repo)

		req := &UpdateTenantRequest{}
		err := service.Update(ctx, "invalid-uuid", req)
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("negative quota values", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return &model.Tenant{
					BaseModel: model.BaseModel{ID: existingID},
					Name:      "Test Tenant",
					Status:    model.TenantStatusActive,
				}, nil
			},
		}
		service := NewTenantService(repo)

		quotaCPU := -1
		req := &UpdateTenantRequest{
			QuotaCPU: &quotaCPU,
		}

		err := service.Update(ctx, existingID.String(), req)
		if err == nil {
			t.Error("Expected error for negative quota, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return &model.Tenant{
					BaseModel: model.BaseModel{ID: existingID},
					Name:      "Test Tenant",
					Status:    model.TenantStatusActive,
				}, nil
			},
		}
		service := NewTenantService(repo)

		status := "invalid-status"
		req := &UpdateTenantRequest{
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

	t.Run("not found", func(t *testing.T) {
		repo := &mockTenantRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Tenant, error) {
				return nil, errors.NewNotFoundError("tenant not found")
			},
		}
		service := NewTenantService(repo)

		req := &UpdateTenantRequest{}
		err := service.Update(ctx, uuid.New().String(), req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantService_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		repo := &mockTenantRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		service := NewTenantService(repo)

		err := service.Delete(ctx, existingID.String())
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTenantRepository{}
		service := NewTenantService(repo)

		err := service.Delete(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTenantRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return errors.NewNotFoundError("tenant not found")
			},
		}
		service := NewTenantService(repo)

		err := service.Delete(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTenantService_Interface(t *testing.T) {
	// Verify that tenantService implements TenantService interface
	var _ TenantService = (*tenantService)(nil)
}
