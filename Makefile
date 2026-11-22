.PHONY: build test up down logs ps

build:
	go build ./...

test:
	go test ./... -count=1

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

ps:
	docker compose ps
