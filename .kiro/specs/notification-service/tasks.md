# Implementation Plan: Notification Service

## Overview

Bottom-up implementation of the Notification Service — a separate Go service (`services/notification/`) that handles all notification delivery to ISP customers via WhatsApp, SMS, and Email. The plan starts with database migrations, builds up domain entities and pure functions with property tests, then layers sqlc queries, repository, provider adapters, usecase (delivery pipeline), handler, and worker components. Each task builds on the previous one, ensuring no orphaned code.

## Tasks

- [x] 1. Database migration for notification tables
  - [x] 1.1 Create `migrations/000001_create_notification_tables.up.sql`
    - Create `notification_configs` table with columns: id (UUID PK), tenant_id (UUID NOT NULL), channel (VARCHAR NOT NULL), provider (VARCHAR NOT NULL), credentials (JSONB NOT NULL), is_enabled (BOOLEAN NOT NULL DEFAULT false), priority (INTEGER NOT NULL DEFAULT 1), settings (JSONB DEFAULT '{}'), created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - Add CHECK constraint on channel: IN ('whatsapp', 'sms', 'email')
    - Add UNIQUE constraint on (tenant_id, channel)
    - Enable RLS with tenant isolation policies (SELECT, INSERT, UPDATE, DELETE)
    - Create index on (tenant_id, is_enabled)
    - Create `notification_templates` table with columns: id (UUID PK), tenant_id (UUID NOT NULL), slug (VARCHAR NOT NULL), name (VARCHAR NOT NULL), category (VARCHAR NOT NULL), event_type (VARCHAR), channels (JSONB NOT NULL DEFAULT '[]'), body_whatsapp (TEXT), body_sms (TEXT), body_email_subject (TEXT), body_email_html (TEXT), variables (JSONB NOT NULL DEFAULT '[]'), is_active (BOOLEAN NOT NULL DEFAULT true), is_default (BOOLEAN NOT NULL DEFAULT false), created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - Add CHECK constraint on category: IN ('transactional', 'reminder', 'promotion', 'information')
    - Add UNIQUE constraint on (tenant_id, slug)
    - Enable RLS with tenant isolation policies
    - Create indexes on (tenant_id, event_type) and (tenant_id, is_active)
    - Create `notification_logs` table with columns: id (UUID PK), tenant_id (UUID NOT NULL), customer_id (UUID NOT NULL), template_id (UUID REFERENCES notification_templates(id)), channel (VARCHAR NOT NULL), provider (VARCHAR NOT NULL), recipient (VARCHAR NOT NULL), subject (TEXT), body (TEXT NOT NULL), status (VARCHAR NOT NULL DEFAULT 'pending'), retry_count (INTEGER NOT NULL DEFAULT 0), max_retries (INTEGER NOT NULL DEFAULT 3), error_message (TEXT), dedup_key (VARCHAR), metadata (JSONB DEFAULT '{}'), sent_at (TIMESTAMPTZ), created_at (TIMESTAMPTZ NOT NULL DEFAULT NOW()), updated_at (TIMESTAMPTZ NOT NULL DEFAULT NOW())
    - Add CHECK constraints on status and channel
    - Enable RLS with tenant isolation policies
    - Create indexes on (tenant_id, customer_id), (tenant_id, status), (tenant_id, created_at DESC), (dedup_key)
    - Create partial unique index on dedup_key for deduplication within 1 hour
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

  - [x] 1.2 Create `migrations/000001_create_notification_tables.down.sql`
    - Drop notification_logs, notification_templates, notification_configs tables in order
    - _Requirements: 1.1, 2.1, 3.1_

