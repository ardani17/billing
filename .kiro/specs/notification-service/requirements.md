# Requirements Document

## Introduction

Dokumen ini mendefinisikan modul Notification Service untuk ISPBoss — platform SaaS billing untuk ISP. Notification Service adalah service Go terpisah (`services/notification/`) yang menangani semua pengiriman pesan ke pelanggan melalui WhatsApp, SMS, dan Email. Service ini menerima event dari Billing API via Redis queue (asynq), melakukan resolusi template dengan substitusi variabel, memilih channel pengiriman berdasarkan konfigurasi tenant, mengirim pesan melalui provider adapter, dan mencatat hasil pengiriman. Fitur utama meliputi: delivery pipeline (resolve template → pick channel → send via adapter → log), retry mechanism dengan fallback channel, deduplication untuk mencegah duplikat, quiet hours, anti-spam throttle, dan API untuk manajemen konfigurasi, template, log, serta pengiriman manual. Semua operasi bersifat tenant-scoped via PostgreSQL RLS. Spec ini TIDAK mencakup broadcast/bulk messaging yang akan ditangani di spec terpisah (11-notification-broadcast).

## Glossary

- **Notification_Service**: Service Go terpisah (`services/notification/`) yang memproses dan mengirim notifikasi ke pelanggan
- **Event_Consumer**: Komponen asynq worker dalam Notification_Service yang menerima event dari Redis queue
- **Delivery_Pipeline**: Alur pemrosesan notifikasi: resolve template → pick channel → send via adapter → log result
- **Template_Engine**: Komponen yang melakukan substitusi variabel pada template notifikasi (contoh: `{nama}` → "Ahmad Rizki")
- **Provider_Adapter**: Interface abstraksi untuk komunikasi dengan provider eksternal (Fonnte, Zenziva, SMTP)
- **Notification_Config**: Konfigurasi provider notifikasi per tenant, disimpan di tabel `notification_configs`
- **Notification_Template**: Template pesan notifikasi dengan variabel dinamis, disimpan di tabel `notification_templates`
- **Notification_Log**: Catatan pengiriman notifikasi beserta status dan timeline, disimpan di tabel `notification_logs`
- **Channel**: Media pengiriman notifikasi: WhatsApp (WA), SMS, atau Email
- **Dedup_Key**: Kunci unik untuk mencegah duplikasi notifikasi, format: `{tenant_id}:{customer_id}:{template_id}:{periode}`
- **Quiet_Hours**: Jam tenang di mana notifikasi otomatis tidak dikirim (default 07:00-21:00 WIB)
- **Throttle**: Mekanisme pembatasan jumlah pesan per pelanggan per hari (max 5) dan cooldown antar pesan (30 menit)
- **Fallback**: Mekanisme pengiriman ke channel berikutnya jika channel utama gagal (WA → SMS → Email)
- **Tenant**: Operator ISP yang menggunakan platform ISPBoss, diidentifikasi oleh `tenant_id`
- **Billing_API**: Service Go (`services/billing-api/`) yang mempublikasikan event ke Redis queue
- **Event_Queue**: Redis-based message queue (asynq) untuk komunikasi antar service
- **TaskEnvelope**: Format standar payload event antar service yang berisi event_type, tenant_id, timestamp, correlation_id, dan payload

## Requirements

### Requirement 1: Database Schema Notification Configs

**User Story:** Sebagai developer, saya ingin tabel dedicated untuk menyimpan konfigurasi provider notifikasi per tenant, sehingga setiap tenant bisa mengkonfigurasi provider WA, SMS, dan Email secara independen.

#### Acceptance Criteria

