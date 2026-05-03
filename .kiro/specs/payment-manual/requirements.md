# Requirements Document

## Introduction

This spec defines the Manual Payment Recording module for ISPBoss billing-api. It extends the existing invoice-generation module (which provides basic single-invoice `RecordPayment` functionality) with a dedicated payment list API, quick payment flow, multi-invoice FIFO allocation, pay-all-arrears, receipt/kwitansi generation for thermal printers, void/reversal with time-limited rollback, bulk payment import via CSV, and payment summary statistics. All operations are tenant-scoped via RLS and logged in the append-only `invoice_audit_logs` table.

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles invoice, billing, customer, and auth operations
- **Payment_Module**: The subsystem within Billing_API responsible for payment listing, recording, receipt generation, void/reversal, and bulk import
- **Invoice_Payment**: A payment record against an invoice, stored in the `invoice_payments` table with amount, method, date, and actor information
- **Payment_Method**: One of `tunai` (cash), `transfer` (bank transfer), or `lainnya` (other) for manual payments. Online methods (`xendit`, `midtrans`) are handled in a separate spec
- **Receipt**: A payment receipt document (kwitansi) formatted for thermal printers (58mm/80mm), numbered with format `PAY-{YYYY}-{MM}-{SEQ}`
- **Receipt_Sequence**: A dedicated sequence table for receipt numbering, auto-incremented per month per tenant
- **FIFO_Allocation**: The rule that when a customer has multiple overdue invoices, payment is allocated to the oldest invoice first (First In, First Out)
- **Void**: The act of reversing a payment record within 24 hours of creation, rolling back invoice status and related side effects
- **Bulk_Import**: The process of uploading a CSV file containing multiple payment records for batch processing
- **Payment_Summary**: Aggregated statistics showing today's transactions, monthly totals, and breakdown by payment method
- **Quick_Payment**: A streamlined flow where a cashier searches for a customer, sees open invoices, selects invoices, and records payment in one action
- **Arrears**: All outstanding (unpaid or partially paid) invoices for a customer
- **Credit_Balance**: Overpayment amount stored on the customer record, automatically applied to the next invoice
- **Optimistic_Locking**: Concurrency control via a `version` field on the invoice to prevent double payment processing
- **Invoice_Audit_Log**: An append-only lifecycle log for invoice operations, stored in the `invoice_audit_logs` table
- **Validator**: The `go-playground/validator` library used for input validation in the Billing_API
- **Actor**: The authenticated user performing an operation, identified by `actor_id` and `actor_name`
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Customer**: An ISP subscriber with a package, due date, and billing cycle, stored in the `customers` table
- **Admin**: A user with the `admin` role who has elevated permissions (e.g., void payments)
- **Kasir**: A user with the `kasir` (cashier) role who records payments but cannot void them

## Requirements

### Requirement 1: Payment List API

**User Story:** As a tenant admin or kasir, I want a dedicated endpoint to list all payment records with filtering, search, and pagination, so that I can efficiently review payment history across all customers.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/payments`, THE Payment_Module SHALL return a paginated list of non-voided payment records for the authenticated tenant, ordered by `payment_date` descending then `created_at` descending
2. THE Payment_Module SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
3. WHEN a `payment_method` query parameter is provided, THE Payment_Module SHALL filter payments by the specified method (`tunai`, `transfer`, or `lainnya`)
4. WHEN `date_from` and `date_to` query parameters are provided, THE Payment_Module SHALL filter payments where `payment_date` falls within the specified date range (inclusive)
5. WHEN a `recorded_by` query parameter is provided, THE Payment_Module SHALL filter payments recorded by the specified user ID
6. WHEN a `search` query parameter is provided, THE Payment_Module SHALL filter payments whose associated customer `name`, customer `customer_id_seq`, or invoice `invoice_number` contains the search term (case-insensitive)
7. THE Payment_Module SHALL return pagination metadata including `total`, `page`, `page_size`, and `total_pages` in the response
8. THE Payment_Module SHALL include the customer `name`, customer `customer_id_seq`, invoice `invoice_number`, and `recorded_by_name` in each payment record in the list response
9. WHEN a `include_voided` query parameter is set to `true`, THE Payment_Module SHALL include voided payment records in the results with a `voided` flag set to true

### Requirement 2: Payment Summary Statistics API

**User Story:** As a tenant admin or kasir, I want summary statistics showing today's transactions, monthly totals, and breakdown by payment method, so that I can monitor collection performance at a glance.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/payments/summary`, THE Payment_Module SHALL return aggregated payment statistics for the authenticated tenant
2. THE Payment_Module SHALL return a `today` object containing the count of payments and sum of amounts for the current date (based on tenant timezone)
3. THE Payment_Module SHALL return a `this_month` object containing the count of payments and sum of amounts for the current calendar month
4. THE Payment_Module SHALL return a `by_method` object containing the count and sum of amounts grouped by each payment method (`tunai`, `transfer`, `lainnya`) for the current month
5. THE Payment_Module SHALL exclude voided payments from all summary calculations
6. WHEN `period_month` and `period_year` query parameters are provided, THE Payment_Module SHALL calculate the monthly and by-method statistics for the specified period instead of the current month

