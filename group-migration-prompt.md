Migrate module `group` từ Next.js/Supabase (`/Users/tranvux/Documents/elc-tem`) sang Go
(`/Users/tranvux/Documents/elc-go`). Đây là module thứ 5 trong một cuộc migration
strangler-fig đang chạy — `contact`, `service-group`, `service`, `brand` đã xong, dùng làm
tham chiếu bắt buộc. Bạn PHẢI đọc kỹ mọi thứ ở mục 1-3 trước khi viết bất kỳ dòng code nào.
Đừng đoán, đừng bịa field/behavior — mọi quyết định thiết kế trong prompt này đều dựa trên
kết quả `psql` thật + đọc code TS thật, không phải suy luận từ tên biến.

**Cảnh báo quan trọng nhất**: `group` trông giống hệt `brand` về shape field
(name/slug/imageUrl/metaTitle/metaDescription/isFeatured/orderIndex/content/faq), nên rất dễ
bị cám dỗ copy-paste 100% code của `brand` rồi đổi tên. ĐỪNG làm vậy một cách mù quáng — có
ít nhất 3 điểm khác biệt hành vi thật sự so với `brand` được liệt kê chi tiết ở mục 4, nếu
copy y nguyên sẽ tạo ra bug. Đọc kỹ mục 4 trước khi bắt tay code.

## 1. Đọc bắt buộc trước khi code (không được bỏ qua)

1. `/Users/tranvux/Documents/elc-go/ARCHITECTURE.md` — quy ước bắt buộc: layering
   (domain/application/infrastructure/presentation), rich domain model (field private,
   `New<Entity>`/`Rehydrate<Entity>`, derived field tính toán chứ không lưu), error handling
   (`apperr`), stack (chi/pgx/zap), quy ước migration table riêng cho mỗi module
   (`schema_migrations_<module>`), deploy, checklist performance/security.
2. Toàn bộ file trong `/Users/tranvux/Documents/elc-go/docs/`:
   - `contact.md` — pilot, gotcha pgx + PgBouncer prepared-statement
     (`QueryExecModeSimpleProtocol`, đã fix ở tầng platform, không cần làm lại).
   - `service-group.md` — resurrect-on-create khi slug unique toàn cục, soft-delete cascade
     phải chạy trong 1 transaction vì FK `ON DELETE SET NULL` không tự chạy khi soft-delete
     (soft-delete là UPDATE, không phải DELETE thật).
   - `service.md` — audit tìm bug tính giá thật (SalePrice), bug camelCase/snake_case,
     dead-code trong join 3 bảng.
   - `brand.md` — **đọc kỹ nhất, đây là template gần nhất với `group`**: pgx jsonb gotcha khi
     encode/decode `[]FAQItem` (bắt buộc phải biết trước, xem mục 5 bên dưới), FK
     contradiction (NOT NULL cột nhưng FK lại `ON DELETE SET NULL` — với `group` KHÔNG có
     tình huống này, xem mục 4), dead-code `getBrandBySlug`/`getByIds`, cách viết
     `docs/brand.md`.
3. Code mẫu cụ thể (đọc toàn bộ, không chỉ lướt):
   - `/Users/tranvux/Documents/elc-go/internal/brand/` — **dùng làm khung sườn chính** vì
     shape field giống hệt (rich domain model, migration, application, infrastructure,
     presentation, integration test). Copy cấu trúc file, ĐỔI theo mục 4 khi cần.
   - `/Users/tranvux/Documents/elc-go/internal/service-group/` — dùng làm tham chiếu cho
     transaction-wrapped cascade soft-delete (`SoftDelete` của service-group null hoá
     `services.group_id` trong 1 transaction) — pattern này liên quan trực tiếp tới `group`,
     xem mục 4.3.
4. `/Users/tranvux/Documents/elc-tem/modules/group/` — code TS hiện tại (đọc hết:
   `domain/types.ts`, `domain/repository.ts`, `domain/validators.ts`, `application/index.ts`,
   `infrastructure/groupRepo.ts`, `presentation/actions.ts`, `presentation/components/*`).
5. `/Users/tranvux/Documents/elc-tem/shared/lib/go-api.ts` — hàm `toSnakeCaseBody` **bắt
   buộc** dùng cho mọi Server Action gửi request qua Go (Go dùng snake_case, TS dùng
   camelCase; quên bước này Go sẽ âm thầm nhận field = 0/false/"" thay vì báo lỗi — đây là
   lỗi thật đã xảy ra nếu quên).

