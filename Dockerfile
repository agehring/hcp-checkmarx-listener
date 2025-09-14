# Build stage
FROM golang:1.22-alpine AS builder

# Install git and ca-certificates (for SSL/TLS)
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd

# Final stage
FROM alpine:latest

# Install ca-certificates for SSL/TLS
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of the app directory
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port 80
EXPOSE 80

# Set environment variables
ENV GIN_MODE=release

# Run the application
ENTRYPOINT ["./main"]