# RFC: AI chat Phase 2 — SSE streaming trên POST /ai/chat

- **Status**: Completed — merged `89892a9` (Phase 2: SSE streaming on POST /ai/chat).
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: Chỉ streaming — không đụng report chi phí/đọc lịch sử (Phase 3, RFC riêng)

## 1. Bối cảnh

Phase 1 (`internal/ai`, đã merge `main`) trả lời non-streaming: khách gửi 1 tin nhắn, server chờ toàn bộ completion (kể cả vòng gọi tool `search_products`) xong mới trả 1 cục JSON. User xác nhận cần streaming thật (chữ hiện dần) — đây là Phase 2, làm riêng, không dồn chung Phase 3.

`/ai/chat` sẽ **đổi hẳn sang SSE**, không giữ song song 2 dạng response (JSON cũ vs SSE mới) — đơn giản hơn, và FE (elc-tem) chưa code gì dựa trên contract cũ nên đổi bây giờ không phá vỡ gì đang chạy thật.

## 2. Thiết kế

### 2.1. Vì sao stream được cả tool-call loop mà không cần stream tool-call deltas

Khi model quyết định gọi tool (`search_products`), field `content` của round đó luôn rỗng — model không sinh text đồng thời với việc gọi tool. Vậy có thể **stream content delta ở MỌI round một cách vô điều kiện** mà không cần biết trước round đó có phải round cuối hay không: round gọi tool sẽ tự nhiên không có gì để stream (content rỗng), round trả lời thật sẽ stream đúng nội dung theo thời gian thực. → **Vẫn phải** parse/accumulate tool-call fragments (id/name/arguments rải rác qua nhiều chunk, giống chuẩn OpenAI streaming) để biết chính xác tool nào cần gọi với argument gì — chỉ là **không cần stream chúng ra cho khách xem** (vì content rỗng nên không có gì để forward), khác với accumulate content text thì luôn forward ngay lập tức từng phần.

### 2.2. domain.LLMClient thêm 1 phương thức stream

```go
type StreamEvent struct {
    ContentDelta string     // "" nếu round này không sinh text (đang gọi tool)
    Done         bool       // true ở event cuối cùng của round
    ToolCalls    []ToolCall // chỉ có giá trị khi Done=true
    Usage        TokenUsage // chỉ có giá trị khi Done=true
}

type LLMClient interface {
    Chat(ctx, messages, tools) (*ChatResult, error)        // giữ nguyên — guardrail classifier vẫn dùng cái này, không cần stream
    ChatStream(ctx, messages, tools) (<-chan StreamEvent, error)
}
```

`OpenAICompatibleClient.ChatStream` gửi `"stream": true` (+ `stream_options.include_usage`), đọc response theo dòng `data: {...}`, decode incremental, đẩy vào channel; đóng channel khi gặp `data: [DONE]` hoặc lỗi.

### 2.3. application.SendChatMessageStream

Thay `sendWithModel` bằng bản stream: mỗi round gọi `ChatStream`, forward `ContentDelta` ra ngoài qua 1 callback `emit func(delta string)` ngay khi nhận được (real-time), đồng thời tự accumulate full content string để còn lưu DB sau khi xong. Khi `Done` với `ToolCalls` rỗng → đó là round trả lời cuối, kết thúc. Khi `Done` với `ToolCalls` khác rỗng → chạy tool (giống Phase 1), nối message, sang round tiếp.

### 2.4. Fallback khi đang stream

- Lỗi xảy ra **trước khi có bất kỳ ContentDelta nào được emit ra client** (network fail, 429, 5xx ngay từ đầu) → coi như round 1 chưa thực sự bắt đầu trả lời, retryable y hệt Phase 1: thử model tiếp theo trong `fallback_priority`, client không biết gì (có thể trễ thêm vài trăm ms).
- Lỗi xảy ra **sau khi đã emit ít nhất 1 delta** cho client → KHÔNG fallback được nữa (khách đã thấy chữ ra rồi, đổi sang câu trả lời khác của model khác là trải nghiệm tệ hơn dừng lại) — gửi 1 SSE `event: error` rồi đóng stream, lưu lại những gì đã có vào DB (đánh dấu incomplete).