## 2. Quy trình — đúng 8 bước đã dùng cho mọi module trước

1. **Audit** code TS hiện tại + audit DB thật bằng `psql` (KHÔNG chỉ tin vào
   `database.types.ts` — file này có thể lỗi thời, không phản ánh đúng schema/trigger/FK
   thật, đã xảy ra với `brand`: `database.types.ts` thiếu cột `content`/`faq` dù cột đó tồn
   tại thật trên DB).
2. **Domain** — rich domain model: field private, constructor `NewGroup`/`RehydrateGroup`,
   method mutate (pointer receiver), tính toán field derived thay vì lưu (nếu có). Unit test
   tầng domain.
3. **Migration** — baseline `golang-migrate` (`IF NOT EXISTS`, vì bảng đã có sẵn ở Supabase).
   `make migrate-create module=group name=baseline_group_categories`, rồi
   `make migrate-up module=group`.
4. **Application** — use case + fake repository cho test (xem
   `internal/brand/application/fake_repository_test.go`).
5. **Infrastructure** — repository `pgx`. Viết test `-tags=integration` chạy với DB thật
   (`DATABASE_URL` đã có sẵn trong `elc-go/.env`), tự dọn dẹp sau khi test.
6. **Presentation** — chi handler + route, đúng format response/DTO như `brand`/
   `service-group` (JSON snake_case, lỗi map qua `apperr`).
7. **Wire vào `cmd/server/main.go`** — nhớ alias import (`groupinfra`, `grouppresentation`)
   vì mọi module đều đặt tên package `infrastructure`/`presentation` giống nhau.
8. **Cutover `elc-tem`** — xem chi tiết mục 6.

Sau MỖI bước Go, chạy: `gofmt -l .` (phải rỗng), `go vet ./...`, `go build ./...`,
`go test ./internal/group/...`. Đừng dồn hết lỗi tới cuối mới build 1 lần.

## 3. Schema thật (đã verify bằng `psql "$DATABASE_URL"`, KHÔNG cần đoán lại — nhưng bạn vẫn
nên tự chạy lại `\d group_categories` và `\d categories` một lần để tự tin, vì DB có thể đã
đổi giữa lúc viết prompt này và lúc bạn code)

**Tên bảng thật là `group_categories`, KHÔNG PHẢI `groups`.** TS code
(`modules/group/infrastructure/groupRepo.ts`) đã xác nhận: `TABLE_NAME = "group_categories"`.

```
Table "public.group_categories"
      Column      |           Type           | Nullable |      Default
------------------+--------------------------+----------+-------------------
 id               | uuid                     | not null | gen_random_uuid()
 name             | text                     | not null |
 created_at       | timestamptz              |          | now()
 updated_at       | timestamptz              |          | now()
 deleted_at       | timestamptz              |          |
 slug             | text                     | not null |
 image_url        | text                     |          |
 meta_title       | text                     |          |
 meta_description | text                     |          |
 is_featured      | boolean                  | not null | false
 order_index      | integer                  | not null | 0
 content          | jsonb                    |          |
 faq              | jsonb                    |          | '[]'::jsonb
Indexes:
    "groups_pkey" PRIMARY KEY, btree (id)
    "group_categories_slug_unique_active" UNIQUE, btree (slug) WHERE deleted_at IS NULL
Referenced by:
    TABLE "categories" CONSTRAINT "category_group_id_fkey"
        FOREIGN KEY (group_id) REFERENCES group_categories(id) ON DELETE SET NULL
Triggers:
    trg_group_slug_registry ... EXECUTE FUNCTION sync_group_slug_registry()
    update_groups_updated_at BEFORE UPDATE ... EXECUTE FUNCTION update_updated_at_column()
```

```
Table "public.categories"  (chỉ phần liên quan tới group — category tự nó chưa migrate)
      Column      |   Type      | Nullable | Default
------------------+-------------+----------+-------------------
 id               | uuid        | not null | gen_random_uuid()
 group_id         | uuid        |          |     <-- NULLABLE, khác với brand/products.brand_id
 ...
Foreign-key constraints:
    "category_group_id_fkey" FOREIGN KEY (group_id)
        REFERENCES group_categories(id) ON DELETE SET NULL
Referenced by:
    TABLE "project_type_category" ... FOREIGN KEY (category_id)
        REFERENCES categories(id) ON DELETE CASCADE
Triggers:
    trg_category_slug_registry ...
    update_category_updated_at BEFORE UPDATE ... EXECUTE FUNCTION update_updated_at_column()
```

