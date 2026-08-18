# RFC: Backend code cleanup (dead code, DB race, error handling)

- **Status**: Draft
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)

## 1. Bối cảnh

Đã chạy audit toàn diện backend `elc-go` (30 domain trong `internal/` + toàn bộ `cmd/*`, build sạch tại thời điểm audit). Phương pháp:

- **Dead code**: xác nhận qua `deadcode` (call graph thực từ entrypoint), `staticcheck` U1000, và `gopls references` cho từng candidate — không chỉ grep, để tránh false positive.
- **Lệch logic DB**: đối chiếu migrations SQL của từng domain với mọi SELECT/INSERT/UPDATE và struct scan trong code.
- **Error handling**: rà theo baseline convention của repo (đã xác nhận qua domain `product`: wrap lỗi `fmt.Errorf("<domain> repository <method>: %w", err)`, transaction luôn `Begin`+`defer Rollback`+`Commit`, `rows.Close()` defer ngay sau Query).

Không có finding critical nào. RFC này gom toàn bộ findings (medium + low/cosmetic) thành một danh sách để làm sạch dần, không phải phản ứng với sự cố đang xảy ra.

## 2. Tổng quan vấn đề

| # | Mức độ | Domain | Vấn đề | Loại |
|---|---|---|---|---|
| 1 | Medium | auth | `SMTPEmailSender.send` không có timeout/context | Error handling |
| 2 | Medium | category (+group/news/project-type/project) | Check-then-act race khi resurrect slug soft-deleted | DB logic |
| 3 | Medium | product/platform/inquiry/event | 5-6 item dead code đã xác nhận | Dead code |
| 4 | Low | shippingzone | Insert từng dòng thay vì batch multi-row | DB logic (nhất quán) |
| 5 | Low | service | Cột `sale_price` write-only, không bao giờ đọc lại | DB logic |
| 6 | Low | page | TOCTOU nhẹ: Update trả 500 thay vì 404 khi row bị xóa giữa chừng | Error handling |
| 7 | Low | upload | Nuốt lỗi `crypto/rand.Read` | Error handling |
| 8 | Low | attribute | `Detach` không check tồn tại trước khi xóa (khác `Attach`) | Error handling |
| 9 | Low | review | `entityID` không validate UUID trước khi vào SQL | Error handling |
| 10 | Low | platform/ratelimit | Map `windows` không bao giờ prune, memory tăng dần | Error handling |
| 11 | Cosmetic | page/system-page/settings | Log message lệch convention wrap lỗi | Error handling |
| 12 | Cosmetic | brand | Doc-comment sai (nói có cascade nhưng thực tế không) | Docs |

Chi tiết file:line từng mục ở phần 3.

## 3. Đề xuất xử lý

### 3a. Dead code removal

Xóa các item đã xác nhận 0 reference thực trong toàn repo (chỉ còn ở comment/test):

- `internal/platform/httpserver/auth_middleware.go:59` — `RequireRole(roles ...string)`
- `internal/platform/media/image_asset.go:67` — `URLs(images []ImageAsset) []string`
- `internal/product/application/vietnamese_normalize.go:24,16` — `normalizeVietnamese`, `diacriticStripper`
- `internal/product/domain/variant.go:199,19` — `NewProductVariant`, `validVariantStockStatus` (chỉ dùng trong `variant_test.go`; production dùng `RehydrateProductVariant`)
- `internal/inquiry/presentation/dto.go:74` — `zaloWebhookEvent` struct (tính năng Zalo OA đã gỡ)
- `internal/event/domain/types.go:93` — `RehydrateEvent` (chỉ dùng trong `fake_repository_test.go`) — cân nhắc riêng, có thể giữ nếu cần cho fake test repo

Với `NewProductVariant`/`validVariantStockStatus`: nếu xóa, sửa `variant_test.go` để test qua `RehydrateProductVariant` hoặc constructor thật đang dùng production, không xóa test coverage.

### 3b. DB race condition — resurrect soft-deleted slug

`internal/category/infrastructure/postgres_repository.go:152-200` (`Create`) và pattern tương tự ở group/news/project-type/project: `SELECT ... WHERE slug = $1 AND deleted_at IS NOT NULL` chạy ngoài transaction rồi mới quyết định UPDATE hay INSERT.

