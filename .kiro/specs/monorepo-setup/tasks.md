# Implementation Plan: Monorepo Setup (ISPBoss)

## Overview

Setup fondasi monorepo ISPBoss dengan pendekatan bottom-up: mulai dari root structure dan shared packages, lalu scaffolding services dan frontend, kemudian database dan infrastructure, dan terakhir wiring semuanya bersama. Setiap task membangun di atas task sebelumnya sehingga tidak ada kode yang orphaned.

## Tasks

- [x] 1. Initialize monorepo root structure and configuration files
  - [x] 1.1 Create root directory structure and Turborepo configuration
    - Create top-level directories: `apps/`, `services/`, `pkg/`, `packages/`, `api-tests/`, `docker/`
    - Create `turbo.json` with pipeline tasks: `build`, `lint`, `dev`, `test`
    - Create root `package.json` with workspaces for `apps/*` and `packages/*`
    - Create `.gitignore` excluding build artifacts, `node_modules`, vendor, `.env`, IDE files
    - _Requirements: 1.1, 1.2, 1.3, 1.5_

  - [x] 1.2 Create Go workspace and root configuration files
    - Create `go.work` file defining Go workspaces for `services/*` and `pkg/*`
    - Create `.env.example` documenting all environment variables with descriptions and defaults
    - Create `.golangci.yml` with linters: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`
    - _Requirements: 1.4, 9.4, 13.1_

  - [x] 1.3 Create root Makefile with all targets
    - Implement targets: `dev`, `build`, `test`, `lint`, `migrate-up`, `migrate-down`, `docker-up`, `docker-down`
    - `lint` target runs `golangci-lint` for Go services/pkg and ESLint for web app
    - `test` target runs `go test ./...` for all Go services and packages
    - _Requirements: 1.6, 13.2, 13.3, 13.4, 13.5_

- [x] 2. Checkpoint — Verify root structure
  - Ensure all root config files are valid (turbo.json, package.json, go.work, Makefile). Ask the user if questions arise.

- [-] 3. Implement shared Go packages (pkg/)
  - [x] 3.1 Implement `pkg/logger` — zerolog logger factory
    - Create `pkg/logger/go.mod` and `pkg/logger/logger.go`
    - Implement `Config` struct with `Level`, `ServiceName`, `Pretty` fields
    - Implement `New(cfg Config) zerolog.Logger` with JSON output, timestamp, service name, configurable level
    - Implement `NewDefault(serviceName string) zerolog.Logger` with info level defaults
    - Support log levels: debug, info, warn, error, fatal
    - If `Pretty=true`, use `ConsoleWriter` for development
    - _Requirements: 7.1, 7.8_

  - [ ] 3.2 Write unit tests for `pkg/logger`
    - Test logger creation with various log levels
    - Test pretty mode vs JSON mode output
    - _Requirements: 7.1, 7.8_

  - [x] 3.3 Implement `pkg/auth` — JWT token generation and validation
    - Create `pkg/auth/go.mod` and `pkg/auth/auth.go`
    - Implement `Claims` struct embedding `jwt.RegisteredClaims` with `TenantID`, `UserID`, `Role`
    - Implement `TokenConfig` struct with `Secret`, `Expiry`, `Issuer`
    - Implement `GenerateToken(cfg TokenConfig, claims Claims) (string, error)`
    - Implement `ValidateToken(secret string, tokenString string) (*Claims, error)` returning decoded claims or error
    - _Requirements: 7.2, 7.3, 7.4_

  - [ ] 3.4 Write unit tests for `pkg/auth`
    - Test token generation and validation round-trip
    - Test expired token returns error
    - Test invalid signature returns error
    - Test claims contain tenant_id, user_id, role
    - _Requirements: 7.2, 7.3, 7.4_

  - [x] 3.5 Implement `pkg/tenant` — tenant context middleware and helpers
    - Create `pkg/tenant/go.mod` and `pkg/tenant/tenant.go`
    - Implement `Middleware(jwtSecret string) fiber.Handler` that extracts tenant_id from JWT and injects into context
    - Implement `FromContext(ctx context.Context) string` to retrieve tenant_id
    - Implement `MustFromContext(ctx context.Context) string` that panics if not found
    - Return HTTP 401 if JWT is invalid or tenant_id is missing
    - _Requirements: 7.5, 7.6, 7.7_

  - [ ] 3.6 Write unit tests for `pkg/tenant`
    - Test middleware extracts tenant_id from valid JWT
    - Test middleware returns 401 for missing/invalid JWT
    - Test FromContext returns empty string when not set
    - Test MustFromContext panics when not set
    - _Requirements: 7.5, 7.6, 7.7_

  - [x] 3.7 Implement `pkg/database` — pgx connection pool and tenant helpers
    - Create `pkg/database/go.mod` and `pkg/database/database.go`
    - Implement `PoolConfig` struct with DSN, MaxConns, MinConns, timeouts
    - Implement `NewPool(ctx, cfg) (*pgxpool.Pool, error)` for connection pool creation
    - Implement `SetTenantID(ctx, pool, tenantID) error` to set `app.tenant_id` session variable
    - Implement `WithTenant(ctx, pool, tenantID, fn) error` to run function in tenant context
    - _Requirements: 4.1, 4.2_

  - [ ] 3.8 Write unit tests for `pkg/database`
    - Test PoolConfig DSN construction
    - Test SetTenantID and WithTenant logic (mock or integration)
    - _Requirements: 4.1, 4.2_

  - [x] 3.9 Implement `pkg/queue` — asynq client/server factory and task envelope
    - Create `pkg/queue/go.mod` and `pkg/queue/queue.go`
    - Implement `TaskEnvelope` struct with `EventType`, `TenantID`, `Timestamp`, `CorrelationID`, `Payload`
    - Implement `ClientConfig` struct with Redis connection parameters
    - Implement `NewClient(cfg) (*asynq.Client, error)` factory
    - Implement `NewServer(cfg, concurrency, queues) (*asynq.Server, error)` factory
    - Implement `EnqueueTask(client, envelope) error` with JSON serialization
    - Implement `DecodeEnvelope(task) (*TaskEnvelope, error)` with JSON deserialization
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

  - [ ] 3.10 Write unit tests for `pkg/queue`
    - Test EnqueueTask serializes envelope to JSON
    - Test DecodeEnvelope deserializes JSON back to envelope
    - Test round-trip serialization/deserialization
    - _Requirements: 5.3, 5.4, 5.5_

- [x] 4. Checkpoint — Verify shared Go packages compile
  - Run `go build ./...` for all packages in `pkg/`. Ensure all tests pass. Ask the user if questions arise.

- [-] 5. Scaffold Go services with clean architecture
  - [x] 5.1 Scaffold `services/billing-api` with clean architecture structure
    - Create directory structure: `cmd/`, `internal/config/`, `internal/domain/`, `internal/usecase/`, `internal/repository/`, `internal/handler/`, `internal/middleware/`, `migrations/`
    - Create `go.mod` with module path and dependencies on `pkg/*`
    - Create `internal/config/config.go` with `AppConfig` struct, `Load()` using Viper (env vars + .env fallback), `Validate()` that exits on missing required vars, `DSN()` helper
    - Create `internal/domain/tenant.go` and `internal/domain/customer.go` entity structs
    - Create `sqlc.yaml` configuration file
    - _Requirements: 3.1, 3.7, 9.1, 9.2, 9.5_

  - [x] 5.2 Implement billing-api health check handler and router
    - Create `internal/handler/health.go` with `HealthHandler` struct
    - Implement `Healthz(c *fiber.Ctx) error` returning 200 with status, service name, timestamp
    - Implement `Readyz(c *fiber.Ctx) error` checking DB and Redis connectivity, returning 200 or 503
    - Create `internal/handler/router.go` for route registration
    - _Requirements: 3.6, 11.1, 11.2, 11.4, 11.5_

  - [x] 5.3 Implement billing-api middleware stack
    - Create `internal/middleware/auth.go` — JWT auth middleware using `pkg/auth`
    - Create `internal/middleware/tenant.go` — tenant context middleware using `pkg/tenant`
    - Create `internal/middleware/logging.go` — request logging middleware using `pkg/logger` (log method, path, status, duration)
    - _Requirements: 3.10, 10.1, 10.2, 10.3_

  - [x] 5.4 Implement billing-api `cmd/main.go` entry point
    - Load config, initialize logger, create DB pool, create Redis client
    - Set up Fiber app with middleware stack (logging, recovery)
    - Register health check routes (public) and protected routes (auth + tenant middleware)
    - Start HTTP server on configured port (default 3001)
    - _Requirements: 3.4, 3.5, 3.7_

  - [ ] 5.5 Write unit tests for billing-api health handler
    - Test `/healthz` returns 200 with correct JSON structure
    - Test `/readyz` returns 200 when dependencies are healthy
    - Test `/readyz` returns 503 when database is unreachable
    - Test `/readyz` returns 503 when Redis is unreachable
    - _Requirements: 11.1, 11.2, 11.4, 11.5_

  - [x] 5.6 Scaffold `services/network-service` with clean architecture structure
    - Clone billing-api structure to `services/network-service/`
    - Create `go.mod`, config (add `NETWORK_MODE` with default `mock`), domain, handler, middleware, `cmd/main.go`
    - Register health check endpoints, set default port 3002
    - _Requirements: 3.2, 3.8, 9.3_

  - [x] 5.7 Scaffold `services/notification` with clean architecture structure
    - Clone billing-api structure to `services/notification/`
    - Create `go.mod`, config, domain, handler, middleware, `cmd/main.go`
    - Register health check endpoints, set default port 3003
    - _Requirements: 3.3, 3.9_

- [x] 6. Checkpoint — Verify Go services compile and health endpoints work
  - Run `go build ./...` for all services. Ensure all tests pass. Ask the user if questions arise.

- [x] 7. Create PostgreSQL migrations and RLS setup
  - [x] 7.1 Create initial migration files for billing-api
    - Create `services/billing-api/migrations/000001_init_tenants.up.sql`:
      - `tenants` table with columns: `id` (UUID PK), `name`, `domain`, `plan`, `status`, `created_at`, `updated_at`
      - Indexes on `domain` and `status`
    - Create `services/billing-api/migrations/000001_init_tenants.down.sql` with rollback
    - _Requirements: 4.5_

  - [x] 7.2 Create customers table migration with RLS
    - Create `services/billing-api/migrations/000002_init_customers.up.sql`:
      - `customers` table with `tenant_id` (UUID, NOT NULL, FK to tenants), `name`, `email`, `phone`, `status`, timestamps
      - Enable RLS on `customers` table
      - Create `tenant_isolation` policy using `current_setting('app.tenant_id')::uuid`
      - Create `tenant_insert` policy for INSERT with CHECK
      - Create indexes on `tenant_id` and `(tenant_id, status)`
    - Create `services/billing-api/migrations/000002_init_customers.down.sql` with rollback
    - _Requirements: 4.6, 4.7, 4.8, 4.9, 10.4, 10.5_

  - [x] 7.3 Create sqlc configuration and sample queries
    - Configure `services/billing-api/sqlc.yaml` pointing to migrations and query files
    - Create sample SQL queries file for tenants and customers
    - Generate Go code with `sqlc generate`
    - _Requirements: 4.10_

- [x] 8. Checkpoint — Verify migrations are valid SQL
  - Review migration files for correctness. Ensure sqlc generates without errors. Ask the user if questions arise.

- [x] 9. Scaffold Next.js frontend and shared TypeScript packages
  - [x] 9.1 Create shared TypeScript packages
    - Create `packages/config/package.json`, `eslint.config.mjs` (shared ESLint for Next.js), `tsconfig.base.json` (strict mode), `prettier.config.mjs`
    - Create `packages/types/package.json` and `packages/types/src/index.ts` with `ApiResponse<T>` interface
    - Create `packages/ui/package.json` initialized as shadcn/ui component package
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

  - [x] 9.2 Scaffold Next.js web application at `apps/web/`
    - Initialize Next.js 15 with App Router and TypeScript
    - Configure Tailwind CSS v4 for styling
    - Configure Geist and Geist Mono fonts
    - Set up ESLint and Prettier extending from `packages/config/`
    - Set up `tsconfig.json` extending from `packages/config/tsconfig.base.json`
    - Configure `NEXT_PUBLIC_API_URL` environment variable (default: `http://localhost:3001`)
    - Ensure dev server runs on port 3000
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_

  - [x] 9.3 Integrate shadcn/ui into the web application
    - Initialize shadcn/ui in `apps/web/` with the project's Tailwind config
    - Configure component imports from `packages/ui/` package
    - _Requirements: 2.3, 8.5_

- [x] 10. Checkpoint — Verify frontend builds and TS packages compile
  - Run `turbo run build` to verify all TypeScript packages and the web app build successfully. Ask the user if questions arise.

- [x] 11. Create Docker Compose development environment
  - [x] 11.1 Create Docker Compose configuration
    - Create `docker/docker-compose.yml` with services: PostgreSQL 16 (port 5432), Redis 7 (port 6379), billing-api (port 3001), network-service (port 3002), notification (port 3003)
    - Configure named volumes for PostgreSQL and Redis data persistence
    - Set dependency ordering: Go services depend on PostgreSQL and Redis with `condition: service_healthy`
    - Add health checks for PostgreSQL (`pg_isready`) and Redis (`redis-cli ping`) with 10s interval, 5s timeout
    - Set `NETWORK_MODE=mock` for network-service
    - Create `docker/.env.example` with all required environment variables and sensible defaults
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_

  - [x] 11.2 Create multi-stage Dockerfiles for each Go service
    - Create `services/billing-api/Dockerfile` with builder stage (golang:1.24-alpine) and runtime stage (alpine:latest)
    - Copy `go.work`, `go.work.sum`, `pkg/`, and service directory in builder stage
    - Copy compiled binary and migrations to runtime stage
    - Create `services/network-service/Dockerfile` and `services/notification/Dockerfile` following the same pattern
    - _Requirements: 6.8_

  - [x] 11.3 Add Docker health checks for Go services
    - Configure health checks in `docker-compose.yml` using `/healthz` endpoint for each Go service
    - Set interval 10s, timeout 5s
    - _Requirements: 11.3_

- [x] 12. Create API test infrastructure with Bruno
  - [x] 12.1 Set up Bruno collection and environment
    - Create `api-tests/bruno.json` collection configuration
    - Create `api-tests/environments/local.bru` with base URL variables for all services (billing-api: localhost:3001, network-service: localhost:3002, notification: localhost:3003)
    - Create sample health check requests: `api-tests/billing-api/healthz.bru`, `api-tests/network-service/healthz.bru`, `api-tests/notification/healthz.bru`
    - _Requirements: 12.1, 12.2, 12.3_

- [x] 13. Final wiring and integration verification
  - [x] 13.1 Update go.work with all module paths
    - Ensure `go.work` includes all service modules and pkg modules
    - Run `go work sync` to generate `go.work.sum`
    - Verify all cross-module imports resolve correctly
    - _Requirements: 1.4, 1.7, 1.8_

  - [x] 13.2 Add Swagger documentation to Go services
    - Add `swaggo/swag` dependency to each Go service
    - Add Swagger annotations to `cmd/main.go` (title, version, description, base URL)
    - Add Swagger annotations to health check handlers (`@Summary`, `@Tags`, `@Success`, `@Router`)
    - Register Swagger route `GET /swagger/*` in each service router
    - Add `swagger` target to Makefile that runs `swag init` for all services
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5_

  - [x] 13.3 Add coding standards documentation and enforcement
    - Create "Coding Standards" section in root `README.md` documenting:
      - Code comments in Bahasa Indonesia
      - Variable/function names in English
      - Max 200 lines per file
      - Clean architecture layer rules
    - Add note in `.golangci.yml` about 200-line file limit convention
    - Verify all existing Go files follow coding standards (comments in Indonesian)
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6_

  - [x] 13.4 Verify Makefile targets work end-to-end
    - Test `make lint` runs golangci-lint on `services/` and `pkg/` and ESLint on `apps/web/`
    - Test `make test` runs Go tests across all services and packages
    - Test `make swagger` generates Swagger docs for all services
    - Test `make docker-up` and `make docker-down` control Docker Compose stack
    - Test `make migrate-up` and `make migrate-down` run migrations
    - _Requirements: 1.6, 13.2, 13.3, 13.4, 13.5, 15.4_

- [x] 14. Final checkpoint — Full build and integration
  - Run `turbo run build` and `go build ./...` for all modules. Ensure all tests pass and Docker Compose stack starts correctly. Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation at each major milestone
- Go services share the same clean architecture pattern — billing-api is implemented first as the template, then network-service and notification follow the same structure
- All shared Go packages are implemented before services to avoid circular dependency issues
