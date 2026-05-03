# Requirements Document

## Introduction

This spec defines the Reseller & Voucher Management module for ISPBoss — a SaaS billing platform for ISPs. The module implements reseller CRUD, reseller authentication (separate from admin auth), reseller balance management, voucher generation, voucher lifecycle management, reseller dashboard API, and voucher bulk actions. Resellers are stored in a dedicated `resellers` table with their own phone+password auth flow, separate from the `users` table. Vouchers are stored in a `vouchers` table with a full status lifecycle (Tersedia → Terjual → Aktif → Selesai/Expired/Void), price snapshots at purchase time, and an append-only audit trail. All data is tenant-scoped via RLS and RBAC-protected. The module publishes events to Redis for downstream services and uses asynq for async voucher generation and daily expiry cron jobs.

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles reseller, voucher, customer, billing, and auth operations
- **Reseller**: A third-party seller (shop/person) who purchases vouchers at a discounted price and resells them to end-users, stored in the `resellers` table
- **Voucher**: A single-use internet access code tied to a Voucher_Package, stored in the `vouchers` table
- **Voucher_Package**: A package with `type = 'voucher'` in the `packages` table, defining bandwidth, duration, sell price, and reseller price
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Reseller_Status**: One of `aktif`, `suspended`, or `nonaktif`
- **Voucher_Status**: One of `tersedia`, `terjual`, `aktif`, `selesai`, `expired`, or `void`
- **Sell_Price_Snapshot**: The end-user sell price captured at the moment a reseller purchases a voucher, immutable after purchase
- **Reseller_Price_Snapshot**: The reseller purchase price captured at the moment of purchase, immutable after purchase
- **Balance**: The reseller's monetary balance (BIGINT, in Rupiah) used to purchase vouchers, stored as `balance` on the `resellers` table
- **Daily_Purchase_Limit**: An optional per-reseller cap on the number of vouchers that can be purchased in a single calendar day (0 = unlimited)
- **Voucher_Code**: A unique alphanumeric code (6-16 characters) with optional prefix, used by end-users to activate internet access
- **Code_Format**: The character set for voucher code generation: `digits` (0-9), `letters` (A-Z), or `mixed` (A-Z + 0-9)
- **Voucher_Expiry_Days**: The number of days after reseller purchase before an unsold voucher expires and the balance is refunded (default 90, configurable per tenant)
- **Voucher_Audit_Log**: An append-only lifecycle log for vouchers, stored in the `voucher_audit_logs` table
- **Reseller_Session**: A login session for a reseller, stored in the existing `sessions` table with a reseller-specific auth flow
- **RLS**: PostgreSQL Row Level Security — a database-level safety net ensuring tenant data isolation
- **Audit_Log**: A record of changes to reseller data, stored in the shared `audit_logs` table with `entity_type = 'reseller'`
- **Event_Queue**: Redis-based message queue (asynq) for inter-service communication and background jobs
- **Validator**: The `go-playground/validator` library used for input validation in the Billing_API
- **Deposit**: A manual top-up of reseller balance performed by an admin (payment gateway integration is out of scope for this spec)
- **Async_Generate_Job**: An asynq background job for generating more than 500 vouchers in a single batch


## Requirements

### Requirement 1: Reseller Database Schema

**User Story:** As a tenant admin, I want a dedicated reseller database schema, so that reseller data (identity, credentials, balance, limits) is stored separately from admin users with proper multi-tenant isolation.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `resellers` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `name` (VARCHAR NOT NULL), `phone` (VARCHAR NOT NULL), `email` (VARCHAR), `address` (TEXT), `password_hash` (VARCHAR NOT NULL), `balance` (BIGINT NOT NULL DEFAULT 0), `daily_purchase_limit` (INTEGER NOT NULL DEFAULT 0), `status` (VARCHAR NOT NULL DEFAULT 'aktif'), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `resellers` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, phone)` to prevent duplicate phone numbers within a tenant
4. THE Billing_API migration SHALL create composite indexes on `(tenant_id, status)` and `(tenant_id, phone)` for query performance
5. THE Billing_API migration SHALL enforce a CHECK constraint on `status` to accept only `aktif`, `suspended`, or `nonaktif`
6. THE Billing_API migration SHALL enforce a CHECK constraint on `balance` to accept only values greater than or equal to 0
7. THE Billing_API migration SHALL enforce a CHECK constraint on `daily_purchase_limit` to accept only values greater than or equal to 0

### Requirement 2: Voucher Database Schema

**User Story:** As a tenant admin, I want a comprehensive voucher database schema, so that voucher codes, ownership, status lifecycle, and price snapshots are tracked with proper multi-tenant isolation.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `vouchers` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `code` (VARCHAR NOT NULL), `package_id` (UUID FK NOT NULL REFERENCES packages(id)), `reseller_id` (UUID FK REFERENCES resellers(id)), `status` (VARCHAR NOT NULL DEFAULT 'tersedia'), `sell_price_snapshot` (BIGINT), `reseller_price_snapshot` (BIGINT), `purchased_at` (TIMESTAMPTZ), `activated_at` (TIMESTAMPTZ), `expires_at` (TIMESTAMPTZ), `voided_at` (TIMESTAMPTZ), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `vouchers` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, code)` to ensure voucher codes are globally unique within a tenant
4. THE Billing_API migration SHALL create composite indexes on `(tenant_id, status)`, `(tenant_id, package_id)`, `(tenant_id, reseller_id)`, and `(tenant_id, status, expires_at)` for query performance
5. THE Billing_API migration SHALL enforce a CHECK constraint on `status` to accept only `tersedia`, `terjual`, `aktif`, `selesai`, `expired`, or `void`

