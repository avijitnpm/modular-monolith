FROM golang:1.24-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG NO_PROXY
ARG http_proxy
ARG https_proxy
ARG no_proxy
ARG GOPROXY

ENV HTTP_PROXY=${HTTP_PROXY} \
    HTTPS_PROXY=${HTTPS_PROXY} \
    NO_PROXY=${NO_PROXY} \
    http_proxy=${http_proxy} \
    https_proxy=${https_proxy} \
    no_proxy=${no_proxy} \
    GOPROXY=${GOPROXY:-https://proxy.golang.org,direct}

COPY go.mod go.sum ./
ENV GOPROXY=https://proxy.golang.org,direct

RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache wget ca-certificates

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY migrations/ /app/migrations/

EXPOSE 8080

ENTRYPOINT ["/app/server"]
