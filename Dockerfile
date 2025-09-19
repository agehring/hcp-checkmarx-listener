# Build stage
FROM golang:1.23-alpine AS builder

# Install git and ca-certificates (for SSL/TLS)
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application for Linux with static linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a -installsuffix cgo -ldflags '-extldflags "-static"' \
    -o main ./cmd

# Final stage - using Google's distroless image (no ncurses, minimal attack surface)
FROM gcr.io/distroless/static-debian12:latest

# Final stage - using Google's distroless image (no ncurses, minimal attack surface)
FROM gcr.io/distroless/static-debian12:latest

# Copy the binary from builder stage
COPY --from=builder /app/main /app/main

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose port 8080 (OpenShift default)
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release \
    PORT=8080

# Use numeric user ID for OpenShift compatibility
USER 65532

# Run the application
ENTRYPOINT ["/app/main"]