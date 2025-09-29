# Build stage
FROM golang:1.24.7-alpine AS builder

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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o k8s-pod-manager .

# Final stage
FROM scratch

# Copy timezone data and CA certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder
COPY --from=builder /app/k8s-pod-manager /k8s-pod-manager

# Set environment
ENV TZ=UTC
ENV PORT=8080
ENV IN_CLUSTER=true
ENV LOG_LEVEL=info

# Expose port
EXPOSE 8080

# Set user (non-root)
USER 1000:1000

# Run the application
ENTRYPOINT ["/k8s-pod-manager"]