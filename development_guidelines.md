# Coding & Folder Structure Guidelines: Koran AI Indonesia
## Perancang: Senior Software Architect & Tech Lead

Dokumen ini mendefinisikan struktur repositori tunggal (monorepo), aturan penulisan kode (coding standards), serta arsitektur backend/frontend untuk **Koran AI Indonesia** guna menjamin konsistensi pengembangan sistem oleh tim maupun AI Coding Agent.

---

## 1. Project Structure (Struktur Repositori)

Repositori diatur sebagai monorepo tanpa Docker, memisahkan backend (Go 1.24) dan frontend (Nuxt 4) ke folder independen.

```
koran-ai/
├── backend/                  # Kode Sumber Backend (Go 1.24)
│   ├── cmd/
│   │   └── api/
│   │       └── main.go       # Entry point aplikasi web API
│   ├── internal/             # Kode Logika Bisnis (Restricted/Private)
│   │   ├── source/           # Modul Sumber Berita
│   │   ├── crawler/          # Modul Ingestor & Parser Crawler
│   │   ├── article/          # Modul Artikel Mentah & Filter
│   │   ├── cluster/          # Modul Pengelompokan Berita (Embeddings)
│   │   ├── summary/          # Modul Rangkuman & Rewrite AI
│   │   ├── edition/          # Modul Pembuat Edisi Koran
│   │   ├── search/           # Modul Full-Text Search (FTS)
│   │   ├── trending/         # Modul Kalkulasi Topik Hangat
│   │   └── shared/           # Helper & Inisialisasi Bersama
│   │       ├── config/       # Parser Environment Variables (.env)
│   │       ├── database/     # Handler PostgreSQL (pgx) & Redis (go-redis)
│   │       ├── middleware/   # Custom Fiber Middleware (Logger, API Key)
│   │       ├── validator/    # Validator struct Go (go-playground/validator)
│   │       ├── logger/       # Wrapper structured log (Zap/Slog)
│   │       └── response/     # Helper format response JSON seragam
│   ├── migrations/           # File Migrasi Database DDL
│   ├── scripts/              # Backup & Deployment automation shell scripts
│   ├── go.mod
│   └── go.sum
│
└── frontend/                 # Kode Sumber Frontend (Nuxt 4 / Vue 3)
    ├── components/           # UI Components Reusable (Layout koran, kolom)
    │   ├── global/           # Komponen global (Header koran, footer)
    │   └── ui/               # Komponen elementar (Button, Input, Badge)
    ├── pages/                # Struktur Routing Halaman (Vue Pages)
    │   ├── index.vue         # Beranda (Edisi Terbaru)
    │   ├── edisi/
    │   │   ├── index.vue     # Arsip List Kalender
    │   │   └── [id].vue      # Detail Edisi Khusus
    │   ├── baca/
    │   │   └── [id].vue      # Detail Rangkuman Berita AI
    │   ├── kategori/
    │   │   └── [slug].vue    # Berita per Kategori
    │   └── pencarian.vue     # Halaman Hasil Pencarian FTS
    ├── layouts/              # Template Tata Letak Halaman (Newspaper / Clean)
    │   ├── default.vue       # Gaya koran cetak klasik lengkap
    │   └── admin.vue         # Gaya dashboard bersih (opsional)
    ├── composables/          # Vue Composable functions (state, utilities)
    ├── services/             # Integrasi REST API (Wrapper Fetch / Axios)
    ├── types/                # TypeScript Interfaces & Types
    ├── utils/                # Helper JavaScript/TypeScript statis
    ├── public/               # File static (favicon, font klasik serif)
    ├── nuxt.config.ts        # Konfigurasi Nuxt 4
    ├── tailwind.config.js    # Konfigurasi Tailwind CSS
    └── package.json
```

### 1.1 Fungsi Folder Utama Backend
* `cmd/api/main.go`: Menginisialisasi koneksi database, memanggil konfigurasi, melakukan registrasi router Go Fiber v3, dan menjalankan web server.
* `internal/<feature>/`: Setiap fitur diatur menggunakan Clean Architecture mandiri yang memuat sub-folder:
  * `domain.go` atau `entity.go`: Menyimpan representasi model data (struct).
  * `repository.go`: Mengatur manipulasi database mentah (PostgreSQL/Redis).
  * `usecase.go`: Menyimpan aturan bisnis logika utama (validasi bisnis, pemanggilan AI API, alur kalkulasi).
  * `handler.go`: Controller HTTP yang mem-parsing input, memanggil usecase, dan mengembalikan response JSON Fiber.
* `internal/shared/`: Menyimpan modul utilitas global agar tidak terjadi *circular dependency* antar fitur.

