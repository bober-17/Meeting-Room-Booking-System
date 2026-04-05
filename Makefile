.PHONY: up down seed test test-integration lint swagger mock

up:
	docker-compose up --build -d

down:
	docker-compose down -v

seed:
	docker-compose exec -T db psql -U $${DATABASE_USER:-postgres} -d $${DATABASE_NAME:-booking} < scripts/seed.sql

test:
	go test -coverprofile=coverage.out \
		-coverpkg=$(shell go list ./... | grep -v '/mocks' | tr '\n' ',') \
		./...
	@go tool cover -func=coverage.out | tail -1

test-integration:
	docker compose -f docker-compose.test.yaml up -d --wait
	TEST_DATABASE_URL="host=localhost port=5433 user=postgres password=postgres dbname=booking_test sslmode=disable" \
	TEST_MIGRATE_URL="pgx5://postgres:postgres@localhost:5433/booking_test?sslmode=disable" \
	  go test -tags integration -v -count=1 -p 1 ./internal/repo/...; \
	  result=$$?; \
	  docker compose -f docker-compose.test.yaml down -v; \
	  exit $$result

test-e2e:
	docker compose -f docker-compose.e2e.yaml up -d --wait
	E2E_DATABASE_URL="host=localhost port=5434 user=postgres password=postgres dbname=booking_e2e sslmode=disable" \
	E2E_MIGRATE_URL="pgx5://postgres:postgres@localhost:5434/booking_e2e?sslmode=disable" \
	  go test -tags e2e -v -count=1 ./e2e/...; \
	  result=$$?; \
	  docker compose -f docker-compose.e2e.yaml down -v; \
	  exit $$result

lint:
	golangci-lint run

mock:
	mockery
