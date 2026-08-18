# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git tzdata

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /bin/anohive \
    ./cmd/server

# Build CLI tool
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /bin/anohive-cli \
    ./cmd/cli

# Frontend build stage
FROM node:22-alpine AS frontend-builder

WORKDIR /app
COPY web/package.json web/package-lock.json* ./
RUN npm ci

COPY web/ ./
RUN npm run build

# Final stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S anohive && adduser -S anohive -G anohive

# Copy binaries
COPY --from=builder /bin/anohive /usr/local/bin/anohive
COPY --from=builder /bin/anohive-cli /usr/local/bin/anohive-cli

# Copy frontend build
COPY --from=frontend-builder /app/build /var/lib/anohive/web

# Create data directory
RUN mkdir -p /var/lib/anohive/data && chown -R anohive:anohive /var/lib/anohive

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Expose port
EXPOSE 8200

# Switch to non-root user
USER anohive

# Set working directory
WORKDIR /var/lib/anohive

# Run the server
ENTRYPOINT ["anohive"]
CMD ["-db", "/var/lib/anohive/data/anohive.db", "-static", "/var/lib/anohive/web", "-port", "8200"]
