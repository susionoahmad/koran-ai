#!/bin/bash
# Script to create a new migration pair
if [ -z "$1" ]; then
    echo "Usage: ./migrate-create.sh <migration_name>"
    exit 1
fi

# Ensure golang-migrate CLI is installed
if ! command -v migrate &> /dev/null; then
    echo "Error: golang-migrate is not installed."
    echo "Please install it: curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz | tar xvz && sudo mv migrate /usr/local/bin/"
    exit 1
fi

migrate create -ext sql -dir ../migrations -seq "$1"
