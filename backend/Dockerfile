# syntax=docker/dockerfile:1.6

# ---- Build Stage ----
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath \
    -ldflags="-s -w -extldflags '-static' \
    -X main.Version=$(git describe --tags --always --dirty 2>/dev/null || echo dev) \
    -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/dujiangyan-server \
    ./cmd/server

# ---- Runtime Stage ----
FROM scratch

WORKDIR /app

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/dujiangyan-server /usr/local/bin/dujiangyan-server
COPY --from=builder /app/.env.example ./.env
COPY --from=builder /app/config ./config

ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

EXPOSE 8080 6060

ENTRYPOINT ["/usr/local/bin/dujiangyan-server"]
