# RFC: Hạ tầng VPS, Next.js, Go API: các vấn đề đã xác nhận + kế hoạch fix

- **Status**: In progress — Phase 0 (V1-V3, V5 một phần), Phase 1 (G2, G3, G5) và G7 (IP khách) đã xong và deploy (xem mục 9). Phase 2 (metrics, Prometheus, Grafana, process-exporter) đang làm (xem cuối mục 9), Phase 3 (Next RAM/pm2) và Phase 4 chưa bắt đầu. Cập nhật Status trong cùng commit/PR khi làm xong từng phase (xem `CLAUDE.md`).
- **Date**: 2026-10-08
- **Phạm vi**: VPS `vps-elc` (103.179.189.179), `elc-go` (repo này), `elc-temp` (Next.js frontend)
- **Cách dùng file này**: mỗi vấn đề có mã (V/G/N/F) + bằng chứng + hướng fix + cách kiểm tra. Phần "Kế hoạch" ở cuối tham chiếu lại các mã này.

## 0. Tóm tắt (BLUF)

VPS dựng được Prometheus + Grafana (~480MB), nhưng đang có 3 chỗ yếu phải vá trước: không có swap, cổng 8090/3000 đang mở ra internet, ổ đĩa bị Docker build cache chiếm 37GB. Code Go chưa có metrics, access log hay endpoint health riêng. Feature flag là việc riêng, tạm hoãn.

Mục tiêu ban đầu của đợt này: thêm observability (metrics/log/traces bằng Prometheus + Go) và feature flag. Khi audit hạ tầng để xem có dựng nổi không thì lòi ra các vấn đề dưới đây, nên chúng phải được xử lý trước hoặc cùng lúc.

## 1. Hiện trạng đã đo (2026-10-08)

Tất cả số liệu lấy bằng lệnh chỉ-đọc trên VPS và đọc code, không sửa gì.

| Hạng mục | Giá trị |
|---|---|
| OS / CPU / RAM / swap | Ubuntu 24.04, 2 vCPU, 3915MB, **swap 0** |
| RAM `MemAvailable` | ~1550MB (gồm ~800MB page cache) |
| Đĩa | 77GB, dùng 47GB (62%) |
| Uptime / load | 205 ngày / 0.09 |
| Process lớn nhất | `next-server` (pm2) RSS **1.45GB** |
| Container | `elc-go` 43MB, `elc-postgres` 107MB, `elc-redis` 4MB |
| Postgres / Redis | 17-alpine / 7-alpine, chỉ nghe trong mạng Docker |
| Reverse proxy | nginx trên host (không trong compose) -> `localhost:3000` |

Luồng request đúng thiết kế:

```
Internet -> nginx :443 -> Next.js (pm2) :3000 -> Go API (docker) :8090 -> Postgres / Redis
                          GO_API_URL=http://localhost:8090 (server-side)
```

## 2. Vấn đề hạ tầng VPS

### V1. Không có swap, Next.js ăn 1.45GB RAM [Cao]

Bằng chứng:
- `free -m`: Swap 0.
- `next-server` RSS 1,450,976kB, trong đó `Pss_Anon` 1,427,092kB và `Pss_File` chỉ 15MB, tức gần như toàn bộ là bộ nhớ cấp phát (không phải code/thư viện).
- Vùng `[heap]` (glibc malloc, không phải V8) là 746MB, kèm nhiều vùng arena 40-64MB.
- `sharp` có trong `node_modules`; `.next/cache/images` có 21,554 file / 803MB; log lỗi có nhiều dòng `upstream image response timed out` từ R2.
- Process không có `NODE_OPTIONS`, `MALLOC_ARENA_MAX`, hay giới hạn bộ nhớ nào.
- Không có OOM kill nào trong 30 ngày; memory pressure (PSI) thấp. Nên đây là rủi ro, chưa phải sự cố đang xảy ra.

Hậu quả: Next phình thêm ~1GB thì kernel OOM-kill một process, nạn nhân có thể là Postgres.

Chưa biết (phải đo, không đoán): thủ phạm cụ thể là sharp, Buffer hay thứ khác; "leak" thật hay "đạt đỉnh rồi giữ nguyên". Ứng viên số 1 là phân mảnh glibc malloc do resize ảnh, nhưng chưa được xác nhận.

