# RFC: AI customer chat Phase 1 — provider/model quản lý qua admin, fallback, guardrail, lưu hội thoại

- **Status**: Draft
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: `internal/ai` — build tiếp trên v1 (đã merge nhánh này ở commit trước), chưa đụng streaming/report

## 1. Bối cảnh

v1 (`internal/ai`, `POST /ai/chat`) là bản MVP tối giản: 1 provider DeepSeek hardcode qua env, stateless (không lưu hội thoại), không guardrail ngoài system prompt. Yêu cầu thực tế lớn hơn nhiều:

1. Provider/model/API key phải quản lý được từ admin panel (không sửa `.env` + redeploy mỗi lần đổi).
2. Tự động fallback sang provider khác khi 1 API die.
3. Lưu toàn bộ hội thoại vào DB — vừa để tra soát, vừa làm data cho ads/marketing sau này.
4. Guardrail: AI chỉ nói về ELC, không nhắc đối thủ, từ chối yêu cầu lạc đề (bài tập, việc cá nhân...).
5. Track chi phí (token × giá thật của provider) để đánh giá ROI.
6. Nguồn giá phải lấy từ docs chính thức của provider tại thời điểm build (đã fetch lại `https://api-docs.deepseek.com/quick_start/pricing/` ngày 2026-08-18: `deepseek-v4-flash`/`deepseek-v4-pro`, giá peak/off-peak + cache-hit/miss — mới hơn kiến thức huấn luyện của assistant, **không** dùng số nhớ sẵn).

Do khối lượng lớn, chia 3 giai đoạn (đã thống nhất với user). **RFC này chỉ cho Phase 1**: nền tảng data model + admin CRUD provider/model + fallback + guardrail (dạng classifier rẻ, chạy trước) + lưu hội thoại — endpoint `/ai/chat` vẫn non-streaming như v1. Streaming (Phase 2) và report chi phí tổng hợp (Phase 3) để sau.

Quyết định thay user (có lý do, xem chi tiết ở kế hoạch đầy đủ trong `/Users/tranvux/.claude/plans/squishy-wobbling-wirth.md`):
- **Danh tính hội thoại**: cookie `visitor_id` có sẵn (`httpserver.EnsureVisitorID`, đang dùng cho wishlist/recently-viewed) làm khóa bắt buộc, cộng `user_id` optional khi khách đang đăng nhập — bắt buộc login sẽ giảm volume hội thoại, ngược lại mục tiêu thu data cho ads.
- **Guardrail**: 1 lệnh gọi classifier riêng, rẻ, chạy **trước** completion chính — tin nhắn lạc đề bị chặn ở bước rẻ này, không tốn tiền completion + tool-call đầy đủ, nên guardrail này **giảm** chi phí trung bình/request chứ không chỉ cộng thêm.

## 2. Data model mới (`internal/ai/migrations`)

4 bảng: `ai_providers` (slug, base_url, `pricing_doc_url` — trang giá chính thức để cron tự sync, api_key mã hóa AES-256-GCM, is_active), `ai_models` (provider_id FK, model_name, role `chat`|`classifier`, `pricing` JSONB tự mô tả, fallback_priority, is_default/is_active), `ai_conversations` (visitor_id bắt buộc, user_id optional FK -> `users(id)`), `ai_messages` (role, content, blocked_reason, provider_id/model_id, token usage, cost_usd, products_shown JSONB cho phân tích ads sau này).

**Pricing JSON — sửa sau khi đọc kỹ docs DeepSeek** (bản đầu tiên lưu `peak`/`off_peak` như 2 số độc lập là sai: docs nói rõ *"Off-peak rates are half of the peak rates"* — 1 công thức, không phải 2 số cần tự đồng bộ tay mỗi lần đổi giá):

```json
{
  "currency": "USD", "per_million_tokens": true,
  "input_cache_hit_peak": 0.044,
  "input_cache_miss_peak": 1.32,
  "output_peak": 3.96,
  "off_peak_multiplier": 0.5,
  "peak_windows_utc": [{"start_hour": 1, "end_hour": 4}, {"start_hour": 6, "end_hour": 10}]
}
```

`Pricing.Cost()` tính giá peak trước, nhân `off_peak_multiplier` nếu giờ request rơi ngoài `peak_windows_utc`. Provider không phân biệt peak/off-peak thì để `off_peak_multiplier: 1.0`, bỏ `peak_windows_utc`. Provider không phân biệt cache hit/miss thì bỏ `input_cache_hit_peak`, mọi input tính theo `input_cache_miss_peak`.

