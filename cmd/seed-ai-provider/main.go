// Command seed-ai-provider creates or updates a single ai_providers row
// (plus its chat model, and an optional classifier model) from environment
// variables — the one-time bootstrap so a fresh deploy doesn't require
// using the admin API by hand before the first customer chat can work.
// After this, provider/model management goes through the admin panel
// (POST/PUT /ai/providers, /ai/models), same break-glass relationship
// cmd/seed-admin has to the users admin screen.
//
// Usage:
//
//	AI_SECRETS_ENCRYPTION_KEY=$(openssl rand -base64 32) \
//	AI_PROVIDER_NAME=deepseek AI_PROVIDER_DISPLAY_NAME="DeepSeek" \
//	AI_PROVIDER_BASE_URL=https://api.deepseek.com/v1 AI_PROVIDER_API_KEY=sk-... \
//	AI_PROVIDER_MODEL=deepseek-chat \
//	AI_PROVIDER_PRICING_DOC_URL=https://api-docs.deepseek.com/quick_start/pricing/ \
//	AI_CLASSIFIER_MODEL=deepseek-chat \
//	  go run ./cmd/seed-ai-provider
//
// Pricing is seeded zeroed — cmd/sync-ai-pricing fills in real numbers from
// AI_PROVIDER_PRICING_DOC_URL on its first run, so no price is ever
// hand-typed here. Re-running is safe: an existing provider's API key is
// rotated (break-glass), but existing model rows are left untouched so it
// never clobbers fallback priority/pricing an admin has since edited.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/ai/application"
	"github.com/trvux/elc-go/internal/ai/domain"
	aiinfra "github.com/trvux/elc-go/internal/ai/infrastructure"
	"github.com/trvux/elc-go/internal/platform/db"
)

func main() {
	_ = godotenv.Load()

	name := envOr("AI_PROVIDER_NAME", "deepseek")
	displayName := envOr("AI_PROVIDER_DISPLAY_NAME", name)
	baseURL := requireEnv("AI_PROVIDER_BASE_URL")
	apiKey := requireEnv("AI_PROVIDER_API_KEY")
	modelName := requireEnv("AI_PROVIDER_MODEL")
	pricingDocURL := os.Getenv("AI_PROVIDER_PRICING_DOC_URL")
	classifierModel := os.Getenv("AI_CLASSIFIER_MODEL")

	ctx := context.Background()

	cipher, err := aiinfra.NewSecretCipherFromEnv("AI_SECRETS_ENCRYPTION_KEY")
	if err != nil {
		fatalf("%v", err)
	}

	pool, err := db.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	providerRepo := aiinfra.NewPostgresProviderRepository(pool, cipher)
	modelRepo := aiinfra.NewPostgresModelRepository(pool, cipher)

	provider, err := findProviderByName(ctx, providerRepo, name)
	if err != nil {
		fatalf("look up existing provider: %v", err)
	}

	if provider == nil {
		provider, err = application.CreateProvider(ctx, providerRepo, application.CreateProviderInput{
			Name: name, DisplayName: displayName, BaseURL: baseURL, APIKey: apiKey, PricingDocURL: pricingDocURL,
		})
		if err != nil {
			fatalf("create provider: %v", err)
		}
		fmt.Printf("seed-ai-provider: created provider %q (%s)\n", name, provider.ID())
	} else {
		if provider, err = application.UpdateProvider(ctx, providerRepo, application.UpdateProviderInput{
			ID: provider.ID(), DisplayName: &displayName, BaseURL: &baseURL, APIKey: &apiKey, PricingDocURL: &pricingDocURL,
		}); err != nil {
			fatalf("update existing provider: %v", err)
		}
		fmt.Printf("seed-ai-provider: updated existing provider %q (%s)\n", name, provider.ID())
	}

	if err := ensureModel(ctx, modelRepo, provider.ID(), modelName, domain.ModelRoleChat); err != nil {
		fatalf("ensure chat model: %v", err)
	}
	if classifierModel != "" {
		if err := ensureModel(ctx, modelRepo, provider.ID(), classifierModel, domain.ModelRoleClassifier); err != nil {
			fatalf("ensure classifier model: %v", err)
		}
	}

	fmt.Println("seed-ai-provider: done")
}

func findProviderByName(ctx context.Context, repo domain.ProviderRepository, name string) (*domain.Provider, error) {
	providers, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range providers {
		if p.Name() == name {
			return p, nil
		}
	}
	return nil, nil
}

// ensureModel creates providerID's (modelName, role) row with zeroed
// pricing if it doesn't already exist; leaves an existing row untouched.
func ensureModel(ctx context.Context, repo domain.ModelRepository, providerID, modelName string, role domain.ModelRole) error {
	models, err := repo.List(ctx)
	if err != nil {
		return err
	}
	for _, m := range models {
		if m.ProviderID() == providerID && m.ModelName() == modelName && m.Role() == role {
			fmt.Printf("seed-ai-provider: model %q (role=%s) already exists, leaving it as-is\n", modelName, role)
			return nil
		}
	}

	model, err := application.CreateModel(ctx, repo, application.CreateModelInput{
		ProviderID: providerID, ModelName: modelName, DisplayName: modelName, Role: role,
		Pricing: domain.Pricing{OffPeakMultiplier: 1.0}, FallbackPriority: 0,
	})
	if err != nil {
		return err
	}
	fmt.Printf("seed-ai-provider: created model %q (role=%s, %s) — run cmd/sync-ai-pricing to fill in real pricing\n", modelName, role, model.ID())
	return nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("%s is required", key)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
