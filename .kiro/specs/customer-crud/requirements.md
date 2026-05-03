# Requirements Document

## Introduction

This spec defines the Customer Management module for ISPBoss — a SaaS billing platform for ISPs. The module replaces the sample `customers` table (migration 000002) with a full customer schema and implements all CRUD operations, area management, bulk actions, import/export, status transitions, and audit trail. All data is tenant-scoped via RLS and RBAC-protected. The module publishes events to Redis for downstream services (Network, Notification).

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles customer, billing, and auth operations
- **Customer**: An ISP subscriber managed by a tenant, stored in the `customers` table
- **Area**: A geographic grouping of customers (e.g., RT 03/05 Sukamaju), stored in the `areas` table
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Customer_ID_Seq**: A human-readable auto-increment identifier per tenant in the format `PLG-001`, `PLG-002`, etc.
- **PPPoE**: Point-to-Point Protocol over Ethernet — a connection method for ISP subscribers
- **Connection_Method**: The type of internet connection: `pppoe`, `hotspot`, `dhcp_binding`, or `static`
- **Customer_Status**: One of `pending`, `aktif`, `isolir`, `suspend`, or `berhenti`
- **Isolir**: A state where the customer's internet is redirected to a walled garden due to unpaid invoices
- **Suspend**: A state where the customer's connection is fully terminated (PPPoE user removed from router)
- **Soft_Delete**: Marking a record as deleted (setting `deleted_at` timestamp) without physically removing it from the database
- **RLS**: PostgreSQL Row Level Security — a database-level safety net ensuring tenant data isolation
- **Audit_Log**: A record of all changes to customer data, stored in the `audit_logs` table
- **Event_Queue**: Redis-based message queue (asynq) for inter-service communication
- **Validator**: The `go-playground/validator` library used for input validation in the Billing_API
- **Import_Job**: An asynchronous background job (asynq) that processes CSV/Excel file imports
- **Export_Job**: An asynchronous background job (asynq) that generates CSV/Excel file exports
- **Credit_Balance**: A monetary balance on a customer account from overpayment, downgrade prorate, or credit notes
- **Prorate**: A proportional charge or credit when a customer changes packages mid-billing cycle

## Requirements

### Requirement 1: Customer Database Schema

**User Story:** As a tenant admin, I want a comprehensive customer database schema, so that all customer data (personal info, service config, network details) is stored in a single, well-indexed table with proper multi-tenant isolation.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `customers` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `customer_id_seq` (VARCHAR, unique per tenant), `name` (VARCHAR NOT NULL), `phone` (VARCHAR NOT NULL), `email` (VARCHAR), `address` (TEXT NOT NULL), `area_id` (UUID FK), `latitude` (DECIMAL NOT NULL), `longitude` (DECIMAL NOT NULL), `package_id` (UUID NOT NULL), `activation_date` (DATE NOT NULL), `due_date` (INTEGER NOT NULL, 1-28), `connection_method` (VARCHAR NOT NULL), `pppoe_username` (VARCHAR), `pppoe_password` (VARCHAR), `mac_address` (VARCHAR), `router_id` (UUID), `odp_port` (VARCHAR), `credit_balance` (BIGINT DEFAULT 0), `notes` (TEXT), `status` (VARCHAR NOT NULL DEFAULT 'pending'), `deleted_at` (TIMESTAMPTZ), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `customers` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL create composite indexes on `(tenant_id, status)`, `(tenant_id, customer_id_seq)`, `(tenant_id, phone)`, `(tenant_id, area_id)`, `(tenant_id, package_id)`, and `(tenant_id, due_date)` for query performance
4. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, phone)` to prevent duplicate phone numbers within a tenant
5. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, customer_id_seq)` to prevent duplicate customer IDs within a tenant
6. THE Billing_API migration SHALL enforce a CHECK constraint on `due_date` to accept only values between 1 and 28
7. THE Billing_API migration SHALL enforce a CHECK constraint on `connection_method` to accept only `pppoe`, `hotspot`, `dhcp_binding`, or `static`
8. THE Billing_API migration SHALL enforce a CHECK constraint on `status` to accept only `pending`, `aktif`, `isolir`, `suspend`, or `berhenti`

