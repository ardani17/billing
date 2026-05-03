# Requirements Document

## Introduction

This spec defines the Invoice Generation module for ISPBoss — a SaaS billing platform for ISPs. The module implements invoice database schema (invoices, invoice_items, invoice_payments tables), automatic invoice generation via daily cron job (H-{generate_days} before due date), manual invoice creation, invoice numbering with auto-increment sequence per month per tenant, invoice status lifecycle (Belum Bayar, Terlambat, Lunas, Bayar Sebagian, Batal, Prorate), invoice list and detail APIs with filtering/pagination, invoice PDF generation with tenant branding, prorate calculation for new customers and mid-cycle package changes, late fee calculation (configurable: fixed/percentage/daily), tax/PPN calculation (configurable percentage), credit balance management (overpayment becomes credit applied to next invoice), recurring items per customer auto-added to invoices, invoice bulk actions (reminder, download PDF, cancel, export CSV), invoice audit trail (append-only), invoice edit (only if Belum Bayar), invoice cancel with confirmation, prepaid billing (3/6/12 months upfront with bundling discount), and credit note/debit note for formal adjustments. All data is tenant-scoped via RLS. The module publishes events for downstream consumption (notification, payment gateway) and uses asynq for background jobs. Payment recording and payment gateway integration are handled in separate specs.

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles invoice, billing, customer, and auth operations
- **Invoice**: A billing document issued to a customer for a specific period, stored in the `invoices` table
- **Invoice_Item**: A line item within an invoice (monthly charge, prorate, penalty, tax, custom item), stored in the `invoice_items` table
- **Invoice_Payment**: A payment record against an invoice, stored in the `invoice_payments` table
- **Invoice_Status**: One of `belum_bayar`, `terlambat`, `lunas`, `bayar_sebagian`, `batal`, or `prorate`
- **Invoice_Number**: A unique identifier in format `{PREFIX}-{YYYY}-{MM}-{SEQ}`, auto-incremented per month per tenant
- **Billing_Settings**: Tenant-level configuration for billing behavior (generate_days, grace_period, tax_rate, penalty settings, invoice prefix, timezone), stored in the `billing_settings` table
- **Generate_Days**: The number of days before a customer's due date when the invoice is auto-generated (default 5)
- **Grace_Period**: The number of days after due date before isolir is triggered (default 7, handled by separate isolir spec)
- **Due_Date**: The day of month (1-28) when a customer's invoice is due, stored on the customer record
- **Prorate**: A partial-month charge calculated using a fixed 30-day month, applied for new customers or mid-cycle package changes
- **Late_Fee**: An optional penalty charged for overdue invoices, configurable as fixed amount, percentage, or daily rate
- **Tax_Rate**: An optional tax percentage (default 11% PPN) applied to the subtotal before penalty
- **Credit_Balance**: Overpayment amount stored on the customer record, automatically applied to the next invoice
- **Recurring_Item**: A per-customer recurring charge (ONT rental, IP public, etc.) auto-added to monthly invoices, stored in the `customer_recurring_items` table
- **Credit_Note**: A formal document for invoice adjustments/refunds, format `CN-{YYYY}-{MM}-{SEQ}`
- **Debit_Note**: A formal document for additional charges outside regular invoices, format `DN-{YYYY}-{MM}-{SEQ}`
- **Invoice_Audit_Log**: An append-only lifecycle log for invoice operations, stored in the `invoice_audit_logs` table
- **Cron_Job**: A daily background job (via asynq scheduler) that generates invoices and updates overdue statuses
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Customer**: An ISP subscriber with a package, due date, and billing cycle, stored in the `customers` table
- **Package**: An internet service plan with monthly_price, stored in the `packages` table
- **Prepaid_Invoice**: A combined invoice for multiple months paid upfront (3/6/12 months) with optional bundling discount
- **Validator**: The `go-playground/validator` library used for input validation in the Billing_API
- **RLS**: PostgreSQL Row Level Security — a database-level safety net ensuring tenant data isolation
- **Event_Queue**: Redis-based message queue (asynq) for inter-service communication and background jobs
- **Prorate_Rounding**: Charges rounded up to nearest Rp 500; credits rounded down to nearest Rp 500

## Requirements

### Requirement 1: Invoice Database Schema

