# API Contract Documentation: Koran AI Indonesia
## Perancang: Senior Backend Architect & API Designer
## Target Framework: Go 1.24 & Go Fiber v3

Dokumen ini mendefinisikan kontrak API antara Frontend (Nuxt 4) dan Backend (Go Fiber v3) untuk MVP **Koran AI Indonesia**.

---

## 1. Spesifikasi Umum

### 1.1 HTTP Status Codes
Backend mengembalikan status code yang relevan sesuai kondisi:
* `200 OK`: Operasi pembacaan (GET) atau pemrosesan sukses.
* `201 Created`: Pembuatan resource baru sukses (POST).
* `400 Bad Request`: Format payload/JSON tidak valid atau query parameter salah.
* `404 Not Found`: Resource yang diminta tidak terdaftar di database.
* `422 Unprocessable Entity / Validation Error`: Validasi field request gagal.
* `409 Conflict`: Bentrok data unik (misal: edisi ganda pada tanggal yang sama).
* `500 Internal Server Error`: Terjadi kegagalan sistem internal.

### 1.2 Format Response Standar

#### A. Response Sukses (Objek)
```json
{
  "success": true,
  "message": "Success",
  "data": {}
}
```

#### B. Response Error (Umum/500/404)
```json
{
  "success": false,
  "message": "Error message"
}
```

#### C. Response Validasi Gagal (422)
Mengembalikan detail field mana yang gagal divalidasi.
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": {
    "q": "query parameter 'q' must be at least 3 characters"
  }
}
```

#### D. Response Terpaginasi (Pagination)
```json
{
  "success": true,
  "data": [],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

---

## 2. Daftar Endpoint (Endpoint Table)

### 2.1 Public API (Base Path: `/api/v1`)

| Method | Endpoint | Deskripsi | Query Parameters |
| :--- | :--- | :--- | :--- |
| **GET** | `/api/v1/home` | Mengambil feed beranda (Headline, Top Stories, Kategori) | - |
| **GET** | `/api/v1/editions/latest` | Mengambil data edisi terbitan paling baru | - |
| **GET** | `/api/v1/editions/{id}` | Mengambil detail edisi berdasarkan ID (UUID) | - |
| **GET** | `/api/v1/editions` | Mengambil daftar edisi lama terpaginasi | `page`, `limit`, `type`, `date` |
| **GET** | `/api/v1/articles/{id}` | Mengambil detail artikel berita terproses hasil AI | - |
| **GET** | `/api/v1/categories` | Mengambil daftar semua kategori berita | - |
| **GET** | `/api/v1/categories/{slug}`| Mengambil daftar artikel di kategori tertentu | `page`, `limit` |
| **GET** | `/api/v1/trending` | Mengambil daftar topik hangat terpopuler saat ini | - |
| **GET** | `/api/v1/search` | Pencarian teks lengkap berita | `q`, `category`, `page`, `limit`, `start_date`, `end_date` |
| **GET** | `/api/v1/archive` | Mengambil arsip kalender edisi terdahulu | `year`, `month`, `page`, `limit` |

### 2.2 Internal API (Base Path: `/internal`, Protected via `X-Internal-Key`)

| Method | Endpoint | Deskripsi | Request Body |
| :--- | :--- | :--- | :--- |
| **POST** | `/internal/crawler/run` | Memicu crawler pada satu sumber berita khusus | `{"source_id": "UUID"}` |
| **POST** | `/internal/crawler/run-all` | Memicu crawler pada semua sumber berita aktif | - |
| **POST** | `/internal/cluster/build` | Memicu AI clustering artikel mentah | - |
| **POST** | `/internal/summary/generate` | Memicu pembuatan rangkuman AI pada klaster baru | - |
| **POST** | `/internal/edition/build` | Membangun draft edisi koran digital | `{"edition_type": "PAGI/SIANG/MALAM"}` |
| **POST** | `/internal/edition/publish` | Mempublikasikan draf edisi menjadi aktif | `{"edition_id": "UUID"}` |
| **POST** | `/internal/trending/rebuild` | Memicu kalkulasi ulang trending topics | - |

---

## 3. Detail Endpoint & Contoh Payload (Go Struct Mappings)

### 3.1 Public API

#### 3.1.1 GET `/api/v1/home`
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Home feed retrieved",
  "data": {
    "latest_edition": {
      "id": "e4f8a123-5678-4a92-b430-bf824128f111",
      "edition_date": "2026-06-05",
      "edition_type": "MALAM",
      "title": "Koran AI - Edisi Malam, Jumat 5 Juni 2026"
    },
    "headline": {
      "cluster_id": "c1b0d2ef-4321-4b71-9251-1d57922d1000",
      "category": "Ekonomi",
      "title": "BI Tahan Suku Bunga Acuan BI-Rate di Level 6,25%",
      "summary_short": "Langkah pre-emptive Bank Indonesia menahan suku bunga acuan guna menstabilkan Rupiah.",
      "image_url": "https://storage.koranai.id/images/cluster_bi_rate.jpg"
    },
    "top_stories": [
      {
        "cluster_id": "c2b0d2ef-4321-4b71-9251-1d57922d1001",
        "category": "Nasional",
        "title": "Tarif KRL Kategori NIK Berlaku Bulan Depan",
        "summary_short": "Kemenhub bersiap merilis skema tarif baru KRL Commuter Line berbasis NIK.",
        "image_url": "https://storage.koranai.id/images/krl_nik.jpg"
      }
    ],
    "trending_topics": [
      {
        "topic": "Suku Bunga BI",
        "article_count": 12,
        "category": "Ekonomi"
      }
    ]
  }
}
```

---

#### 3.1.2 GET `/api/v1/editions/latest`
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Latest edition retrieved",
  "data": {
    "edition": {
      "id": "e4f8a123-5678-4a92-b430-bf824128f111",
      "edition_date": "2026-06-05",
      "edition_type": "MALAM",
      "title": "Koran AI - Edisi Malam, Jumat 5 Juni 2026",
      "published_at": "2026-06-05T18:00:00Z"
    },
    "sections": {
      "FRONT_PAGE": [
        {
          "cluster_id": "c1b0d2ef-4321-4b71-9251-1d57922d1000",
          "title": "BI Tahan Suku Bunga Acuan BI-Rate di Level 6,25%",
          "summary_short": "Bank Indonesia mempertahankan BI-Rate pada RDG Juni untuk menjaga inflasi.",
          "image_url": "https://storage.koranai.id/images/cluster_bi_rate.jpg",
          "priority": 1,
          "position": 1
        }
      ],
      "NASIONAL": [
        {
          "cluster_id": "c2b0d2ef-4321-4b71-9251-1d57922d1001",
          "title": "Tarif KRL Berbasis NIK Mulai Disosialisasikan",
          "summary_short": "Sosialisasi tarif berbasis NIK menuai ragam respons dari pengguna komuter Jabodetabek.",
          "image_url": "https://storage.koranai.id/images/krl_nik.jpg",
          "priority": 2,
          "position": 1
        }
      ]
    }
  }
}
```

