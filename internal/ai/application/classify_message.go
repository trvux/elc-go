package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

// ClassifyMessage runs the guardrail check against the first classifier
// model that responds, falling back through classifierModels on a
// retryable error same as SendChatMessage. Returns (nil, nil) — not an
// error — if classifierModels is empty, every model failed, or the
// response couldn't be parsed as a verdict: fail open. Blocking real
// customers because the guardrail is unconfigured or hiccuped is worse than
// occasionally letting one off-topic message reach the (still-guarded-by-
// its-own-system-prompt) main model — same "optional, degrade gracefully"
// precedent as SMTP elsewhere in this codebase.
func ClassifyMessage(ctx context.Context, clientFactory domain.LLMClientFactory, classifierModels []domain.ModelConfig, message string) *domain.ClassificationResult {
	if len(classifierModels) == 0 {
		return nil
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
		return verdict
	}

	return nil
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
