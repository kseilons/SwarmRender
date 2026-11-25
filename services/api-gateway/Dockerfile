FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o api-gateway ./cmd/main.go

FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/api-gateway .
COPY config/ config/

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8000

CMD ["./api-gateway"]