**User Story:** As a tenant admin, I want a comprehensive invoice database schema, so that invoices, line items, and payment records are stored with proper multi-tenant isolation and support the full invoice lifecycle.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `invoices` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `customer_id` (UUID FK NOT NULL REFERENCES customers(id)), `invoice_number` (VARCHAR NOT NULL), `period_month` (INTEGER NOT NULL, 1-12), `period_year` (INTEGER NOT NULL), `due_date` (DATE NOT NULL), `subtotal` (BIGINT NOT NULL DEFAULT 0), `tax_amount` (BIGINT NOT NULL DEFAULT 0), `penalty_amount` (BIGINT NOT NULL DEFAULT 0), `discount_amount` (BIGINT NOT NULL DEFAULT 0), `credit_applied` (BIGINT NOT NULL DEFAULT 0), `total_amount` (BIGINT NOT NULL DEFAULT 0), `paid_amount` (BIGINT NOT NULL DEFAULT 0), `status` (VARCHAR NOT NULL DEFAULT 'belum_bayar'), `notes` (TEXT), `is_prepaid` (BOOLEAN NOT NULL DEFAULT FALSE), `prepaid_months` (INTEGER), `version` (INTEGER NOT NULL DEFAULT 1), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `invoices` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, invoice_number)` to prevent duplicate invoice numbers within a tenant
4. THE Billing_API migration SHALL create composite indexes on `(tenant_id, status)`, `(tenant_id, customer_id)`, `(tenant_id, period_year, period_month)`, and `(tenant_id, due_date, status)` for query performance
5. THE Billing_API migration SHALL enforce a CHECK constraint on `status` to accept only `belum_bayar`, `terlambat`, `lunas`, `bayar_sebagian`, `batal`, or `prorate`
6. THE Billing_API migration SHALL enforce CHECK constraints on monetary columns (`subtotal`, `tax_amount`, `penalty_amount`, `discount_amount`, `credit_applied`, `total_amount`, `paid_amount`) to accept only values greater than or equal to 0

### Requirement 2: Invoice Items Schema

**User Story:** As a tenant admin, I want invoice line items stored separately, so that each charge (monthly fee, prorate, penalty, tax, custom items, recurring items) is individually tracked and auditable.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `invoice_items` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `invoice_id` (UUID FK NOT NULL REFERENCES invoices(id)), `item_type` (VARCHAR NOT NULL), `description` (VARCHAR NOT NULL), `quantity` (INTEGER NOT NULL DEFAULT 1), `unit_price` (BIGINT NOT NULL), `amount` (BIGINT NOT NULL), `sort_order` (INTEGER NOT NULL DEFAULT 0), `metadata` (JSONB), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `invoice_items` table with tenant isolation policies
3. THE Billing_API migration SHALL enforce a CHECK constraint on `item_type` to accept only `monthly`, `installation`, `prorate_charge`, `prorate_credit`, `penalty`, `tax`, `discount`, `recurring`, `custom`, or `credit_applied`
4. THE Billing_API migration SHALL create a composite index on `(tenant_id, invoice_id)` for query performance
5. FOR ALL invoice items, the `amount` SHALL equal `quantity` multiplied by `unit_price` (amount consistency property)

### Requirement 3: Invoice Payments Schema

**User Story:** As a tenant admin, I want payment records linked to invoices, so that partial payments, overpayments, and payment history are tracked per invoice.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `invoice_payments` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `invoice_id` (UUID FK NOT NULL REFERENCES invoices(id)), `amount` (BIGINT NOT NULL), `payment_method` (VARCHAR NOT NULL), `payment_date` (DATE NOT NULL), `reference_number` (VARCHAR), `notes` (TEXT), `recorded_by_id` (UUID NOT NULL), `recorded_by_name` (VARCHAR NOT NULL), `voided` (BOOLEAN NOT NULL DEFAULT FALSE), `voided_at` (TIMESTAMPTZ), `voided_by` (VARCHAR), `void_reason` (TEXT), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `invoice_payments` table with tenant isolation policies
3. THE Billing_API migration SHALL enforce a CHECK constraint on `payment_method` to accept only `tunai`, `transfer`, `xendit`, `midtrans`, or `lainnya`
4. THE Billing_API migration SHALL create composite indexes on `(tenant_id, invoice_id)` and `(tenant_id, payment_date)` for query performance
5. THE Billing_API migration SHALL enforce a CHECK constraint on `amount` to accept only values greater than 0

### Requirement 4: Billing Settings Schema

**User Story:** As a tenant admin, I want tenant-level billing configuration stored in the database, so that each tenant can customize invoice generation behavior, tax, penalties, and numbering.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `billing_settings` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL UNIQUE), `generate_days` (INTEGER NOT NULL DEFAULT 5), `grace_period_days` (INTEGER NOT NULL DEFAULT 7), `suspend_days` (INTEGER NOT NULL DEFAULT 30), `tax_enabled` (BOOLEAN NOT NULL DEFAULT FALSE), `tax_rate` (DECIMAL(5,2) NOT NULL DEFAULT 11.00), `penalty_enabled` (BOOLEAN NOT NULL DEFAULT FALSE), `penalty_type` (VARCHAR NOT NULL DEFAULT 'fixed'), `penalty_amount` (BIGINT NOT NULL DEFAULT 0), `penalty_percentage` (DECIMAL(5,2) NOT NULL DEFAULT 0), `penalty_daily_amount` (BIGINT NOT NULL DEFAULT 0), `penalty_max_amount` (BIGINT NOT NULL DEFAULT 0), `invoice_prefix` (VARCHAR NOT NULL DEFAULT 'INV'), `new_customer_billing` (VARCHAR NOT NULL DEFAULT 'prorate'), `timezone` (VARCHAR NOT NULL DEFAULT 'Asia/Jakarta'), `auto_isolir` (BOOLEAN NOT NULL DEFAULT TRUE), `auto_open_isolir` (BOOLEAN NOT NULL DEFAULT TRUE), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `billing_settings` table with tenant isolation policies
3. THE Billing_API migration SHALL enforce a CHECK constraint on `penalty_type` to accept only `fixed`, `percentage`, or `daily`
4. THE Billing_API migration SHALL enforce a CHECK constraint on `new_customer_billing` to accept only `prorate` or `full_month`
5. THE Billing_API migration SHALL enforce a CHECK constraint on `generate_days` to accept only values between 1 and 14 inclusive

### Requirement 5: Customer Recurring Items Schema

**User Story:** As a tenant admin, I want per-customer recurring charges stored in a dedicated table, so that items like ONT rental and IP public fees are automatically included in monthly invoices.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `customer_recurring_items` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `customer_id` (UUID FK NOT NULL REFERENCES customers(id)), `description` (VARCHAR NOT NULL), `amount` (BIGINT NOT NULL), `is_active` (BOOLEAN NOT NULL DEFAULT TRUE), `start_date` (DATE NOT NULL), `end_date` (DATE), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `customer_recurring_items` table with tenant isolation policies
3. THE Billing_API migration SHALL create a composite index on `(tenant_id, customer_id, is_active)` for query performance
4. THE Billing_API migration SHALL enforce a CHECK constraint on `amount` to accept only values greater than 0

### Requirement 6: Invoice Audit Log Schema

**User Story:** As a tenant admin, I want an append-only lifecycle log for every invoice operation, so that I can trace the full history of each invoice for financial reconciliation and auditing.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `invoice_audit_logs` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `invoice_id` (UUID FK NOT NULL REFERENCES invoices(id)), `action` (VARCHAR NOT NULL), `actor_id` (VARCHAR NOT NULL), `actor_name` (VARCHAR NOT NULL), `metadata` (JSONB), `created_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enable Row Level Security on the `invoice_audit_logs` table with tenant isolation policies
3. THE Billing_API migration SHALL create a composite index on `(tenant_id, invoice_id)` for query performance
4. THE Billing_API SHALL treat the `invoice_audit_logs` table as append-only — no UPDATE or DELETE operations are permitted on this table