### Requirement 3: Voucher Audit Log Schema

**User Story:** As a tenant admin, I want an append-only lifecycle log for every voucher, so that I can trace the full history of each voucher for financial reconciliation and troubleshooting.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `voucher_audit_logs` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `voucher_id` (UUID FK NOT NULL REFERENCES vouchers(id)), `action` (VARCHAR NOT NULL), `actor_id` (VARCHAR NOT NULL), `actor_name` (VARCHAR NOT NULL), `metadata` (JSONB), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `voucher_audit_logs` table with tenant isolation policies
3. THE Billing_API migration SHALL create a composite index on `(tenant_id, voucher_id)` for query performance
4. THE Billing_API SHALL treat the `voucher_audit_logs` table as append-only — no UPDATE or DELETE operations are permitted on this table


### Requirement 4: Reseller Create API

**User Story:** As a tenant admin, I want to create a new reseller with full validation, so that resellers can be onboarded with proper credentials and configuration.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/resellers` with valid data, THE Billing_API SHALL create a new reseller with status `aktif` and return the created reseller with HTTP 201
2. THE Validator SHALL require the following fields: `name` (min 3, max 255 characters), `phone` (Indonesian phone format, starting with `+62` or `08`, 10-15 digits), `password` (min 8 characters)
3. THE Validator SHALL accept the following optional fields: `email` (valid email format), `address` (max 1000 characters), `balance` (non-negative integer, default 0), `daily_purchase_limit` (non-negative integer, default 0 meaning unlimited)
4. IF the `phone` number already exists for the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PHONE_DUPLICATE`
5. THE Billing_API SHALL hash the password using bcrypt before storing it in the `password_hash` column
6. WHEN a reseller is successfully created, THE Billing_API SHALL write an audit log entry with action `reseller.created`

### Requirement 5: Reseller List API

**User Story:** As a tenant admin, I want to list resellers with pagination and filtering, so that I can manage and monitor all resellers.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/resellers`, THE Billing_API SHALL return a paginated list of resellers for the authenticated tenant
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `search` query parameter is provided, THE Billing_API SHALL filter resellers whose `name` or `phone` contains the search term (case-insensitive)
4. WHEN a `status` query parameter is provided (`aktif`, `suspended`, or `nonaktif`), THE Billing_API SHALL filter resellers by the specified status
5. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
6. THE Billing_API SHALL include computed fields `total_vouchers_sold` (count of vouchers with status not `tersedia` and not `void` owned by the reseller) for each reseller in the list response

### Requirement 6: Reseller Detail API

**User Story:** As a tenant admin, I want to view a reseller's full details including balance and sales summary, so that I can understand the reseller's current state and performance.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/resellers/:id`, THE Billing_API SHALL return the full reseller record including all fields
2. WHEN a GET request is made to `/v1/resellers/:id` with `include=audit_logs`, THE Billing_API SHALL include the reseller's audit log entries in the response
3. IF the reseller ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `RESELLER_NOT_FOUND`

### Requirement 7: Reseller Update API

**User Story:** As a tenant admin, I want to update reseller data, so that I can correct information or adjust configuration.

#### Acceptance Criteria