- [x] 2. Domain entities, constants, errors, and template engine
  - [x] 2.1 Create `internal/domain/constants.go`
    - Define Channel type and constants: ChannelWhatsApp, ChannelSMS, ChannelEmail
    - Define LogStatus type and constants: StatusPending, StatusSending, StatusSent, StatusDelivered, StatusRead, StatusFailed, StatusRetrying, StatusSkipped
    - Define TemplateCategory type and constants: CategoryTransactional, CategoryReminder, CategoryPromotion, CategoryInformation
    - Define bypass event types list for quiet hours and throttle exemptions
    - All comments in Indonesian, max 200 lines
    - _Requirements: 18.4, 18.5, 18.6_

  - [x] 2.2 Create `internal/domain/notification.go`
    - Define NotificationConfig struct with all fields including ConfigSettings, WhatsAppCredentials, SMSCredentials, EmailCredentials
    - Define NotificationTemplate struct with all fields
    - Define NotificationLog struct with all fields including joined fields (CustomerName, TemplateName)
    - Define LogListParams and LogListResult structs for pagination
    - All comments in Indonesian, max 200 lines
    - _Requirements: 18.1, 18.2, 18.3_

  - [x] 2.3 Create `internal/domain/errors.go`
    - Define domain errors: ErrTemplateNotFound, ErrTemplateSlugExists, ErrTemplateNotDeletable, ErrConfigNotFound, ErrProviderNotConfigured, ErrCustomerNotFound, ErrLogNotFound, ErrNotResendable, ErrInvalidCredentials, ErrDailyLimitExceeded, ErrInvalidTimezone, ErrInvalidQuietHours
    - All comments in Indonesian
    - _Requirements: 18.1_

  - [x] 2.4 Create `internal/domain/dto.go`
    - Define request/response DTOs: UpdateConfigRequest, CreateTemplateRequest, UpdateTemplateRequest, TestSendRequest, ManualSendRequest, ResendRequest
    - Define API response helpers (reuse pattern from billing-api)
    - All comments in Indonesian, max 200 lines
    - _Requirements: 12.1, 13.1, 14.1, 15.1, 16.1, 17.1_

  - [x] 2.5 Create `internal/domain/repository.go`
    - Define ConfigRepository interface: GetByTenant, GetByTenantAndChannel, Upsert, GetSettings, UpdateSettings
    - Define TemplateRepository interface: Create, GetByID, GetBySlug, GetByEventType, Update, SoftDelete, ListByTenant, BulkCreate, SlugExists
    - Define LogRepository interface: Create, GetByID, Update, List, FindByDedupKey, CountTodayByCustomer, LastSentToCustomer
    - All comments in Indonesian, max 200 lines
    - _Requirements: 1.1, 2.1, 3.1_

  - [x] 2.6 Create `internal/domain/provider.go`
    - Define WhatsAppProvider interface with Send method
    - Define SMSProvider interface with Send method
    - Define EmailProvider interface with Send method
    - Define WhatsAppMessage, SMSMessage, EmailMessage, SendResult structs
    - All comments in Indonesian, max 100 lines
    - _Requirements: 6.1, 6.2, 6.3, 6.7_

  - [x] 2.7 Create `internal/domain/template_engine.go`
    - Implement TemplateEngine struct with Render and ExtractVariables methods
    - Implement FormatMoney(amount int64) string — format to "Rp 388.500"
    - Implement FormatDateID(t time.Time) string — format to "5 April 2026"
    - Implement credential masking helper: MaskCredential(value string) string
    - Implement validation helpers: ValidateQuietHours, ValidateTimezone, ValidateSettings, ValidateTemplateBody, ValidateCredentials, NormalizePageSize
    - All comments in Indonesian, max 200 lines
    - _Requirements: 5.1, 5.2, 5.3, 5.5, 5.6, 13.6, 20.3, 20.4, 20.5, 20.6_

  - [x] 2.8 Write property tests for template engine (`internal/domain/template_engine_test.go`)
    - **Property 1: Template rendering completeness** — after rendering, output contains no `{variable}` patterns
    - **Property 2: Template render idempotence** — render(render(body, data), data) == render(body, data)
    - **Validates: Requirements 5.1, 5.3, 5.4**

  - [x] 2.9 Write property tests for FormatMoney and FormatDateID (`internal/domain/format_test.go`)
    - **Property 3: FormatMoney round-trip** — parsing back the numeric part yields original amount
    - **Property 4: FormatDateID contains valid Indonesian month** — output contains one of 12 Indonesian month names
    - **Validates: Requirements 5.5, 5.6**

  - [x] 2.10 Write property tests for validation helpers (`internal/domain/validation_test.go`)
    - **Property 13: Credential masking preserves last 4 characters** — masked output ends with last 4 chars of original
    - **Property 14: Config validation — credentials required when enabled** — is_enabled=true with empty fields fails
    - **Property 15: Template validation — at least one channel body required** — all empty bodies fails
    - **Property 16: Settings range validation** — daily_limit in [1,20] passes, outside fails; cooldown in [5,120] passes, outside fails
    - **Property 17: Quiet hours time validation** — start before end passes, start >= end fails
    - **Property 18: Page size normalization** — values in {10,25,50} pass through, others default to 25
    - **Validates: Requirements 13.6, 13.3, 14.5, 20.3, 20.5, 20.6, 12.3**

  - [x] 2.11 Create `internal/domain/seed.go`
    - Define DefaultTemplates slice with all default templates: invoice_new, reminder_h1, payment_confirm, isolir_notice, suspend_notice, unblock_notice, reactivated_notice
    - Each template includes body_whatsapp and body_sms variants, variables list, category, event_type
    - All comments in Indonesian, max 200 lines
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5_

