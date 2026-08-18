# RFC: AI chat — bổ sung specs/highlights thật vào tool search_products

- **Status**: Draft
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: Chỉ `internal/ai/infrastructure/product_search_tool.go` — không đụng `/ai/chat`, streaming, fallback, guardrail, report

## 1. Bối cảnh

User hỏi trực tiếp khi test: tool `search_products` (Phase 1) chỉ trả `name/slug/brand/category/price/stock_status` — không có thông số kỹ thuật (specs) hay highlights, dù `internal/product` đã lưu đầy đủ (`AttributeValueRef` qua `GetByIDsWithAttributeValues`, `Highlights()` trên `Product`). Hệ quả: khi khách hỏi tính năng cụ thể ("có chế độ ngủ êm không", "công suất bao nhiêu BTU"), AI **không tra được DB thật cho phần này** — trả lời bằng suy đoán từ kiến thức chung, có thể sai/không khớp với sản phẩm thật trên web. Đây là rủi ro grounding thật, không phải giả thuyết — cần vá trước khi đẩy traffic thật qua chat.

## 2. Thiết kế

`search_products`'s executor hiện gọi 1 lần `productRepo.GetAll(ctx, filter)`. Thêm bước 2: với đúng các product ID vừa match (tối đa `searchResultLimit` = 5, không phải toàn bộ catalog), gọi `productRepo.GetByIDsWithAttributeValues(ctx, ids)` — method đã có sẵn (dùng bởi tính năng Compare Products), không viết SQL mới.

`productSummary` (JSON trả cho model) thêm 2 field:
```json
{
  "name": "...", "slug": "...", "brand": "...", "category": "...",
  "price_from": 12500000, "price_to": 12500000, "stock_status": "in_stock",
  "highlights": ["Ngủ êm dB thấp", "Lọc bụi mịn PM2.5"],
  "specs": [
    { "label": "Công suất làm lạnh", "value": "9000 BTU/h" },
    { "label": "Loại máy", "value": "1 chiều" }
  ]
}
```
`specs` map từ `AttributeValueRef`: `Name` (+ `Unit` nếu có) làm `label`; `value` render theo `DataType` — number ghép `Unit`, boolean → "Có"/"Không", text → nguyên văn, multiselect (`ValueOptions`) → nối bằng dấu phẩy. Bỏ qua attribute nào rỗng cả 4 field giá trị (không nhồi noise vào prompt).

System prompt (Phase 1, `SystemPrompt` trong `application/send_chat_message.go`) **không đổi nội dung**, chỉ hưởng lợi gián tiếp: dặn dò "không tự bịa" đã có sẵn từ Phase 1, giờ có data thật để tuân theo dặn dò đó thay vì buộc phải suy đoán vì thiếu data.

**Không làm trong RFC này** (out of scope, để sau nếu cần):
- Kiểm tra chéo số AI viết ra với số tool trả về (rủi ro AI gõ nhầm số khi tường thuật — vấn đề khác, khó hơn).
- Cơ chế phát hiện data sai do nhân viên nhập nhầm (thuộc quy trình duyệt sản phẩm sẵn có draft→proposed→published, không phải việc của AI).

## 3. Rủi ro & an toàn

- **Risk level**: Low — chỉ thêm field vào response 1 tool nội bộ, không đổi schema DB, không đổi hành vi endpoint nào khác.
- Thêm 1 query DB/lượt chat có gọi tool (bounded ≤5 sản phẩm) — chấp nhận được, không phải N+1 trên toàn catalog.
- Payload tool-result gửi cho model tăng kích thước (specs text) → tăng nhẹ input token, do đó tăng nhẹ `cost_usd` mỗi lượt có tool-call — đánh đổi hợp lý cho việc giảm hallucination.
- `go build/vet/test` xanh; test tay: hỏi 1 câu về tính năng cụ thể của sản phẩm có specs thật trong dev DB, xác nhận AI trích đúng số/chữ khớp `specs` trả về (không đối chiếu tự động, đọc bằng mắt).

## 4. Kế hoạch

1. Nhánh `feature/ai-chat-product-specs-grounding` (đã tạo).
2. Implement.
3. `/code-review`.
4. Merge `main`, deploy, verify.
