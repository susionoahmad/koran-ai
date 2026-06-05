#!/bin/bash
# Script to run migrations UP

# Load environment variables from .env
if [ -f ../.env ]; then
    export $(cat ../.env | grep -v '^#' | xargs)
elif [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-koran_ai_prod}
DB_SSLMODE=${DB_SSLMODE:-disable}

if ! command -v migrate &> /dev/null; then
    echo "Warning: golang-migrate is not installed. Downloading binary to /tmp..."
    curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | tar xvz -C /tmp migrate
    MIGRATE_BIN="/tmp/migrate"
else
    MIGRATE_BIN="migrate"
fi

DATABASE_URL="postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

echo "Running migrations UP against database: $DB_HOST:$DB_PORT/$DB_NAME..."
$MIGRATE_BIN -path ../migrations -database "$DATABASE_URL" up
