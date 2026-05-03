# Requirements Document

## Introduction

This spec defines the Payment Gateway module for ISPBoss billing-api. It extends the existing billing system with online payment capabilities through Xendit and Midtrans payment gateways. The module builds ON TOP of the existing payment-manual module (which handles tunai, transfer, lainnya) by adding: payment link generation and management, webhook handlers for payment notifications, multi-gateway configuration per tenant, Virtual Account/QRIS/e-wallet/credit card support, payment link expiry with auto-regeneration, idempotent webhook processing with signature verification, and walled garden integration.

All operations are tenant-scoped via PostgreSQL RLS. The module integrates with the existing invoice, payment, and customer modules within the billing-api service (Go, Fiber, sqlc, pgx, asynq).

## Glossary

- **Billing_API**: The Go backend service (`services/billing-api`) that handles invoice, billing, customer, and auth operations
- **Payment_Gateway_Module**: The subsystem within Billing_API responsible for online payment link generation, webhook processing, gateway configuration, and payment status synchronization
- **Payment_Gateway**: An external payment service provider (Xendit or Midtrans) that processes online payments and sends webhook notifications
- **Xendit**: A payment gateway provider supporting Virtual Account, QRIS, e-wallet, and credit card payments via REST API
- **Midtrans**: A payment gateway provider supporting Virtual Account, QRIS, e-wallet, and credit card payments via REST API (Snap/Core API)
- **Payment_Link**: A URL generated via a Payment_Gateway that allows a customer to complete an online payment through their chosen payment method
- **Payment_Method_Online**: One of `virtual_account`, `qris`, `ewallet`, or `credit_card` — the online payment channels available through Payment_Gateway
- **Virtual_Account**: A temporary bank account number (BCA, BNI, BRI, Mandiri, Permata) generated for a specific payment, allowing bank transfer without manual reconciliation
- **QRIS**: Quick Response Code Indonesian Standard — a unified QR code payment standard accepted by multiple e-wallets and mobile banking apps
- **E_Wallet**: Electronic wallet payment methods including OVO, GoPay, DANA, and ShopeePay
- **Webhook**: An HTTP callback from a Payment_Gateway to the Billing_API notifying about payment status changes (paid, expired, failed)
- **Webhook_Signature**: A cryptographic signature (HMAC or RSA) attached to webhook requests by the Payment_Gateway for authenticity verification
- **Callback_Token**: A secret token configured per gateway used to verify webhook authenticity (Xendit uses callback token, Midtrans uses server key signature)
- **Idempotency_Key**: A unique identifier from the Payment_Gateway (e.g., payment ID or transaction ID) used to prevent duplicate processing of the same webhook event
- **Payment_Link_Expiry**: The configurable duration (default 7 days) after which a Payment_Link becomes invalid and must be regenerated
- **Gateway_Configuration**: Per-tenant settings including API keys, enabled payment methods, and callback URLs stored in the `payment_gateway_configs` table
- **Webhook_Log**: An append-only record of all incoming webhook requests (including failed verifications) stored in the `webhook_logs` table
- **IP_Whitelist**: A list of IP addresses from which webhook requests are accepted, sourced from Xendit and Midtrans documentation
- **Optimistic_Locking**: Concurrency control via a `version` field on the invoice to prevent double payment processing
- **Double_Payment**: A scenario where both a manual payment and a webhook payment are processed for the same invoice concurrently, resulting in overpayment
- **Overpayment**: When total payments received exceed the invoice total, the excess is credited to the customer's `credit_balance`
- **Walled_Garden**: The captive portal page shown to isolated (isolir) customers, which includes a "Pay Now" button linking to the Payment_Link
- **Event_Queue**: The Asynq task queue used for asynchronous event processing (e.g., triggering un-isolir after payment)
- **Tenant**: An ISP operator using the ISPBoss platform, identified by `tenant_id`
- **Invoice**: A billing document with status, amount, and version field for Optimistic_Locking
- **Customer**: An ISP subscriber with invoices, payment history, and credit_balance

## Requirements

### Requirement 1: Gateway Configuration Management

**User Story:** As a tenant admin, I want to configure one or more payment gateways (Xendit and/or Midtrans) with API keys and enabled payment methods, so that my customers can pay online through their preferred channels.

