package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/trvux/elc-go/internal/ai/domain"
)

// ClassifierSystemPrompt instructs the guardrail model to answer with
// nothing but a JSON verdict — short input/output on purpose, this call is
// meant to be cheap: run before (and instead of, if it rejects) the full
// chat completion, so blocking off-topic traffic *reduces* average cost per
// request rather than adding to it.
const ClassifierSystemPrompt = `Bạn là bộ lọc cho chatbot tư vấn bán hàng của ELC (cửa hàng điện máy). ` +
	`Xác định tin nhắn của khách có liên quan đến tìm hiểu/tư vấn/mua sản phẩm điện máy tại ELC hay không. ` +
	`Từ chối (on_topic=false) nếu: tin nhắn không liên quan mua sắm điện máy (bài tập, việc cá nhân, chủ đề khác), ` +
	`hoặc yêu cầu so sánh/nhắc tên đối thủ cạnh tranh (Điện Máy Xanh, Điện Máy Chợ Lớn, Nguyễn Kim, Pico...). ` +
	`Chấp nhận (on_topic=true) mọi câu hỏi về sản phẩm, giá, tư vấn mua hàng điện máy, kể cả chào hỏi thông thường. ` +
	`CHỈ trả lời đúng 1 dòng JSON, không thêm chữ nào khác: {"on_topic": true|false, "reason": "..."}`

// classifyCacheTTL: the verdict for an exact message text doesn't change
// over time, so this is generous — 1 hour balances catching genuinely
// repeated/common questions (the actual cost-saving case) against a
// borderline on/off-topic misclassification lingering too long if the
// classifier got one wrong.
const classifyCacheTTL = 1 * time.Hour

// ClassifyMessage runs the guardrail check against the first classifier
// model that responds, falling back through classifierModels on a
// retryable error same as SendChatMessage. Returns (nil, nil) — not an
// error — if classifierModels is empty, every model failed, or the
// response couldn't be parsed as a verdict: fail open. Blocking real
// customers because the guardrail is unconfigured or hiccuped is worse than
// occasionally letting one off-topic message reach the (still-guarded-by-
// its-own-system-prompt) main model — same "optional, degrade gracefully"
// precedent as SMTP elsewhere in this codebase.
//
// cache is optional (nil disables it, see domain.Cache's doc comment) —
// when set, an identical message text (trimmed+lowercased) skips the LLM
// call entirely for classifyCacheTTL, the real cost saving this exists for
// (see docs/rfc/2026-08-18-ai-chat-redis.md, unlike caching search_products
// which only saves DB load/latency, not LLM spend).
func ClassifyMessage(ctx context.Context, clientFactory domain.LLMClientFactory, classifierModels []domain.ModelConfig, cache domain.Cache, message string) *domain.ClassificationResult {
	if len(classifierModels) == 0 {
		return nil
	}

	cacheKey := classifyCacheKey(message)
	if cache != nil {
		if cached, ok, err := cache.Get(ctx, cacheKey); err == nil && ok {
			if verdict, err := decodeClassification(cached); err == nil {
				return verdict
			}
		}
	}

	messages := []domain.Message{
		{Role: domain.RoleSystem, Content: ClassifierSystemPrompt},
		{Role: domain.RoleUser, Content: message},
	}

	for _, cfg := range classifierModels {
		client := clientFactory(cfg)
		result, err := client.Chat(ctx, messages, nil)
		if err != nil {
			if isRetryable(err) {
				continue
			}
			return nil
		}

		verdict, err := parseClassification(result.Message.Content)
		if err != nil {
			return nil
		}
		if cache != nil {
			if encoded, err := json.Marshal(verdict); err == nil {
				_ = cache.Set(ctx, cacheKey, string(encoded), classifyCacheTTL)
			}
		}
		return verdict
	}

	return nil
}

// classifyCacheKey normalizes message (trim + lowercase) before hashing so
// trivial whitespace/casing differences still hit the same cache entry.
func classifyCacheKey(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	sum := sha256.Sum256([]byte(normalized))
	return "classify:" + hex.EncodeToString(sum[:])
}

func decodeClassification(raw string) (*domain.ClassificationResult, error) {
	var verdict domain.ClassificationResult
	if err := json.Unmarshal([]byte(raw), &verdict); err != nil {
		return nil, err
	}
	return &verdict, nil
}

func parseClassification(content string) (*domain.ClassificationResult, error) {
	start := strings.IndexByte(content, '{')
	end := strings.LastIndexByte(content, '}')
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("ai: classifier response has no JSON object: %q", content)
	}

	var raw struct {
		OnTopic bool   `json:"on_topic"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("ai: decode classifier response: %w", err)
	}
	return &domain.ClassificationResult{OnTopic: raw.OnTopic, Reason: raw.Reason}, nil
}
