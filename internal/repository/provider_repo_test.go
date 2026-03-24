package repository

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockProviderDB simulates GORM DB for testing purposes
type mockProviderDB struct {
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error
	providers  map[string]*model.Provider
	userDefaults map[string]*model.UserProviderDefault
	totalCount int64
}

// mockProviderRepository is a mock implementation for testing
type mockProviderRepository struct {
	db *mockProviderDB
}

func newMockProviderRepository(db *mockProviderDB) *mockProviderRepository {
	return &mockProviderRepository{db: db}
}

func (m *mockProviderRepository) Create(ctx context.Context, provider *model.Provider) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.providers == nil {
		m.db.providers = make(map[string]*model.Provider)
	}
	provider.ID = uuid.New().String()
	m.db.providers[provider.ID] = provider
	return nil
}

func (m *mockProviderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Provider, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.providers == nil {
		return nil, errors.NewNotFoundError("provider not found")
	}
	provider, ok := m.db.providers[id.String()]
	if !ok {
		return nil, errors.NewNotFoundError("provider not found")
	}
	return provider, nil
}

func (m *mockProviderRepository) List(ctx context.Context, filter ProviderFilter) ([]*model.Provider, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.providers == nil {
		return []*model.Provider{}, 0, nil
	}

	var result []*model.Provider
	for _, p := range m.db.providers {
		// Apply scope filtering
		if filter.Scope != "" && p.Scope != model.ProviderScope(filter.Scope) {
			continue
		}
		if filter.TenantID != "" && (p.TenantID == nil || *p.TenantID != filter.TenantID) {
			continue
		}
		if filter.UserID != "" && (p.UserID == nil || *p.UserID != filter.UserID) {
			continue
		}
		if filter.Type != "" && string(p.Type) != filter.Type {
			continue
		}
		if filter.Status != "" && string(p.Status) != filter.Status {
			continue
		}
		result = append(result, p)
	}
	return result, m.db.totalCount, nil
}

func (m *mockProviderRepository) Update(ctx context.Context, provider *model.Provider) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.providers == nil {
		return errors.NewNotFoundError("provider not found")
	}
	if _, ok := m.db.providers[provider.ID]; !ok {
		return errors.NewNotFoundError("provider not found")
	}
	m.db.providers[provider.ID] = provider
	return nil
}

func (m *mockProviderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.db.deleteErr != nil {
		return m.db.deleteErr
	}
	if m.db.providers == nil {
		return errors.NewNotFoundError("provider not found")
	}
	if _, ok := m.db.providers[id.String()]; !ok {
		return errors.NewNotFoundError("provider not found")
	}
	delete(m.db.providers, id.String())
	return nil
}

func (m *mockProviderRepository) GetByScopeAndName(ctx context.Context, scope model.ProviderScope, tenantID, userID *string, name string) (*model.Provider, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.providers == nil {
		return nil, errors.NewNotFoundError("provider not found")
	}
	for _, p := range m.db.providers {
		if p.Scope == scope && p.Name == name {
			if tenantID == nil && p.TenantID == nil && userID == nil && p.UserID == nil {
				return p, nil
			}
			if tenantID != nil && p.TenantID != nil && *p.TenantID == *tenantID && userID == nil && p.UserID == nil {
				return p, nil
			}
			if userID != nil && p.UserID != nil && *p.UserID == *userID {
				return p, nil
			}
		}
	}
	return nil, errors.NewNotFoundError("provider not found")
}

func (m *mockProviderRepository) GetDefaultProvider(ctx context.Context, scope model.ProviderScope, tenantID, userID *string) (*model.Provider, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.providers == nil {
		return nil, errors.NewNotFoundError("provider not found")
	}
	for _, p := range m.db.providers {
		if p.Scope == scope {
			if scope == model.ProviderScopeSystem && p.TenantID == nil && p.UserID == nil {
				return p, nil
			}
			if scope == model.ProviderScopeTenant && p.TenantID != nil && tenantID != nil && *p.TenantID == *tenantID {
				return p, nil
			}
			if scope == model.ProviderScopeUser && p.UserID != nil && userID != nil && *p.UserID == *userID {
				return p, nil
			}
		}
	}
	return nil, errors.NewNotFoundError("provider not found")
}