**Fallback trigger** — cũng lấy từ docs chính thức thay vì đoán: DeepSeek rate-limit theo **concurrency** (không phải request/phút), trả **HTTP 429** khi vượt giới hạn (500 concurrent cho `-pro`, 2500 cho `-flash`, tính theo account chứ không theo riêng từng API key). `SendChatMessage` coi 429, timeout, và 5xx là tín hiệu chuyển sang model kế tiếp theo `fallback_priority` — không coi 4xx khác (vd 400 do lỗi payload) là lý do fallback, vì thử lại provider khác cũng lỗi y hệt.

## 3. Thay đổi chính

- `internal/ai/domain`: `Pricing.Cost(usage, at time.Time) float64` (logic thuần, test được không cần DB), `ModelConfig` (view đã giải mã key, dùng lúc runtime), `ProviderRepository`/`ModelRepository`/`ConversationRepository` — theo đúng pattern `Update(input)` một hàm (không per-field UpdateX(), module mới nên làm chuẩn từ đầu, không nợ kỹ thuật như branch trước khi refactor).
- `internal/ai/infrastructure`: `secret_crypto.go` (AES-GCM, key từ `AI_SECRETS_ENCRYPTION_KEY`, không phải global — inject qua constructor), postgres repos cho provider/model/conversation, parse thêm `usage` từ response OpenAI-compatible.
- `internal/ai/application`: `classify_message.go` (guardrail, fail-open nếu chưa cấu hình classifier — không chặn hết chat vì thiếu config), `send_chat_message.go` viết lại: guardrail gate → vòng fallback qua các model `role=chat` theo `fallback_priority` → lưu conversation/message + cost.
- `internal/ai/presentation`: CRUD admin cho provider/model (permission `authdomain.CanManageSettings`, tái dùng permission có sẵn của `internal/settings` — không thêm permission mới), API key chỉ ghi, không bao giờ trả về plaintext.
- `internal/platform/httpserver`: thêm `OptionalAuth` (như `RequireAuth` nhưng không fail nếu thiếu token) — dùng để gắn `user_id` optional vào hội thoại.
- `cmd/seed-ai-provider` (mới, theo mẫu `cmd/seed-admin`): seed provider/model đầu tiên từ env, để ngày deploy đầu tiên không phải thao tác tay qua admin API. Giá khởi tạo để trống/0 — cron sync (dưới) điền số thật ngay lần chạy đầu, không hardcode giá vào seed.
- `cmd/sync-ai-pricing` (mới): với mỗi `ai_providers.pricing_doc_url` đã set, fetch trang giá, đưa nội dung cho 1 model `role=classifier` (rẻ) trích xuất JSON đúng shape `pricing` ở trên, validate rồi `UPDATE ai_models.pricing` cho các `model_name` đã khớp sẵn trong DB. Model xuất hiện trong docs nhưng chưa có row trong `ai_models` thì chỉ log cảnh báo, **không tự tạo** — tránh tool tự ý đẩy model chưa duyệt vào fallback chain. Chạy định kỳ qua GitHub Actions cron (mẫu: `.github/workflows/migrate-images.yml`, scheduled workflow có sẵn gọi vào DB), ví dụ 1 lần/ngày.

## 4. Rủi ro & an toàn

- **Risk level**: Medium-High — có migration DB mới, có mã hóa secret, có thay đổi cách `/ai/chat` hoạt động (guardrail có thể chặn nhầm câu hỏi hợp lệ nếu prompt classifier viết dở).
- **An toàn**: `Pricing.Cost` là domain logic thuần — viết unit test độc lập DB. Guardrail fail-open khi thiếu config classifier (không tự khóa toàn bộ tính năng nếu quên setup). Rollback: `down.sql` cho từng migration, xóa route admin không ảnh hưởng `/ai/chat` (route cũ vẫn chạy nếu revert application layer).
- `go build ./... && go vet ./...` xanh trước mỗi commit; test thủ công theo mục Verification trong plan file (seed → chat on-topic → chat off-topic bị chặn không tốn completion → tắt provider chính, xác nhận fallback).

## 5. Kế hoạch

1. Nhánh `feature/ai-chat-assistant` (đã tạo, chứa sẵn v1).
2. Implement Phase 1 trên cùng nhánh này (không tách nhánh riêng — cùng 1 tính năng, review 1 lần cho cả v1+Phase1 hoặc chia nhỏ commit theo layer nếu diff quá lớn).
3. Code review qua `/code-review` trước khi merge.
4. Merge vào `main` sau khi build/vet xanh + review pass + verification thủ công ở mục 4 đạt.
5. Phase 2 (streaming) và Phase 3 (report chi phí) là RFC/nhánh riêng sau khi Phase 1 chạy ổn thực tế.