```
Table "public.project_type_category"  (bảng join, không có deleted_at — hard delete only)
 project_type_id | uuid | not null
 category_id     | uuid | not null
 created_at      | timestamptz | not null
PRIMARY KEY (project_type_id, category_id)
FK category_id -> categories(id) ON DELETE CASCADE
FK project_type_id -> project_type(id) ON DELETE CASCADE
```

`update_updated_at_column()` (trigger function, áp dụng cho CẢ `group_categories` VÀ
`categories`):
```sql
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
```

## 4. Ba điểm khác biệt thật sự so với `brand` — bắt buộc đọc trước khi copy code brand

### 4.1. KHÔNG có unique constraint trên `name`

`brand` có `brands_name_key` (UNIQUE toàn cục trên `name`, kể cả row đã soft-delete) — đó là
một gotcha đã ghi trong `docs/brand.md`. `group_categories` **KHÔNG có** constraint tương tự
trên `name` (đã confirm bằng `\d group_categories` ở trên — chỉ có `groups_pkey` và
`group_categories_slug_unique_active`). Đừng bịa thêm validate/xử lý cho một constraint không
tồn tại. Tự chạy lại `\d group_categories` để xác nhận trước khi kết luận trong docs.

### 4.2. `updated_at` được set tự động bởi trigger BEFORE UPDATE — khác `brand`/`service-group`

`brand`/`service-group` KHÔNG có trigger auto-update `updated_at` — mọi mutator trong domain
Go tự set `updatedAt = time.Now()`, và repository UPDATE query set `updated_at = $N` bằng tay.
Với `group_categories` VÀ `categories`, có trigger `update_groups_updated_at` /
`update_category_updated_at` chạy `NEW.updated_at = now()` TRƯỚC MỌI UPDATE, tự động ghi đè
bất kể Go gửi giá trị gì. Điều này **không phải bug, không cần né tránh** — cứ viết domain
mutator/repository UPDATE giống hệt pattern của `brand` (vẫn set `updatedAt` trong Go, vẫn
truyền vào query như bình thường), vì:
- Query vẫn dùng `RETURNING ...` để lấy lại row thật sau UPDATE (đúng pattern đã có), nên giá
  trị `updated_at` trả về cho client luôn là giá trị THẬT sự nằm trong DB sau trigger chạy,
  không phải giá trị Go tính trước đó.
- Chỉ cần lưu ý trong `docs/group.md`: đừng ngạc nhiên nếu viết integration test so sánh
  `updatedAt` Go tính trước UPDATE với `updatedAt` trả về sau UPDATE mà thấy khác — đó là
  trigger, không phải lỗi code.

### 4.3. `SoftDelete` (delete) có cascade 2 tầng, KHÔNG đơn giản như service-group

Đây là điểm khác biệt **quan trọng nhất**, đọc kỹ code TS thật ở
`modules/group/infrastructure/groupRepo.ts` hàm `delete()`:

1. Soft-delete row `group_categories` (set `deleted_at = now()`).
2. Lấy tất cả `categories` đang active (`deleted_at IS NULL`) có `group_id` = id vừa xoá.
3. **Soft-delete luôn các `categories` đó** (set `deleted_at = now()`) — KHÔNG PHẢI chỉ null
   hoá `group_id` như cách `service-group` null hoá `services.group_id`! Đây là khác biệt lớn
   nhất so với pattern `service-group` đã dùng — đừng copy pattern "null hoá FK" của
   service-group vào đây, vì hành vi thật của TS là soft-delete category, không phải gỡ liên
   kết.
4. Với các `categories` vừa bị soft-delete ở bước 3, **hard-delete** (xoá thật) toàn bộ dòng
   trong bảng join `project_type_category` có `category_id` nằm trong danh sách đó.

