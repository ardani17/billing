# Tasks: MikroTik Bulk Actions

- [x] 1. Backend storage
  - [x] 1.1 Add `mikrotik_bulk_jobs` migration
  - [x] 1.2 Add domain DTOs and repository interface
  - [x] 1.3 Add SQL repository

- [x] 2. Backend usecase
  - [x] 2.1 Resolve selected/all active routers
  - [x] 2.2 Execute backup action
  - [x] 2.3 Execute firmware check action
  - [x] 2.4 Execute PPPoE sync action
  - [x] 2.5 Persist per-router result and final status

- [x] 3. API
  - [x] 3.1 Add bulk handler
  - [x] 3.2 Register routes
  - [x] 3.3 Wire dependencies in `cmd/main.go`

- [x] 4. Frontend
  - [x] 4.1 Add Next proxy routes
  - [x] 4.2 Add `/mikrotik/bulk` page
  - [x] 4.3 Add sidebar/nav entry
  - [x] 4.4 Show recent jobs and result summary

- [x] 5. Verification
  - [x] 5.1 Apply local migration
  - [x] 5.2 Run Go tests
  - [x] 5.3 Run web build
  - [x] 5.4 Smoke test against local CHR
