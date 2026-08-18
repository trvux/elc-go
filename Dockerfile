# Multi-stage build: the build stage has the full Go toolchain, the final
# image only has the compiled binary — keeps the shipped image small and
# avoids exposing source/toolchain in production.

FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

FROM alpine:3.20
# libwebp-tools: gives internal/upload/infrastructure's crop-on-upload
# feature the real cwebp encoder — imaging's pure-Go Encode has no WEBP
# case, and no pure-Go lossy WebP encoder here has libwebp's track record,
# so this shells out to the reference tool instead.
RUN apk add --no-cache ca-certificates curl libwebp-tools
COPY --from=build /bin/server /bin/server

EXPOSE 8090
# Cho phep "docker compose up --wait" / deploy pipeline biet khi nao container
# moi thuc su san sang nhan request, thay vi coi swap la xong ngay khi tien
# trinh khoi dong (co the con dang ket noi DB, chua nhan request duoc).
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=5 \
  CMD curl -sf http://localhost:${PORT:-8090}/brands || exit 1
ENTRYPOINT ["/bin/server"]