Hướng fix (theo thứ tự):
1. Tạo swapfile 2GB, `vm.swappiness=10`. Đây là bước duy nhất của V1 làm trước khi đo.
2. Ghi RSS của Next theo thời gian (process-exporter, xem mục 5) trong 48-72h để phân biệt leak với đỉnh. **Phải xong bước này trước khi đặt trần ở bước 3.**
3. Đặt `max_memory_restart` **dựa trên số đo**, không đặt trước. Process đang ổn định ở 1.45GB, nên đặt `1G` sẽ khiến pm2 restart Next liên tục sau mỗi lần warm-up (mỗi lần là vài giây downtime ở fork mode). Nếu đo ra là đỉnh ổn định ~1.5GB thì trần hợp lý là ~2GB; nếu là leak thì trần là biện pháp tạm.
4. `MALLOC_ARENA_MAX=2` (hoặc jemalloc); thử xem RSS có giảm không. Đặt qua template trong `elc-temp` `deploy.yml` (xem lưu ý bên dưới), không sửa tay trên VPS.
5. Xem lại cấu hình `next/image` (số kích thước, TTL cache) nếu đo ra sharp là thủ phạm.

Lưu ý: `ecosystem.config.js` được **sinh lại mỗi lần deploy** (xem V4/V5), nên mọi thay đổi env/`max_memory_restart` sửa tay trên VPS sẽ bị ghi đè ở deploy kế tiếp. Các thay đổi này phải sửa trong bước `python3 -c ...` của `elc-temp/.github/workflows/deploy.yml`, tức là có đụng code (repo `elc-temp`).

Kiểm tra: `free -m` thấy swap 2GB; RSS Next sau 24-48h không vượt trần; không có OOM trong `journalctl -k`; không có chuỗi restart do `max_memory_restart` trong `pm2 logs`.

### V2. Cổng 8090 (Go) và 3000 (Next) đang public, bỏ qua nginx [Cao]