### Requirement 7: Invoice Number Sequence Schema

**User Story:** As a developer, I want a dedicated sequence table for invoice numbering, so that invoice numbers auto-increment per month per tenant without race conditions.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create an `invoice_sequences` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `year` (INTEGER NOT NULL), `month` (INTEGER NOT NULL), `last_seq` (INTEGER NOT NULL DEFAULT 0), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, year, month)` to ensure one sequence per month per tenant
3. WHEN a new invoice number is needed, THE Billing_API SHALL atomically increment `last_seq` using SELECT FOR UPDATE and return the new sequence value
4. THE Billing_API SHALL format the invoice number as `{prefix}-{YYYY}-{MM}-{SEQ}` where SEQ is zero-padded to 3 digits minimum (e.g., INV-2026-04-001), expanding automatically for sequences above 999


### Requirement 8: Auto-Generate Invoice Cron Job

**User Story:** As a tenant admin, I want invoices automatically generated for customers whose due date is approaching, so that customers receive their bills on time without manual intervention.

#### Acceptance Criteria

1. THE Billing_API SHALL run a daily cron job (via asynq scheduler) that scans all tenants and generates invoices for eligible customers
2. WHEN the cron job runs, THE Billing_API SHALL identify customers where the current date equals (due_date minus generate_days) for the upcoming period, the customer status is `aktif`, and no invoice exists for that customer and period
3. WHEN an eligible customer is found, THE Billing_API SHALL create an invoice with status `belum_bayar` containing the monthly package charge as a line item
4. WHEN an eligible customer has active recurring items, THE Billing_API SHALL include each active recurring item as a separate line item in the generated invoice
5. WHEN the tenant has tax enabled, THE Billing_API SHALL calculate tax as `subtotal * tax_rate / 100` and add a tax line item to the invoice
6. WHEN the customer has a positive credit_balance, THE Billing_API SHALL apply the credit (up to the total amount) as a negative line item and reduce the customer's credit_balance accordingly
7. THE Billing_API SHALL be idempotent — running the cron job multiple times for the same day SHALL NOT generate duplicate invoices for the same customer and period
8. WHEN an invoice is successfully generated, THE Billing_API SHALL write an invoice audit log entry with action `invoice.generated` and actor set to `System`
9. WHEN an invoice is successfully generated, THE Billing_API SHALL publish an `invoice.created` event to the Event_Queue for downstream notification services

### Requirement 9: Invoice Status Lifecycle

**User Story:** As a developer, I want the invoice status lifecycle enforced at the domain level, so that invalid transitions are impossible regardless of the caller.

#### Acceptance Criteria

1. THE Billing_API domain layer SHALL define the valid invoice status transitions as: `belum_bayar` -> [`terlambat`, `lunas`, `bayar_sebagian`, `batal`], `terlambat` -> [`lunas`, `bayar_sebagian`, `batal`], `bayar_sebagian` -> [`lunas`, `batal`], `prorate` -> [`lunas`, `bayar_sebagian`, `batal`], `lunas` -> [] (terminal), `batal` -> [] (terminal)
2. WHEN an invalid invoice status transition is attempted, THE Billing_API SHALL return HTTP 422 with error code `INVALID_INVOICE_STATUS_TRANSITION` and include the current status and the list of allowed target statuses
3. FOR ALL valid Invoice_Status values and FOR ALL valid transitions, applying a transition and then checking the resulting status SHALL yield the expected target status (state machine determinism property)
4. WHEN any invoice status transition occurs, THE Billing_API SHALL write an invoice audit log entry with the action describing the transition and the actor who triggered it

### Requirement 10: Overdue Invoice Status Update

**User Story:** As a tenant admin, I want invoices automatically marked as overdue when past their due date, so that the system accurately reflects payment status without manual intervention.

#### Acceptance Criteria

1. THE Billing_API SHALL run a daily cron job that scans for invoices with status `belum_bayar` where the `due_date` is before the current date
2. WHEN an overdue invoice is found, THE Billing_API SHALL transition the invoice status from `belum_bayar` to `terlambat`
3. WHEN an invoice transitions to `terlambat`, THE Billing_API SHALL write an invoice audit log entry with action `invoice.overdue` and actor set to `System`
4. WHEN an invoice transitions to `terlambat`, THE Billing_API SHALL publish an `invoice.overdue` event to the Event_Queue for downstream notification services

### Requirement 11: Invoice Manual Creation API

**User Story:** As a tenant admin, I want to create custom invoices with arbitrary line items, so that I can bill customers for non-standard charges like installation fees, equipment, or custom services.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/invoices` with valid data, THE Billing_API SHALL create a new invoice with the specified line items and return the created invoice with HTTP 201
2. THE Validator SHALL require the following fields: `customer_id` (UUID, must reference an existing active customer), `due_date` (date, must be today or in the future), `items` (array of at least 1 item, each with `description` (required, max 500 chars), `quantity` (positive integer), `unit_price` (positive integer in Rupiah))
3. THE Validator SHALL accept the following optional fields: `notes` (max 1000 characters), `apply_tax` (boolean, default follows tenant setting), `apply_credit` (boolean, default true)
4. THE Billing_API SHALL auto-generate the invoice number using the sequence for the invoice's period month and year
5. WHEN `apply_tax` is true and the tenant has tax enabled, THE Billing_API SHALL calculate and add a tax line item
6. WHEN `apply_credit` is true and the customer has a positive credit_balance, THE Billing_API SHALL apply credit up to the total amount
7. IF the `customer_id` does not reference an existing customer or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`
8. WHEN a manual invoice is successfully created, THE Billing_API SHALL write an invoice audit log entry with action `invoice.created_manual`

### Requirement 12: Invoice List API

**User Story:** As a tenant admin, I want to list invoices with filtering by status, period, package, and area, with search and pagination, so that I can efficiently manage and monitor all invoices.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/invoices`, THE Billing_API SHALL return a paginated list of invoices for the authenticated tenant
2. THE Billing_API SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `status` query parameter is provided, THE Billing_API SHALL filter invoices by the specified status
4. WHEN `period_month` and `period_year` query parameters are provided, THE Billing_API SHALL filter invoices by the specified billing period
5. WHEN a `package_id` query parameter is provided, THE Billing_API SHALL filter invoices by customers subscribed to the specified package
6. WHEN an `area_id` query parameter is provided, THE Billing_API SHALL filter invoices by customers in the specified area
7. WHEN a `search` query parameter is provided, THE Billing_API SHALL filter invoices whose `invoice_number` or customer `name` or customer `customer_id_seq` contains the search term (case-insensitive)
8. THE Billing_API SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
9. THE Billing_API SHALL include the customer `name`, `customer_id_seq`, and `package_name` in each invoice record in the list response