#### Acceptance Criteria

1. THE Payment_Gateway_Module SHALL store gateway configuration per tenant in a `payment_gateway_configs` table with columns: `id` (UUID PK), `tenant_id` (UUID FK), `gateway_provider` (enum: `xendit`, `midtrans`), `is_active` (boolean), `api_key_encrypted` (text), `webhook_secret_encrypted` (text), `enabled_methods` (JSONB array), `payment_link_expiry_days` (integer, default 7), `created_at`, `updated_at`
2. WHEN a POST request is made to `/v1/settings/payment-gateways` with valid gateway configuration, THE Payment_Gateway_Module SHALL create a new gateway configuration for the authenticated tenant
3. WHEN a PUT request is made to `/v1/settings/payment-gateways/:id`, THE Payment_Gateway_Module SHALL update the specified gateway configuration
4. WHEN a GET request is made to `/v1/settings/payment-gateways`, THE Payment_Gateway_Module SHALL return all gateway configurations for the authenticated tenant with API keys masked (showing only last 4 characters)
5. THE Payment_Gateway_Module SHALL allow a tenant to have both Xendit and Midtrans active simultaneously
6. THE Payment_Gateway_Module SHALL encrypt API keys and webhook secrets at rest using AES-256-GCM before storing in the database
7. THE Validator SHALL require the following fields for gateway configuration: `gateway_provider` (one of `xendit`, `midtrans`), `api_key` (non-empty string), `webhook_secret` (non-empty string), `enabled_methods` (non-empty array of valid method identifiers)
8. THE Payment_Gateway_Module SHALL validate that `enabled_methods` contains only valid values: for Xendit (`va_bca`, `va_bni`, `va_bri`, `va_mandiri`, `va_permata`, `qris`, `ewallet_ovo`, `ewallet_gopay`, `ewallet_dana`, `ewallet_shopeepay`, `credit_card`), for Midtrans (`va_bca`, `va_bni`, `va_bri`, `va_mandiri`, `va_permata`, `qris`, `ewallet_gopay`, `ewallet_shopeepay`, `credit_card`)
9. WHEN a DELETE request is made to `/v1/settings/payment-gateways/:id`, THE Payment_Gateway_Module SHALL soft-delete the gateway configuration by setting `is_active` to false
10. IF a tenant has no active gateway configuration, THEN THE Payment_Gateway_Module SHALL skip payment link generation during invoice creation

### Requirement 2: Payment Link Generation

**User Story:** As a tenant admin, I want payment links automatically generated when invoices are created (if a gateway is active), so that customers receive a ready-to-pay link with their invoice notification.

#### Acceptance Criteria

1. WHEN an invoice is generated and the tenant has at least one active Gateway_Configuration, THE Payment_Gateway_Module SHALL create a payment link via the configured Payment_Gateway API
2. THE Payment_Gateway_Module SHALL store payment links in a `payment_links` table with columns: `id` (UUID PK), `tenant_id` (UUID FK), `invoice_id` (UUID FK), `gateway_provider` (enum), `external_id` (string, unique per gateway), `payment_url` (text), `amount` (bigint), `status` (enum: `active`, `expired`, `paid`), `expires_at` (timestamptz), `created_at`, `updated_at`
3. THE Payment_Gateway_Module SHALL set the payment link expiry based on the tenant's `payment_link_expiry_days` configuration (default 7 days), calculated as `created_at + (payment_link_expiry_days × 24 hours)`
4. THE Payment_Gateway_Module SHALL include the customer name, invoice number, and total amount in the payment link description sent to the gateway
5. WHEN a payment link is generated, THE Payment_Gateway_Module SHALL include the `payment_url` in the invoice notification sent to the customer
6. IF the Payment_Gateway API call fails during link generation, THEN THE Payment_Gateway_Module SHALL log the error, mark the payment link as `failed`, and enqueue a retry task (max 3 retries with exponential backoff: 1min, 5min, 15min)
7. THE Payment_Gateway_Module SHALL generate payment links asynchronously via the Event_Queue to avoid blocking invoice generation

### Requirement 3: Multi-Invoice Payment Link