### Requirement 2: Area Database Schema

**User Story:** As a tenant admin, I want to organize customers by geographic area, so that I can manage, filter, and report on customers by location.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `areas` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `name` (VARCHAR NOT NULL), `description` (TEXT), `odp_id` (VARCHAR), `center_lat` (DECIMAL), `center_lng` (DECIMAL), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `areas` table with tenant isolation policies
3. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, name)` to prevent duplicate area names within a tenant
4. THE Billing_API migration SHALL create an index on `(tenant_id)` for the `areas` table

### Requirement 3: Audit Log Schema

**User Story:** As a tenant admin, I want all customer changes logged, so that I can review who changed what and when for accountability and troubleshooting.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `audit_logs` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `entity_type` (VARCHAR NOT NULL), `entity_id` (UUID NOT NULL), `action` (VARCHAR NOT NULL), `actor_id` (UUID NOT NULL), `actor_name` (VARCHAR NOT NULL), `changes` (JSONB), `metadata` (JSONB), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `audit_logs` table with tenant isolation policies
3. THE Billing_API migration SHALL create composite indexes on `(tenant_id, entity_type, entity_id)` and `(tenant_id, created_at)` for query performance

### Requirement 4: Customer ID Auto-Generation

**User Story:** As a tenant admin, I want each customer to receive a unique, human-readable ID (e.g., PLG-001), so that customers can be easily identified in conversations and documents.

#### Acceptance Criteria

1. WHEN a new customer is created, THE Billing_API SHALL auto-generate a `customer_id_seq` by finding the highest existing sequence number for the tenant and incrementing by one
2. THE Billing_API SHALL format the customer ID as `PLG-{seq}` where `{seq}` is zero-padded to 3 digits (e.g., PLG-001, PLG-002), expanding to more digits when the sequence exceeds 999 (e.g., PLG-1000)
3. THE Billing_API SHALL ensure the generated `customer_id_seq` is unique per tenant using a database-level unique constraint and application-level retry on conflict

### Requirement 5: PPPoE Username Auto-Generation

**User Story:** As an operator, I want PPPoE usernames to be auto-generated from the customer's name and ID, so that I don't have to manually create unique usernames for each customer.

#### Acceptance Criteria

1. WHEN a customer is created with `connection_method` set to `pppoe` and no `pppoe_username` is provided, THE Billing_API SHALL auto-generate a username in the format `{first-name-lowercase}-{customer-id-lowercase}` (e.g., `ahmad-plg001`)
2. WHEN a customer is created with `connection_method` set to `pppoe` and no `pppoe_password` is provided, THE Billing_API SHALL auto-generate a random alphanumeric password of 8 characters
3. WHILE `connection_method` is set to `pppoe`, THE Billing_API SHALL require both `pppoe_username` and `pppoe_password` to be present (either provided or auto-generated)

### Requirement 6: Customer List API

**User Story:** As an operator, I want to list customers with pagination, search, filtering, and sorting, so that I can quickly find and manage customers.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/customers`, THE Billing_API SHALL return a paginated list of customers for the authenticated tenant
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `search` query parameter is provided, THE Billing_API SHALL filter customers whose `name`, `customer_id_seq`, `address`, or `phone` contains the search term (case-insensitive)
4. WHEN `status`, `package_id`, `area_id`, or `due_date` query parameters are provided, THE Billing_API SHALL filter customers matching the specified values
5. WHEN a `sort_by` query parameter is provided, THE Billing_API SHALL sort results by the specified column (`name`, `customer_id_seq`, `status`, `created_at`, `due_date`) in the direction specified by `sort_order` (asc or desc, default asc)
6. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
7. THE Billing_API SHALL exclude soft-deleted customers (where `deleted_at` is not null) from list results

