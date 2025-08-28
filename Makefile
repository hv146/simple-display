build:
	@go build -o bin/simple-display

run: build
	@./bin/simple-display

test:
	@gotest -v ./...
