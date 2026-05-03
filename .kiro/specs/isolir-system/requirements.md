# Requirements Document

## Introduction

This spec defines the Isolir System module for ISPBoss — a SaaS billing platform for ISPs. The module implements automatic isolation (isolir) of customers who have unpaid invoices past the grace period, automatic un-isolation (buka isolir) when payment is received, suspension of customers who exceed the tolerance limit, late fee (denda) calculation and application to overdue invoices, retry mechanism with exponential backoff for MikroTik command delivery, pending sync tracking for failed router operations, and periodic sync background jobs. The module publishes events to the asynq queue for downstream consumption by the MikroTik module (future), notification module, and payment gateway module. All operations are tenant-scoped and configurable via billing_settings. Customer status transitions follow the existing state machine (aktif → isolir → suspend). The module integrates with existing invoice, payment, and customer infrastructure.

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles invoice, billing, customer, and auth operations
- **Isolir_Worker**: The asynq worker component within Billing_API that processes isolir, un-isolir, and suspend cron tasks
- **Isolir_Usecase**: The usecase layer component that orchestrates isolir business logic including customer status transitions, event publishing, and audit logging
- **Customer**: An ISP subscriber with a status field (aktif, isolir, suspend, berhenti), stored in the `customers` table
- **Customer_Status**: One of `pending`, `aktif`, `isolir`, `suspend`, or `berhenti` as defined in the existing state machine
- **Invoice**: A billing document with status (belum_bayar, terlambat, lunas, bayar_sebagian, batal, prorate), stored in the `invoices` table
- **Billing_Settings**: Tenant-level configuration including `grace_period_days`, `suspend_days`, `auto_isolir`, `auto_open_isolir`, and penalty settings
- **Grace_Period**: The number of days after invoice due_date before auto-isolir is triggered (default 7, configurable per tenant)
- **Suspend_Days**: The number of days after invoice due_date before customer is suspended (default 30, configurable per tenant)
- **Pending_Sync**: A tracking record indicating that a customer's status in the database differs from the router state, requiring synchronization by the MikroTik module
- **Event_Queue**: Redis-based message queue (asynq) for inter-service communication and background jobs
- **Late_Fee**: An optional penalty charged for overdue invoices, configurable as fixed amount, percentage of subtotal, or daily rate with a maximum cap
- **Invoice_Audit_Log**: An append-only lifecycle log for invoice operations, stored in the `invoice_audit_logs` table
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **MikroTik_Module**: A future module (08-mikrotik) that will consume isolir/un-isolir/suspend events and execute router commands
- **Sync_Operation**: A record tracking the type of router operation needed (isolir, un_isolir, suspend) with retry state and timestamps

## Requirements

### Requirement 1: Pending Sync Database Schema

**User Story:** As a developer, I want a dedicated table to track pending router synchronization operations, so that the MikroTik module can pick up and execute operations that need to be synced to routers.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `pending_syncs` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `customer_id` (UUID FK NOT NULL REFERENCES customers(id)), `operation_type` (VARCHAR NOT NULL), `status` (VARCHAR NOT NULL DEFAULT 'pending'), `retry_count` (INTEGER NOT NULL DEFAULT 0), `max_retries` (INTEGER NOT NULL DEFAULT 5), `last_retry_at` (TIMESTAMPTZ), `next_retry_at` (TIMESTAMPTZ), `error_message` (TEXT), `metadata` (JSONB), `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW()), `updated_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())
2. THE Billing_API migration SHALL enable Row Level Security on the `pending_syncs` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL enforce a CHECK constraint on `operation_type` to accept only `isolir`, `un_isolir`, or `suspend`
4. THE Billing_API migration SHALL enforce a CHECK constraint on `status` to accept only `pending`, `in_progress`, `completed`, or `failed`
5. THE Billing_API migration SHALL create composite indexes on `(tenant_id, customer_id)`, `(tenant_id, status)`, and `(status, next_retry_at)` for query performance
6. THE Billing_API migration SHALL enforce a CHECK constraint on `retry_count` to accept only values between 0 and `max_retries` inclusive

