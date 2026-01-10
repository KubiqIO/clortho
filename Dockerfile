# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o clortho-server ./cmd/server/main.go

# Runtime stage
FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/clortho-server .
# Copy migrations directory (required by the app on startup)
COPY --from=builder /app/migrations ./migrations

RUN adduser -D clortho
USER clortho

EXPOSE 8080

CMD ["./clortho-server"]
