# Requirements Document — Monorepo Setup (ISPBoss)

## Introduction

Dokumen ini mendefinisikan requirements untuk setup fondasi proyek ISPBoss — sebuah SaaS billing dan network management platform untuk ISP dan RT/RW Net. Spec ini mencakup: monorepo structure (Turborepo), project scaffolding (Next.js + Go services), database setup (PostgreSQL multi-tenant dengan RLS), Redis setup (caching + asynq queue), Docker Compose development environment, shared Go packages, dan konfigurasi dasar. Ini adalah spec pertama yang menjadi fondasi semua spec berikutnya.

## Glossary

- **Monorepo**: Repository tunggal yang berisi semua aplikasi, service, dan shared package dalam satu codebase, dikelola dengan Turborepo
- **Turborepo**: Build system untuk monorepo JavaScript/TypeScript yang mengelola task orchestration dan caching
- **Tenant**: Operator ISP/RT-RW Net yang berlangganan ISPBoss; setiap tenant memiliki data terisolasi via tenant_id
- **RLS (Row Level Security)**: Fitur PostgreSQL yang membatasi akses baris data berdasarkan policy, digunakan sebagai safety net isolasi tenant
- **Billing_API**: Service Golang utama yang menangani customer, invoice, payment, auth, dan RBAC
- **Network_Service**: Service Golang yang menangani integrasi MikroTik, OLT, dan FTTH mapping
- **Notification_Service**: Service Golang yang menangani pengiriman notifikasi (WhatsApp, SMS, Email)
- **Web_App**: Aplikasi frontend Next.js di `apps/web/`
- **Shared_Go_Package**: Package Go di direktori `pkg/` yang digunakan bersama oleh semua Go service
- **Shared_TS_Package**: Package TypeScript di direktori `packages/` yang digunakan oleh Web_App
- **Docker_Compose_Stack**: Kumpulan container Docker yang mendefinisikan development environment lengkap
- **Migration_Runner**: Tool golang-migrate yang menjalankan file SQL migrasi database secara berurutan
- **Tenant_Context**: Mekanisme propagasi tenant_id dari JWT middleware ke semua layer aplikasi via Go context
- **Asynq**: Library Go berbasis Redis untuk background job processing dan task queue
- **pgx**: PostgreSQL driver untuk Go dengan dukungan connection pooling
- **sqlc**: Code generator yang menghasilkan type-safe Go code dari SQL query
- **Viper**: Library Go untuk manajemen konfigurasi dari file, environment variable, dan flag

---

## Requirements

### Requirement 1: Monorepo Structure dengan Turborepo

**User Story:** As a developer, I want a well-organized monorepo with Turborepo, so that all applications and services live in one repository with efficient build orchestration.

#### Acceptance Criteria

1. THE Monorepo SHALL contain the following top-level directories: `apps/`, `services/`, `pkg/`, `packages/`, `api-tests/`, `docker/`
2. THE Monorepo SHALL include a root `turbo.json` configuration file that defines build, lint, dev, and test pipeline tasks
3. THE Monorepo SHALL include a root `package.json` that defines workspaces for `apps/*` and `packages/*`
4. THE Monorepo SHALL include a root `go.work` file that defines Go workspaces for `services/*` and `pkg/*`
5. THE Monorepo SHALL include a `.gitignore` file that excludes build artifacts, node_modules, vendor directories, `.env` files, and IDE-specific files
6. THE Monorepo SHALL include a root `Makefile` that provides targets for common operations: `dev`, `build`, `test`, `lint`, `migrate-up`, `migrate-down`, `docker-up`, `docker-down`
7. WHEN `turbo run build` is executed, THE Turborepo SHALL build all applications and services respecting dependency order
8. WHEN `turbo run test` is executed, THE Turborepo SHALL run tests for all applications and services

---

### Requirement 2: Next.js Frontend Application Scaffolding

**User Story:** As a frontend developer, I want a properly configured Next.js application, so that I can build the ISPBoss dashboard UI.

#### Acceptance Criteria

1. THE Web_App SHALL be located at `apps/web/` and use Next.js with App Router and TypeScript
2. THE Web_App SHALL be configured with Tailwind CSS v4 for styling
3. THE Web_App SHALL include shadcn/ui as the component library foundation
4. THE Web_App SHALL be configured with Geist and Geist Mono fonts
5. THE Web_App SHALL include ESLint and Prettier configuration extending from Shared_TS_Package `packages/config/`
6. THE Web_App SHALL include a `tsconfig.json` extending from Shared_TS_Package `packages/config/`
7. THE Web_App SHALL include environment variable configuration for `NEXT_PUBLIC_API_URL` (default: `http://localhost:3001`)
8. WHEN the Web_App is started in development mode, THE Web_App SHALL be accessible on port 3000

