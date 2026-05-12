.PHONY: build run test docker-build clean

APP_NAME = auth-service
BUILD_DIR = bin

build:
	go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/main.go

run:
	go run ./cmd/main.go

test:
	go test ./...

test-integration:
	go test -v -tags=integration ./tests/integration/...

docker-build:
	docker build -t $(APP_NAME) .

clean:
	rm -rf $(BUILD_DIR)