1. WHEN a PUT request is made to `/v1/resellers/:id` with valid data, THE Billing_API SHALL update the reseller record and return the updated reseller
2. THE Validator SHALL apply the same validation rules as reseller creation for all provided fields
3. IF the updated `phone` number already exists for another reseller in the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PHONE_DUPLICATE`
4. IF the reseller ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `RESELLER_NOT_FOUND`
5. WHEN a reseller is successfully updated, THE Billing_API SHALL write an audit log entry with action `reseller.updated` including the old and new values of changed fields


### Requirement 8: Reseller Status Management

**User Story:** As a tenant admin, I want to manage reseller status (suspend, activate, deactivate), so that I can control reseller access based on business needs.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/resellers/:id/suspend`, THE Billing_API SHALL transition the reseller's status from `aktif` to `suspended`
2. WHEN a POST request is made to `/v1/resellers/:id/activate`, THE Billing_API SHALL transition the reseller's status from `suspended` to `aktif`
3. WHEN a POST request is made to `/v1/resellers/:id/deactivate` with a `confirmation_name` field matching the reseller's name, THE Billing_API SHALL transition the reseller's status to `nonaktif` and void all vouchers with status `tersedia` owned by the reseller
4. IF the `confirmation_name` does not match the reseller's name (case-sensitive), THEN THE Billing_API SHALL return HTTP 400 with error code `CONFIRMATION_MISMATCH`
5. IF a status transition is requested that violates the state machine, THEN THE Billing_API SHALL return HTTP 422 with error code `INVALID_STATUS_TRANSITION`
6. THE Billing_API SHALL enforce the following valid transitions: `aktif` → [`suspended`, `nonaktif`], `suspended` → [`aktif`, `nonaktif`], `nonaktif` → [] (terminal state, no transitions out)
7. WHILE a reseller's status is `suspended`, THE Billing_API SHALL prevent the reseller from logging in and purchasing vouchers
8. WHEN any status transition occurs, THE Billing_API SHALL write an audit log entry with action `reseller.status_changed` including the old and new status

### Requirement 9: Reseller Password Reset

**User Story:** As a tenant admin, I want to reset a reseller's password, so that I can help resellers who have lost access to their account.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/resellers/:id/reset-password`, THE Billing_API SHALL generate a new random alphanumeric password of 8 characters, hash it with bcrypt, and update the reseller's `password_hash`
2. THE Billing_API SHALL return the new plaintext password in the response so the admin can communicate it to the reseller
3. THE Billing_API SHALL invalidate all existing sessions for the reseller after a password reset
4. IF the reseller ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `RESELLER_NOT_FOUND`
5. WHEN a password is reset, THE Billing_API SHALL write an audit log entry with action `reseller.password_reset`

### Requirement 10: Reseller Balance Top-Up (Deposit)

**User Story:** As a tenant admin, I want to manually top up a reseller's balance, so that the reseller can purchase vouchers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/resellers/:id/deposit` with a valid `amount` and optional `notes`, THE Billing_API SHALL atomically increase the reseller's `balance` by the specified amount and return the updated reseller
2. THE Validator SHALL require `amount` as a positive integer (minimum 1 Rupiah)
3. THE Validator SHALL accept an optional `notes` field (max 500 characters) for recording the deposit reason (e.g., "Transfer BCA 15:30")
4. IF the reseller ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `RESELLER_NOT_FOUND`
5. WHEN a deposit is successful, THE Billing_API SHALL write an audit log entry with action `reseller.deposit` including the amount, new balance, and notes
6. THE Billing_API SHALL record the deposit as a transaction in the `reseller_transactions` table with type `deposit`


### Requirement 11: Reseller Balance Withdraw (Admin Refund)

**User Story:** As a tenant admin, I want to manually withdraw (refund) a reseller's balance, so that I can return funds when a reseller is deactivated or when a manual correction is needed.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/resellers/:id/withdraw` with a valid `amount` and optional `notes`, THE Billing_API SHALL atomically decrease the reseller's `balance` by the specified amount and return the updated reseller
2. THE Validator SHALL require `amount` as a positive integer (minimum 1 Rupiah)
3. THE Validator SHALL accept an optional `notes` field (max 500 characters) for recording the withdrawal reason (e.g., "Refund saldo nonaktif — transfer BCA")
4. IF the reseller's balance is less than the requested `amount`, THEN THE Billing_API SHALL return HTTP 400 with error code `INSUFFICIENT_BALANCE` and include the current balance
5. IF the reseller ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `RESELLER_NOT_FOUND`
6. WHEN a withdrawal is successful, THE Billing_API SHALL write an audit log entry with action `reseller.withdraw` including the amount, new balance, and notes
7. THE Billing_API SHALL record the withdrawal as a transaction in the `reseller_transactions` table with type `withdraw`

### Requirement 12: Reseller Transaction Log Schema

