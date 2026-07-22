# Multi-stage build: the build stage has the full Go toolchain, the final
# image only has the compiled binary — keeps the shipped image small and
# avoids exposing source/toolchain in production.

# fastText CLI (chat-search category classifier — see
# internal/product/application/chat_classifier.go). No pure-Go fastText
# inference library exists, so this compiles the real C++ binary from
# source rather than pulling in CGO for the rest of an otherwise
# CGO_ENABLED=0 build. Pinned to a release tag for reproducible builds.
FROM alpine:3.20 AS fasttext-build
RUN apk add --no-cache git build-base
RUN git clone --depth 1 --branch v0.9.2 https://github.com/facebookresearch/fastText.git /fasttext-src
WORKDIR /fasttext-src
# v0.9.2 relies on <cstdint> being pulled in transitively, which newer GCC
# (Alpine's, stricter than what this was released against) no longer does
# — args.h uses int64_t without including it directly.
RUN sed -i '1i #include <cstdint>' src/args.h && make

FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

# Train + quantize the chat-search category classifier. Training data is
# generated from the lookup table in cmd/train-chat-classifier (terms ×
# templates), not hand-written — see that file to add a new category.
# -bucket 20000 matters: fastText's default (2,000,000) hash buckets is
# wildly oversized for this ~70-word vocabulary and bloats the quantized
# model to 30MB+ with worse accuracy; tuned to the vocabulary size it
# quantizes to ~330KB.
COPY --from=fasttext-build /fasttext-src/fasttext /usr/local/bin/fasttext
# libstdc++/libgcc: needed to *run* the fasttext binary here at build time
# (training), same reason the final stage below needs them to run it too.
RUN apk add --no-cache libstdc++ libgcc
RUN go run ./cmd/train-chat-classifier > /tmp/chat-classifier-train.txt && \
    /usr/local/bin/fasttext supervised \
      -input /tmp/chat-classifier-train.txt \
      -output /app/chat-classifier \
      -lr 1.0 -epoch 30 -wordNgrams 2 -dim 30 -bucket 20000 && \
    /usr/local/bin/fasttext quantize \
      -input /tmp/chat-classifier-train.txt \
      -output /app/chat-classifier \
      -retrain -epoch 30 -lr 1.0 -wordNgrams 2 -dim 30 -bucket 20000 && \
    rm /app/chat-classifier.bin

FROM alpine:3.20
# libstdc++/libgcc: the fasttext binary is C++, not statically linked.
RUN apk add --no-cache ca-certificates curl libstdc++ libgcc
COPY --from=build /bin/server /bin/server
COPY --from=fasttext-build /fasttext-src/fasttext /usr/local/bin/fasttext
COPY --from=build /app/chat-classifier.ftz /app/chat-classifier.ftz
ENV FASTTEXT_BIN=/usr/local/bin/fasttext
ENV CHAT_CLASSIFIER_MODEL=/app/chat-classifier.ftz

EXPOSE 8090
# Cho phep "docker compose up --wait" / deploy pipeline biet khi nao container
# moi thuc su san sang nhan request, thay vi coi swap la xong ngay khi tien
# trinh khoi dong (co the con dang ket noi DB, chua nhan request duoc).
HEALTHCHECK --interval=5s --timeout=3s --start-period=10s --retries=5 \
  CMD curl -sf http://localhost:${PORT:-8090}/brands || exit 1
ENTRYPOINT ["/bin/server"]