### 1.2 Fungsi Folder Utama Frontend
* `components/`: Kumpulan potongan UI. Folder `components/global` untuk elemen permanen seperti header koran klasik berlogo serif besar, dan `components/ui` untuk komponen visual atomic.
* `pages/`: Nuxt 4 menggunakan routing berbasis file. File di dalam folder ini otomatis dipetakan menjadi URL.
* `services/`: Berisi implementasi pemanggilan API backend yang terstruktur dan ter-tipe menggunakan TypeScript.

---

## 2. Coding Standards (Standar Penulisan Kode)

### 2.1 Naming Convention
* **Go (Backend)**:
  * Gunakan **CamelCase** (bukan snake_case) untuk seluruh variabel, fungsi, struct, dan parameter di Go.
  * Nama variabel privat dimulai huruf kecil (`dbConn`), variabel publik dimulai huruf besar (`DBConn`).
  * Singkatan harus konsisten ditulis kapital (gunakan `APIKey` bukan `ApiKey`, `UUID` bukan `Uuid`).
  * Nama file menggunakan `snake_case.go` (contoh: `source_repository.go`).
* **Vue/TypeScript (Frontend)**:
  * Nama file component menggunakan **PascalCase** (contoh: `NewspaperHeader.vue`).
  * Nama variabel & fungsi menggunakan **camelCase** (contoh: `fetchLatestEdition()`).
  * File halaman menggunakan **kebab-case** sesuai aturan URL (contoh: `pencarian.vue`, `[slug].vue`).

### 2.2 Package Convention (Go)
* Nama package di Go harus pendek, bermakna, menggunakan huruf kecil semua tanpa separator (contoh: `package source`, `package crawler`, `package config`).
* Hindari nama package umum yang ambigu seperti `package helper` atau `package utils`. Gunakan package yang spesifik.

### 2.3 Error Handling (Go)
* **Selalu periksa error**: Jangan pernah mengabaikan nilai kembalian error di Go (jangan gunakan `_ = doSomething()`).
* **Error Wrapping**: Tambahkan konteks yang jelas pada error saat meneruskannya menggunakan `fmt.Errorf("context: %w", err)`.
* **Sentinel Error**: Definisikan error statis di tingkat domain (misal: `var ErrSourceNotFound = errors.New("source not found")`) untuk memudahkan pengecekan di handler HTTP.

### 2.4 Logging (Go)
* Logging menggunakan structured logger (slog bawaan Go 1.24 atau Zap).
* Jangan pernah menulis data rahasia (credential, token API) ke dalam file log.
* Tingkatan Log:
  * `Info`: Aliran proses normal (misal: "Crawler started for Source X").
  * `Warn`: Hal janggal tetapi aplikasi masih bisa berjalan (misal: "Gemini API rate limited, retrying...").
  * `Error`: Kegagalan operasi (misal: "Failed to insert article: ...").
  * `Fatal`: Aplikasi tidak bisa menyala saat inisialisasi awal.

### 2.5 Request Validation
* Validasi input request dikerjakan di lapisan **Handler** (delivery) menggunakan tag `validate` dari pustaka `go-playground/validator/v10`.
* Tag validasi dicantumkan pada struct request DTO (Data Transfer Object).
* Contoh validator:
```go
type CreateSourceRequest struct {
	Name      string `json:"name" validate:"required,min=3,max=100"`
	BaseURL   string `json:"base_url" validate:"required,url"`
	RSSURL    string `json:"rss_url" validate:"required,url"`
	SourceType string `json:"source_type" validate:"required,oneof=rss sitemap"`
}
```

### 2.6 Format Response Helper
Untuk memastikan response yang konsisten sesuai API Contract, gunakan fungsi helper terstandarisasi di `internal/shared/response/response.go`:
```go
package response

import "github.com/gofiber/fiber/v3"

type SuccessEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorEnvelope struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type MetaBlock struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type PaginatedEnvelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Meta    MetaBlock   `json:"meta"`
}

func JSON(c fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(SuccessEnvelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c fiber.Ctx, status int, message string, errs ...interface{}) error {
	var detail interface{}
	if len(errs) > 0 {
		detail = errs[0]
	}
	return c.Status(status).JSON(ErrorEnvelope{
		Success: false,
		Message: message,
		Errors:  detail,
	})
}

func Paginated(c fiber.Ctx, status int, data interface{}, page, limit, total, totalPages int) error {
	return c.Status(status).JSON(PaginatedEnvelope{
		Success: true,
		Data:    data,
		Meta: MetaBlock{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
```
Guidelines di atas wajib dipatuhi oleh seluruh pengembang untuk menjamin kualitas kode produksi.
