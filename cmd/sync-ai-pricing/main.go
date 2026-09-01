// Command sync-ai-pricing refreshes ai_models.pricing from each provider's
// official pricing page (ai_providers.pricing_doc_url) — pricing is never
// hand-typed into a migration/seed, see the Phase 1 RFC's reasoning: prices
// change, and a one-time scraped snapshot goes stale silently. Only updates
// pricing for model_names that already exist in ai_models for that
// provider; a model the page mentions that isn't already configured is only
// logged, never auto-created — which models enter the fallback chain stays
// an admin (human) decision.
//
// A plain one-shot CLI like every other cmd/ here, meant to run on a
// schedule via a GitHub Actions cron workflow (mirrors
// .github/workflows/migrate-images.yml's scheduled-workflow-hitting-the-DB
// pattern) — `go run ./cmd/sync-ai-pricing` runs it once.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/trvux/elc-go/internal/ai/domain"
	aiinfra "github.com/trvux/elc-go/internal/ai/infrastructure"
	"github.com/trvux/elc-go/internal/platform/db"
)

// maxDocChars bounds how much of a pricing page is handed to the
// extraction model — pricing tables are always near the top, and a hard
// cap keeps this a cheap, bounded-cost call regardless of page size.
const maxDocChars = 60_000

func main() {
	_ = godotenv.Load()

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

	// Whatever chat model is already configured does the extraction — this
	// is a business call (parsing a page), not customer-facing, so it
	// doesn't need its own "extractor" role: any active chat model is cheap
	// enough for one page a day.
	extractionModels, err := modelRepo.ListActiveConfigsByRole(ctx, domain.ModelRoleChat)
	if err != nil {
		fatalf("resolve extraction model: %v", err)
	}
	if len(extractionModels) == 0 {
		fatalf("no active chat model configured — cannot extract pricing")
	}
	client := aiinfra.NewLLMClient(extractionModels[0])

	providers, err := providerRepo.List(ctx)
	if err != nil {
		fatalf("list providers: %v", err)
	}

	for _, provider := range providers {
		if provider.PricingDocURL() == "" {
			continue
		}
		if err := syncProviderPricing(ctx, client, modelRepo, provider); err != nil {
			fmt.Fprintf(os.Stderr, "sync-ai-pricing: %s: %v\n", provider.Name(), err)
		}
	}
}

func syncProviderPricing(ctx context.Context, client domain.LLMClient, modelRepo domain.ModelRepository, provider *domain.Provider) error {
	allModels, err := modelRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("list models: %w", err)
	}
	var modelNames []string
	existingByName := map[string]domain.Pricing{}
	for _, m := range allModels {
		if m.ProviderID() == provider.ID() {
			modelNames = append(modelNames, m.ModelName())
			existingByName[m.ModelName()] = m.Pricing()
		}
	}
	if len(modelNames) == 0 {
		return nil
	}

	docText, err := fetchDocText(provider.PricingDocURL())
	if err != nil {
		return fmt.Errorf("fetch pricing doc: %w", err)
	}

	extracted, err := extractPricing(ctx, client, docText, modelNames)
	if err != nil {
		return fmt.Errorf("extract pricing: %w", err)
	}

	for _, modelName := range modelNames {
		pricing, ok := extracted[modelName]
		if !ok {
			fmt.Printf("sync-ai-pricing: %s: no pricing found for model %q in doc, skipping\n", provider.Name(), modelName)
			continue
		}
		if err := validateExtractedPricing(existingByName[modelName], pricing); err != nil {
			fmt.Fprintf(os.Stderr, "sync-ai-pricing: %s: %q: rejecting extracted pricing: %v\n", provider.Name(), modelName, err)
			continue
		}
		if _, err := modelRepo.UpdatePricing(ctx, provider.ID(), modelName, pricing); err != nil {
			return fmt.Errorf("update pricing for %q: %w", modelName, err)
		}
		fmt.Printf("sync-ai-pricing: %s: updated pricing for %q\n", provider.Name(), modelName)
	}

	for name := range extracted {
		if !contains(modelNames, name) {
			fmt.Printf("sync-ai-pricing: %s: pricing page mentions model %q, not configured — add it via the admin panel if it should be usable\n", provider.Name(), name)
		}
	}
	return nil
}

