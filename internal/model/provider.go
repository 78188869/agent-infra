// Package model provides database models for the application.
package model

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProviderScope represents the scope of a provider.
type ProviderScope string

const (
	ProviderScopeSystem ProviderScope = "system"
	ProviderScopeTenant ProviderScope = "tenant"
	ProviderScopeUser   ProviderScope = "user"
)

// ProviderType represents the type of a provider.
type ProviderType string

const (
	ProviderTypeClaudeCode        ProviderType = "claude_code"
	ProviderTypeAnthropicCompat   ProviderType = "anthropic_compatible"
	ProviderTypeOpenAICompat      ProviderType = "openai_compatible"
	ProviderTypeCustom            ProviderType = "custom"
)

// ProviderStatus represents the status of a provider.
type ProviderStatus string

const (
	ProviderStatusActive     ProviderStatus = "active"
	ProviderStatusInactive   ProviderStatus = "inactive"
	ProviderStatusDeprecated ProviderStatus = "deprecated"
)

// RuntimeType represents the runtime type of a provider.
type RuntimeType string

const (
	RuntimeTypeCLI RuntimeType = "cli"
	RuntimeTypeAPI RuntimeType = "api"
	RuntimeTypeSDK RuntimeType = "sdk"
)

// Provider represents an Agent runtime configuration.
// Providers support multiple AI models and runtimes, similar to "cc switch".
type Provider struct {
	ID string `gorm:"type:varchar(36);primaryKey" json:"id"`

	// Scope (three-tier: system/tenant/user)
	Scope   ProviderScope `gorm:"type:enum('system','tenant','user');not null;default:'system';uniqueIndex:uk_scope_name;index:idx_scope_tenant;index:idx_scope_user" json:"scope"`
	TenantID *string       `gorm:"type:varchar(36);uniqueIndex:uk_scope_name;index:idx_scope_tenant" json:"tenant_id"`
	UserID   *string       `gorm:"type:varchar(36);uniqueIndex:uk_scope_name;index:idx_scope_user" json:"user_id"`

	// Basic Information
	Name        string       `gorm:"type:varchar(64);not null;uniqueIndex:uk_scope_name" json:"name"`
	Type        ProviderType `gorm:"type:enum('claude_code','anthropic_compatible','openai_compatible','custom');not null;index:idx_type_status" json:"type"`
	Description string       `gorm:"type:text" json:"description"`

	// API Configuration
	APIEndpoint string `gorm:"type:varchar(512)" json:"api_endpoint"` // ANTHROPIC_BASE_URL
	APIKeyRef   string `gorm:"type:varchar(256)" json:"api_key_ref"`  // K8s Secret reference

	// Model Mapping (compatible with cc switch)
	ModelMapping datatypes.JSON `gorm:"type:json" json:"model_mapping"`
	// Example: {"default": "glm-5", "opus": "glm-5", "sonnet": "glm-4.7", "haiku": "glm-4.5-air"}

	// Runtime Configuration
	RuntimeType    RuntimeType    `gorm:"type:enum('cli','api','sdk');default:'cli'" json:"runtime_type"`
	RuntimeImage   string         `gorm:"type:varchar(256)" json:"runtime_image"`
	RuntimeCommand datatypes.JSON `gorm:"type:json" json:"runtime_command"`

	// cc switch compatible configuration
	EnvVars        datatypes.JSON `gorm:"type:json" json:"env_vars"`         // Environment variables
	Permissions    datatypes.JSON `gorm:"type:json" json:"permissions"`      // Permission config
	EnabledPlugins datatypes.JSON `gorm:"type:json" json:"enabled_plugins"` // Enabled plugins

	// Extra parameters
	ExtraParams datatypes.JSON `gorm:"type:json" json:"extra_params"`

	// Status
	Status ProviderStatus `gorm:"type:enum('active','inactive','deprecated');default:'active';index:idx_type_status" json:"status"`

	// Timestamps
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName returns the table name for Provider.
func (Provider) TableName() string {
	return "providers"
}

// BeforeCreate is a GORM hook that generates a UUID before creating a record.
func (p *Provider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = generateUUID()
	}
	return nil
}

// IsActive checks if the provider is active.
func (p *Provider) IsActive() bool {
	return p.Status == ProviderStatusActive
}

// IsSystemProvider checks if the provider is a system-level provider.
func (p *Provider) IsSystemProvider() bool {
	return p.Scope == ProviderScopeSystem
}

// IsTenantProvider checks if the provider is a tenant-level provider.
func (p *Provider) IsTenantProvider() bool {
	return p.Scope == ProviderScopeTenant
}

// IsUserProvider checks if the provider is a user-level provider.
func (p *Provider) IsUserProvider() bool {
	return p.Scope == ProviderScopeUser
}

// ModelMappingConfig represents the model mapping configuration.
type ModelMappingConfig struct {
	DefaultModel   string `json:"default,omitempty"`
	OpusModel      string `json:"opus,omitempty"`
	SonnetModel    string `json:"sonnet,omitempty"`
	HaikuModel     string `json:"haiku,omitempty"`
	ReasoningModel string `json:"reasoning,omitempty"`
}

// PermissionsConfig represents the permissions configuration.
type PermissionsConfig struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// RuntimeCommandConfig represents the runtime command configuration.
type RuntimeCommandConfig struct {
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
}

// EnvVarConfig represents environment variable configuration.
type EnvVarConfig map[string]string

// EnabledPluginsConfig represents enabled plugins configuration.
type EnabledPluginsConfig map[string]bool
