# UI-Bill: CGRateS Web GUI Platform
# Makefile для локальной разработки и CI/CD

.PHONY: help up down build migrate seed test lint fmt clean k8s-deploy

# Переменные
COMPOSE_FILE := infra/docker/docker-compose.yml
DB_DSN := postgres://postgres:postgres@localhost:5432/uibill?sslmode=disable
MIGRATE_CMD := migrate -path database/migrations -database $(DB_DSN)

help: ## Показать список команд
	@echo "UI-Bill Development Commands"
	@echo "============================"
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Docker Compose ---
up: ## Поднять локальное окружение (PostgreSQL, Redis, Pulsar, MinIO)
	docker-compose -f $(COMPOSE_FILE) up -d

down: ## Остановить локальное окружение
	docker-compose -f $(COMPOSE_FILE) down -v

logs: ## Показать логи всех сервисов
	docker-compose -f $(COMPOSE_FILE) logs -f

# --- База данных ---
migrate-up: ## Применить миграции
	$(MIGRATE_CMD) up

migrate-down: ## Откатить последнюю миграцию
	$(MIGRATE_CMD) down 1

migrate-create: ## Создать новую миграцию (usage: make migrate-create name=add_users)
	migrate create -ext sql -dir database/migrations -seq $(name)

seed: ## Загрузить seed-данные (root, языки, дефолтный tenant)
	psql $(DB_DSN) -f database/seeds/001_initial_seed.sql

# --- Backend ---
build-gateway-%: ## Собрать gateway (usage: make build-gateway-selfcare)
	cd backend/$*-gateway && go build -o ../../bin/$*-gateway ./cmd/main.go

run-gateway-%: ## Запустить gateway локально (usage: make run-gateway-selfcare)
	cd backend/$*-gateway && go run ./cmd/main.go

# --- Frontend ---
build-ui-%: ## Собрать frontend (usage: make build-ui-selfcare)
	cd frontend/$*-ui && npm run build

dev-ui-%: ## Запустить dev-сервер frontend (usage: make dev-ui-selfcare)
	cd frontend/$*-ui && npm run dev

# --- Workers ---
build-worker-%: ## Собрать worker (usage: make build-worker-audit-consumer)
	cd workers/$* && go build -o ../../bin/$* ./cmd/main.go

# --- Quality ---
test: ## Запустить тесты backend
	cd backend && go test ./...

lint: ## Линтинг Go и SQL
	golangci-lint run ./backend/...
	# sqlfluff lint database/migrations/

fmt: ## Форматирование кода
	cd backend && gofmt -w .
	cd workers && gofmt -w .

# --- Infrastructure ---
k8s-deploy: ## Применить Kubernetes манифесты (требует kubectl и namespace)
	kubectl apply -f infra/k8s/namespaces/
	kubectl apply -f infra/k8s/database/
	kubectl apply -f infra/k8s/messaging/
	kubectl apply -f infra/k8s/selfcare/
	kubectl apply -f infra/k8s/operator/
	kubectl apply -f infra/k8s/admin/

clean: ## Очистить артефакты сборки
	rm -rf bin/
	docker-compose -f $(COMPOSE_FILE) down -v --rmi local
