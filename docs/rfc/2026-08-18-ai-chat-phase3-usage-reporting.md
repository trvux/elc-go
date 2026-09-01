# RFC: AI chat Phase 3 — báo cáo chi phí/usage + đọc lịch sử hội thoại

- **Status**: Completed — merged `c76af16` (Phase 3: usage/cost report + conversation history read API).
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: Chỉ 2 nhóm API đọc (report + lịch sử) — không đụng `/ai/chat`, `/ai/providers`, `/ai/models`

## 1. Bối cảnh

Phase 1 đã lưu đầy đủ `cost_usd`/token usage/`products_shown` vào `ai_messages`, Phase 2 thêm `incomplete`. Nhưng chưa có API nào đọc lại data này — muốn xem đã tốn bao nhiêu tiền, hay xem lại 1 cuộc hội thoại cụ thể, phải query SQL tay. Đây là yêu cầu gốc #9 (cost tracking để đánh giá ROI) và #8 (log hội thoại để phát triển ads/marketing) — data đã có, chỉ thiếu cách đọc ra qua admin panel.

Scope: chỉ backend read API (giống Phase 1's `/ai/providers`, `/ai/models` — CRUD/list, permission-gated). Không làm UI (bên `elc-tem`, sau khi Phase 3 merge).

## 2. Thiết kế

### 2.1. `GET /ai/usage` — báo cáo chi phí/usage tổng hợp

Query params: `from`, `to` (RFC3339 date, mặc định 30 ngày gần nhất nếu bỏ trống), `groupBy` = `day` | `provider` | `model` (mặc định `day`).

Response:
```json
{
  "groupBy": "day",
  "rows": [
    {
      "key": "2026-08-18",
      "messageCount": 42,
      "blockedCount": 5,
      "inputTokens": 12000,
      "outputTokens": 8000,
      "costUsd": 0.0234
    }
  ]
}
```
`key` là ngày (groupBy=day), tên provider (groupBy=provider), hoặc `"{providerName}/{modelName}"` (groupBy=model). 3 query SQL riêng theo từng `groupBy` (không viết 1 query tổng quát động cột GROUP BY bằng string concat — tránh SQL injection qua param, và mỗi query rõ ràng dễ đọc hơn 1 query "thông minh").

`blockedCount` đếm từ `ai_messages.blocked_reason IS NOT NULL` trong cùng khoảng thời gian — để thấy guardrail chặn bao nhiêu % traffic (liên quan trực tiếp tới mục tiêu tiết kiệm chi phí đã bàn ở Phase 1).

### 2.2. `GET /ai/conversations` — danh sách hội thoại

Query params: `from`, `to`, `limit`, `offset` (mặc định 20/0 — tự chọn, không phải copy từ module nào: kiểm tra lại `internal/inquiry` sau khi review thì module đó thực ra không tự implement limit/offset, `GetAll` không áp LIMIT/OFFSET ở tầng SQL, nên "giống inquiry" ở bản RFC đầu là trích dẫn sai).

Response — bọc trong object có `total`/`limit`/`offset` để FE phân trang được, không phải mảng trần như bản nháp đầu:
```json
{
  "conversations": [
    { "id": "...", "visitorId": "...", "userId": "...", "messageCount": 6, "totalCostUsd": 0.0012, "createdAt": "...", "updatedAt": "..." }
  ],
  "total": 128, "limit": 20, "offset": 0
}
```

### 2.3. `GET /ai/conversations/{id}/messages` — chi tiết 1 hội thoại

Trả **toàn bộ** message (khác `ListRecentMessages` nội bộ chỉ lấy user/assistant không-blocked/không-incomplete để feed lại cho model) — bao gồm cả message bị chặn (`blockedReason`) và bị cắt (`incomplete`), để admin xem đúng những gì đã thực sự xảy ra.

### 2.4. Quyền hạn

Cả 3 endpoint dưới `RequireAuth` + `RequirePermission(authdomain.CanManageSettings)` — y hệt `/ai/providers`, `/ai/models`, không thêm permission mới (đúng lý do Phase 1 đã nêu).

### 2.5. Repository

`ConversationRepository` thêm 3 method:
```go
ListConversations(ctx, filter ConversationFilter) ([]*ConversationSummary, int, error) // int = total count, cho phân trang
GetMessages(ctx, conversationID string) ([]*ConversationMessage, error)
GetUsageReport(ctx, from, to time.Time, groupBy UsageGroupBy) ([]UsageReportRow, error)
```
`ConversationSummary` là type mới (view riêng cho list — có `MessageCount`/`TotalCostUSD` tính sẵn qua SQL aggregate, không phải `Conversation` gắn thêm field ngẫu nhiên).

## 3. Rủi ro & an toàn

- **Risk level**: Low — chỉ thêm endpoint đọc (`GET`), không đổi hành vi `/ai/chat` hay bất kữ write path nào đang chạy thật. Không có migration mới (dùng đúng cột đã có từ Phase 1/2).
- 3 query SQL riêng theo `groupBy` (không nối chuỗi động) — loại trừ rủi ro SQL injection qua param `groupBy`.
- `go build/vet` + test tay qua `curl` với token admin thật.

## 4. Kế hoạch

1. Nhánh `feature/ai-chat-phase3-reporting` (đã tạo).
2. Implement.
3. `/code-review` (Standards + Spec).
4. Merge `main`.
5. Sau khi merge: gửi prompt cập nhật cho `elc-tem` để build màn hình report/lịch sử trong admin panel (nối tiếp 2 prompt trước đó).
