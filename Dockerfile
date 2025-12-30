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
# ENTRYPOINT contains fixed args, CMD contains overridable defaults
# Usage: docker run nanolog --role=engine --admin-addr=admin:8080
ENTRYPOINT ["./nanolog", "--port=8080", "--data=/root/data", "--web=./web"]
CMD ["--role=standalone"]
