# Requirements Document

## Introduction

This spec defines the Package Management module for ISPBoss — a SaaS billing platform for ISPs. The module implements CRUD operations for internet packages, covering both PPPoE/Static packages (monthly billing for fixed customers) and Hotspot/Voucher packages (per-voucher billing for end-users via resellers). Both package types are stored in a single `packages` table with a `type` discriminator column. All data is tenant-scoped via RLS and RBAC-protected. The module publishes events to Redis for downstream services (Notification) when package prices change.

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles package, customer, billing, and auth operations
- **Package**: An internet service plan offered by a tenant, stored in the `packages` table
- **PPPoE_Package**: A package type for fixed customers (home/office) billed monthly (e.g., "Pro 50M — Rp 350,000/month")
- **Voucher_Package**: A package type for end-users via resellers, billed per voucher with a fixed duration (e.g., "1 Day 5M — Rp 3,000")
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Package_Type**: The discriminator column indicating whether a package is `pppoe` or `voucher`
- **Bandwidth_Type**: Whether bandwidth is `dedicated` (guaranteed) or `shared` (up-to/best-effort)
- **Quota_Type_PPPoE**: One of `unlimited`, `monthly_quota`, or `fup` (Fair Usage Policy) for PPPoE packages
- **Quota_Type_Voucher**: One of `unlimited` or `quota` for Voucher packages
- **Quota_Action**: The action taken when quota is exhausted: `throttle` (reduce speed) or `disconnect`
- **Duration_Unit**: The time unit for voucher package duration: `hours`, `days`, `weeks`, or `months`
- **Burst**: MikroTik bandwidth burst feature allowing temporary speed increases above the configured rate
- **Reseller_Price**: The price at which a reseller purchases a voucher, which must be lower than the sell price with a minimum margin of Rp 500
- **Sell_Price**: The end-user price for a voucher package
- **Installation_Fee**: A one-time fee charged when activating a PPPoE customer, stored as a default on the package but customizable per customer at activation
- **Quota_MB**: The quota allowance stored in megabytes (MB). The frontend may display in GB (1 GB = 1024 MB) but the backend always stores in MB for precision with smaller voucher quotas
- **Customer_Count**: A computed field (COUNT from customers table) showing how many active customers use a given package — not stored on the package record
- **RLS**: PostgreSQL Row Level Security — a database-level safety net ensuring tenant data isolation
- **Audit_Log**: A record of all changes to package data, stored in the `audit_logs` table (shared with customer module)
- **Event_Queue**: Redis-based message queue (asynq) for inter-service communication
- **Validator**: The `go-playground/validator` library used for input validation in the Billing_API
- **MikroTik_Profile**: A MikroTik router configuration profile name, auto-generated from the package name. The backend always accepts and stores this field, but the frontend hides it when the MikroTik module is not active

## Requirements

### Requirement 1: Package Database Schema