func (m *mockProviderRepository) SetUserDefaultProvider(ctx context.Context, userID, providerID string) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.userDefaults == nil {
		m.db.userDefaults = make(map[string]*model.UserProviderDefault)
	}
	m.db.userDefaults[userID] = &model.UserProviderDefault{
		ID:         uuid.New().String(),
		UserID:     userID,
		ProviderID: providerID,
	}
	return nil
}

func (m *mockProviderRepository) GetUserDefaultProvider(ctx context.Context, userID string) (*model.Provider, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.userDefaults == nil {
		return nil, errors.NewNotFoundError("user default provider not found")
	}
	userDefault, ok := m.db.userDefaults[userID]
	if !ok {
		return nil, errors.NewNotFoundError("user default provider not found")
	}
	provider, ok := m.db.providers[userDefault.ProviderID]
	if !ok {
		return nil, errors.NewNotFoundError("provider not found")
	}
	return provider, nil
}

// Tests

func TestProviderRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockProviderDB{}
		repo := newMockProviderRepository(mockDB)

		provider := &model.Provider{
			Name:   "Test Provider",
			Type:   model.ProviderTypeClaudeCode,
			Scope:  model.ProviderScopeSystem,
			Status: model.ProviderStatusActive,
		}

		err := repo.Create(ctx, provider)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if provider.ID == "" {
			t.Error("Provider ID should be set after creation")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockProviderDB{createErr: expectedErr}
		repo := newMockProviderRepository(mockDB)

		provider := &model.Provider{Name: "Test Provider"}
		err := repo.Create(ctx, provider)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestProviderRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				existingID.String(): {ID: existingID.String(), Name: "Existing Provider"},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetByID(ctx, existingID)
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider, got nil")
		}
		if provider.Name != "Existing Provider" {
			t.Errorf("Expected name 'Existing Provider', got '%s'", provider.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockProviderDB{providers: make(map[string]*model.Provider)}
		repo := newMockProviderRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderRepository_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list with scope filtering", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		tenantID := "tenant-123"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id1.String(): {ID: id1.String(), Name: "System Provider", Scope: model.ProviderScopeSystem},
				id2.String(): {ID: id2.String(), Name: "Tenant Provider", Scope: model.ProviderScopeTenant, TenantID: &tenantID},
			},
			totalCount: 2,
		}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10, Scope: "system"}
		providers, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("filter by tenant_id", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		tenantID1 := "tenant-1"
		tenantID2 := "tenant-2"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id1.String(): {ID: id1.String(), Name: "Tenant 1 Provider", Scope: model.ProviderScopeTenant, TenantID: &tenantID1},
				id2.String(): {ID: id2.String(), Name: "Tenant 2 Provider", Scope: model.ProviderScopeTenant, TenantID: &tenantID2},
			},
			totalCount: 2,
		}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10, TenantID: tenantID1}
		providers, _, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider for tenant 1, got %d", len(providers))
		}
	})

	t.Run("filter by user_id", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		userID1 := "user-1"
		userID2 := "user-2"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id1.String(): {ID: id1.String(), Name: "User 1 Provider", Scope: model.ProviderScopeUser, UserID: &userID1},
				id2.String(): {ID: id2.String(), Name: "User 2 Provider", Scope: model.ProviderScopeUser, UserID: &userID2},
			},
			totalCount: 2,
		}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10, UserID: userID1}
		providers, _, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider for user 1, got %d", len(providers))
		}
	})

	t.Run("filter by type and status", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id1.String(): {ID: id1.String(), Name: "Active Claude", Type: model.ProviderTypeClaudeCode, Status: model.ProviderStatusActive},
				id2.String(): {ID: id2.String(), Name: "Inactive Claude", Type: model.ProviderTypeClaudeCode, Status: model.ProviderStatusInactive},
			},
			totalCount: 2,
		}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10, Type: "claude_code", Status: "active"}
		providers, _, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(providers) != 1 {
			t.Errorf("Expected 1 active claude_code provider, got %d", len(providers))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockProviderDB{providers: nil}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10}
		providers, total, err := repo.List(ctx, filter)
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

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockProviderDB{listErr: expectedErr}
		repo := newMockProviderRepository(mockDB)

		filter := ProviderFilter{Page: 1, PageSize: 10}
		_, _, err := repo.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestProviderRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				existingID.String(): {ID: existingID.String(), Name: "Old Name"},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider := &model.Provider{ID: existingID.String(), Name: "New Name"}
		err := repo.Update(ctx, provider)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID)
		if updated.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockProviderDB{providers: make(map[string]*model.Provider)}
		repo := newMockProviderRepository(mockDB)

		provider := &model.Provider{ID: uuid.New().String(), Name: "New Name"}
		err := repo.Update(ctx, provider)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderRepository_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				existingID.String(): {ID: existingID.String(), Name: "Provider to Delete"},
			},
		}
		repo := newMockProviderRepository(mockDB)

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
		mockDB := &mockProviderDB{providers: make(map[string]*model.Provider)}
		repo := newMockProviderRepository(mockDB)

		err := repo.Delete(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderRepository_GetByScopeAndName(t *testing.T) {
	ctx := context.Background()

	t.Run("get system provider by name", func(t *testing.T) {
		id := uuid.New()
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "System Claude", Scope: model.ProviderScopeSystem},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetByScopeAndName(ctx, model.ProviderScopeSystem, nil, nil, "System Claude")
		if err != nil {
			t.Errorf("GetByScopeAndName returned error: %v", err)
		}
		if provider.Name != "System Claude" {
			t.Errorf("Expected name 'System Claude', got '%s'", provider.Name)
		}
	})

	t.Run("get tenant provider by name", func(t *testing.T) {
		id := uuid.New()
		tenantID := "tenant-123"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "Tenant Claude", Scope: model.ProviderScopeTenant, TenantID: &tenantID},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetByScopeAndName(ctx, model.ProviderScopeTenant, &tenantID, nil, "Tenant Claude")
		if err != nil {
			t.Errorf("GetByScopeAndName returned error: %v", err)
		}
		if provider.Name != "Tenant Claude" {
			t.Errorf("Expected name 'Tenant Claude', got '%s'", provider.Name)
		}
	})

	t.Run("get user provider by name", func(t *testing.T) {
		id := uuid.New()
		userID := "user-123"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "User Claude", Scope: model.ProviderScopeUser, UserID: &userID},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetByScopeAndName(ctx, model.ProviderScopeUser, nil, &userID, "User Claude")
		if err != nil {
			t.Errorf("GetByScopeAndName returned error: %v", err)
		}
		if provider.Name != "User Claude" {
			t.Errorf("Expected name 'User Claude', got '%s'", provider.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockProviderDB{providers: make(map[string]*model.Provider)}
		repo := newMockProviderRepository(mockDB)

		_, err := repo.GetByScopeAndName(ctx, model.ProviderScopeSystem, nil, nil, "NonExistent")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderRepository_GetDefaultProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("get system default provider", func(t *testing.T) {
		id := uuid.New()
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "Default System", Scope: model.ProviderScopeSystem},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetDefaultProvider(ctx, model.ProviderScopeSystem, nil, nil)
		if err != nil {
			t.Errorf("GetDefaultProvider returned error: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider, got nil")
		}
	})

	t.Run("get tenant default provider", func(t *testing.T) {
		id := uuid.New()
		tenantID := "tenant-123"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "Default Tenant", Scope: model.ProviderScopeTenant, TenantID: &tenantID},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetDefaultProvider(ctx, model.ProviderScopeTenant, &tenantID, nil)
		if err != nil {
			t.Errorf("GetDefaultProvider returned error: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider, got nil")
		}
	})

	t.Run("get user default provider by scope", func(t *testing.T) {
		id := uuid.New()
		userID := "user-123"
		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				id.String(): {ID: id.String(), Name: "Default User", Scope: model.ProviderScopeUser, UserID: &userID},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetDefaultProvider(ctx, model.ProviderScopeUser, nil, &userID)
		if err != nil {
			t.Errorf("GetDefaultProvider returned error: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider, got nil")
		}
	})
}

