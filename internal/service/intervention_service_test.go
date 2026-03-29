package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/example/agent-infra/internal/model"
	"github.com/example/agent-infra/internal/repository"
	"github.com/example/agent-infra/pkg/errors"
	"github.com/google/uuid"
)

// mockInterventionRepository implements repository.InterventionRepository for testing
type mockInterventionRepository struct {
	createFunc    func(ctx context.Context, intervention *model.Intervention) error
	getByIDFunc   func(ctx context.Context, id string) (*model.Intervention, error)
	listByTaskFunc func(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error)
	updateFunc    func(ctx context.Context, intervention *model.Intervention) error
}

func (m *mockInterventionRepository) Create(ctx context.Context, intervention *model.Intervention) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, intervention)
	}
	return nil
}

func (m *mockInterventionRepository) GetByID(ctx context.Context, id string) (*model.Intervention, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockInterventionRepository) ListByTask(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error) {
	if m.listByTaskFunc != nil {
		return m.listByTaskFunc(ctx, taskID, filter)
	}
	return nil, 0, nil
}

func (m *mockInterventionRepository) Update(ctx context.Context, intervention *model.Intervention) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, intervention)
	}
	return nil
}

func TestInterventionService_Pause(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		taskID      string
		operatorID  string
		reason      string
		mockSetup   func(*mockTaskRepository, *mockInterventionRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful pause with running task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Need to review progress",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusRunning,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "invalid task ID format",
			taskID:      "invalid-uuid",
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name:        "invalid operator ID format",
			taskID:      taskID.String(),
			operatorID:  "invalid-uuid",
			reason:      "Test",
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid operator ID format",
		},
		{
			name:       "task not found",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			wantErr:     true,
			errContains: "task not found",
		},
		{
			name:       "cannot pause pending task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPending,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
		{
			name:       "cannot pause scheduled task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusScheduled,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
		{
			name:       "cannot pause paused task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPaused,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
		{
			name:       "cannot pause succeeded task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusSucceeded,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
		{
			name:       "cannot pause failed task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusFailed,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
		{
			name:       "cannot pause cancelled task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusCancelled,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot pause task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(taskRepo, intRepo)

			service := NewInterventionService(taskRepo, intRepo)
			intervention, err := service.Pause(ctx, tt.taskID, tt.operatorID, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Pause() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Pause() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Pause() unexpected error = %v", err)
				return
			}

			if intervention == nil {
				t.Error("Pause() returned nil intervention")
				return
			}

			if intervention.Action != model.InterventionActionPause {
				t.Errorf("Pause() intervention action = %v, want %v", intervention.Action, model.InterventionActionPause)
			}

			if intervention.Status != model.InterventionStatusApplied {
				t.Errorf("Pause() intervention status = %v, want %v", intervention.Status, model.InterventionStatusApplied)
			}
		})
	}
}

func TestInterventionService_Resume(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		taskID      string
		operatorID  string
		reason      string
		mockSetup   func(*mockTaskRepository, *mockInterventionRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful resume with paused task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Ready to continue",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPaused,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "invalid task ID format",
			taskID:      "invalid-uuid",
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name:       "cannot resume running task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusRunning,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot resume task",
		},
		{
			name:       "cannot resume pending task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPending,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot resume task",
		},
		{
			name:       "cannot resume succeeded task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusSucceeded,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot resume task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(taskRepo, intRepo)

			service := NewInterventionService(taskRepo, intRepo)
			intervention, err := service.Resume(ctx, tt.taskID, tt.operatorID, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Resume() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Resume() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Resume() unexpected error = %v", err)
				return
			}

			if intervention == nil {
				t.Error("Resume() returned nil intervention")
				return
			}

			if intervention.Action != model.InterventionActionResume {
				t.Errorf("Resume() intervention action = %v, want %v", intervention.Action, model.InterventionActionResume)
			}

			if intervention.Status != model.InterventionStatusApplied {
				t.Errorf("Resume() intervention status = %v, want %v", intervention.Status, model.InterventionStatusApplied)
			}
		})
	}
}

func TestInterventionService_Cancel(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		taskID      string
		operatorID  string
		reason      string
		mockSetup   func(*mockTaskRepository, *mockInterventionRepository)
		wantErr     bool
		errContains string
	}{
		{
			name:       "successful cancel with pending task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "No longer needed",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPending,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "successful cancel with scheduled task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Schedule changed",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusScheduled,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "successful cancel with running task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Stopping execution",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusRunning,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "successful cancel with paused task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Task no longer needed",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPaused,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "successful cancel with waiting_approval task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Approval rejected",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusWaitingApproval,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:       "successful cancel with retrying task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Stop retrying",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusRetrying,
					}, nil
				}
				taskRepo.updateFunc = func(ctx context.Context, task *model.Task) error {
					return nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "invalid task ID format",
			taskID:      "invalid-uuid",
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name:       "cannot cancel succeeded task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusSucceeded,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot cancel task",
		},
		{
			name:       "cannot cancel failed task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusFailed,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot cancel task",
		},
		{
			name:       "cannot cancel already cancelled task",
			taskID:     taskID.String(),
			operatorID: operatorID.String(),
			reason:     "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusCancelled,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot cancel task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(taskRepo, intRepo)

			service := NewInterventionService(taskRepo, intRepo)
			intervention, err := service.Cancel(ctx, tt.taskID, tt.operatorID, tt.reason)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Cancel() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Cancel() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Cancel() unexpected error = %v", err)
				return
			}

			if intervention == nil {
				t.Error("Cancel() returned nil intervention")
				return
			}

			if intervention.Action != model.InterventionActionCancel {
				t.Errorf("Cancel() intervention action = %v, want %v", intervention.Action, model.InterventionActionCancel)
			}

			if intervention.Status != model.InterventionStatusApplied {
				t.Errorf("Cancel() intervention status = %v, want %v", intervention.Status, model.InterventionStatusApplied)
			}
		})
	}
}