**User Story:** As a tenant admin, I want a comprehensive package database schema, so that both PPPoE/Static and Hotspot/Voucher packages are stored in a single well-indexed table with proper multi-tenant isolation.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `packages` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `type` (VARCHAR NOT NULL), `name` (VARCHAR NOT NULL), `description` (TEXT), `is_active` (BOOLEAN NOT NULL DEFAULT true), `download_mbps` (INTEGER NOT NULL), `upload_mbps` (INTEGER NOT NULL), `bandwidth_type` (VARCHAR), `burst_download_mbps` (INTEGER), `burst_upload_mbps` (INTEGER), `burst_threshold_mbps` (INTEGER), `burst_time_seconds` (INTEGER), `quota_type` (VARCHAR NOT NULL), `quota_mb` (INTEGER), `quota_action` (VARCHAR), `throttle_mbps` (INTEGER), `monthly_price` (BIGINT), `installation_fee` (BIGINT NOT NULL DEFAULT 0), `sell_price` (BIGINT), `reseller_price` (BIGINT), `duration_value` (INTEGER), `duration_unit` (VARCHAR), `shared_users` (INTEGER NOT NULL DEFAULT 1), `mikrotik_profile_name` (VARCHAR), `address_pool` (VARCHAR), `parent_queue` (VARCHAR), `hotspot_profile_name` (VARCHAR), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `packages` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL create composite indexes on `(tenant_id, type)`, `(tenant_id, is_active)`, and `(tenant_id, type, is_active)` for query performance
4. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, name)` to prevent duplicate package names within a tenant
5. THE Billing_API migration SHALL enforce a CHECK constraint on `type` to accept only `pppoe` or `voucher`
6. THE Billing_API migration SHALL enforce a CHECK constraint on `quota_type` to accept only `unlimited`, `monthly_quota`, `fup`, or `quota`
7. THE Billing_API migration SHALL enforce a CHECK constraint on `download_mbps` and `upload_mbps` to accept only values greater than 0
8. THE Billing_API migration SHALL add a foreign key from the existing `customers.package_id` column to `packages.id` to establish the relationship between customers and packages

### Requirement 2: Package Create API (PPPoE)

**User Story:** As a tenant admin, I want to create a PPPoE/Static package with full validation, so that I can define internet plans for fixed customers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/packages` with `type` set to `pppoe` and valid data, THE Billing_API SHALL create a new package with `is_active` set to true and return the created package with HTTP 201
2. THE Validator SHALL require the following fields for PPPoE packages: `name` (min 2, max 255 characters), `type` (must be `pppoe`), `download_mbps` (positive integer), `upload_mbps` (positive integer), `bandwidth_type` (one of `dedicated` or `shared`), `quota_type` (one of `unlimited`, `monthly_quota`, or `fup`), `monthly_price` (positive integer, in Rupiah)
3. WHEN `quota_type` is `monthly_quota` or `fup`, THE Validator SHALL require `quota_mb` (positive integer, in megabytes) and `quota_action` (one of `throttle` or `disconnect`)
4. WHEN `quota_action` is `throttle`, THE Validator SHALL require `throttle_mbps` (positive integer)
5. WHEN burst fields are provided, THE Validator SHALL require all four burst fields together: `burst_download_mbps`, `burst_upload_mbps`, `burst_threshold_mbps` (all positive integers), and `burst_time_seconds` (positive integer)
6. IF the `name` already exists for the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PACKAGE_NAME_DUPLICATE`
7. WHEN a PPPoE package is successfully created, THE Billing_API SHALL write an audit log entry with action `package.created`
8. THE Billing_API SHALL auto-generate `mikrotik_profile_name` from the package name (lowercase, spaces replaced with hyphens) when not explicitly provided

### Requirement 3: Package Create API (Voucher)

**User Story:** As a tenant admin, I want to create a Hotspot/Voucher package with full validation, so that I can define internet plans for end-users purchased through resellers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/packages` with `type` set to `voucher` and valid data, THE Billing_API SHALL create a new package with `is_active` set to true and return the created package with HTTP 201
2. THE Validator SHALL require the following fields for Voucher packages: `name` (min 2, max 255 characters), `type` (must be `voucher`), `download_mbps` (positive integer), `upload_mbps` (positive integer), `quota_type` (one of `unlimited` or `quota`), `sell_price` (positive integer, in Rupiah), `reseller_price` (positive integer, in Rupiah), `duration_value` (positive integer), `duration_unit` (one of `hours`, `days`, `weeks`, or `months`)
3. WHEN `quota_type` is `quota`, THE Validator SHALL require `quota_mb` (positive integer, in megabytes)
4. THE Validator SHALL enforce that `reseller_price` is less than `sell_price` and that the margin (`sell_price` minus `reseller_price`) is at least 500 Rupiah
5. IF the `name` already exists for the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PACKAGE_NAME_DUPLICATE`
6. WHEN a Voucher package is successfully created, THE Billing_API SHALL write an audit log entry with action `package.created`
7. THE Billing_API SHALL auto-generate `hotspot_profile_name` from the package name (lowercase, spaces replaced with hyphens) when not explicitly provided

### Requirement 4: Package Update API

**User Story:** As a tenant admin, I want to update package details, so that I can adjust pricing, bandwidth, or other configuration as business needs change.

#### Acceptance Criteria

1. WHEN a PUT request is made to `/v1/packages/:id` with valid data, THE Billing_API SHALL update the package record and return the updated package
2. THE Validator SHALL apply the same validation rules as package creation for all provided fields, based on the package `type`
3. IF the updated `name` already exists for another package in the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PACKAGE_NAME_DUPLICATE`
4. IF the package ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `PACKAGE_NOT_FOUND`
5. WHEN a package is successfully updated, THE Billing_API SHALL write an audit log entry with action `package.updated` including the old and new values of changed fields
6. WHEN the `monthly_price` of a PPPoE package or the `sell_price` of a Voucher package is changed, THE Billing_API SHALL publish a `package.price_changed` event to the Event_Queue containing `package_id`, `package_name`, `old_price`, `new_price`, and `package_type`