Bằng chứng:
- `curl http://103.179.189.179:8090/brands` từ ngoài -> HTTP 200. `:3000` -> 200.
- `docker-compose.yml`: `ports: "8090:8090"` (bind 0.0.0.0).
- UFW đang `allow 3000`; chain `DOCKER-USER` trống; iptables có `DNAT ... --dport 8090 -> 172.18.0.4:8090` cho mọi nguồn.
- Cơ chế: Docker DNAT ở PREROUTING nên gói không bao giờ vào chain INPUT của UFW, vì vậy `ufw deny 8090` không có tác dụng. ([giải thích](https://www.ssdnodes.com/learn/docker-ports-bypass-ufw), [4 cách fix](https://www.virtua.cloud/learn/en/tutorials/docker-ufw-firewall-fix-vps))

```
đang:   Internet --> nginx :443 --> Next :3000 --> Go :8090        (đúng)
        Internet --> Next :3000 trực tiếp                          (lỗ hổng)
        Internet --> Go   :8090 trực tiếp                          (lỗ hổng)
```

Hậu quả: bất kỳ ai gọi thẳng API Go, bỏ qua nginx (không có TLS, giới hạn body, header proxy). Nếu thêm `/metrics` lên cùng cổng thì số liệu nội bộ cũng lộ.

Đã xác nhận an toàn để đóng: Next gọi Go qua `localhost:8090`; 3 workflow phụ (`migrate-images`, `detect-attribute-anomalies`, `sync-ai-pricing`) SSH vào VPS chứ không gọi qua cổng này.

Hướng fix:
- compose (repo `elc-go`): `ports: ["127.0.0.1:8090:8090"]`.
- Next (repo `elc-temp`): thêm `HOSTNAME=127.0.0.1` vào env trong bước sinh `ecosystem.config.js` của `deploy.yml`. **Không sửa tay trên VPS**: file đó bị sinh lại mỗi deploy, sửa tay sẽ mở lại :3000 ở deploy kế tiếp. Xóa rule UFW cho 3000.
- Không cần DOCKER-USER/ufw-docker vì không có service nào cần public trực tiếp.

Đã kiểm tra trước khi đóng (tránh làm hỏng thứ đang chạy):
- Trên VPS `localhost` chỉ resolve ra `127.0.0.1` (không có `::1`), Node 20.20.2; nginx `proxy_pass http://localhost:3000` và Next `GO_API_URL=http://localhost:8090` đều đi qua IPv4 loopback, không bị lệch khi bind `127.0.0.1`.
- Không có workflow nào (elc-go hoặc elc-temp) và không có cấu hình nginx nào gọi qua IP public `103.179.189.179` hay cổng 8090/3000 từ ngoài; các lần gọi `localhost:8090` (smoke check trong `elc-temp` `deploy.yml`, `ci.yml`) đều chạy trên chính VPS hoặc runner.
- Sau khi đổi, vẫn phải kiểm tra lại bằng test thật bên dưới, đặc biệt đường Next -> Go.

Kiểm tra: từ máy ngoài `curl :8090` và `curl :3000` phải "connection refused"; site qua `https://dienmayelc.com.vn` vẫn bình thường; trang có gọi Go API (danh sách sản phẩm) vẫn render đủ dữ liệu.

### V3. Docker (image + build cache) chiếm 37GB đĩa, log container không giới hạn [Trung bình]

Bằng chứng (`du`):

```
/ 47GB
+-- /var/lib/containerd  37GB   image cũ + 355 bản build cache (Docker)
+-- /var/log/journal      4GB   journald (đang chạm mức trần mặc định ~4GB)
+-- /root                1.3GB
+-- /var/www/elc-tem     892MB  (.next/cache/images 803MB)
+-- volume Postgres      215MB  <- dữ liệu thật, rất nhỏ
```

- Nguồn gốc: `deploy.yml` của elc-go rsync source lên VPS rồi `docker compose build` ngay trên VPS (kéo `golang:1.26` 1.29GB, `golang:1.26-alpine`, ...) và không dọn sau build.
- Docker log: `json-file` không có `max-size`/`max-file` (container hiện mới tạo lại nên chưa phình).
- Số "reclaimable" của `docker system df` tự mâu thuẫn (Images 36GB, Build cache 3.4GB), nên chưa biết dọn sẽ giải phóng bao nhiêu -> phải đo trước/sau.
- Ổ còn 30GB nên chưa gấp; mỗi deploy đang cộng thêm.

Hướng fix:
1. Dọn một lần (liệt kê trước, user duyệt): `docker builder prune`, `docker image prune -a`; ghi dung lượng trước/sau.
2. Chặn tái diễn: thêm vào cuối bước deploy **cả hai** lệnh `docker image prune -f` (mỗi `compose build` để lại image Go cũ dạng dangling) và `docker builder prune -f --filter until=72h` (build cache). Chỉ dọn build cache là không đủ: theo `docker system df`, nhóm Images mới là nhóm lớn (38.6GB, 36GB "reclaimable"), còn Build Cache 36.9GB nhưng chỉ 3.4GB "reclaimable"; hai con số này chồng lấn/mâu thuẫn nên chưa tách được chính xác từng nhóm chiếm bao nhiêu trong 37GB.
3. journald: mức hiện tại ~4GB đúng bằng trần mặc định (10% dung lượng FS, tối đa 4GB), tức là có giới hạn nhưng quá rộng. Đặt `SystemMaxUse=500M` + `journalctl --vacuum-size=500M`.
4. compose: `logging: {driver: json-file, options: {max-size: "10m", max-file: "3"}}` cho cả 3 service.
5. Dài hạn (tách riêng, xem G6): build image trên GitHub Actions, VPS chỉ pull.

Kiểm tra: `df -h /` giảm rõ rệt; sau 2-3 deploy `du -sh /var/lib/containerd` không tăng tuyến tính.

### V4. pm2 chạy `fork` 1 instance, trái với `deploy.yml` (cluster 2 instance) [Trung bình]

Bằng chứng:
- `ecosystem.config.js` trên VPS: `instances: 2, exec_mode: cluster`. Process thật: `fork_mode`, 1 process (`pm2 describe`, `/proc/<pid>/environ`: `exec_mode=fork_mode`).
- `deploy.yml` của elc-temp: `pm2 reload ecosystem.config.js ... || (pm2 delete && pm2 start)`. `reload` thành công nhưng giữ nguyên fork mode, nên nhánh fallback không bao giờ chạy -> chưa từng chuyển sang cluster.
- Comment trong workflow tuyên bố "zero-downtime thật" là sai với thực tế: mỗi deploy Next có downtime vài giây.
- `pm2.log`: 7-18 lần exit/ngày từ 22/9, trùng nhịp push -> restart chủ yếu do deploy, không phải crash loop (`unstable restarts 0`). Lịch sử có 24 lần SIGKILL + 1 SIGABRT chưa được soi từng cái.

Quyết định phụ thuộc V1 (RAM):

```
Hướng 1: giữ fork, 1 instance        Hướng 2: cluster 2 instance
  Next ~1.45GB                         Next ~2.9GB -> còn ~0.1GB trống
  còn ~1.5GB cho obs stack             không còn chỗ cho Postgres/obs
  chấp nhận downtime vài giây/deploy   chỉ khả thi nếu cắt RAM mỗi worker
                                       xuống ~600MB (V1) trước
```

Đề xuất: **Hướng 1** cho tới khi V1 đo xong và cắt được RAM. Sửa `deploy.yml`/comment cho khớp thực tế thay vì giữ lời hứa sai.

Kiểm tra: `pm2 describe elc-tem` khớp với workflow; comment trong workflow không còn nói "zero-downtime" nếu vẫn fork.

### V5. Secret dạng plaintext trong `ecosystem.config.js` trên VPS [Thấp-Trung bình]

- `GEOCODING_API_KEY` (và `GOOGLE_SERVICE_ACCOUNT_JSON`, `NEXT_SERVER_ACTIONS_ENCRYPTION_KEY`) được ghi plaintext vào `/var/www/elc-tem/ecosystem.config.js` mỗi deploy.
- Trong lúc audit, `GEOCODING_API_KEY` đã bị in ra terminal một lần (nằm trong transcript phiên làm việc này).
- Hướng fix: `chmod 600` file; cân nhắc đổi (rotate) key Geocoding; xem xét nạp secret từ file `.env` quyền 600 thay vì nhét vào ecosystem.

## 3. Vấn đề code Go (`elc-go`)

### G1. Không có metrics [Cao, mục tiêu chính]

- `/metrics` trả 404; `go.mod` không có `prometheus/client_golang` hay OTel.
- Không đo được số request, latency, lỗi, trạng thái pool DB.

Hướng fix:
- `internal/platform/metrics/`: registry riêng + chi middleware (`http_requests_total{method,route,status}`, histogram `http_request_duration_seconds`, gauge in-flight).
- **Label `route` lấy từ `chi.RouteContext(r.Context()).RoutePattern()` sau khi `next.ServeHTTP` chạy**, không dùng raw path (raw path làm nổ cardinality -> Prometheus ăn hết RAM).
- Listener riêng cho `/metrics` (ví dụ `:9091`), không mount lên router public 8090. **Bind `0.0.0.0` bên trong container và KHÔNG publish cổng này ra host**: Prometheus nằm trong cùng mạng compose và scrape `elc-go:9091`. Bind `127.0.0.1` bên trong container là sai, vì đó là loopback riêng của container, Prometheus không với tới được. Không dùng cổng 9090 (là cổng mặc định của Prometheus), cũng tránh 9100 (node_exporter).
- Collector: Go runtime/process, `pgxpool.Stat()` (acquired/idle/max, wait), vài business metric ít series (inquiry mới, gọi AI theo provider, lỗi push GA4).

### G2. Không có access log [Cao]

- `internal/platform/httpserver/router.go` chỉ có `middleware.RequestID` và `recoverer`. Log chỉ ghi khi panic.
- Không biết route nào chậm / trả lỗi gì.

Hướng fix: middleware access log (zap): method, route pattern, status, duration, bytes, `request_id`, IP thật (dùng `client_ip.go`). Bỏ qua `/healthz`, `/readyz` (`/metrics` nằm ở listener riêng nên không đi qua middleware này). Gắn `request_id` vào logger của context để mọi dòng log trong cùng request đều có.

### G3. Healthcheck gọi `/brands` mỗi 5 giây [Trung bình]

- `Dockerfile`: `HEALTHCHECK --interval=5s ... curl -sf http://localhost:8090/brands`. Lý do lịch sử: commit `9eb425d` (2026-07-05) cần health check cho pipeline deploy và chọn `/brands` vì là GET public có sẵn, chưa có endpoint health.
- `/brands` chạy `GetBrands` = query DB thật -> ~17,280 query thừa mỗi ngày, và sẽ làm nhiễu metrics.
- `elc-temp` `deploy.yml` cũng chờ Go bằng `curl /brands` (30 lần x 2s).

Hướng fix: thêm `/healthz` (chỉ báo process sống, không đụng DB) và `/readyz` (ping DB); đổi `HEALTHCHECK`, đổi bước chờ trong workflow Next; loại cả hai khỏi metrics/access log.

### G4. Cấu hình `pgxpool` và comment cũ trong `db.go` [Thấp, ngoài phạm vi chính]

- `internal/platform/db/db.go` chưa set `MaxConns`/`MinConns`/lifetime; chưa có stats (xử lý ở G1).
- Comment và `QueryExecModeSimpleProtocol` còn từ thời Supabase PgBouncer, trong khi giờ Postgres tự host kết nối trực tiếp. Có thể đang trả chi phí không cần thiết. Chỉ ghi nhận, cần đo/benchmark trước khi đổi.

### G5. `http.Server` chưa có timeout [Thấp]

- `cmd/server/main.go`: `&http.Server{Addr, Handler}`, không có `ReadHeaderTimeout`/`ReadTimeout`/`WriteTimeout`/`IdleTimeout`. Kết hợp V2 (API public), client chậm có thể giữ kết nối vô hạn. Sau khi đóng V2 thì rủi ro giảm. Chỉ ghi nhận; thêm `ReadHeaderTimeout` là thay đổi nhỏ, đi kèm G2.

### G6. Build image ngay trên VPS [Trung bình, dài hạn]

- Mỗi push `main` -> rsync + `docker compose build` trên VPS 4GB, tranh RAM/CPU với Next, và là nguồn gốc của V3.
- Hướng fix: build image trên GitHub Actions, push lên GHCR, VPS chỉ `docker compose pull && up -d`. Tách riêng, không chặn các phase đầu.

### G7. Go không bao giờ thấy IP thật của khách: rate limit theo IP đang dùng chung một "xô" [Cao, có sẵn từ trước]

Phát hiện khi chuẩn bị access log (2026-10-08), xác nhận bằng dữ liệu thật:
- Bảng `inquiries`: 40 dòng, **1 giá trị `source_ip` duy nhất (100%)**. Chỉ đếm tổng hợp, không đọc IP.
- Nguyên nhân: nginx (`/etc/nginx/sites-enabled/dienmayelc.com.vn`) không đặt `X-Forwarded-For`/`X-Real-IP`; code Next (`app`, `modules`, `shared`, `proxy.ts`) không đọc hay chuyển tiếp header IP nào; Go gọi qua `localhost:8090` nên `ClientIP()` rơi về `r.RemoteAddr`, luôn là một địa chỉ nội bộ giống nhau. Trước nginx còn có Cloudflare (IP kết nối tới nginx là dải Cloudflare), nên IP thật nằm ở `CF-Connecting-IP`.
- Hậu quả: mọi giới hạn theo IP dùng chung một xô cho toàn bộ khách: `inquiry` (tạo, click, upload), đăng nhập Google, magic link (`email|ip`), `ai/chat`, event. Một vài khách đủ để chặn tất cả người còn lại; và `source_ip` lưu trong DB vô nghĩa.
- Hướng fix (cần phối hợp 3 chỗ, làm riêng, không gộp vào Phase 1): (1) nginx dùng `real_ip` với dải IP Cloudflare hoặc chuyển `CF-Connecting-IP`; (2) Next đọc header đó ở server action/route handler và gắn `X-Forwarded-For` khi gọi Go; (3) `ClientIP()` chỉ tin header khi request đến từ loopback/Docker gateway. Rà các `limiter` xem ngưỡng có hợp lý khi đã tách theo IP thật.
- Thứ tự triển khai (bắt buộc, vì lý do bảo mật): (1) Go: `ClientIP()` chỉ tin `X-Forwarded-For` khi peer là loopback/mạng riêng và giá trị là IP hợp lệ; access log ghi `client_ip` (an toàn khi deploy: chưa ai gửi header). (2) nginx ghi đè `X-Forwarded-For` bằng IP thật TRƯỚC khi (3) Next chuyển tiếp nó; nếu Next chuyển tiếp trước, header do khách tự gửi sẽ lọt xuyên tới Go và vượt giới hạn tốc độ. (4) Kiểm tra `client_ip` trong access log ra nhiều IP thật khác nhau.
- **Tiến độ: XONG và đã kiểm chứng đầu-cuối trên production (2026-10-08).**
  - (1) Go `ad3e8ec`: `ClientIP()` + `client_ip` trong access log.
  - (2) nginx: `/etc/nginx/conf.d/cloudflare-realip.conf` (22 dải IP Cloudflare, `real_ip_header CF-Connecting-IP`) + `proxy_set_header X-Forwarded-For $remote_addr;` trong `sites-enabled/dienmayelc.com.vn`. Bản sao lưu: `/root/nginx-backups/g7-20261008-180148/`. `nginx -t` + reload, không downtime.
  - (3) Next `418abaa`: `clientIpHeaders()` trong `shared/lib/go-api.ts`, gắn vào inquiry create/upload/click, Google login, magic link, review create, events, và wishlist/recently-viewed (qua `forwardVisitorCookieHeader`).
  - (4) Kiểm chứng: `GET /api/wishlist` qua site -> Go ghi `client_ip` đúng IP Cloudflare báo cho cùng kết nối; thử giả mạo `X-Forwarded-For: 9.9.9.9` qua Cloudflare và gọi thẳng IP gốc kèm `CF-Connecting-IP`/`X-Forwarded-For` giả -> Go luôn thấy IP thật, không bao giờ thấy địa chỉ giả.
  - Còn lại / lưu ý: (a) 40 inquiry cũ có chung một `source_ip`, không backfill được. (b) Endpoint `ai/chat` cũng giới hạn theo IP nhưng không tìm thấy nơi nào trong `elc-temp` gọi nó nên chưa gắn header; nếu sau này frontend gọi thì phải dùng `clientIpHeaders()`. (c) Dải IP Cloudflare trong file nginx là danh sách tĩnh, cần làm mới nếu Cloudflare đổi (https://www.cloudflare.com/ips-v4, /ips-v6). (d) Sau khi giới hạn tách theo IP thật, ngưỡng của các limiter nên được rà lại vì trước đây không bao giờ phản ánh tải theo từng khách. (e) 3 test sẵn có trong `elc-temp` `modules/catalog/__tests__/domain/validators.test.ts` đang fail trên `main` từ trước (không do thay đổi này).

## 4. Vấn đề Next.js (`elc-temp`)

### N1. Image optimizer: 803MB cache, timeout từ R2

- Xem V1. `.next/cache/images` 21,554 file; lỗi `upstream image response timed out` từ `pub-d68f2955d9cf48a697d203e342f5ac2b.r2.dev` (504). Cần đo xem đây có phải nguồn RAM chính.

### N2. Log pm2 nhiễu và không xoay vòng

- `elc-tem-error.log` 19MB, trong đó ~24,000 dòng là `Failed to find Server Action "x"` (trông giống bot quét, không phải lỗi thật).
- Chưa thấy `pm2-logrotate`. Hướng fix: cài `pm2-logrotate`; cân nhắc chặn request Server Action không hợp lệ ở nginx nếu nhiễu tiếp diễn.

### N3. `deploy.yml` mô tả sai thực tế

- Xem V4. Phải sửa comment/logic cho khớp, kèm việc đổi bước chờ Go sang `/readyz` (G3).

## 5. Observability: stack và ngân sách RAM

| Thành phần | Giới hạn RAM | Ghi chú |
|---|---|---|
| Prometheus | 256MB | scrape 30s, retention 15d |
| Grafana | 192MB | bind 127.0.0.1, truy cập qua SSH tunnel |
| node_exporter | 32MB | metric host (RAM, disk, CPU, load) |
| process-exporter | ~30MB | RSS và thời điểm khởi động theo process (`next-server`, `postgres`, `dockerd`); cần cho theo dõi RSS Next (V1) và phát hiện restart |
| Tổng | ~510MB | cộng thêm vào bộ nhớ anonymous đang dùng |

Cách tính chỗ trống: `MemAvailable` ~1550MB đã gồm ~800MB page cache có thể thu hồi. Bộ nhớ anonymous mới (~510MB) trừ thẳng vào `MemAvailable`, nên sau khi dựng còn khoảng 1.0GB, và page cache bị ép nhỏ lại (Postgres đọc đĩa nhiều hơn, chậm hơn một chút). Swap **không** tăng chỗ trống; nó chỉ là nơi kernel đẩy bớt anonymous page khi thiếu, đóng vai lưới an toàn trước OOM kill. Vì vậy ngân sách này hẹp hơn vẻ ngoài và phải đo lại `MemAvailable` ngay sau khi dựng stack, không giả định.

Không dùng cAdvisor (100MB+), nên:
- Metric RSS theo process và phát hiện restart Next/Postgres/dockerd lấy từ process-exporter (không phải node_exporter, vì node_exporter chỉ có metric cấp host).
- Phát hiện restart của Go: metric `process_start_time_seconds` từ Go process collector (G1).
- Restart container nói chung: dựa vào `restart: unless-stopped` + `HEALTHCHECK`; chưa có metric riêng cho "container restart count". Nếu cần, thêm sau bằng textfile collector đọc `docker inspect`.

Chưa dựng Loki/Tempo trên VPS này: log đọc bằng `docker logs`/`grep request_id`; traces làm sau, đẩy sang Grafana Cloud nếu cần.

Alert tối thiểu: RAM available < 400MB, disk > 85%, 5xx rate > 2%, p95 latency > 1s, Go restart (`process_start_time_seconds` đổi), RSS `next-server` vượt ngưỡng đã chọn ở V1.

Mọi container mới có `mem_limit`, để nếu thiếu RAM thì nó chết trước chứ không kéo cả máy.

## 6. Feature flag (F1): tạm hoãn

- Hiện chưa có. `internal/settings` là `site_settings(key VARCHAR PRIMARY KEY, value TEXT)`, `GET /settings` public, `PUT` cần `CanManageSettings`. Không có kiểu bool/phần trăm/audit nên không đủ làm flag.
- Hướng đã thống nhất sơ bộ: module mới `internal/featureflag/`, bảng `feature_flags(key, enabled, rollout_pct, description, updated_by, updated_at)`, cache `atomic.Pointer[map]` refresh 30s, rollout bằng hash ổn định theo `visitor_id`/user, admin API tái dùng `RequireAuth` + `CanManageSettings`.
- Lưu ý phía Next: trang prerender tĩnh sẽ "đóng băng" giá trị flag lúc build; flag đổi UI phải fetch với `revalidate` ngắn hoặc đọc phía client.
- **Chưa làm** cho tới khi V1-V3, G1-G3 xong (cần metrics để rollout an toàn).

## 7. Kế hoạch (thứ tự thực thi)

```
Phase 0  VPS + deploy config    V1(swap) V2 V3 V5      <- rủi ro thấp; V2/V5 sửa deploy.yml
                                                          của elc-temp, không chỉ sửa tay VPS
Phase 1  Go: health + log       G3 G2 G5   (XONG, deployed)
Phase 2  Go: metrics + stack    G1 G4(stats) + Prometheus/Grafana/node_exporter
Phase 3  Next                   V1(đo, trần, MALLOC) V4 N1 N2 N3
Phase 4  Dài hạn / tùy chọn     G6 (build trên CI), traces, F1 (feature flag)
G7 (IP khách): XONG, deployed. TODO sau observability: workflow Sync AI pricing fail
```

Phụ thuộc:
- V2, V3 độc lập, làm ngay.
- V1(swap) phải xong trước khi dựng obs stack (Phase 2); `max_memory_restart` của V1 chỉ đặt sau khi đo xong (Phase 3).
- Mọi thay đổi env của Next (V1 `MALLOC_ARENA_MAX`, V2 `HOSTNAME`, V5 secret) đi qua `elc-temp/.github/workflows/deploy.yml` vì `ecosystem.config.js` bị sinh lại mỗi deploy; sửa tay trên VPS chỉ có tác dụng tới lần deploy kế tiếp.
- V4 phải đợi V1 đo xong mới quyết 1 hay 2 instance.
- G3 đổi cả `Dockerfile` (Go) và `deploy.yml` (Next) -> deploy theo thứ tự: Go ra `/healthz` trước, rồi mới đổi bước chờ bên Next.

Nguyên tắc thực thi:
- Mỗi lệnh sửa trên VPS: đưa lệnh cụ thể cho user duyệt trước khi chạy; ghi trạng thái trước/sau.
- Mỗi phase code: làm trên branch riêng, `go build ./... && go vet ./... && go test ./...` (Next: lint + test), dừng cho user xem diff, **không commit/push `main` khi chưa được xác nhận** (push `main` = deploy lên site live).
- Mỗi phase xong: cập nhật `Status` ở đầu file này trong cùng commit.

## 8. Câu hỏi còn mở

1. Phase 0 trên VPS: user tự chạy từng lệnh hay Claude chạy sau khi user duyệt từng bước?
2. Docker image prune: duyệt danh sách trước khi xóa (xóa image không đảo ngược được).
3. V4: chốt Hướng 1 (giữ fork) hay Hướng 2 (cluster, cần cắt RAM trước)? Đề xuất Hướng 1.
4. Grafana: chỉ SSH tunnel (đề xuất) hay mở subdomain có auth?
5. G6: gộp việc build trên CI vào đợt này hay tách RFC riêng?
6. V5: có đổi (rotate) `GEOCODING_API_KEY` không?

## 9. Tiến độ (cập nhật 2026-10-08)

Đã áp dụng trực tiếp trên VPS (đã verify site `https://dienmayelc.com.vn` 200 và 3 container healthy sau mỗi bước):

| Mã | Việc | Kết quả |
|---|---|---|
| V1 (phần swap) | swapfile 2GB, `vm.swappiness=10`, ghi vào `/etc/fstab` + `/etc/sysctl.d/99-swap.conf` | Swap 2047MB |
| V3 | `journald` `SystemMaxUse=500M` (`/etc/systemd/journald.conf.d/size.conf`) + vacuum | 3.9G -> 472M |
| V3 | `docker builder prune -a`, `docker rmi migrate/migrate:latest` | đĩa dùng 46G -> 12G; containerd 37G -> 2.8G |
| V2 (cổng 3000) | xóa rule UFW allow 3000 | từ ngoài timeout; nginx -> Next vẫn 200 |
| V5 | gỡ `GEOCODING_API_KEY` khỏi `ecosystem.config.js` + `/root/.pm2/dump.pm2`, xóa GitHub secret (sếp từ chối kích hoạt Maps Platform) | process pm2 đang chạy còn giữ key trong RAM tới lần deploy kế |

Quyết định không làm: `docker image prune -a` (cron backup 3h sáng dùng `docker run amazon/aws-cli`, `migrate-all.sh` pin `migrate/migrate:v4.18.1`).

Đã sửa trong working tree, **chưa commit/push** (push `main` = deploy live):
- `elc-go/docker-compose.yml`: bind `127.0.0.1:8090:8090`; log rotation `10m x 3` cho 3 service. Lưu ý: đổi cấu hình logging làm `compose up -d` recreate cả `postgres` và `redis` lúc deploy (vài giây; dữ liệu Postgres nằm trong volume).
- `elc-go/.github/workflows/deploy.yml`: thêm `docker image prune -f` + `docker builder prune -f --filter until=72h` cuối deploy.
- `elc-temp/.github/workflows/deploy.yml`: bỏ `GEOCODING_API_KEY`; thêm `HOSTNAME=127.0.0.1` cho Next.

Còn lại của Phase 0: deploy các thay đổi trên để đóng cổng 8090 (hiện vẫn public).

Việc user cần tự làm (tool bị chặn hoặc ngoài tầm): xóa API key trong Google Cloud Console (project `dien-may-elc`); xóa file local `ads-api/geocoding-api-config.json`.

Quan sát mới: code xin quyền vị trí trình duyệt (`elc-temp/shared/lib/geolocation.ts`) vẫn chạy nên khách vẫn thấy popup xin vị trí, nhưng không còn reverse-geocode nên không có lợi ích. **TODO (làm sau, thay đổi riêng):** gỡ `geolocation.ts` và các chỗ gọi (ZaloContactModal, LeadFormScreen, ContactLink), cùng phần `lat`/`lng` gửi lên `contact-click` và `createInquiryAction`.

### Phase 1 (cập nhật 2026-10-08, commit `02f6e38`, deployed và kiểm chứng trên production)

- G3: `internal/platform/httpserver/health.go`: `/healthz` (liveness, không đụng DB) và `/readyz` (ping DB, timeout 2s, 503 không lộ lỗi). `Dockerfile` HEALTHCHECK đổi từ `/brands` sang `/healthz`. Còn lại: đổi vòng chờ Go trong `elc-temp/.github/workflows/deploy.yml` sang `/healthz` (làm SAU khi Go đã deploy).
- G2: `accesslog.go`: một dòng log mỗi request (request_id, method, route pattern, status, duration, bytes). Không ghi path/query (token magic-link, email); `client_ip` được thêm ở G7. 5xx -> Error, còn lại Info. Bỏ qua `/healthz`, `/readyz`. Đặt NGOÀI `recoverer` để panic vẫn thành dòng 500. Giữ `http.Flusher` cho SSE của AI chat.
- G5: `http.Server` thêm `ReadHeaderTimeout 10s`, `IdleTimeout 120s`. Cố ý KHÔNG đặt Read/WriteTimeout (upload 10MB chậm, SSE AI chat dài).
- Test: `go build`, `go vet`, `go test ./...` xanh; test mới chạy `-race`; đã thử đảo thứ tự middleware thì test panic fail như mong đợi.

### Việc sau khi xong observability

- Điều tra workflow `Sync AI provider pricing` đang fail (run `37758819980`, head `c09e526`, trước các thay đổi của RFC này).

### Quan sát cho Phase 2

Trường `bytes` của access log cho thấy vài endpoint trả payload rất lớn mỗi lần gọi: `/news` ~1.7MB, `/products` ~2MB (một lần 58KB, một lần 1.98MB tùy tham số), `/projects` ~610KB, `/brands` ~135KB. Đáng đo bằng histogram kích thước/độ trễ khi có metrics, vì chúng vừa tốn băng thông nội bộ vừa góp vào bộ nhớ của Next (xem V1).

### Phase 2 (đang làm, 2026-10-08)

Code (G1, G4-stats): `internal/platform/metrics` (registry riêng; counter/histogram theo **route pattern**, method chuẩn hóa về danh sách cố định để scanner không tạo series vô hạn; kích thước response; in-flight; collector cho `pgxpool.Stat()`; Go runtime/process/build-info). `httpserver.New(logger, extra...)` đặt middleware metrics NGOÀI `recoverer` nên panic vẫn được đếm là 500. `/metrics` nằm ở listener riêng `:9091` (không đi qua router công khai, không qua nginx), compose publish `127.0.0.1:9091` cho Prometheus scrape. Test: cardinality (3 id khác nhau -> 1 series), không rò token/email vào output, panic = 500, method lạ -> `OTHER`, SSE giữ Flusher; đã thử phá code (nhãn bằng raw path, bỏ chuẩn hóa method) thì test fail.

Hạ tầng (`monitoring/`, compose project riêng `elc-monitoring`, tất cả `network_mode: host` + bind `127.0.0.1`): Prometheus v3.15.0 (256MB, giữ 15 ngày / 1GB), Grafana 13.2.3 (256MB, truy cập qua SSH tunnel cổng 3001), node-exporter v1.12.1 (64MB), process-exporter 0.8.7 (64MB). Mỗi service có `mem_limit` = `memswap_limit` để vượt giới hạn thì chỉ nó chết và restart, không kéo cả máy. Dashboard `ELC overview` (16 panel) và 9 luật cảnh báo (RAM, swap, đĩa, 5xx, p95, restart Go, DB pool, RSS Next, target down) được provision từ file. `promtool` xác nhận config và rule hợp lệ.

Chưa có kênh gửi cảnh báo (Alertmanager/Telegram/email): hiện chỉ xem được trên UI Prometheus/Grafana. Là một quyết định còn mở.