---

#### 3.1.3 GET `/api/v1/editions/{id}`
* **Validation**: ID harus berformat UUID v4 valid. Jika salah format -> `400 Bad Request`. Jika data tidak ada -> `404 Not Found`.
* **Response (200 OK)**: Format data serupa dengan `/api/v1/editions/latest` tetapi menyajikan edisi dengan UUID spesifik.

---

#### 3.1.4 GET `/api/v1/editions`
* **Query Parameters (Go Fiber Request Bind)**:
  * `page` (int, default=1, min=1)
  * `limit` (int, default=20, max=50)
  * `type` (string, optional, values: `PAGI`, `SIANG`, `MALAM`)
  * `date` (string, optional, format: `YYYY-MM-DD`)
* **Response (200 OK - Terpaginasi)**:
```json
{
  "success": true,
  "data": [
    {
      "id": "e4f8a123-5678-4a92-b430-bf824128f111",
      "edition_date": "2026-06-05",
      "edition_type": "MALAM",
      "title": "Koran AI - Edisi Malam, Jumat 5 Juni 2026",
      "published_at": "2026-06-05T18:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

---

#### 3.1.5 GET `/api/v1/articles/{id}`
* **Validation**: ID wajib format UUID v4.
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Article details retrieved",
  "data": {
    "cluster_id": "c1b0d2ef-4321-4b71-9251-1d57922d1000",
    "category": {
      "name": "Ekonomi",
      "slug": "ekonomi"
    },
    "headline": "BI Tahan Suku Bunga Acuan BI-Rate di Level 6,25%",
    "subheadline": "Langkah Moneter untuk Memperkuat Penguatan Nilai Tukar Rupiah dan Kendalikan Inflasi",
    "summary": "Jakarta, Koran AI - Rapat Dewan Gubernur Bank Indonesia memutuskan untuk mempertahankan suku bunga acuan BI-Rate sebesar 6,25%...",
    "impact": "Suku bunga kredit perbankan berpotensi tertahan di level tinggi, sementara aliran modal asing diperkirakan masuk kembali ke pasar domestik.",
    "context": "Keputusan ini diambil setelah nilai tukar rupiah mengalami tekanan eksternal akibat penundaan pemotongan suku bunga bank sentral AS (The Fed).",
    "references": [
      {
        "source_name": "Kompas",
        "url": "https://ekonomi.kompas.com/read/2026/06/..."
      },
      {
        "source_name": "Detik Finance",
        "url": "https://finance.detik.com/read/2026/06/..."
      }
    ],
    "images": [
      {
        "url": "https://storage.koranai.id/images/cluster_bi_rate.jpg",
        "caption": "Gedung Bank Indonesia di Jakarta."
      }
    ]
  }
}
```