### Requirement 5: Package List API

**User Story:** As an operator, I want to list packages with filtering, search, and pagination, so that I can quickly find and manage packages.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/packages`, THE Billing_API SHALL return a paginated list of packages for the authenticated tenant
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `type` query parameter is provided (`pppoe` or `voucher`), THE Billing_API SHALL filter packages by the specified type
4. WHEN a `search` query parameter is provided, THE Billing_API SHALL filter packages whose `name` contains the search term (case-insensitive)
5. WHEN an `is_active` query parameter is provided (`true` or `false`), THE Billing_API SHALL filter packages by their active status
6. WHEN a `sort_by` query parameter is provided, THE Billing_API SHALL sort results by the specified column (`name`, `monthly_price`, `sell_price`, `download_mbps`, `created_at`) in the direction specified by `sort_order` (asc or desc, default asc)
7. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
8. THE Billing_API SHALL include the `customer_count` (computed via COUNT from the customers table where `deleted_at IS NULL`) for each package in the list response

### Requirement 6: Package Detail API

**User Story:** As an operator, I want to view a package's full details including customer count, so that I can understand the package's current usage and configuration.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/packages/:id`, THE Billing_API SHALL return the full package record including all fields and the computed `customer_count`
2. WHEN a GET request is made to `/v1/packages/:id` with `include=audit_logs`, THE Billing_API SHALL include the package's audit log entries in the response
3. IF the package ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `PACKAGE_NOT_FOUND`

### Requirement 7: Package Deactivate and Activate API

**User Story:** As a tenant admin, I want to deactivate a package so it is hidden from new customer registration, while existing customers continue using it, and reactivate it when needed.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/packages/:id/deactivate`, THE Billing_API SHALL set the package's `is_active` to false and return the updated package
2. WHEN a POST request is made to `/v1/packages/:id/activate`, THE Billing_API SHALL set the package's `is_active` to true and return the updated package
3. IF the package ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `PACKAGE_NOT_FOUND`
4. IF the package is already in the requested state (e.g., deactivating an already inactive package), THEN THE Billing_API SHALL return HTTP 400 with error code `PACKAGE_ALREADY_INACTIVE` or `PACKAGE_ALREADY_ACTIVE`
5. WHEN a package is deactivated or activated, THE Billing_API SHALL write an audit log entry with action `package.deactivated` or `package.activated`

### Requirement 8: Package Delete API

**User Story:** As a tenant admin, I want to permanently delete a package that has zero customers, so that I can clean up unused packages from the system.

#### Acceptance Criteria

1. WHEN a DELETE request is made to `/v1/packages/:id` with a `confirmation_name` field matching the package's name, THE Billing_API SHALL permanently delete the package (hard delete) and return HTTP 200
2. IF the `confirmation_name` does not match the package's name (case-sensitive), THEN THE Billing_API SHALL return HTTP 400 with error code `CONFIRMATION_MISMATCH`
3. IF the package has one or more customers using it (customer_count > 0, excluding soft-deleted customers), THEN THE Billing_API SHALL return HTTP 409 with error code `PACKAGE_HAS_CUSTOMERS` and include the customer count and a suggestion to deactivate instead
4. IF the package ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `PACKAGE_NOT_FOUND`
5. WHEN a package is successfully deleted, THE Billing_API SHALL write an audit log entry with action `package.deleted`

### Requirement 9: Package Duplicate API

**User Story:** As a tenant admin, I want to duplicate an existing package, so that I can quickly create a new package based on an existing one without re-entering all fields.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/packages/:id/duplicate`, THE Billing_API SHALL create a new package with all fields copied from the source package, except `id` (new UUID), `name` (appended with " (Copy)"), `is_active` (set to true), and `created_at`/`updated_at` (set to current time)
2. IF the generated name (original name + " (Copy)") already exists for the tenant, THE Billing_API SHALL append a numeric suffix (e.g., " (Copy 2)", " (Copy 3)") until a unique name is found
3. IF the source package ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `PACKAGE_NOT_FOUND`
4. WHEN a package is successfully duplicated, THE Billing_API SHALL write an audit log entry with action `package.duplicated` including the source package ID

