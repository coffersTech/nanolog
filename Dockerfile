# Build Stage
FROM golang:1.22-alpine AS builder

# Install git for private modules or non-proxied dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod and sum files
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy the source code
COPY server/ .

# Build the application
RUN go build -o nanolog cmd/nanolog/main.go

# Run Stage
FROM alpine:latest

# Install base dependencies
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/nanolog .

# Copy the web directory
COPY web/ ./web/

# Setup volume for data
VOLUME /root/data

# Expose the default port
EXPOSE 8080

# Run the application
# We use --data=/root/data to ensure it points to the container volume
# and --web=./web to point to the copied web files
ENTRYPOINT ["./nanolog", "--port=8080", "--data=/root/data", "--web=./web"]