- [x] 3. Checkpoint — Domain layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. sqlc queries and code generation
  - [x] 4.1 Create `sqlc.yaml` at service root
    - Configure sqlc v2 with engine postgresql, queries "queries/", schema "migrations/", output "internal/repository", package "repository", sql_package "pgx/v5", emit_json_tags true, emit_empty_slices true
    - _Requirements: 1.1, 2.1, 3.1_

  - [x] 4.2 Create `queries/notification_configs.sql`
    - Write queries: GetConfigsByTenant, GetConfigByTenantAndChannel, UpsertConfig, GetSettingsByTenant, UpdateSettings
    - _Requirements: 1.1, 13.1, 13.2, 20.1_

  - [x] 4.3 Create `queries/notification_templates.sql`
    - Write queries: CreateTemplate, GetTemplateByID, GetTemplateBySlug, GetTemplateByEventType, UpdateTemplate, SoftDeleteTemplate, ListTemplatesByTenant, BulkCreateTemplates, SlugExists
    - _Requirements: 2.1, 14.1, 14.2, 14.3, 14.4, 14.7_

  - [x] 4.4 Create `queries/notification_logs.sql`
    - Write queries: CreateLog, GetLogByID, UpdateLog, ListLogs (with filters: channel, status, customer_id, template_id, date_from, date_to + pagination), FindByDedupKey (within hours window), CountTodayByCustomer (with timezone), LastSentToCustomer
    - _Requirements: 3.1, 9.2, 11.1, 11.3, 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 4.5 Create `queries/customer_data.sql`
    - Write queries: GetCustomerByID (fetch nama, telepon, email, paket, id_pelanggan from shared customers table), GetTenantByID (fetch nama, telepon, timezone from shared tenants table)
    - _Requirements: 7.2_

  - [x] 4.6 Run `sqlc generate` to produce Go code
    - Execute sqlc generate in the notification service directory
    - _Requirements: 1.1, 2.1, 3.1_

- [x] 5. Repository implementations
  - [x] 5.1 Create `internal/repository/config_repo.go`
    - Implement ConfigRepository interface using generated sqlc code
    - Map between domain.NotificationConfig and sqlc-generated types
    - Handle JSONB marshaling for credentials and settings
    - Max 200 lines, comments in Indonesian
    - _Requirements: 1.1, 13.1, 13.2, 20.1_

  - [x] 5.2 Create `internal/repository/template_repo.go`
    - Implement TemplateRepository interface using generated sqlc code
    - Map between domain.NotificationTemplate and sqlc-generated types
    - Handle JSONB marshaling for channels and variables arrays
    - Max 200 lines, comments in Indonesian
    - _Requirements: 2.1, 14.1, 14.2, 14.3, 14.4, 14.7, 19.1_

  - [x] 5.3 Create `internal/repository/log_repo.go`
    - Implement LogRepository interface using generated sqlc code
    - Map between domain.NotificationLog and sqlc-generated types
    - Handle JSONB marshaling for metadata
    - Implement pagination logic for List method
    - Max 200 lines, comments in Indonesian
    - _Requirements: 3.1, 9.2, 11.1, 11.3, 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 5.4 Create `internal/repository/customer_repo.go`
    - Implement CustomerDataFetcher and TenantDataFetcher interfaces using generated sqlc code
    - Map to usecase.CustomerData and usecase.TenantData structs
    - Max 100 lines, comments in Indonesian
    - _Requirements: 7.2_