**User Story:** As a tenant admin, I want all reseller financial transactions (deposits, purchases, refunds, withdrawals) recorded in a dedicated table, so that I can audit and reconcile reseller finances.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `reseller_transactions` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `reseller_id` (UUID FK NOT NULL REFERENCES resellers(id)), `type` (VARCHAR NOT NULL), `amount` (BIGINT NOT NULL), `balance_before` (BIGINT NOT NULL), `balance_after` (BIGINT NOT NULL), `reference_id` (UUID), `notes` (TEXT), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `reseller_transactions` table with tenant isolation policies
3. THE Billing_API migration SHALL enforce a CHECK constraint on `type` to accept only `deposit`, `purchase`, `refund`, or `withdraw`
4. THE Billing_API migration SHALL create composite indexes on `(tenant_id, reseller_id)` and `(tenant_id, reseller_id, created_at)` for query performance
5. FOR ALL reseller transactions, the `balance_after` SHALL equal `balance_before` plus `amount` for deposits and refunds, and `balance_before` minus `amount` for purchases and withdrawals (balance consistency property)

### Requirement 13: Reseller Authentication

**User Story:** As a reseller, I want to log in using my phone number and password, so that I can access my dashboard to buy and manage vouchers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/reseller/auth/login` with a valid `phone` and `password`, THE Billing_API SHALL authenticate the reseller, create a session in the `sessions` table, and return an access token (JWT) and refresh token
2. THE Billing_API SHALL include `reseller_id`, `tenant_id`, `name`, and `role` (set to `reseller`) in the JWT claims
3. IF the phone number does not match any reseller in the system, THEN THE Billing_API SHALL return HTTP 401 with error code `INVALID_CREDENTIALS`
4. IF the password does not match the stored hash, THEN THE Billing_API SHALL return HTTP 401 with error code `INVALID_CREDENTIALS`
5. IF the reseller's status is `suspended` or `nonaktif`, THEN THE Billing_API SHALL return HTTP 403 with error code `ACCOUNT_DISABLED` and a message indicating the account status
6. WHEN a reseller logs in successfully, THE Billing_API SHALL update a `last_login` timestamp on the reseller record

### Requirement 14: Reseller Login Rate Limiting

**User Story:** As a platform operator, I want reseller login attempts rate-limited, so that brute-force attacks on reseller accounts are prevented.

#### Acceptance Criteria

1. THE Billing_API SHALL track failed login attempts per reseller phone number using Redis
2. WHEN a reseller fails 5 consecutive login attempts, THE Billing_API SHALL lock the account for 15 minutes and return HTTP 429 with error code `ACCOUNT_LOCKED` and the remaining lock time in seconds
3. WHEN a reseller logs in successfully, THE Billing_API SHALL reset the failed attempt counter for that phone number
4. THE Billing_API SHALL use the existing `LoginRateLimiter` middleware infrastructure, adapted for phone-based identification instead of email-based

### Requirement 15: Reseller Session Management

**User Story:** As a reseller, I want my sessions managed with auto-logout, so that my account stays secure even if I forget to log out.

#### Acceptance Criteria

1. THE Billing_API SHALL set reseller session expiry to 24 hours from the time of login
2. WHEN a POST request is made to `/v1/reseller/auth/logout`, THE Billing_API SHALL invalidate the current session
3. WHEN a POST request is made to `/v1/reseller/auth/refresh` with a valid refresh token, THE Billing_API SHALL issue a new access token and refresh token pair
4. THE Billing_API SHALL store reseller sessions in the existing `sessions` table, using the reseller's UUID as the `user_id` field
5. WHEN a session expires (24 hours of inactivity), THE Billing_API SHALL require the reseller to log in again


### Requirement 16: Voucher Generate API

**User Story:** As a tenant admin, I want to generate batches of voucher codes for a specific package, so that vouchers are available for resellers to purchase.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/vouchers/generate` with valid data, THE Billing_API SHALL generate the requested number of unique voucher codes and insert them into the `vouchers` table with status `tersedia`
2. THE Validator SHALL require the following fields: `package_id` (must reference an existing voucher-type package), `quantity` (positive integer, minimum 1), `code_format` (one of `digits`, `letters`, or `mixed`), `code_length` (integer between 6 and 16 inclusive)
3. THE Validator SHALL accept an optional `prefix` field (max 10 characters, alphanumeric and hyphens only)
4. WHEN `quantity` is 500 or fewer, THE Billing_API SHALL generate vouchers synchronously and return the list of generated voucher codes with HTTP 201
5. WHEN `quantity` is greater than 500, THE Billing_API SHALL enqueue an Async_Generate_Job and return HTTP 202 with a `job_id`
6. THE Billing_API SHALL implement collision avoidance: for each generated code, check uniqueness within the tenant; if a collision occurs, retry up to 3 times with a new random code; if all retries fail, skip that code
7. THE Billing_API SHALL return a generation summary including `total_requested`, `total_generated`, and `total_failed` (codes that could not be generated due to persistent collisions)
8. WHEN vouchers are successfully generated, THE Billing_API SHALL write a voucher audit log entry with action `voucher.generated` for each voucher, with the admin as the actor
9. IF the `package_id` does not reference an existing package with `type = 'voucher'`, THEN THE Billing_API SHALL return HTTP 400 with error code `INVALID_PACKAGE_TYPE`