---

### Requirement 3: Go Service Scaffolding (Clean Architecture)

**User Story:** As a backend developer, I want properly structured Go services following clean architecture, so that each service is maintainable and consistent.

#### Acceptance Criteria

1. THE Billing_API SHALL be located at `services/billing-api/` with the directory structure: `cmd/`, `internal/domain/`, `internal/usecase/`, `internal/repository/`, `internal/handler/`, `internal/middleware/`, `internal/config/`
2. THE Network_Service SHALL be located at `services/network-service/` with the same clean architecture directory structure as Billing_API
3. THE Notification_Service SHALL be located at `services/notification/` with the same clean architecture directory structure as Billing_API
4. WHEN a Go service is started, THE Go service SHALL load configuration from environment variables using Viper
5. WHEN a Go service is started, THE Go service SHALL initialize structured logging using zerolog with JSON output format
6. WHEN a Go service is started, THE Go service SHALL expose a `GET /healthz` endpoint that returns HTTP 200 with service name and status
7. THE Billing_API SHALL listen on port 3001 by default
8. THE Network_Service SHALL listen on port 3002 by default
9. THE Notification_Service SHALL listen on port 3003 by default
10. WHEN a Go service receives a request, THE Go service SHALL log the request method, path, status code, and duration using zerolog

---

### Requirement 4: PostgreSQL Database Setup with Multi-Tenant Schema

**User Story:** As a platform engineer, I want a PostgreSQL database with multi-tenant isolation, so that each tenant's data is securely separated.

#### Acceptance Criteria

1. THE Shared_Go_Package `pkg/database/` SHALL provide a connection pool manager using pgx that supports configurable pool size, connection timeout, and max connection lifetime
2. THE Shared_Go_Package `pkg/database/` SHALL provide a function to set `app.tenant_id` as a PostgreSQL session variable before executing tenant-scoped queries
3. THE Migration_Runner SHALL use golang-migrate with SQL migration files stored in `services/billing-api/migrations/`
4. THE Migration_Runner SHALL support `up`, `down`, and `version` commands via Makefile targets
5. WHEN the initial migration is applied, THE Migration_Runner SHALL create a `tenants` table with columns: `id` (UUID, PK), `name`, `domain`, `plan`, `status`, `created_at`, `updated_at`
6. WHEN the initial migration is applied, THE Migration_Runner SHALL enable Row Level Security on the `tenants` table
7. WHEN the initial migration is applied, THE Migration_Runner SHALL create an RLS policy on tenant-scoped tables using `tenant_id = current_setting('app.tenant_id')::uuid`
8. THE Migration_Runner SHALL create a sample tenant-scoped table (e.g., `customers`) with a `tenant_id` column (UUID, NOT NULL, FK to tenants) to demonstrate the RLS pattern
9. IF a query is executed without setting `app.tenant_id` session variable, THEN THE RLS policy SHALL deny access to all rows in tenant-scoped tables
10. THE Shared_Go_Package `pkg/database/` SHALL include sqlc configuration (`sqlc.yaml`) for generating type-safe Go code from SQL queries

---

### Requirement 5: Redis Setup for Caching and Queue

**User Story:** As a backend developer, I want a shared Redis client and asynq task queue, so that all services can use caching and background job processing.

#### Acceptance Criteria

1. THE Shared_Go_Package `pkg/queue/` SHALL provide an asynq client factory that creates asynq clients from Redis connection configuration
2. THE Shared_Go_Package `pkg/queue/` SHALL provide an asynq server factory that creates asynq worker servers with configurable concurrency and queue priorities
3. THE Shared_Go_Package `pkg/queue/` SHALL define a base task envelope struct containing `event_type`, `tenant_id`, `timestamp`, `correlation_id`, and `payload` fields matching the event contract schema
4. WHEN a task is enqueued, THE Shared_Go_Package `pkg/queue/` SHALL serialize the task payload as JSON
5. WHEN a task is dequeued, THE Shared_Go_Package `pkg/queue/` SHALL deserialize the task payload from JSON back to the envelope struct
6. THE Shared_Go_Package `pkg/queue/` SHALL support configurable Redis connection parameters: host, port, password, and database number

---

### Requirement 6: Docker Compose Development Environment

**User Story:** As a developer, I want a Docker Compose setup that runs all services and dependencies, so that I can develop locally with a single command.

#### Acceptance Criteria

