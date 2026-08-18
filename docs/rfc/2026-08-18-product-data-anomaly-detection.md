# RFC: Phát hiện data khả nghi (giá ≤ 0, spec lệch bất thường) — tự động, không nhập tay

- **Status**: Draft
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: `internal/product` (validate giá + cột đánh dấu spec bất thường), `cmd/detect-attribute-anomalies` (mới), `internal/ai/infrastructure/product_search_tool.go` (dùng kết quả để không trích dẫn data khả nghi cho khách)

## 1. Bối cảnh

Trong lúc test AI chat, phát hiện: nếu nhân viên nhập sai giá (0đ, âm) hoặc spec (số vô lý), AI sẽ trung thành trích dẫn đúng cái sai đó cho khách — không có lớp nào bắt lỗi này (đã audit: `cmd/audit-attribute-values` chỉ là script chạy tay 1 lần cho việc khác — kiểm tra tính nhất quán sau migrate specs cũ, không phải cơ chế vận hành liên tục, và không đụng tới giá).

User ban đầu đề xuất admin tự khai báo khoảng hợp lệ (min/max) cho từng thuộc tính — **bị bác bỏ**: quá nhiều thuộc tính (hàng chục), bắt nhập tay không thực tế. Chuyển sang **tự động thống kê từ chính data đã có**, không cần ai khai báo gì.

## 2. Thiết kế

### 2.1. Giá ≤ 0 — validate ngay lúc nhập, quy tắc cứng

`ProductVariantInput.OriginalPrice`/`SalePrice` hiện **không có validation nào** (đã confirm bằng grep — trước giờ chưa từng chặn). Thêm 1 bước validate trong `application.CreateProduct`/`UpdateProduct`: `OriginalPrice <= 0` → lỗi; `SalePrice` (nếu có) `<= 0` → lỗi. Không kiểm tra `SalePrice > OriginalPrice` hay gì khác — ngoài phạm vi yêu cầu, không tự thêm.

### 2.2. Spec lệch bất thường — tự động thống kê, không nhập tay

Với mỗi `attribute_definition` kiểu `number` (vd "Công suất làm lạnh"), gom **toàn bộ giá trị đang có** của thuộc tính đó trên catalog, tính bằng phương pháp IQR (interquartile range — chuẩn thống kê, không giả định phân phối chuẩn, chịu được lệch/outlier tốt hơn mean±stddev):

```
Q1, Q3 = phân vị 25%, 75% của tập giá trị
IQR = Q3 - Q1
Ngưỡng dưới = Q1 - 1.5 * IQR
Ngưỡng trên = Q3 + 1.5 * IQR
Giá trị nằm ngoài [ngưỡng dưới, ngưỡng trên] → đánh dấu bất thường
```

**Ngưỡng mẫu tối thiểu**: thuộc tính nào có **dưới 10 sản phẩm** ghi nhận giá trị → **bỏ qua, không chấm** — quá ít data để thống kê có ý nghĩa, tránh báo sai vì mẫu nhỏ.

Chạy bằng 1 CLI mới `cmd/detect-attribute-anomalies` (raw SQL trực tiếp qua pool, cùng phong cách `cmd/audit-attribute-values` — không qua `ProductRepository` interface vì cần quét toàn catalog theo từng thuộc tính, không phải theo từng sản phẩm). Mỗi lần chạy: tính lại từ đầu, `UPDATE product_attribute_values SET flagged_anomaly = true/false` cho **toàn bộ** row của thuộc tính đó (không cộng dồn — luôn phản ánh đúng trạng thái data hiện tại, sửa data thật thì cờ tự gỡ ở lần chạy sau).

Migration mới: `product_attribute_values` thêm cột `flagged_anomaly BOOLEAN NOT NULL DEFAULT false`.

Lịch chạy: GitHub Actions cron, theo đúng mẫu `.github/workflows/sync-ai-pricing.yml` (SSH vào VPS, `docker run golang:1.26-alpine go run ./cmd/detect-attribute-anomalies`) — 1 lần/ngày.

### 2.3. AI không trích dẫn data khả nghi

`internal/ai/infrastructure/product_search_tool.go`:
- `AttributeValueRef` (domain, `internal/product`) thêm field `FlaggedAnomaly bool`, lấy từ cột mới qua đúng join sẵn có (không thêm query) — `renderSpecs` **bỏ qua hoàn toàn** attribute nào `FlaggedAnomaly = true` (không đưa vào JSON gửi cho model — model không thấy thì không trích dẫn được, đơn giản hơn là cố dặn model "đừng tin cái này").
- Giá: `PriceFrom`/`PriceTo` nếu `<= 0` (data cũ trước khi có validate 2.1, hoặc lỗi tính toán denormalize) → set `nil`, JSON `omitempty` tự bỏ field — model đã được dặn "không tìm thấy thì nói thật, đừng đoán" (system prompt Phase 1, không đổi) nên tự xử lý đúng khi thiếu giá.

## 3. Rủi ro & an toàn

- **Risk level**: Medium — có migration mới (nhưng chỉ thêm 1 cột boolean, an toàn), có thêm validate chặn ghi (đổi hành vi API tạo/sửa sản phẩm — cần confirm không có luồng nào đang cố tình tạo sản phẩm giá 0 hợp lệ, vd hàng khuyến mãi tặng kèm — **giả định**: sản phẩm bán độc lập trong catalog này luôn phải có giá, hàng tặng kèm không phải là 1 "product" riêng trong hệ thống này. Nếu sai giả định, cần biết trước khi merge).
- IQR tính trên **toàn bộ** giá trị hiện có mỗi lần chạy — không có race condition vì luôn ghi đè lại từ đầu, idempotent.
- `go build/vet/test` xanh; test tay: chạy `cmd/detect-attribute-anomalies` trên dev DB, xác nhận cột `flagged_anomaly` được set đúng cho vài giá trị test cố tình nhập lệch; xác nhận `search_products` không còn trả spec đã bị đánh cờ.

## 4. Kế hoạch

1. Nhánh `feature/product-data-anomaly-detection` (đã tạo).
2. Implement.
3. `/code-review`.
4. Merge `main`, deploy — **lưu ý**: sau merge cần chạy `cmd/detect-attribute-anomalies` lần đầu trên prod DB (không tự chạy khi deploy, giống `cmd/sync-ai-pricing`) để có cờ ngay, không đợi tới lần cron đầu tiên.