### Requirement 10: RBAC for Package Endpoints

**User Story:** As a tenant admin, I want package endpoints protected by role-based access control, so that only authorized users can view or modify package data.

#### Acceptance Criteria

1. THE Billing_API SHALL allow `super_admin`, `tenant_admin`, and `operator` roles read access (GET method) to package list and detail endpoints
2. THE Billing_API SHALL allow `kasir` role read-only access (GET method only) to package list and detail endpoints
3. THE Billing_API SHALL allow only `tenant_admin` (and `super_admin`) roles to create, update, delete, activate, deactivate, and duplicate packages
4. THE Billing_API SHALL deny access to `teknisi` and `reseller` roles for all package endpoints, returning HTTP 403 with error code `FORBIDDEN`

### Requirement 11: Multi-Tenant Data Isolation for Packages

**User Story:** As a platform operator, I want all package data strictly isolated per tenant, so that no tenant can access another tenant's packages.

#### Acceptance Criteria

1. THE Billing_API SHALL set the PostgreSQL session variable `app.tenant_id` from the JWT claims before every database query on the `packages` table
2. THE Billing_API SHALL include `WHERE tenant_id = ?` in all package repository queries as the application-level filter
3. THE Billing_API RLS policies SHALL act as a safety net, blocking any query where `tenant_id` does not match the session variable
4. THE Billing_API SHALL prevent any package API endpoint from accepting a `tenant_id` in the request body — the tenant is always derived from the authenticated JWT token

### Requirement 12: Audit Trail for Package Operations

**User Story:** As a tenant admin, I want a complete audit trail of all package changes, so that I can track who changed what and when for compliance and troubleshooting.

#### Acceptance Criteria

1. WHEN a package is created, updated, deleted, activated, deactivated, or duplicated, THE Billing_API SHALL insert a record into the `audit_logs` table with `entity_type` set to `package`, the `entity_id` set to the package's UUID, the `action` describing the operation, and the `actor_id` and `actor_name` from the authenticated user
2. WHEN a package is updated, THE Billing_API SHALL store the old and new values of changed fields in the `changes` JSONB column
3. THE Billing_API SHALL store audit log entries for the following actions: `package.created`, `package.updated`, `package.deleted`, `package.activated`, `package.deactivated`, `package.duplicated`

### Requirement 13: Event Publishing for Package Price Changes

**User Story:** As a platform developer, I want package price change events published to the event queue, so that downstream services (Notification) can notify affected customers.

#### Acceptance Criteria

1. WHEN the `monthly_price` of a PPPoE package is changed via update, THE Billing_API SHALL publish a `package.price_changed` event to the Event_Queue containing `tenant_id`, `package_id`, `package_name`, `package_type`, `old_price`, and `new_price`
2. WHEN the `sell_price` of a Voucher package is changed via update, THE Billing_API SHALL publish a `package.price_changed` event to the Event_Queue containing `tenant_id`, `package_id`, `package_name`, `package_type`, `old_price`, and `new_price`
3. THE Billing_API SHALL include `tenant_id`, `timestamp`, and `correlation_id` (UUID v4) in every published event envelope

### Requirement 14: Field Validation Rules