**User Story:** As a tenant admin, I want a single payment link generated for the total of all outstanding invoices when a customer has multiple arrears, so that customers can clear all debts in one transaction.

#### Acceptance Criteria

1. WHEN a customer has multiple open invoices (status `belum_bayar`, `terlambat`, or `bayar_sebagian`) and a payment link is requested, THE Payment_Gateway_Module SHALL generate a single payment link for the sum of all remaining amounts
2. THE Payment_Gateway_Module SHALL store the multi-invoice payment link with a reference to all covered invoice IDs in a `payment_link_invoices` junction table
3. WHEN a multi-invoice payment link is paid, THE Payment_Gateway_Module SHALL allocate the payment across invoices using FIFO order (oldest `due_date` first) via the existing `AllocatePaymentFIFO` function
4. WHEN a GET request is made to `/v1/customers/:customer_id/payment-link`, THE Payment_Gateway_Module SHALL return the active payment link for the customer (single or multi-invoice)
5. IF an active payment link already exists for the customer, THEN THE Payment_Gateway_Module SHALL return the existing link instead of generating a new one
6. WHEN a POST request is made to `/v1/customers/:customer_id/payment-link/regenerate`, THE Payment_Gateway_Module SHALL expire the existing link and generate a new one with updated amounts

### Requirement 4: Payment Link Expiry and Regeneration

**User Story:** As a tenant admin, I want expired payment links to be automatically regenerated when reminders are sent, so that customers always receive a valid payment link.

#### Acceptance Criteria

1. WHEN a payment link's `expires_at` timestamp is reached, THE Payment_Gateway_Module SHALL update the payment link status to `expired`
2. WHEN a billing reminder notification is triggered for an invoice with an expired payment link, THE Payment_Gateway_Module SHALL generate a new payment link and include the new URL in the reminder
3. WHEN a customer requests a new payment link via the Walled_Garden, THE Payment_Gateway_Module SHALL expire the current link (if any) and generate a new one
4. THE Payment_Gateway_Module SHALL run a periodic background job (every hour) to mark expired payment links based on `expires_at`
5. THE Payment_Gateway_Module SHALL NOT generate a new payment link for invoices that are already `lunas` or `batal`

### Requirement 5: Webhook Endpoint — Request Reception

**User Story:** As a developer, I want dedicated webhook endpoints for each payment gateway that receive and log all incoming notifications, so that payment status updates are captured reliably.

#### Acceptance Criteria

1. THE Payment_Gateway_Module SHALL expose a POST endpoint at `/webhooks/xendit` for receiving Xendit payment notifications
2. THE Payment_Gateway_Module SHALL expose a POST endpoint at `/webhooks/midtrans` for receiving Midtrans payment notifications
3. WHEN a webhook request is received, THE Payment_Gateway_Module SHALL log the complete request (headers, body, source IP, timestamp) in the `webhook_logs` table before any processing
4. THE Payment_Gateway_Module SHALL store webhook logs with columns: `id` (UUID PK), `tenant_id` (UUID, nullable for pre-identification), `gateway_provider` (enum), `event_type` (string), `external_id` (string), `request_body` (JSONB), `source_ip` (inet), `signature_valid` (boolean), `processing_status` (enum: `received`, `verified`, `processed`, `failed`, `duplicate`), `error_message` (text, nullable), `created_at`
5. THE Payment_Gateway_Module SHALL return HTTP 200 immediately after logging the webhook request to prevent gateway retries, then process the webhook asynchronously via the Event_Queue

### Requirement 6: Webhook Signature Verification

**User Story:** As a developer, I want all webhook requests verified using cryptographic signatures, so that only authentic notifications from payment gateways are processed.

#### Acceptance Criteria

1. WHEN a Xendit webhook is received, THE Payment_Gateway_Module SHALL verify the request by comparing the `x-callback-token` header value against the stored `webhook_secret` for the tenant's Xendit configuration
2. WHEN a Midtrans webhook is received, THE Payment_Gateway_Module SHALL verify the request by computing SHA-512 hash of `order_id + status_code + gross_amount + server_key` and comparing against the `signature_key` field in the notification body
3. IF the signature verification fails, THEN THE Payment_Gateway_Module SHALL log the request with `signature_valid = false` and `processing_status = failed`, and return HTTP 200 (to prevent retries) without processing the payment
4. THE Payment_Gateway_Module SHALL identify the tenant from the `external_id` field in the webhook payload (which contains the tenant-scoped payment link ID)
5. IF the `external_id` in the webhook payload does not correspond to any existing payment link in the database, THEN THE Payment_Gateway_Module SHALL log the request with `processing_status = failed` and error message `payment_link_not_found`, and stop processing without recording a payment

