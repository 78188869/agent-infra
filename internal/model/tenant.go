package model

// Tenant status constants.
const (
	TenantStatusActive    = "active"
	TenantStatusSuspended = "suspended"
)

// Tenant represents a tenant in the system with resource quotas.
type Tenant struct {
	BaseModel
	Name             string `gorm:"type:varchar(128);not null" json:"name"`
	QuotaCPU         int    `gorm:"default:4" json:"quota_cpu"`
	QuotaMemory      int64  `gorm:"default:16" json:"quota_memory"`       // GB
	QuotaConcurrency int    `gorm:"default:10" json:"quota_concurrency"`  // concurrent tasks
	QuotaDailyTasks  int    `gorm:"default:100" json:"quota_daily_tasks"` // daily task limit
	Status           string `gorm:"type:enum('active','suspended');default:'active'" json:"status"`
}

// TableName returns the table name for the Tenant model.
func (Tenant) TableName() string {
	return "tenants"
}
