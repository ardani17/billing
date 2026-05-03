# Implementation Plan: Payment Gateway Module

## Overview

Bottom-up implementation of the Payment Gateway module for ISPBoss billing-api. Starts with database migrations (payment_gateway_configs, payment_links, webhook_logs tables), then domain entities (types, interfaces, errors, DTOs in domain/gateway.go), gateway adapter layer (PaymentGatewayAdapter interface, XenditAdapter, MidtransAdapter, AES-256-GCM crypto), property tests for pure functions, repository implementations (gateway_config_repo, payment_link_repo, webhook_log_repo), usecase layer (gateway_usecase, webhook_usecase), handler layer (gateway_handler, webhook_handler), worker layer (gateway_worker), router wiring and main.go DI, and unit tests. Each task builds on the previous and is independently testable. All code is Go, using existing patterns (Fiber, sqlc, pgx, asynq, go-playground/validator, rapid). Monetary values are BIGINT (Rupiah). The module REUSES existing domain functions (AllocatePaymentFIFO, FormatReceiptNumber), ReceiptSequenceRepository, and InvoicePaymentRepository. Migration numbering starts at 000028 (after payment-manual migrations 000025-000027). Max 200 lines per file. All code comments in Indonesian.

## Tasks

- [x] 1. Database migrations
  - [x] 1.1 Create migration: payment_gateway_configs table
    - Create `services/billing-api/migrations/000028_create_payment_gateway_configs.up.sql` with: `payment_gateway_configs` table (id UUID PK DEFAULT gen_random_uuid(), tenant_id UUID NOT NULL FK tenants(id), gateway_provider VARCHAR(20) NOT NULL CHECK IN ('xendit','midtrans'), is_active BOOLEAN NOT NULL DEFAULT true, api_key_encrypted TEXT NOT NULL, webhook_secret_encrypted TEXT NOT NULL, enabled_methods JSONB NOT NULL DEFAULT '[]'::jsonb, payment_link_expiry_days INTEGER NOT NULL DEFAULT 7, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()), UNIQUE (tenant_id, gateway_provider), RLS policy payment_gateway_configs_tenant_policy, index idx_payment_gateway_configs_tenant_active ON (tenant_id, is_active) WHERE is_active = true
    - Create `services/billing-api/migrations/000028_create_payment_gateway_configs.down.sql` -- drop policies, indexes, table
    - _Requirements: 1.1, 1.5, 1.6_

  - [x] 1.2 Create migration: payment_links and payment_link_invoices tables
    - Create `services/billing-api/migrations/000029_create_payment_links.up.sql` with: `payment_links` table (id UUID PK, tenant_id UUID NOT NULL FK, customer_id UUID NOT NULL FK customers(id), gateway_provider VARCHAR(20) NOT NULL CHECK, gateway_config_id UUID NOT NULL FK payment_gateway_configs(id), external_id VARCHAR(255) NOT NULL, payment_url TEXT NOT NULL, amount BIGINT NOT NULL CHECK (amount > 0), status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK IN ('active','expired','paid','failed'), expires_at TIMESTAMPTZ NOT NULL, paid_at TIMESTAMPTZ, paid_method VARCHAR(50), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()), RLS policy, UNIQUE INDEX idx_payment_links_external_id ON (external_id), INDEX idx_payment_links_customer_active ON (customer_id, status) WHERE status='active', INDEX idx_payment_links_expires_at ON (expires_at) WHERE status='active'. Junction table `payment_link_invoices` (id UUID PK, payment_link_id UUID FK ON DELETE CASCADE, invoice_id UUID FK, UNIQUE (payment_link_id, invoice_id)), indexes on both FK columns
    - Create `services/billing-api/migrations/000029_create_payment_links.down.sql` -- drop tables
    - _Requirements: 2.1, 2.2, 2.3, 3.2, 4.1_

  - [x] 1.3 Create migration: webhook_logs table
    - Create `services/billing-api/migrations/000030_create_webhook_logs.up.sql` with: `webhook_logs` table (id UUID PK DEFAULT gen_random_uuid(), tenant_id UUID FK tenants(id) nullable, gateway_provider VARCHAR(20) NOT NULL CHECK, event_type VARCHAR(100) NOT NULL, external_id VARCHAR(255) NOT NULL, request_body JSONB NOT NULL, source_ip INET NOT NULL, signature_valid BOOLEAN, processing_status VARCHAR(20) NOT NULL DEFAULT 'received' CHECK IN ('received','verified','processed','failed','duplicate'), error_message TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()). NO RLS. Indexes: idx_webhook_logs_idempotency ON (external_id, event_type) WHERE processing_status='processed', idx_webhook_logs_external_id ON (external_id, created_at DESC), idx_webhook_logs_cleanup ON (created_at) WHERE processing_status NOT IN ('failed') AND (signature_valid IS NULL OR signature_valid = true), idx_webhook_logs_tenant ON (tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL
    - Create `services/billing-api/migrations/000030_create_webhook_logs.down.sql` -- drop table
    - _Requirements: 5.3, 5.4, 8.1, 16.1, 16.2, 16.3_

- [x] 2. Domain entities -- Gateway types, interfaces, errors, DTOs
  - [x] 2.1 Create domain/gateway.go with enums, entity types, and domain errors
    - Create `services/billing-api/internal/domain/gateway.go` with: `GatewayProvider` enum (xendit, midtrans), `PaymentLinkStatus` enum (active, expired, paid, failed), `WebhookProcessingStatus` enum (received, verified, processed, failed, duplicate), `GatewayConfig` struct (ID, TenantID, GatewayProvider, IsActive, APIKeyEncrypted, WebhookSecretEncrypted, APIKeyMasked, EnabledMethods, PaymentLinkExpiryDays, CreatedAt, UpdatedAt), `PaymentLink` struct (ID, TenantID, CustomerID, GatewayProvider, GatewayConfigID, ExternalID, PaymentURL, Amount, Status, ExpiresAt, PaidAt, PaidMethod, CreatedAt, UpdatedAt), `PaymentLinkInvoice` struct, `WebhookLog` struct (ID, TenantID, GatewayProvider, EventType, ExternalID, RequestBody, SourceIP, SignatureValid, ProcessingStatus, ErrorMessage, CreatedAt), domain error variables (ErrGatewayConfigNotFound, ErrGatewayConfigDuplicate, ErrPaymentLinkNotFound, ErrPaymentLinkAlreadyActive, ErrPaymentLinkExpired, ErrWebhookSignatureInvalid, ErrWebhookDuplicate, ErrWebhookIPNotWhitelisted, ErrGatewayUnavailable, ErrGatewayInvalidAPIKey, ErrNoActiveGateway, ErrInvalidEnabledMethods, ErrEncryptionFailed, ErrDecryptionFailed), `ValidXenditMethods` map, `ValidMidtransMethods` map, `ValidateEnabledMethods(provider, methods) error` pure function
    - _Requirements: 1.1, 1.8, 2.2, 5.4, 15.1_

  - [x] 2.2 Add gateway DTOs to domain/gateway.go (request/response types)
    - Append to `services/billing-api/internal/domain/gateway.go`: `CreateGatewayConfigRequest` struct (GatewayProvider, APIKey, WebhookSecret, EnabledMethods, PaymentLinkExpiryDays with validation tags), `UpdateGatewayConfigRequest` struct, `GeneratePaymentLinkRequest` struct (TenantID, CustomerID, InvoiceIDs), `RegeneratePaymentLinkRequest` struct, `PaymentLinkResponse` struct (ExternalID, PaymentURL, ExpiresAt), `CustomerPaymentLinkResponse` struct (PaymentLink, Invoices, TotalArrears), `WebhookEvent` struct (EventType, ExternalID, TransactionID, Amount, PaidMethod, GatewayProvider, RawPayload), `InvoicePaymentLinksResponse` struct, `PaymentLinkWebhooksResponse` struct, `WalledGardenPaymentInfo` struct (PaymentURL, TotalArrears, Invoices, CustomerName), `GatewayTestResult` struct (Success, ErrorCode, ErrorMessage, LatencyMs)
    - Note: If domain/gateway.go exceeds 200 lines, split DTOs into `domain/gateway_dto.go`
    - _Requirements: 1.2, 1.3, 1.4, 2.4, 2.5, 3.4, 3.6, 5.5, 6.4, 13.1, 14.1, 14.2, 14.3, 14.4, 18.1, 18.2_

  - [x] 2.3 Add repository interfaces to domain/gateway.go
    - Append to domain/gateway.go (or new file if >200 lines): `GatewayConfigRepository` interface (Create, GetByID, Update, Deactivate, ListByTenant, GetActiveByTenant, GetActiveByProvider, ExistsByProvider), `PaymentLinkRepository` interface (Create, GetByID, GetByExternalID, GetActiveByCustomer, GetInvoiceIDsByLinkID, UpdateStatus, UpdateStatusPaid, ListByInvoice, FindExpired, ExpireByID), `WebhookLogRepository` interface (Create, GetByID, UpdateStatus, UpdateSignatureValid, IsAlreadyProcessed, ListByPaymentLink, DeleteOlderThan)
    - _Requirements: 1.1, 2.2, 5.4, 8.1, 8.2, 14.1, 14.3, 16.2, 16.3_

- [x] 3. Gateway adapter layer
  - [x] 3.1 Create gateway/adapter.go with PaymentGatewayAdapter interface and factory
    - Create `services/billing-api/internal/gateway/adapter.go` with: `PaymentGatewayAdapter` interface (CreatePaymentLink, VerifyWebhookSignature, ParseWebhookPayload, ExpirePaymentLink, TestConnection), `CreateLinkRequest` struct (ExternalID, Amount, Description, CustomerName, CustomerEmail, ExpiryDuration, EnabledMethods), `NewAdapter(provider, apiKey) (PaymentGatewayAdapter, error)` factory function that returns XenditAdapter or MidtransAdapter based on provider
    - _Requirements: 15.1, 15.2, 15.3, 15.4_

  - [x] 3.2 Create gateway/xendit.go with XenditAdapter implementation
    - Create `services/billing-api/internal/gateway/xendit.go` with: `XenditAdapter` struct (apiKey, httpClient, baseURL), `NewXenditAdapter(apiKey) *XenditAdapter`, implement all PaymentGatewayAdapter methods. `CreatePaymentLink` calls Xendit Invoice API v2 (POST /v2/invoices). `VerifyWebhookSignature` compares x-callback-token header with stored secret. `ParseWebhookPayload` parses Xendit notification JSON into WebhookEvent. `ExpirePaymentLink` calls Xendit expire endpoint. `TestConnection` makes a test API call with 10s timeout
    - _Requirements: 6.1, 15.2, 15.5, 18.1, 18.2, 18.3_

  - [x] 3.3 Create gateway/midtrans.go with MidtransAdapter implementation
    - Create `services/billing-api/internal/gateway/midtrans.go` with: `MidtransAdapter` struct (serverKey, httpClient, baseURL), `NewMidtransAdapter(serverKey) *MidtransAdapter`, implement all PaymentGatewayAdapter methods. `CreatePaymentLink` calls Midtrans Snap API (POST /snap/v1/transactions). `VerifyWebhookSignature` computes SHA-512 hash of (order_id + status_code + gross_amount + server_key) and compares with signature_key in body. `ParseWebhookPayload` parses Midtrans notification JSON into WebhookEvent (maps transaction_status to event types). `ExpirePaymentLink` calls Midtrans cancel endpoint. `TestConnection` makes a test API call with 10s timeout
    - _Requirements: 6.2, 15.3, 15.5, 18.1, 18.2, 18.3_

  - [x] 3.4 Create gateway/crypto.go with AES-256-GCM encryption utilities
    - Create `services/billing-api/internal/gateway/crypto.go` with: `EncryptAESGCM(plaintext string, masterKey []byte) (string, error)` -- encrypts using AES-256-GCM, returns base64-encoded (nonce + ciphertext + tag), masterKey must be 32 bytes. `DecryptAESGCM(ciphertext string, masterKey []byte) (string, error)` -- decrypts base64-encoded ciphertext. `MaskAPIKey(apiKey string) string` -- returns masked key showing only last 4 characters (e.g., "****xyz"), keys < 4 chars returned as-is
    - _Requirements: 1.4, 1.6_

- [x] 4. Property tests for pure functions
  - [x] 4.1 Write property test: AES-256-GCM Encryption Round-Trip (Property 1)
    - **Property 1: AES-256-GCM Encryption Round-Trip**
    - In `services/billing-api/internal/gateway/crypto_test.go`, use `rapid.Check` to verify that for any plaintext string and any valid 32-byte master key, `EncryptAESGCM(plaintext, masterKey)` produces a ciphertext that when decrypted with `DecryptAESGCM(ciphertext, masterKey)` yields the original plaintext. Additionally verify that ciphertext differs from plaintext (encryption actually transforms data).
    - **Validates: Requirements 1.6**

  - [x] 4.2 Write property test: API Key Masking (Property 2)
    - **Property 2: API Key Masking**
    - In `services/billing-api/internal/gateway/crypto_test.go`, use `rapid.Check` to verify that for any API key string of length >= 4, `MaskAPIKey(apiKey)` returns a string where the last 4 characters match the last 4 characters of the original API key, and all preceding characters are asterisks. For strings of length < 4, the entire string is returned as-is.
    - **Validates: Requirements 1.4**

  - [x] 4.3 Write property test: Enabled Methods Validation Consistency (Property 3)
    - **Property 3: Enabled Methods Validation Consistency**
    - In `services/billing-api/internal/domain/gateway_test.go`, use `rapid.Check` to verify that for any gateway provider (Xendit or Midtrans) and for any method string, `ValidateEnabledMethods(provider, []string{method})` returns nil if and only if the method exists in the valid methods set for that provider. Returns ErrInvalidEnabledMethods for any method not in the valid set.
    - **Validates: Requirements 1.8**

  - [x] 4.4 Write property test: Midtrans Webhook Signature Verification (Property 4)
    - **Property 4: Midtrans Webhook Signature Verification**
    - In `services/billing-api/internal/gateway/midtrans_test.go`, use `rapid.Check` to verify that for any combination of order_id, status_code, gross_amount, and server_key strings, the Midtrans `VerifyWebhookSignature` function returns true when the provided signature equals SHA512(order_id + status_code + gross_amount + server_key), and returns false for any other signature value.
    - **Validates: Requirements 6.2**

- [x] 5. Checkpoint -- Domain + adapter layer complete
  - Ensure all domain and gateway adapter files compile (`go build ./...` in `services/billing-api`). Ensure property tests pass. Ask the user if questions arise.

- [x] 6. sqlc queries -- gateway tables
  - [x] 6.1 Create queries/payment_gateway_configs.sql
    - Create `services/billing-api/queries/payment_gateway_configs.sql` with sqlc queries: `CreateGatewayConfig` (:one, INSERT RETURNING *), `GetGatewayConfigByID` (:one, SELECT), `UpdateGatewayConfig` (:one, UPDATE SET ... RETURNING *), `DeactivateGatewayConfig` (:exec, UPDATE SET is_active=false, updated_at=NOW()), `ListGatewayConfigsByTenant` (:many, SELECT WHERE tenant_id=$1), `GetActiveGatewayConfigsByTenant` (:many, SELECT WHERE tenant_id=$1 AND is_active=true), `GetActiveGatewayConfigByProvider` (:one, SELECT WHERE tenant_id=$1 AND gateway_provider=$2 AND is_active=true), `ExistsGatewayConfigByProvider` (:one, SELECT EXISTS WHERE tenant_id=$1 AND gateway_provider=$2 AND is_active=true)
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.9_

  - [x] 6.2 Create queries/payment_links.sql
    - Create `services/billing-api/queries/payment_links.sql` with sqlc queries: `CreatePaymentLink` (:one, INSERT RETURNING *), `GetPaymentLinkByID` (:one, SELECT), `GetPaymentLinkByExternalID` (:one, SELECT WHERE external_id=$1), `GetActivePaymentLinkByCustomer` (:one, SELECT WHERE customer_id=$1 AND status='active'), `UpdatePaymentLinkStatus` (:exec, UPDATE SET status=$2, updated_at=NOW()), `UpdatePaymentLinkPaid` (:exec, UPDATE SET status='paid', paid_method=$2, paid_at=$3, updated_at=NOW()), `ListPaymentLinksByInvoice` (:many, SELECT pl.* FROM payment_links pl JOIN payment_link_invoices pli ON pl.id=pli.payment_link_id WHERE pli.invoice_id=$1 ORDER BY pl.created_at DESC), `FindExpiredPaymentLinks` (:many, SELECT WHERE status='active' AND expires_at < NOW() LIMIT $1), `ExpirePaymentLinkByID` (:exec, UPDATE SET status='expired', updated_at=NOW() WHERE id=$1 AND status='active'), `CreatePaymentLinkInvoice` (:exec, INSERT INTO payment_link_invoices), `GetInvoiceIDsByPaymentLinkID` (:many, SELECT invoice_id FROM payment_link_invoices WHERE payment_link_id=$1)
    - _Requirements: 2.2, 3.2, 3.4, 3.5, 4.1, 4.4, 14.1_

  - [x] 6.3 Create queries/webhook_logs.sql
    - Create `services/billing-api/queries/webhook_logs.sql` with sqlc queries: `CreateWebhookLog` (:one, INSERT RETURNING *), `GetWebhookLogByID` (:one, SELECT), `UpdateWebhookLogStatus` (:exec, UPDATE SET processing_status=$2, error_message=$3), `UpdateWebhookLogSignatureValid` (:exec, UPDATE SET signature_valid=$2), `IsWebhookAlreadyProcessed` (:one, SELECT EXISTS WHERE external_id=$1 AND event_type=$2 AND processing_status='processed'), `ListWebhookLogsByExternalID` (:many, SELECT WHERE external_id=$1 ORDER BY created_at DESC), `DeleteWebhookLogsOlderThan` (:execrows, DELETE WHERE created_at < $1 AND processing_status != 'failed' AND (signature_valid IS NULL OR signature_valid = true))
    - _Requirements: 5.3, 5.4, 8.1, 8.2, 14.3, 14.4, 16.2, 16.3_

  - [x] 6.4 Run sqlc generate to produce Go code
    - Run `sqlc generate` in `services/billing-api/` to regenerate repository files (adds payment_gateway_configs.sql.go, payment_links.sql.go, webhook_logs.sql.go, updates models.go)
    - Verify generated code compiles
    - _Requirements: 1.1, 2.2, 5.4_

- [x] 7. Repository implementations
  - [x] 7.1 Create repository/gateway_config_repo.go
    - Create `services/billing-api/internal/repository/gateway_config_repo.go` implementing `domain.GatewayConfigRepository` -- wraps sqlc-generated queries. Constructor `NewGatewayConfigRepo(queries, pool)`. All methods delegate to sqlc queries with appropriate type mapping between sqlc models and domain types.
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.9, 1.10_

  - [x] 7.2 Create repository/payment_link_repo.go
    - Create `services/billing-api/internal/repository/payment_link_repo.go` implementing `domain.PaymentLinkRepository` -- wraps sqlc-generated queries. Constructor `NewPaymentLinkRepo(queries, pool)`. `Create` method inserts payment_link row AND payment_link_invoices junction rows in a transaction. Other methods delegate to sqlc queries.
    - _Requirements: 2.2, 3.2, 3.4, 3.5, 4.1, 4.4, 14.1_

  - [x] 7.3 Create repository/webhook_log_repo.go
    - Create `services/billing-api/internal/repository/webhook_log_repo.go` implementing `domain.WebhookLogRepository` -- wraps sqlc-generated queries. Constructor `NewWebhookLogRepo(queries)`. `DeleteOlderThan` returns count of deleted rows. `IsAlreadyProcessed` returns bool from EXISTS query.
    - _Requirements: 5.3, 5.4, 8.1, 8.2, 14.3, 14.4, 16.2, 16.3_

- [x] 8. Checkpoint -- Data layer complete
  - Ensure all repository files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 9. Usecase layer
  - [x] 9.1 Create usecase/gateway_usecase.go with GatewayUsecase struct and config management methods
    - Create `services/billing-api/internal/usecase/gateway_usecase.go` with: `GatewayUsecase` struct (configRepo, linkRepo, invoiceRepo, customerRepo, settingsRepo, pool, queueClient, masterKey []byte, logger), constructor `NewGatewayUsecase(...)`, methods: `CreateConfig(ctx, tenantID, req) (*GatewayConfig, error)` -- validate enabled_methods via ValidateEnabledMethods, check duplicate via ExistsByProvider, encrypt API key and webhook secret via EncryptAESGCM, create config. `UpdateConfig(ctx, id, req) (*GatewayConfig, error)` -- get existing, encrypt new keys if provided, update. `DeactivateConfig(ctx, id) error` -- soft delete. `ListConfigs(ctx, tenantID) ([]*GatewayConfig, error)` -- list all, mask API keys via MaskAPIKey. `TestConfig(ctx, id) (*GatewayTestResult, error)` -- decrypt API key, create adapter, call TestConnection with 10s timeout
    - _Requirements: 1.2, 1.3, 1.4, 1.6, 1.7, 1.8, 1.9, 18.1, 18.2, 18.3_

  - [x] 9.2 Create usecase/gateway_link.go with payment link generation and management methods
    - Create `services/billing-api/internal/usecase/gateway_link.go` with methods on GatewayUsecase: `GeneratePaymentLink(ctx, req GeneratePaymentLinkRequest) (*PaymentLink, error)` -- get active gateway config, decrypt API key, get open invoices for customer, calculate total remaining amount, create adapter, call CreatePaymentLink, store payment_link + junction rows. `GetCustomerPaymentLink(ctx, customerID) (*CustomerPaymentLinkResponse, error)` -- get active link for customer, if exists return with invoices and total_arrears, if not return nil. `RegeneratePaymentLink(ctx, customerID) (*PaymentLink, error)` -- expire existing active link (if any) via adapter.ExpirePaymentLink + repo.ExpireByID, then generate new link. `GetInvoicePaymentLinks(ctx, invoiceID) ([]*PaymentLink, error)` -- list all links for invoice. `GetWalledGardenPaymentInfo(ctx, customerID) (*WalledGardenPaymentInfo, error)` -- get active link (generate on-demand if none/expired), return payment URL + arrears info. `SyncPaymentLinkAmount(ctx, invoiceID) error` -- find active link covering this invoice, expire it, generate new link with updated amount (triggered by invoice.payment_recorded or invoice.penalty_added events)
    - _Requirements: 2.1, 2.3, 2.4, 2.5, 2.6, 2.7, 3.1, 3.4, 3.5, 3.6, 4.2, 4.3, 4.5, 13.1, 13.2, 13.3, 14.1, 14.2, 17.1, 17.2, 17.3, 17.4_

  - [x] 9.3 Create usecase/webhook_usecase.go with WebhookUsecase struct and webhook processing
    - Create `services/billing-api/internal/usecase/webhook_usecase.go` with: `WebhookUsecase` struct (webhookRepo, linkRepo, invoiceRepo, paymentRepo, auditRepo, receiptSeqRepo, customerRepo, configRepo, pool, queueClient, masterKey, logger), constructor `NewWebhookUsecase(...)`, method: `ProcessWebhook(ctx, webhookLogID) error` -- fetch webhook log by ID, lookup payment link by external_id (if not found: update log status=failed with error_message="payment_link_not_found", return nil), identify tenant from payment link, get gateway config, decrypt webhook secret, create adapter, call VerifyWebhookSignature (if invalid: update log signature_valid=false, status=failed, return nil), call ParseWebhookPayload, check duplicate via IsAlreadyProcessed (if duplicate: update log status=duplicate, return nil), acquire pg_advisory_xact_lock on payment link ID, dispatch to processPaymentPaid/processPaymentExpired/processPaymentFailed based on event type, update webhook log status=processed
    - _Requirements: 5.5, 6.1, 6.2, 6.3, 6.4, 6.5, 7.1, 7.2, 7.3, 8.1, 8.2, 8.3, 8.4_

  - [x] 9.4 Create usecase/webhook_payment.go with payment processing methods
    - Create `services/billing-api/internal/usecase/webhook_payment.go` with methods on WebhookUsecase: `processPaymentPaid(ctx, event, link) error` -- BEGIN tx, get invoice IDs from link, SELECT invoices FOR UPDATE, call AllocatePaymentFIFO, for each allocation: INSERT invoice_payment (payment_method = event.PaidMethod, receipt via ReceiptSequenceRepository + FormatReceiptNumber, receipt_group_id for multi-invoice), UPDATE invoice paid_amount + status (optimistic locking via version), INSERT invoice_audit_log (action: invoice.payment_online). If invoice already lunas (double payment): add full amount to customer credit_balance, publish payment.double_payment event, INSERT audit log (invoice.double_payment_detected). If excess > 0: UPDATE customer credit_balance += excess. UPDATE payment_link status=paid. COMMIT. Publish payment.online.received event (for un-isolir). Publish payment.online.confirmation event (for notification). `processPaymentExpired(ctx, event, link) error` -- update payment_link status=expired, do NOT change invoice status. `processPaymentFailed(ctx, event, link) error` -- log failure, publish payment.online.failed event for customer notification, do NOT change invoice status
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8, 9.9, 10.1, 10.2, 10.3, 11.1, 11.2, 11.3, 12.1, 12.2, 12.3, 12.4, 12.5_

- [x] 10. Checkpoint -- Usecase layer complete
  - Ensure all usecase files compile (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 11. Handler layer
  - [x] 11.1 Create handler/gateway_handler.go with GatewayHandler struct and all endpoint methods
    - Create `services/billing-api/internal/handler/gateway_handler.go` with: `GatewayHandler` struct (gatewayUsecase, logger), constructor `NewGatewayHandler(gatewayUsecase, logger)`, methods:
    - `CreateConfig(c *fiber.Ctx) error` -- parse body, validate, extract tenantID from context, call usecase.CreateConfig, return 201
    - `ListConfigs(c *fiber.Ctx) error` -- extract tenantID, call usecase.ListConfigs, return 200
    - `UpdateConfig(c *fiber.Ctx) error` -- parse :id param + body, validate, call usecase.UpdateConfig, return 200
    - `DeactivateConfig(c *fiber.Ctx) error` -- parse :id param, call usecase.DeactivateConfig, return 200
    - `TestConfig(c *fiber.Ctx) error` -- parse :id param, call usecase.TestConfig, return 200
    - `GetCustomerPaymentLink(c *fiber.Ctx) error` -- parse :id (customer_id), call usecase.GetCustomerPaymentLink, return 200
    - `RegeneratePaymentLink(c *fiber.Ctx) error` -- parse :id (customer_id), call usecase.RegeneratePaymentLink, return 200
    - `GetInvoicePaymentLinks(c *fiber.Ctx) error` -- parse :id (invoice_id), call usecase.GetInvoicePaymentLinks, return 200
    - `GetPaymentLinkWebhooks(c *fiber.Ctx) error` -- parse :id (payment_link_id), call webhookRepo.ListByPaymentLink, return 200
    - `WalledGardenPaymentInfo(c *fiber.Ctx) error` -- parse :customer_id, call usecase.GetWalledGardenPaymentInfo, return 200
    - Include `mapGatewayError` helper mapping domain errors to HTTP responses
    - _Requirements: 1.2, 1.3, 1.4, 1.9, 3.4, 3.6, 13.1, 13.4, 14.1, 14.3, 18.1_

  - [x] 11.2 Create handler/webhook_handler.go with WebhookHandler struct and public webhook endpoints
    - Create `services/billing-api/internal/handler/webhook_handler.go` with: `WebhookHandler` struct (webhookLogRepo, queueClient, xenditIPs []string, midtransIPs []string, logger), constructor `NewWebhookHandler(webhookLogRepo, queueClient, xenditIPs, midtransIPs, logger)`, methods:
    - `HandleXendit(c *fiber.Ctx) error` -- check source IP against xenditIPs whitelist (if whitelist not empty), if IP not whitelisted: log with status=failed + error_message="ip_not_whitelisted", return 403. Parse body as JSON. Extract external_id and event_type from Xendit payload. INSERT webhook_log (gateway_provider=xendit, processing_status=received). Enqueue TaskProcessWebhook task. Return 200 immediately.
    - `HandleMidtrans(c *fiber.Ctx) error` -- same pattern as HandleXendit but for Midtrans (check midtransIPs, gateway_provider=midtrans, extract order_id as external_id, map transaction_status to event_type)
    - `checkIPWhitelist(sourceIP string, whitelist []string) bool` helper
    - Note: These endpoints are PUBLIC (no auth middleware). Security via IP whitelist + signature verification (signature verified async in webhook usecase)
    - _Requirements: 5.1, 5.2, 5.3, 5.5, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 12. Worker layer
  - [x] 12.1 Create worker/gateway_worker.go with GatewayWorker and async task handlers
    - Create `services/billing-api/internal/worker/gateway_worker.go` with: task type constants (TaskGeneratePaymentLink = "gateway.generate_payment_link", TaskProcessWebhook = "gateway.process_webhook", TaskExpirePaymentLinks = "gateway.expire_payment_links", TaskCleanupWebhookLogs = "gateway.cleanup_webhook_logs", TaskSyncPaymentLinkAmount = "gateway.sync_payment_link_amount"), `GatewayWorker` struct (gatewayUsecase, webhookUsecase, linkRepo, webhookRepo, logger), constructor `NewGatewayWorker(...)`, `RegisterHandlers(mux *asynq.ServeMux)` method
    - Handler methods: `handleGeneratePaymentLink` -- deserialize GeneratePaymentLinkRequest from task payload, call gatewayUsecase.GeneratePaymentLink. `handleProcessWebhook` -- deserialize webhookLogID from payload, call webhookUsecase.ProcessWebhook. `handleExpirePaymentLinks` -- call linkRepo.FindExpired(batchSize=100), for each: call linkRepo.ExpireByID. `handleCleanupWebhookLogs` -- calculate retention cutoff (90 days default from config), call webhookRepo.DeleteOlderThan. `handleSyncPaymentLinkAmount` -- deserialize invoiceID from payload, call gatewayUsecase.SyncPaymentLinkAmount
    - _Requirements: 2.6, 2.7, 4.1, 4.4, 5.5, 16.1, 16.2, 16.3, 17.1, 17.2, 17.3, 17.4_

- [x] 13. Router wiring and main.go dependency injection
  - [x] 13.1 Update handler/router.go with gateway and webhook routes
    - Modify `services/billing-api/internal/handler/router.go`: add `GatewayHandler *GatewayHandler` and `WebhookHandler *WebhookHandler` to `RouterConfig` struct
    - Register PUBLIC webhook routes (no auth): `POST /webhooks/xendit` (HandleXendit), `POST /webhooks/midtrans` (HandleMidtrans)
    - Register PUBLIC walled garden route (no auth, rate-limited): `GET /api/v1/public/walled-garden/:customer_id/payment-info` (WalledGardenPaymentInfo)
    - Register gateway config routes under settings (auth + tenant + RBAC, tenant_admin only): `POST /v1/settings/payment-gateways`, `GET /v1/settings/payment-gateways`, `PUT /v1/settings/payment-gateways/:id`, `DELETE /v1/settings/payment-gateways/:id`, `POST /v1/settings/payment-gateways/:id/test`
    - Register payment link routes under existing customer routes: `GET /v1/customers/:id/payment-link` (admin+kasir read), `POST /v1/customers/:id/payment-link/regenerate` (admin+operator write)
    - Register payment status query routes: `GET /v1/invoices/:id/payment-links` (existing invoicesRead group), `GET /v1/payment-links/:id/webhooks` (new group, admin+kasir)
    - _Requirements: 1.2, 3.4, 3.6, 5.1, 5.2, 13.1, 13.4, 13.5, 14.1, 14.3_

  - [x] 13.2 Update config/config.go with gateway configuration fields
    - Modify `services/billing-api/internal/config/config.go`: add fields to AppConfig: `GatewayMasterKey string` (mapstructure:"GATEWAY_MASTER_KEY"), `XenditWebhookIPs string` (mapstructure:"XENDIT_WEBHOOK_IPS"), `MidtransWebhookIPs string` (mapstructure:"MIDTRANS_WEBHOOK_IPS"), `WebhookLogRetentionDays int` (mapstructure:"WEBHOOK_LOG_RETENTION_DAYS", default 90). Add GATEWAY_MASTER_KEY to required validation (must be 64 hex chars = 32 bytes). Add helper method `ParseWebhookIPs() (xenditIPs, midtransIPs []string)` to split comma-separated IP strings. Add helper `MasterKeyBytes() ([]byte, error)` to decode hex string to 32 bytes
    - _Requirements: 1.6, 7.1, 7.2, 7.4, 16.2_

  - [x] 13.3 Update cmd/main.go to wire gateway dependencies
    - Modify `services/billing-api/cmd/main.go`: parse master key bytes from config (`cfg.MasterKeyBytes()`), parse webhook IPs (`cfg.ParseWebhookIPs()`). Instantiate repos: `gatewayConfigRepo := repository.NewGatewayConfigRepo(queries, dbPool)`, `paymentLinkRepo := repository.NewPaymentLinkRepo(queries, dbPool)`, `webhookLogRepo := repository.NewWebhookLogRepo(queries)`. Instantiate usecases: `gatewayUsecase := usecase.NewGatewayUsecase(gatewayConfigRepo, paymentLinkRepo, invoiceRepo, customerRepo, billingSettingsRepo, dbPool, queueClient, masterKeyBytes, appLogger)`, `webhookUsecase := usecase.NewWebhookUsecase(webhookLogRepo, paymentLinkRepo, invoiceRepo, invoicePaymentRepo, invoiceAuditLogRepo, receiptSequenceRepo, customerRepo, gatewayConfigRepo, dbPool, queueClient, masterKeyBytes, appLogger)`. Instantiate handlers: `gatewayHandler := handler.NewGatewayHandler(gatewayUsecase, appLogger)`, `webhookHandler := handler.NewWebhookHandler(webhookLogRepo, queueClient, xenditIPs, midtransIPs, appLogger)`. Add to RouterConfig: `GatewayHandler: gatewayHandler`, `WebhookHandler: webhookHandler`. Instantiate worker: `gatewayWorker := worker.NewGatewayWorker(gatewayUsecase, webhookUsecase, paymentLinkRepo, webhookLogRepo, appLogger)`, call `gatewayWorker.RegisterHandlers(mux)`. Register cron jobs: TaskExpirePaymentLinks every hour ("0 * * * *"), TaskCleanupWebhookLogs daily at 02:00 ("0 2 * * *")
    - _Requirements: 1.1, 2.7, 4.4, 5.5, 16.2_

  - [x] 13.4 Add event publishing to existing payment-manual module
    - Modify `services/billing-api/internal/usecase/payment_multi.go`: after COMMIT in `RecordMultiPayment`, publish `invoice.payment_recorded` event to asynq queue with payload `{"tenant_id": tenantID, "invoice_id": invoiceID, "customer_id": customerID}` for each invoice that received an allocation. This triggers `SyncPaymentLinkAmount` in the gateway worker to expire stale payment links.
    - Modify `services/billing-api/internal/usecase/payment_void.go`: after COMMIT in `VoidPayment`, publish `invoice.payment_recorded` event (same payload) to trigger payment link sync for the affected invoice.
    - Note: `invoice.penalty_added` event will be published by the invoice cron module when late fees are added (future task in isolir-system spec). For now, the gateway worker handler for this event can be a no-op stub.
    - _Requirements: 17.4, 17.5, 17.6_

- [x] 14. Checkpoint -- Full module compiles and routes registered
  - Ensure the full service compiles (`go build ./...` in `services/billing-api`). Ask the user if questions arise.

- [x] 15. Unit tests
  - [x] 15.1 Write unit tests for GatewayHandler
    - In `services/billing-api/internal/handler/gateway_handler_test.go`, test HTTP status codes, request parsing, response format for: CreateConfig (201 success, 400 validation, 409 duplicate provider), ListConfigs (200 with masked keys), UpdateConfig (200 success, 404 not found), DeactivateConfig (200 success, 404 not found), TestConfig (200 success/failure response), GetCustomerPaymentLink (200 with link, 200 with null if no link), RegeneratePaymentLink (200 success), GetInvoicePaymentLinks (200 list), WalledGardenPaymentInfo (200 public endpoint, 429 rate limited)
    - _Requirements: 1.2, 1.3, 1.4, 1.9, 3.4, 3.6, 13.1, 13.4, 13.5, 14.1, 18.1, 18.2_

  - [x] 15.2 Write unit tests for WebhookHandler
    - In `services/billing-api/internal/handler/webhook_handler_test.go`, test: HandleXendit with valid IP (200 + log created + task enqueued), HandleXendit with invalid IP (403), HandleMidtrans with valid IP (200), HandleMidtrans with invalid IP (403), empty whitelist skips IP check (200), correct extraction of external_id and event_type from both Xendit and Midtrans payloads
    - _Requirements: 5.1, 5.2, 5.3, 5.5, 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x] 15.3 Write unit tests for GatewayUsecase -- config management
    - In `services/billing-api/internal/usecase/gateway_usecase_test.go`, test: CreateConfig with valid input (encrypts keys, stores config), CreateConfig with invalid methods (returns ErrInvalidEnabledMethods), CreateConfig duplicate provider (returns ErrGatewayConfigDuplicate), UpdateConfig partial update (only updates provided fields), DeactivateConfig success, ListConfigs returns masked keys, TestConfig success and failure scenarios
    - _Requirements: 1.2, 1.4, 1.6, 1.7, 1.8, 1.9, 18.1, 18.2, 18.3_

  - [x] 15.4 Write unit tests for GatewayUsecase -- payment link management
    - In `services/billing-api/internal/usecase/gateway_link_test.go`, test: GeneratePaymentLink single invoice (creates link with correct amount and expiry), GeneratePaymentLink multi-invoice (sums remaining amounts), GetCustomerPaymentLink returns existing active link, GetCustomerPaymentLink returns nil when no active link, RegeneratePaymentLink expires old and creates new, WalledGardenPaymentInfo generates on-demand when no active link, WalledGardenPaymentInfo regenerates when link expired, SyncPaymentLinkAmount expires and regenerates with updated amount, GeneratePaymentLink skips lunas/batal invoices
    - _Requirements: 2.1, 2.3, 2.4, 2.6, 3.1, 3.4, 3.5, 3.6, 4.2, 4.3, 4.5, 13.1, 13.2, 13.3, 17.1, 17.2, 17.3_

  - [x] 15.5 Write unit tests for WebhookUsecase -- webhook processing
    - In `services/billing-api/internal/usecase/webhook_usecase_test.go`, test: ProcessWebhook with external_id not found (marks failed with "payment_link_not_found", no payment), ProcessWebhook with invalid signature (marks failed, no payment), ProcessWebhook duplicate (marks duplicate, no processing), ProcessWebhook payment.paid single invoice (records payment, updates invoice to lunas, generates receipt, publishes events), ProcessWebhook payment.paid multi-invoice (FIFO allocation across invoices), ProcessWebhook payment.paid with excess (adds to credit_balance), ProcessWebhook payment.paid double payment (invoice already lunas, full amount to credit, publishes double_payment event), ProcessWebhook payment.expired (updates link status, does NOT change invoice), ProcessWebhook payment.failed (logs failure, publishes notification event, does NOT change invoice)
    - _Requirements: 6.1, 6.2, 6.3, 6.5, 8.1, 8.2, 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 9.8, 9.9, 10.1, 10.2, 10.3, 11.1, 11.2, 11.3, 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 15.6 Write unit tests for GatewayWorker
    - In `services/billing-api/internal/worker/gateway_worker_test.go`, test: handleGeneratePaymentLink deserializes payload and calls usecase, handleProcessWebhook deserializes webhookLogID and calls usecase, handleExpirePaymentLinks finds and expires batch, handleCleanupWebhookLogs calls DeleteOlderThan with correct retention, handleSyncPaymentLinkAmount deserializes invoiceID and calls usecase
    - _Requirements: 2.6, 2.7, 4.4, 5.5, 16.2, 17.4_

- [x] 16. Final checkpoint -- All tests pass
  - Ensure all tests pass (`go test ./...` in `services/billing-api`). Ensure full service compiles. Ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document
- Unit tests validate specific examples, edge cases, and integration points
- The module reuses existing infrastructure: AllocatePaymentFIFO, FormatReceiptNumber, ReceiptSequenceRepository, InvoicePaymentRepository
- Webhook endpoints are PUBLIC (no auth middleware) -- security via IP whitelist + signature verification
- All monetary values stored as BIGINT (Rupiah)
- Max 200 lines per file -- split into multiple files if needed
- All code comments in Indonesian