func TestInterventionService_Inject(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		req         *InjectInterventionRequest
		mockSetup   func(*mockTaskRepository, *mockInterventionRepository)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful inject with running task",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "Please review the output before proceeding",
				Context:     "Quality check required",
			},
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusRunning,
					}, nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "successful inject with waiting_approval task",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "Approve the deployment",
				Context:     "Final approval needed",
			},
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusWaitingApproval,
					}, nil
				}
				intRepo.createFunc = func(ctx context.Context, intervention *model.Intervention) error {
					intervention.ID = uuid.New()
					return nil
				}
				intRepo.updateFunc = func(ctx context.Context, intervention *model.Intervention) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name: "invalid task ID format",
			req: &InjectInterventionRequest{
				TaskID:      "invalid-uuid",
				OperatorID:  operatorID.String(),
				Instruction: "Test",
			},
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name: "invalid operator ID format",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  "invalid-uuid",
				Instruction: "Test",
			},
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid operator ID format",
		},
		{
			name: "empty instruction",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "",
			},
			mockSetup:   func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "instruction is required",
		},
		{
			name: "cannot inject into pending task",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "Test",
			},
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPending,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot inject into task",
		},
		{
			name: "cannot inject into paused task",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "Test",
			},
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusPaused,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot inject into task",
		},
		{
			name: "cannot inject into succeeded task",
			req: &InjectInterventionRequest{
				TaskID:      taskID.String(),
				OperatorID:  operatorID.String(),
				Instruction: "Test",
			},
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return &model.Task{
						BaseModel: model.BaseModel{ID: taskID},
						Status:    model.TaskStatusSucceeded,
					}, nil
				}
			},
			wantErr:     true,
			errContains: "cannot inject into task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(taskRepo, intRepo)

			service := NewInterventionService(taskRepo, intRepo)
			intervention, err := service.Inject(ctx, tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Inject() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("Inject() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Inject() unexpected error = %v", err)
				return
			}

			if intervention == nil {
				t.Error("Inject() returned nil intervention")
				return
			}

			if intervention.Action != model.InterventionActionInject {
				t.Errorf("Inject() intervention action = %v, want %v", intervention.Action, model.InterventionActionInject)
			}

			if intervention.Status != model.InterventionStatusApplied {
				t.Errorf("Inject() intervention status = %v, want %v", intervention.Status, model.InterventionStatusApplied)
			}

			// Verify content
			if len(intervention.Content) > 0 {
				var content model.InterventionContent
				if err := json.Unmarshal(intervention.Content, &content); err != nil {
					t.Errorf("Inject() failed to unmarshal content: %v", err)
				}
				if content.Instruction != tt.req.Instruction {
					t.Errorf("Inject() content instruction = %v, want %v", content.Instruction, tt.req.Instruction)
				}
			}
		})
	}
}

