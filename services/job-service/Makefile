.PHONY: build run test test-integration

APP_NAME = job-service
BUILD_DIR = bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./tests/integration/...