### Requirement 7: Customer Detail API

**User Story:** As an operator, I want to view a customer's full details including service info, so that I can understand the customer's current state and history.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/customers/:id`, THE Billing_API SHALL return the full customer record including all fields
2. WHEN a GET request is made to `/v1/customers/:id` with `include=audit_logs`, THE Billing_API SHALL include the customer's audit log entries in the response
3. IF the customer ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`
4. IF the customer has been soft-deleted, THEN THE Billing_API SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`

### Requirement 8: Customer Create API

**User Story:** As an operator, I want to create a new customer with full validation, so that customer data is complete and consistent from the start.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers` with valid data, THE Billing_API SHALL create a new customer with status `pending` and return the created customer with HTTP 201
2. THE Validator SHALL require the following fields: `name` (min 3 characters), `phone` (format +62, 10-15 digits), `address`, `latitude`, `longitude`, `package_id` (must reference an existing package), `activation_date`, `due_date` (1-28), `connection_method` (one of pppoe/hotspot/dhcp_binding/static)
3. WHILE `connection_method` is `dhcp_binding`, THE Validator SHALL require `mac_address` in the format `AA:BB:CC:DD:EE:FF`
4. IF the `phone` number already exists for the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PHONE_DUPLICATE`
5. WHEN a customer is successfully created, THE Billing_API SHALL publish a `customer.created` event to the Event_Queue with the customer's data
6. WHEN a customer is successfully created, THE Billing_API SHALL write an audit log entry with action `customer.created`

### Requirement 9: Customer Update API

**User Story:** As an operator, I want to update customer data, so that I can correct information or change service configuration.

#### Acceptance Criteria

1. WHEN a PUT request is made to `/v1/customers/:id` with valid data, THE Billing_API SHALL update the customer record and return the updated customer
2. THE Validator SHALL apply the same validation rules as customer creation for all provided fields
3. IF the updated `phone` number already exists for another customer in the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `PHONE_DUPLICATE`
4. IF the customer ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`
5. WHEN a customer is successfully updated, THE Billing_API SHALL write an audit log entry with action `customer.updated` including the old and new values of changed fields

### Requirement 10: Customer Soft Delete API

**User Story:** As a tenant admin, I want to soft-delete a customer with a safety confirmation, so that accidental deletions are prevented and data can be recovered.

#### Acceptance Criteria

1. WHEN a DELETE request is made to `/v1/customers/:id` with a `confirmation_name` field matching the customer's name, THE Billing_API SHALL set the `deleted_at` timestamp and return HTTP 200
2. IF the `confirmation_name` does not match the customer's name (case-sensitive), THEN THE Billing_API SHALL return HTTP 400 with error code `CONFIRMATION_MISMATCH`
3. IF the customer ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`
4. WHEN a customer is successfully soft-deleted, THE Billing_API SHALL write an audit log entry with action `customer.deleted`
5. WHEN a customer is successfully soft-deleted, THE Billing_API SHALL publish a `customer.terminated` event to the Event_Queue

### Requirement 11: Customer Status Transitions

**User Story:** As an operator, I want to change a customer's status (isolir, activate, suspend), so that I can manage their internet service based on payment status.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/:id/isolir`, THE Billing_API SHALL transition the customer's status from `aktif` to `isolir` and publish a `customer.isolated` event to the Event_Queue
2. WHEN a POST request is made to `/v1/customers/:id/activate`, THE Billing_API SHALL transition the customer's status from `isolir` or `suspend` or `pending` to `aktif` and publish a `customer.activated` or `customer.unblocked` event to the Event_Queue
3. IF a status transition is requested that violates the state machine (e.g., `berhenti` to `aktif`), THEN THE Billing_API SHALL return HTTP 422 with error code `INVALID_STATUS_TRANSITION` and a message describing the allowed transitions
4. THE Billing_API SHALL enforce the following valid transitions: `pending` → `aktif`, `aktif` → `isolir`, `aktif` → `berhenti`, `isolir` → `aktif`, `isolir` → `suspend`, `isolir` → `berhenti`, `suspend` → `aktif`, `suspend` → `berhenti`
5. WHEN any status transition occurs, THE Billing_API SHALL write an audit log entry with action `customer.status_changed` including the old and new status