### Requirement 3: Quick Payment — Customer Search

**User Story:** As a kasir, I want to search for a customer by name, ID, or phone number and see their open invoices, so that I can quickly record a payment without navigating through multiple pages.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/payments/quick/customers?search={term}`, THE Payment_Module SHALL return a list of customers matching the search term by `name`, `customer_id_seq`, or `phone` (case-insensitive, partial match)
2. THE Payment_Module SHALL limit the search results to a maximum of 10 customers
3. THE Payment_Module SHALL only return customers with status `aktif` or `isolir` (customers who can receive payments)
4. THE Payment_Module SHALL include the following fields for each customer in the response: `id`, `name`, `customer_id_seq`, `phone`, `package_name`, `status`, and `credit_balance`
5. IF the search term is fewer than 2 characters, THEN THE Payment_Module SHALL return HTTP 400 with error code `SEARCH_TERM_TOO_SHORT`

### Requirement 4: Quick Payment — Open Invoices

**User Story:** As a kasir, I want to see all open invoices for a selected customer with outstanding amounts, so that I can select which invoices to pay.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/payments/quick/customers/:customer_id/invoices`, THE Payment_Module SHALL return all invoices for the specified customer with status `belum_bayar`, `terlambat`, or `bayar_sebagian`, ordered by `due_date` ascending (oldest first)
2. THE Payment_Module SHALL include the following fields for each invoice: `id`, `invoice_number`, `period_month`, `period_year`, `total_amount`, `paid_amount`, `remaining_amount` (calculated as `total_amount - paid_amount`), `status`, and `due_date`
3. THE Payment_Module SHALL include a `total_arrears` field in the response representing the sum of all `remaining_amount` values across all open invoices
4. IF the customer ID does not exist or belongs to a different tenant, THEN THE Payment_Module SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`

### Requirement 5: Multi-Invoice Payment with FIFO Allocation

**User Story:** As a kasir, I want to record a single payment that is automatically allocated across multiple invoices starting from the oldest, so that arrears are cleared in the correct order without manual per-invoice entry.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/payments/multi` with a `customer_id` and `amount`, THE Payment_Module SHALL allocate the payment to the customer's open invoices in FIFO order (oldest `due_date` first)
2. THE Validator SHALL require the following fields: `customer_id` (UUID), `amount` (positive integer in Rupiah), `payment_method` (one of `tunai`, `transfer`, `lainnya`), `payment_date` (date format YYYY-MM-DD)
3. THE Validator SHALL accept the following optional fields: `reference_number` (string), `notes` (max 500 characters), `invoice_ids` (array of UUIDs for admin override)
4. WHEN `invoice_ids` is provided, THE Payment_Module SHALL allocate the payment only to the specified invoices in the order provided (admin override of FIFO)
5. WHEN `invoice_ids` is not provided, THE Payment_Module SHALL allocate the payment to open invoices ordered by `due_date` ascending, allocating the full remaining amount to each invoice before moving to the next
6. WHEN the payment fully covers an invoice's remaining amount, THE Payment_Module SHALL transition that invoice's status to `lunas` using Optimistic_Locking
7. WHEN the payment partially covers an invoice's remaining amount, THE Payment_Module SHALL transition that invoice's status to `bayar_sebagian` and update `paid_amount` accordingly
8. WHEN the total payment amount exceeds the sum of all open invoices' remaining amounts, THE Payment_Module SHALL add the excess to the customer's `credit_balance`
9. THE Payment_Module SHALL create a separate `invoice_payments` record for each invoice that receives an allocation, linking the payment to the specific invoice
10. THE Payment_Module SHALL use Optimistic_Locking (version field) on each invoice update to prevent double payment processing
11. THE Payment_Module SHALL write an Invoice_Audit_Log entry with action `invoice.payment_recorded` for each invoice that receives an allocation
12. THE Payment_Module SHALL return a response containing: list of allocations (invoice_id, invoice_number, allocated_amount, new_status), total_allocated, excess_to_credit, and receipt information
13. IF any invoice specified in `invoice_ids` does not belong to the customer or is already `lunas` or `batal`, THEN THE Payment_Module SHALL return HTTP 422 with error code `INVALID_INVOICE_SELECTION` identifying the problematic invoice

