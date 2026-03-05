# --- Frontend builder ---
FROM node:22-alpine AS frontend

WORKDIR /app/web

COPY web/package.json web/package-lock.json* ./
RUN npm ci --ignore-scripts

COPY web/ .
RUN npm run build

# --- Backend builder ---
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Install goose for migrations (v3.24.1 compatible with Go 1.24)
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

# --- Runtime ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=frontend /app/web/dist ./static
COPY deploy/entrypoint.sh .

RUN mkdir -p /app/uploads && chmod +x /app/entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["./entrypoint.sh"]
