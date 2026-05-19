# Stage 1: Build binary inside golang alpine image
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy all source files
COPY . .

# Run tidy to fetch imported dependencies (like Fiber) during Docker build
RUN go mod tidy

# Build static binary optimized for Linux environments
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api/main.go

# Stage 2: Clean and lightweight runner
FROM alpine:latest

# Install basic networking utilities and certificates
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/main .

# Expose Go server default port
EXPOSE 12000

# Start server
CMD ["./main"]