### Requirement 2: Auto-Isolir Cron Job

**User Story:** As a tenant admin, I want customers with unpaid invoices past the grace period to be automatically isolated, so that non-paying customers have their internet access restricted without manual intervention.

#### Acceptance Criteria

1. THE Isolir_Worker SHALL register a daily cron job task `isolir.auto_isolir_cron` scheduled at 01:00 (using the tenant's configured timezone)
2. WHEN the auto-isolir cron job runs, THE Isolir_Usecase SHALL scan all tenants where `auto_isolir` is enabled in Billing_Settings
3. WHEN scanning a tenant, THE Isolir_Usecase SHALL identify customers with status `aktif` who have at least one invoice with status `belum_bayar` or `terlambat` where the current date exceeds `due_date` plus `grace_period_days`
4. WHEN an eligible customer is found, THE Isolir_Usecase SHALL transition the customer status from `aktif` to `isolir` using the existing state machine
5. WHEN a customer is transitioned to `isolir`, THE Isolir_Usecase SHALL publish a `customer.isolir` event to the Event_Queue containing the customer_id, tenant_id, customer name, router_id, pppoe_username, and connection_method
6. WHEN a customer is transitioned to `isolir`, THE Isolir_Usecase SHALL create a pending_sync record with operation_type `isolir` and status `pending`
7. WHEN a customer is transitioned to `isolir`, THE Isolir_Usecase SHALL write an invoice audit log entry with action `customer.isolir` and actor `System` with metadata containing the number of overdue days
8. THE Isolir_Usecase SHALL be idempotent — running the cron job multiple times for the same day SHALL NOT re-isolir customers who are already in `isolir` status
9. WHEN a customer is transitioned to `isolir`, THE Isolir_Usecase SHALL publish a `notification.isolir` event to the Event_Queue for downstream notification delivery

### Requirement 3: Auto-Buka Isolir (Un-Isolation on Payment)

**User Story:** As a tenant admin, I want isolated customers to be automatically un-isolated when their payment is received, so that paying customers regain internet access immediately without waiting for manual intervention.

#### Acceptance Criteria

1. WHEN a `payment.online.received` or `payment.recorded` event is received and the tenant's `auto_open_isolir` setting is enabled, THE Isolir_Usecase SHALL check if the paying customer's status is `isolir`
2. WHEN the paying customer's status is `isolir` and all outstanding invoices for that customer are now `lunas`, THE Isolir_Usecase SHALL transition the customer status from `isolir` to `aktif` using the existing state machine
3. WHEN a customer is transitioned from `isolir` to `aktif`, THE Isolir_Usecase SHALL publish a `customer.un_isolir` event to the Event_Queue containing the customer_id, tenant_id, customer name, router_id, pppoe_username, and connection_method
4. WHEN a customer is transitioned from `isolir` to `aktif`, THE Isolir_Usecase SHALL create a pending_sync record with operation_type `un_isolir` and status `pending`
5. WHEN a customer is transitioned from `isolir` to `aktif`, THE Isolir_Usecase SHALL write an invoice audit log entry with action `customer.un_isolir` and actor `System` with metadata indicating payment received
6. WHEN a customer is transitioned from `isolir` to `aktif`, THE Isolir_Usecase SHALL publish a `notification.un_isolir` event to the Event_Queue for downstream notification delivery
7. IF the customer still has outstanding invoices with status `belum_bayar`, `terlambat`, or `bayar_sebagian` after the payment, THEN THE Isolir_Usecase SHALL NOT transition the customer status and SHALL remain in `isolir`
8. WHEN a `payment.voided.re_isolir` event is received, THE Isolir_Usecase SHALL check if the customer's status is `aktif` and the customer has outstanding invoices past the grace period, and if so, transition the customer back to `isolir` (re-isolir after void)
9. WHEN a customer is re-isolated due to payment void, THE Isolir_Usecase SHALL create a pending_sync record with operation_type `isolir`, publish `customer.isolir` and `notification.isolir` events, and write an audit log entry with action `customer.re_isolir` and metadata indicating void triggered the re-isolation

### Requirement 4: Suspend Cron Job (Tolerance Limit Exceeded)

**User Story:** As a tenant admin, I want customers who remain isolated beyond the tolerance limit to be automatically suspended, so that long-term non-paying customers are fully disconnected and their router resources are freed.

#### Acceptance Criteria

1. THE Isolir_Worker SHALL register a daily cron job task `isolir.suspend_cron` scheduled at 02:00 (using the tenant's configured timezone)
2. WHEN the suspend cron job runs, THE Isolir_Usecase SHALL scan all tenants and identify customers with status `isolir` who have at least one invoice with status `belum_bayar` or `terlambat` where the current date exceeds `due_date` plus `suspend_days`
3. WHEN an eligible customer is found, THE Isolir_Usecase SHALL transition the customer status from `isolir` to `suspend` using the existing state machine
4. WHEN a customer is transitioned to `suspend`, THE Isolir_Usecase SHALL publish a `customer.suspend` event to the Event_Queue containing the customer_id, tenant_id, customer name, router_id, pppoe_username, and connection_method
5. WHEN a customer is transitioned to `suspend`, THE Isolir_Usecase SHALL create a pending_sync record with operation_type `suspend` and status `pending`
6. WHEN a customer is transitioned to `suspend`, THE Isolir_Usecase SHALL write an invoice audit log entry with action `customer.suspend` and actor `System` with metadata containing the number of overdue days
7. WHEN a customer is transitioned to `suspend`, THE Isolir_Usecase SHALL publish a `notification.suspend` event to the Event_Queue for downstream notification delivery
8. THE Isolir_Usecase SHALL be idempotent — running the cron job multiple times SHALL NOT re-suspend customers who are already in `suspend` status

### Requirement 5: Pending Sync Retry Mechanism

**User Story:** As a developer, I want pending sync operations to be retried with exponential backoff, so that transient router failures are automatically recovered without manual intervention.

#### Acceptance Criteria

1. THE Isolir_Worker SHALL register a periodic task `isolir.periodic_sync` that runs every 15 minutes
2. WHEN the periodic sync task runs, THE Isolir_Usecase SHALL query all pending_sync records with status `pending` where `next_retry_at` is null or less than or equal to the current time
3. WHEN a pending_sync record is processed, THE Isolir_Usecase SHALL publish the corresponding event (`customer.isolir`, `customer.un_isolir`, or `customer.suspend`) to the Event_Queue and increment the `retry_count`
4. WHEN a pending_sync record's `retry_count` is incremented, THE Isolir_Usecase SHALL calculate the `next_retry_at` using exponential backoff: retry 1 = immediate, retry 2 = 5 minutes, retry 3 = 30 minutes, retry 4 = 2 hours, retry 5 = 6 hours
5. WHEN a pending_sync record reaches `max_retries` (default 5) without completion, THE Isolir_Usecase SHALL update the status to `failed` and publish a `notification.pending_sync_failed` event to notify the admin
6. WHEN the MikroTik module confirms a sync operation is complete, THE Isolir_Usecase SHALL update the pending_sync record status to `completed`
7. THE Isolir_Usecase SHALL process pending_sync records in batches of 50 to avoid overwhelming the queue

### Requirement 6: Manual Sync Trigger API

**User Story:** As a tenant admin, I want to manually trigger synchronization for a specific customer or all pending operations, so that I can resolve sync issues without waiting for the periodic job.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/isolir/sync/:customer_id`, THE Billing_API SHALL re-publish the pending sync event for the specified customer and reset the retry_count to 0
2. WHEN a POST request is made to `/v1/isolir/sync-all`, THE Billing_API SHALL re-publish events for all pending_sync records with status `pending` or `failed` for the authenticated tenant
3. IF the customer_id does not have any pending_sync records, THEN THE Billing_API SHALL return HTTP 404 with error code `NO_PENDING_SYNC`
4. WHEN a manual sync is triggered, THE Billing_API SHALL write an invoice audit log entry with action `sync.manual_trigger` and the actor set to the authenticated admin
5. THE Billing_API SHALL return the count of re-published sync operations in the response

### Requirement 7: Pending Sync Status API

**User Story:** As a tenant admin, I want to view the list of customers with pending sync operations, so that I can identify and resolve synchronization issues between the database and routers.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/isolir/pending-syncs`, THE Billing_API SHALL return a paginated list of pending_sync records for the authenticated tenant
2. THE Billing_API SHALL include the customer name, customer_id_seq, operation_type, status, retry_count, last_retry_at, and error_message in each record
3. WHEN a `status` query parameter is provided, THE Billing_API SHALL filter records by the specified status (pending, in_progress, completed, failed)
4. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
5. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response

### Requirement 8: Invoice Overdue with Late Fee Processing

**User Story:** As a tenant admin, I want late fees automatically calculated and added to overdue invoices, so that penalty policies are consistently enforced when invoices pass their due date.

#### Acceptance Criteria

1. WHEN the existing `invoice.overdue_cron` task transitions an invoice from `belum_bayar` to `terlambat`, THE Isolir_Usecase SHALL check if the tenant's `penalty_enabled` setting is true
2. WHILE the tenant's `penalty_enabled` setting is true, THE Isolir_Usecase SHALL calculate the late fee based on the configured `penalty_type`: fixed amount, percentage of subtotal, or daily rate multiplied by days overdue
3. WHEN `penalty_type` is `fixed`, THE Isolir_Usecase SHALL apply the configured `penalty_amount` as the late fee
4. WHEN `penalty_type` is `percentage`, THE Isolir_Usecase SHALL calculate the late fee as `invoice.subtotal * penalty_percentage / 100`
5. WHEN `penalty_type` is `daily`, THE Isolir_Usecase SHALL calculate the late fee as `penalty_daily_amount * days_overdue` where days_overdue is the number of days between the due_date and the current date
6. WHEN a `penalty_max_amount` is configured and greater than 0, THE Isolir_Usecase SHALL cap the calculated late fee at the `penalty_max_amount`
7. WHEN a late fee is calculated, THE Isolir_Usecase SHALL add a line item with `item_type` = `penalty` to the invoice and update the invoice's `penalty_amount` and `total_amount` fields
8. WHEN a late fee is added, THE Isolir_Usecase SHALL publish an `invoice.penalty_added` event to the Event_Queue for payment link synchronization
9. WHEN a late fee is added, THE Isolir_Usecase SHALL write an invoice audit log entry with action `invoice.penalty_added` and actor `System` with metadata containing the fee amount and calculation method

### Requirement 9: Late Fee Waive API

**User Story:** As a tenant admin, I want to manually waive (remove) a late fee from a specific invoice, so that I can handle special cases where the penalty should not apply.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/invoices/:id/waive-penalty`, THE Billing_API SHALL remove the penalty line item from the invoice and recalculate `penalty_amount` and `total_amount`
2. IF the invoice does not have a penalty line item, THEN THE Billing_API SHALL return HTTP 422 with error code `NO_PENALTY_TO_WAIVE`
3. IF the invoice status is `lunas` or `batal`, THEN THE Billing_API SHALL return HTTP 422 with error code `INVOICE_NOT_EDITABLE`
4. WHEN a penalty is waived, THE Billing_API SHALL write an invoice audit log entry with action `invoice.penalty_waived` and the actor set to the authenticated admin
5. WHEN a penalty is waived, THE Billing_API SHALL publish an `invoice.penalty_added` event to the Event_Queue to trigger payment link amount synchronization
6. IF the invoice ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `INVOICE_NOT_FOUND`

### Requirement 10: Admin Reactivation of Suspended Customers

**User Story:** As a tenant admin, I want to manually reactivate suspended customers after they pay all outstanding invoices, so that customers who resolve their arrears can regain service.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/:id/reactivate`, THE Billing_API SHALL transition the customer status from `suspend` to `aktif` using the existing state machine
2. THE Billing_API SHALL require that all invoices with status `belum_bayar`, `terlambat`, or `bayar_sebagian` for the customer are resolved (status `lunas` or `batal`) before allowing reactivation
3. IF the customer has outstanding invoices, THEN THE Billing_API SHALL return HTTP 422 with error code `OUTSTANDING_INVOICES_EXIST` and include the count and total amount of outstanding invoices
4. IF the customer status is not `suspend`, THEN THE Billing_API SHALL return HTTP 422 with error code `INVALID_STATUS_TRANSITION` and include the current status and allowed transitions
5. WHEN a customer is reactivated, THE Billing_API SHALL publish a `customer.un_isolir` event to the Event_Queue and create a pending_sync record with operation_type `un_isolir`
6. WHEN a customer is reactivated, THE Billing_API SHALL write an invoice audit log entry with action `customer.reactivated` and the actor set to the authenticated admin
7. WHEN a customer is reactivated, THE Billing_API SHALL publish a `notification.reactivated` event to the Event_Queue for downstream notification delivery

### Requirement 11: Isolir Event Payloads

**User Story:** As a developer, I want well-defined event payloads for isolir operations, so that downstream modules (MikroTik, notification) can consume events with all necessary information.

#### Acceptance Criteria

1. THE Billing_API SHALL define a `CustomerIsolirPayload` struct containing: `customer_id`, `tenant_id`, `customer_name`, `router_id`, `pppoe_username`, `connection_method`, `reason` (string describing why isolir was triggered), and `overdue_days` (integer)
2. THE Billing_API SHALL define a `CustomerUnIsolirPayload` struct containing: `customer_id`, `tenant_id`, `customer_name`, `router_id`, `pppoe_username`, `connection_method`, and `trigger` (string: "payment_received" or "admin_manual")
3. THE Billing_API SHALL define a `CustomerSuspendPayload` struct containing: `customer_id`, `tenant_id`, `customer_name`, `router_id`, `pppoe_username`, `connection_method`, and `overdue_days` (integer)
4. THE Billing_API SHALL define a `PenaltyAddedPayload` struct containing: `invoice_id`, `tenant_id`, `customer_id`, `penalty_amount`, `penalty_type`, and `invoice_number`
5. FOR ALL event payloads, the `tenant_id` and `customer_id` fields SHALL be non-empty strings (payload completeness property)

### Requirement 12: Isolir Cron Timezone Handling

**User Story:** As a tenant admin, I want isolir cron jobs to respect my configured timezone, so that isolation and suspension happen at the expected local time regardless of server timezone.

#### Acceptance Criteria

1. WHEN the auto-isolir cron job calculates overdue days, THE Isolir_Usecase SHALL use the tenant's configured `timezone` from Billing_Settings to determine the current date
2. WHEN the suspend cron job calculates overdue days, THE Isolir_Usecase SHALL use the tenant's configured `timezone` from Billing_Settings to determine the current date
3. THE Billing_Settings `timezone` field SHALL accept values `Asia/Jakarta` (WIB), `Asia/Makassar` (WITA), or `Asia/Jayapura` (WIT)
4. IF the tenant's timezone is not configured or invalid, THEN THE Isolir_Usecase SHALL default to `Asia/Jakarta` (WIB/UTC+7)

### Requirement 13: Isolir Dashboard Summary API

**User Story:** As a tenant admin, I want a summary of isolir-related statistics, so that I can monitor the overall isolation status of my customer base from the dashboard.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/isolir/summary`, THE Billing_API SHALL return a summary object containing: total customers in `isolir` status, total customers in `suspend` status, total pending_sync records with status `pending` or `failed`, total revenue at risk (sum of outstanding invoice amounts for isolated and suspended customers)
2. THE Billing_API SHALL scope all counts to the authenticated tenant
3. THE Billing_API SHALL return monetary values as BIGINT (Rupiah) consistent with the rest of the system