### Requirement 17: Voucher Code Generation Logic

**User Story:** As a developer, I want voucher code generation to be deterministic in format and collision-resistant, so that codes are unique, readable, and follow the configured format.

#### Acceptance Criteria

1. WHEN `code_format` is `digits`, THE Billing_API SHALL generate codes using only characters 0-9
2. WHEN `code_format` is `letters`, THE Billing_API SHALL generate codes using only uppercase characters A-Z
3. WHEN `code_format` is `mixed`, THE Billing_API SHALL generate codes using uppercase characters A-Z and digits 0-9
4. THE Billing_API SHALL generate codes of exactly the length specified by `code_length` (excluding the prefix)
5. WHEN a `prefix` is provided, THE Billing_API SHALL prepend the prefix to each generated code (e.g., prefix "ISP-" + code "AB12CD" = "ISP-AB12CD")
6. THE Billing_API SHALL use a cryptographically secure random number generator for code generation
7. FOR ALL generated voucher codes within a tenant, the full code (prefix + random part) SHALL be unique (uniqueness property enforced by database constraint)

### Requirement 18: Voucher List API

**User Story:** As a tenant admin, I want to list vouchers with filtering by package, status, and reseller, so that I can monitor and manage the voucher inventory.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/vouchers`, THE Billing_API SHALL return a paginated list of vouchers for the authenticated tenant
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `package_id` query parameter is provided, THE Billing_API SHALL filter vouchers by the specified package
4. WHEN a `status` query parameter is provided, THE Billing_API SHALL filter vouchers by the specified status
5. WHEN a `reseller_id` query parameter is provided, THE Billing_API SHALL filter vouchers by the specified reseller
6. WHEN a `search` query parameter is provided, THE Billing_API SHALL filter vouchers whose `code` contains the search term (case-insensitive)
7. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
8. THE Billing_API SHALL include the related `package_name` and `reseller_name` (resolved from their respective tables) in each voucher record in the response


### Requirement 19: Voucher Status Lifecycle

**User Story:** As a developer, I want the voucher status lifecycle enforced at the domain level, so that invalid transitions are impossible regardless of the caller.

#### Acceptance Criteria

1. THE Billing_API domain layer SHALL define the valid voucher status transitions as: `tersedia` → [`terjual`, `void`], `terjual` → [`aktif`, `expired`, `void`], `aktif` → [`selesai`], `selesai` → [] (terminal), `expired` → [] (terminal), `void` → [] (terminal)
2. WHEN an invalid voucher status transition is attempted, THE Billing_API SHALL return HTTP 422 with error code `INVALID_VOUCHER_STATUS_TRANSITION` and include the current status and the list of allowed target statuses
3. FOR ALL valid Voucher_Status values and FOR ALL valid transitions, applying a transition and then checking the resulting status SHALL yield the expected target status (state machine determinism property)
4. WHEN any voucher status transition occurs, THE Billing_API SHALL write a voucher audit log entry with the action describing the transition and the actor who triggered it

### Requirement 20: Voucher Bulk Actions API

**User Story:** As a tenant admin, I want to perform bulk actions on vouchers (print, void, assign to reseller, export CSV), so that I can efficiently manage large numbers of vouchers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/vouchers/bulk/print` with an array of voucher IDs, THE Billing_API SHALL generate a PDF document containing the selected vouchers formatted for printing (8-12 vouchers per A4 page) and return the PDF file
2. WHEN a POST request is made to `/v1/vouchers/bulk/void` with an array of voucher IDs, THE Billing_API SHALL transition each eligible voucher (status `tersedia` only) to `void` and return a summary of successes and failures
3. WHEN a POST request is made to `/v1/vouchers/bulk/assign` with an array of voucher IDs and a `reseller_id`, THE Billing_API SHALL assign each eligible voucher (status `tersedia` only) to the specified reseller without deducting balance (admin assignment) and return a summary of successes and failures
4. WHEN a GET request is made to `/v1/vouchers/export` with optional filter parameters, THE Billing_API SHALL generate a CSV file containing the filtered voucher list and return it for download
5. THE Billing_API SHALL return a response containing `total`, `success_count`, `failure_count`, and an array of `failures` with `voucher_id` and `reason` for each failed operation in bulk void and bulk assign actions
6. WHEN bulk void or bulk assign is performed, THE Billing_API SHALL write a voucher audit log entry for each individual voucher affected

### Requirement 21: Voucher PDF Print Format

