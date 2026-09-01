# RFC: Gộp per-field UpdateX() của Branch domain thành 1 hàm Update

- **Status**: Completed — landed in `bfcd4b6` (branch domain refactored to single Update(input) method; see CLAUDE.md for the lazy-rollout note on other modules).
- **Date**: 2026-08-18
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)
- **Scope**: Pilot — chỉ `internal/branch`, chưa áp dụng module khác

## 1. Bối cảnh

Audit tổng quan `internal/` cho thấy các module CRUD nhỏ (branch, contact, tag, author, event...) bị áp cùng khuôn kiến trúc 5 lớp với module lớn (`product`, `auth`) dù độ phức tạp thấp hơn nhiều. Trong đó có 1 điểm dư thừa rõ ràng, không đụng tới encapsulation/Clean Architecture: domain entity có 1 hàm `UpdateX()` riêng cho từng field (13 hàm), trong khi thực tế chỉ có đúng 1 use case gọi chúng — `application.UpdateBranch`, luôn cập nhật cả form một lượt.

Mục tiêu người dùng: giữ nguyên Clean Architecture (domain/application/infrastructure/presentation tách lớp, private field + getter), giữ SOLID — chỉ giảm số lượng method trùng lặp không cần thiết trên domain entity.

`branch` được chọn làm pilot vì blast radius nhỏ, có sẵn test bao phủ (`domain/types_test.go`, `application/branch_test.go`), phù hợp để làm mẫu trước khi cân nhắc áp dụng cho ~15-20 module CRUD còn lại.

## 2. Blast radius (đã xác nhận bằng grep, không chỉ đoán)

Các hàm bị gộp — **chỉ** được gọi từ đúng 1 nơi (`internal/branch/application/update_branch.go`), không có call site nào khác trong repo:

`UpdateName`, `UpdateSlug`, `UpdateAddress`, `UpdatePhone`, `UpdateEmail`, `UpdateMapsURL`, `UpdateMapsEmbed`, `UpdateLocation`, `UpdateDescription`, `UpdateImages`, `SetPublished`, `UpdateMetaTitle`, `UpdateMetaDescription`.

**Không gộp** `Reorder(orderIndex int)` — được dùng độc lập bởi `UpdateBranchOrder` (endpoint `PUT /branches/{id}/order`, tính năng kéo-thả sắp xếp riêng biệt với form edit). Giữ nguyên, và hàm `Update` mới sẽ gọi lại `Reorder` nội bộ khi `input.OrderIndex != nil` thay vì tự set field, để không có 2 nguồn set `orderIndex`.

## 3. Thay đổi

**Trước**: `application.UpdateBranch` gọi tuần tự 13 `UpdateX()`/`SetX()` riêng lẻ, mỗi hàm tự validate + set field + set `updatedAt`.

**Sau**: 1 hàm domain method:

```go
func (b *Branch) Update(input UpdateBranchInput) error
```

- Nhận thẳng `UpdateBranchInput` (đã tồn tại sẵn, không tạo type mới).
- Giữ đúng **thứ tự validate và fail-fast** hiện tại (dừng ở field lỗi đầu tiên, cùng thứ tự name→slug→address→phone→email→mapsUrl→mapsEmbed→location→description→images→isPublished→orderIndex→metaTitle→metaDescription) — không gộp lỗi kiểu `NewBranch` để tránh đổi observable behavior (response error hiện tại chỉ chứa 1 field lỗi, giữ nguyên).
- `application.UpdateBranch` gọi 1 lần `b.Update(input)` thay vì 13 `if input.X != nil { b.UpdateX(...) }`.
- Field validate helper (`validateName`, `validateSlug`...) giữ nguyên, tái dùng trong `Update`.
- Getter, private field, `NewBranch`, `RehydrateBranch`, `Reorder`, `MarkDeleted`, `Restore` — **không đổi**.

## 4. Rủi ro & an toàn

- **Risk level**: Medium (thay đổi API domain entity, nhưng chỉ 1 call site nội bộ package, có test).
- **Safety net**: `domain/types_test.go` (`TestBranch_UpdateFields`) cập nhật theo API mới (gọi `Update(input)` thay vì `UpdateName`/`UpdateEmail`); `application/branch_test.go` không cần đổi vì chỉ gọi `application.UpdateBranch`, không gọi domain method trực tiếp. Chạy `go build ./... && go vet ./... && go test ./internal/branch/...` xanh trước khi merge.
- **Structural-only**: không đổi behavior HTTP-facing (request/response JSON, validation message, thứ tự lỗi) — thuần túy tổ chức lại code nội bộ.

## 5. Kế hoạch

1. Branch riêng `refactor/branch-consolidate-update`.
2. 1 PR duy nhất (đổi contained trong 1 package, ~100-150 dòng diff) — không cần chia nhỏ theo threshold 100-500 dòng/PR.
3. Code review qua `/code-review` (Standards + Spec).
4. Merge vào `main` sau khi build/vet/test xanh và review pass.
5. Nếu pattern này ổn sau khi chạy thực tế, cân nhắc RFC riêng để lan ra các module CRUD còn lại — **không tự động lan**, chờ quyết định riêng.
