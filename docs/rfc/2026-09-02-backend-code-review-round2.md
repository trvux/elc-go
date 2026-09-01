# RFC: Backend code review round 2 (toàn bộ elc-go)

- **Status**: Completed — 11/11 finding gốc đã fix (8 ở mục 10, 3 còn lại 3.5/3.7/3.10 fix ở mục 11-12 theo yêu cầu thiết kế lại thay vì bỏ qua), cộng 4 finding từ `/code-review high` lần 1 trên chính diff này (mục 13, đã fix hết). `go build ./...`, `go vet ./...`, `go test ./...` sạch sau mỗi bước. Review lần 2 (double-check sau khi fix 4 finding) không chạy được do hết session API limit (không phải do lỗi code) — đã push dựa trên review lần 1 + verify thủ công (build/test xanh, disable-rồi-test lại để xác nhận 2 test mới bắt đúng lỗi, chạy trực tiếp trên DB dev thật để xác nhận fix N+1 khớp hành vi cũ).
- **Date bắt đầu**: 2026-09-02
- **Tác giả**: Trần Vũ (với hỗ trợ Claude Code)

## 1. Bối cảnh

Đợt audit trước (`2026-08-18-backend-code-cleanup.md`, Completed) đã rà 30 domain + cmd/* tại thời điểm đó. Từ đó tới nay có thêm nhiều tính năng lớn (AI chat 3 phase, Redis rate-limit/cache, anomaly detection, backup redesign). Đợt review này (round 2) mục tiêu:

- Rà lại toàn bộ 29 domain dưới `internal/` (không lặp lại finding cũ đã fix), tập trung vào phần code mới từ 2026-08-18 tới nay.
- Không giới hạn phạm vi như RFC trước (dead code / DB race / error handling) — review rộng hơn: business logic, transaction correctness, input validation, auth wiring, tổ chức code.
- Mở rộng ra ngoài `internal/`: `cmd/*` (11 binary), CI/CD (`.github/workflows`), Docker, các script deploy/backup đang chạy live.
- Tài liệu này CHỈ ghi nhận finding. Việc implement fix sẽ làm ở bước sau, có thể tách RFC/PR riêng.

Phương pháp: đọc trực tiếp domain/application/infrastructure/presentation của từng module cho các module lớn/rủi ro cao; với các module còn lại dùng kết hợp đọc trực tiếp + quét script theo các lớp lỗi đã xác nhận (race điều kiện resurrect-slug, transaction hygiene DELETE+INSERT không transaction, SQL injection qua string-concat, route thiếu auth middleware).

## 2. Phạm vi đã review

| Phần | Quy mô | Mức độ đọc |
|---|---|---|
| `internal/platform`, `internal/product`, `internal/ai`, `internal/auth`, `internal/project` | ~15,000 loc | Đọc sâu toàn bộ domain/application/infrastructure/presentation |
| 24 module còn lại (shippingzone, news, service, branch, project-type, category, attribute, group, hp-page, review, brand, service-group, page, inquiry, author, contact, tag, wishlist, system-page, upload, recently-viewed, event, settings, slug-registry) | ~15,000 loc | Đọc domain đầy đủ + spot-check application/infrastructure/presentation, đọc kỹ nhất: service, branch, service-group, review, inquiry, hp-page, shippingzone |
| `cmd/server/main.go` (composition root) | 346 loc | Đọc toàn bộ |
| `cmd/seed-admin`, `cmd/seed-ai-provider` | ~256 loc | Đọc toàn bộ |
| `cmd/sync-ai-pricing`, `cmd/detect-attribute-anomalies` | ~375 loc | Đọc toàn bộ (2 cron job đang chạy live) |
| `cmd/migrate-images`, `cmd/migrate-heading-levels`, `cmd/rename-products-standard`, `cmd/generate-image-crops`, `cmd/audit-attribute-values` | ~1400 loc | Spot-check (transaction/SQL-injection pattern), không đọc business logic chi tiết — đây là tool one-off/đã chạy xong |
| `cmd/migrate-specs-to-attributes`, `cmd/migrate-specs-to-attributes-v2` | ~1200 loc | Chỉ đọc doc-comment + spot-check pattern — migration một-lần đã chạy xong trên production (gated bởi sentinel file), không còn re-run trong vận hành bình thường |
| `.github/workflows/*.yml` (4 file) | — | Đọc toàn bộ |
| `Dockerfile`, `docker-compose.yml` | — | Đọc toàn bộ |
| `scripts/backup-postgres.sh`, `scripts/migrate-all.sh`, `scripts/deploy-attribute-migration.sh`, `scripts/migrate-postgres.sh` | — | Đọc toàn bộ (đang chạy live trong deploy/cron) |
| `internal/upload/infrastructure/crop.go` (shell ra `cwebp`) | — | Đọc — kiểm tra command injection |
| `scripts/*.sql` (~20 file backfill/seed thủ công), `internal/*/migrations/*.sql` (schema gốc) | — | **Chưa đọc** — các fix dữ liệu one-off đã chạy xong trong quá khứ, giá trị review thấp |
| Test suite (`*_test.go`) | — | **Chưa đánh giá** coverage/chất lượng test |

## 3. Findings (đã xác nhận bằng cách đọc code, không phải đoán)

### 3.1 [Nghiêm trọng] `service-group` còn bug race điều kiện resurrect-slug mà RFC trước đã fix ở nơi khác

**File**: `internal/service-group/infrastructure/postgres_repository.go:99` (`Create`)

RFC `2026-08-18-backend-code-cleanup.md` mục 3b đã fix race "check-then-act khi resurrect slug soft-deleted" cho category/group/news/project-type/project — SELECT tìm row soft-deleted trùng slug rồi UPDATE/INSERT ngoài transaction. RFC đó liệt kê "5 domain bị ảnh hưởng" nhưng **bỏ sót `service-group`**, vốn có structure y hệt (comment tại dòng 96-98 còn ghi rõ "slug is globally unique even for deleted rows").

Đã verify bằng script quét toàn bộ `internal/*/infrastructure/*.go`: đây là **trường hợp duy nhất còn lại** trong cả repo có pattern SELECT+INSERT trong `Create()` mà không có transaction.

Hậu quả: 2 request `Create` đồng thời cùng slug (vừa bị soft-delete) → cả hai cùng SELECT thấy `existingID`, cả hai cùng UPDATE resurrect (lost-update, không lỗi nhưng dữ liệu cuối phụ thuộc thứ tự thực thi không xác định). Nếu slug hoàn toàn mới (không có row soft-deleted), 2 request cùng lúc → cả hai SELECT rỗng → cả hai INSERT → tùy schema có unique index toàn cục (`docs/service-group.md` nói "slug is globally unique") hay không mà 1 trong 2 sẽ lỗi constraint (500 thô) hoặc tệ hơn tạo ra 2 row trùng slug active.

**Đề xuất fix**: áp dụng đúng pattern đã dùng cho `project` (`internal/project/infrastructure/postgres_repository.go:538`) — bọc trong transaction, `SELECT ... FOR UPDATE` trước khi quyết định UPDATE hay INSERT.

### 3.2 [Cao] Race điều kiện tạo trùng `ai_conversations` — bug mới, chưa từng được audit

**File**: `internal/ai/infrastructure/postgres_conversation_repository.go:43` (`GetOrCreateByVisitor`)

Cùng lớp bug với 3.1 nhưng áp dụng logic "conversation trong 24h gần nhất": SELECT tìm conversation gần nhất trong 24h của visitor, không thấy thì INSERT — không transaction, không `FOR UPDATE`. Migration (`internal/ai/migrations/000001_baseline_ai.up.sql:59`) chỉ có index thường (`idx_ai_conversations_visitor_id`), **không có unique constraint** nên DB không tự chặn được.

Hậu quả: double-click gửi tin nhắn hoặc mở 2 tab cùng lúc bởi 1 visitor mới (chưa có conversation nào trong 24h) → cả 2 request cùng SELECT rỗng → cả 2 INSERT → 2 conversation row riêng biệt cho cùng 1 visitor, lịch sử chat bị chia làm đôi (ảnh hưởng UX + số liệu usage report theo conversation).

**Đề xuất fix**: bọc SELECT+INSERT trong transaction với `FOR UPDATE`-style lock (khó lock trực tiếp một hàng chưa tồn tại — cân nhắc advisory lock theo `visitor_id`, hoặc unique index dạng `(visitor_id)` có điều kiện + `INSERT ... ON CONFLICT`, tùy thiết kế lại một chút logic "trong 24h" hiện tại).

### 3.3 [Trung bình] `product`: thiếu ràng buộc giữa `IsDefault` và `IsComponentOnly`

**File**: `internal/product/application/create_product.go` (`resolveDefaultVariant`, dùng chung cho Create/Update)

`resolveDefaultVariant` chỉ kiểm tra "đúng 1 variant được đánh dấu default", không kiểm tra variant đó có `IsComponentOnly == true` (tức `is_standalone = false`, chỉ tồn tại như 1 phần của bundle) hay không.

`RecomputeDisplayCache` (`internal/product/infrastructure/variant_repository.go:434`, CTE `dv`) lọc `is_standalone = true` khi tìm default variant để cache `display_price`/`default_variant_id`/`display_stock_status`. Nếu admin lỡ đánh dấu default cho 1 variant component-only → CTE `dv` không match → các cột cache này bị set NULL cho cả sản phẩm, dù sản phẩm vẫn có variant active khác — sản phẩm hiển thị giá trống trên listing/card.

**Đề xuất fix**: `resolveDefaultVariant` reject nếu variant được chọn làm default có `IsComponentOnly == true`.

### 3.4 [Thấp, lặp lại ở nhiều module] Thiếu validate cận dưới cho `limit`/`offset` query param

**File**: lặp lại ở `internal/product/presentation/handler.go` (`parseProductFilter`), `internal/news/presentation/handler.go`, `internal/project/presentation/handler.go`, `internal/brand/presentation/handler.go` — mỗi module tự parse `limit`/`offset` bằng `strconv.Atoi`, không có helper dùng chung.

- `product`/`project`/`brand`: parse lỗi → trả lỗi 400 (product/brand) nhưng **không chặn giá trị âm hoặc 0** — `limit <= 0` bị hiểu ngầm là "không giới hạn" ở tầng SQL (`if filter.Limit > 0 { LIMIT $N }`), và `offset` âm được đưa thẳng vào SQL `OFFSET` → Postgres lỗi "OFFSET must not be negative" → lộ ra thành 500 thay vì 400 sạch.
- `news`: còn lỏng hơn — parse lỗi bị nuốt im lặng (`if n, err := strconv.Atoi(v); err == nil { filter.Limit = n }`), không trả lỗi 400 khi giá trị không phải số, chỉ đơn giản bỏ qua.

**Đề xuất fix**: tách thành 1 helper dùng chung trong `internal/platform/httpserver` (ví dụ `ParsePagination(r *http.Request, defaultLimit, maxLimit int) (limit, offset int, err error)`) — vừa dọn trùng lặp code vừa vá lỗi validate cận dưới tại một chỗ duy nhất. Đây cũng là 1 hạng mục "tổ chức lại" (xem mục 6).

### 3.5 [Thấp] `auth`: goroutine SMTP có thể rò rỉ khi timeout

**File**: `internal/auth/infrastructure/email_sender.go:73` (`send`)

`net/smtp.SendMail` không nhận context nên được bọc bằng goroutine + `select` trên channel `done` / `ctx.Done()`. Khi nhánh timeout thắng, goroutine chạy `smtp.SendMail` vẫn treo tới khi OS tự timeout TCP (có thể vài phút), không bị hủy — request đã trả lỗi cho client nhưng goroutine vẫn tồn tại trong nền. Tần suất gửi email thấp (invite/reset password) nên rủi ro thực tế nhỏ, nhưng đáng ghi nhận nếu SMTP provider hay bị treo.

**Đề xuất fix** (không khẩn): thay `smtp.SendMail` bằng `net.Dialer.DialContext` + các bước SMTP thủ công để tôn trọng ctx thật sự, hoặc chấp nhận rủi ro vì tần suất thấp.

### 3.6 [Rất thấp] `auth`: race hiếm khi accept-invite trùng lúc trả lỗi 500 thay vì 409

**File**: `internal/auth/application/accept_invite.go` (check `ExistsByUsernameOrEmail` rồi mới `Create`), lỗi wrap generic ở `internal/auth/infrastructure/postgres_user_repository.go:38`

DB có unique constraint (`username`, `email` — `internal/auth/migrations/000001_baseline_auth.up.sql:14-15`) nên không tạo dữ liệu trùng, nhưng lỗi unique-violation từ Postgres không được map sang `apperr.NewConflictError` — người thua trong race hiếm (2 người dùng cùng username/email accept-invite đồng thời) nhận 500 thay vì 409. Cùng dạng finding mà RFC trước đã chủ động không fix hết mọi TOCTOU tương tự (ưu tiên thấp).

### 3.7 [Cần verify ngoài repo] `httpserver.ClientIP` tin tưởng tuyệt đối `X-Forwarded-For`

**File**: `internal/platform/httpserver/client_ip.go:14`

Chỉ lấy phần tử đầu của `X-Forwarded-For` do client gửi, không tự xác minh. Đúng nếu Nginx phía trước **ghi đè** (không phải append) header này. Cấu hình Nginx không nằm trong repo elc-go nên không verify được từ đây. Nếu Nginx dùng `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;` (append) mà không có `ngx_http_realip_module`, client có thể tự chèn IP giả để né rate-limit theo IP (login, forgot-password, ai/chat, inquiry, event).

**Đề xuất**: xác nhận cấu hình Nginx thực tế (không phải việc của elc-go repo, chỉ ghi nhận).

### 3.8 [Trung bình] `service`: không validate khoảng hợp lệ của `discountPercent`/`originalPrice`

**File**: `internal/service/domain/types.go` (`NewService`, `UpdatePricing`), `internal/service/application/create_service.go`, `internal/service/application/update_service.go`

`SalePrice()` (dòng 181-189) tính `original - (original*discount)/100` nhưng không nơi nào (domain lẫn application) kiểm tra `discountPercent` nằm trong [0, 100] hay `originalPrice >= 0`. `discountPercent > 100` → giá bán ra âm; `discountPercent < 0` → giá bán ra cao hơn giá gốc. Giá trị này hiển thị trực tiếp cho khách hàng trên trang dịch vụ. Cùng loại lỗ hổng validate giá mà `docs/rfc/2026-08-18-product-data-anomaly-detection.md` đã vá cho `product`'s variant price (`OriginalPrice <= 0` / `SalePrice <= 0` reject), nhưng chưa từng áp dụng cho `service`.

**Đề xuất fix**: thêm validate `0 <= discountPercent <= 100` và `originalPrice >= 0` vào `NewService`/`UpdatePricing`, cùng chỗ với các validate khác.

### 3.9 [Trung bình, cần xác nhận frontend] `branch`: `mapsEmbed` không sanitize, khả năng stored XSS

**File**: `internal/branch/domain/types.go:371` (`validateMapsEmbed`)

`validateMapsEmbed` chỉ kiểm tra chuỗi không rỗng — không sanitize hay validate cấu trúc HTML. Trường này theo thiết kế là mã nhúng iframe Google Maps ("Share > Embed a map"), tức là chuỗi HTML thô (`<iframe src="...">...</iframe>`) được admin dán vào. Nếu frontend render trường này bằng raw HTML injection (gần như chắc chắn, vì bản chất "mã nhúng" là vậy) thì bất kỳ tài khoản nào có quyền `CanWriteContent` (không chỉ super-admin) đều có thể chèn `<script>` hoặc HTML độc hại vào trang chi tiết chi nhánh — stored XSS ảnh hưởng khách truy cập trang đó. Chưa xác nhận được cách frontend (elc-tem, không nằm trong repo này) render trường này.

**Đề xuất fix**: nếu frontend render raw HTML, giới hạn `mapsEmbed` chỉ chấp nhận `<iframe>` với `src` thuộc domain Google Maps đã biết (whitelist domain + strip mọi attribute/tag khác), thay vì chấp nhận HTML tuỳ ý.

### 3.10 [Ghi nhận hiệu năng, không phải bug] `product`: facet theo attribute là N+1 query

**File**: `internal/product/infrastructure/facet_repository.go:139` (`computeAttributeFacets`)

Với mỗi attribute definition áp dụng cho category đang xem, hàm chạy 1 query riêng (`computeAttributeRangeFacet` hoặc `computeAttributeTokenFacet`) — nghĩa là 1 category có N attribute definition sẽ tốn thêm N query mỗi lần gọi `GetAll` (ngoài 5 query chạy song song qua errgroup đã có). Không sai, nhưng số lượng query tăng tuyến tính theo số attribute của category — đáng chú ý nếu catalog mở rộng nhiều attribute/category hơn trong tương lai và listing page bắt đầu chậm.

### 3.11 [Trung bình] `sync-ai-pricing`: ghi giá AI provider vào DB không có validate/sanity-check, phụ thuộc hoàn toàn vào LLM extraction từ trang web bên ngoài

**File**: `cmd/sync-ai-pricing/main.go` (`extractPricing`, `syncProviderPricing`), `internal/ai/infrastructure/postgres_model_repository.go:185` (`UpdatePricing`)

Cron job chạy hàng ngày: fetch nội dung thô trang pricing của provider (`fetchDocText`, tối đa 60,000 ký tự), nhét thẳng vào prompt gửi cho LLM để tự trích xuất JSON giá, rồi `UpdatePricing` ghi thẳng kết quả vào `ai_models.pricing` — **không có bất kỳ validate nào**: không kiểm tra số dương, không kiểm tra `currency == "USD"`, không so sánh với giá cũ để phát hiện thay đổi bất thường (khác hẳn cách `detect-attribute-anomalies` xử lý spec value — có hẳn IQR outlier check trước khi trust dữ liệu). Hai rủi ro:

1. Nội dung trang pricing được đưa thẳng vào prompt như văn bản không tin cậy — nếu trang chứa nội dung gây nhiễu (vô tình hoặc cố ý, kiểu prompt injection) hoặc trang tạm thời lỗi/đổi bố cục, LLM có thể trích xuất sai số mà không có gì chặn lại.
2. Giá sai (ví dụ 0, âm, hoặc gấp 1000 lần) sẽ được ghi thẳng vào DB, làm sai lệch usage/cost report (Phase 3 RFC) — không có cảnh báo, không có giới hạn % thay đổi tối đa so với lần chạy trước.

**Đề xuất fix**: thêm sanity check trước khi `UpdatePricing` — giá phải dương, `currency` khớp giá trị mong đợi, và log cảnh báo (không tự ghi) nếu giá mới lệch quá X% so với giá đang lưu.

## 4. Kết quả quét toàn repo theo pattern (áp dụng cho tất cả 29 module `internal/`)

Đã chạy script quét `internal/*/infrastructure/*.go` và `internal/*/presentation/routes.go` cho:

1. **Race điều kiện resurrect-slug** (SELECT+INSERT trong `Create()` không transaction): chỉ còn `service-group` (mục 3.1). Tất cả module khác đã transaction hóa đúng.
2. **Update() có DELETE+INSERT (thay thế quan hệ con) không transaction**: sạch — không có module nào vi phạm.
3. **SQL injection qua string-concat giá trị (không phải placeholder `$N`)**: sạch — toàn bộ dynamic WHERE/filter building đều dùng `$N` parameterized, kể cả các query động phức tạp nhất (product's search/facet). `cmd/migrate-images`/`cmd/migrate-heading-levels` dùng `fmt.Sprintf` để build tên bảng/cột nhưng từ hardcoded slice literal trong chính source file, không phải input runtime — an toàn.
4. **`tx.Begin()` thiếu `defer tx.Rollback()`**: sạch.
5. **Route ghi (POST/PUT/PATCH/DELETE) không có `RequireAuth`**: chỉ có các route công khai có chủ đích (chat, login/register, contact/inquiry form, batch product lookup, recently-viewed, review submit, wishlist add/remove theo visitor cookie) — không có lỗ hổng thiếu auth thật sự.
6. **RFC/docs status hygiene**: sạch — toàn bộ `docs/rfc/*.md` đã ở trạng thái Completed đúng convention, không có tài liệu lỗi thời.

## 5. Ghi nhận riêng cho `cmd/*` / CI-CD / Docker / scripts

- `cmd/server/main.go` (composition root): wiring đúng cho toàn bộ 29 handler, đúng convention fail-fast cho secret bắt buộc (`JWT_SECRET`, `AI_SECRETS_ENCRYPTION_KEY`), graceful-degrade đúng cho phần optional (Redis, SMTP, R2). Thứ tự defer `pool.Close()`/`log.Sync()` đúng — pool đóng trước khi log flush lần cuối, không có request nào bị cắt ngang.
- `cmd/seed-admin`, `cmd/seed-ai-provider`: sạch, được thiết kế đúng làm break-glass tool (không log secret, chỉ lấy từ env).
- `cmd/detect-attribute-anomalies`: thuật toán IQR (Tukey's fences) đúng, transaction đúng, idempotent thật sự (tự reset flag khi sample nhỏ lại).
- `internal/upload/infrastructure/crop.go` (shell ra `cwebp`): không có command injection — ảnh truyền qua stdin/stdout (`-o -`, đối số cuối `-`), không đối số nào lấy từ input người dùng.
- `Dockerfile`/`docker-compose.yml`: multi-stage build hợp lý, healthcheck đúng, Postgres/Redis không expose port ra ngoài mạng Docker — sạch.
- `.github/workflows/deploy.yml`: pipeline deploy rất kỹ — build trước khi swap container, migration chạy trước khi đổi container mới (container cũ vẫn phục vụ nếu migration lỗi), rsync loại trừ đúng thư mục runtime-only (`backups/`, `.env`). Một bug crontab đã từng xảy ra (dòng comment 126-134 trong file) đã được tìm ra và fix, không phải finding mới.
- `scripts/backup-postgres.sh`: có integrity check bằng `pg_restore --list` trước khi tin tưởng backup, retention đúng cho cả local lẫn R2 — chất lượng vận hành rất tốt.
- `scripts/migrate-postgres.sh`, `scripts/deploy-attribute-migration.sh`: migration một-lần, gated bằng sentinel file trong `backups/` (thư mục duy nhất rsync loại trừ) — đã chạy xong trên production, không còn nguy cơ tái diễn trong vận hành bình thường.

## 6. Đề xuất "tổ chức lại" (không phải bug, chờ quyết định riêng)

- Tách helper `ParsePagination` dùng chung (xem 3.4) — giảm trùng lặp giữa product/news/project/brand và có thể cả các module khác chưa kiểm tra kỹ.
- Hạng mục "consolidate per-field UpdateX() into one Update(input)" đã có sẵn trong CLAUDE.md (áp dụng lazy khi chạm vào module) — không lặp lại ở đây, chỉ nhắc để không quên khi làm implement.

## 7. Tổng kết finding (11 mục, xếp theo mức độ)

| # | Mức độ | Vị trí | Vấn đề |
|---|---|---|---|
| 3.1 | Nghiêm trọng | internal/service-group | Race resurrect-slug — bug đã fix ở 5 domain khác nhưng bỏ sót domain này |
| 3.2 | Cao | internal/ai | Race tạo trùng `ai_conversations` cho cùng visitor |
| 3.3 | Trung bình | internal/product | Thiếu ràng buộc `IsDefault` + `IsComponentOnly` → display_price NULL |
| 3.8 | Trung bình | internal/service | Không validate khoảng `discountPercent`/`originalPrice` → giá âm |
| 3.9 | Trung bình (cần xác nhận frontend) | internal/branch | `mapsEmbed` không sanitize — khả năng stored XSS |
| 3.11 | Trung bình | cmd/sync-ai-pricing | Ghi giá AI vào DB không sanity-check, phụ thuộc LLM extraction từ web |
| 3.4 | Thấp (lặp lại ≥4 module) | internal/product,news,project,brand | Thiếu validate cận dưới `limit`/`offset` |
| 3.5 | Thấp | internal/auth | Goroutine SMTP có thể rò rỉ khi timeout |
| 3.6 | Rất thấp | internal/auth | Race hiếm accept-invite → 500 thay vì 409 |
| 3.7 | Cần verify ngoài repo | internal/platform | `ClientIP` tin `X-Forwarded-For` — phụ thuộc cấu hình Nginx |
| 3.10 | Ghi nhận hiệu năng | internal/product | Facet theo attribute là N+1 query |

Đã đọc toàn bộ `internal/` (29 module) + `cmd/*` (11 binary) + CI/CD + Docker + script deploy/backup live. Không tìm thấy SQL injection, thiếu transaction, hay route ghi thiếu auth ở bất kỳ đâu ngoài các mục đã liệt kê. Hạ tầng vận hành (deploy pipeline, backup, migration) có chất lượng cao, nhiều bài học sự cố thật đã được ghi lại và sửa.

## 8. Chưa review (nếu cần làm tiếp)

- `scripts/*.sql` (~20 file backfill/seed thủ công) — các fix dữ liệu one-off đã chạy xong, giá trị review thấp trừ khi nghi ngờ một fix cụ thể còn tác dụng phụ.
- `internal/*/migrations/*.sql` — chưa đọc schema gốc từng dòng (chỉ grep các đoạn liên quan đến finding cụ thể).
- Chất lượng/coverage của test suite (`*_test.go`) — chưa đánh giá test có che được các finding ở mục 3 không.
- `cmd/migrate-specs-to-attributes(-v2)`, `cmd/audit-attribute-values`, `cmd/rename-products-standard`, `cmd/migrate-heading-levels`, `cmd/generate-image-crops` — chỉ spot-check pattern (transaction/injection), chưa đọc business logic chi tiết vì đều là tool one-off đã chạy xong hoặc rủi ro thấp (audit-attribute-values chỉ đọc, không ghi).

## 9. Kế hoạch triển khai fix

Thứ tự ưu tiên: rủi ro thấp → cao (mechanical trước, đổi transaction/logic race phức tạp để cuối cùng), cùng nguyên tắc RFC trước đã dùng. Mỗi mục là 1 đổi thay độc lập, verify bằng `go build ./...` + `go vet ./...` + `go test ./...` sau mỗi bước.

| Thứ tự | Mục | Việc làm |
|---|---|---|
| 1 | 3.4 | Thêm `httpserver.ParsePagination`, áp dụng cho product/news/project/brand |
| 2 | 3.6 | Map lỗi unique-violation (Postgres code 23505) sang `apperr.NewConflictError` trong `postgres_user_repository.go` |
| 3 | 3.3 | `resolveDefaultVariant` reject nếu variant default có `IsComponentOnly == true` |
| 4 | 3.8 | Validate `0 <= discountPercent <= 100`, `originalPrice >= 0` trong `service` domain |
| 5 | 3.9 | Hardening `validateMapsEmbed`: chỉ chấp nhận `<iframe>` với `src` thuộc `google.com/maps` (best-effort ở backend — chưa xác nhận được cách frontend render nên đây là lớp phòng thủ thêm, không phải fix triệt để) |
| 6 | 3.11 | Thêm sanity check giá (dương, lệch % so với giá cũ) trước `UpdatePricing` trong `cmd/sync-ai-pricing` |
| 7 | 3.1 | `service-group.Create`: bọc transaction + `SELECT ... FOR UPDATE`, theo đúng pattern `project` |
| 8 | 3.2 | `ai.GetOrCreateByVisitor`: bọc transaction + Postgres advisory lock theo `visitor_id` |
| — | 3.5, 3.7, 3.10 | Không fix trong đợt này — 3.5 rủi ro thấp/tần suất thấp (chấp nhận), 3.7 ngoài phạm vi repo (không phải code Go), 3.10 là ghi nhận hiệu năng không phải bug |

## 10. Kết quả implement

Tất cả đã build/vet/test sạch (`go build ./...`, `go vet ./...`, `go test ./...`). Không tạo git commit — working tree đang có các thay đổi chưa commit, chờ user review.

| Mục | Trạng thái | File chính đã đổi |
|---|---|---|
| 3.4 (limit/offset) | **Đã fix** | `internal/platform/httpserver/pagination.go` (mới, `ParsePagination`), áp dụng vào `product`/`news`/`project`/`brand` handler |
| 3.6 (accept-invite 500→409) | **Đã fix** | `internal/auth/infrastructure/postgres_user_repository.go` (map Postgres 23505 → `apperr.NewConflictError`), `internal/auth/application/accept_invite.go` (pass-through AppError thay vì luôn bọc `NewInternalError`) |
| 3.3 (product IsDefault+IsComponentOnly) | **Đã fix** | `internal/product/application/create_product.go` (`resolveDefaultVariant` reject variant component-only làm default) |
| 3.8 (service discount/price) | **Đã fix** | `internal/service/domain/types.go` (`validatePricing`, áp dụng vào `NewService` + `UpdatePricing`), `internal/service/application/update_service.go` (xử lý lỗi mới từ `UpdatePricing`) |
| 3.9 (branch mapsEmbed XSS) | **Đã fix** (best-effort, backend-only) | `internal/branch/domain/types.go` (`validateMapsEmbed` giờ chỉ chấp nhận 1 thẻ `<iframe>` với `src` whitelist domain Google + whitelist tên thuộc tính); cập nhật 9 chỗ test fixture dùng `"<iframe></iframe>"` cũ sang snippet hợp lệ; thêm `TestValidateMapsEmbed` (8 case, gồm cả injection) |
| 3.11 (sync-ai-pricing) | **Đã fix** | `cmd/sync-ai-pricing/main.go` (`validateExtractedPricing` — chặn giá âm/0, chặn lệch >5x so với giá đang lưu), thêm `cmd/sync-ai-pricing/main_test.go` (7 case) |
| 3.1 (service-group race) | **Đã fix** | `internal/service-group/infrastructure/postgres_repository.go` (`Create` bọc transaction + `SELECT ... FOR UPDATE`, đúng pattern `project`) |
| 3.2 (ai conversation race) | **Đã fix** | `internal/ai/infrastructure/postgres_conversation_repository.go` (`GetOrCreateByVisitor` bọc transaction + `pg_advisory_xact_lock(hashtext(visitor_id))` — không dùng `FOR UPDATE` được vì có thể chưa có row nào để lock trên 1 visitor hoàn toàn mới) |
| 3.5 (SMTP goroutine leak) | Không fix | Rủi ro thấp/tần suất thấp, chấp nhận theo kế hoạch mục 9 |
| 3.7 (X-Forwarded-For) | Không fix | Ngoài phạm vi code Go — cần xác nhận cấu hình Nginx, không phải việc sửa trong repo này |
| 3.10 (N+1 facet query) | Không fix | Ghi nhận hiệu năng, không phải bug |

## 11. Kế hoạch fix bổ sung (3.5, 3.7, 3.10 — theo yêu cầu tiếp theo của user)

| Mục | Thiết kế |
|---|---|
| 3.5 | Bỏ hẳn goroutine + `select`. Viết lại `send()` bằng `net.Dialer.DialContext` (tôn trọng `ctx` ngay ở bước dial) + `conn.SetDeadline(...)` trên chính connection trước khi chạy giao thức SMTP thủ công qua `smtp.NewClient` (STARTTLS nếu server hỗ trợ, giống hệt `smtp.SendMail` chuẩn) — khi hết deadline, các lệnh Read/Write trên `conn` tự trả lỗi ngay lập tức, không có goroutine nền nào tồn tại sau khi hàm return. |
| 3.7 | Đổi `ClientIP` từ "lấy hop ĐẦU TIÊN" sang "lấy hop CUỐI CÙNG" trong `X-Forwarded-For`. Với đúng 1 reverse proxy phía trước (Nginx, theo ARCHITECTURE.md §12) dùng `$proxy_add_x_forwarded_for` (kiểu append chuẩn), hop cuối luôn là địa chỉ Nginx trực tiếp thấy được — client không thể giả mạo được hop này dù có tự chèn X-Forwarded-For giả ở đầu chuỗi. Đây là pattern chuẩn "trust N proxy hops, đọc từ phải sang" (giống `trust proxy` của Express/`XForwardedForMiddleware` của nhiều framework khác) — sửa được ngay trong code Go, không cần đụng tới cấu hình Nginx. Nếu Nginx ghi đè (set, không append) thì chỉ có 1 hop, lấy đầu hay cuối đều như nhau — không có regression. |
| 3.10 | Bỏ vòng lặp N query tuần tự (1 query/attribute definition). Thay bằng đúng 2 query gộp (UNION ALL) cho toàn bộ facet: 1 query cho mọi attribute number-type, 1 query cho mọi attribute select/multiselect/boolean-type — mỗi nhánh UNION vẫn giữ nguyên điều kiện "exclude own dimension" y hệt logic cũ (không đổi hành vi, chỉ gộp round-trip). `buildFilterConditions` được refactor thêm biến thể nhận `startArgN` để đánh số `$N` liên tục qua các nhánh UNION. Kết quả: số round-trip cho attribute facets từ O(N) tuần tự xuống còn tối đa 2, không phụ thuộc số lượng attribute của category. |

## 12. Kết quả fix 3.5 / 3.7 / 3.10

| Mục | File chính | Verify |
|---|---|---|
| 3.5 | `internal/auth/infrastructure/email_sender.go` (`send` viết lại hoàn toàn, bỏ goroutine) | `go build`/`vet`/`test` sạch. Không thêm test (tầng infra SMTP vốn không có test, cần server thật) |
| 3.7 | `internal/platform/httpserver/client_ip.go` (đọc hop cuối thay vì đầu) | Thêm `client_ip_test.go` (4 case, gồm case spoofed-first-hop) — pass |
| 3.10 | `internal/product/infrastructure/facet_repository.go` (2 hàm batch mới thay N query cũ), `internal/product/infrastructure/postgres_repository.go` (`buildFilterConditionsFrom` — biến thể nhận `startArgN` để đánh số `$N` liên tục qua các nhánh UNION ALL) | **Verify bằng dữ liệu thật**: chạy thử trực tiếp trên DB dev đang chạy (docker container `elc-postgres`, 179 sản phẩm/125 attribute definition/3274 giá trị) qua 1 tool tạm thời (đã xoá sau khi xong) — so khớp chính xác kết quả count với raw SQL, và xác nhận đúng hành vi "exclude own dimension" (filter `cong_nghe_inverter:true` giảm total 59→56, facet `cong_nghe_inverter` giữ nguyên cả 2 option, facet `xuat_xu` tính lại theo tập đã lọc 34/9→33/7) — khớp 100% hành vi cũ. |

### 3.12 [Phát hiện mới, ngoài phạm vi đợt fix này] Integration test đã stale ở ít nhất 4 module

Trong lúc verify 3.10 bằng `go vet -tags=integration ./...`, phát hiện **integration test đã lỗi thời, không compile được**, ở:
- `internal/branch/infrastructure/postgres_repository_integration_test.go` — thiếu tham số so với `domain.NewBranch` hiện tại (chữ ký đã đổi khi thêm field `mapsEmbed`/địa chỉ hành chính mới)
- `internal/brand/infrastructure/postgres_repository_integration_test.go` — thiếu tham số so với `domain.NewBrand` hiện tại
- `internal/product/infrastructure/postgres_repository_integration_test.go` + `variant_repository_integration_test.go` — thừa tham số so với `domain.NewProduct` hiện tại (chữ ký đã đổi sau redesign v2)
- `internal/project/infrastructure/postgres_repository_integration_test.go` — gọi `repo.GetBySlug` với 3 tham số trong khi interface hiện chỉ nhận 2

Vì các file này build fail, `go test -tags=integration ./internal/<module>/infrastructure/...` **không chạy được** cho cả 4 module — an toàn lưới integration test coi như không tồn tại cho các module này dù file vẫn còn trong repo (dễ gây hiểu lầm là "có test"). Đây là rot tích luỹ qua nhiều lần đổi domain constructor mà không cập nhật lại test — không liên quan trực tiếp tới 11 finding gốc, không fix trong đợt này (sửa cả 4 file là một việc riêng, cần đối chiếu kỹ từng chữ ký hiện tại). Ghi nhận để làm sau.

## 13. Kết quả `/code-review high` lần 1 trên diff (trước khi push) — 4 finding, đã fix hết

Chạy `/code-review high` trên toàn bộ diff trước khi push main. Cả 4 finding đều xác nhận là thật (không phải false positive), đã sửa:

1. **`resolveDefaultVariant` (3.3) auto-default vẫn chọn variant component-only nếu nó đứng đầu danh sách** — khi không ai đánh dấu default tường minh, logic cũ chọn `variants[0]` vô điều kiện rồi mới kiểm tra `IsComponentOnly`, khiến case `[componentOnly, standalone, standalone]` (không đánh dấu default) bị reject dù `variants[1]`/`[2]` là lựa chọn hợp lệ — biến 1 request trước đây "thành công nhưng có bug 3.3" thành "luôn bị từ chối", kể cả khi có cách chọn default hợp lệ. **Fix**: auto-default giờ bỏ qua variant component-only khi tìm ứng viên, chỉ reject khi được đánh dấu default tường minh trên 1 variant component-only, hoặc khi toàn bộ variant đều component-only (không còn lựa chọn hợp lệ). Thêm 3 test case mới (`TestResolveDefaultVariant_SkipsComponentOnlyWhenAutoDefaulting`, `_AllComponentOnlyIsRejected`, `_ExplicitComponentOnlyDefaultIsRejected`).
2. **`mapsEmbed` (3.9) whitelist thuộc tính `style` nhưng không validate giá trị** — cho phép `style="background:url('https://evil.example.com/exfil')"` lọt qua dù attribute name hợp lệ, làm giảm hiệu quả chống injection của cả fix. **Fix**: thêm `mapsEmbedStyleRe` chỉ chấp nhận đúng giá trị Google thực sự phát ra (`border:0;` hoặc `border:0`), mọi giá trị khác bị từ chối. Thêm 2 test case (chặn CSS injection, chấp nhận giá trị Google thật).
3. **`email_sender.go` (3.5) chỉ tính deadline 1 lần lúc đầu, không theo dõi ctx bị cancel giữa chừng sau khi đã kết nối** — nếu ctx bị cancel tường minh (không phải hết hạn) sau khi TCP connection đã mở, code mới không phản ứng ngay mà chờ tới deadline đã tính từ đầu — hồi quy nhẹ so với bản goroutine+select cũ (luôn theo dõi `ctx.Done()` liên tục). **Fix**: thêm 1 goroutine watcher nhẹ chỉ `select` giữa `ctx.Done()` và `done` (đóng qua `defer` khi hàm return) — vòng đời luôn có giới hạn, không lặp lại bug rò rỉ ban đầu (khác goroutine cũ chạy `smtp.SendMail` block thật sự). Thêm test `TestSMTPEmailSender_send_RespectsContextCancellation`, **đã verify bằng cách tắt tạm fix rồi chạy lại — xác nhận test thật sự bắt được lỗi (không có watcher thì mất đúng 10s thay vì phản hồi ngay)**.
4. **`service` domain vẫn giữ pattern 13 hàm `UpdateX()`/`SetX()` riêng lẻ dù CLAUDE.md nêu đích danh module này cần gộp khi bị đụng tới** — đợt fix 3.8 đụng vào `UpdatePricing` đúng lúc trigger convention "áp dụng lazy khi chạm vào module". **Fix**: gộp toàn bộ 13 hàm (`UpdateTitle`, `UpdateSlug`, `UpdateGroupID`, `UpdateCategoryID`, `UpdatePricing`, `UpdatePriceDisplayText`, `SetLabels`, `UpdateDescription`, `UpdateContent`, `UpdateImages`, `UpdateMetaTitle`, `UpdateMetaDescription`, `SetFeatured`, `SetPublished`) thành 1 hàm `Update(input UpdateServiceInput) error` duy nhất, đúng pattern đã dùng cho `branch` (`docs/rfc/2026-08-18-branch-domain-consolidate-update.md`) — verify bằng grep xác nhận mỗi hàm cũ chỉ có đúng 1 call site (`application/update_service.go`) trước khi gộp. `application.UpdateService` giờ chỉ còn gọi `service.Update(input)`. `Reorder` giữ nguyên riêng (không có endpoint reorder độc lập như branch nhưng giữ để đồng nhất convention). Cập nhật test hiện có (`TestService_UpdatePricing_NeverGoesStale`) gọi qua `Update()` thay vì `UpdatePricing()` trực tiếp, thêm 2 test mới (`_RejectsInvalidDiscountPercent`, `_NoOpLeavesUpdatedAtUnchanged`).

Build/vet/test toàn repo sạch sau cả 4 fix.

**Rủi ro cần lưu ý khi review**:
- 3.1/3.2 đổi transaction logic — khuyến nghị test thủ công 2 request đồng thời (hoặc integration test có DB thật) trước khi deploy, script tự động chỉ verify build/test đơn vị hiện có, không có test race-condition thật.
- 3.9 làm chặt hẳn validate `mapsEmbed` — nếu chi nhánh nào đang có dữ liệu `mapsEmbed` cũ không đúng format `<iframe src="https://www.google.com/...">` (ví dụ chỉ có URL trần, hoặc dùng domain khác), **lần Update tiếp theo của chi nhánh đó sẽ bị chặn** cho tới khi sửa lại đúng định dạng — dữ liệu cũ trong DB không bị ảnh hưởng (chỉ áp dụng ở write path), nhưng nên kiểm tra dữ liệu hiện có trước khi merge.
