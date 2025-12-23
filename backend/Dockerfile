# Build stage
FROM golang:1.21-alpine AS builder

# Install and update CA certificates for TLS verification
# Also install git which might be needed for some Go modules
RUN apk --no-cache add ca-certificates git && \
    update-ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
# Download dependencies
# Note: GOSUMDB=off disables checksum verification (DEVELOPMENT ONLY - NOT FOR PRODUCTION)
# This is a workaround for Windows Docker Desktop certificate validation issues.
# For production, remove GOSUMDB=off and ensure proper CA certificates are configured.
# GOPROXY=direct fetches directly from source (GitHub) instead of proxy.golang.org
ARG BUILD_ENV=development
RUN if [ "$BUILD_ENV" = "production" ]; then \
        go mod download; \
    else \
        GOPROXY=direct GOSUMDB=off go mod download; \
    fi

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]