### Requirement 12: Customer Package Change API

**User Story:** As an operator, I want to change a customer's internet package, so that upgrades and downgrades are tracked and trigger the appropriate network and billing actions.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/:id/change-package` with a valid `package_id`, THE Billing_API SHALL update the customer's `package_id` and publish a `package.changed` event to the Event_Queue
2. IF the `package_id` does not reference an existing package, THEN THE Billing_API SHALL return HTTP 400 with error code `PACKAGE_NOT_FOUND`
3. IF the customer's current `package_id` is the same as the requested `package_id`, THEN THE Billing_API SHALL return HTTP 400 with error code `SAME_PACKAGE`
4. WHEN a package change is successful, THE Billing_API SHALL write an audit log entry with action `customer.package_changed` including the old and new package IDs

### Requirement 13: Area CRUD API

**User Story:** As a tenant admin, I want to manage areas (create, list, update, delete), so that I can organize customers by geographic location.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/areas`, THE Billing_API SHALL return all areas for the authenticated tenant with the count of customers in each area
2. WHEN a POST request is made to `/v1/areas` with valid data, THE Billing_API SHALL create a new area and return it with HTTP 201
3. THE Validator SHALL require the `name` field (min 2 characters) when creating or updating an area
4. IF the area `name` already exists for the same tenant, THEN THE Billing_API SHALL return HTTP 409 with error code `AREA_NAME_DUPLICATE`
5. WHEN a PUT request is made to `/v1/areas/:id` with valid data, THE Billing_API SHALL update the area and return the updated area
6. WHEN a DELETE request is made to `/v1/areas/:id`, THE Billing_API SHALL delete the area only if no customers reference it
7. IF the area has associated customers, THEN THE Billing_API SHALL return HTTP 409 with error code `AREA_HAS_CUSTOMERS` and include the count of associated customers

### Requirement 14: Bulk Actions API

**User Story:** As an operator, I want to perform actions on multiple customers at once (isolir, activate, notify, change package, edit, delete), so that I can efficiently manage large groups of customers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/bulk/isolir` with an array of customer IDs, THE Billing_API SHALL transition each eligible customer's status to `isolir` and return a summary of successes and failures
2. WHEN a POST request is made to `/v1/customers/bulk/activate` with an array of customer IDs, THE Billing_API SHALL transition each eligible customer's status to `aktif` and return a summary of successes and failures
3. WHEN a POST request is made to `/v1/customers/bulk/notification` with an array of customer IDs and a notification template, THE Billing_API SHALL publish notification events for each customer to the Event_Queue
4. WHEN a POST request is made to `/v1/customers/bulk/change-package` with an array of customer IDs and a `package_id`, THE Billing_API SHALL update each customer's package and publish `package.changed` events
5. WHEN a POST request is made to `/v1/customers/bulk/edit` with an array of customer IDs and fields to update (`area_id`, `due_date`, `notes`), THE Billing_API SHALL update the specified fields for each customer
6. WHEN a DELETE request is made to `/v1/customers/bulk` with an array of customer IDs, THE Billing_API SHALL soft-delete each customer and publish `customer.terminated` events
7. THE Billing_API SHALL return a response containing `total`, `success_count`, `failure_count`, and an array of `failures` with `customer_id` and `reason` for each failed operation
8. THE Billing_API SHALL write audit log entries for each individual customer affected by a bulk action

### Requirement 15: Customer Import API