func TestInterventionService_ListInterventions(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		taskID      string
		filter      *InterventionFilter
		mockSetup   func(*mockInterventionRepository)
		wantErr     bool
		errContains string
		wantCount   int
		wantTotal   int64
	}{
		{
			name:   "successful list with pagination",
			taskID: taskID.String(),
			filter: &InterventionFilter{
				Page:     1,
				PageSize: 10,
			},
			mockSetup: func(intRepo *mockInterventionRepository) {
				intRepo.listByTaskFunc = func(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{
						{
							BaseModel:  model.BaseModel{ID: uuid.New()},
							TaskID:     taskID,
							OperatorID: operatorID.String(),
							Action:     model.InterventionActionPause,
							Status:     model.InterventionStatusApplied,
						},
						{
							BaseModel:  model.BaseModel{ID: uuid.New()},
							TaskID:     taskID,
							OperatorID: operatorID.String(),
							Action:     model.InterventionActionResume,
							Status:     model.InterventionStatusApplied,
						},
					}, 2, nil
				}
			},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name:   "successful list with action filter",
			taskID: taskID.String(),
			filter: &InterventionFilter{
				Page:   1,
				Action: "pause",
			},
			mockSetup: func(intRepo *mockInterventionRepository) {
				intRepo.listByTaskFunc = func(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{
						{
							BaseModel:  model.BaseModel{ID: uuid.New()},
							TaskID:     taskID,
							OperatorID: operatorID.String(),
							Action:     model.InterventionActionPause,
							Status:     model.InterventionStatusApplied,
						},
					}, 1, nil
				}
			},
			wantErr:   false,
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:   "successful list with status filter",
			taskID: taskID.String(),
			filter: &InterventionFilter{
				Page:   1,
				Status: "applied",
			},
			mockSetup: func(intRepo *mockInterventionRepository) {
				intRepo.listByTaskFunc = func(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{
						{
							BaseModel:  model.BaseModel{ID: uuid.New()},
							TaskID:     taskID,
							OperatorID: operatorID.String(),
							Action:     model.InterventionActionPause,
							Status:     model.InterventionStatusApplied,
						},
					}, 1, nil
				}
			},
			wantErr:   false,
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name:   "empty list",
			taskID: taskID.String(),
			filter: &InterventionFilter{
				Page:     1,
				PageSize: 10,
			},
			mockSetup: func(intRepo *mockInterventionRepository) {
				intRepo.listByTaskFunc = func(ctx context.Context, taskID string, filter repository.InterventionFilter) ([]*model.Intervention, int64, error) {
					return []*model.Intervention{}, 0, nil
				}
			},
			wantErr:   false,
			wantCount: 0,
			wantTotal: 0,
		},
		{
			name:   "invalid task ID format",
			taskID: "invalid-uuid",
			filter: &InterventionFilter{
				Page:     1,
				PageSize: 10,
			},
			mockSetup:   func(intRepo *mockInterventionRepository) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(intRepo)

			service := NewInterventionService(taskRepo, intRepo)
			interventions, total, err := service.ListInterventions(ctx, tt.taskID, tt.filter)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListInterventions() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("ListInterventions() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ListInterventions() unexpected error = %v", err)
				return
			}

			if len(interventions) != tt.wantCount {
				t.Errorf("ListInterventions() returned %d interventions, want %d", len(interventions), tt.wantCount)
			}

			if total != tt.wantTotal {
				t.Errorf("ListInterventions() returned total %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestInterventionService_NonExistentTask(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()
	operatorID := uuid.New()

	tests := []struct {
		name        string
		action      string
		taskID      string
		operatorID  string
		reason      string
		mockSetup   func(*mockTaskRepository, *mockInterventionRepository)
		errContains string
	}{
		{
			name:        "pause non-existent task",
			action:      "pause",
			taskID:      taskID.String(),
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			errContains: "task not found",
		},
		{
			name:        "resume non-existent task",
			action:      "resume",
			taskID:      taskID.String(),
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			errContains: "task not found",
		},
		{
			name:        "cancel non-existent task",
			action:      "cancel",
			taskID:      taskID.String(),
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			errContains: "task not found",
		},
		{
			name:        "inject into non-existent task",
			action:      "inject",
			taskID:      taskID.String(),
			operatorID:  operatorID.String(),
			reason:      "Test",
			mockSetup: func(taskRepo *mockTaskRepository, intRepo *mockInterventionRepository) {
				taskRepo.getByIDFunc = func(ctx context.Context, id uuid.UUID) (*model.Task, error) {
					return nil, errors.NewNotFoundError("task not found")
				}
			},
			errContains: "task not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			tt.mockSetup(taskRepo, intRepo)

			service := NewInterventionService(taskRepo, intRepo)

			var err error
			switch tt.action {
			case "pause":
				_, err = service.Pause(ctx, tt.taskID, tt.operatorID, tt.reason)
			case "resume":
				_, err = service.Resume(ctx, tt.taskID, tt.operatorID, tt.reason)
			case "cancel":
				_, err = service.Cancel(ctx, tt.taskID, tt.operatorID, tt.reason)
			case "inject":
				_, err = service.Inject(ctx, &InjectInterventionRequest{
					TaskID:      tt.taskID,
					OperatorID:  tt.operatorID,
					Instruction: "Test instruction",
				})
			}

			if err == nil {
				t.Errorf("%s() expected error, got nil", tt.action)
				return
			}

			if !contains(err.Error(), tt.errContains) {
				t.Errorf("%s() error = %v, want error containing %v", tt.action, err, tt.errContains)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mockTaskEventHandler implements TaskEventHandler for testing
type mockTaskEventHandler struct {
	handleTaskEventFunc func(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error
}

func (m *mockTaskEventHandler) HandleTaskEvent(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
	if m.handleTaskEventFunc != nil {
		return m.handleTaskEventFunc(ctx, taskID, eventType, payload)
	}
	return nil
}

func TestInterventionService_HandleWrapperEvent(t *testing.T) {
	ctx := context.Background()
	taskID := uuid.New()

	tests := []struct {
		name        string
		taskID      string
		eventType   string
		payload     map[string]interface{}
		mockSetup   func(*mockTaskEventHandler)
		wantErr     bool
		errContains string
	}{
		{
			name:      "successful heartbeat event",
			taskID:    taskID.String(),
			eventType: "heartbeat",
			payload: map[string]interface{}{
				"status":   "running",
				"progress": float64(50),
			},
			mockSetup: func(handler *mockTaskEventHandler) {
				handler.handleTaskEventFunc = func(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:      "successful status_change event",
			taskID:    taskID.String(),
			eventType: "status_change",
			payload: map[string]interface{}{
				"status":  "running",
				"message": "task started",
			},
			mockSetup: func(handler *mockTaskEventHandler) {
				handler.handleTaskEventFunc = func(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
					return nil
				}
			},
			wantErr: false,
		},
		{
			name:        "invalid task ID format",
			taskID:      "invalid-uuid",
			eventType:   "heartbeat",
			payload:     map[string]interface{}{},
			mockSetup:   func(handler *mockTaskEventHandler) {},
			wantErr:     true,
			errContains: "invalid task ID format",
		},
		{
			name:        "empty event type",
			taskID:      taskID.String(),
			eventType:   "",
			payload:     map[string]interface{}{},
			mockSetup:   func(handler *mockTaskEventHandler) {},
			wantErr:     true,
			errContains: "event_type is required",
		},
		{
			name:      "event handler not configured",
			taskID:    taskID.String(),
			eventType: "heartbeat",
			payload:   map[string]interface{}{},
			mockSetup: func(handler *mockTaskEventHandler) {},
			wantErr:   true,
		},
		{
			name:      "event handler returns error",
			taskID:    taskID.String(),
			eventType: "failed",
			payload: map[string]interface{}{
				"error": "something went wrong",
			},
			mockSetup: func(handler *mockTaskEventHandler) {
				handler.handleTaskEventFunc = func(ctx context.Context, taskID string, eventType string, payload map[string]interface{}) error {
					return errors.NewInternalError("internal error")
				}
			},
			wantErr:     true,
			errContains: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskRepo := &mockTaskRepository{}
			intRepo := &mockInterventionRepository{}
			eventHandler := &mockTaskEventHandler{}
			tt.mockSetup(eventHandler)

			svc := NewInterventionService(taskRepo, intRepo).(*interventionService)

			// Only set handler for tests that are not testing "not configured"
			if tt.name != "event handler not configured" {
				svc.SetEventHandler(eventHandler)
			}

			err := svc.HandleWrapperEvent(ctx, tt.taskID, tt.eventType, tt.payload)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HandleWrapperEvent() expected error, got nil")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("HandleWrapperEvent() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("HandleWrapperEvent() unexpected error = %v", err)
			}
		})
	}
}
