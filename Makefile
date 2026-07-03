-include .env
export

.PHONY: up down down-clean seed \
        booking-up booking-down \
        notification-up notification-down \
        booking-test booking-test-integration booking-test-e2e \
        notification-test-integration \
        test-e2e-system \
        lint booking-mock booking-load-test

# ── Инфраструктура ────────────────────────────────────────────────────────────

up:
	docker network create kafka-net 2>/dev/null || true
	docker compose up --build -d

down:
	docker compose down

down-clean:
	docker compose down -v

seed:
	docker compose exec -T booking-db \
	  psql -U $(BOOKING_DATABASE_USER) -d $(BOOKING_DATABASE_NAME) \
	  < booking-service/scripts/seed.sql

# ── Standalone (отдельный сервис без корневого compose) ───────────────────────
# kafka-net создаётся вручную, т.к. в standalone-режиме Kafka не поднимается.
# Kafka должна быть доступна по KAFKA_BROKERS (например, localhost:9094).

booking-up:
	docker network create kafka-net 2>/dev/null || true
	docker compose -f booking-service/docker-compose.yaml up --build -d

booking-down:
	docker compose -f booking-service/docker-compose.yaml down

notification-up:
	docker network create kafka-net 2>/dev/null || true
	docker compose -f notification-service/docker-compose.yaml up --build -d

notification-down:
	docker compose -f notification-service/docker-compose.yaml down

# ── booking-service ───────────────────────────────────────────────────────────

booking-test:
	go test -coverprofile=coverage.out \
		-coverpkg=$(shell go list ./booking-service/... | grep -v '/mocks' | tr '\n' ',') \
		./booking-service/...
	@go tool cover -func=coverage.out | tail -1

booking-test-integration:
	docker compose -f booking-service/docker-compose.test.yaml up -d --wait
	TEST_DATABASE_URL="host=localhost port=5433 user=postgres password=postgres dbname=booking_test sslmode=disable" \
	TEST_MIGRATE_URL="pgx5://postgres:postgres@localhost:5433/booking_test?sslmode=disable" \
	  go test -tags integration -v -count=1 -p 1 ./booking-service/internal/repo/...; \
	  result=$$?; \
	  docker compose -f booking-service/docker-compose.test.yaml down -v; \
	  exit $$result

booking-test-e2e:
	docker compose -f booking-service/docker-compose.e2e.yaml up -d --wait
	E2E_DATABASE_URL="host=localhost port=5434 user=postgres password=postgres dbname=booking_e2e sslmode=disable" \
	E2E_MIGRATE_URL="pgx5://postgres:postgres@localhost:5434/booking_e2e?sslmode=disable" \
	  go test -tags e2e -v -count=1 ./booking-service/e2e/...; \
	  result=$$?; \
	  docker compose -f booking-service/docker-compose.e2e.yaml down -v; \
	  exit $$result

booking-mock:
	cd booking-service && mockery

booking-load-test:
	@echo "Сервис должен быть запущен: make up && make seed"
	@mkdir -p booking-service/loadtest/results
	docker run --rm -i --network host \
		--user $(shell id -u):$(shell id -g) \
		-v $(shell pwd)/booking-service/loadtest:/loadtest \
		grafana/k6 run - < booking-service/loadtest/script.js

# ── notification-service ──────────────────────────────────────────────────────

notification-test-integration:
	docker compose -f notification-service/docker-compose.test.yaml up -d --wait
	TEST_DATABASE_URL="host=localhost port=5435 user=postgres password=postgres dbname=notifications_test sslmode=disable" \
	TEST_MIGRATE_URL="pgx5://postgres:postgres@localhost:5435/notifications_test?sslmode=disable" \
	  go test -tags integration -v -count=1 -p 1 ./notification-service/internal/repo/...; \
	  result=$$?; \
	  docker compose -f notification-service/docker-compose.test.yaml down -v; \
	  exit $$result

# ── System E2E (оба сервиса через docker-compose) ────────────────────────────

test-e2e-system:
	@echo "Сервисы должны быть запущены: make up"
	go test -tags e2e_system -v -count=1 -timeout 60s ./e2e/...

# ── Качество кода ─────────────────────────────────────────────────────────────

lint:
	cd booking-service && golangci-lint run
	cd notification-service && golangci-lint run