### Requirement 13: Invoice Detail API

**User Story:** As a tenant admin, I want to view a full invoice detail including line items, payment history, and audit logs, so that I can understand the complete state and history of any invoice.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/invoices/:id`, THE Billing_API SHALL return the full invoice record including all line items ordered by `sort_order`
2. THE Billing_API SHALL include the customer details (name, customer_id_seq, phone, address) in the invoice detail response
3. THE Billing_API SHALL include the list of payments (non-voided) with amount, method, date, and recorded_by_name
4. WHEN a GET request is made to `/v1/invoices/:id` with `include=audit_logs`, THE Billing_API SHALL include the invoice's audit log entries in the response
5. IF the invoice ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `INVOICE_NOT_FOUND`

### Requirement 14: Invoice Edit API

**User Story:** As a tenant admin, I want to edit an invoice's line items and due date, so that I can correct mistakes before the customer pays.

#### Acceptance Criteria

1. WHEN a PUT request is made to `/v1/invoices/:id` with valid data, THE Billing_API SHALL update the invoice and return the updated invoice
2. THE Billing_API SHALL only allow editing invoices with status `belum_bayar`
3. IF the invoice status is not `belum_bayar`, THEN THE Billing_API SHALL return HTTP 422 with error code `INVOICE_NOT_EDITABLE` and a message indicating only unpaid invoices can be edited
4. THE Validator SHALL accept the following fields: `due_date` (date, must be today or in the future), `items` (array of items with same validation as creation), `notes` (max 1000 characters)
5. WHEN an invoice is edited, THE Billing_API SHALL recalculate subtotal, tax, credit_applied, and total_amount based on the new items
6. WHEN an invoice is successfully edited, THE Billing_API SHALL increment the `version` field and write an invoice audit log entry with action `invoice.edited` including the old and new values
7. IF the invoice ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `INVOICE_NOT_FOUND`

### Requirement 15: Invoice Cancel API

**User Story:** As a tenant admin, I want to cancel an invoice with confirmation, so that incorrect or unnecessary invoices can be voided with a clear audit trail.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/invoices/:id/cancel` with a valid `confirmation_number` matching the invoice number, THE Billing_API SHALL transition the invoice status to `batal`
2. IF the `confirmation_number` does not match the invoice's `invoice_number` (case-sensitive), THEN THE Billing_API SHALL return HTTP 400 with error code `CONFIRMATION_MISMATCH`
3. IF the invoice status is `lunas` or `batal`, THEN THE Billing_API SHALL return HTTP 422 with error code `INVOICE_NOT_CANCELLABLE`
4. WHEN an invoice with `credit_applied` greater than 0 is cancelled, THE Billing_API SHALL restore the credit amount back to the customer's `credit_balance`
5. THE Validator SHALL require a `reason` field (min 5, max 500 characters) explaining why the invoice is being cancelled
6. WHEN an invoice is successfully cancelled, THE Billing_API SHALL write an invoice audit log entry with action `invoice.cancelled` including the reason
7. WHEN an invoice is successfully cancelled, THE Billing_API SHALL publish an `invoice.cancelled` event to the Event_Queue

