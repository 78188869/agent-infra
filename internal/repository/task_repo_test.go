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

// mockTaskDB simulates GORM DB for testing purposes
type mockTaskDB struct {
	createErr  error
	findErr    error
	listErr    error
	updateErr  error
	deleteErr  error
	tasks      map[uuid.UUID]*model.Task
	totalCount int64
}

// mockTaskRepository is a mock implementation for testing
type mockTaskRepository struct {
	db *mockTaskDB
}

func newMockTaskRepository(db *mockTaskDB) *mockTaskRepository {
	return &mockTaskRepository{db: db}
}

func (m *mockTaskRepository) Create(ctx context.Context, task *model.Task) error {
	if m.db.createErr != nil {
		return m.db.createErr
	}
	if m.db.tasks == nil {
		m.db.tasks = make(map[uuid.UUID]*model.Task)
	}
	task.ID = uuid.New()
	m.db.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	if m.db.findErr != nil {
		return nil, m.db.findErr
	}
	if m.db.tasks == nil {
		return nil, errors.NewNotFoundError("task not found")
	}
	task, ok := m.db.tasks[id]
	if !ok {
		return nil, errors.NewNotFoundError("task not found")
	}
	return task, nil
}

func (m *mockTaskRepository) List(ctx context.Context, filter TaskFilter) ([]*model.Task, int64, error) {
	if m.db.listErr != nil {
		return nil, 0, m.db.listErr
	}
	if m.db.tasks == nil {
		return []*model.Task{}, 0, nil
	}

	var result []*model.Task
	for _, t := range m.db.tasks {
		// Apply filters
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.TenantID != "" && t.TenantID != filter.TenantID {
			continue
		}
		result = append(result, t)
	}
	return result, m.db.totalCount, nil
}

func (m *mockTaskRepository) Update(ctx context.Context, task *model.Task) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.tasks == nil {
		return errors.NewNotFoundError("task not found")
	}
	if _, ok := m.db.tasks[task.ID]; !ok {
		return errors.NewNotFoundError("task not found")
	}
	m.db.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.db.deleteErr != nil {
		return m.db.deleteErr
	}
	if m.db.tasks == nil {
		return errors.NewNotFoundError("task not found")
	}
	if _, ok := m.db.tasks[id]; !ok {
		return errors.NewNotFoundError("task not found")
	}
	delete(m.db.tasks, id)
	return nil
}

