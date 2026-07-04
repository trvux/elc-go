# Multi-stage build: the build stage has the full Go toolchain, the final
# image only has the compiled binary — keeps the shipped image small and
# avoids exposing source/toolchain in production.
FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/server /bin/server

EXPOSE 8090
ENTRYPOINT ["/bin/server"]
