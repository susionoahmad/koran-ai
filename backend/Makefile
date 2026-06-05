# Makefile for Koran AI Indonesia Backend

.PHONY: run build test migrate-up migrate-down

# Run the API web server locally
run:
	go run cmd/api/main.go

# Build the compiled Go binary
build:
	mkdir -p bin
	go build -o bin/koran-api cmd/api/main.go

# Run all unit tests with code coverage
test:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

# Run migrations up
migrate-up:
	cd scripts && chmod +x *.sh && ./migrate-up.sh

# Run migrations down (rollback)
migrate-down:
	cd scripts && chmod +x *.sh && ./migrate-down.sh