**User Story:** As a tenant admin, I want printed vouchers to include tenant branding, voucher code, package info, price, and expiry date, so that resellers can sell professional-looking voucher cards to end-users.

#### Acceptance Criteria

1. THE Billing_API SHALL generate PDF voucher cards with the following information per voucher: tenant name (ISP branding), voucher code, package name, bandwidth (download/upload), duration, sell price, and expiry date (if purchased)
2. THE Billing_API SHALL layout 8 to 12 voucher cards per A4 page in a grid format suitable for cutting
3. THE Billing_API SHALL include the tenant's contact information (phone number) on each voucher card if configured
4. WHEN printing vouchers that have been purchased by a reseller, THE Billing_API SHALL display the sell price from the `sell_price_snapshot` (not the current package price)
5. WHEN printing vouchers that have not been purchased (status `tersedia`), THE Billing_API SHALL display the current sell price from the package

### Requirement 22: Voucher Expiry Background Job

**User Story:** As a platform operator, I want a daily background job that expires unsold vouchers past their expiry date and refunds the reseller's balance, so that resellers are not penalized for vouchers they could not sell.

#### Acceptance Criteria

1. THE Billing_API SHALL run a daily cron job (via asynq scheduler) that scans for vouchers with status `terjual` where `expires_at` is before the current time
2. WHEN an expired voucher is found, THE Billing_API SHALL atomically transition the voucher status to `expired` and refund the `reseller_price_snapshot` amount to the reseller's balance
3. THE Billing_API SHALL record the refund as a transaction in the `reseller_transactions` table with type `refund` and `reference_id` set to the voucher ID
4. WHEN a voucher is expired and refunded, THE Billing_API SHALL write a voucher audit log entry with action `voucher.expired` and actor set to `System`
5. THE Billing_API SHALL process expired vouchers in batches to avoid long-running transactions
6. FOR ALL expired vouchers, the reseller's balance SHALL increase by exactly the `reseller_price_snapshot` amount (refund integrity property)

### Requirement 23: Voucher Expiry Configuration

**User Story:** As a tenant admin, I want to configure the default voucher expiry period for my tenant, so that I can adjust the policy based on my business needs.

#### Acceptance Criteria

1. THE Billing_API SHALL store a `voucher_expiry_days` setting per tenant with a default value of 90 days
2. WHEN a reseller purchases a voucher, THE Billing_API SHALL set the voucher's `expires_at` to the purchase timestamp plus the tenant's configured `voucher_expiry_days`
3. WHEN a tenant admin updates the `voucher_expiry_days` setting, THE Billing_API SHALL apply the new value only to future purchases — existing vouchers retain their original `expires_at`


### Requirement 24: Reseller Dashboard Summary API

**User Story:** As a reseller, I want to see a summary of my balance, today's sales, and available vouchers on my dashboard, so that I can quickly understand my current business state.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/reseller/dashboard`, THE Billing_API SHALL return a summary containing: current `balance`, `sold_today` (count of vouchers purchased today), and `available_vouchers` (count of vouchers with status `terjual` owned by the reseller that have not been used or expired)
2. THE Billing_API SHALL scope the response to the authenticated reseller's data only
3. IF the reseller's status is not `aktif`, THEN THE Billing_API SHALL return HTTP 403 with error code `ACCOUNT_DISABLED`

### Requirement 25: Reseller Buy Voucher API

**User Story:** As a reseller, I want to buy vouchers by selecting a package and quantity, so that I can obtain voucher codes to resell to end-users.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/reseller/vouchers/buy` with a valid `package_id` and `quantity`, THE Billing_API SHALL atomically: verify the reseller is active, check the daily purchase limit, verify sufficient balance, deduct the total cost from the reseller's balance, generate voucher codes, snapshot the current sell price and reseller price, set `purchased_at` to the current time, set `expires_at` based on the tenant's `voucher_expiry_days`, transition voucher status to `terjual`, and assign the voucher to the reseller
2. THE Validator SHALL require `package_id` (must reference an existing active voucher-type package) and `quantity` (positive integer, minimum 1, maximum 100 per purchase)
3. IF the reseller's status is not `aktif`, THEN THE Billing_API SHALL return HTTP 403 with error code `ACCOUNT_DISABLED`
4. IF the reseller has a `daily_purchase_limit` greater than 0 and the total vouchers purchased today plus the requested quantity exceeds the limit, THEN THE Billing_API SHALL return HTTP 400 with error code `DAILY_LIMIT_EXCEEDED` and include the remaining daily allowance
5. IF the reseller's balance is less than the total cost (quantity multiplied by the package's reseller price), THEN THE Billing_API SHALL return HTTP 400 with error code `INSUFFICIENT_BALANCE` and include the current balance and the required amount
6. THE Billing_API SHALL record the purchase as a transaction in the `reseller_transactions` table with type `purchase`
7. WHEN vouchers are purchased, THE Billing_API SHALL write a voucher audit log entry with action `voucher.sold` for each voucher, with the reseller as the actor
8. THE Billing_API SHALL use a database transaction to ensure atomicity — if any step fails, the entire operation is rolled back (no partial deductions or partial voucher assignments)
9. FOR ALL voucher purchases, the reseller's balance after purchase SHALL equal the balance before purchase minus (quantity multiplied by reseller_price_snapshot) (balance deduction integrity property)

### Requirement 26: Reseller My Vouchers API

**User Story:** As a reseller, I want to list all vouchers I own with filtering by status, so that I can track my inventory and sales.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/reseller/vouchers`, THE Billing_API SHALL return a paginated list of vouchers owned by the authenticated reseller
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `status` query parameter is provided, THE Billing_API SHALL filter vouchers by the specified status
4. WHEN a `package_id` query parameter is provided, THE Billing_API SHALL filter vouchers by the specified package
5. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
6. THE Billing_API SHALL include the related `package_name` in each voucher record