TS hiện tại chạy 4 bước này bằng **4 lệnh Supabase tuần tự, KHÔNG có transaction** — nếu bước
3 fail sau khi bước 1-2 đã chạy xong, dữ liệu sẽ ở trạng thái nửa vời (group đã xoá nhưng
category con vẫn sống nhăn, hoặc category đã xoá nhưng join table chưa dọn). Đây là một rủi ro
thật, đúng loại bug mà `service-group`'s `SoftDelete` đã né bằng cách gom vào 1 transaction
(`pool.Begin(ctx)` ... `tx.Commit(ctx)`). **Bắt buộc**: viết `SoftDelete` của Go module này
chạy toàn bộ 4 bước trên trong **một transaction duy nhất** (`tx.Exec` cho từng bước, dùng
`SELECT ... FOR UPDATE` hoặc đơn giản chỉ `SELECT id FROM categories WHERE group_id = $1 AND
deleted_at IS NULL` rồi collect ids, tương tự cách TS làm nhưng atomic). Đây KHÔNG phải bịa
thêm tính năng — đây là bug fix giống hệt tinh thần của `service-group`'s SoftDelete, chỉ khác
ở chỗ cascade sâu hơn 1 tầng và có thêm 1 bảng join cần dọn.

Viết integration test xác nhận CẢ 3 hiệu ứng sau khi `SoftDelete(groupID)`:
- `group_categories.deleted_at` được set.
- `categories` từng thuộc group đó (đang active) cũng bị soft-delete.
- `project_type_category` không còn dòng nào tham chiếu tới các category vừa xoá.
Tự tạo dữ liệu test (1 group, 1-2 category thuộc group đó, 1 project_type + 1 dòng join), tự
dọn dẹp sau test giống các integration test khác trong `internal/brand`/`internal/service-group`.

### 4.4. Resurrect-on-create — GIỮ LẠI dù constraint không bắt buộc

Giống `brand`, `slug` chỉ unique trong phạm vi row chưa xoá (partial index
`group_categories_slug_unique_active`), nên về mặt constraint, `INSERT` thường (không cần
resurrect) vẫn chạy được khi tái sử dụng slug của 1 row đã soft-delete — y hệt kết luận đã ghi
trong `docs/brand.md`.

**NHƯNG** — khác với `brand`, code TS hiện tại của `group`
(`modules/group/infrastructure/groupRepo.ts`, hàm `create()`) **đã chủ động cài resurrect
logic**: tìm row soft-delete có cùng slug, nếu có thì UPDATE lại row đó (giữ nguyên `id`) thay
vì INSERT mới. Đây là hành vi nghiệp vụ có chủ đích (giữ nguyên ID/lịch sử khi "tạo lại" một
group đã xoá trùng slug), không phải code thừa. **Phải giữ nguyên hành vi này** trong Go
(giống `service-group`'s `Create`, dù lý do kỹ thuật ép buộc resurrect ở service-group khác lý
do nghiệp vụ ở đây) — đừng đơn giản hoá thành plain `INSERT` như đã làm với `brand` (ở `brand`
việc bỏ resurrect là ĐÚNG vì TS gốc của brand không hề có resurrect logic; ở `group` thì
NGƯỢC LẠI, TS gốc có resurrect logic nên Go phải giữ).

## 5. Gotcha pgx đã biết trước — áp dụng ngay, đừng để tự phát hiện lại

`internal/platform/db` chạy pool ở `pgx.QueryExecModeSimpleProtocol` (fix PgBouncer, xem
`docs/contact.md`). Chế độ này encode/decode `jsonb` tốt cho `json.RawMessage`/`[]byte`
(dùng cho cột `content`) nhưng **KHÔNG tự encode được một Go struct slice tuỳ ý** như
`[]domain.FAQItem` — sẽ lỗi `unable to encode ... into text format for unknown type (OID 0)`.
Và nếu tự `json.Marshal` ra `[]byte` thường rồi truyền vào, pgx sẽ encode nó như bytea literal
chứ không phải JSON text, Postgres sẽ báo `invalid input syntax for type json`.

**Cách làm đúng** (đã verify chạy được trong `internal/brand/infrastructure/postgres_repository.go`,
copy nguyên xi cách làm, đổi tên type):
- Viết hàm `marshalFAQ(faq []domain.FAQItem) (json.RawMessage, error)` — return type BẮT BUỘC
  là `json.RawMessage` (named type), KHÔNG được là `[]byte` trần, vì pgx chỉ đặc cách nhận diện
  `json.RawMessage` cho cột jsonb.
