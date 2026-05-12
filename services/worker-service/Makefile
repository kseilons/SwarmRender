.PHONY: test-1 test-10 run build test

# Default file for testing
TEST_FILE ?= cmd/test-client/test.blend

# Run a single frame task (frame 1)
test-1:
	go run cmd/test-client/main.go -file $(TEST_FILE) -project test-1-frame -start 1 -end 1

# Run a 10 frame task (frames 1-10)
test-10:
	go run cmd/test-client/main.go -file $(TEST_FILE) -project test-10-frames -start 1 -end 10

# Helper to run the test client with custom args (e.g. make client ARGS="-project myproj -file file.blend")
client:
	go run cmd/test-client/main.go $(ARGS)

# Run the worker service
run:
	go run cmd/app/main.go

# Build the worker service
build:
	go build -o bin/worker cmd/app/main.go

# Run tests
test:
	go test ./... -v

test-integration:
	go test -v -tags=integration ./tests/integration/...
