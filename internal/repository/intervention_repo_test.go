package repository

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// mockInterventionDB simulates GORM DB for testing purposes
type mockInterventionDB struct {
	createErr     error
	findErr       error
	listErr       error
	updateErr     error
	interventions map[string]*model.Intervention
	totalCount    int64
}

// mockInterventionRepository is a mock implementation for testing
type mockInterventionRepository struct {
	db *mockInterventionDB
}

func newMockInterventionRepository(db *mockInterventionDB) *mockInterventionRepository {
	return &mockInterventionRepository{db: db}
}

func (m *mockInterventionRepository) Create(ctx context.Context, intervention *model.Intervention) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.interventions == nil {
		m.db.interventions = make(map[string]*model.Intervention)
	}
	intervention.ID = uuid.New()
	m.db.interventions[intervention.ID.String()] = intervention
	return nil
}

func (m *mockInterventionRepository) GetByID(ctx context.Context, id string) (*model.Intervention, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.interventions == nil {
		return nil, errors.NewNotFoundError("intervention not found")
	}
	intervention, ok := m.db.interventions[id]
	if !ok {
		return nil, errors.NewNotFoundError("intervention not found")
	}
	return intervention, nil
}

func (m *mockInterventionRepository) ListByTask(ctx context.Context, taskID string, filter InterventionFilter) ([]*model.Intervention, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.interventions == nil {
		return []*model.Intervention{}, 0, nil
	}

	var result []*model.Intervention
	for _, i := range m.db.interventions {
		// Filter by task ID
		if i.TaskID != taskID {
			continue
		}
		// Apply filters
		if filter.Action != "" && string(i.Action) != filter.Action {
			continue
		}
		if filter.Status != "" && string(i.Status) != filter.Status {
			continue
		}
		if filter.OperatorID != "" && i.OperatorID != filter.OperatorID {
			continue
		}
		result = append(result, i)
	}
	return result, m.db.totalCount, nil
}

func (m *mockInterventionRepository) Update(ctx context.Context, intervention *model.Intervention) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.interventions == nil {
		return errors.NewNotFoundError("intervention not found")
	}
	id := intervention.ID.String()
	if _, ok := m.db.interventions[id]; !ok {
		return errors.NewNotFoundError("intervention not found")
	}
	m.db.interventions[id] = intervention
	return nil
}

// Tests

func TestInterventionRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockInterventionDB{}
		repo := newMockInterventionRepository(mockDB)

		intervention := &model.Intervention{
			TaskID:     "task-123",
			OperatorID: "operator-456",
			Action:     model.InterventionActionPause,
			Status:     model.InterventionStatusPending,
		}

		err := repo.Create(ctx, intervention)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if intervention.ID == uuid.Nil {
			t.Error("Intervention ID should be set after creation")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockInterventionDB{createErr: expectedErr}
		repo := newMockInterventionRepository(mockDB)

		intervention := &model.Intervention{TaskID: "task-123"}
		err := repo.Create(ctx, intervention)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestInterventionRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				existingID.String(): {
					BaseModel:  model.BaseModel{ID: existingID},
					TaskID:     "task-123",
					OperatorID: "operator-456",
					Action:     model.InterventionActionPause,
					Status:     model.InterventionStatusPending,
				},
			},
		}
		repo := newMockInterventionRepository(mockDB)

		intervention, err := repo.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if intervention == nil {
			t.Error("Expected intervention, got nil")
		}
		if intervention.TaskID != "task-123" {
			t.Errorf("Expected TaskID 'task-123', got '%s'", intervention.TaskID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockInterventionDB{interventions: make(map[string]*model.Intervention)}
		repo := newMockInterventionRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestInterventionRepository_ListByTask(t *testing.T) {
	ctx := context.Background()
	taskID := "task-123"
	otherTaskID := "task-456"

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				id1.String(): {BaseModel: model.BaseModel{ID: id1}, TaskID: taskID, Action: model.InterventionActionPause, Status: model.InterventionStatusPending},
				id2.String(): {BaseModel: model.BaseModel{ID: id2}, TaskID: taskID, Action: model.InterventionActionResume, Status: model.InterventionStatusApplied},
			},
			totalCount: 2,
		}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10}
		interventions, total, err := repo.ListByTask(ctx, taskID, filter)
		if err != nil {
			t.Errorf("ListByTask returned error: %v", err)
		}
		if len(interventions) != 2 {
			t.Errorf("Expected 2 interventions, got %d", len(interventions))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("list with action filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				id1.String(): {BaseModel: model.BaseModel{ID: id1}, TaskID: taskID, Action: model.InterventionActionPause, Status: model.InterventionStatusPending},
				id2.String(): {BaseModel: model.BaseModel{ID: id2}, TaskID: taskID, Action: model.InterventionActionResume, Status: model.InterventionStatusApplied},
			},
			totalCount: 1,
		}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10, Action: string(model.InterventionActionPause)}
		interventions, _, err := repo.ListByTask(ctx, taskID, filter)
		if err != nil {
			t.Errorf("ListByTask returned error: %v", err)
		}
		if len(interventions) != 1 {
			t.Errorf("Expected 1 intervention with pause action, got %d", len(interventions))
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				id1.String(): {BaseModel: model.BaseModel{ID: id1}, TaskID: taskID, Action: model.InterventionActionPause, Status: model.InterventionStatusPending},
				id2.String(): {BaseModel: model.BaseModel{ID: id2}, TaskID: taskID, Action: model.InterventionActionResume, Status: model.InterventionStatusApplied},
			},
			totalCount: 1,
		}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10, Status: string(model.InterventionStatusApplied)}
		interventions, _, err := repo.ListByTask(ctx, taskID, filter)
		if err != nil {
			t.Errorf("ListByTask returned error: %v", err)
		}
		if len(interventions) != 1 {
			t.Errorf("Expected 1 intervention with applied status, got %d", len(interventions))
		}
	})

	t.Run("list filters by task ID", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				id1.String(): {BaseModel: model.BaseModel{ID: id1}, TaskID: taskID, Action: model.InterventionActionPause},
				id2.String(): {BaseModel: model.BaseModel{ID: id2}, TaskID: otherTaskID, Action: model.InterventionActionResume},
			},
			totalCount: 1,
		}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10}
		interventions, _, err := repo.ListByTask(ctx, taskID, filter)
		if err != nil {
			t.Errorf("ListByTask returned error: %v", err)
		}
		if len(interventions) != 1 {
			t.Errorf("Expected 1 intervention for task-123, got %d", len(interventions))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockInterventionDB{interventions: nil}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10}
		interventions, total, err := repo.ListByTask(ctx, taskID, filter)
		if err != nil {
			t.Errorf("ListByTask returned error: %v", err)
		}
		if len(interventions) != 0 {
			t.Errorf("Expected 0 interventions, got %d", len(interventions))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockInterventionDB{listErr: expectedErr}
		repo := newMockInterventionRepository(mockDB)

		filter := InterventionFilter{Page: 1, PageSize: 10}
		_, _, err := repo.ListByTask(ctx, taskID, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestInterventionRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockInterventionDB{
			interventions: map[string]*model.Intervention{
				existingID.String(): {
					BaseModel:  model.BaseModel{ID: existingID},
					TaskID:     "task-123",
					OperatorID: "operator-456",
					Action:     model.InterventionActionPause,
					Status:     model.InterventionStatusPending,
				},
			},
		}
		repo := newMockInterventionRepository(mockDB)

		intervention := &model.Intervention{
			BaseModel:  model.BaseModel{ID: existingID},
			TaskID:     "task-123",
			OperatorID: "operator-456",
			Action:     model.InterventionActionPause,
			Status:     model.InterventionStatusApplied,
		}
		err := repo.Update(ctx, intervention)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID.String())
		if updated.Status != model.InterventionStatusApplied {
			t.Errorf("Expected status 'applied', got '%s'", updated.Status)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockInterventionDB{interventions: make(map[string]*model.Intervention)}
		repo := newMockInterventionRepository(mockDB)

		intervention := &model.Intervention{BaseModel: model.BaseModel{ID: uuid.New()}, TaskID: "task-123"}
		err := repo.Update(ctx, intervention)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestInterventionFilter_Defaults(t *testing.T) {
	filter := InterventionFilter{}

	// Verify filter fields exist
	_ = filter.Page
	_ = filter.PageSize
	_ = filter.Action
	_ = filter.Status
	_ = filter.OperatorID
}

func TestInterventionFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := InterventionFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := InterventionFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := InterventionFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		filter := InterventionFilter{Page: -1, PageSize: -5}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})
}

func TestInterventionFilter_Offset(t *testing.T) {
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
		filter := InterventionFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestInterventionRepository_Interface(t *testing.T) {
	// Verify that mockInterventionRepository implements InterventionRepository interface
	var _ InterventionRepository = (*mockInterventionRepository)(nil)
}

func TestNewInterventionRepository(t *testing.T) {
	// Test that NewInterventionRepository returns a non-nil implementation
	repo := NewInterventionRepository(nil)
	if repo == nil {
		t.Error("NewInterventionRepository should return non-nil interface value")
	}
}

// Test Intervention with JSON fields
func TestIntervention_JSONFieldsInMock(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockInterventionDB{}
	repo := newMockInterventionRepository(mockDB)

	intervention := &model.Intervention{
		TaskID:     "task-123",
		OperatorID: "operator-456",
		Action:     model.InterventionActionInject,
		Status:     model.InterventionStatusPending,
		Content:    datatypes.JSON(`{"instruction": "test"}`),
		Result:     datatypes.JSON(`{"success": true}`),
	}

	err := repo.Create(ctx, intervention)
	if err != nil {
		t.Errorf("Create returned error: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, intervention.ID.String())
	if err != nil {
		t.Errorf("GetByID returned error: %v", err)
	}

	if string(retrieved.Content) != `{"instruction": "test"}` {
		t.Errorf("Expected Content '{\"instruction\": \"test\"}', got '%s'", retrieved.Content)
	}
	if string(retrieved.Result) != `{"success": true}` {
		t.Errorf("Expected Result '{\"success\": true}', got '%s'", retrieved.Result)
	}
}
