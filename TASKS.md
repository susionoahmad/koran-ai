# TASKS & ROADMAP PLAN: KORAN AI INDONESIA
## Perancang: Engineering Manager & Tech Lead

Dokumen ini berisi panduan tahap-demi-tahap (milestones), pembagian fase kerja, rancangan roadmap jangka panjang, serta standar Definition of Done (DoD) untuk proyek **Koran AI Indonesia**.

---

## 1. Development Roadmap (Peta Jalan Pengembangan)

* **MVP (Minimum Viable Product) - 30 Hari**: Portal berita digital bergaya koran klasik yang mengumpulkan data secara otomatis via RSS/Sitemap, melakukan pemrosesan klasifikasi/clustering/summarization via Gemini API, serta menampilkan 3 edisi harian (Pagi, Siang, Malam) lengkap dengan fitur Search (FTS) dan Arsip. Tanpa sistem login/premium.
* **V1 (Production Launch) - 90 Hari**: Dashboard Admin lengkap untuk moderasi manual hasil AI, optimasi performa cache halaman (Redis), table partitioning penuh pada PostgreSQL, visualisasi monitoring AI Jobs & crawler log, serta perbaikan visual layout multi-kolom yang dinamis di berbagai ukuran layar.
* **V2 (Monetization & Scale) - 180 Hari**: Sistem langganan premium (paywall) terintegrasi Midtrans, pengiriman E-Paper PDF otomatis ke email pelanggan, newsletter berbayar, rekomendasi topik AI personal, dan ekspansi crawler ke situs berita berbahasa daerah/lokal.

---

## 2. Rencana Fase Kerja (Tasks by Phase)

Setiap fase dirancang untuk durasi **1 - 3 hari kerja**.

### PHASE 1: Infrastructure Setup (Durasi: 2 Hari)
* **Task 1.1**: Inisialisasi proyek monorepo (Go module pada `backend/` dan proyek Nuxt 4 pada `frontend/`).
* **Task 1.2**: Setup konfigurasi backend (`config.env` dan file parser menggunakan package `shared/config`).
* **Task 1.3**: Konfigurasi logging terstruktur menggunakan `shared/logger` (wrapper slog/zap).
* **Task 1.4**: Setup koneksi database PostgreSQL 18 native (menggunakan driver pgx pool) dan Redis native.
* **Task 1.5**: Jalankan SQL DDL awal (`migrations_up.sql`) dan buat utility scripts untuk migrasi naik/turun.
* **Task 1.6**: Implementasikan REST endpoint `/api/v1/health` untuk memonitor kesiapan DB & Redis.

### PHASE 2: Source Module (Durasi: 2 Hari)
* **Task 2.1**: Buat entitas domain `Source` (struct Go) dan skema validasi request CRUD.
* **Task 2.2**: Buat `SourceRepository` untuk interaksi data master sumber berita dengan PostgreSQL.
* **Task 2.3**: Buat `SourceUsecase` untuk menampung aturan bisnis (misal: validasi keaktifan URL).
* **Task 2.4**: Buat `SourceHandler` (Go Fiber Router) untuk CRUD API (`GET /api/v1/sources`, `POST`, `PUT`, `DELETE`).

### PHASE 3: Crawler Engine (Durasi: 3 Hari)
* **Task 3.1**: Implementasikan parser RSS XML terstruktur untuk mengambil metadata artikel berita baru.
* **Task 3.2**: Implementasikan parser XML Sitemap untuk mengekstrak URL berita dari situs media nasional.
* **Task 3.3**: Buat HTML content cleaner untuk mengekstrak isi teks bersih dari badan berita, membuang iklan/tracker.
* **Task 3.4**: Implementasikan mekanisme deduplikasi artikel berbasis MD5 hashing (`hash_content`) sebelum disimpan.
* **Task 3.5**: Integrasikan scheduler cron native di Go untuk memicu crawler secara otomatis setiap 30 menit.

### PHASE 4: Article Processing (Durasi: 2 Hari)
* **Task 4.1**: Buat repositori penyimpanan artikel mentah ke tabel `articles` (termasuk penentuan partisi bulanan).
* **Task 4.2**: Implementasikan klasifikasi otomatis kategori berita (8 kategori) menggunakan Gemini 1.5 Flash API.
* **Task 4.3**: Buat antrean pekerjaan (*processing queue*) berbasis Redis untuk memproses artikel mentah secara asinkron.
* **Task 4.4**: Catat log penjelajahan crawler ke tabel `crawl_logs` untuk audit internal.