**User Story:** As a developer, I want all package input validated consistently, so that invalid data never enters the database.

#### Acceptance Criteria

1. THE Validator SHALL validate `name` as a string with minimum 2 characters and maximum 255 characters
2. THE Validator SHALL validate `type` as one of `pppoe` or `voucher`
3. THE Validator SHALL validate `download_mbps` and `upload_mbps` as positive integers
4. THE Validator SHALL validate `bandwidth_type` as one of `dedicated` or `shared` when provided (required for PPPoE)
5. THE Validator SHALL validate `quota_type` as one of `unlimited`, `monthly_quota`, `fup` for PPPoE packages, or one of `unlimited`, `quota` for Voucher packages
6. THE Validator SHALL validate `monthly_price` as a positive integer (in Rupiah) when `type` is `pppoe`
7. THE Validator SHALL validate `sell_price` and `reseller_price` as positive integers (in Rupiah) when `type` is `voucher`
8. THE Validator SHALL validate that `reseller_price` is strictly less than `sell_price` and that `sell_price - reseller_price >= 500` when `type` is `voucher`
9. THE Validator SHALL validate `duration_value` as a positive integer and `duration_unit` as one of `hours`, `days`, `weeks`, `months` when `type` is `voucher`
10. THE Validator SHALL validate burst fields (`burst_download_mbps`, `burst_upload_mbps`, `burst_threshold_mbps`, `burst_time_seconds`) as positive integers when provided, and require all four to be present together or none at all
11. THE Validator SHALL validate `installation_fee` as a non-negative integer (default 0) when `type` is `pppoe`
12. THE Validator SHALL validate `shared_users` as a positive integer (default 1) when `type` is `voucher`
13. THE Validator SHALL return all validation errors in a single response with HTTP 400, error code `VALIDATION_ERROR`, and an array of field-level error details

### Requirement 15: Reseller Price Margin Integrity

**User Story:** As a developer, I want the reseller price margin enforced at the domain level, so that no voucher package can be created or updated with an insufficient margin regardless of the caller.

#### Acceptance Criteria

1. THE Billing_API domain layer SHALL enforce that for all Voucher packages, `reseller_price < sell_price` and `sell_price - reseller_price >= 500`
2. WHEN a voucher package is created or updated with a margin less than 500 Rupiah, THE Billing_API SHALL return HTTP 400 with error code `INSUFFICIENT_MARGIN` and include the current margin and the minimum required margin of 500
3. FOR ALL valid Voucher_Package sell_price and reseller_price combinations, the margin (sell_price minus reseller_price) SHALL be at least 500 Rupiah (margin integrity property)

### Requirement 16: Package Type Field Consistency

**User Story:** As a developer, I want the package type to determine which fields are required and which are ignored, so that PPPoE and Voucher packages have consistent and correct data.

#### Acceptance Criteria

1. WHEN `type` is `pppoe`, THE Billing_API SHALL require `monthly_price` and `bandwidth_type`, and SHALL ignore `sell_price`, `reseller_price`, `duration_value`, `duration_unit`, and `shared_users`
2. WHEN `type` is `voucher`, THE Billing_API SHALL require `sell_price`, `reseller_price`, `duration_value`, and `duration_unit`, and SHALL ignore `monthly_price`, `installation_fee`, and `bandwidth_type`
3. THE Billing_API SHALL NOT allow the `type` field to be changed after package creation
4. FOR ALL packages, the set of non-null type-specific fields SHALL be consistent with the package type (type-field consistency property)

### Requirement 17: Customer Count Computation

**User Story:** As an operator, I want to see how many customers are using each package, so that I can make informed decisions about package changes and deletions.

#### Acceptance Criteria

1. WHEN listing packages, THE Billing_API SHALL compute `customer_count` for each package by counting rows in the `customers` table where `package_id` matches and `deleted_at IS NULL`
2. WHEN viewing a package detail, THE Billing_API SHALL include the computed `customer_count` in the response
3. THE Billing_API SHALL NOT store `customer_count` as a column on the `packages` table — the value is always computed at query time