- [x] 6. Checkpoint — Repository layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Provider adapters
  - [x] 7.1 Create `internal/provider/fonnte.go`
    - Implement FonnteAdapter struct implementing WhatsAppProvider interface
    - Send WhatsApp message via Fonnte HTTP API (POST with api_token header)
    - Parse response to extract message_id and status
    - Handle HTTP errors and timeouts
    - Max 150 lines, comments in Indonesian
    - _Requirements: 6.1, 6.4, 6.7_

  - [x] 7.2 Create `internal/provider/zenziva.go`
    - Implement ZenzivaAdapter struct implementing SMSProvider interface
    - Send SMS via Zenziva HTTP API (POST with api_key and user_key)
    - Parse response to extract message_id and status
    - Handle HTTP errors and timeouts
    - Max 150 lines, comments in Indonesian
    - _Requirements: 6.2, 6.5, 6.7_

  - [x] 7.3 Create `internal/provider/smtp.go`
    - Implement SMTPAdapter struct implementing EmailProvider interface
    - Send email via net/smtp with configured host, port, username, password
    - Build MIME message with HTML body, From header, Subject
    - Handle connection errors and auth failures
    - Max 150 lines, comments in Indonesian
    - _Requirements: 6.3, 6.6, 6.7_

- [x] 8. Usecase layer (delivery pipeline)
  - [x] 8.1 Create `internal/usecase/dedup.go`
    - Implement DedupChecker struct with LogRepository dependency
    - Implement GenerateDedupKey(tenantID, customerID, templateSlug, periode string) string
    - Implement CheckDuplicate(ctx, dedupKey string) (bool, error) — check within 1-hour window
    - Max 80 lines, comments in Indonesian
    - _Requirements: 9.1, 9.2, 9.3, 9.4_

  - [x] 8.2 Write property test for dedup key format (`internal/usecase/dedup_test.go`)
    - **Property 5: Dedup key format consistency** — splitting by ":" yields exactly 4 original components
    - **Validates: Requirements 9.1**

  - [x] 8.3 Create `internal/usecase/quiet_hours.go`
    - Implement QuietHoursChecker struct
    - Implement IsQuietHours(now time.Time, tz, start, end string) bool — check if current time is outside active hours
    - Implement IsBypassEvent(eventType string) bool — check if event is in bypass list
    - Implement CalculateScheduledAt(tz, start string) time.Time — calculate next active period start
    - Max 100 lines, comments in Indonesian
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

  - [x] 8.4 Write property tests for quiet hours (`internal/usecase/quiet_hours_test.go`)
    - **Property 7: Quiet hours blocking** — time outside [start, end) returns true (is quiet)
    - **Property 8: Quiet hours bypass for exempt events** — bypass events always return false regardless of time
    - **Validates: Requirements 10.1, 10.4**

  - [x] 8.5 Create `internal/usecase/throttle.go`
    - Implement ThrottleChecker struct with LogRepository dependency
    - Implement CheckDailyLimit(ctx, tenantID, customerID, tz string, limit int) (bool, error) — check count today
    - Implement CheckCooldown(ctx, tenantID, customerID string, cooldownMinutes int) (bool, *time.Time, error) — check last sent time
    - Implement IsBypassEvent(eventType string) bool — check if event is in bypass list
    - Max 100 lines, comments in Indonesian
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_

  - [x] 8.6 Write property tests for throttle (`internal/usecase/throttle_test.go`)
    - **Property 9: Throttle daily limit enforcement** — count >= limit returns skip; count < limit proceeds
    - **Property 10: Throttle cooldown delay** — last_sent + cooldown > now returns delay; otherwise proceeds
    - **Property 11: Throttle bypass for exempt events** — bypass events always return allow
    - **Validates: Requirements 11.2, 11.4, 11.5**

  - [x] 8.7 Create `internal/usecase/delivery_pipeline.go`
    - Define DeliveryPipeline struct with all dependencies (configRepo, templateRepo, logRepo, customerRepo, tenantRepo, engine, providers, logger)
    - Implement NewDeliveryPipeline constructor
    - Implement ProcessEvent(ctx, envelope): resolve template by event_type → dedup check → quiet hours check → throttle check → render template → select channel → send via adapter → log result
    - Implement channel selection logic respecting priority order
    - Implement retry + fallback logic (3 attempts on primary, then fallback to next channel)
    - Max 200 lines, comments in Indonesian
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6, 7.7, 8.1, 8.2, 8.3, 8.4, 8.5, 8.6_

  - [x] 8.8 Write property test for channel selection (`internal/usecase/delivery_pipeline_test.go`)
    - **Property 12: Channel selection respects priority order** — selected channel is first in priority that is in template channels and has enabled config
    - **Validates: Requirements 7.4**

  - [x] 8.9 Write property test for deduplication invariant (`internal/usecase/dedup_test.go`)
    - **Property 6: Deduplication invariant** — for N notifications with same dedup_key in 1-hour window, only first is delivered
    - **Validates: Requirements 9.5**

  - [x] 8.10 Create `internal/usecase/send_operations.go`
    - Implement SendManual(ctx, req ManualSendRequest): resolve customer, render template or use custom body, respect throttle, send, log
    - Implement SendTest(ctx, req TestSendRequest): render with sample data, bypass guards, send, log with is_test metadata
    - Implement Resend(ctx, logID string): fetch original log, validate status=failed, bypass dedup, respect quiet hours + throttle, send, create new log with resend_of metadata
    - Max 200 lines, comments in Indonesian
    - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.5, 15.6, 16.1, 16.2, 16.3, 16.4, 16.5, 16.6, 17.1, 17.2, 17.3, 17.4, 17.5, 17.6_

