# =============================================
# Multi-stage build - Resilient HTTP Proxy
# =============================================

# ------------------- Builder Stage -------------------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first (better caching)
#COPY go.mod go.sum ./
#RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w -extldflags '-static'" \
    -o resilient-proxy resilient-proxy.go && ls -l


# ------------------- Final Minimal Image -------------------
FROM alpine:3.22.4

WORKDIR /app


# Copy the binary
COPY --from=builder /app/resilient-proxy .

RUN mkdir /etc/resilient-proxy && chown 10001:10001 /etc/resilient-proxy


# Non-root user
USER 10001:10001

# Expose default port
EXPOSE 8443


# Run the proxy
ENTRYPOINT ["/app/resilient-proxy"]