- Viết hàm `unmarshalFAQ(raw []byte) ([]domain.FAQItem, error)` để giải mã lúc scan.
- `content` (đã là `json.RawMessage` sẵn từ domain) thì cứ truyền thẳng, không cần marshal
  tay, vì nó tự nhiên đã đúng type.

## 6. Cutover `elc-tem` — người tiêu thụ thật đã tìm được (grep xong), xử lý đúng những chỗ này

Đã grep toàn bộ `elc-tem` tìm nơi import `modules/group/application`,
`modules/group/infrastructure`, `groupRepo`, và `modules/group/domain` — bạn vẫn PHẢI tự chạy
lại các lệnh grep này sau khi sửa xong actions.ts, TRƯỚC KHI xoá `application`/`infrastructure`,
vì code có thể đã thay đổi từ lúc viết prompt này:

```bash
grep -rln "modules/group/application\|modules/group/infrastructure\|groupRepo\b" \
  --include="*.ts" --include="*.tsx" . | grep -v "^modules/group/" | grep -v __tests__
grep -rln "modules/group/domain\|from \"@/modules/group\"" \
  --include="*.ts" --include="*.tsx" . | grep -v "^modules/group/" | grep -v __tests__
```

**Nơi PHẢI sửa (import trực tiếp `groupRepo`/`application`, bắt buộc chuyển qua
`getGroupsAction()`):**
- `app/(public)/tin-tuc/[slug]/page.tsx` — dòng có
  `import { getGroups } from "@/modules/group/application"` và
  `import { groupRepo } from "@/modules/group/infrastructure/groupRepo"`, gọi
  `getGroups(groupRepo)`. Đổi thành gọi `getGroupsAction()` (giống pattern
  `getContactsAction()`/`getBrandsAction()` đã dùng ở các trang khác), lấy `.data`.

**Nơi CHỈ import type (`Group`/`CreateGroupInput`/`UpdateGroupInput`) — GIỮ NGUYÊN, không sửa,
đây là domain type thuần TS, đúng tiền lệ `contact`'s `getContactHref` / `brand`'s
`brand-showcase.tsx`:**
- `app/(admin)/admin/(dashboard)/group-categories/page.tsx` — import component
  `GroupManagement`, không phải data layer.
- `modules/category/domain/types.ts` — import type `Group` để định nghĩa
  `CategoryWithGroup.group?: Group | null`.
- `modules/catalog/presentation/components/form/ProductGeneralTab.tsx` — chỉ nhận `groups:
  Group[]` qua prop, không tự fetch.
- `modules/catalog/infrastructure/resolveProductPath.ts` — import type `Group`, NHƯNG file
  này CŨNG tự query bảng `group_categories` trực tiếp qua `supabase-js` (dòng có
  `.from("group_categories")`) để resolve product path — đây là phạm vi của `catalog` (chưa
  migrate), giống hệt tiền lệ đã ghi trong `docs/brand.md` mục "Not migrated" — **không sửa,
  chỉ ghi chú lại trong `docs/group.md`** kèm lý do (catalog tự migrate sau sẽ dọn).

**Dead code tìm được — ghi vào docs, không cần thêm tính năng để "lấp đầy":**
- `getGroupById` (trong `modules/group/application/index.ts`) có **0 caller** ở bất kỳ đâu
  trong `elc-tem` — không hề được expose thành Server Action trong `presentation/actions.ts`
  (giống hệt tình huống `getBrandBySlug` đã ghi trong `docs/brand.md`). Vẫn nên implement
  `GetByID` ở tầng Go domain/application/infrastructure cho đầy đủ REST resource (giống các
  module khác), nhưng KHÔNG bắt buộc thêm `getGroupByIdAction` vào `actions.ts` nếu không có
  nơi nào gọi — ghi rõ lý do trong `docs/group.md`.

**Response shape phải giữ nguyên khi viết lại `presentation/actions.ts`:**
`deleteGroupAction` hiện tại trả về `{ error: null }` (KHÔNG có field `success`), khác với
`brand`'s `deleteBrandAction` trả `{ success: true, error: null }`. Đọc kỹ
`modules/group/presentation/components/GroupManagement.tsx` xem nó destructure field gì từ
kết quả `deleteGroupAction` trước khi quyết định giữ hay đổi shape — nếu component chỉ check
`error`, giữ nguyên `{ error }`, đừng tự ý thêm `success` vì "thấy brand có".