**User Story:** As a tenant admin, I want to import customers from CSV or Excel files, so that I can migrate existing customer data into ISPBoss without manual entry.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/import` with a CSV or Excel file, THE Billing_API SHALL enqueue an Import_Job and return HTTP 202 with a `job_id`
2. THE Import_Job SHALL validate each row against the same rules as customer creation (name required, phone format +62, coordinates required, valid connection method)
3. WHEN the Import_Job completes, THE Billing_API SHALL store the result including `total_rows`, `success_count`, `failure_count`, and a downloadable error log with row numbers and error descriptions
4. IF a row has a phone number that already exists for the tenant, THEN THE Import_Job SHALL skip that row and record it as a failure with reason `PHONE_DUPLICATE`
5. WHEN a GET request is made to `/v1/customers/import/template`, THE Billing_API SHALL return a downloadable CSV template file with the correct column headers and one example row

### Requirement 16: Customer Export API

**User Story:** As a tenant admin, I want to export customer data to CSV or Excel, so that I can use the data in external tools or for reporting.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/customers/export` with a `format` parameter (`csv` or `xlsx`), THE Billing_API SHALL enqueue an Export_Job and return HTTP 202 with a `job_id`
2. THE Export_Job SHALL apply the same filters as the customer list API (status, package_id, area_id, due_date, search) if provided as query parameters
3. WHEN the Export_Job completes, THE Billing_API SHALL store the generated file and make it available for download
4. THE Export_Job SHALL include all customer fields in the export, with `area_name` and `package_name` resolved from their respective tables
5. WHEN a `columns` query parameter is provided as a comma-separated list of field names, THE Export_Job SHALL include only the specified columns in the export; IF `columns` is not provided, THE Export_Job SHALL include all columns

### Requirement 17: Customer Quick Stats API

