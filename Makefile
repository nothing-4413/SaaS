.PHONY: run test build fmt docker-up docker-down

run:
	go run ./cmd/api

test:
	go test ./...

build:
	go build ./...

fmt:
	go fmt ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down
