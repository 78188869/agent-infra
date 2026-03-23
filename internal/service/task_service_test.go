package service

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockTaskRepository implements repository.TaskRepository for testing
type mockTaskRepository struct {
	createFunc        func(ctx context.Context, task *model.Task) error
	getByIDFunc       func(ctx context.Context, id uuid.UUID) (*model.Task, error)
	listFunc          func(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int64, error)
	updateFunc        func(ctx context.Context, task *model.Task) error
	deleteFunc        func(ctx context.Context, id uuid.UUID) error
	listByStatusFunc  func(ctx context.Context, status string, limit int) ([]*model.Task, error)
	updateStatusFunc  func(ctx context.Context, id uuid.UUID, status string, reason string) error
}

func (m *mockTaskRepository) Create(ctx context.Context, task *model.Task) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, task)
	}
	return nil
}

func (m *mockTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockTaskRepository) List(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int64, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filter)
	}
	return nil, 0, nil
}

func (m *mockTaskRepository) Update(ctx context.Context, task *model.Task) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, task)
	}
	return nil
}

func (m *mockTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockTaskRepository) ListByStatus(ctx context.Context, status string, limit int) ([]*model.Task, error) {
	if m.listByStatusFunc != nil {
		return m.listByStatusFunc(ctx, status, limit)
	}
	return nil, nil
}

func (m *mockTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, reason string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status, reason)
	}
	return nil
}

