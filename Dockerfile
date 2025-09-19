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

# Final stage - using Red Hat UBI for OpenShift compatibility
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

# Install ca-certificates and timezone data
RUN microdnf update -y && \
    microdnf install -y ca-certificates tzdata && \
    microdnf clean all

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy ca-certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Create app directory with proper permissions for OpenShift
RUN chmod +x /app/main && \
    chgrp -R 0 /app && \
    chmod -R g=u /app

# OpenShift runs containers with arbitrary UIDs, so don't create specific user
# The container will run with a random UID that's part of the root group

# Expose port 8080 (OpenShift default)
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release \
    PORT=8080

# Use numeric user ID for OpenShift compatibility
USER 1001

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./main"]