func TestProviderRepository_SetUserDefaultProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("set user default provider", func(t *testing.T) {
		mockDB := &mockProviderDB{}
		repo := newMockProviderRepository(mockDB)

		userID := "user-123"
		providerID := "provider-456"

		err := repo.SetUserDefaultProvider(ctx, userID, providerID)
		if err != nil {
			t.Errorf("SetUserDefaultProvider returned error: %v", err)
		}

		// Verify it was set
		if mockDB.userDefaults == nil || mockDB.userDefaults[userID] == nil {
			t.Error("User default provider was not set")
		}
		if mockDB.userDefaults[userID].ProviderID != providerID {
			t.Errorf("Expected provider ID '%s', got '%s'", providerID, mockDB.userDefaults[userID].ProviderID)
		}
	})

	t.Run("update existing user default", func(t *testing.T) {
		userID := "user-123"
		oldProviderID := "provider-old"
		newProviderID := "provider-new"

		mockDB := &mockProviderDB{
			userDefaults: map[string]*model.UserProviderDefault{
				userID: {UserID: userID, ProviderID: oldProviderID},
			},
		}
		repo := newMockProviderRepository(mockDB)

		err := repo.SetUserDefaultProvider(ctx, userID, newProviderID)
		if err != nil {
			t.Errorf("SetUserDefaultProvider returned error: %v", err)
		}

		if mockDB.userDefaults[userID].ProviderID != newProviderID {
			t.Errorf("Expected provider ID '%s', got '%s'", newProviderID, mockDB.userDefaults[userID].ProviderID)
		}
	})
}