- [x] 9. Checkpoint — Usecase layer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Handler layer
  - [x] 10.1 Create `internal/handler/log_handler.go`
    - Define LogHandler struct with LogRepository dependency
    - Implement List (GET /api/v1/notifications/logs) — parse query params (channel, status, customer_id, template_id, date_from, date_to, page, page_size), call repo, return paginated response
    - Implement GetByID (GET /api/v1/notifications/logs/:id) — fetch single log with details
    - Max 150 lines, comments in Indonesian
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 10.2 Create `internal/handler/config_handler.go`
    - Define ConfigHandler struct with ConfigRepository and TemplateRepository dependencies
    - Implement Get (GET /api/v1/notifications/config) — return config with masked credentials
    - Implement Update (PUT /api/v1/notifications/config) — validate request, upsert config, seed default templates on first config creation
    - Max 180 lines, comments in Indonesian
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 19.1, 20.1, 20.2, 20.3, 20.4, 20.5, 20.6_

  - [x] 10.3 Create `internal/handler/template_handler.go`
    - Define TemplateHandler struct with TemplateRepository dependency
    - Implement List (GET /api/v1/notifications/templates) — return all templates for tenant
    - Implement Create (POST /api/v1/notifications/templates) — validate, check slug uniqueness, create
    - Implement Update (PUT /api/v1/notifications/templates/:id) — validate, update
    - Implement Delete (DELETE /api/v1/notifications/templates/:id) — check is_default, soft-delete
    - Max 200 lines, comments in Indonesian
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7_

  - [x] 10.4 Create `internal/handler/send_handler.go`
    - Define SendHandler struct with DeliveryPipeline dependency
    - Implement TestSend (POST /api/v1/notifications/test) — validate, call pipeline.SendTest
    - Implement ManualSend (POST /api/v1/notifications/send) — validate, call pipeline.SendManual
    - Implement Resend (POST /api/v1/notifications/logs/:id/resend) — validate, call pipeline.Resend
    - Max 150 lines, comments in Indonesian
    - _Requirements: 15.1, 15.5, 15.6, 16.1, 16.6, 17.1, 17.3, 17.6_