### Requirement 27: Reseller Voucher Print API

**User Story:** As a reseller, I want to print my vouchers as a PDF, so that I can give physical voucher cards to end-users.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/reseller/vouchers/print` with an array of voucher IDs, THE Billing_API SHALL generate a PDF document containing the selected vouchers and return the PDF file
2. THE Billing_API SHALL verify that all requested voucher IDs belong to the authenticated reseller
3. IF any voucher ID does not belong to the authenticated reseller, THEN THE Billing_API SHALL return HTTP 403 with error code `FORBIDDEN`
4. THE Billing_API SHALL use the same PDF format as the admin voucher print (tenant branding, 8-12 per A4 page)

### Requirement 28: Reseller Deposit History API

**User Story:** As a reseller, I want to view my deposit history, so that I can track all balance top-ups.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/reseller/deposit`, THE Billing_API SHALL return a paginated list of transactions with type `deposit` for the authenticated reseller, sorted by `created_at` descending
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response

### Requirement 29: Reseller Transaction History API

**User Story:** As a reseller, I want to view my full transaction history (purchases, deposits, refunds), so that I can reconcile my finances.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/reseller/history`, THE Billing_API SHALL return a paginated list of all transactions for the authenticated reseller, sorted by `created_at` descending
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `type` query parameter is provided (`deposit`, `purchase`, or `refund`), THE Billing_API SHALL filter transactions by the specified type
4. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response


### Requirement 30: RBAC for Reseller and Voucher Endpoints

**User Story:** As a tenant admin, I want reseller and voucher endpoints protected by role-based access control, so that only authorized users can manage resellers and vouchers.

#### Acceptance Criteria

1. THE Billing_API SHALL allow `super_admin` and `tenant_admin` roles full access to all admin-facing reseller endpoints (`/v1/resellers/*`)
2. THE Billing_API SHALL allow `super_admin` and `tenant_admin` roles full access to all admin-facing voucher endpoints (`/v1/vouchers/*`)
3. THE Billing_API SHALL allow `operator` role read-only access (GET method only) to reseller list, reseller detail, voucher list, and voucher export endpoints
4. THE Billing_API SHALL deny access to `teknisi`, `kasir`, and `reseller` roles for all admin-facing reseller and voucher endpoints, returning HTTP 403 with error code `FORBIDDEN`
5. THE Billing_API SHALL allow only authenticated resellers (via reseller auth) to access reseller-facing endpoints (`/v1/reseller/*`)
6. THE Billing_API SHALL deny access to admin users (from the `users` table) for reseller-facing endpoints

### Requirement 31: Multi-Tenant Data Isolation for Resellers and Vouchers

**User Story:** As a platform operator, I want all reseller and voucher data strictly isolated per tenant, so that no tenant can access another tenant's data.

#### Acceptance Criteria

1. THE Billing_API SHALL set the PostgreSQL session variable `app.tenant_id` from the JWT claims before every database query on the `resellers`, `vouchers`, `voucher_audit_logs`, and `reseller_transactions` tables
2. THE Billing_API SHALL include `WHERE tenant_id = ?` in all reseller and voucher repository queries as the application-level filter
3. THE Billing_API RLS policies SHALL act as a safety net, blocking any query where `tenant_id` does not match the session variable
4. THE Billing_API SHALL prevent any reseller or voucher API endpoint from accepting a `tenant_id` in the request body — the tenant is always derived from the authenticated JWT token

### Requirement 32: Voucher Price Snapshot Integrity

**User Story:** As a developer, I want voucher price snapshots to be immutable after purchase, so that price changes to packages do not retroactively affect purchased vouchers.

#### Acceptance Criteria

1. WHEN a reseller purchases a voucher, THE Billing_API SHALL copy the current `sell_price` and `reseller_price` from the package into the voucher's `sell_price_snapshot` and `reseller_price_snapshot` columns
2. THE Billing_API SHALL NOT update `sell_price_snapshot` or `reseller_price_snapshot` after the initial purchase, regardless of subsequent package price changes
3. FOR ALL purchased vouchers (status not `tersedia`), the `sell_price_snapshot` and `reseller_price_snapshot` SHALL be non-null (snapshot completeness property)
4. FOR ALL purchased vouchers, the `reseller_price_snapshot` SHALL be less than the `sell_price_snapshot` (margin integrity property)

### Requirement 33: Reseller Status State Machine Integrity

**User Story:** As a developer, I want the reseller status state machine enforced at the domain level, so that invalid transitions are impossible regardless of the caller.

#### Acceptance Criteria

1. THE Billing_API domain layer SHALL define the valid reseller status transitions as: `aktif` → [`suspended`, `nonaktif`], `suspended` → [`aktif`, `nonaktif`], `nonaktif` → [] (terminal state, no transitions out)
2. WHEN an invalid reseller status transition is attempted, THE Billing_API SHALL return HTTP 422 with error code `INVALID_STATUS_TRANSITION` and include the current status and the list of allowed target statuses
3. FOR ALL valid Reseller_Status values and FOR ALL valid transitions, applying a transition and then checking the resulting status SHALL yield the expected target status (state machine determinism property)

### Requirement 34: Atomic Balance Operations

**User Story:** As a developer, I want all balance-modifying operations (deposit, withdraw, purchase, refund) to be atomic, so that concurrent operations cannot cause balance inconsistencies.

#### Acceptance Criteria

1. THE Billing_API SHALL use PostgreSQL row-level locking (`SELECT ... FOR UPDATE`) on the reseller row when modifying the balance
2. THE Billing_API SHALL wrap all balance-modifying operations (deposit, withdraw, purchase deduction, expiry refund) in a database transaction
3. THE Billing_API SHALL prevent the balance from going below zero — any operation that would result in a negative balance SHALL be rejected with an appropriate error
4. FOR ALL sequences of deposits, withdrawals, purchases, and refunds applied to a reseller, the final balance SHALL equal the initial balance plus the sum of all deposits and refunds minus the sum of all purchases and withdrawals (balance conservation property)

### Requirement 35: Event Publishing for Reseller and Voucher Operations

**User Story:** As a platform developer, I want reseller and voucher lifecycle events published to the event queue, so that downstream services can react to changes.

#### Acceptance Criteria

1. WHEN a reseller is created, THE Billing_API SHALL publish a `reseller.created` event to the Event_Queue containing `tenant_id`, `reseller_id`, and `name`
2. WHEN a reseller's status changes, THE Billing_API SHALL publish a `reseller.status_changed` event to the Event_Queue containing `tenant_id`, `reseller_id`, `old_status`, and `new_status`
3. WHEN vouchers are generated, THE Billing_API SHALL publish a `voucher.batch_generated` event to the Event_Queue containing `tenant_id`, `package_id`, `quantity`, and `generated_by`
4. WHEN a reseller purchases vouchers, THE Billing_API SHALL publish a `voucher.purchased` event to the Event_Queue containing `tenant_id`, `reseller_id`, `package_id`, `quantity`, and `total_cost`
5. THE Billing_API SHALL include `tenant_id`, `timestamp`, and `correlation_id` (UUID v4) in every published event envelope

### Requirement 36: Field Validation Rules for Reseller and Voucher

**User Story:** As a developer, I want all reseller and voucher input validated consistently, so that invalid data never enters the database.

#### Acceptance Criteria

1. THE Validator SHALL validate reseller `phone` as an Indonesian phone number starting with `+62` or `08`, with 10 to 15 digits total
2. THE Validator SHALL validate reseller `email` as a valid email format when provided (optional field)
3. THE Validator SHALL validate reseller `name` as a string with minimum 3 characters and maximum 255 characters
4. THE Validator SHALL validate reseller `password` as a string with minimum 8 characters
5. THE Validator SHALL validate voucher `code_length` as an integer between 6 and 16 inclusive
6. THE Validator SHALL validate voucher `code_format` as one of `digits`, `letters`, or `mixed`
7. THE Validator SHALL validate voucher `prefix` as alphanumeric characters and hyphens only, maximum 10 characters, when provided
8. THE Validator SHALL validate deposit `amount` as a positive integer (minimum 1)
9. THE Validator SHALL return all validation errors in a single response with HTTP 400, error code `VALIDATION_ERROR`, and an array of field-level error details