func (m *mockTaskRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*model.Task, error) {
	if m.db.listErr != nil {
		return nil, m.db.listErr
	}
	if m.db.tasks == nil {
		return []*model.Task{}, nil
	}

	var result []*model.Task
	count := 0
	for _, t := range m.db.tasks {
		if t.Status == status {
			result = append(result, t)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return result, nil
}

func (m *mockTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, reason string) error {
	if m.db.updateErr != nil {
		return m.db.updateErr
	}
	if m.db.tasks == nil {
		return errors.NewNotFoundError("task not found")
	}
	task, ok := m.db.tasks[id]
	if !ok {
		return errors.NewNotFoundError("task not found")
	}
	task.Status = status
	if status == model.TaskStatusFailed && reason != "" {
		task.ErrorMessage = reason
	}
	return nil
}

// Tests

func TestTaskRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("successful create", func(t *testing.T) {
		mockDB := &mockTaskDB{}
		repo := newMockTaskRepository(mockDB)

		task := &model.Task{
			Name:       "Test Task",
			Status:     model.TaskStatusPending,
			Priority:   model.TaskPriorityNormal,
			TenantID:   "tenant-123",
			CreatorID:  "creator-123",
			ProviderID: "provider-123",
		}

		err := repo.Create(ctx, task)
		if err != nil {
			t.Errorf("Create returned error: %v", err)
		}
		if task.ID == uuid.Nil {
			t.Error("Task ID should be set after creation")
		}
	})

	t.Run("create error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockTaskDB{createErr: expectedErr}
		repo := newMockTaskRepository(mockDB)

		task := &model.Task{Name: "Test Task"}
		err := repo.Create(ctx, task)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTaskRepository_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				existingID: {
					BaseModel:  model.BaseModel{ID: existingID},
					Name:       "Existing Task",
					Status:     model.TaskStatusPending,
					TenantID:   "tenant-123",
					ProviderID: "provider-123",
				},
			},
		}
		repo := newMockTaskRepository(mockDB)

		task, err := repo.GetByID(ctx, existingID)
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if task == nil {
			t.Error("Expected task, got nil")
		}
		if task.Name != "Existing Task" {
			t.Errorf("Expected name 'Existing Task', got '%s'", task.Name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		mockDB := &mockTaskDB{tasks: make(map[uuid.UUID]*model.Task)}
		repo := newMockTaskRepository(mockDB)

		_, err := repo.GetByID(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTaskRepository_List(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Task 1", Status: model.TaskStatusPending},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Task 2", Status: model.TaskStatusRunning},
			},
			totalCount: 2,
		}
		repo := newMockTaskRepository(mockDB)

		filter := TaskFilter{Page: 1, PageSize: 10}
		tasks, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("Expected 2 tasks, got %d", len(tasks))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})

	t.Run("list with status filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Task 1", Status: model.TaskStatusPending},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Task 2", Status: model.TaskStatusRunning},
			},
			totalCount: 1,
		}
		repo := newMockTaskRepository(mockDB)

		filter := TaskFilter{Page: 1, PageSize: 10, Status: model.TaskStatusPending}
		tasks, _, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task with pending status, got %d", len(tasks))
		}
	})

	t.Run("list with tenant filter", func(t *testing.T) {
		id1, id2 := uuid.New(), uuid.New()
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Task 1", TenantID: "tenant-1"},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Task 2", TenantID: "tenant-2"},
			},
			totalCount: 1,
		}
		repo := newMockTaskRepository(mockDB)

		filter := TaskFilter{Page: 1, PageSize: 10, TenantID: "tenant-1"}
		tasks, _, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tasks) != 1 {
			t.Errorf("Expected 1 task with tenant-1, got %d", len(tasks))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mockDB := &mockTaskDB{tasks: nil}
		repo := newMockTaskRepository(mockDB)

		filter := TaskFilter{Page: 1, PageSize: 10}
		tasks, total, err := repo.List(ctx, filter)
		if err != nil {
			t.Errorf("List returned error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("Expected 0 tasks, got %d", len(tasks))
		}
		if total != 0 {
			t.Errorf("Expected total 0, got %d", total)
		}
	})

	t.Run("list error", func(t *testing.T) {
		expectedErr := errors.NewInternalError("database error")
		mockDB := &mockTaskDB{listErr: expectedErr}
		repo := newMockTaskRepository(mockDB)

		filter := TaskFilter{Page: 1, PageSize: 10}
		_, _, err := repo.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTaskRepository_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful update", func(t *testing.T) {
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Old Name", Status: model.TaskStatusPending},
			},
		}
		repo := newMockTaskRepository(mockDB)

		task := &model.Task{BaseModel: model.BaseModel{ID: existingID}, Name: "New Name", Status: model.TaskStatusRunning}
		err := repo.Update(ctx, task)
		if err != nil {
			t.Errorf("Update returned error: %v", err)
		}

		updated, _ := repo.GetByID(ctx, existingID)
		if updated.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got '%s'", updated.Name)
		}
	})

	t.Run("update non-existent", func(t *testing.T) {
		mockDB := &mockTaskDB{tasks: make(map[uuid.UUID]*model.Task)}
		repo := newMockTaskRepository(mockDB)

		task := &model.Task{BaseModel: model.BaseModel{ID: uuid.New()}, Name: "New Name"}
		err := repo.Update(ctx, task)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTaskRepository_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Task to Delete"},
			},
		}
		repo := newMockTaskRepository(mockDB)

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
		mockDB := &mockTaskDB{tasks: make(map[uuid.UUID]*model.Task)}
		repo := newMockTaskRepository(mockDB)

		err := repo.Delete(ctx, uuid.New())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTaskRepository_ListByStatus(t *testing.T) {
	ctx := context.Background()

	t.Run("successful list by status", func(t *testing.T) {
		id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Task 1", Status: model.TaskStatusPending},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Task 2", Status: model.TaskStatusPending},
				id3: {BaseModel: model.BaseModel{ID: id3}, Name: "Task 3", Status: model.TaskStatusRunning},
			},
		}
		repo := newMockTaskRepository(mockDB)

		tasks, err := repo.ListByStatus(ctx, model.TaskStatusPending, 0)
		if err != nil {
			t.Errorf("ListByStatus returned error: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("Expected 2 pending tasks, got %d", len(tasks))
		}
	})

	t.Run("list by status with limit", func(t *testing.T) {
		id1, id2, id3 := uuid.New(), uuid.New(), uuid.New()
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				id1: {BaseModel: model.BaseModel{ID: id1}, Name: "Task 1", Status: model.TaskStatusPending},
				id2: {BaseModel: model.BaseModel{ID: id2}, Name: "Task 2", Status: model.TaskStatusPending},
				id3: {BaseModel: model.BaseModel{ID: id3}, Name: "Task 3", Status: model.TaskStatusPending},
			},
		}
		repo := newMockTaskRepository(mockDB)

		tasks, err := repo.ListByStatus(ctx, model.TaskStatusPending, 2)
		if err != nil {
			t.Errorf("ListByStatus returned error: %v", err)
		}
		if len(tasks) > 2 {
			t.Errorf("Expected at most 2 tasks, got %d", len(tasks))
		}
	})

	t.Run("empty list by status", func(t *testing.T) {
		mockDB := &mockTaskDB{tasks: make(map[uuid.UUID]*model.Task)}
		repo := newMockTaskRepository(mockDB)

		tasks, err := repo.ListByStatus(ctx, model.TaskStatusPending, 0)
		if err != nil {
			t.Errorf("ListByStatus returned error: %v", err)
		}
		if len(tasks) != 0 {
			t.Errorf("Expected 0 tasks, got %d", len(tasks))
		}
	})
}