### Requirement 16: Invoice PDF Generation

**User Story:** As a tenant admin, I want to generate a branded PDF for any invoice, so that I can provide professional billing documents to customers for their records.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/invoices/:id/pdf`, THE Billing_API SHALL generate and return a PDF document for the specified invoice
2. THE Billing_API SHALL include the following in the PDF: tenant name, tenant address, tenant phone, invoice number, invoice date, due date, status, customer name, customer ID, customer address, customer phone, all line items with description/quantity/unit_price/amount, subtotal, tax amount (if applicable), penalty amount (if applicable), discount amount (if applicable), credit applied (if applicable), total amount, and payment history
3. THE Billing_API SHALL use the tenant's configured invoice prefix and branding information from the tenant record
4. THE Billing_API SHALL generate the PDF using the maroto or gofpdf library in Go
5. IF the invoice ID does not exist or belongs to a different tenant, THEN THE Billing_API SHALL return HTTP 404 with error code `INVOICE_NOT_FOUND`

### Requirement 17: Prorate Calculation for New Customers

**User Story:** As a tenant admin, I want new customers billed proportionally for their first partial month, so that customers are charged fairly based on their activation date.

#### Acceptance Criteria

1. WHEN a new customer is activated mid-cycle and the tenant's `new_customer_billing` setting is `prorate`, THE Billing_API SHALL calculate the first invoice as: `monthly_price * (remaining_days / 30)` where remaining_days is the number of days from activation_date to the next due_date
2. THE Billing_API SHALL always use 30 as the fixed days-per-month divisor for prorate calculations regardless of the actual month length
3. WHEN a prorate charge results in a fractional amount, THE Billing_API SHALL round up to the nearest Rp 500
4. THE Billing_API SHALL add the prorate charge as a line item with `item_type` = `prorate_charge` and a description indicating the partial period
5. WHEN the tenant's `new_customer_billing` setting is `full_month`, THE Billing_API SHALL charge the full monthly_price for the first invoice regardless of activation date

### Requirement 18: Prorate Calculation for Package Changes

**User Story:** As a tenant admin, I want mid-cycle package upgrades and downgrades to generate correct prorate invoices, so that customers are charged or credited fairly for the remaining days in their billing cycle.

#### Acceptance Criteria

1. WHEN a customer upgrades their package mid-cycle, THE Billing_API SHALL generate a prorate invoice with the calculation: `(new_price - old_price) * (remaining_days / 30)` where remaining_days is the number of days from the change date to the next due_date
2. WHEN a customer downgrades their package mid-cycle, THE Billing_API SHALL calculate the credit as: `(old_price - new_price) * (remaining_days / 30)` and add the credit to the customer's `credit_balance`
3. THE Billing_API SHALL always use 30 as the fixed days-per-month divisor for prorate calculations
4. WHEN a prorate charge results in a fractional amount, THE Billing_API SHALL round up to the nearest Rp 500
5. WHEN a prorate credit results in a fractional amount, THE Billing_API SHALL round down to the nearest Rp 500
6. THE Billing_API SHALL create the prorate invoice with status `prorate` and `item_type` = `prorate_charge` for upgrades
7. FOR ALL prorate calculations, the result SHALL be non-negative for upgrades and the credit SHALL be non-negative for downgrades (prorate sign correctness property)


### Requirement 19: Late Fee Calculation

**User Story:** As a tenant admin, I want late fees automatically calculated when payments are recorded for overdue invoices, so that penalty policies are consistently enforced according to tenant configuration.

#### Acceptance Criteria

1. WHILE the tenant's `penalty_enabled` setting is true, THE Billing_API SHALL calculate a late fee when a payment is recorded for an invoice with status `terlambat`
2. WHEN `penalty_type` is `fixed`, THE Billing_API SHALL apply the configured `penalty_amount` as the late fee
3. WHEN `penalty_type` is `percentage`, THE Billing_API SHALL calculate the late fee as `subtotal * penalty_percentage / 100`
4. WHEN `penalty_type` is `daily`, THE Billing_API SHALL calculate the late fee as `penalty_daily_amount * days_overdue` where days_overdue is the number of days between the due_date and the payment_date
5. WHEN a `penalty_max_amount` is configured and greater than 0, THE Billing_API SHALL cap the calculated late fee at the `penalty_max_amount`
6. THE Billing_API SHALL add the late fee as a line item with `item_type` = `penalty` to the invoice when the payment is recorded
7. THE Billing_API SHALL recalculate the invoice `total_amount` after adding the penalty

### Requirement 20: Tax/PPN Calculation

**User Story:** As a tenant admin, I want tax automatically calculated on invoices when enabled, so that invoices comply with Indonesian tax regulations.

#### Acceptance Criteria

1. WHILE the tenant's `tax_enabled` setting is true, THE Billing_API SHALL calculate tax for every generated invoice
2. THE Billing_API SHALL calculate tax as: `subtotal * tax_rate / 100` where subtotal is the sum of all non-tax, non-penalty line items
3. THE Billing_API SHALL add the tax as a line item with `item_type` = `tax` and description including the tax rate (e.g., "PPN 11%")
4. THE Billing_API SHALL calculate tax from the subtotal before penalty (penalty is not taxed)
5. WHEN the tax calculation results in a fractional amount, THE Billing_API SHALL round to the nearest Rupiah (standard rounding)

### Requirement 21: Credit Balance Management

**User Story:** As a tenant admin, I want overpayments automatically stored as credit and applied to the next invoice, so that customers are not overcharged and the billing system handles excess payments gracefully.

#### Acceptance Criteria

1. WHEN a payment amount exceeds the invoice's remaining balance (total_amount minus paid_amount), THE Billing_API SHALL store the excess as credit on the customer's `credit_balance` field
2. WHEN generating a new invoice for a customer with a positive `credit_balance`, THE Billing_API SHALL apply the credit (up to the invoice total) as a line item with `item_type` = `credit_applied` and a negative amount
3. WHEN credit is applied to an invoice, THE Billing_API SHALL atomically reduce the customer's `credit_balance` by the applied amount
4. WHEN an invoice with applied credit is cancelled, THE Billing_API SHALL restore the credit amount back to the customer's `credit_balance`
5. FOR ALL credit operations, the customer's `credit_balance` SHALL remain greater than or equal to 0 (non-negative credit invariant)

### Requirement 22: Recurring Items Management API

**User Story:** As a tenant admin, I want to manage per-customer recurring charges, so that items like ONT rental and IP public fees are automatically included in monthly invoices.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/customers/:id/recurring-items` with valid data, THE Billing_API SHALL create a new recurring item for the customer and return it with HTTP 201
2. THE Validator SHALL require the following fields: `description` (min 3, max 255 characters), `amount` (positive integer in Rupiah), `start_date` (date, must be today or in the future)
3. THE Validator SHALL accept the following optional fields: `end_date` (date, must be after start_date)
4. WHEN a GET request is made to `/v1/customers/:id/recurring-items`, THE Billing_API SHALL return all recurring items for the customer (both active and inactive)
5. WHEN a PUT request is made to `/v1/customers/:id/recurring-items/:item_id`, THE Billing_API SHALL update the recurring item
6. WHEN a DELETE request is made to `/v1/customers/:id/recurring-items/:item_id`, THE Billing_API SHALL deactivate the recurring item by setting `is_active` to false (soft delete)
7. WHEN generating an invoice, THE Billing_API SHALL include all active recurring items where `start_date` is on or before the invoice period and `end_date` is null or after the invoice period