### Requirement 6: Pay All Arrears

**User Story:** As a kasir, I want a single action to pay all outstanding invoices for a customer in one transaction, so that customers clearing their full balance can be processed quickly.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/payments/pay-all` with a `customer_id`, THE Payment_Module SHALL calculate the total remaining amount across all open invoices for the customer and record a payment for that exact amount
2. THE Validator SHALL require the following fields: `customer_id` (UUID), `payment_method` (one of `tunai`, `transfer`, `lainnya`), `payment_date` (date format YYYY-MM-DD)
3. THE Validator SHALL accept the following optional fields: `reference_number` (string), `notes` (max 500 characters)
4. THE Payment_Module SHALL allocate the payment across all open invoices in FIFO order, transitioning each invoice to `lunas`
5. THE Payment_Module SHALL return the total amount paid, the number of invoices cleared, and receipt information
6. IF the customer has no open invoices, THEN THE Payment_Module SHALL return HTTP 422 with error code `NO_OPEN_INVOICES`

### Requirement 7: Receipt Sequence Schema

**User Story:** As a developer, I want a dedicated sequence table for receipt numbering, so that receipt numbers auto-increment per month per tenant without race conditions.

#### Acceptance Criteria

1. THE Billing_API migration SHALL create a `receipt_sequences` table with the following columns: `id` (UUID PK), `tenant_id` (UUID FK NOT NULL), `year` (INTEGER NOT NULL), `month` (INTEGER NOT NULL), `last_seq` (INTEGER NOT NULL DEFAULT 0), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. THE Billing_API migration SHALL enforce a unique constraint on `(tenant_id, year, month)` to ensure one sequence per month per tenant
3. WHEN a new receipt number is needed, THE Payment_Module SHALL atomically increment `last_seq` using SELECT FOR UPDATE and return the new sequence value
4. THE Payment_Module SHALL format the receipt number as `PAY-{YYYY}-{MM}-{SEQ}` where SEQ is zero-padded to 4 digits minimum (e.g., PAY-2026-04-0001), expanding automatically for sequences above 9999

### Requirement 8: Receipt Generation

**User Story:** As a kasir, I want a payment receipt (kwitansi) generated after recording a payment, so that I can print a proof of payment for the customer using a thermal printer.

#### Acceptance Criteria

1. WHEN a payment is successfully recorded (via single-invoice, multi-invoice, or pay-all endpoints), THE Payment_Module SHALL generate a receipt with a unique receipt number in format `PAY-{YYYY}-{MM}-{SEQ}`
2. THE Payment_Module SHALL include the following information in the receipt: tenant name, receipt number, payment date and time, customer name, customer ID (customer_id_seq), invoice number(s) covered, total amount paid, payment method, and kasir/actor name
3. WHEN a GET request is made to `/v1/payments/:payment_id/receipt`, THE Payment_Module SHALL return the receipt data in a structured JSON format suitable for thermal printer rendering (58mm/80mm width)
4. THE Payment_Module SHALL include a `receipt_number` field in the response of all payment recording endpoints
5. WHEN a multi-invoice payment is recorded, THE Payment_Module SHALL generate a single receipt covering all invoices in that transaction
6. FOR ALL receipt numbers generated within the same tenant, month, and year, the sequence SHALL be strictly monotonically increasing (receipt number uniqueness property)

### Requirement 9: Receipt Reprint

**User Story:** As a kasir, I want to reprint a receipt from payment history, so that I can provide a duplicate receipt if the customer requests one.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/payments/:payment_id/receipt`, THE Payment_Module SHALL return the receipt data for the specified payment regardless of when the payment was recorded
2. IF the payment ID does not exist or belongs to a different tenant, THEN THE Payment_Module SHALL return HTTP 404 with error code `PAYMENT_NOT_FOUND`
3. IF the payment has been voided, THEN THE Payment_Module SHALL return the receipt data with a `voided` flag set to true and include the void reason

### Requirement 10: Void Payment — Authorization and Time Limit

