// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// CapabilityType represents the type of a capability.
type CapabilityType string

const (
	CapabilityTypeTool         CapabilityType = "tool"
	CapabilityTypeSkill        CapabilityType = "skill"
	CapabilityTypeAgentRuntime CapabilityType = "agent_runtime"
)

// CapabilityStatus represents the status of a capability.
type CapabilityStatus string

const (
	CapabilityStatusActive   CapabilityStatus = "active"
	CapabilityStatusInactive CapabilityStatus = "inactive"
)

// PermissionLevel represents the permission level of a capability.
type PermissionLevel string

const (
	PermissionLevelPublic     PermissionLevel = "public"
	PermissionLevelRestricted PermissionLevel = "restricted"
	PermissionLevelAdminOnly  PermissionLevel = "admin_only"
)

// Capability represents a registered capability in the system.
// Capabilities can be tools, skills, or agent runtimes.
type Capability struct {
	ID       string           `gorm:"type:varchar(36);primaryKey" json:"id"`
	TenantID *string          `gorm:"type:varchar(36);uniqueIndex:uk_tenant_type_name" json:"tenant_id"` // NULL means global capability

	// Capability Information
	Type        CapabilityType `gorm:"type:enum('tool','skill','agent_runtime');not null;uniqueIndex:uk_tenant_type_name;index:idx_type_status" json:"type"`
	Name        string         `gorm:"type:varchar(64);not null;uniqueIndex:uk_tenant_type_name" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Version     string         `gorm:"type:varchar(32);default:'1.0.0'" json:"version"`

	// Configuration
	Config datatypes.JSON `gorm:"type:json" json:"config"`
	Schema datatypes.JSON `gorm:"type:json" json:"schema"` // Parameter schema

	// Permissions
	PermissionLevel PermissionLevel `gorm:"type:enum('public','restricted','admin_only');default:'public'" json:"permission_level"`

	// Status
	Status CapabilityStatus `gorm:"type:enum('active','inactive');default:'active';index:idx_type_status" json:"status"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

// TableName returns the table name for Capability.
func (Capability) TableName() string {
	return "capabilities"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (c *Capability) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = generateUUID()
	}
	return nil
}

// IsActive checks if the capability is active.
func (c *Capability) IsActive() bool {
	return c.Status == CapabilityStatusActive
}

// IsGlobal checks if the capability is a global (system-wide) capability.
func (c *Capability) IsGlobal() bool {
	return c.TenantID == nil
}

// IsTool checks if the capability is a tool.
func (c *Capability) IsTool() bool {
	return c.Type == CapabilityTypeTool
}

// IsSkill checks if the capability is a skill.
func (c *Capability) IsSkill() bool {
	return c.Type == CapabilityTypeSkill
}

// IsAgentRuntime checks if the capability is an agent runtime.
func (c *Capability) IsAgentRuntime() bool {
	return c.Type == CapabilityTypeAgentRuntime
}

// CapabilityConfig represents the configuration of a capability.
type CapabilityConfig struct {
	// For tools
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`

	// For skills
	PackageURL  string            `json:"package_url,omitempty"`
	Entry       string            `json:"entry,omitempty"`

	// For agent runtimes
	RuntimeType string            `json:"runtime_type,omitempty"`
	Image       string            `json:"image,omitempty"`
}

// CapabilitySchema represents the parameter schema of a capability.
type CapabilitySchema struct {
	Type       string                            `json:"type"`
	Properties map[string]CapabilitySchemaProp   `json:"properties,omitempty"`
	Required   []string                          `json:"required,omitempty"`
}

// CapabilitySchemaProp represents a property in the capability schema.
type CapabilitySchemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Default     interface{} `json:"default,omitempty"`
}