### Requirement 7: Webhook IP Whitelist

**User Story:** As a developer, I want webhook endpoints to only accept requests from known payment gateway IP addresses, so that spoofed webhook requests are rejected at the network level.

#### Acceptance Criteria

1. THE Payment_Gateway_Module SHALL maintain a configurable IP whitelist for Xendit webhook source IPs
2. THE Payment_Gateway_Module SHALL maintain a configurable IP whitelist for Midtrans webhook source IPs
3. WHEN a webhook request is received from an IP not in the whitelist for the respective gateway, THE Payment_Gateway_Module SHALL log the request with `processing_status = failed` and error message `ip_not_whitelisted`, and return HTTP 403
4. THE Payment_Gateway_Module SHALL load IP whitelists from application configuration (environment variables or config file) to allow updates without code deployment
5. IF the IP whitelist configuration is empty for a gateway, THEN THE Payment_Gateway_Module SHALL skip IP validation for that gateway (to support development/testing environments)

### Requirement 8: Webhook Idempotency

**User Story:** As a developer, I want webhook processing to be idempotent using the gateway's payment ID as a deduplication key, so that retried or duplicate webhooks do not cause double payment recording.

#### Acceptance Criteria

1. WHEN a webhook is received, THE Payment_Gateway_Module SHALL check if a webhook with the same `external_id` and `event_type` has already been successfully processed (status `processed` in `webhook_logs`)
2. IF a duplicate webhook is detected, THEN THE Payment_Gateway_Module SHALL log the request with `processing_status = duplicate` and return HTTP 200 without further processing
3. THE Payment_Gateway_Module SHALL use the Payment_Gateway's unique transaction/payment ID as the Idempotency_Key (Xendit: `id` field, Midtrans: `transaction_id` field)
4. THE Payment_Gateway_Module SHALL acquire a row-level advisory lock on the payment link ID before processing to prevent race conditions between concurrent webhook deliveries

### Requirement 9: Webhook Event Processing — Payment Paid

**User Story:** As a developer, I want the system to automatically update invoice status, record payment, and trigger un-isolir when a `payment.paid` webhook is received, so that the full payment flow completes without manual intervention.

#### Acceptance Criteria

1. WHEN a `payment.paid` (Xendit: `invoice.paid`) or equivalent Midtrans notification (transaction_status: `capture`/`settlement`) is received and verified, THE Payment_Gateway_Module SHALL locate the corresponding payment link and associated invoice(s)
2. THE Payment_Gateway_Module SHALL record a payment in the `invoice_payments` table with `payment_method` set to the specific online method used (e.g., `va_bca`, `qris`, `ewallet_gopay`, `credit_card`)
3. THE Payment_Gateway_Module SHALL update the invoice status using Optimistic_Locking: transition to `lunas` if fully paid, or `bayar_sebagian` if partially paid
4. WHEN the payment covers a multi-invoice payment link, THE Payment_Gateway_Module SHALL allocate the payment across invoices using the `AllocatePaymentFIFO` function
5. WHEN the payment amount exceeds the total remaining on all linked invoices, THE Payment_Gateway_Module SHALL add the excess to the customer's `credit_balance`
6. THE Payment_Gateway_Module SHALL generate a receipt number using the existing receipt sequence mechanism (format `PAY-{YYYY}-{MM}-{SEQ}`)
7. THE Payment_Gateway_Module SHALL write an Invoice_Audit_Log entry with action `invoice.payment_online` for each invoice that receives an allocation
8. WHEN an invoice transitions to `lunas` and the customer status is `isolir`, THE Payment_Gateway_Module SHALL publish a `payment.online.received` event to the Event_Queue to trigger automatic un-isolir
9. THE Payment_Gateway_Module SHALL publish a `payment.online.confirmation` event to the Event_Queue to trigger a payment confirmation notification to the customer

