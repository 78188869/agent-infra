// Package seed provides database seeding functionality.
package seed

import (
	"context"
	"log"

	"github.com/example/agent-infra/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SystemProviders defines the system preset providers.
var SystemProviders = []model.Provider{
	{
		Scope:        model.ProviderScopeSystem,
		Name:         "claude-code",
		Type:         model.ProviderTypeClaudeCode,
		Description:  "Official Claude Code CLI from Anthropic",
		APIEndpoint:  "https://api.anthropic.com",
		RuntimeType:  model.RuntimeTypeCLI,
		Status:       model.ProviderStatusActive,
	},
	{
		Scope:        model.ProviderScopeSystem,
		Name:         "zhipu-glm",
		Type:         model.ProviderTypeAnthropicCompat,
		Description:  "Zhipu GLM via Anthropic compatible API",
		APIEndpoint:  "https://open.bigmodel.cn/api/anthropic",
		APIKeyRef:    "zhipu-api-key",
		ModelMapping: datatypes.JSON(`{"default":"glm-5","opus":"glm-5","sonnet":"glm-4.7","haiku":"glm-4.5-air"}`),
		RuntimeType:  model.RuntimeTypeCLI,
		Status:       model.ProviderStatusActive,
	},
	{
		Scope:        model.ProviderScopeSystem,
		Name:         "deepseek",
		Type:         model.ProviderTypeAnthropicCompat,
		Description:  "DeepSeek via Anthropic compatible API",
		APIEndpoint:  "https://api.deepseek.com",
		APIKeyRef:    "deepseek-api-key",
		ModelMapping: datatypes.JSON(`{"default":"deepseek-chat","opus":"deepseek-reasoner","sonnet":"deepseek-chat","haiku":"deepseek-chat"}`),
		RuntimeType:  model.RuntimeTypeCLI,
		Status:       model.ProviderStatusActive,
	},
}

// SeedProviders initializes system providers if they don't exist.
func SeedProviders(db *gorm.DB) error {
	ctx := context.Background()

	for _, provider := range SystemProviders {
		var existing model.Provider
		err := db.WithContext(ctx).
			Where("scope = ? AND name = ?", model.ProviderScopeSystem, provider.Name).
			First(&existing).Error

		if err == gorm.ErrRecordNotFound {
			if err := db.WithContext(ctx).Create(&provider).Error; err != nil {
				return err
			}
			log.Printf("Seeded provider: %s", provider.Name)
		} else if err != nil {
			return err
		}
	}

	return nil
}
