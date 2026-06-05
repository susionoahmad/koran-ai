#!/bin/bash
# ============================================================
# Deploy Koran AI Backend to Debian 24.04 Server
# Usage: ./deploy.sh
# ============================================================
set -e

APP_NAME="koran-api"
BUILD_DIR="./bin"
DEPLOY_DIR="/var/www/koran-ai/backend"
SERVICE_NAME="koran-backend"

echo "==> Building Go binary..."
mkdir -p $BUILD_DIR
GOOS=linux GOARCH=amd64 go build -o $BUILD_DIR/$APP_NAME ./cmd/api/main.go
echo "    Binary built: $BUILD_DIR/$APP_NAME"

echo "==> Stopping existing service..."
sudo systemctl stop $SERVICE_NAME || true

echo "==> Copying binary to deployment directory..."
sudo mkdir -p $DEPLOY_DIR
sudo cp $BUILD_DIR/$APP_NAME $DEPLOY_DIR/$APP_NAME
sudo chmod +x $DEPLOY_DIR/$APP_NAME

echo "==> Copying .env to deployment directory (if local)..."
if [ -f .env ]; then
    sudo cp .env $DEPLOY_DIR/.env
fi

echo "==> Registering systemd service..."
sudo cp scripts/koran-backend.service /etc/systemd/system/$SERVICE_NAME.service
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME

echo "==> Starting service..."
sudo systemctl start $SERVICE_NAME
sleep 2

echo "==> Service status:"
sudo systemctl status $SERVICE_NAME --no-pager

echo ""
echo "==> Deployment complete! Check health at http://localhost:8080/health"
