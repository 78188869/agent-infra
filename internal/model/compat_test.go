package model

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open SQLite in-memory db: %v", err)
	}
	return db
}

func TestSQLite_AllModelsAutoMigrate(t *testing.T) {
	// SQLite requires globally unique index names. Some models share index
	// names (e.g. idx_type_status on both Capability and Provider), so we
	// migrate each model into its own in-memory database to verify that every
	// model's schema is individually compatible with SQLite.
	models := AllModels()
	for _, mdl := range models {
		t.Run(fmt.Sprintf("%T", mdl), func(t *testing.T) {
			db := setupSQLiteTestDB(t)
			if err := db.AutoMigrate(mdl); err != nil {
				t.Errorf("AutoMigrate(%T) failed on SQLite: %v", mdl, err)
			}
		})
	}
}

func TestSQLite_TenantCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Tenant{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	tenant := &Tenant{
		Name:             "TestTenant",
		QuotaCPU:         4,
		QuotaMemory:      16,
		QuotaConcurrency: 10,
		QuotaDailyTasks:  100,
		Status:           TenantStatusActive,
	}
	tenant.ID = uuid.New()
	if err := db.Create(tenant).Error; err != nil {
		t.Fatalf("Create tenant failed: %v", err)
	}

	var found Tenant
	if err := db.First(&found, "id = ?", tenant.ID).Error; err != nil {
		t.Fatalf("Read tenant failed: %v", err)
	}
	if found.Name != "TestTenant" {
		t.Errorf("Name = %q, want TestTenant", found.Name)
	}
	if found.Status != TenantStatusActive {
		t.Errorf("Status = %q, want %q", found.Status, TenantStatusActive)
	}
}

func TestSQLite_TaskCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Task{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	task := &Task{
		TenantID:   generateUUID(),
		CreatorID:  generateUUID(),
		ProviderID: generateUUID(),
		Name:       "TestTask",
		Status:     TaskStatusPending,
		Priority:   TaskPriorityHigh,
	}
	task.ID = uuid.New()
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("Create task failed: %v", err)
	}

	var found Task
	if err := db.First(&found, "id = ?", task.ID).Error; err != nil {
		t.Fatalf("Read task failed: %v", err)
	}
	if found.Status != TaskStatusPending {
		t.Errorf("Status = %q, want %q", found.Status, TaskStatusPending)
	}
	if found.Priority != TaskPriorityHigh {
		t.Errorf("Priority = %q, want %q", found.Priority, TaskPriorityHigh)
	}
}

func TestSQLite_TemplateWithLongSpec(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&Template{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	longSpec := ""
	for i := 0; i < 10000; i++ {
		longSpec += "x"
	}

	tmpl := &Template{
		TenantID:  generateUUID(),
		Name:      "test-template",
		Spec:      longSpec,
		SceneType: TemplateSceneTypeCoding,
		Status:    TemplateStatusDraft,
	}
	tmpl.ID = uuid.New()
	if err := db.Create(tmpl).Error; err != nil {
		t.Fatalf("Create template with long spec failed: %v", err)
	}

	var found Template
	if err := db.First(&found, "id = ?", tmpl.ID).Error; err != nil {
		t.Fatalf("Read template failed: %v", err)
	}
	if len(found.Spec) != 10000 {
		t.Errorf("Spec length = %d, want 10000", len(found.Spec))
	}
}

func TestSQLite_ExecutionLogCRUD(t *testing.T) {
	db := setupSQLiteTestDB(t)
	if err := db.AutoMigrate(&ExecutionLog{}); err != nil {
		t.Fatalf("AutoMigrate failed: %v", err)
	}

	log := &ExecutionLog{
		TaskID:    generateUUID(),
		EventType: EventTypeStatusChange,
		EventName: "status_changed",
	}
	if err := db.Create(log).Error; err != nil {
		t.Fatalf("Create execution log failed: %v", err)
	}

	var found ExecutionLog
	if err := db.First(&found, "task_id = ?", log.TaskID).Error; err != nil {
		t.Fatalf("Read execution log failed: %v", err)
	}
	if found.EventType != EventTypeStatusChange {
		t.Errorf("EventType = %q, want %q", found.EventType, EventTypeStatusChange)
	}
}
