# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./
# Download dependencies (if any are added later)
RUN go mod download || true

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o jellyfin-sidecar ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/jellyfin-sidecar .

# Create directory for config
RUN mkdir -p /etc/jellyfin-sidecar

# Create directory for symlinks (can be overridden by volume mount)
RUN mkdir -p /data/leaving-soon

# Expose API port
EXPOSE 8090

# Run as non-root user
RUN adduser -D -u 1000 jellyfin
USER jellyfin

ENTRYPOINT ["/app/jellyfin-sidecar"]
CMD ["-config", "/etc/jellyfin-sidecar/config.json"]