---

#### 3.1.6 GET `/api/v1/categories/{slug}`
* **Query Parameters**: `page` (default=1), `limit` (default=20)
* **Response (200 OK - Terpaginasi)**:
```json
{
  "success": true,
  "data": [
    {
      "cluster_id": "c1b0d2ef-4321-4b71-9251-1d57922d1000",
      "title": "BI Tahan Suku Bunga Acuan BI-Rate di Level 6,25%",
      "summary_short": "Keputusan Dewan Gubernur Bank Indonesia menahan suku bunga di 6,25%...",
      "image_url": "https://storage.koranai.id/images/cluster_bi_rate.jpg",
      "created_at": "2026-06-05T18:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 120,
    "total_pages": 6
  }
}
```

---

#### 3.1.7 GET `/api/v1/search`
* **Query Parameters & Validation (Go Struct Binder)**:
  * `q` (string, required, min=3): Kata kunci.
  * `category` (string, optional): Slug kategori.
  * `start_date` (string, optional, format: YYYY-MM-DD).
  * `end_date` (string, optional, format: YYYY-MM-DD).
  * `page` (int, default=1).
  * `limit` (int, default=20).
* **Validation Failure (422 Validation Error)**: Jika parameter `q` kurang dari 3 karakter.
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": {
    "q": "kata kunci pencarian 'q' minimal harus 3 karakter"
  }
}
```
* **Response (200 OK - Terpaginasi)**:
```json
{
  "success": true,
  "data": [
    {
      "cluster_id": "c1b0d2ef-4321-4b71-9251-1d57922d1000",
      "title": "BI Tahan Suku Bunga Acuan BI-Rate di Level 6,25%",
      "excerpt": "...mempertahankan BI-Rate sebesar 6,25%...",
      "category": "Ekonomi",
      "image_url": "https://storage.koranai.id/images/cluster_bi_rate.jpg",
      "created_at": "2026-06-05T18:00:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### 3.2 Internal API (Protected via Header `X-Internal-Key`)

Setiap request ke `/internal/*` wajib menyertakan header:
* `X-Internal-Key: <secret_api_key>`
Jika tidak valid atau tidak disertakan -> `401 Unauthorized` atau `403 Forbidden`.

#### 3.2.1 POST `/internal/crawler/run`
* **Request Struct Validation**:
```go
type RunCrawlerRequest struct {
	SourceID string `json:"source_id" validate:"required,uuid4"`
}
```
* **Request Payload**:
```json
{
  "source_id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
}
```
* **Response (202 Accepted)**:
```json
{
  "success": true,
  "message": "Crawler job scheduled successfully",
  "data": {
    "job_id": "f8d38b02-5cde-4702-861f-13a8549e3940"
  }
}
```

---

#### 3.2.2 POST `/internal/cluster/build`
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "AI clustering completed",
  "data": {
    "cluster_count": 14
  }
}
```

---

#### 3.2.3 POST `/internal/summary/generate`
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "AI summaries generated successfully",
  "data": {
    "processed_count": 14
  }
}
```

---

#### 3.2.4 POST `/internal/edition/build`
* **Request Struct Validation**:
```go
type BuildEditionRequest struct {
	EditionType string `json:"edition_type" validate:"required,oneof=PAGI SIANG MALAM"`
}
```
* **Request Payload**:
```json
{
  "edition_type": "MALAM"
}
```
* **Response (201 Created)**:
```json
{
  "success": true,
  "message": "Edition draft generated successfully",
  "data": {
    "edition_id": "e4f8a123-5678-4a92-b430-bf824128f111"
  }
}
```

---

#### 3.2.5 POST `/internal/edition/publish`
* **Request Struct Validation**:
```go
type PublishEditionRequest struct {
	EditionID string `json:"edition_id" validate:"required,uuid4"`
}
```
* **Request Payload**:
```json
{
  "edition_id": "e4f8a123-5678-4a92-b430-bf824128f111"
}
```
* **Response (200 OK)**:
```json
{
  "success": true,
  "message": "Edition published successfully",
  "data": {
    "edition_id": "e4f8a123-5678-4a92-b430-bf824128f111",
    "status": "PUBLISHED"
  }
}
```