1. THE Docker_Compose_Stack SHALL be defined in `docker/docker-compose.yml`
2. THE Docker_Compose_Stack SHALL include services for: PostgreSQL (port 5432), Redis (port 6379), Billing_API (port 3001), Network_Service (port 3002), Notification_Service (port 3003)
3. THE Docker_Compose_Stack SHALL use named volumes for PostgreSQL data persistence and Redis data persistence across container restarts
4. THE Docker_Compose_Stack SHALL include a `docker/.env.example` file documenting all required environment variables with sensible defaults
5. WHEN `docker compose up` is executed, THE Docker_Compose_Stack SHALL start all services with correct dependency ordering (PostgreSQL and Redis start before Go services)
6. THE Docker_Compose_Stack SHALL include health checks for PostgreSQL and Redis that Go services depend on before starting
7. THE Docker_Compose_Stack SHALL set `NETWORK_MODE=mock` as the default environment variable for Network_Service
8. THE Docker_Compose_Stack SHALL include Dockerfiles for each Go service using multi-stage builds (builder + runtime) to minimize image size

---

### Requirement 7: Shared Go Packages

**User Story:** As a backend developer, I want shared Go packages for cross-cutting concerns, so that all services use consistent implementations for logging, auth, tenant context, and configuration.

#### Acceptance Criteria

1. THE Shared_Go_Package `pkg/logger/` SHALL provide a zerolog logger factory that creates pre-configured loggers with JSON output, timestamp, service name, and configurable log level
2. THE Shared_Go_Package `pkg/auth/` SHALL provide JWT token generation and validation functions using golang-jwt
3. THE Shared_Go_Package `pkg/auth/` SHALL support embedding `tenant_id`, `user_id`, and `role` claims in JWT tokens
4. WHEN a JWT token is validated, THE Shared_Go_Package `pkg/auth/` SHALL return the decoded claims or an error if the token is invalid or expired
5. THE Shared_Go_Package `pkg/tenant/` SHALL provide a middleware function that extracts `tenant_id` from JWT claims and injects the value into the Go request context
6. THE Shared_Go_Package `pkg/tenant/` SHALL provide helper functions to retrieve `tenant_id` from Go request context
7. WHEN the tenant middleware receives a request without a valid `tenant_id` in the JWT, THEN THE Shared_Go_Package `pkg/tenant/` SHALL return HTTP 401 Unauthorized
8. THE Shared_Go_Package `pkg/logger/` SHALL support log levels: debug, info, warn, error, fatal

---

### Requirement 8: Shared TypeScript Packages

**User Story:** As a frontend developer, I want shared TypeScript packages for configuration and types, so that all frontend code follows consistent standards.

#### Acceptance Criteria

1. THE Shared_TS_Package `packages/config/` SHALL export a shared ESLint configuration for Next.js projects
2. THE Shared_TS_Package `packages/config/` SHALL export a shared TypeScript configuration (`tsconfig.base.json`) with strict mode enabled
3. THE Shared_TS_Package `packages/config/` SHALL export a shared Prettier configuration
4. THE Shared_TS_Package `packages/types/` SHALL export a base API response type with fields: `success` (boolean), `data` (generic), `error` (optional object with `code` and `message`)
5. THE Shared_TS_Package `packages/ui/` SHALL be initialized as a shadcn/ui component package that Web_App can import from

---

### Requirement 9: Configuration Management

**User Story:** As a developer, I want a consistent configuration approach across all services, so that environment-specific settings are easy to manage.

#### Acceptance Criteria

