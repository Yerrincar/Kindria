SHELL := /bin/bash

APP_NAME := kindria
DB_PATH := books.db
MIGRATIONS_DIR := internal/core/platform/storage/migrations

.PHONY: help init run build clean deps check-go check-calibre check-gio check-sqlite db-init fmt test

help:
	@echo "Targets:"
	@echo "  make init      - Prepare project (dirs, deps, database migrations)"
	@echo "  make run       - Run the app"
	@echo "  make build     - Build binary to ./bin/$(APP_NAME)"
	@echo "  make fmt       - Format Go code"
	@echo "  make test      - Run tests"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make deps      - Check optional runtime dependencies"

init: check-go deps
	@mkdir -p books cache/covers bin
	@go mod download
	@$(MAKE) db-init
	@echo "Initialization complete."

deps: check-calibre check-gio check-sqlite

check-go:
	@command -v go >/dev/null 2>&1 || { echo "Error: Go is required."; exit 1; }

check-calibre:
	@if ! command -v ebook-convert >/dev/null 2>&1; then \
		echo "Warning: 'ebook-convert' (Calibre) not found. Kindle sync conversion will not work."; \
		echo "         Install Calibre and ensure 'ebook-convert' is in PATH."; \
	fi

check-gio:
	@if ! command -v gio >/dev/null 2>&1; then \
		echo "Warning: 'gio' not found. Kindle MTP scan/copy may fail."; \
		echo "         Install GVFS tools for your distro."; \
	fi

check-sqlite:
	@if ! command -v sqlite3 >/dev/null 2>&1; then \
		echo "Warning: 'sqlite3' CLI not found. 'make db-init' needs it to create a fresh DB."; \
	fi

# Applies only the '-- +goose Up' section of each migration file.
db-init:
	@if [ -f "$(DB_PATH)" ]; then \
		echo "Database already exists at $(DB_PATH). Skipping migration bootstrap."; \
		echo "If you need a fresh DB: rm -f $(DB_PATH) && make db-init"; \
	else \
		command -v sqlite3 >/dev/null 2>&1 || { echo "Error: sqlite3 CLI required for db-init."; exit 1; }; \
		touch "$(DB_PATH)"; \
		set -e; \
		for f in $$(ls "$(MIGRATIONS_DIR)"/*.sql | sort); do \
			echo "Applying migration: $$f"; \
			awk 'BEGIN{on=0} /-- \+goose Up/{on=1; next} /-- \+goose Down/{on=0} on{print}' "$$f" | sqlite3 "$(DB_PATH)"; \
		done; \
		echo "Database bootstrap complete: $(DB_PATH)"; \
	fi

run: check-go
	@go run .

build: check-go
	@mkdir -p bin
	@go build -o bin/$(APP_NAME) .
	@echo "Built bin/$(APP_NAME)"

fmt: check-go
	@go fmt ./...

test: check-go
	@go test ./...

clean:
	@rm -rf bin
	@echo "Cleaned build artifacts."
