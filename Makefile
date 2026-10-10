# MealSwipe — common tasks. `make help` lists targets.
SHELL := /bin/bash
GO ?= go
COMPOSE ?= docker compose

.PHONY: help up down dev-db backend frontend build test test-backend test-frontend \
        lint fmt vet migrate seed import-mealdb backup restore e2e docker-build clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# ---------- docker ----------
up: ## Build and start the full stack (db + backend + frontend)
	$(COMPOSE) up -d --build

down: ## Stop the stack (data volume is kept)
	$(COMPOSE) down

docker-build: ## Build images without starting
	$(COMPOSE) build

# ---------- local dev ----------
dev-db: ## Start dev PostgreSQL only (compose.dev.yaml, port 5432)
	$(COMPOSE) -f compose.dev.yaml up -d

backend: ## Run backend natively (needs dev-db + DATABASE_URL)
	cd backend && $(GO) run ./cmd/server serve

frontend: ## Run frontend dev server (Vite, proxies /api -> :8080)
	cd frontend && npm run dev

# ---------- db ----------
migrate: ## Apply migrations only
	cd backend && $(GO) run ./cmd/server migrate

seed: ## Load bundled seed recipes
	cd backend && $(GO) run ./cmd/server seed

import-mealdb: ## Import N random meals from TheMealDB (N=25)
	cd backend && $(GO) run ./cmd/server import-mealdb -n $${N:-25}

# ---------- checks ----------
build: ## Compile backend + build frontend
	cd backend && $(GO) build ./...
	cd frontend && npm run build

test: test-backend test-frontend ## All tests

test-backend: ## Go unit tests (integration needs TEST_DATABASE_URL)
	cd backend && $(GO) test ./...

test-frontend: ## Frontend unit tests
	cd frontend && npm test

lint: ## go vet + eslint
	cd backend && $(GO) vet ./...
	cd frontend && npm run lint

vet:
	cd backend && $(GO) vet ./...

fmt: ## gofmt + prettier(if configured)
	cd backend && $(GO)fmt -w .

e2e: ## Playwright e2e (requires stack running: make up)
	cd tests/e2e && npm test

# ---------- ops ----------
backup: ## pg_dump to ./backups/
	bash scripts/backup.sh

restore: ## Restore from dump: make restore FILE=backups/xxx.sql.gz
	bash scripts/restore.sh $(FILE)

clean: ## Remove build artifacts (keeps volumes)
	rm -rf backend/bin frontend/dist