1. WHEN a Go service starts, THE Go service SHALL read configuration from environment variables with a fallback to a `.env` file in the service directory
2. THE Go service configuration SHALL support the following base variables: `APP_NAME`, `APP_PORT`, `APP_ENV` (development/staging/production), `LOG_LEVEL`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSL_MODE`, `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD`, `JWT_SECRET`, `JWT_EXPIRY`
3. THE Network_Service configuration SHALL additionally support `NETWORK_MODE` (mock/live) with default value `mock`
4. THE root directory SHALL include a `.env.example` file documenting all environment variables with descriptions and default values
5. IF a required environment variable is missing at startup, THEN THE Go service SHALL log an error message specifying the missing variable and exit with a non-zero status code

---

### Requirement 10: Multi-Tenant Isolation Middleware

**User Story:** As a platform engineer, I want tenant isolation enforced at every layer, so that no tenant can access another tenant's data.

#### Acceptance Criteria

1. WHEN a request passes through the tenant middleware, THE tenant middleware SHALL extract `tenant_id` from the JWT claims and set the PostgreSQL session variable `app.tenant_id` before any database query is executed
2. WHEN a request passes through the tenant middleware, THE tenant middleware SHALL inject `tenant_id` into the Go request context for use by all downstream handlers
3. IF a request does not contain a valid JWT with `tenant_id` claim, THEN THE tenant middleware SHALL reject the request with HTTP 401 and a JSON error response
4. THE RLS policy on tenant-scoped tables SHALL ensure that SELECT, INSERT, UPDATE, and DELETE operations only affect rows where `tenant_id` matches `current_setting('app.tenant_id')::uuid`
5. FOR ALL tenant-scoped database tables, THE Migration_Runner SHALL create an index on the `tenant_id` column
6. FOR ALL valid tenant_id values, setting `app.tenant_id` then querying a tenant-scoped table then parsing the results SHALL return only rows belonging to that tenant (round-trip isolation property)

---

### Requirement 11: Health Check and Service Discovery

**User Story:** As a DevOps engineer, I want health check endpoints on every service, so that Docker and orchestrators can monitor service availability.

#### Acceptance Criteria

1. WHEN a GET request is sent to `/healthz`, THE Go service SHALL return HTTP 200 with a JSON body containing `status` ("ok"), `service` (service name), and `timestamp` (ISO 8601)
2. WHEN a GET request is sent to `/readyz`, THE Go service SHALL check database and Redis connectivity and return HTTP 200 if all dependencies are reachable, or HTTP 503 with details of which dependency is unreachable
3. THE Docker_Compose_Stack SHALL use the `/healthz` endpoint for container health checks with an interval of 10 seconds and a timeout of 5 seconds
4. IF the database connection is lost, THEN THE `/readyz` endpoint SHALL return HTTP 503 with `"database": "unreachable"` in the response body
5. IF the Redis connection is lost, THEN THE `/readyz` endpoint SHALL return HTTP 503 with `"redis": "unreachable"` in the response body

---

### Requirement 12: API Test Infrastructure

**User Story:** As a QA engineer, I want a Bruno API test collection scaffolding, so that API tests are version-controlled alongside the code.

#### Acceptance Criteria

1. THE Monorepo SHALL include an `api-tests/` directory with a Bruno collection configuration file (`bruno.json`)
2. THE Bruno collection SHALL include an `environments/` directory with `local.bru` environment file containing base URL variables for all services
3. THE Bruno collection SHALL include a sample health check request for each Go service (`billing-api/healthz.bru`, `network-service/healthz.bru`, `notification/healthz.bru`)

---

### Requirement 13: Code Quality and Linting

**User Story:** As a developer, I want consistent code quality tooling, so that all code follows the same standards.

#### Acceptance Criteria

1. THE Monorepo SHALL include a `golangci-lint` configuration file (`.golangci.yml`) at the root with linters enabled: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`
2. THE Makefile SHALL include a `lint` target that runs `golangci-lint` for all Go services and ESLint for the Web_App
3. THE Makefile SHALL include a `test` target that runs `go test ./...` for all Go services and packages
4. WHEN `make lint` is executed, THE linter SHALL check all Go source files in `services/` and `pkg/` directories
5. WHEN `make test` is executed, THE test runner SHALL execute all Go tests in `services/` and `pkg/` directories and report results

---

### Requirement 14: Coding Standards and Conventions

**User Story:** As a developer, I want clear coding standards enforced across the project, so that all code is consistent and maintainable.

#### Acceptance Criteria

1. ALL Go source files SHALL have code comments written in **Bahasa Indonesia** (Indonesian language)
2. ALL Go variable names, function names, struct names, and interface names SHALL be written in **English**
3. ALL Go source files SHALL NOT exceed **200 lines** per file. If a file exceeds 200 lines, it SHALL be split into smaller files
4. ALL Go code SHALL follow clean architecture separation: domain layer SHALL NOT import from handler or repository layers
5. THE `.golangci.yml` SHALL include a custom linter rule or documentation note enforcing the 200-line file limit
6. THE root `README.md` SHALL document these coding conventions in a "Coding Standards" section

---

### Requirement 15: API Documentation with Swagger

**User Story:** As a developer, I want auto-generated API documentation, so that all endpoints are discoverable and testable from a browser.

#### Acceptance Criteria

1. THE Billing_API SHALL use `swaggo/swag` to auto-generate Swagger/OpenAPI documentation from Go code annotations
2. WHEN the Billing_API is running, THE Swagger UI SHALL be accessible at `GET /swagger/*`
3. THE Swagger documentation SHALL include API title, version, description, and base URL
4. THE Makefile SHALL include a `swagger` target that runs `swag init` to regenerate Swagger docs
5. THE Network_Service and Notification_Service SHALL also include Swagger documentation following the same pattern