1. THE Notification_Service migration SHALL create a `notification_configs` table with columns: `id` (UUID PK), `tenant_id` (UUID NOT NULL), `channel` (VARCHAR NOT NULL), `provider` (VARCHAR NOT NULL), `credentials` (JSONB NOT NULL), `is_enabled` (BOOLEAN NOT NULL DEFAULT false), `priority` (INTEGER NOT NULL DEFAULT 1), `settings` (JSONB DEFAULT '{}'), `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW()), `updated_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())
2. THE Notification_Service migration SHALL enable Row Level Security on the `notification_configs` table with tenant isolation policies for SELECT, INSERT, UPDATE, and DELETE operations
3. THE Notification_Service migration SHALL enforce a CHECK constraint on `channel` to accept only `whatsapp`, `sms`, or `email`
4. THE Notification_Service migration SHALL create a UNIQUE constraint on `(tenant_id, channel)` to ensure one config per channel per tenant
5. THE Notification_Service migration SHALL create an index on `(tenant_id, is_enabled)` for query performance
6. THE Notification_Service migration SHALL store `credentials` as encrypted JSONB containing provider-specific fields (api_token, sender_number for WA; api_key, user_key for SMS; smtp_host, smtp_port, username, password, from_name, from_email for Email)

### Requirement 2: Database Schema Notification Templates

**User Story:** Sebagai developer, saya ingin tabel dedicated untuk menyimpan template notifikasi, sehingga setiap tenant bisa mengelola template pesan dengan variabel dinamis untuk berbagai event.

#### Acceptance Criteria

1. THE Notification_Service migration SHALL create a `notification_templates` table with columns: `id` (UUID PK), `tenant_id` (UUID NOT NULL), `slug` (VARCHAR NOT NULL), `name` (VARCHAR NOT NULL), `category` (VARCHAR NOT NULL), `event_type` (VARCHAR), `channels` (JSONB NOT NULL DEFAULT '[]'), `body_whatsapp` (TEXT), `body_sms` (TEXT), `body_email_subject` (TEXT), `body_email_html` (TEXT), `variables` (JSONB NOT NULL DEFAULT '[]'), `is_active` (BOOLEAN NOT NULL DEFAULT true), `is_default` (BOOLEAN NOT NULL DEFAULT false), `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW()), `updated_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())
2. THE Notification_Service migration SHALL enable Row Level Security on the `notification_templates` table with tenant isolation policies
3. THE Notification_Service migration SHALL enforce a CHECK constraint on `category` to accept only `transactional`, `reminder`, `promotion`, or `information`
4. THE Notification_Service migration SHALL create a UNIQUE constraint on `(tenant_id, slug)` to ensure unique template slugs per tenant
5. THE Notification_Service migration SHALL create an index on `(tenant_id, event_type)` for event-based template lookup
6. THE Notification_Service migration SHALL create an index on `(tenant_id, is_active)` for active template queries

### Requirement 3: Database Schema Notification Logs

**User Story:** Sebagai developer, saya ingin tabel dedicated untuk mencatat setiap pengiriman notifikasi, sehingga admin bisa melihat riwayat, status, dan detail pengiriman.

#### Acceptance Criteria

