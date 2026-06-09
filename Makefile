.PHONY: up down build dev migrate seed logs

# Start all services
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Build Go server binary
build:
	go build -o bin/server ./cmd/server

# Run server locally (no Docker)
dev:
	go run ./cmd/server

# Run database migrations
migrate:
	docker-compose exec server go run ./cmd/migrate

# Seed test data
seed:
	docker-compose exec postgres psql -U deploydock -d deploydock -f /scripts/seed.sql

# Tail all logs
logs:
	docker-compose logs -f

# Tail server logs only
logs-server:
	docker-compose logs -f server

# Run tests
test:
	go test ./...

# Tidy Go modules
tidy:
	go mod tidy