### Requirement 23: Invoice Bulk Actions API

**User Story:** As a tenant admin, I want to perform bulk actions on invoices (send reminders, download PDFs, cancel, export CSV), so that I can efficiently manage large numbers of invoices.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/invoices/bulk/reminder` with an array of invoice IDs, THE Billing_API SHALL publish a reminder event for each eligible invoice (status `belum_bayar` or `terlambat`) and return a summary of successes and failures
2. WHEN a POST request is made to `/v1/invoices/bulk/cancel` with an array of invoice IDs and a `reason`, THE Billing_API SHALL cancel each eligible invoice (status `belum_bayar` or `terlambat`) and return a summary of successes and failures
3. WHEN a POST request is made to `/v1/invoices/bulk/pdf` with an array of invoice IDs, THE Billing_API SHALL generate a ZIP file containing individual PDF files for each invoice and return it for download
4. WHEN a GET request is made to `/v1/invoices/export` with optional filter parameters, THE Billing_API SHALL generate a CSV file containing the filtered invoice list with columns: invoice_number, customer_name, customer_id_seq, period, due_date, subtotal, tax, penalty, total, paid, status
5. THE Billing_API SHALL return a response containing `total`, `success_count`, `failure_count`, and an array of `failures` with `invoice_id` and `reason` for each failed operation in bulk reminder and bulk cancel actions
6. WHEN bulk cancel is performed, THE Billing_API SHALL write an invoice audit log entry for each individual invoice affected

### Requirement 24: Invoice Audit Trail

**User Story:** As a tenant admin, I want a complete append-only audit trail of all invoice operations, so that I can trace every change for financial reconciliation and compliance.

#### Acceptance Criteria

1. THE Billing_API SHALL write an audit log entry for every invoice operation including: `invoice.generated`, `invoice.created_manual`, `invoice.edited`, `invoice.cancelled`, `invoice.overdue`, `invoice.payment_recorded`, `invoice.status_changed`, `invoice.pdf_generated`, `invoice.reminder_sent`
2. THE Billing_API SHALL include the following in each audit log entry: `invoice_id`, `action`, `actor_id`, `actor_name`, `metadata` (JSONB with operation-specific details), and `created_at`
3. THE Billing_API SHALL set actor to `System` for automated operations (cron jobs, auto-status changes) and to the authenticated user for manual operations
4. THE Billing_API SHALL never update or delete existing audit log entries (append-only invariant)
5. WHEN a GET request is made to `/v1/invoices/:id/audit-logs`, THE Billing_API SHALL return all audit log entries for the specified invoice ordered by `created_at` ascending

### Requirement 25: Prepaid Billing (Multi-Month Invoice)

**User Story:** As a tenant admin, I want to create a combined invoice for customers paying multiple months upfront, so that customers can benefit from bundling discounts and the system skips auto-generation for prepaid periods.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/invoices/prepaid` with valid data, THE Billing_API SHALL create a single combined invoice covering the specified number of months
2. THE Validator SHALL require the following fields: `customer_id` (UUID, must reference an existing active customer), `months` (integer, one of 3, 6, or 12), `start_period_month` (integer 1-12), `start_period_year` (integer)
3. THE Validator SHALL accept an optional `discount_months` field (integer, default 0) representing the number of free months included in the bundle
4. THE Billing_API SHALL create line items for each month in the prepaid period (e.g., "Paket Pro 50M (Apr 2026)", "Paket Pro 50M (Mei 2026)", etc.)
5. WHEN `discount_months` is greater than 0, THE Billing_API SHALL add a discount line item with `item_type` = `discount` and amount equal to `monthly_price * discount_months`
6. THE Billing_API SHALL set `is_prepaid` to true and `prepaid_months` to the total months covered on the created invoice
7. WHILE a customer has a prepaid invoice covering a future period, THE Billing_API auto-generate cron job SHALL skip invoice generation for that customer and period
8. WHEN a prepaid invoice is successfully created, THE Billing_API SHALL write an invoice audit log entry with action `invoice.created_prepaid`

