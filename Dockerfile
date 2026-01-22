# Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary
# -ldflags="-w -s" to reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o myeventstream \
    ./cmd

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/myeventstream .

# Expose metrics port (default, can be overridden)
EXPOSE 8080

# Healthcheck placeholder
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD echo "Healthcheck not implemented yet" || exit 1

# Run the application
CMD ["./myeventstream"]
