# RFC: Redis cho AI chat — rate limit dùng chung + cache classifier/search_products

- **Status**: Completed — merged `2488bd5` (Redis-backed rate limiting + AI response cache).
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: Redis mới hoàn toàn trong hạ tầng elc-go — chỉ áp dụng cho `internal/ai`, không đụng rate limiter của module khác (auth, inquiry, review, event, wishlist vẫn dùng in-memory như cũ)

## 1. Bối cảnh

User xác nhận 2 mục đích: (1) rate limit chat chuyển từ in-memory (RAM 1 container, mất khi restart, không đồng bộ nếu sau này chạy nhiều instance) sang Redis; (2) cache kết quả `search_products`/classifier để đỡ tốn — "hiệu quả, hiệu năng, tiết kiệm cost".

**Cần nói rõ trước khi làm** (để không hiểu lầm): 2 cái này tiết kiệm cost theo 2 cách khác nhau:
- Cache **classifier**: tiết kiệm cost thật — 1 lần gọi LLM ít tốn hơn cho câu hỏi giống/gần giống đã hỏi trước đó (vd nhiều khách cùng hỏi "còn hàng không", "giá bao nhiêu" theo cách gần giống nhau).
- Cache **search_products**: **không** tiết kiệm cost LLM (đây là query DB, không phải gọi AI) — chỉ giảm tải DB + giảm độ trễ. Vẫn làm vì user yêu cầu, nhưng giá trị chính là hiệu năng, không phải tiền.

## 2. Thiết kế

### 2.1. Hạ tầng Redis (mới hoàn toàn)

- `docker-compose.yml` thêm service `redis` (image `redis:7-alpine`, không publish port ra ngoài — cùng lý do postgres không publish, chỉ elc-go reach qua network nội bộ). Không cần volume — data rate-limit/cache mất khi restart là chấp nhận được (ephemeral by design).
- `go.mod` thêm `github.com/redis/go-redis/v9`.
- `.env` thêm `REDIS_URL` — **optional**: nếu không set, server chạy bình thường, rate limit fallback về in-memory như hiện tại, cache tắt hẳn (không lỗi, không crash) — đúng pattern "optional, degrade gracefully" đã dùng cho SMTP/AI provider ở Phase 1.

### 2.2. Rate limit — chỉ AI, không đụng module khác

Thêm interface hẹp trong `internal/platform/ratelimit` (package đã có sẵn, không tạo package mới):
```go
type RateLimiter interface {
    Allow(key string) bool // giữ đúng chữ ký hiện có của *Limiter, không đổi gì ở các module đang dùng
}
```
`*ratelimit.Limiter` (in-memory, hiện có) tự động thỏa interface này, không sửa gì. Thêm `RedisLimiter` (file mới `redis_limiter.go`) cùng thỏa interface, dùng **1 Lua script atomic** (`INCR` + `PEXPIRE` chạy chung 1 lệnh trên server Redis, không phải 2 round-trip riêng — sửa sau khi `/code-review` bắt được: bản đầu dùng 2 lệnh tách rời, nếu `EXPIRE` fail đúng lúc `count==1` thì key không bao giờ hết hạn, chặn vĩnh viễn 1 key thay vì đúng 1 window) — cùng ngữ nghĩa "tối đa N lần/window/key" như bản in-memory. Lỗi kết nối Redis khi gọi `Allow` → **fail open** (cho qua) — rate limit là lớp chống spam, không phải business logic đúng/sai; Redis chết không được kéo sập luôn tính năng chat.

`AIHandler.chatLimiter` đổi kiểu từ `*ratelimit.Limiter` sang interface `ratelimit.RateLimiter` — composition root (`main.go`) chọn implementation nào theo `REDIS_URL` có set hay không.

### 2.3. Cache — port `domain.Cache`, 2 nơi dùng

```go
type Cache interface {
    Get(ctx context.Context, key string) (value string, ok bool, err error)
    Set(ctx context.Context, key, value string, ttl time.Duration) error
}
```
`infrastructure.RedisCache` implement bằng `go-redis`. Cache là **tham số optional** (`nil` = tắt cache, không lỗi) truyền vào `ClassifyMessage` và `NewProductSearchTool` — cùng nguyên tắc graceful-degrade.

- **Classifier**: key = `ai:classify:{sha256(message đã trim+lowercase)}`, TTL **1 giờ** (kết quả phân loại "có liên quan ELC không" không đổi theo thời gian cho cùng 1 câu chữ).
- **search_products**: key = `ai:search:{sha256(argumentsJSON)}`, TTL **5 phút** (ngắn — giá/tồn kho có thể đổi, không muốn cache giữ data cũ quá lâu, nhất là sau khi vừa làm xong RFC chống data sai ở phase trước).

### 2.4. Không đổi

- Guardrail logic, fallback logic, streaming, persistence, report — nguyên xi. Cache chỉ chèn thêm 1 bước "check trước, lưu sau" quanh đúng chỗ đang gọi LLM/DB, không đổi luồng.
- Rate limit của auth/inquiry/review/event/wishlist — vẫn `ratelimit.New(...)` in-memory như cũ, không sửa.

## 3. Rủi ro & an toàn

- **Risk level**: Medium — thêm dependency hạ tầng mới (Redis), nhưng thiết kế fail-open + optional nên không có đường nào Redis chết làm sập site.
- Cache classifier sai 1 lần (vd model classify nhầm 1 câu biên giới on-topic/off-topic) sẽ **lặp lại đúng cái sai đó trong 1 giờ** cho cùng câu chữ y hệt — chấp nhận được vì TTL ngắn, và câu hỏi y hệt từng ký tự từ nhiều khách khác nhau không phổ biến.
- `go build/vet/test` xanh; test tay: gọi `/ai/chat` 2 lần cùng câu hỏi y hệt trong vòng 1 giờ, xác nhận lần 2 không tốn thêm lệnh gọi classifier (kiểm tra qua log hoặc `cost_usd` không tăng thêm phần classifier).

## 4. Kế hoạch

1. Nhánh `feature/ai-chat-redis` (đã tạo).
2. Implement.
3. `/code-review`.
4. Merge `main`, deploy — **lưu ý**: cần thêm `REDIS_URL` vào GitHub Secrets + `deploy.yml` (giống bài học `AI_SECRETS_ENCRYPTION_KEY` bị quên ở Phase 1 — lần này làm ngay trong cùng PR, không tách riêng).