### Requirement 26: Credit Note API

**User Story:** As a tenant admin, I want to issue credit notes for invoice adjustments and refunds, so that formal financial documents exist for cancellations, overpayment refunds, and service compensation.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/credit-notes` with valid data, THE Billing_API SHALL create a credit note document and return it with HTTP 201
2. THE Validator SHALL require the following fields: `invoice_id` (UUID, must reference an existing invoice), `amount` (positive integer in Rupiah), `reason` (min 5, max 500 characters)
3. THE Billing_API SHALL auto-generate the credit note number in format `CN-{YYYY}-{MM}-{SEQ}` using a separate sequence from invoices
4. THE Validator SHALL accept an optional `apply_to_credit` field (boolean, default true) — when true, the amount is added to the customer's `credit_balance`; when false, it is recorded as a manual refund
5. WHEN `apply_to_credit` is true, THE Billing_API SHALL atomically increase the customer's `credit_balance` by the credit note amount
6. WHEN a credit note is created, THE Billing_API SHALL write an invoice audit log entry with action `credit_note.created` on the referenced invoice

### Requirement 27: Debit Note API

**User Story:** As a tenant admin, I want to issue debit notes for additional charges outside regular invoices, so that formal financial documents exist for equipment replacement, reactivation fees, and other ad-hoc charges.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/debit-notes` with valid data, THE Billing_API SHALL create a debit note document and return it with HTTP 201
2. THE Validator SHALL require the following fields: `customer_id` (UUID, must reference an existing customer), `items` (array of at least 1 item with `description` and `amount`), `due_date` (date, must be today or in the future)
3. THE Billing_API SHALL auto-generate the debit note number in format `DN-{YYYY}-{MM}-{SEQ}` using a separate sequence
4. THE Validator SHALL accept an optional `create_invoice` field (boolean, default false) — when true, the Billing_API SHALL also create a corresponding invoice with the debit note items
5. WHEN a debit note is created, THE Billing_API SHALL write an audit log entry with action `debit_note.created`

