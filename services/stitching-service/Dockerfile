FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o stitching-service ./cmd/stitching/main.go

FROM alpine:latest

# Install FFmpeg
RUN apk add --no-cache ffmpeg

WORKDIR /app

COPY --from=builder /app/stitching-service .
COPY config/config.yaml ./config/config.yaml

CMD ["./stitching-service"]
