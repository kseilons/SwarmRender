FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o stitching-service ./cmd/stitching/main.go

FROM alpine:latest

# Install FFmpeg and create non-root user
RUN apk add --no-cache ffmpeg && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/stitching-service .
COPY config/ config/

RUN chown -R appuser:appgroup /app

USER appuser

CMD ["./stitching-service"]