### Requirement 28: Invoice Summary Statistics API

**User Story:** As a tenant admin, I want summary statistics for invoices (total count, amounts by status), so that I can see a billing overview on the dashboard.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/invoices/summary`, THE Billing_API SHALL return aggregated invoice statistics for the authenticated tenant
2. THE Billing_API SHALL return the following per-status aggregations: count of invoices and sum of total_amount for each status (`belum_bayar`, `terlambat`, `lunas`, `bayar_sebagian`, `batal`, `prorate`)
3. WHEN `period_month` and `period_year` query parameters are provided, THE Billing_API SHALL filter the summary to the specified period
4. THE Billing_API SHALL include a `total` field with the overall count and sum across all statuses (excluding `batal`)


### Requirement 29: Installation Fee on First Invoice

**User Story:** As a tenant admin, I want installation fees automatically included in a new customer's first invoice, so that the one-time setup cost is billed together with the first month's charge without manual intervention.

#### Acceptance Criteria

1. WHEN the auto-generate cron job creates the first invoice for a newly activated customer, THE Billing_API SHALL check if the customer's package has an `installation_fee` greater than 0
2. WHEN the package has a positive `installation_fee`, THE Billing_API SHALL add an installation fee line item with `item_type` = `installation`, description "Biaya Pasang", quantity 1, and `unit_price` equal to the package's `installation_fee`
3. THE Billing_API SHALL only add the installation fee to the **first invoice** for the customer — subsequent monthly invoices SHALL NOT include the installation fee
4. THE Billing_API SHALL determine "first invoice" by checking that no prior invoice exists for the customer in the same tenant (excluding cancelled invoices)
5. WHEN a manual invoice is created for a new customer, THE Billing_API SHALL NOT automatically add the installation fee — the admin can add it manually as a custom line item if needed

### Requirement 30: Invoice Amount Immutability After Generation

**User Story:** As a tenant admin, I want invoice amounts to remain unchanged when package prices are updated, so that already-issued invoices reflect the price at the time of generation and customers are not retroactively charged different amounts.

#### Acceptance Criteria

1. WHEN an invoice is generated (either automatically or manually), THE Billing_API SHALL capture the package's current `monthly_price` at the time of generation and store it as the `unit_price` in the invoice item — subsequent changes to the package price SHALL NOT affect existing invoices
2. WHEN a package price is changed by an admin, THE Billing_API SHALL apply the new price only to invoices generated **after** the price change — all existing invoices (status `belum_bayar`, `terlambat`, `bayar_sebagian`, `prorate`) SHALL retain their original amounts
3. WHEN a customer has a prepaid invoice (`is_prepaid` = true), THE Billing_API SHALL NOT modify the prepaid invoice amounts even if the package price changes during the prepaid period — the price is locked at the time of prepaid invoice creation
4. FOR ALL generated invoices, the `unit_price` stored in invoice items SHALL represent a point-in-time snapshot of the price and SHALL NOT be updated by any subsequent package price change (price snapshot immutability property)
