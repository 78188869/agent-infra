// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// TemplateStatus represents the status of a template.
type TemplateStatus string

const (
	TemplateStatusDraft      TemplateStatus = "draft"
	TemplateStatusPublished  TemplateStatus = "published"
	TemplateStatusDeprecated TemplateStatus = "deprecated"
)

// SceneType represents the scene type of a template.
type SceneType string

const (
	SceneTypeCoding  SceneType = "coding"
	SceneTypeOps     SceneType = "ops"
	SceneTypeAnalysis SceneType = "analysis"
	SceneTypeContent SceneType = "content"
	SceneTypeCustom  SceneType = "custom"
)

// Template represents a task template.
// Templates are reusable task configurations that can be parameterized.
type Template struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID    string         `gorm:"type:varchar(36);not null;index:idx_tenant_status;uniqueIndex:uk_tenant_name_version" json:"tenant_id"`
	Name        string         `gorm:"type:varchar(128);not null;uniqueIndex:uk_tenant_name_version" json:"name"`
	Version     string         `gorm:"type:varchar(32);default:'1.0.0';uniqueIndex:uk_tenant_name_version" json:"version"`
	Description string         `gorm:"type:text" json:"description"`

	// Template Definition
	Spec      datatypes.JSON `gorm:"type:mediumtext;not null" json:"spec"`
	SceneType SceneType      `gorm:"type:enum('coding','ops','analysis','content','custom');default:'coding';index" json:"scene_type"`

	// Provider Configuration
	ProviderID *string `gorm:"type:varchar(36)" json:"provider_id"`

	// Status
	Status TemplateStatus `gorm:"type:enum('draft','published','deprecated');default:'draft';index:idx_tenant_status" json:"status"`

	// Audit
	CreatedBy   string     `gorm:"type:varchar(36)" json:"created_by"`
	PublishedAt *time.Time `gorm:"type:timestamp" json:"published_at"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Tenant   *Tenant   `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Provider *Provider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	Tasks    []Task    `gorm:"foreignKey:TemplateID" json:"tasks,omitempty"`
}

// TableName returns the table name for Template.
func (Template) TableName() string {
	return "templates"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (t *Template) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = generateUUID()
	}
	return nil
}

// IsPublished checks if the template is published.
func (t *Template) IsPublished() bool {
	return t.Status == TemplateStatusPublished
}

// CanUse checks if the template can be used for creating tasks.
func (t *Template) CanUse() bool {
	return t.Status == TemplateStatusPublished
}

// TemplateSpec represents the YAML spec of a template.
type TemplateSpec struct {
	Goal        string                 `json:"goal"`
	Context     TemplateContext        `json:"context"`
	Execution   TemplateExecution      `json:"execution"`
	Parameters  []TemplateParameter    `json:"parameters,omitempty"`
}

// TemplateContext represents the context configuration.
type TemplateContext struct {
	Repo           string            `json:"repo,omitempty"`
	Branch         string            `json:"branch,omitempty"`
	InitialContext string            `json:"initial_context,omitempty"`
	EnvVars        map[string]string `json:"env_vars,omitempty"`
}

// TemplateExecution represents the execution configuration.
type TemplateExecution struct {
	Timeout      int                    `json:"timeout,omitempty"`      // seconds
	MaxRetries   int                    `json:"max_retries,omitempty"`
	Resources    TemplateResources      `json:"resources,omitempty"`
	Capabilities TemplateCapabilities   `json:"capabilities,omitempty"`
}

// TemplateResources represents resource limits.
type TemplateResources struct {
	CPU        int   `json:"cpu,omitempty"`         // cores
	Memory     int64 `json:"memory,omitempty"`      // MB
	TokenLimit int64 `json:"token_limit,omitempty"` // max tokens
}

// TemplateCapabilities represents capability configuration.
type TemplateCapabilities struct {
	Tools  []string `json:"tools,omitempty"`
	Skills []string `json:"skills,omitempty"`
}

// TemplateParameter represents a template parameter.
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}
