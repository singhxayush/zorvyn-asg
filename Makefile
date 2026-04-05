# Simple Makefile for a Go project

# Load environment variables from .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Ensure DB_URL has a default if not specified in .env
DB_URL ?= finance.db
# golang-migrate requires the sqlite3:// prefix
MIGRATION_URL = sqlite3://$(DB_URL)

# Build the application
all: build test

build:
	@echo "Building..."
	@mkdir -p bin
	@go build -o bin/main cmd/api/main.go

# Run the application
run:
	@go run cmd/api/main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -rf bin

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

# ==============================================================================
# Database Migrations
# ==============================================================================

# Prerequisite check for golang-migrate
check-migrate:
	@if command -v migrate > /dev/null; then \
		exit 0; \
	else \
		read -p "Go's 'migrate' is not installed. Do you want to install it? [Y/n] " choice; \
		if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
			echo "Installing golang-migrate with sqlite3 support..."; \
			go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest; \
		else \
			echo "You chose not to install migrate. Exiting..."; \
			exit 1; \
		fi; \
	fi

# Install golang-migrate with SQLite support
install-migrate:
	@echo "Installing golang-migrate with SQLite3 support..."
	@go install -tags 'sqlite3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "Installation complete. Make sure your Go bin directory is in your PATH."

# Notice how these targets now depend on `check-migrate`
db-migrate-up: check-migrate
	@echo "Running migrations up..."
	@migrate -database "$(MIGRATION_URL)" -path ./migrations up

db-migrate-down: check-migrate
	@echo "Running migrations down..."
	@migrate -database "$(MIGRATION_URL)" -path ./migrations down

db-migrate-create: check-migrate
	@if [ -z "$(name)" ]; then \
		echo "Error: migration name is required."; \
		echo "Usage: make db-migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	@echo "Creating new migration: $(name)..."
	@migrate create -seq -ext sql -dir ./migrations $(name)

# ==============================================================================
# Admin Init
# ==============================================================================
seed:
	@go run cmd/seed/main.go

# ==============================================================================
# Help
# ==============================================================================

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@echo "  build                Build the application into bin/ directory"
	@echo "  run                  Run the application (using go run)"
	@echo "  test                 Run tests"
	@echo "  clean                Remove the bin/ directory and compiled binary"
	@echo "  watch                Run the application with live reload (air)"
	@echo "  db-migrate-up        Run all pending database migrations"
	@echo "  db-migrate-down      Rollback the last database migration"
	@echo "  db-migrate-create    Create a new migration file (e.g., make db-migrate-create name=init)"
	@eco "   seed                 Initialise superadmin"
	@echo "  help                 Show this help message"

.PHONY: all build run test clean watch check-migrate db-migrate-up db-migrate-down db-migrate-create help