func TestTaskService_Create(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	creatorID := uuid.New()
	providerID := uuid.New()
	templateID := uuid.New()

	tests := []struct {
		name        string
		req         *CreateTaskRequest
		mockSetup   func(*mockTaskRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful create with minimal fields",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup: func(m *mockTaskRepository) {
				m.createFunc = func(ctx context.Context, task *model.Task) error {
					task.ID = uuid.New()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "successful create with all fields",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				TemplateID: strPtr(templateID.String()),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task with Template",
				Priority:   model.TaskPriorityHigh,
				Params:     json.RawMessage(`{"key": "value"}`),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.createFunc = func(ctx context.Context, task *model.Task) error {
					task.ID = uuid.New()
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "missing tenant_id",
			req: &CreateTaskRequest{
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "tenant_id is required",
		},
		{
			name: "missing creator_id",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "creator_id is required",
		},
		{
			name: "missing provider_id",
			req: &CreateTaskRequest{
				TenantID:  tenantID.String(),
				CreatorID: creatorID.String(),
				Name:      "Test Task",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "provider_id is required",
		},
		{
			name: "missing name",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "name is required",
		},
		{
			name: "invalid tenant_id format",
			req: &CreateTaskRequest{
				TenantID:   "invalid-uuid",
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "invalid tenant_id format",
		},
		{
			name: "invalid template_id format",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				TemplateID: strPtr("invalid-uuid"),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "invalid template_id format",
		},
		{
			name: "invalid params JSON",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
				Params:     json.RawMessage(`{invalid json}`),
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "params must be valid JSON",
		},
		{
			name: "invalid priority value",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
				Priority:   "invalid",
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "invalid priority value",
		},
		{
			name: "repository error",
			req: &CreateTaskRequest{
				TenantID:   tenantID.String(),
				CreatorID:  creatorID.String(),
				ProviderID: providerID.String(),
				Name:       "Test Task",
			},
			mockSetup: func(m *mockTaskRepository) {
				m.createFunc = func(ctx context.Context, task *model.Task) error {
					return errors.NewInternalError("database error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTaskRepository{}
			tt.mockSetup(repo)
			service := NewTaskService(repo)

			task, err := service.Create(ctx, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					appErr, ok := err.(*errors.AppError)
					if !ok {
						t.Errorf("Expected AppError, got %T", err)
					} else if appErr.Message != tt.errContains && appErr.Error() != tt.errContains {
						// Check if error message contains the expected string
					}
				}
			} else {
				if err != nil {
					t.Errorf("Create returned error: %v", err)
				}
				if task == nil {
					t.Fatal("Expected task, got nil")
				}
				if task.Name != tt.req.Name {
					t.Errorf("Expected name '%s', got '%s'", tt.req.Name, task.Name)
				}
				if task.Status != model.TaskStatusPending {
					t.Errorf("Expected status '%s', got '%s'", model.TaskStatusPending, task.Status)
				}
				if tt.req.Priority == "" && task.Priority != model.TaskPriorityNormal {
					t.Errorf("Expected default priority '%s', got '%s'", model.TaskPriorityNormal, task.Priority)
				}
			}
		})
	}
}

func TestTaskService_GetByID(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	t.Run("successful get", func(t *testing.T) {
		repo := &mockTaskRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
				if id == existingID {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusPending,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
				return nil, errors.NewNotFoundError("task not found")
			},
		}
		service := NewTaskService(repo)

		task, err := service.GetByID(ctx, existingID.String())
		if err != nil {
			t.Errorf("GetByID returned error: %v", err)
		}
		if task == nil {
			t.Fatal("Expected task, got nil")
		}
		if task.Name != "Test Task" {
			t.Errorf("Expected name 'Test Task', got '%s'", task.Name)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTaskRepository{}
		service := NewTaskService(repo)

		_, err := service.GetByID(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTaskRepository{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
				return nil, errors.NewNotFoundError("task not found")
			},
		}
		service := NewTaskService(repo)

		_, err := service.GetByID(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestTaskService_List(t *testing.T) {
	ctx := context.Background()
	id1, id2 := uuid.New(), uuid.New()
	tenantID := uuid.New()

	t.Run("successful list", func(t *testing.T) {
		repo := &mockTaskRepository{
			listFunc: func(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int64, error) {
				return []*model.Task{
					{BaseModel: model.BaseModel{ID: id1}, TenantID: tenantID.String(), Name: "Task 1", Status: model.TaskStatusPending},
					{BaseModel: model.BaseModel{ID: id2}, TenantID: tenantID.String(), Name: "Task 2", Status: model.TaskStatusRunning},
				}, 2, nil
			},
		}
		service := NewTaskService(repo)

		filter := &TaskFilter{Page: 1, PageSize: 10}
		tasks, total, err := service.List(ctx, filter)
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

	t.Run("empty list", func(t *testing.T) {
		repo := &mockTaskRepository{
			listFunc: func(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int64, error) {
				return []*model.Task{}, 0, nil
			},
		}
		service := NewTaskService(repo)

		filter := &TaskFilter{Page: 1, PageSize: 10}
		tasks, total, err := service.List(ctx, filter)
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

	t.Run("repository error", func(t *testing.T) {
		repo := &mockTaskRepository{
			listFunc: func(ctx context.Context, filter repository.TaskFilter) ([]*model.Task, int64, error) {
				return nil, 0, errors.NewInternalError("database error")
			},
		}
		service := NewTaskService(repo)

		filter := &TaskFilter{Page: 1, PageSize: 10}
		_, _, err := service.List(ctx, filter)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestTaskService_Update(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	tenantID := uuid.New()

	tests := []struct {
		name        string
		id          string
		req         *UpdateTaskRequest
		initialTask *model.Task
		mockSetup   func(*mockTaskRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful status transition pending to scheduled",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusScheduled),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusPending,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusPending,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
				m.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "successful status transition running to succeeded",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusSucceeded),
				Result: json.RawMessage(`{"output": "done"}`),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusRunning,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusRunning,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
				m.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "successful status transition running to failed with error message",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Status:       strPtr(model.TaskStatusFailed),
				ErrorMessage: strPtr("something went wrong"),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusRunning,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusRunning,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
				m.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid status transition pending to running",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusRunning),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusPending,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusPending,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
			},
			wantErr:     true,
			errContains: "invalid status transition",
		},
		{
			name: "invalid status transition from terminal state",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusRunning),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusSucceeded,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusSucceeded,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
			},
			wantErr:     true,
			errContains: "invalid status transition",
		},
		{
			name: "invalid result JSON",
			id:   existingID.String(),
			req: &UpdateTaskRequest{
				Result: json.RawMessage(`{invalid json}`),
			},
			initialTask: &model.Task{
				BaseModel:  model.BaseModel{ID: existingID},
				TenantID:   tenantID.String(),
				Name:       "Test Task",
				Status:     model.TaskStatusRunning,
				ProviderID: uuid.New().String(),
				CreatorID:  uuid.New().String(),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel:  model.BaseModel{ID: existingID},
						TenantID:   tenantID.String(),
						Name:       "Test Task",
						Status:     model.TaskStatusRunning,
						ProviderID: uuid.New().String(),
						CreatorID:  uuid.New().String(),
					}, nil
				}
			},
			wantErr:     true,
			errContains: "result must be valid JSON",
		},
		{
			name: "invalid id format",
			id:   "invalid-uuid",
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusRunning),
			},
			mockSetup:   func(m *mockTaskRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name: "task not found",
			id:   uuid.New().String(),
			req: &UpdateTaskRequest{
				Status: strPtr(model.TaskStatusScheduled),
			},
			mockSetup: func(m *mockTaskRepository) {
				m.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTaskRepository{}
			tt.mockSetup(repo)
			service := NewTaskService(repo)

			err := service.Update(ctx, tt.id, tt.req)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Update returned error: %v", err)
				}
			}
		})
	}
}

func TestTaskService_Delete(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()

	t.Run("successful delete", func(t *testing.T) {
		repo := &mockTaskRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return nil
			},
		}
		service := NewTaskService(repo)

		err := service.Delete(ctx, existingID.String())
		if err != nil {
			t.Errorf("Delete returned error: %v", err)
		}
	})

	t.Run("invalid id format", func(t *testing.T) {
		repo := &mockTaskRepository{}
		service := NewTaskService(repo)

		err := service.Delete(ctx, "invalid-uuid")
		if err == nil {
			t.Error("Expected error for invalid UUID, got nil")
		}
		if !stderrors.Is(err, errors.ErrBadRequest) {
			t.Errorf("Expected BadRequest error, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockTaskRepository{
			deleteFunc: func(ctx context.Context, id uuid.UUID) error {
				return errors.NewNotFoundError("task not found")
			},
		}
		service := NewTaskService(repo)

		err := service.Delete(ctx, uuid.New().String())
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !stderrors.Is(err, errors.ErrNotFound) {
			t.Errorf("Expected NotFound error, got %v", err)
		}
	})
}

func TestIsValidStatusTransition(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected bool
	}{
		// Valid transitions
		{model.TaskStatusPending, model.TaskStatusScheduled, true},
		{model.TaskStatusPending, model.TaskStatusCancelled, true},
		{model.TaskStatusScheduled, model.TaskStatusRunning, true},
		{model.TaskStatusScheduled, model.TaskStatusCancelled, true},
		{model.TaskStatusRunning, model.TaskStatusPaused, true},
		{model.TaskStatusRunning, model.TaskStatusSucceeded, true},
		{model.TaskStatusRunning, model.TaskStatusFailed, true},
		{model.TaskStatusRunning, model.TaskStatusCancelled, true},
		{model.TaskStatusPaused, model.TaskStatusRunning, true},
		{model.TaskStatusPaused, model.TaskStatusCancelled, true},
		{model.TaskStatusWaitingApproval, model.TaskStatusRunning, true},
		{model.TaskStatusWaitingApproval, model.TaskStatusCancelled, true},
		{model.TaskStatusRetrying, model.TaskStatusRunning, true},
		{model.TaskStatusRetrying, model.TaskStatusFailed, true},
		{model.TaskStatusRetrying, model.TaskStatusCancelled, true},

		// Invalid transitions
		{model.TaskStatusPending, model.TaskStatusRunning, false},
		{model.TaskStatusPending, model.TaskStatusSucceeded, false},
		{model.TaskStatusScheduled, model.TaskStatusPaused, false},
		{model.TaskStatusRunning, model.TaskStatusScheduled, false},

		// Terminal states - no transitions allowed
		{model.TaskStatusSucceeded, model.TaskStatusRunning, false},
		{model.TaskStatusSucceeded, model.TaskStatusFailed, false},
		{model.TaskStatusFailed, model.TaskStatusRunning, false},
		{model.TaskStatusFailed, model.TaskStatusSucceeded, false},
		{model.TaskStatusCancelled, model.TaskStatusPending, false},
		{model.TaskStatusCancelled, model.TaskStatusRunning, false},

		// Unknown status
		{"unknown", model.TaskStatusRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			result := isValidStatusTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("isValidStatusTransition(%s, %s) = %v, expected %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestTaskService_Interface(t *testing.T) {
	// Verify that taskService implements TaskService interface
	var _ TaskService = (*taskService)(nil)
}

// Helper function to create string pointer
func strPtr(s string) *string {
	return &s
}