// maxPricingChangeRatio bounds how far a newly-extracted price may move from
// whatever is already stored, once a real (non-zero) price is already on
// file. The extraction pipeline (fetch an untrusted web page, hand it to an
// LLM, parse whatever JSON comes back, see extractPricing) has no
// independent way to confirm the number is actually right — a jump this
// large is far more likely a bad extraction (page layout changed, LLM
// misread the table, ...) than a real same-day price change, so it's logged
// and rejected rather than silently written into ai_models.pricing, which
// internal/ai's Cost() feeds directly into usage/cost reporting. See
// docs/rfc/2026-09-02-backend-code-review-round2.md §3.11.
const maxPricingChangeRatio = 5.0

// validateExtractedPricing rejects a newly-extracted Pricing that can't be a
// real price (negative, or zero when a genuine provider pricing page always
// lists a non-zero rate) or that swings implausibly far from what's already
// stored for this model. old is the zero Pricing{} on a model's first ever
// sync, in which case the change-ratio check is skipped (nothing on file yet
// to compare against).
func validateExtractedPricing(old, extracted domain.Pricing) error {
	if extracted.InputCacheMissPeak < 0 || extracted.OutputPeak < 0 {
		return fmt.Errorf("negative price (input=%v, output=%v)", extracted.InputCacheMissPeak, extracted.OutputPeak)
	}
	if extracted.InputCacheHitPeak != nil && *extracted.InputCacheHitPeak < 0 {
		return fmt.Errorf("negative cache-hit price (%v)", *extracted.InputCacheHitPeak)
	}
	if extracted.InputCacheMissPeak == 0 || extracted.OutputPeak == 0 {
		return fmt.Errorf("zero price extracted (input=%v, output=%v)", extracted.InputCacheMissPeak, extracted.OutputPeak)
	}
	if pricingChangedTooMuch(old.InputCacheMissPeak, extracted.InputCacheMissPeak) {
		return fmt.Errorf("input price moved more than %vx vs. stored (%v -> %v)", maxPricingChangeRatio, old.InputCacheMissPeak, extracted.InputCacheMissPeak)
	}
	if pricingChangedTooMuch(old.OutputPeak, extracted.OutputPeak) {
		return fmt.Errorf("output price moved more than %vx vs. stored (%v -> %v)", maxPricingChangeRatio, old.OutputPeak, extracted.OutputPeak)
	}
	return nil
}

func pricingChangedTooMuch(old, extracted float64) bool {
	if old <= 0 {
		return false
	}
	ratio := extracted / old
	return ratio > maxPricingChangeRatio || ratio < 1/maxPricingChangeRatio
}

func fetchDocText(url string) (string, error) {
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDocChars))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func extractPricing(ctx context.Context, client domain.LLMClient, docText string, modelNames []string) (map[string]domain.Pricing, error) {
	prompt := fmt.Sprintf(`Trích xuất bảng giá cho các model sau từ nội dung trang docs bên dưới: %s.
Chỉ trả về JSON object, key là tên model (đúng như trong danh sách trên), value là object đúng shape:
{"currency":"USD","perMillionTokens":true,"inputCacheHitPeak":<number hoặc null>,"inputCacheMissPeak":<number>,"outputPeak":<number>,"offPeakMultiplier":<number, dùng 1.0 nếu trang không phân biệt peak/off-peak>,"peakWindowsUtc":[{"startHour":<int>,"endHour":<int>}] (bỏ trống nếu không có khái niệm peak/off-peak)}.
Model nào không tìm thấy trong trang thì bỏ qua, đừng bịa số. CHỈ trả JSON, không thêm chữ nào khác.

Nội dung trang:
%s`, strings.Join(modelNames, ", "), docText)

	result, err := client.Chat(ctx, []domain.Message{
		{Role: domain.RoleUser, Content: prompt},
	}, nil)
	if err != nil {
		return nil, err
	}

	content := result.Message.Content
	start := strings.IndexByte(content, '{')
	end := strings.LastIndexByte(content, '}')
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON object in extraction response")
	}

	var raw map[string]domain.Pricing
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("decode extraction response: %w", err)
	}
	return raw, nil
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