**`revalidatePath`/`revalidateTag` phải giữ nguyên đúng như bản cũ** (đọc
`modules/group/presentation/actions.ts` hiện tại): path là `/admin/group-categories` (không
phải `/admin/groups`), và có `revalidateTag("products", { expire: 0 })` thêm vào (khác
`brand` chỉvrevalidate `"layout"`). Copy nguyên các revalidate call, chỉ đổi phần fetch logic
sang gọi Go.

**Sau khi sửa xong, chỉ xoá `modules/group/{application,infrastructure}` khi cả 2 lệnh grep ở
trên đều rỗng.** Cập nhật `modules/group/index.ts` (barrel) bỏ `export * from "./application"`
và `"./infrastructure"`. Module `group` không có thư mục `__tests__` nên không cần dọn test cũ.

## 7. HTTP route — mount path

Mount route Go tại `/groups` (khớp tên module `group`, theo đúng quy ước
`service-group` → `/service-groups`, `brand` → `/brands` — route path theo TÊN MODULE, không
theo tên bảng SQL `group_categories`). Endpoints tối thiểu, khớp REST shape đã dùng cho
`brand`/`service-group`: `GET /groups`, `GET /groups/{id}`, `GET /groups/slug/{slug}`,
`POST /groups`, `PUT /groups/{id}`, `DELETE /groups/{id}` (soft delete, chạy cascade transaction
ở mục 4.3), `POST /groups/{id}/restore`.

## 8. Sau khi xong — checklist bắt buộc trước khi báo cáo hoàn thành

- [ ] `cd elc-go && gofmt -l .` rỗng, `go vet ./...` sạch, `go build ./...` sạch.
- [ ] `go test ./internal/group/...` pass (domain + application, fake repo, không cần DB
      thật).
- [ ] `go test -tags=integration ./internal/group/infrastructure/...` pass với DB thật —
      **đặc biệt phải có test cho cascade soft-delete ở mục 4.3** (3 hiệu ứng), và test cho
      resurrect-on-create ở mục 4.4.
- [ ] Chạy thử server thật (`air` hoặc `go run ./cmd/server`, chú ý PORT có thể trùng nếu đã
      có instance chạy sẵn — kiểm tra `lsof -i :8090` trước), gọi thử `curl` vào `/groups`,
      `/groups/{id}` xác nhận response JSON đúng shape snake_case.
- [ ] Bên `elc-tem`: `npx tsc --noEmit -p tsconfig.json` sạch, `npx eslint <các file đã
      sửa>` sạch, `npx vitest run modules/group` pass (nếu còn test nào sau khi dọn).
- [ ] `pnpm build` chạy hết, không lỗi prerender (bài học từ PR #588: có Server Action nào
      gọi Go mà THIẾU `try/catch` quanh `fetch` sẽ làm sập cả build lúc `GO_API_URL` là
      `undefined` trong CI — kiểm tra kỹ MỌI function mới viết trong
      `modules/group/presentation/actions.ts` đều có try/catch bọc `fetch`, không có ngoại lệ
      nào, kể cả những function tưởng chừng đơn giản).
- [ ] Viết `docs/group.md` theo đúng cấu trúc 4 file đã có (đặc biệt: mô tả rõ cascade
      soft-delete ở mục 4.3, khác biệt updated_at trigger ở mục 4.2, và phần "Not migrated"
      cho `resolveProductPath.ts`). Cập nhật bảng trạng thái ở `docs/README.md` và
      `ARCHITECTURE.md` (thêm dòng `group` = migrated, xoá "group" khỏi câu liệt kê ví dụ ở
      mục "Migration order" nếu câu đó liệt kê tên module theo dạng ví dụ chưa migrate).

Đừng tự ý thêm tính năng ngoài phạm vi (không thêm cache/pagination/rate-limit không được yêu
cầu). Không rename biến/pattern đã thống nhất trong 4 module trước. Nếu gặp điểm mơ hồ không
chắc (ví dụ constraint nào đó psql trả về khác với mô tả trong prompt này), DỪNG LẠI, ghi rõ
phát hiện thực tế vào `docs/group.md`, và chọn phương án an toàn nhất (giữ nguyên hành vi TS
cũ) thay vì tự sáng tạo hành vi mới.
