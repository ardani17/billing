# =============================================================================
# ISPBoss Monorepo — Makefile
# =============================================================================
# Target utama untuk development, build, test, lint, migrasi, dan Docker.
# Kompatibel dengan Linux, macOS, dan Windows (Git Bash).
# =============================================================================

# Muat variabel dari .env jika file tersedia
-include .env
export

# --- Variabel Umum -----------------------------------------------------------

# Daftar Go service di monorepo
GO_SERVICES := services/billing-api services/network-service services/notification

# Daftar shared Go package di monorepo
GO_PACKAGES := pkg/auth pkg/database pkg/logger pkg/queue pkg/tenant

# Semua modul Go (services + packages)
GO_ALL := $(GO_SERVICES) $(GO_PACKAGES)

# Docker Compose file path
DOCKER_COMPOSE := docker/docker-compose.yml

# Konfigurasi database untuk migrasi
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= ispboss
DB_PASSWORD ?= ispboss_secret
DB_NAME ?= ispboss
DB_SSL_MODE ?= disable

# Connection string PostgreSQL untuk golang-migrate
DB_DSN := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

# Path migrasi billing-api
BILLING_MIGRATIONS := services/billing-api/migrations

# --- Default Target -----------------------------------------------------------

.DEFAULT_GOAL := help

# --- Phony Targets ------------------------------------------------------------

.PHONY: dev build test lint swagger migrate-up migrate-down docker-up docker-down help

# --- Development --------------------------------------------------------------

## dev: Jalankan semua service dalam mode development
dev:
	turbo run dev

# --- Build --------------------------------------------------------------------

## build: Build semua aplikasi dan service
build:
	turbo run build
	@for svc in $(GO_SERVICES); do \
		echo "Building $$svc..."; \
		cd $$svc && go build ./... && cd ../..; \
	done

# --- Test ---------------------------------------------------------------------

## test: Jalankan semua test (Go + frontend)
test:
	@for dir in $(GO_ALL); do \
		echo "Testing $$dir..."; \
		cd $$dir && go test ./... && cd ../..; \
	done
	turbo run test

# --- Lint ---------------------------------------------------------------------

## lint: Jalankan linter untuk Go (golangci-lint) dan frontend (ESLint)
lint:
	@for dir in $(GO_ALL); do \
		echo "Linting $$dir..."; \
		cd $$dir && golangci-lint run ./... && cd ../..; \
	done
	turbo run lint

# --- Swagger ------------------------------------------------------------------

## swagger: Generate Swagger docs untuk semua Go service
swagger:
	@for svc in $(GO_SERVICES); do \
		echo "Generating Swagger for $$svc..."; \
		cd $$svc && swag init -g cmd/main.go -o docs && cd ../..; \
	done

# --- Database Migrations ------------------------------------------------------

## migrate-up: Jalankan migrasi database (up) untuk billing-api
migrate-up:
	migrate -path $(BILLING_MIGRATIONS) -database "$(DB_DSN)" up

## migrate-down: Rollback migrasi database (down) untuk billing-api
migrate-down:
	migrate -path $(BILLING_MIGRATIONS) -database "$(DB_DSN)" down

# --- Docker -------------------------------------------------------------------

## docker-up: Jalankan Docker Compose stack (detached)
docker-up:
	docker compose -f $(DOCKER_COMPOSE) up -d

## docker-down: Hentikan Docker Compose stack
docker-down:
	docker compose -f $(DOCKER_COMPOSE) down

# --- Help ---------------------------------------------------------------------

## help: Tampilkan daftar target yang tersedia
help:
	@echo ""
	@echo "ISPBoss Monorepo — Available Targets"
	@echo "====================================="
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
	@echo ""