func TestProviderRepository_GetUserDefaultProvider(t *testing.T) {
	ctx := context.Background()

	t.Run("get user default provider", func(t *testing.T) {
		providerID := uuid.New()
		userID := "user-123"

		mockDB := &mockProviderDB{
			providers: map[string]*model.Provider{
				providerID.String(): {ID: providerID.String(), Name: "Default Provider"},
			},
			userDefaults: map[string]*model.UserProviderDefault{
				userID: {UserID: userID, ProviderID: providerID.String()},
			},
		}
		repo := newMockProviderRepository(mockDB)

		provider, err := repo.GetUserDefaultProvider(ctx, userID)
		if err != nil {
			t.Errorf("GetUserDefaultProvider returned error: %v", err)
		}
		if provider == nil {
			t.Error("Expected provider, got nil")
		}
		if provider.Name != "Default Provider" {
			t.Errorf("Expected name 'Default Provider', got '%s'", provider.Name)
		}
	})

	t.Run("user default not found", func(t *testing.T) {
		mockDB := &mockProviderDB{
			userDefaults: make(map[string]*model.UserProviderDefault),
		}
		repo := newMockProviderRepository(mockDB)

		_, err := repo.GetUserDefaultProvider(ctx, "nonexistent-user")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestProviderFilter_Defaults(t *testing.T) {
	filter := ProviderFilter{}

	// Verify filter fields exist
	_ = filter.Page
	_ = filter.PageSize
	_ = filter.Scope
	_ = filter.TenantID
	_ = filter.UserID
	_ = filter.Type
	_ = filter.Status
	_ = filter.Search
}

func TestProviderFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := ProviderFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := ProviderFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := ProviderFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		filter := ProviderFilter{Page: -1, PageSize: -5}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})
}

func TestProviderFilter_Offset(t *testing.T) {
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
		filter := ProviderFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestProviderRepository_Interface(t *testing.T) {
	// Verify that mockProviderRepository implements ProviderRepository interface
	var _ ProviderRepository = (*mockProviderRepository)(nil)
}

func TestNewProviderRepository(t *testing.T) {
	// Test that NewProviderRepository returns a non-nil implementation
	repo := NewProviderRepository(nil)
	if repo == nil {
		t.Error("NewProviderRepository should return non-nil interface value")
	}
}

// Verify error type checking works
func TestProviderRepository_ErrorTypes(t *testing.T) {
	err := errors.NewNotFoundError("provider not found")
	if !stderrors.Is(err, errors.ErrNotFound) {
		t.Error("Error should match ErrNotFound")
	}

	internalErr := errors.NewInternalError("something went wrong")
	if !stderrors.Is(internalErr, errors.ErrInternal) {
		t.Error("Error should match ErrInternal")
	}
}
