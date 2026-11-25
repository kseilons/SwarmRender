FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o auth-service ./cmd/main.go

FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/auth-service .
COPY config/ config/

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8082

CMD ["./auth-service"]