### PHASE 5: AI Engine (Durasi: 3 Hari)
* **Task 5.1**: Desain prompt Gemini API untuk menghasilkan ringkasan (summary) objektif bebas clickbait.
* **Task 5.2**: Integrasikan pipeline AI untuk mengekstrak sub-headline, poin fakta penting, dampak, serta konteks berita.
* **Task 5.3**: Buat generator headline utama yang lugas menggunakan Gemini 1.5 Pro.
* **Task 5.4**: Catat status konsumsi token dan estimasi biaya ke tabel `ai_jobs`.

### PHASE 6: Cluster Engine (Durasi: 3 Hari)
* **Task 6.1**: Setup model embedding Google (`text-embedding-004`) untuk mengonversi isi artikel berita menjadi vector.
* **Task 6.2**: Implementasikan pencarian kemiripan vector (*cosine similarity*) menggunakan ekstensi `pgvector` di PostgreSQL.
* **Task 6.3**: Buat logika pembuat klaster (`Cluster Builder`) untuk menggabungkan artikel serupa dalam jendela waktu 12 jam.
* **Task 6.4**: Hubungkan artikel-artikel pendukung ke ID klaster terkait di tabel `cluster_articles`.

### PHASE 7: Edition Builder (Durasi: 3 Hari)
* **Task 7.1**: Implementasikan scheduler pembuat edisi otomatis 3 kali sehari (PAGI, SIANG, MALAM).
* **Task 7.2**: Buat algoritma pembuat draf edisi (AI memilih Top Story untuk halaman depan dan menyusun kategori berita).
* **Task 7.3**: Implementasikan repositori penyimpanan edisi ke tabel `editions` dan `edition_articles`.
* **Task 7.4**: Buat endpoint persetujuan penerbitan draf edisi (`POST /internal/edition/publish`).

### PHASE 8: Public API (Durasi: 2 Hari)
* **Task 8.1**: Implementasikan API `/api/v1/home` (Headline utama, Top Stories, dan list kategori aktif).
* **Task 8.2**: Buat API `/api/v1/editions/latest` dan detail edisi `/api/v1/editions/{id}`.
* **Task 8.3**: Buat API `/api/v1/articles/{id}` untuk menyajikan rangkuman berita beserta daftar URL referensi media asli.
* **Task 8.4**: Implementasikan pencarian teks penuh `/api/v1/search` menggunakan PostgreSQL `tsvector` terindeks GIN.
* **Task 8.5**: Sediakan kalender arsip `/api/v1/archive` terpaginasi.

### PHASE 9: Frontend Layout & UI (Durasi: 3 Hari)
* **Task 9.1**: Desain gaya visual koran cetak digital klasik di Nuxt 4 (Font Playfair Display / Georgia, grid hitam tebal).
* **Task 9.2**: Buat halaman beranda (`index.vue`) dengan layout multi-kolom yang adaptif.
* **Task 9.3**: Buat halaman detail baca rangkuman berita (`baca/[id].vue`) dengan kotak poin penting & dampak.
* **Task 9.4**: Bangun halaman arsip kalender edisi terdahulu dan integrasikan dengan API backend.

### PHASE 10: SEO & Metadata (Durasi: 2 Hari)
* **Task 10.1**: Konfigurasikan dynamic meta tags (title, description, open graph) di setiap halaman baca Nuxt 4.
* **Task 10.2**: Implementasikan generator otomatis file `sitemap.xml` yang memuat tautan seluruh edisi koran.
* **Task 10.3**: Buat file `robots.txt` standar produksi.
* **Task 10.4**: Tambahkan JSON-LD Structured Data (skema NewsArticle) pada halaman baca berita untuk optimasi Google Search.

### PHASE 11: Monitoring & Logging Dashboard (Durasi: 2 Hari)
* **Task 11.1**: Buat endpoint internal untuk memantau status crawler (`GET /internal/crawler/status`).
* **Task 11.2**: Implementasikan logging error AI pipeline dan query lambat database ke log file server.
* **Task 11.3**: Buat visualisasi status antrean Redis (`ai_jobs`) untuk mendeteksi penumpukan tugas AI.

