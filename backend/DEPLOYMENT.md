# Panduan Deployment Backend di Debian 24.04 (Native, Tanpa Docker)

## Prasyarat Server

Pastikan server Debian 24.04 LTS sudah berjalan dan memiliki akses `sudo`.

---

## 1. Install Go 1.24

```bash
# Download Go 1.24 binary
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz

# Extract ke /usr/local
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz

# Tambahkan ke PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verifikasi
go version
# Expected: go version go1.24.0 linux/amd64
```

---

## 2. Install PostgreSQL 18

```bash
# Install PGDG repository
sudo apt install -y curl ca-certificates
sudo install -d /usr/share/postgresql-common/pgdg
sudo curl -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc --fail \
  https://www.postgresql.org/media/keys/ACCC4CF8.asc
sudo sh -c 'echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] \
  https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
  > /etc/apt/sources.list.d/pgdg.list'

sudo apt update
sudo apt install -y postgresql-18

# Mulai dan aktifkan service PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Buat user dan database
sudo -u postgres psql -c "CREATE USER koran_user WITH ENCRYPTED PASSWORD 'koran_password_2026';"
sudo -u postgres psql -c "CREATE DATABASE koran_ai_prod OWNER koran_user;"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE koran_ai_prod TO koran_user;"
```

---

## 3. Install Redis 8

```bash
# Install Redis dari package manager
sudo apt install -y redis-server

# Mulai dan aktifkan service Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Verifikasi koneksi
redis-cli ping
# Expected: PONG
```

---

## 4. Deploy Backend Go Binary

```bash
# Clone repository (atau copy file)
mkdir -p /var/www/koran-ai/backend
cd /path/to/koran-ai/backend

# Build binary untuk Linux amd64
make build
# atau:
GOOS=linux GOARCH=amd64 go build -o bin/koran-api cmd/api/main.go

# Copy binary ke deployment directory
sudo cp bin/koran-api /var/www/koran-ai/backend/koran-api
sudo chmod +x /var/www/koran-ai/backend/koran-api

# Copy dan sesuaikan konfigurasi environment
sudo cp .env.example /var/www/koran-ai/backend/.env
sudo nano /var/www/koran-ai/backend/.env
# Ubah: DB_USER, DB_PASSWORD, DB_NAME, INTERNAL_API_KEY, APP_ENV=production
```

---

## 5. Jalankan Migrasi Database

```bash
# Install golang-migrate CLI (jika belum ada)
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz \
  | tar xvz
sudo mv migrate /usr/local/bin/

# Jalankan migrasi UP
cd /path/to/koran-ai/backend
make migrate-up
# atau langsung:
migrate -path migrations -database "postgres://koran_user:koran_password_2026@localhost:5432/koran_ai_prod?sslmode=disable" up
```

---

## 6. Setup Systemd Service

```bash
# Copy service file
sudo cp scripts/koran-backend.service /etc/systemd/system/koran-backend.service

# Reload systemd daemon
sudo systemctl daemon-reload

# Aktifkan auto-start saat boot
sudo systemctl enable koran-backend

# Mulai service
sudo systemctl start koran-backend

# Cek status
sudo systemctl status koran-backend

# Cek log real-time
sudo journalctl -u koran-backend -f
```

---

## 7. Verifikasi Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "success": true,
  "message": "OK",
  "data": {
    "service": "koran-ai",
    "database": "UP",
    "redis": "UP",
    "timestamp": "2026-06-05T18:00:00Z"
  }
}
```

Jika `database` atau `redis` mengembalikan `"DOWN"`, cek koneksi dengan:
```bash
# Test PostgreSQL
psql -U koran_user -h localhost -d koran_ai_prod -c "SELECT version();"

# Test Redis
redis-cli ping
```

---

## 8. Perintah Umum Makefile

```bash
# Menjalankan server lokal (development)
make run

# Build binary produksi
make build

# Menjalankan semua unit tests
make test

# Migrasi naik
make migrate-up

# Migrasi turun (rollback semua)
make migrate-down
```

---

## 9. Troubleshooting Umum

| Masalah | Solusi |
|---|---|
| Service backend tidak menyala | `sudo journalctl -u koran-backend -n 50` untuk melihat error |
| Database connection refused | Pastikan PostgreSQL berjalan: `sudo systemctl status postgresql` |
| Redis connection refused | Pastikan Redis berjalan: `sudo systemctl status redis-server` |
| Port 8080 sudah digunakan | Ganti `APP_PORT` di `.env` atau periksa proses yang menggunakan port: `sudo lsof -i :8080` |
