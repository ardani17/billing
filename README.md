# ISPBoss

Platform billing dan manajemen jaringan all-in-one untuk ISP dan RT/RW Net.

## Quick Start

```bash
# Clone repository
git clone https://github.com/ispboss/ispboss.git
cd ispboss

# Copy environment file
cp .env.example .env

# Jalankan dengan Docker
make docker-up

# Atau jalankan development mode
npm install
make dev
```

## Struktur Monorepo

```
apps/web/              → Next.js frontend (port 3000)
services/billing-api/  → Go service (port 3001)
services/network-service/ → Go service (port 3002)
services/notification/ → Go service (port 3003)
pkg/                   → Shared Go packages
packages/              → Shared TypeScript packages
api-tests/             → Bruno API test collection
docker/                → Docker Compose + Dockerfiles
```

## Makefile Targets

```bash
make dev          # Jalankan semua service (development)
make build        # Build semua aplikasi
make test         # Jalankan semua test
make lint         # Jalankan linter (Go + TS)
make swagger      # Generate Swagger docs
make migrate-up   # Jalankan migrasi database
make migrate-down # Rollback migrasi database
make docker-up    # Jalankan Docker Compose
make docker-down  # Hentikan Docker Compose
make help         # Tampilkan semua target
```

## Tech Stack

| Layer | Teknologi |
|---|---|
| Frontend | Next.js 15, Tailwind CSS v4, shadcn/ui |
| Backend | Go (Fiber v2), Clean Architecture |
| Database | PostgreSQL 16 (pgx + sqlc) |
| Cache/Queue | Redis 7 (asynq) |
| Auth | JWT (golang-jwt) |
| Logging | zerolog |
| Config | Viper |

## Coding Standards

### Bahasa Komentar
- **Komentar kode WAJIB dalam Bahasa Indonesia**
- Nama variabel, fungsi, struct, dan interface dalam **bahasa Inggris**

### Batas File
- **Maksimal 200 baris per file**
- Jika melebihi 200 baris, pecah ke file terpisah

### Arsitektur
- Clean architecture: domain → usecase → repository → handler
- Domain layer **tidak boleh** import dari handler atau repository
- Setiap file punya 1 tanggung jawab jelas

### Multi-Tenant
- Semua tabel tenant-scoped WAJIB punya kolom `tenant_id`
- Row Level Security (RLS) sebagai safety net
- Setiap query WAJIB filter `tenant_id`

## Environment Variables

Lihat `.env.example` untuk daftar lengkap variabel environment.

## License

MIT