### PHASE 12: Deployment & Server Configuration (Durasi: 3 Hari)
* **Task 12.1**: Compile binary Go backend dan pasang ke server Debian 24.04 menggunakan `systemd` service manager.
* **Task 12.2**: Jalankan build statis/SSR frontend Nuxt 4 dan atur prosesnya menggunakan Node.js/PM2.
* **Task 12.3**: Konfigurasikan Nginx sebagai reverse proxy dengan SSL HTTPS dari Let's Encrypt (Certbot).
* **Task 12.4**: Pasang script backup otomatis PostgreSQL harian ke cloud storage S3 compatible.

---

## 3. Definition of Done (DoD) per Fase

Fase dinyatakan selesai jika memenuhi seluruh kriteria centang (checkpoints) berikut:

* **PHASE 1 (Infrastructure Setup)** selesai jika:
  ✓ Proyek backend Go menyala dan frontend Nuxt 4 dapat diakses di localhost.
  ✓ Migrasi PostgreSQL berhasil dijalankan tanpa error sintaks.
  ✓ Health check endpoint `/api/v1/health` mengembalikan status sukses untuk koneksi DB dan Redis.

* **PHASE 2 (Source Module)** selesai jika:
  ✓ API CRUD untuk mengelola sumber berita berfungsi penuh dan divalidasi.
  ✓ Unit test pada repository & usecase Source lulus 100%.

* **PHASE 3 (Crawler Engine)** selesai jika:
  ✓ Script crawler berhasil mem-parsing artikel baru dari setidaknya 3 alamat RSS feed media berbeda.
  ✓ Duplikasi berita yang sama terdeteksi secara akurat via MD5 hash.

* **PHASE 4 (Article Processing)** selesai jika:
  ✓ Artikel tersimpan rapi di tabel PostgreSQL (masuk ke partisi bulan yang bersangkutan).
  ✓ Klasifikasi kategori (8 kategori utama) berhasil dieksekusi oleh Gemini API tanpa kegagalan parsing JSON.

* **PHASE 5 (AI Engine)** selesai jika:
  ✓ Gemini API menghasilkan rangkuman objektif, poin penting, konteks, dan dampak berita.
  ✓ Biaya token tersimpan akurat pada tabel `ai_jobs`.

* **PHASE 6 (Cluster Engine)** selesai jika:
  ✓ Vector embeddings terbentuk dan proses grouping berita sejenis via `pgvector` berjalan cepat (< 200ms).
  ✓ Pivot table `cluster_articles` terisi relasi secara tepat.

* **PHASE 7 (Edition Builder)** selesai jika:
  ✓ Draft edisi pagi, siang, dan malam dapat dihasilkan otomatis berdasarkan rentang waktu ingest artikel.
  ✓ Pengubahan status draft menjadi `PUBLISHED` merilis data secara instan ke publik.

* **PHASE 8 (Public API)** selesai jika:
  ✓ Endpoint API Home, Edisi, Detail Artikel, Kategori, Search, dan Arsip mengembalikan data sesuai API Contract.
  ✓ Full-Text Search PostgreSQL (GIN index) merespon pencarian kata kunci dengan performa tinggi.

* **PHASE 9 (Frontend Layout & UI)** selesai jika:
  ✓ Tampilan koran cetak digital klasik ter-render rapi tanpa layout rusak pada desktop maupun smartphone.
  ✓ Interaksi perpindahan halaman (routing) berjalan lancar dan bebas error Javascript.

* **PHASE 10 (SEO & Metadata)** selesai jika:
  ✓ Pengujian Google Rich Results Test mendeteksi skema NewsArticle terpasang dengan benar pada halaman berita.
  ✓ File `sitemap.xml` dan `robots.txt` termuat sempurna di browser.

* **PHASE 11 (Monitoring & Logging Dashboard)** selesai jika:
  ✓ Tim pengembang dapat memantau logs kegagalan crawling & antrean AI secara real-time.

* **PHASE 12 (Deployment & Server Configuration)** selesai jika:
  ✓ Aplikasi backend Go dan frontend Nuxt berjalan kokoh di server Debian 24.04 pasca restart OS (auto-start via systemd).
  ✓ Domain menggunakan HTTPS (A+ SSL Labs rating) dan backup database otomatis terjadwal via cron.
