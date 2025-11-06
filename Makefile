build:
	@go build -o bin/totion

run: build
	@./bin/totion
