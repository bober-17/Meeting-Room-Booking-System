.PHONY: up down seed test lint swagger

up:
	docker-compose up --build -d

down:
	docker-compose down -v

seed:
	docker-compose exec db psql -U $${DATABASE_USER:-postgres} -d $${DATABASE_NAME:-booking} -f /migrations/0006_seed_dummy_users.up.sql

test:
	go test ./...

lint:
	golangci-lint run

swagger:
	swag init -g cmd/main.go -o docs