1. THE Notification_Service migration SHALL create a `notification_logs` table with columns: `id` (UUID PK), `tenant_id` (UUID NOT NULL), `customer_id` (UUID NOT NULL), `template_id` (UUID REFERENCES notification_templates(id)), `channel` (VARCHAR NOT NULL), `provider` (VARCHAR NOT NULL), `recipient` (VARCHAR NOT NULL), `subject` (TEXT), `body` (TEXT NOT NULL), `status` (VARCHAR NOT NULL DEFAULT 'pending'), `retry_count` (INTEGER NOT NULL DEFAULT 0), `max_retries` (INTEGER NOT NULL DEFAULT 3), `error_message` (TEXT), `dedup_key` (VARCHAR), `metadata` (JSONB DEFAULT '{}'), `sent_at` (TIMESTAMPTZ), `created_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW()), `updated_at` (TIMESTAMPTZ NOT NULL DEFAULT NOW())
2. THE Notification_Service migration SHALL enable Row Level Security on the `notification_logs` table with tenant isolation policies
3. THE Notification_Service migration SHALL enforce a CHECK constraint on `status` to accept only `pending`, `sending`, `sent`, `delivered`, `read`, `failed`, `retrying`, `skipped`
4. THE Notification_Service migration SHALL enforce a CHECK constraint on `channel` to accept only `whatsapp`, `sms`, or `email`
5. THE Notification_Service migration SHALL create indexes on `(tenant_id, customer_id)`, `(tenant_id, status)`, `(tenant_id, created_at DESC)`, and `(dedup_key)` for query performance
6. THE Notification_Service migration SHALL create a partial unique index on `dedup_key` WHERE `dedup_key IS NOT NULL AND status != 'skipped'` with a time window filter to enforce deduplication within 1 hour

### Requirement 4: Event Consumer (Asynq Worker)

**User Story:** Sebagai developer, saya ingin Notification Service menerima event dari Billing API via Redis queue, sehingga notifikasi dikirim secara otomatis berdasarkan event bisnis.

#### Acceptance Criteria

1. THE Event_Consumer SHALL register handlers for the following event types: `invoice.created`, `invoice.reminder`, `payment.online.received`, `payment.recorded`, `customer.isolir`, `customer.un_isolir`, `customer.suspend`, `notification.isolir`, `notification.un_isolir`, `notification.suspend`, `notification.reactivated`, `notification.pending_sync_failed`, `invoice.penalty_added`
2. WHEN an event is received, THE Event_Consumer SHALL decode the TaskEnvelope using `queue.DecodeEnvelope` to extract event_type, tenant_id, timestamp, correlation_id, and payload
3. WHEN an event is received, THE Event_Consumer SHALL pass the decoded envelope to the Delivery_Pipeline for processing
4. IF the event payload cannot be decoded, THEN THE Event_Consumer SHALL log the error with correlation_id and skip the event without retrying
5. THE Event_Consumer SHALL run with configurable concurrency (default 10 workers) and queue priorities: `critical: 6, default: 3, low: 1`

### Requirement 5: Template Engine (Variable Substitution)

**User Story:** Sebagai developer, saya ingin template engine yang melakukan substitusi variabel pada template, sehingga pesan yang dikirim berisi data pelanggan dan transaksi yang relevan.

#### Acceptance Criteria

1. THE Template_Engine SHALL replace all variables in template body with actual values from the event payload and customer data, using the format `{variable_name}`
2. THE Template_Engine SHALL support the following variables: `{nama}`, `{id_pelanggan}`, `{nama_isp}`, `{telepon_isp}`, `{paket}`, `{harga}`, `{periode}`, `{no_invoice}`, `{total_tagihan}`, `{jatuh_tempo}`, `{sisa_hari}`, `{terlambat_hari}`, `{link_bayar}`, `{link_bayar_short}`, `{tanggal_bayar}`, `{metode_bayar}`, `{jumlah_bayar}`
3. IF a variable in the template has no corresponding value in the data context, THEN THE Template_Engine SHALL replace it with an empty string and log a warning
4. FOR ALL valid template bodies, rendering then extracting variables then re-rendering with the same data SHALL produce an identical output (idempotence property)
5. THE Template_Engine SHALL format monetary values with prefix "Rp " and thousand separators (contoh: "Rp 388.500")
6. THE Template_Engine SHALL format date values in Indonesian locale (contoh: "5 April 2026")

### Requirement 6: Provider Adapter Interfaces

**User Story:** Sebagai developer, saya ingin interface adapter untuk setiap channel notifikasi, sehingga provider bisa diganti tanpa mengubah business logic.

#### Acceptance Criteria

1. THE Notification_Service SHALL define a `WhatsAppProvider` interface with method `Send(ctx context.Context, req WhatsAppMessage) (SendResult, error)` where WhatsAppMessage contains recipient phone, body text, and optional media URL
2. THE Notification_Service SHALL define a `SMSProvider` interface with method `Send(ctx context.Context, req SMSMessage) (SendResult, error)` where SMSMessage contains recipient phone and body text (max 160 characters)
3. THE Notification_Service SHALL define an `EmailProvider` interface with method `Send(ctx context.Context, req EmailMessage) (SendResult, error)` where EmailMessage contains recipient email, subject, and HTML body
4. THE Notification_Service SHALL implement `FonnteAdapter` as the first WhatsApp provider implementation that calls Fonnte HTTP API with the configured API token
5. THE Notification_Service SHALL implement `ZenzivaAdapter` as the first SMS provider implementation that calls Zenziva HTTP API with the configured API key and user key
6. THE Notification_Service SHALL implement `SMTPAdapter` as the first Email provider implementation that sends email via SMTP with the configured host, port, username, and password
7. FOR ALL provider adapters, THE SendResult SHALL contain: `message_id` (string from provider), `status` (sent/failed), and `error_detail` (string, empty if success)

### Requirement 7: Delivery Pipeline

**User Story:** Sebagai developer, saya ingin delivery pipeline yang mengorkestrasikan seluruh alur pengiriman notifikasi, sehingga setiap event diproses secara konsisten dari resolusi template hingga logging hasil.

#### Acceptance Criteria

1. WHEN an event enters the Delivery_Pipeline, THE Delivery_Pipeline SHALL resolve the appropriate template by matching `event_type` to `notification_templates.event_type` for the tenant
2. WHEN a template is resolved, THE Delivery_Pipeline SHALL fetch customer data (nama, telepon, email, paket, id_pelanggan) and tenant data (nama_isp, telepon_isp) to build the variable context
3. WHEN the variable context is built, THE Delivery_Pipeline SHALL render the template body using the Template_Engine for each configured channel
4. WHEN the rendered message is ready, THE Delivery_Pipeline SHALL select the channel based on the tenant's configured priority order and template's channel configuration
5. WHEN the channel is selected, THE Delivery_Pipeline SHALL send the message via the corresponding Provider_Adapter
6. WHEN the send result is received, THE Delivery_Pipeline SHALL create a Notification_Log record with the full details (channel, provider, recipient, body, status, metadata)
7. IF the template for the event_type is not found or is inactive, THEN THE Delivery_Pipeline SHALL log a warning and skip the notification without creating a log record

### Requirement 8: Retry Mechanism dengan Fallback Channel

**User Story:** Sebagai tenant admin, saya ingin notifikasi yang gagal dikirim otomatis di-retry dan fallback ke channel lain, sehingga pelanggan tetap menerima pesan meskipun channel utama bermasalah.

#### Acceptance Criteria

1. WHEN a notification send fails, THE Delivery_Pipeline SHALL retry on the same channel up to 2 times with 5-minute intervals between retries
2. WHEN all retries on the primary channel are exhausted (total 3 attempts including initial), THE Delivery_Pipeline SHALL attempt to send via the next channel in the tenant's priority order (fallback)
3. WHEN falling back to the next channel, THE Delivery_Pipeline SHALL render the template body for that channel (contoh: body_sms untuk SMS fallback) and send via the corresponding adapter
4. WHEN all channels and retries are exhausted, THE Delivery_Pipeline SHALL mark the Notification_Log status as `failed` with the accumulated error messages
5. THE Delivery_Pipeline SHALL update the Notification_Log `retry_count` and `status` after each attempt (status `retrying` during retries, `sent` on success, `failed` on final failure)
6. WHEN a notification is successfully sent after retry or fallback, THE Delivery_Pipeline SHALL update the Notification_Log with the actual channel and provider used

### Requirement 9: Deduplication

**User Story:** Sebagai developer, saya ingin mekanisme deduplication untuk mencegah notifikasi duplikat, sehingga pelanggan tidak menerima pesan yang sama berkali-kali akibat event retry atau cron overlap.

#### Acceptance Criteria

1. WHEN a notification enters the Delivery_Pipeline, THE Delivery_Pipeline SHALL generate a dedup_key with format `{tenant_id}:{customer_id}:{template_slug}:{periode}` where periode is derived from the event context (contoh: "2026-04" untuk invoice bulanan)
2. WHEN a dedup_key is generated, THE Delivery_Pipeline SHALL check if a Notification_Log with the same dedup_key and status not `skipped` exists within the last 1 hour
3. IF a duplicate is detected, THEN THE Delivery_Pipeline SHALL skip the notification and create a Notification_Log with status `skipped` and metadata containing reason "duplicate"
4. IF no duplicate is detected, THEN THE Delivery_Pipeline SHALL proceed with normal delivery
5. FOR ALL notifications with the same dedup_key within a 1-hour window, only the first notification SHALL be delivered (deduplication invariant)

### Requirement 10: Quiet Hours

**User Story:** Sebagai tenant admin, saya ingin notifikasi otomatis tidak dikirim di luar jam operasional, sehingga pelanggan tidak terganggu di malam hari.

#### Acceptance Criteria

1. WHILE the current time (in tenant's timezone) is outside the configured quiet hours (default 07:00-21:00), THE Delivery_Pipeline SHALL queue the notification for delivery at the start of the next active period
2. THE Notification_Config settings SHALL store `quiet_hours_start` (default "07:00") and `quiet_hours_end` (default "21:00") per tenant
3. WHEN a notification is queued due to quiet hours, THE Delivery_Pipeline SHALL create a Notification_Log with status `pending` and metadata containing `delayed_reason: "quiet_hours"` and `scheduled_at` timestamp
4. IF the notification event_type is `payment.online.received`, `payment.recorded`, `notification.un_isolir`, or `notification.reactivated`, THEN THE Delivery_Pipeline SHALL bypass quiet hours and send immediately (exception for payment confirmation and un-isolir)
5. THE Delivery_Pipeline SHALL use the tenant's configured timezone from Notification_Config settings (default `Asia/Jakarta`) for quiet hours calculation

### Requirement 11: Anti-Spam Throttle

**User Story:** Sebagai tenant admin, saya ingin pembatasan jumlah pesan per pelanggan per hari, sehingga pelanggan tidak menerima terlalu banyak pesan dalam waktu singkat.

#### Acceptance Criteria

1. WHEN a notification is about to be sent, THE Delivery_Pipeline SHALL check the count of notifications sent to the same customer in the current day (based on tenant timezone)
2. IF the daily count for the customer is 5 or more, THEN THE Delivery_Pipeline SHALL skip the notification and create a Notification_Log with status `skipped` and metadata containing reason "daily_limit_exceeded"
3. WHEN a notification is about to be sent, THE Delivery_Pipeline SHALL check the timestamp of the last notification sent to the same customer
4. IF the last notification was sent less than 30 minutes ago, THEN THE Delivery_Pipeline SHALL delay the notification to 30 minutes after the last sent notification
5. IF the notification event_type is `payment.online.received`, `payment.recorded`, `notification.un_isolir`, or `notification.reactivated`, THEN THE Delivery_Pipeline SHALL bypass throttle limits (exception for payment confirmation and un-isolir)
6. THE Delivery_Pipeline SHALL count only notifications with status `sent` or `delivered` toward the daily limit (not `skipped` or `failed`)

### Requirement 12: Notification Log API

**User Story:** Sebagai tenant admin, saya ingin melihat riwayat pengiriman notifikasi dengan filter dan pagination, sehingga saya bisa memantau status pengiriman dan mendiagnosis masalah.

#### Acceptance Criteria

1. WHEN a GET request is made to `/api/v1/notifications/logs`, THE Notification_Service SHALL return a paginated list of Notification_Log records for the authenticated tenant sorted by `created_at` DESC
2. THE Notification_Service SHALL support query parameters: `channel` (whatsapp/sms/email), `status` (pending/sending/sent/delivered/read/failed/retrying/skipped), `customer_id`, `template_id`, `date_from` (ISO date), `date_to` (ISO date)
3. THE Notification_Service SHALL default to 25 items per page and support `page_size` values of 10, 25, or 50
4. THE Notification_Service SHALL return pagination metadata including `total`, `page`, `page_size`, `total_pages` in the response
5. THE Notification_Service SHALL include in each log record: id, customer_id, customer_name, channel, provider, template_name, recipient, status, retry_count, error_message, sent_at, created_at

### Requirement 13: Notification Config API

**User Story:** Sebagai tenant admin, saya ingin mengelola konfigurasi provider notifikasi, sehingga saya bisa mengatur provider WA, SMS, Email, prioritas channel, dan quiet hours.

#### Acceptance Criteria

1. WHEN a GET request is made to `/api/v1/notifications/config`, THE Notification_Service SHALL return the notification configuration for the authenticated tenant including all channel configs and general settings (quiet hours, channel priority)
2. WHEN a PUT request is made to `/api/v1/notifications/config`, THE Notification_Service SHALL update the notification configuration for the authenticated tenant
3. THE Notification_Service SHALL validate that credentials are provided when a channel is enabled (`is_enabled: true`)
4. IF credentials validation fails (missing required fields per provider), THEN THE Notification_Service SHALL return HTTP 422 with error details specifying which fields are missing
5. THE Notification_Service SHALL store credentials in encrypted JSONB format in the database
6. THE Notification_Service SHALL NOT return raw credential values in GET responses — credentials SHALL be masked (contoh: "••••••••" with last 4 characters visible)

### Requirement 14: Template CRUD API

**User Story:** Sebagai tenant admin, saya ingin mengelola template notifikasi (buat, lihat, edit), sehingga saya bisa menyesuaikan pesan yang dikirim ke pelanggan.

#### Acceptance Criteria

1. WHEN a GET request is made to `/api/v1/notifications/templates`, THE Notification_Service SHALL return a list of all notification templates for the authenticated tenant
2. WHEN a POST request is made to `/api/v1/notifications/templates`, THE Notification_Service SHALL create a new custom template with the provided slug, name, category, channels, and body content
3. WHEN a PUT request is made to `/api/v1/notifications/templates/:id`, THE Notification_Service SHALL update the specified template's name, channels, body content, and is_active status
4. IF the template slug already exists for the tenant, THEN THE Notification_Service SHALL return HTTP 409 with error code `TEMPLATE_SLUG_EXISTS`
5. THE Notification_Service SHALL validate that at least one channel body is provided (body_whatsapp, body_sms, or body_email_subject + body_email_html)
6. THE Notification_Service SHALL NOT allow deletion of default templates (`is_default: true`) — only custom templates can be deleted
7. WHEN a DELETE request is made to `/api/v1/notifications/templates/:id` for a custom template, THE Notification_Service SHALL soft-delete the template by setting `is_active: false`

### Requirement 15: Send Test Notification API

**User Story:** Sebagai tenant admin, saya ingin mengirim notifikasi test ke nomor/email admin, sehingga saya bisa memverifikasi konfigurasi provider dan template sebelum digunakan.

#### Acceptance Criteria

1. WHEN a POST request is made to `/api/v1/notifications/test`, THE Notification_Service SHALL send a test notification using the specified template_id, channel, and recipient (phone number or email)
2. THE Notification_Service SHALL render the template with sample data (contoh: nama="Test User", total_tagihan="Rp 100.000") for preview purposes
3. THE Notification_Service SHALL bypass deduplication, quiet hours, and throttle checks for test notifications
4. THE Notification_Service SHALL create a Notification_Log record with metadata containing `is_test: true`
5. IF the provider is not configured or disabled for the specified channel, THEN THE Notification_Service SHALL return HTTP 422 with error code `PROVIDER_NOT_CONFIGURED`
6. THE Notification_Service SHALL return the send result (success/failed) and message_id from the provider in the response

### Requirement 16: Manual Send Notification API

**User Story:** Sebagai tenant admin, saya ingin mengirim notifikasi manual ke pelanggan tertentu, sehingga saya bisa mengirim pesan custom atau menggunakan template yang ada di luar trigger otomatis.

#### Acceptance Criteria

1. WHEN a POST request is made to `/api/v1/notifications/send`, THE Notification_Service SHALL send a notification to the specified customer_id using the specified template_id or custom body
2. THE Notification_Service SHALL resolve customer data (nama, telepon, email) from the database for variable substitution
3. IF template_id is provided, THE Notification_Service SHALL render the template with customer data; IF custom body is provided, THE Notification_Service SHALL use the custom body directly
4. THE Notification_Service SHALL respect throttle limits for manual sends (daily limit and cooldown apply)
5. THE Notification_Service SHALL create a Notification_Log record with metadata containing `trigger: "manual"` and `actor` set to the authenticated admin user ID
6. IF the customer_id does not exist or belongs to a different tenant, THEN THE Notification_Service SHALL return HTTP 404 with error code `CUSTOMER_NOT_FOUND`

### Requirement 17: Manual Resend Notification API

**User Story:** Sebagai tenant admin, saya ingin mengirim ulang notifikasi yang gagal dari log, sehingga saya bisa memastikan pelanggan menerima pesan penting yang sebelumnya gagal terkirim.

#### Acceptance Criteria

1. WHEN a POST request is made to `/api/v1/notifications/logs/:id/resend`, THE Notification_Service SHALL resend the notification using the same template, channel, and recipient from the original log record
2. THE Notification_Service SHALL create a new Notification_Log record (not update the original) with metadata containing `resend_of: original_log_id`
3. IF the original log status is not `failed`, THEN THE Notification_Service SHALL return HTTP 422 with error code `NOT_RESENDABLE` (only failed notifications can be resent)
4. THE Notification_Service SHALL bypass deduplication for resend operations (dedup_key is not checked)
5. THE Notification_Service SHALL respect quiet hours and throttle limits for resend operations
6. IF the log_id does not exist or belongs to a different tenant, THEN THE Notification_Service SHALL return HTTP 404 with error code `LOG_NOT_FOUND`

### Requirement 18: Domain Entities

**User Story:** Sebagai developer, saya ingin domain entities yang terdefinisi dengan baik untuk notification service, sehingga business logic terstruktur dan testable.

#### Acceptance Criteria

1. THE Notification_Service SHALL define a `NotificationConfig` entity with fields matching the `notification_configs` table schema including channel, provider, credentials (as typed struct per provider), is_enabled, priority, and settings
2. THE Notification_Service SHALL define a `NotificationTemplate` entity with fields matching the `notification_templates` table schema including slug, name, category, event_type, channels, body per channel, variables list, is_active, and is_default
3. THE Notification_Service SHALL define a `NotificationLog` entity with fields matching the `notification_logs` table schema including customer_id, template_id, channel, provider, recipient, subject, body, status, retry_count, error_message, dedup_key, metadata, and sent_at
4. THE Notification_Service SHALL define status constants for NotificationLog: `StatusPending`, `StatusSending`, `StatusSent`, `StatusDelivered`, `StatusRead`, `StatusFailed`, `StatusRetrying`, `StatusSkipped`
5. THE Notification_Service SHALL define category constants for NotificationTemplate: `CategoryTransactional`, `CategoryReminder`, `CategoryPromotion`, `CategoryInformation`
6. THE Notification_Service SHALL define channel constants: `ChannelWhatsApp`, `ChannelSMS`, `ChannelEmail`

### Requirement 19: Default Templates Seeding

**User Story:** Sebagai tenant admin, saya ingin template bawaan tersedia saat pertama kali menggunakan notification service, sehingga notifikasi otomatis bisa langsung berjalan tanpa konfigurasi template manual.

#### Acceptance Criteria

1. THE Notification_Service SHALL provide a seeding mechanism that creates default templates for a tenant when notification config is first created
2. THE Notification_Service SHALL seed the following default templates with `is_default: true`: `invoice_new` (event: invoice.created), `reminder_h1` (event: invoice.reminder), `payment_confirm` (event: payment.online.received/payment.recorded), `isolir_notice` (event: notification.isolir), `suspend_notice` (event: notification.suspend), `unblock_notice` (event: notification.un_isolir), `reactivated_notice` (event: notification.reactivated)
3. THE Notification_Service SHALL include both WhatsApp and SMS body variants for each default template
4. THE Notification_Service SHALL mark default templates as `is_default: true` so they cannot be deleted but can be edited
5. THE Notification_Service SHALL include the list of available variables in the `variables` JSONB field for each template

### Requirement 20: Notification Config General Settings

**User Story:** Sebagai tenant admin, saya ingin mengatur pengaturan umum notifikasi seperti prioritas channel dan quiet hours, sehingga perilaku pengiriman sesuai dengan kebutuhan operasional ISP saya.

#### Acceptance Criteria

1. THE Notification_Config settings JSONB SHALL store: `channel_priority` (array of channels in order, default: ["whatsapp", "sms", "email"]), `quiet_hours_start` (string HH:MM, default "07:00"), `quiet_hours_end` (string HH:MM, default "21:00"), `timezone` (string, default "Asia/Jakarta"), `daily_limit_per_customer` (integer, default 5), `cooldown_minutes` (integer, default 30)
2. WHEN channel_priority is configured, THE Delivery_Pipeline SHALL use the configured order for primary channel selection and fallback sequence
3. THE Notification_Service SHALL validate that `quiet_hours_start` is before `quiet_hours_end` in the same day
4. THE Notification_Service SHALL validate that `timezone` is one of `Asia/Jakarta`, `Asia/Makassar`, or `Asia/Jayapura`
5. THE Notification_Service SHALL validate that `daily_limit_per_customer` is between 1 and 20 inclusive
6. THE Notification_Service SHALL validate that `cooldown_minutes` is between 5 and 120 inclusive
