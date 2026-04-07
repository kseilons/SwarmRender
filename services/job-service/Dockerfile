FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o job-service ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/job-service .
COPY config/ config/

EXPOSE 8081

CMD ["./job-service"]
