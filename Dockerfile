# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# Runtime stage
FROM alpine:3.21.3

RUN addgroup -g 10001 -S appgroup \
    && adduser -u 10001 -S appuser -G appgroup -h /app -s /sbin/nologin

WORKDIR /app

COPY --from=builder --chown=10001:10001 /out/server /app/server
COPY --chown=10001:10001 migrations/ /app/migrations/

USER 10001:10001

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health/live || exit 1

ENTRYPOINT ["/app/server"]
