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
    -o /bin/pulse \
    ./cmd/server

# Build CLI tool
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /bin/pulse-cli \
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
RUN addgroup -S pulse && adduser -S pulse -G pulse

# Copy binaries
COPY --from=builder /bin/pulse /usr/local/bin/pulse
COPY --from=builder /bin/pulse-cli /usr/local/bin/pulse-cli

# Copy frontend build
COPY --from=frontend-builder /app/build /var/lib/pulse/web

# Create data directory
RUN mkdir -p /var/lib/pulse/data && chown -R pulse:pulse /var/lib/pulse

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Expose port
EXPOSE 8080

# Switch to non-root user
USER pulse

# Set working directory
WORKDIR /var/lib/pulse

# Run the server
ENTRYPOINT ["pulse"]
CMD ["-db", "/var/lib/pulse/data/pulse.db", "-static", "/var/lib/pulse/web", "-port", "8080"]