- [x] 11. Worker layer (event consumer)
  - [x] 11.1 Create `internal/worker/event_consumer.go`
    - Define EventConsumer struct with DeliveryPipeline and logger dependencies
    - Implement NewEventConsumer constructor
    - Implement RegisterHandlers(mux *asynq.ServeMux) — register handlers for all event types: invoice.created, invoice.reminder, payment.online.received, payment.recorded, customer.isolir, customer.un_isolir, customer.suspend, notification.isolir, notification.un_isolir, notification.suspend, notification.reactivated, notification.pending_sync_failed, invoice.penalty_added
    - Implement handleEvent — decode TaskEnvelope, pass to pipeline.ProcessEvent, handle decode errors gracefully
    - Max 150 lines, comments in Indonesian
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 12. Router wiring and main.go DI
  - [x] 12.1 Update `internal/handler/router.go`
    - Add LogHandler, ConfigHandler, TemplateHandler, SendHandler fields to RouterConfig
    - Register notification route group under api (auth + tenant middleware):
      - GET /notifications/logs, GET /notifications/logs/:id
      - GET /notifications/config, PUT /notifications/config
      - GET /notifications/templates, POST /notifications/templates, PUT /notifications/templates/:id, DELETE /notifications/templates/:id
      - POST /notifications/test, POST /notifications/send, POST /notifications/logs/:id/resend
    - _Requirements: 12.1, 13.1, 14.1, 15.1, 16.1, 17.1_

  - [x] 12.2 Update `internal/config/config.go`
    - Add worker config fields: WorkerConcurrency (int, default 10), QueuePriorities (map)
    - Add Fonnte, Zenziva, SMTP default timeout config
    - _Requirements: 4.5_

  - [x] 12.3 Update `cmd/main.go`
    - Instantiate repositories: ConfigRepo, TemplateRepo, LogRepo, CustomerRepo
    - Instantiate provider adapters: FonnteAdapter, ZenzivaAdapter, SMTPAdapter
    - Instantiate TemplateEngine
    - Instantiate DeliveryPipeline with all dependencies
    - Instantiate handlers: LogHandler, ConfigHandler, TemplateHandler, SendHandler
    - Add handlers to RouterConfig
    - Instantiate asynq Server using pkg/queue.NewServer
    - Instantiate EventConsumer and call RegisterHandlers
    - Start asynq server in goroutine, add to graceful shutdown
    - _Requirements: 4.1, 4.5, 7.1_

- [x] 13. Unit tests for handlers and worker
  - [x] 13.1 Write unit tests for LogHandler (`internal/handler/log_handler_test.go`)
    - Test List: 200 with pagination, filters (channel, status, date range)
    - Test GetByID: 200 success, 404 not found
    - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.5_

  - [x] 13.2 Write unit tests for ConfigHandler (`internal/handler/config_handler_test.go`)
    - Test Get: 200 returns masked credentials
    - Test Update: 200 success, 422 validation errors (missing credentials, invalid timezone, invalid quiet hours, invalid settings range)
    - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.5, 13.6, 20.3, 20.4, 20.5, 20.6_

  - [x] 13.3 Write unit tests for TemplateHandler (`internal/handler/template_handler_test.go`)
    - Test List: 200 with templates
    - Test Create: 201 success, 409 slug exists, 422 no body provided
    - Test Update: 200 success, 404 not found
    - Test Delete: 200 success (custom), 422 not deletable (default)
    - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.5, 14.6, 14.7_

  - [x] 13.4 Write unit tests for SendHandler (`internal/handler/send_handler_test.go`)
    - Test TestSend: 200 success, 422 provider not configured
    - Test ManualSend: 200 success, 404 customer not found
    - Test Resend: 200 success, 404 log not found, 422 not resendable
    - _Requirements: 15.1, 15.5, 15.6, 16.1, 16.6, 17.1, 17.3, 17.6_

  - [x] 13.5 Write unit tests for EventConsumer (`internal/worker/event_consumer_test.go`)
    - Test handler registration for all event types
    - Test handleEvent: successful processing, invalid payload skip
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 14. Default template seeding integration
  - [x] 14.1 Implement seed logic in ConfigHandler.Update
    - When notification config is first created for a tenant (no existing config), call TemplateRepository.BulkCreate with DefaultTemplates from domain/seed.go
    - Set tenant_id on each template before bulk insert
    - Log seeding result
    - _Requirements: 19.1, 19.2, 19.3, 19.4, 19.5_

- [x] 15. Final checkpoint
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties from the design document (18 properties total)
- Unit tests validate specific examples and edge cases
- Migration numbering starts at 000001 (first migration for notification service)
- Max 200 lines per file constraint applies to all new files
- All code comments must be in Indonesian
- Property tests use `pgregory.net/rapid` library (consistent with billing-api)
- The service shares the same PostgreSQL database as billing-api (RLS for tenant isolation)
- sqlc.yaml goes at `services/notification/sqlc.yaml`, queries at `services/notification/queries/`
- This spec does NOT cover broadcast/bulk messaging (separate spec)
