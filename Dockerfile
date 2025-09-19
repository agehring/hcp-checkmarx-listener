# Build stage - using Chainguard Go image for security (pinned version)
FROM cgr.dev/chainguard/go:1.23@sha256:4d4b56f8858db8a0aebf4fdb2db4f5e5dfa1194b486f1cf5a606c78ff2ad95f9 AS builder

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

# Final stage - using Chainguard static image (most secure, zero CVEs, pinned version)
FROM cgr.dev/chainguard/static:latest@sha256:5ff428f8a48241da4e78d1c31f6e1f92d4c725f4e8ba7ff44ef64cecac6fa47

# Copy the binary from builder stage
COPY --from=builder /app/main /app/main

# Expose port 8080 (OpenShift default)
EXPOSE 8080

# Set environment variables
ENV GIN_MODE=release \
    PORT=8080

# Use numeric user ID for OpenShift compatibility
USER 65532

# Run the application
ENTRYPOINT ["/app/main"]