### Requirement 10: Webhook Event Processing — Payment Expired

**User Story:** As a developer, I want expired payment notifications to update the payment link status without affecting the invoice, so that the system accurately reflects payment link lifecycle.

#### Acceptance Criteria

1. WHEN a `payment.expired` (Xendit: `invoice.expired`) or equivalent Midtrans notification (transaction_status: `expire`) is received and verified, THE Payment_Gateway_Module SHALL update the corresponding payment link status to `expired`
2. THE Payment_Gateway_Module SHALL NOT change the invoice status when a payment link expires (invoice remains in its current status)
3. THE Payment_Gateway_Module SHALL write a Webhook_Log entry with `processing_status = processed` and `event_type = payment.expired`

### Requirement 11: Webhook Event Processing — Payment Failed

**User Story:** As a developer, I want failed payment notifications logged and optionally communicated to the customer, so that payment issues are visible for troubleshooting.

#### Acceptance Criteria

1. WHEN a `payment.failed` or equivalent Midtrans notification (transaction_status: `deny`/`cancel`) is received and verified, THE Payment_Gateway_Module SHALL log the failure details in the Webhook_Log
2. THE Payment_Gateway_Module SHALL NOT change the invoice status when a payment fails
3. THE Payment_Gateway_Module SHALL publish a `payment.online.failed` event to the Event_Queue to trigger a payment failure notification to the customer (informing them to retry)

### Requirement 12: Concurrency and Double Payment Prevention

**User Story:** As a developer, I want the system to handle concurrent manual and online payments gracefully, so that double payments result in customer credit rather than data corruption.

#### Acceptance Criteria

1. WHEN processing a webhook payment, THE Payment_Gateway_Module SHALL acquire a SELECT FOR UPDATE lock on the invoice(s) being updated within a database transaction
2. WHEN an Optimistic_Locking conflict is detected (invoice version mismatch), THE Payment_Gateway_Module SHALL re-read the invoice state and determine if the invoice is already `lunas`
3. IF the invoice is already `lunas` when the webhook payment is being processed, THEN THE Payment_Gateway_Module SHALL treat the payment as an Overpayment and add the full amount to the customer's `credit_balance`
4. WHEN a Double_Payment is detected (invoice already paid by another source), THE Payment_Gateway_Module SHALL publish a `payment.double_payment` event to the Event_Queue to notify the admin
5. THE Payment_Gateway_Module SHALL write an Invoice_Audit_Log entry with action `invoice.double_payment_detected` including both payment sources (manual vs online, or gateway A vs gateway B)

### Requirement 13: Walled Garden Integration