### 2.5. Wire format SSE gửi cho client

```
event: delta
data: {"content":"Máy lạnh"}

event: delta
data: {"content":" 1.5HP..."}

event: done
data: {"blocked":false}

```
Trường hợp bị guardrail chặn (vẫn chạy classifier non-stream như Phase 1, TRƯỚC khi mở stream) → mở stream, emit đúng 1 `event: delta` chứa câu từ chối, rồi `event: done` với `"blocked":true` — tái dùng cùng 1 khung SSE, FE không cần code nhánh riêng cho trường hợp bị chặn.

Lỗi giữa chừng: `event: error\ndata: {"message":"..."}\n\n` rồi đóng kết nối — không có HTTP status code nào gửi thêm được nữa vì header đã flush. Phần text đã kịp stream cho khách trước khi lỗi **vẫn được lưu vào `ai_messages`**, đánh dấu `incomplete = true` (migration `000002_message_incomplete_flag`) — không vứt bỏ, và không được đưa lại vào context của các lượt chat sau (`ListRecentMessages` lọc `NOT incomplete`) để tránh model tưởng đó là câu trả lời đầy đủ.

### 2.5.1. Sửa sau review (`/code-review`, trước khi merge)

- Timeout tầng HTTP client (`llmClientHTTPTimeout`) ban đầu để 30s trong khi timeout tổng của request (`chatTimeout`) nâng lên 60s cho streaming — client tự cắt stream ở giây 30 dù model vẫn đang trả lời bình thường. Sửa: `llmClientHTTPTimeout` = 120s (chỉ là lưới an toàn cho caller không tự set deadline như `cmd/sync-ai-pricing`; mọi request thật đều bị `ctx` của caller giới hạn trước, nên 120s không bao giờ là cái cắt ngang 1 câu trả lời hợp lệ).
- Lỗi giữa chừng ban đầu vứt bỏ toàn bộ phần đã stream thay vì lưu `incomplete` như mục 2.5 mô tả — đã sửa (xem migration + `partialOutcome`/`persistAssistantMessage` trong code).

### 2.6. Những cái KHÔNG đổi

- Rate limit, validate input, lấy visitor_id/user_id, guardrail classifier, persistence logic (lưu conversation/message + cost) — y hệt Phase 1, chỉ đổi chỗ "gọi model" từ `Chat` non-stream sang `ChatStream`.
- `/ai/providers`, `/ai/models` — không đổi.
- Fallback theo `fallback_priority`, điều kiện retry (429/timeout/5xx) — không đổi, chỉ áp dụng thêm ràng buộc 2.4 ở trên (chỉ fallback được nếu chưa emit gì).

## 3. Rủi ro & an toàn

- **Risk level**: Medium — thay đổi hẳn response contract của endpoint public đang chạy thật (nhưng FE chưa consume). `go build/vet` không bắt được lỗi logic streaming (khó unit test HTTP streaming đầy đủ) — cần test tay bằng `curl -N` để xem chữ ra dần thật.
- Context hủy giữa chừng (khách đóng tab) phải dừng gọi model ngay — dùng `r.Context()` xuyên suốt, không tạo `context.Background()` con nào tách rời.
- `http.ResponseWriter` phải là `http.Flusher` — chi router mặc định hỗ trợ, nhưng phải flush sau mỗi write, không quên.

## 4. Kế hoạch

1. Nhánh `feature/ai-chat-phase2-streaming` (đã tạo).
2. Implement trên nhánh này — không dồn Phase 3 vào cùng nhánh/PR.
3. Test tay bằng `curl -N` xác nhận chữ ra dần thật (không phải toàn bộ response về cùng lúc).
4. `/code-review` (Standards + Spec) trước khi merge.
5. Merge `main` sau khi review pass.
6. Phase 3 (report chi phí + đọc lịch sử hội thoại) là RFC/nhánh riêng, làm sau khi Phase 2 merge xong.