**User Story:** As an admin, I want to void a payment within 24 hours of recording with a mandatory reason, so that cashier mistakes can be corrected with proper authorization and audit trail.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/payments/:payment_id/void` by a user with `admin` role, THE Payment_Module SHALL void the specified payment
2. THE Validator SHALL require a `reason` field (min 5, max 500 characters) explaining why the payment is being voided
3. IF the authenticated user does not have the `admin` role, THEN THE Payment_Module SHALL return HTTP 403 with error code `FORBIDDEN` and message indicating only admins can void payments
4. IF the payment was recorded more than 24 hours ago (based on `created_at`), THEN THE Payment_Module SHALL return HTTP 422 with error code `VOID_TIME_LIMIT_EXCEEDED` and message indicating that payments older than 24 hours must be reversed via credit note
5. IF the payment has already been voided, THEN THE Payment_Module SHALL return HTTP 422 with error code `PAYMENT_ALREADY_VOIDED`
6. IF the payment ID does not exist or belongs to a different tenant, THEN THE Payment_Module SHALL return HTTP 404 with error code `PAYMENT_NOT_FOUND`

### Requirement 11: Void Payment — Invoice Status Rollback

**User Story:** As an admin, I want voiding a payment to automatically roll back the invoice status and related side effects, so that the system state is consistent after a void.

#### Acceptance Criteria

1. WHEN a payment is voided, THE Payment_Module SHALL reduce the invoice's `paid_amount` by the voided payment amount using Optimistic_Locking
2. WHEN the invoice's new `paid_amount` becomes 0 and the invoice `due_date` is in the future, THE Payment_Module SHALL transition the invoice status back to `belum_bayar`
3. WHEN the invoice's new `paid_amount` becomes 0 and the invoice `due_date` is in the past, THE Payment_Module SHALL transition the invoice status back to `terlambat`
4. WHEN the invoice's new `paid_amount` is greater than 0 but less than `total_amount`, THE Payment_Module SHALL transition the invoice status to `bayar_sebagian`
5. WHEN the voided payment had caused an overpayment (excess added to credit_balance), THE Payment_Module SHALL reduce the customer's `credit_balance` by the excess amount that was previously added
6. IF reducing the customer's `credit_balance` would result in a negative value (because credit was already spent), THEN THE Payment_Module SHALL set `credit_balance` to 0 and log a warning for admin review
7. THE Payment_Module SHALL write an Invoice_Audit_Log entry with action `invoice.payment_voided` including the void reason, voided amount, and resulting invoice status
8. WHEN a payment void causes an invoice to return to `terlambat` status and the customer was previously un-isolated due to that payment, THE Payment_Module SHALL publish a `payment.voided.re_isolir` event to the Event_Queue for the isolir module to process

### Requirement 12: Bulk Payment Import — CSV Upload

**User Story:** As a tenant admin, I want to upload a CSV file containing multiple payment records, so that I can batch-process payments collected offline (e.g., from field collectors or bank reconciliation).

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/payments/import` with a CSV file, THE Payment_Module SHALL parse and validate the CSV contents before processing
2. THE Payment_Module SHALL expect the CSV to have the following columns: `customer_id_seq` (e.g., PLG-001), `amount` (positive integer), `payment_method` (one of `tunai`, `transfer`, `lainnya`), `payment_date` (format YYYY-MM-DD)
3. THE Payment_Module SHALL accept the following optional CSV columns: `reference_number`, `notes`
4. THE Payment_Module SHALL validate each row independently and collect all validation errors before processing any payments
5. IF any row fails validation (invalid customer_id_seq, invalid amount, invalid method, invalid date format), THEN THE Payment_Module SHALL return HTTP 422 with a detailed error response listing each failed row number and the specific validation error
6. WHEN all rows pass validation, THE Payment_Module SHALL process each payment using FIFO allocation against the customer's open invoices
7. THE Payment_Module SHALL return a response containing: `total_rows`, `success_count`, `failure_count`, and an array of `results` with per-row status (success with receipt_number, or failure with reason)
8. THE Payment_Module SHALL limit the CSV file to a maximum of 500 rows per upload
9. IF the CSV file exceeds 500 rows, THEN THE Payment_Module SHALL return HTTP 400 with error code `CSV_TOO_LARGE`
10. THE Payment_Module SHALL write an Invoice_Audit_Log entry with action `invoice.payment_recorded_bulk` for each successfully processed payment

### Requirement 13: Bulk Payment Import — Duplicate Detection

**User Story:** As a tenant admin, I want the bulk import to detect potential duplicate payments, so that the same payment is not accidentally recorded twice from re-uploaded CSV files.

#### Acceptance Criteria

