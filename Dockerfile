# Build Stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .

RUN go build -o nanolog cmd/nanolog/main.go

# Frontend Build Stage
FROM node:18-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package*.json ./
RUN npm install

COPY web/ .
RUN npm run build

# Run Stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/nanolog .

COPY --from=frontend-builder /app/web/dist ./web/

VOLUME /root/data

EXPOSE 8080

ENTRYPOINT ["./nanolog", "--port=8080", "--data=/root/data", "--web=./web"]
CMD ["--role=standalone"]
