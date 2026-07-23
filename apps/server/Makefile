.PHONY: run build tidy test

run:
	go run ./cmd/server

build:
	go build -ldflags="-s -w" -o bin/booknav ./cmd/server

tidy:
	go mod tidy

test:
	go test ./...