1. WHEN processing a bulk import row, THE Payment_Module SHALL check for an existing non-voided payment with the same `customer_id`, `amount`, `payment_method`, and `payment_date` recorded within the last 24 hours
2. IF a potential duplicate is detected, THEN THE Payment_Module SHALL skip that row and include it in the response with status `skipped` and reason `potential_duplicate`
3. THE Payment_Module SHALL include a `duplicates_skipped` count in the bulk import response summary

### Requirement 14: Receipt Number Formatting

**User Story:** As a developer, I want receipt numbers formatted consistently, so that receipts are uniquely identifiable and follow the established numbering convention.

#### Acceptance Criteria

1. THE Payment_Module SHALL format receipt numbers as `PAY-{YYYY}-{MM}-{SEQ}` where YYYY is the 4-digit year, MM is the 2-digit zero-padded month, and SEQ is zero-padded to 4 digits minimum
2. WHEN the sequence exceeds 9999, THE Payment_Module SHALL expand the digit count automatically (e.g., PAY-2026-04-10000)
3. FOR ALL receipt numbers, parsing the formatted receipt number and re-formatting it SHALL produce the same string (round-trip formatting property)

### Requirement 15: Payment Recording with Late Fee Calculation

**User Story:** As a kasir, I want late fees automatically calculated when recording payment for overdue invoices in the multi-invoice flow, so that penalty policies are consistently enforced.

#### Acceptance Criteria

1. WHILE the tenant's `penalty_enabled` setting is true, THE Payment_Module SHALL calculate a late fee for each invoice with status `terlambat` that receives a payment allocation
2. THE Payment_Module SHALL calculate the late fee using the same logic as the existing `CalculateLateFee` function: fixed amount, percentage of subtotal, or daily rate based on tenant configuration
3. WHEN a late fee is calculated for an invoice that does not already have a penalty item, THE Payment_Module SHALL add a penalty line item to the invoice and recalculate `total_amount` before allocating the payment
4. THE Payment_Module SHALL recalculate the invoice's `remaining_amount` after adding the late fee, which may result in the payment only partially covering the invoice

### Requirement 16: Concurrency Safety for Multi-Invoice Payment

**User Story:** As a developer, I want multi-invoice payment processing to be safe against concurrent modifications, so that double payments and race conditions are prevented.

#### Acceptance Criteria

1. THE Payment_Module SHALL acquire a row-level lock (SELECT FOR UPDATE) on each invoice being updated during multi-invoice payment processing
2. WHEN an optimistic locking conflict is detected (version mismatch), THE Payment_Module SHALL retry the operation once by re-reading the invoice state
3. IF the retry also fails due to a version conflict, THEN THE Payment_Module SHALL return HTTP 409 with error code `CONCURRENT_MODIFICATION` and message indicating the payment should be retried
4. THE Payment_Module SHALL process all invoice allocations within a single database transaction to ensure atomicity — either all allocations succeed or none are applied
5. FOR ALL successful multi-invoice payments, the sum of individual allocation amounts SHALL equal the total payment amount minus any excess credited to the customer (allocation sum invariant)

### Requirement 17: Payment Proof Upload (Bukti Transfer)

**User Story:** As a kasir, I want to optionally upload a proof of transfer image when recording a payment, so that there is visual evidence of bank transfers for reconciliation and dispute resolution.

#### Acceptance Criteria

1. THE Validator SHALL accept an optional `proof_image` field (multipart file upload) when recording a payment via `/v1/payments/multi`, `/v1/payments/pay-all`, or the existing single-invoice payment endpoint
2. THE Payment_Module SHALL accept image files in JPEG, PNG, or WebP format with a maximum file size of 5 MB
3. IF the uploaded file exceeds 5 MB, THEN THE Payment_Module SHALL return HTTP 400 with error code `FILE_TOO_LARGE`
4. IF the uploaded file is not a valid image format (JPEG, PNG, WebP), THEN THE Payment_Module SHALL return HTTP 400 with error code `INVALID_FILE_FORMAT`
5. WHEN a proof image is uploaded, THE Payment_Module SHALL store the file in the configured storage path (local filesystem or object storage) and save the file path/URL in the `invoice_payments.proof_image_url` column
6. WHEN a GET request is made to `/v1/payments/:payment_id/proof`, THE Payment_Module SHALL return the proof image file for the specified payment
7. IF the payment has no proof image, THEN THE Payment_Module SHALL return HTTP 404 with error code `PROOF_NOT_FOUND`
8. THE Payment_Module SHALL include the `proof_image_url` field in the payment list response and receipt data when a proof image exists

