# Multi-stage build for optimized production image
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o d2mcp ./cmd

# Final stage - minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    librsvg \
    librsvg-tools \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/d2mcp .

# Create necessary directories and set permissions
RUN mkdir -p /tmp && \
    chown -R appuser:appgroup /app /tmp

# Switch to non-root user
USER appuser

# Expose port (Cloud Run will override this with PORT env var)
EXPOSE 8080


# Set environment variables
ENV PORT=8080
ENV D2_LOG_LEVEL=NONE

# Default command - run in streamable HTTP mode for Cloud Run
CMD ["./d2mcp", "-transport=streamable", "-addr=:8080", "-stateless=false", "-cors-origins=*", "-cors-credentials=false"]