func TestTaskRepository_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful status update", func(t *testing.T) {
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Task", Status: model.TaskStatusPending},
			},
		}
		repo := newMockTaskRepository(mockDB)

		err := repo.UpdateStatus(ctx, existingID, model.TaskStatusRunning, "")
		if err != nil {
			t.Errorf("UpdateStatus returned error: %v", err)
		}

		task, _ := repo.GetByID(ctx, existingID)
		if task.Status != model.TaskStatusRunning {
			t.Errorf("Expected status 'running', got '%s'", task.Status)
		}
	})

	t.Run("status update with error message for failed", func(t *testing.T) {
		mockDB := &mockTaskDB{
			tasks: map[uuid.UUID]*model.Task{
				existingID: {BaseModel: model.BaseModel{ID: existingID}, Name: "Task", Status: model.TaskStatusRunning},
			},
		}
		repo := newMockTaskRepository(mockDB)

		err := repo.UpdateStatus(ctx, existingID, model.TaskStatusFailed, "Something went wrong")
		if err != nil {
			t.Errorf("UpdateStatus returned error: %v", err)
		}

		task, _ := repo.GetByID(ctx, existingID)
		if task.Status != model.TaskStatusFailed {
			t.Errorf("Expected status 'failed', got '%s'", task.Status)
		}
		if task.ErrorMessage != "Something went wrong" {
			t.Errorf("Expected error message 'Something went wrong', got '%s'", task.ErrorMessage)
		}
	})

	t.Run("update status non-existent", func(t *testing.T) {
		mockDB := &mockTaskDB{tasks: make(map[uuid.UUID]*model.Task)}
		repo := newMockTaskRepository(mockDB)

		err := repo.UpdateStatus(ctx, uuid.New(), model.TaskStatusRunning, "")
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTaskFilter_Defaults(t *testing.T) {
	filter := TaskFilter{}

	// Verify filter fields exist
	_ = filter.Page
	_ = filter.PageSize
	_ = filter.Status
	_ = filter.TenantID
	_ = filter.Search
}

func TestTaskFilter_SetDefaults(t *testing.T) {
	t.Run("sets default page", func(t *testing.T) {
		filter := TaskFilter{Page: 0, PageSize: 0}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})

	t.Run("respects valid values", func(t *testing.T) {
		filter := TaskFilter{Page: 5, PageSize: 20}
		filter.SetDefaults()
		if filter.Page != 5 {
			t.Errorf("Expected Page=5, got %d", filter.Page)
		}
		if filter.PageSize != 20 {
			t.Errorf("Expected PageSize=20, got %d", filter.PageSize)
		}
	})

	t.Run("caps page size at 100", func(t *testing.T) {
		filter := TaskFilter{Page: 1, PageSize: 200}
		filter.SetDefaults()
		if filter.PageSize != 100 {
			t.Errorf("Expected PageSize capped at 100, got %d", filter.PageSize)
		}
	})

	t.Run("negative values get defaults", func(t *testing.T) {
		filter := TaskFilter{Page: -1, PageSize: -5}
		filter.SetDefaults()
		if filter.Page != 1 {
			t.Errorf("Expected Page=1, got %d", filter.Page)
		}
		if filter.PageSize != 10 {
			t.Errorf("Expected PageSize=10, got %d", filter.PageSize)
		}
	})
}

func TestTaskFilter_Offset(t *testing.T) {
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
		filter := TaskFilter{Page: tt.page, PageSize: tt.pageSize}
		if got := filter.Offset(); got != tt.expected {
			t.Errorf("Offset(Page=%d, PageSize=%d) = %d, want %d",
				tt.page, tt.pageSize, got, tt.expected)
		}
	}
}

func TestTaskRepository_Interface(t *testing.T) {
	// Verify that mockTaskRepository implements TaskRepository interface
	var _ TaskRepository = (*mockTaskRepository)(nil)
}

func TestNewTaskRepository(t *testing.T) {
	// Test that NewTaskRepository returns a non-nil implementation
	repo := NewTaskRepository(nil)
	if repo == nil {
		t.Error("NewTaskRepository should return non-nil interface value")
	}
}

// Test Task with JSON fields
func TestTask_JSONFieldsInMock(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTaskDB{}
	repo := newMockTaskRepository(mockDB)

	task := &model.Task{
		Name:       "Task with JSON",
		Status:     model.TaskStatusPending,
		TenantID:   "tenant-123",
		ProviderID: "provider-123",
		Params:     datatypes.JSON(`{"input": "test"}`),
		Result:     datatypes.JSON(`{"output": "success"}`),
	}

	err := repo.Create(ctx, task)
	if err != nil {
		t.Errorf("Create returned error: %v", err)
	}

	retrieved, err := repo.GetByID(ctx, task.ID)
	if err != nil {
		t.Errorf("GetByID returned error: %v", err)
	}

	if string(retrieved.Params) != `{"input": "test"}` {
		t.Errorf("Expected Params '{\"input\": \"test\"}', got '%s'", retrieved.Params)
	}
	if string(retrieved.Result) != `{"output": "success"}` {
		t.Errorf("Expected Result '{\"output\": \"success\"}', got '%s'", retrieved.Result)
	}
}