Đề xuất: gộp SELECT + UPDATE/INSERT vào cùng một transaction với `SELECT ... FOR UPDATE`, hoặc dùng `INSERT ... ON CONFLICT (slug) WHERE deleted_at IS NULL DO UPDATE ...` nếu unique index hỗ trợ. Áp dụng đồng nhất cho cả 5 domain bị ảnh hưởng (category, group, news, project-type, project) vì cùng pattern.

### 3c. Error handling fixes

- **SMTP timeout** (`internal/auth/infrastructure/email_sender.go:68`): thread `context` từ request vào, dùng `net.Dialer` với `DialContext` + deadline thay vì `smtp.SendMail` trần, hoặc set timeout qua goroutine + `context.WithTimeout` bọc ngoài.
- **`crypto/rand.Read` bị nuốt lỗi** (`internal/upload/presentation/handler.go:134`): trả lỗi 500 rõ ràng thay vì `_, _ = rand.Read(...)`.
- **`Detach` thiếu check tồn tại** (`internal/attribute/application/category_association.go:26-28`): thêm `GetByID` check giống `Attach` nếu muốn nhất quán 404 thay vì 204 im lặng — cân nhắc: DELETE idempotent là hành vi REST hợp lệ, có thể giữ nguyên nếu không gây nhầm lẫn thực tế.
- **UUID validate ở review** (`internal/review/presentation/handler.go:65,127`): validate `entityID` là UUID hợp lệ trước khi vào query, trả 400 thay vì để Postgres lỗi 500.
- **Rate-limit map không prune** (`internal/platform/ratelimit/ratelimit.go:21-53`): thêm goroutine dọn định kỳ hoặc lazy-eviction khi window hết hạn.
- **page Update TOCTOU** (`internal/page/infrastructure/postgres_repository.go:173-192`): map `pgx.ErrNoRows` thành `apperr.NewNotFoundError` giống các domain khác.

### 3d. Cosmetic cleanup

- Đồng bộ log message wrap lỗi ở `page`, `system-page`, `settings` theo convention `"<domain> repository <method>: %w"`.
- Sửa doc-comment sai ở `internal/brand/application/delete_brand.go:9-10` (bỏ câu nói có cascade).

### Không làm trong RFC này (out of scope)

- `shippingzone` batch-insert và `service.sale_price` write-only: không phải bug, chỉ là inconsistency nhẹ — để riêng, không ưu tiên.
- Không thêm tính năng, không đổi kiến trúc, không refactor ngoài phạm vi finding đã liệt kê.

## 4. Kế hoạch triển khai

Thứ tự ưu tiên (rủi ro thấp → cao, giá trị cao → thấp):

1. Dead code removal (3a) — rủi ro thấp nhất, verify bằng `go build ./...` + `go test ./...`
2. Cosmetic cleanup (3d) — không đổi behavior
3. Error handling fixes (3c) — mỗi fix là 1 commit riêng, test theo domain
4. DB race condition fix (3b) — rủi ro cao nhất vì đổi transaction logic ở 5 domain, cần review kỹ + test race scenario nếu khả thi

Verify sau mỗi nhóm: `go build ./...`, `go vet ./...`, chạy test suite hiện có của domain bị đổi. Không có staging/CI riêng cho race condition test — verify bằng review logic thủ công + test đơn vị insert 2 lần liên tiếp cùng slug.

## 5. Rủi ro & rollback

- Xóa dead code sai (còn dùng qua reflection/dynamic dispatch không thấy trong static analysis): rủi ro thấp vì codebase không dùng reflection cho các case này, đã verify qua `gopls references`.
- Đổi transaction logic ở 3b có thể ảnh hưởng hành vi hiện tại nếu có edge case chưa lường trước — mỗi domain sửa riêng 1 commit để dễ revert độc lập.
- Không đổi schema DB, không cần migration mới cho toàn bộ RFC này.

## 6. Không nằm trong scope audit gốc

Theo báo cáo audit, dữ liệu seed dạng INSERT VALUES thuần túy (vd shippingzone ~3350 dòng phường/xã) không được đọc chi tiết vì không phải logic — không liên quan RFC này.
