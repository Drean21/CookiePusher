# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder

# Install git and build dependencies
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs
RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/api/main.go -o docs

# Build the application
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.version=${GITHUB_REF##*/}" \
    -a \
    -o cookiepusher \
    ./cmd/api

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/cookiepusher .

# Create data directory
RUN mkdir -p /root/data

# Expose port
EXPOSE 8080

# Environment variables
# Default to SQLite. For external DBs, these should be set in docker-compose.yml or `docker run -e`
ENV DB_TYPE=sqlite
ENV DSN=/root/data/CookiePusher.db
ENV ADMIN_KEY=
ENV POOL_ACCESS_KEY=
ENV PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

# Run the binary
CMD ["./cookiepusher"]