**User Story:** As an isolated customer viewing the walled garden page, I want a "Pay Now" button that takes me directly to the payment link, so that I can pay and restore my internet access immediately.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/public/walled-garden/:customer_id/payment-info`, THE Payment_Gateway_Module SHALL return the active payment link URL, total arrears amount, and invoice details for the isolated customer
2. IF no active payment link exists for the customer, THEN THE Payment_Gateway_Module SHALL generate a new payment link on-demand and return it in the response
3. IF the existing payment link has expired, THEN THE Payment_Gateway_Module SHALL regenerate a new payment link and return the new URL
4. THE Payment_Gateway_Module SHALL NOT require authentication for the walled garden payment info endpoint (public endpoint, rate-limited)
5. THE Payment_Gateway_Module SHALL rate-limit the walled garden endpoint to 10 requests per minute per customer ID to prevent abuse

### Requirement 14: Payment Status Query API

**User Story:** As a tenant admin, I want to view the status of online payments and payment links for any invoice, so that I can troubleshoot payment issues and provide customer support.

#### Acceptance Criteria

1. WHEN a GET request is made to `/v1/invoices/:invoice_id/payment-links`, THE Payment_Gateway_Module SHALL return all payment links (active, expired, paid) associated with the specified invoice
2. THE Payment_Gateway_Module SHALL include the following fields for each payment link: `id`, `gateway_provider`, `payment_url`, `amount`, `status`, `expires_at`, `created_at`, and the specific payment method used (if paid)
3. WHEN a GET request is made to `/v1/payment-links/:id/webhooks`, THE Payment_Gateway_Module SHALL return all webhook logs associated with the specified payment link
4. THE Payment_Gateway_Module SHALL include webhook log fields: `event_type`, `processing_status`, `source_ip`, `signature_valid`, `error_message`, `created_at`

### Requirement 15: Gateway Provider Adapter Pattern

**User Story:** As a developer, I want a unified interface for interacting with different payment gateways, so that adding new gateways in the future requires minimal code changes.

#### Acceptance Criteria

1. THE Payment_Gateway_Module SHALL define a `PaymentGatewayAdapter` interface with methods: `CreatePaymentLink(ctx, request) (PaymentLinkResponse, error)`, `VerifyWebhookSignature(ctx, headers, body, secret) (bool, error)`, `ParseWebhookPayload(body) (WebhookEvent, error)`, `ExpirePaymentLink(ctx, externalID) error`
2. THE Payment_Gateway_Module SHALL implement the `PaymentGatewayAdapter` interface for Xendit as `XenditAdapter`
3. THE Payment_Gateway_Module SHALL implement the `PaymentGatewayAdapter` interface for Midtrans as `MidtransAdapter`
4. THE Payment_Gateway_Module SHALL select the appropriate adapter at runtime based on the `gateway_provider` field in the payment link or webhook request
5. FOR ALL implementations of `PaymentGatewayAdapter`, calling `CreatePaymentLink` with valid input SHALL return a `PaymentLinkResponse` containing a non-empty `payment_url` and `external_id` (adapter contract property)

### Requirement 16: Webhook Log Retention and Cleanup

**User Story:** As a developer, I want webhook logs retained for audit purposes with configurable cleanup, so that the database does not grow unbounded while maintaining compliance records.

#### Acceptance Criteria

1. THE Payment_Gateway_Module SHALL retain all webhook logs for a minimum of 90 days
2. THE Payment_Gateway_Module SHALL run a daily background job to delete webhook logs older than the configured retention period (default 90 days)
3. THE Payment_Gateway_Module SHALL never delete webhook logs with `processing_status = failed` or `signature_valid = false` (security-relevant logs are retained indefinitely)

### Requirement 17: Payment Link Amount Synchronization

**User Story:** As a developer, I want payment link amounts to stay synchronized with invoice totals, so that late fees added after link generation do not cause underpayment.

#### Acceptance Criteria

1. WHEN a late fee is added to an invoice that has an active payment link, THE Payment_Gateway_Module SHALL expire the current payment link and generate a new one with the updated total amount
2. WHEN an invoice is partially paid (manually) while an active payment link exists, THE Payment_Gateway_Module SHALL expire the current payment link and generate a new one with the remaining amount
3. WHEN an invoice is fully paid (manually) while an active payment link exists, THE Payment_Gateway_Module SHALL expire the current payment link and update its status to `paid`
4. THE Payment_Gateway_Module SHALL listen for `invoice.payment_recorded` and `invoice.penalty_added` events from the Event_Queue to trigger payment link synchronization
5. THE existing payment-manual module (PaymentUsecase.RecordMultiPayment) SHALL publish an `invoice.payment_recorded` event to the Event_Queue after successfully recording a manual payment, with payload containing `tenant_id`, `invoice_id`, and `customer_id`
6. THE existing invoice module (InvoiceCronUsecase or InvoiceActionUsecase) SHALL publish an `invoice.penalty_added` event to the Event_Queue after adding a late fee to an invoice, with payload containing `tenant_id`, `invoice_id`, and `new_total_amount`

### Requirement 18: Payment Gateway Health Check

**User Story:** As a tenant admin, I want to verify that my payment gateway configuration is working correctly, so that I can troubleshoot connectivity issues before they affect customers.

#### Acceptance Criteria

1. WHEN a POST request is made to `/v1/settings/payment-gateways/:id/test`, THE Payment_Gateway_Module SHALL make a test API call to the configured gateway to verify credentials and connectivity
2. THE Payment_Gateway_Module SHALL return a response indicating success or failure with a descriptive error message (e.g., `invalid_api_key`, `network_timeout`, `gateway_unavailable`)
3. THE Payment_Gateway_Module SHALL timeout the test call after 10 seconds and return `gateway_unavailable` if no response is received