**User Story:** As an operator, I want to see a count of customers per status at a glance, so that I can quickly understand the health of the customer base.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/customers/stats`, THE Billing_API SHALL return the count of customers grouped by status (`aktif`, `pending`, `isolir`, `suspend`, `berhenti`) for the authenticated tenant
2. THE Billing_API SHALL exclude soft-deleted customers from the stats counts

### Requirement 18: RBAC for Customer Endpoints

**User Story:** As a tenant admin, I want customer endpoints protected by role-based access control, so that only authorized users can view or modify customer data.

#### Acceptance Criteria

1. THE Billing_API SHALL allow `super_admin`, `tenant_admin`, and `operator` roles full access (all HTTP methods) to all customer and area endpoints
2. THE Billing_API SHALL allow the `kasir` role read-only access (GET method only) to customer list and detail endpoints
3. THE Billing_API SHALL deny access to `teknisi` and `reseller` roles for all customer and area endpoints, returning HTTP 403 with error code `FORBIDDEN`
4. THE Billing_API SHALL allow only `tenant_admin` (and `super_admin`) roles to access import, export, and bulk delete endpoints

### Requirement 19: Multi-Tenant Data Isolation

**User Story:** As a platform operator, I want all customer data strictly isolated per tenant, so that no tenant can access another tenant's customer data.

#### Acceptance Criteria

1. THE Billing_API SHALL set the PostgreSQL session variable `app.tenant_id` from the JWT claims before every database query on tenant-scoped tables
2. THE Billing_API SHALL include `WHERE tenant_id = ?` in all customer and area repository queries as the application-level filter
3. THE Billing_API RLS policies SHALL act as a safety net, blocking any query where `tenant_id` does not match the session variable
4. THE Billing_API SHALL prevent any customer or area API endpoint from accepting a `tenant_id` in the request body — the tenant is always derived from the authenticated JWT token

### Requirement 20: Audit Trail for Customer Operations

**User Story:** As a tenant admin, I want a complete audit trail of all customer changes, so that I can track who did what and when for compliance and troubleshooting.

#### Acceptance Criteria

1. WHEN a customer is created, updated, soft-deleted, or has a status change, THE Billing_API SHALL insert a record into the `audit_logs` table with `entity_type` set to `customer`, the `entity_id` set to the customer's UUID, the `action` describing the operation, and the `actor_id` and `actor_name` from the authenticated user
2. WHEN a customer is updated, THE Billing_API SHALL store the old and new values of changed fields in the `changes` JSONB column
3. WHEN a GET request is made to `/v1/customers/:id` with `include=audit_logs`, THE Billing_API SHALL return audit log entries for that customer sorted by `created_at` descending
4. THE Billing_API SHALL store audit log entries for the following actions: `customer.created`, `customer.updated`, `customer.deleted`, `customer.status_changed`, `customer.package_changed`

### Requirement 21: Event Publishing for Inter-Service Communication

**User Story:** As a platform developer, I want customer lifecycle events published to the event queue, so that downstream services (Network, Notification) can react to customer changes.

#### Acceptance Criteria

1. WHEN a customer is created, THE Billing_API SHALL publish a `customer.created` event to the Event_Queue containing `tenant_id`, `customer_id`, `name`, `package_id`, `connection_method`, and `router_id`
2. WHEN a customer is activated (status → `aktif`), THE Billing_API SHALL publish a `customer.activated` event containing `customer_id`, `name`, `package_id`, `connection_method`, `pppoe_username`, `pppoe_password`, and `router_id`
3. WHEN a customer is isolated (status → `isolir`), THE Billing_API SHALL publish a `customer.isolated` event containing `customer_id`, `name`, `router_id`, and `pppoe_username`
4. WHEN a customer is unblocked (isolir → `aktif`), THE Billing_API SHALL publish a `customer.unblocked` event containing `customer_id`, `name`, `router_id`, and `pppoe_username`
5. WHEN a customer is terminated (soft-deleted or status → `berhenti`), THE Billing_API SHALL publish a `customer.terminated` event containing `customer_id`, `name`, `router_id`, and `pppoe_username`
6. WHEN a customer's package is changed, THE Billing_API SHALL publish a `package.changed` event containing `customer_id`, `old_package_id`, `new_package_id`, `connection_method`, and `router_id`
7. THE Billing_API SHALL include `tenant_id`, `timestamp`, and `correlation_id` (UUID v4) in every published event envelope

### Requirement 22: Field Validation Rules

**User Story:** As a developer, I want all customer input validated consistently, so that invalid data never enters the database.

#### Acceptance Criteria

1. THE Validator SHALL validate `phone` as starting with `+62` followed by 9 to 13 digits (total 12-16 characters including the `+62` prefix)
2. THE Validator SHALL validate `email` as a valid email format when provided (optional field)
3. THE Validator SHALL validate `latitude` as a decimal between -90 and 90, and `longitude` as a decimal between -180 and 180
4. THE Validator SHALL validate `mac_address` as six groups of two hexadecimal digits separated by colons (e.g., `AA:BB:CC:DD:EE:FF`) when `connection_method` is `dhcp_binding`
5. THE Validator SHALL validate `due_date` as an integer between 1 and 28 inclusive
6. THE Validator SHALL validate `name` as a string with minimum 3 characters and maximum 255 characters
7. THE Validator SHALL validate `address` as a non-empty string with maximum 1000 characters
8. THE Validator SHALL return all validation errors in a single response with HTTP 400, error code `VALIDATION_ERROR`, and an array of field-level error details

### Requirement 23: Customer Status State Machine Integrity

**User Story:** As a developer, I want the customer status state machine enforced at the domain level, so that invalid transitions are impossible regardless of the caller.

#### Acceptance Criteria

1. THE Billing_API domain layer SHALL define the valid status transitions as: `pending` → [`aktif`], `aktif` → [`isolir`, `berhenti`], `isolir` → [`aktif`, `suspend`, `berhenti`], `suspend` → [`aktif`, `berhenti`], `berhenti` → [] (terminal state, no transitions out)
2. WHEN an invalid status transition is attempted, THE Billing_API SHALL return HTTP 422 with error code `INVALID_STATUS_TRANSITION` and include the current status and the list of allowed target statuses
3. FOR ALL valid Customer_Status values and FOR ALL valid transitions, applying a transition and then checking the resulting status SHALL yield the expected target status (state machine determinism property)
