package model

// Template scene type constants.
const (
	TemplateSceneTypeCoding   = "coding"
	TemplateSceneTypeOps      = "ops"
	TemplateSceneTypeAnalysis = "analysis"
	TemplateSceneTypeContent  = "content"
	TemplateSceneTypeCustom   = "custom"
)

// Template status constants.
const (
	TemplateStatusDraft      = "draft"
	TemplateStatusPublished  = "published"
	TemplateStatusDeprecated = "deprecated"
)

// Template represents an agent template with its configuration specification.
type Template struct {
	BaseModel
	TenantID   string  `gorm:"type:varchar(36);index" json:"tenant_id"`
	Name       string  `gorm:"type:varchar(128);not null" json:"name"`
	Version    string  `gorm:"type:varchar(32);default:'1.0.0'" json:"version"`
	Spec       string  `gorm:"type:text" json:"spec"`
	SceneType  string  `gorm:"type:varchar(20);default:'custom'" json:"scene_type"`
	Status     string  `gorm:"type:varchar(20);default:'draft'" json:"status"`
	ProviderID *string `gorm:"type:varchar(36)" json:"provider_id,omitempty"`
}

// TableName returns the table name for the Template model.
func (Template) TableName() string {
	return "templates"
}
