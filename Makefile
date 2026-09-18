.PHONY: run worker test build fmt docker-up docker-down

run:
	go run ./cmd/api

worker:
	go run ./cmd/worker

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
