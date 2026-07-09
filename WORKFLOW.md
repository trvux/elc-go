# Workflow — Dev, Push, Deploy

Ghi lai quy trinh lam viec hang ngay sau khi fix loi deploy khong chay
migration va thoi quen push thang len main (2026-07-08). Ap dung cho ca hai
repo `elc-go` va `elc-tem` — file nay duoc dat o ca hai, noi dung giong nhau.

## 1. Bat dau 1 viec moi

Khong code truc tiep tren `main` — `main` la production, push len la deploy
ngay lap tuc (xem `.github/workflows/deploy.yml`).

```
git checkout main && git pull
git checkout -b feature/ten-viec
```

## 2. Dev hang ngay tren branch

**elc-go**
- `make run` (chay `air`, tro Postgres local qua `docker-compose.override.yml`,
  host port 5434).
- Them migration moi: `make migrate-create module=X name=...`, roi
  `make migrate-up-all` de ap ngay len DB local — biet loi luc code, khong
  phai doi den luc deploy moi biet.

**elc-tem**
- `pnpm dev` (tro `GO_API_URL=http://localhost:8090` qua `.env.local`).

Push len branch bao nhieu lan cung duoc — khong dung production. Moi lan
push, `.github/workflows/ci.yml` tu chay:
- elc-go: `go build ./...`, `go vet ./...`, `go test ./...`
- elc-tem: `tsc --noEmit`, `next build` (lint co chay nhung khong chan —
  dang co ~36 loi cu chua don, xem comment trong ci.yml)

## 3. Merge vao main khi 1 phan viec chay duoc that

Khong doi nguyen feature lon xong moi merge — merge tung phan nho *chay
duoc* an toan hon nhieu so voi 1 cuc to. Test local xong (buoc 2) roi moi
merge.

```
git checkout main && git pull && git merge feature/ten-viec && git push
```

(hoac mo Pull Request tren GitHub roi merge qua do, khong bat buoc).

## 4. Push len main = tu dong deploy, khong can lam gi them

**elc-go** (`.github/workflows/deploy.yml`):
1. rsync code len VPS
2. `docker compose build`
3. Postgres len, cho healthy
4. `scripts/migrate-all.sh` — ap moi migration con treo cho tat ca module,
   TRUOC khi doi container moi (day la buoc bi thieu truoc day, la nguyen
   nhan that su cua loi "deploy Go la site sap")
5. Container elc-go moi len, health-check qua `/brands` — container cu chi
   tat khi cai moi da khoe
6. Purge Cloudflare cache

**elc-tem** (`.github/workflows/deploy.yml`):
1. Build (`pnpm build`, standalone output)
2. rsync len VPS
3. PM2 reload cluster mode (2 worker, rolling restart — luon co it nhat 1
   worker phuc vu request trong luc doi worker)
4. Purge Cloudflare cache

Ca 2 pipeline tu bao ve o muc "chay duoc hay khong chay duoc" (container/
worker cu khong tat neu cai moi chua khoe) — khong bao ve duoc logic
nghiep vu sai, van phai tu test local o buoc 2 truoc khi merge.

## 5. Viec dong tham ca 2 repo (Go them field, Next.js dung field do)

Thu tu bat buoc, khong duoc dao:
1. Merge + deploy **elc-go** truoc — migration phai additive (cot moi
   nullable/co default, khong rename/drop cung luc code dang can no).
2. Xac nhan chay dung that tren production (curl endpoint moi, hoac check
   log container).
3. Roi moi merge + deploy **elc-tem** goi field/API moi do.

Khong bao gio de elc-tem len truoc khi elc-go da that su song voi field do.

## Con thieu (optional, chua lam)

Bat branch protection cho `main` tren GitHub settings cua tung repo (yeu
cau CI pass moi merge duoc PR) — chan han viec lo tay push thang vao main.
Day la setting tren GitHub UI, khong sua qua code duoc.

## Rieng Zalo OA webhook

Can 1 URL public that de nhan webhook — khong test duoc bang branch/local
thuan tuy. Chua dang ky Zalo OA doanh nghiep nen chua can xu ly, ghi chu de
nho: khi lam lai, dung tunnel (Cloudflare Tunnel/ngrok) de expose local
dev ra URL public tam thoi, tranh phai debug truc tiep tren production nhu
truoc.
