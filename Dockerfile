# Build stage - using Chainguard Go image for security 
# NOTE: Using latest-dev as Chainguard provides current Go version builds
# For production, consider pinning to specific digests: docker pull cgr.dev/chainguard/go:latest-dev && docker inspect
FROM cgr.dev/chainguard/go:latest-dev AS builder

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

# Final stage - using Chainguard static image (most secure, zero CVEs)
# NOTE: Using latest as Chainguard maintains current secure static base
# For production, consider pinning to specific digests: docker pull cgr.dev/chainguard/static:latest && docker inspect  
FROM cgr.dev/chainguard/static:latest

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