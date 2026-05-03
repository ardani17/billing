# Requirements Document — Authentication & Role-Based Access Control (ISPBoss)

## Introduction

Dokumen ini mendefinisikan requirements untuk sistem autentikasi dan role-based access control (RBAC) ISPBoss. Spec ini mencakup: registrasi tenant baru, login (email/password + Google OAuth), verifikasi email, lupa password, manajemen session, JWT token lifecycle, RBAC dengan 6 role, dan keamanan (rate limiting, bcrypt hashing, CSRF). Spec ini dibangun di atas fondasi monorepo-setup (Spec 01) yang sudah menyediakan `pkg/auth` (JWT), `pkg/tenant` (middleware), PostgreSQL dengan RLS, dan clean architecture di semua Go service.

## Glossary

- **Auth_Service**: Modul autentikasi di dalam Billing_API yang menangani register, login, verifikasi email, lupa password, dan manajemen session
- **Billing_API**: Service Golang utama di `services/billing-api/` yang menangani auth, customer, invoice, dan payment
- **Tenant**: Operator ISP/RT-RW Net yang berlangganan ISPBoss; setiap tenant memiliki data terisolasi via tenant_id
- **Tenant_Admin**: Pemilik ISP yang mendaftarkan tenant baru; memiliki full access ke tenant sendiri
- **Operator**: Staff operasional tenant yang mengelola pelanggan, billing, dan notifikasi
- **Teknisi**: Staff teknis tenant yang mengelola MikroTik, OLT, dan peta jaringan
- **Kasir**: Staff keuangan tenant yang menginput pembayaran dan melihat data pelanggan (read-only)
- **Reseller**: Mitra penjual voucher dengan dashboard terpisah untuk beli, print voucher, dan deposit
- **Super_Admin**: Tim internal ISPBoss dengan akses lintas tenant, manage subscription, dan impersonate
- **JWT_Token**: JSON Web Token yang berisi claims tenant_id, user_id, dan role; ditandatangani dengan HS256
- **Refresh_Token**: Token opaque yang disimpan di database untuk memperpanjang JWT tanpa login ulang
- **Session**: Record di tabel sessions yang merepresentasikan satu sesi login aktif dari satu device
- **Email_Verification_Token**: Token hash yang dikirim via email untuk memverifikasi alamat email pengguna baru
- **Password_Reset_Token**: Token hash yang dikirim via email untuk mereset password; berlaku 1 jam
- **Rate_Limiter**: Mekanisme pembatasan jumlah percobaan login gagal per email; maksimal 5 percobaan sebelum lock 15 menit
- **RLS (Row Level Security)**: Fitur PostgreSQL yang membatasi akses baris data berdasarkan policy tenant_id
- **Google_OAuth**: Autentikasi via Google OAuth 2.0 sebagai alternatif registrasi dan login manual
- **Bcrypt**: Algoritma hashing password yang digunakan untuk menyimpan password secara aman
- **RBAC_Middleware**: Middleware yang memeriksa role pengguna dari JWT claims dan menentukan akses ke endpoint

---

## Requirements

### Requirement 1: Database Schema untuk Auth

**User Story:** As a platform engineer, I want auth-related database tables with proper multi-tenant isolation, so that user credentials, sessions, and tokens are stored securely and scoped per tenant.

#### Acceptance Criteria

1. WHEN the auth migration is applied, THE Migration_Runner SHALL create a `users` table with columns: `id` (UUID, PK), `tenant_id` (UUID, FK to tenants, NOT NULL), `name` (VARCHAR 255, NOT NULL), `email` (VARCHAR 255, NOT NULL), `phone` (VARCHAR 50), `password_hash` (VARCHAR 255), `role` (VARCHAR 50, NOT NULL), `email_verified` (BOOLEAN, default false), `google_id` (VARCHAR 255), `status` (VARCHAR 50, default 'active'), `last_login` (TIMESTAMPTZ), `created_at` (TIMESTAMPTZ), `updated_at` (TIMESTAMPTZ)
2. WHEN the auth migration is applied, THE Migration_Runner SHALL create a `sessions` table with columns: `id` (UUID, PK), `user_id` (UUID, FK to users, NOT NULL), `token_hash` (VARCHAR 255, NOT NULL), `device_info` (VARCHAR 500), `ip_address` (VARCHAR 45), `expires_at` (TIMESTAMPTZ, NOT NULL), `created_at` (TIMESTAMPTZ)
3. WHEN the auth migration is applied, THE Migration_Runner SHALL create a `password_resets` table with columns: `id` (UUID, PK), `user_id` (UUID, FK to users, NOT NULL), `token_hash` (VARCHAR 255, NOT NULL, UNIQUE), `expires_at` (TIMESTAMPTZ, NOT NULL), `used` (BOOLEAN, default false), `created_at` (TIMESTAMPTZ)
4. WHEN the auth migration is applied, THE Migration_Runner SHALL create an `email_verifications` table with columns: `id` (UUID, PK), `user_id` (UUID, FK to users, NOT NULL), `token_hash` (VARCHAR 255, NOT NULL, UNIQUE), `expires_at` (TIMESTAMPTZ, NOT NULL), `used` (BOOLEAN, default false), `created_at` (TIMESTAMPTZ)
5. THE Migration_Runner SHALL enable Row Level Security on the `users` table with a tenant isolation policy using `tenant_id = current_setting('app.tenant_id')::uuid`
6. THE Migration_Runner SHALL create a UNIQUE constraint on `(tenant_id, email)` in the `users` table to prevent duplicate emails within a tenant
7. THE Migration_Runner SHALL create indexes on: `users(tenant_id)`, `users(tenant_id, email)`, `users(google_id)`, `sessions(user_id)`, `sessions(token_hash)`, `password_resets(token_hash)`, `email_verifications(token_hash)`

---

### Requirement 2: Tenant Registration

**User Story:** As an ISP owner, I want to register my ISP on ISPBoss, so that I can start managing my customers and billing.

#### Acceptance Criteria

1. WHEN a POST request is sent to `/v1/auth/register` with valid fields (name, email, phone, company_name, password, password_confirmation, agree_terms), THE Auth_Service SHALL create a new tenant record with plan 'starter' and status 'trial', create a user record with role 'tenant_admin' and email_verified false, and return HTTP 201 with user_id and tenant_id
2. WHEN a registration request is received, THE Auth_Service SHALL validate that: name is at least 3 characters, email is a valid format, phone starts with '+62', password is at least 8 characters, password matches password_confirmation, and agree_terms is true
3. WHEN a registration request is received, THE Auth_Service SHALL hash the password using bcrypt with a cost factor of at least 10 before storing in the database
4. WHEN a registration is successful, THE Auth_Service SHALL generate an Email_Verification_Token, store its hash in the email_verifications table with expires_at set to 24 hours from creation, and enqueue an email verification task to the Notification_Service
5. IF a registration request contains an email that already exists for any tenant, THEN THE Auth_Service SHALL return HTTP 409 with error code 'EMAIL_ALREADY_EXISTS'
6. IF a registration request contains invalid or missing fields, THEN THE Auth_Service SHALL return HTTP 422 with error code 'VALIDATION_ERROR' and a list of field-level errors

---

### Requirement 3: Email Verification

**User Story:** As a new user, I want to verify my email address, so that I can activate my account and start using ISPBoss.

#### Acceptance Criteria

1. WHEN a POST request is sent to `/v1/auth/verify-email` with a valid and unexpired token, THE Auth_Service SHALL set the user's email_verified to true, mark the token as used, generate a JWT_Token and Refresh_Token, create a Session record, and return HTTP 200 with the tokens and user info
2. IF a verify-email request contains an expired token (older than 24 hours), THEN THE Auth_Service SHALL return HTTP 410 with error code 'TOKEN_EXPIRED'
3. IF a verify-email request contains a token that has already been used, THEN THE Auth_Service SHALL return HTTP 410 with error code 'TOKEN_ALREADY_USED'
4. IF a verify-email request contains an invalid token (not found in database), THEN THE Auth_Service SHALL return HTTP 404 with error code 'TOKEN_NOT_FOUND'
5. WHEN a POST request is sent to `/v1/auth/resend-verification` with a valid email, THE Auth_Service SHALL invalidate any existing unused verification tokens for that user, generate a new Email_Verification_Token with 24-hour expiry, and enqueue a new verification email
6. WHEN a resend-verification request is received, THE Auth_Service SHALL enforce a cooldown of 60 seconds between resend requests per email; IF the cooldown has not elapsed, THEN THE Auth_Service SHALL return HTTP 429 with error code 'RESEND_COOLDOWN' and the remaining seconds

---

### Requirement 4: Login with Email and Password

**User Story:** As a registered user, I want to log in with my email and password, so that I can access the ISPBoss dashboard.

#### Acceptance Criteria

1. WHEN a POST request is sent to `/v1/auth/login` with valid email, password, and optional remember_me flag, THE Auth_Service SHALL verify the password against the stored bcrypt hash, generate a JWT_Token (expiry 24 hours if remember_me is false, 7 days if true) and a Refresh_Token, create a Session record with device_info and ip_address, update the user's last_login timestamp, and return HTTP 200 with the tokens, user info, and redirect path based on role
2. WHEN a login is successful, THE Auth_Service SHALL return a redirect_path based on the user's role: 'tenant_admin' maps to '/dashboard', 'operator' maps to '/dashboard', 'teknisi' maps to '/network', 'kasir' maps to '/payments'
3. IF a login request contains an email that does not exist, THEN THE Auth_Service SHALL return HTTP 401 with error code 'INVALID_CREDENTIALS' and a generic message that does not reveal whether the email exists
4. IF a login request contains an incorrect password, THEN THE Auth_Service SHALL return HTTP 401 with error code 'INVALID_CREDENTIALS' and the same generic message
5. IF a login request is for a user whose email_verified is false, THEN THE Auth_Service SHALL return HTTP 403 with error code 'EMAIL_NOT_VERIFIED' and include the user's email for the frontend to offer resend
6. IF a login request is for a user whose status is not 'active', THEN THE Auth_Service SHALL return HTTP 403 with error code 'ACCOUNT_DISABLED'

---

### Requirement 5: Login Rate Limiting

**User Story:** As a security engineer, I want login attempts to be rate-limited, so that brute-force attacks are mitigated.

#### Acceptance Criteria

1. WHEN a login attempt fails, THE Rate_Limiter SHALL increment a counter for that email address stored in Redis with a TTL of 15 minutes
2. WHEN the failed login counter for an email reaches 5, THE Rate_Limiter SHALL lock the account for 15 minutes; any subsequent login attempt for that email SHALL return HTTP 429 with error code 'ACCOUNT_LOCKED' and the remaining lock duration in seconds
3. WHEN a login attempt succeeds, THE Rate_Limiter SHALL reset the failed login counter for that email to zero
4. WHEN the 15-minute lock period expires, THE Rate_Limiter SHALL allow login attempts again by letting the Redis key expire naturally
5. THE Rate_Limiter SHALL use the email address (not IP address) as the rate limit key to prevent attackers from bypassing limits by switching IPs

---

### Requirement 6: Google OAuth Login and Registration

**User Story:** As a user, I want to register or log in using my Google account, so that I can access ISPBoss without creating a separate password.

#### Acceptance Criteria

1. WHEN a POST request is sent to `/v1/auth/google` with a valid Google OAuth id_token, THE Auth_Service SHALL verify the token with Google's public keys, extract the user's email, name, and google_id from the token claims
2. IF the Google email does not exist in the users table, THEN THE Auth_Service SHALL create a new tenant (plan 'starter', status 'trial'), create a user with role 'tenant_admin', email_verified set to true, google_id set from the token, and password_hash set to empty; THE Auth_Service SHALL then generate JWT and Refresh tokens and return HTTP 201
3. IF the Google email already exists and the user has a google_id matching the token, THEN THE Auth_Service SHALL treat the request as a login: generate JWT and Refresh tokens, create a Session, and return HTTP 200
4. IF the Google email already exists but the user has no google_id (registered via email/password), THEN THE Auth_Service SHALL link the Google account by setting the google_id on the existing user, and proceed with login
5. IF the Google OAuth id_token is invalid or expired, THEN THE Auth_Service SHALL return HTTP 401 with error code 'INVALID_GOOGLE_TOKEN'

---

### Requirement 7: Forgot Password and Reset

**User Story:** As a user who forgot my password, I want to reset it via email, so that I can regain access to my account.

#### Acceptance Criteria

1. WHEN a POST request is sent to `/v1/auth/forgot-password` with an email, THE Auth_Service SHALL generate a Password_Reset_Token, store its hash in the password_resets table with expires_at set to 1 hour from creation, invalidate any existing unused reset tokens for that user, and enqueue a password reset email task
2. IF the email does not exist in the users table, THEN THE Auth_Service SHALL still return HTTP 200 with a generic success message to prevent email enumeration
3. WHEN a POST request is sent to `/v1/auth/reset-password` with a valid and unexpired token and a new password (at least 8 characters), THE Auth_Service SHALL hash the new password with bcrypt, update the user's password_hash, mark the token as used, invalidate all existing sessions for that user, generate new JWT and Refresh tokens, create a new Session, and return HTTP 200 with the tokens
4. IF a reset-password request contains an expired token (older than 1 hour), THEN THE Auth_Service SHALL return HTTP 410 with error code 'TOKEN_EXPIRED'
5. IF a reset-password request contains a token that has already been used, THEN THE Auth_Service SHALL return HTTP 410 with error code 'TOKEN_ALREADY_USED'
6. IF a reset-password request contains an invalid token, THEN THE Auth_Service SHALL return HTTP 404 with error code 'TOKEN_NOT_FOUND'

---

### Requirement 8: JWT Token Lifecycle and Session Management

**User Story:** As a platform engineer, I want proper JWT token lifecycle with refresh capability and session tracking, so that users stay authenticated securely across devices.

#### Acceptance Criteria

1. WHEN a JWT_Token is generated, THE Auth_Service SHALL include the following claims: `tenant_id`, `user_id`, `role`, `iss` (issuer: 'ispboss'), `iat` (issued at), `exp` (expiry), and sign the token using HS256 with the configured JWT_SECRET
2. WHEN a POST request is sent to `/v1/auth/refresh` with a valid Refresh_Token, THE Auth_Service SHALL verify the token exists in the sessions table and has not expired, generate a new JWT_Token with the same claims, rotate the Refresh_Token (invalidate old, create new), and return HTTP 200 with the new tokens
3. WHEN a POST request is sent to `/v1/auth/logout`, THE Auth_Service SHALL invalidate the current session by deleting the Session record associated with the provided Refresh_Token, and return HTTP 200
4. WHEN a GET request is sent to `/v1/auth/me` with a valid JWT_Token in the Authorization header, THE Auth_Service SHALL return HTTP 200 with the current user's id, name, email, phone, role, tenant_id, email_verified status, and last_login
5. IF a refresh request contains an expired or invalid Refresh_Token, THEN THE Auth_Service SHALL return HTTP 401 with error code 'INVALID_REFRESH_TOKEN'
6. FOR ALL valid JWT_Tokens, generating a token then validating the token SHALL return the original claims (tenant_id, user_id, role) unchanged (round-trip property)

---

### Requirement 9: Role-Based Access Control (RBAC)

**User Story:** As a Tenant Admin, I want to assign roles to my staff, so that each person only has access to the features they need.

#### Acceptance Criteria

1. THE RBAC_Middleware SHALL support 6 roles with the following hierarchy: Super_Admin (cross-tenant), Tenant_Admin (full access own tenant), Operator (daily operations), Teknisi (network), Kasir (payments), Reseller (voucher dashboard)
2. WHEN a request reaches a protected endpoint, THE RBAC_Middleware SHALL extract the role from JWT claims and compare the role against the endpoint's allowed roles; IF the role is not allowed, THEN THE RBAC_Middleware SHALL return HTTP 403 with error code 'FORBIDDEN'
3. THE RBAC_Middleware SHALL enforce the following endpoint access rules: `/v1/customers/*` is accessible by Super_Admin, Tenant_Admin, Operator (full), and Kasir (GET only); `/v1/invoices/*` is accessible by Super_Admin, Tenant_Admin, Operator, and Kasir; `/v1/payments/*` is accessible by Super_Admin, Tenant_Admin, Operator, and Kasir; `/v1/mikrotik/*` is accessible by Super_Admin, Tenant_Admin, Operator, and Teknisi; `/v1/olt/*` is accessible by Super_Admin, Tenant_Admin, and Teknisi; `/v1/network-map/*` is accessible by Super_Admin, Tenant_Admin, and Teknisi; `/v1/settings/*` is accessible by Super_Admin and Tenant_Admin only; `/v1/reports/*` is accessible by Super_Admin, Tenant_Admin, and Operator (GET only)
4. WHEN a Super_Admin sends a request, THE RBAC_Middleware SHALL allow access to all endpoints across all tenants without tenant_id filtering
5. WHEN a non-Super_Admin user sends a request, THE RBAC_Middleware SHALL enforce tenant isolation by ensuring the user can only access data belonging to the user's own tenant_id
6. THE RBAC_Middleware SHALL support method-level restrictions: Kasir role SHALL have GET-only access to `/v1/customers/*` and full access to `/v1/payments/*`; Operator role SHALL have GET-only access to `/v1/reports/*`

---

### Requirement 10: User Management by Tenant Admin

**User Story:** As a Tenant Admin, I want to manage users within my tenant, so that I can add staff, assign roles, and control access.

#### Acceptance Criteria

1. WHEN a Tenant_Admin sends a POST request to `/v1/settings/users` with valid fields (name, email, phone, password, role), THE Auth_Service SHALL create a new user within the admin's tenant, hash the password with bcrypt, set email_verified to true (admin-created users skip verification), and return HTTP 201 with the new user's info
2. WHEN a Tenant_Admin sends a PUT request to `/v1/settings/users/{id}`, THE Auth_Service SHALL update the user's name, phone, and role; THE Auth_Service SHALL NOT allow changing the user's tenant_id
3. WHEN a Tenant_Admin sends a POST request to `/v1/settings/users/{id}/deactivate`, THE Auth_Service SHALL set the user's status to 'inactive', invalidate all active sessions for that user, and return HTTP 200
4. WHEN a Tenant_Admin sends a POST request to `/v1/settings/users/{id}/activate`, THE Auth_Service SHALL set the user's status to 'active' and return HTTP 200
5. WHEN a Tenant_Admin sends a DELETE request to `/v1/settings/users/{id}`, THE Auth_Service SHALL permanently delete the user record and all associated sessions, password_resets, and email_verifications; THE Auth_Service SHALL require a confirmation field matching the user's name
6. WHEN a Tenant_Admin sends a POST request to `/v1/settings/users/{id}/reset-password`, THE Auth_Service SHALL generate a Password_Reset_Token and enqueue a reset email to the target user
7. THE Auth_Service SHALL NOT allow a Tenant_Admin to create users with the Super_Admin role
8. THE Auth_Service SHALL NOT allow a user to delete or deactivate their own account

---

### Requirement 11: Password Security

**User Story:** As a security engineer, I want passwords to be stored and validated securely, so that user credentials are protected against breaches.

#### Acceptance Criteria

1. THE Auth_Service SHALL hash all passwords using bcrypt with a minimum cost factor of 10
2. WHEN a password is submitted for registration or reset, THE Auth_Service SHALL validate that the password is at least 8 characters long
3. THE Auth_Service SHALL NOT store plaintext passwords in any database table, log file, or API response
4. WHEN a user changes their password via `/v1/settings/security/change-password`, THE Auth_Service SHALL require the current password, verify the current password against the stored hash, hash the new password, update the password_hash, and invalidate all other active sessions for that user
5. FOR ALL valid passwords, hashing the password with bcrypt then comparing the original password against the hash SHALL return true (round-trip property)
6. FOR ALL pairs of different passwords, hashing each password SHALL produce different hashes (collision resistance property)

---

### Requirement 12: Token Security

**User Story:** As a security engineer, I want tokens (email verification, password reset, refresh) to be stored securely, so that intercepted database data cannot be used to forge tokens.

#### Acceptance Criteria

1. THE Auth_Service SHALL generate all tokens (email verification, password reset, refresh) as cryptographically random strings of at least 32 bytes using a secure random generator
2. THE Auth_Service SHALL store only the SHA-256 hash of each token in the database; the plaintext token SHALL only be sent to the user via email or API response
3. WHEN validating a token, THE Auth_Service SHALL hash the provided plaintext token with SHA-256 and compare the hash against the stored hash in the database
4. THE Auth_Service SHALL mark tokens as used after successful consumption; a used token SHALL NOT be accepted for any subsequent request
5. FOR ALL generated tokens, hashing the token then looking up the hash in the database SHALL find the correct record (round-trip property)

---

### Requirement 13: Super Admin Impersonation

**User Story:** As a Super Admin, I want to impersonate a Tenant Admin, so that I can troubleshoot issues from the tenant's perspective without knowing their password.

#### Acceptance Criteria

1. WHEN a Super_Admin sends a POST request to `/v1/admin/impersonate` with a target tenant_id and user_id, THE Auth_Service SHALL generate a JWT_Token with the target user's tenant_id, user_id, and role, plus an additional claim `impersonator_id` containing the Super_Admin's user_id
2. WHILE a Super_Admin is impersonating a user, THE Auth_Service SHALL log all actions with both the impersonator_id and the impersonated user_id in the audit trail
3. THE Auth_Service SHALL only allow impersonation of users with role Tenant_Admin; impersonation of other Super_Admin accounts SHALL be rejected with HTTP 403
4. WHEN a Super_Admin sends a POST request to `/v1/admin/stop-impersonate`, THE Auth_Service SHALL generate a new JWT_Token with the Super_Admin's original claims and return HTTP 200

---

### Requirement 14: Reseller Authentication

**User Story:** As a Reseller, I want to log in to my dedicated dashboard, so that I can manage vouchers and deposits.

#### Acceptance Criteria

1. WHEN a Reseller logs in successfully, THE Auth_Service SHALL return a redirect_path of '/reseller/dashboard'
2. WHEN a Reseller sends a request to any endpoint outside `/v1/reseller/*` and `/v1/auth/*`, THE RBAC_Middleware SHALL return HTTP 403 with error code 'FORBIDDEN'
3. THE Auth_Service SHALL NOT allow self-registration for the Reseller role; Reseller accounts SHALL only be created by a Tenant_Admin via the user management endpoints

---

### Requirement 15: Active Session Management

**User Story:** As a user, I want to view and manage my active sessions, so that I can log out from devices I no longer use.

#### Acceptance Criteria

1. WHEN a GET request is sent to `/v1/auth/sessions` with a valid JWT_Token, THE Auth_Service SHALL return a list of all active (non-expired) sessions for the current user, including session id, device_info, ip_address, created_at, and a flag indicating whether the session is the current one
2. WHEN a DELETE request is sent to `/v1/auth/sessions/{id}`, THE Auth_Service SHALL delete the specified session if it belongs to the current user, effectively logging out that device; IF the session does not belong to the current user, THEN THE Auth_Service SHALL return HTTP 403
3. WHEN a DELETE request is sent to `/v1/auth/sessions` (without id) with a query parameter `other=true`, THE Auth_Service SHALL delete all sessions for the current user except the current session

---

### Requirement 16: API Response Format and Error Handling

**User Story:** As a frontend developer, I want consistent API response formats for all auth endpoints, so that I can handle success and error states uniformly.

#### Acceptance Criteria

1. THE Auth_Service SHALL return all successful responses in the format: `{"success": true, "data": {...}}`
2. THE Auth_Service SHALL return all error responses in the format: `{"success": false, "error": {"code": "ERROR_CODE", "message": "Human-readable message", "details": [...]}}`
3. WHEN a validation error occurs, THE Auth_Service SHALL return HTTP 422 with error code 'VALIDATION_ERROR' and a details array containing objects with `field` and `message` for each invalid field
4. THE Auth_Service SHALL NOT expose internal error details (stack traces, SQL errors, internal paths) in any API response; internal errors SHALL be logged server-side and the API SHALL return a generic error message with HTTP 500

---

### Requirement 17: Audit Logging for Auth Events

**User Story:** As a Tenant Admin, I want all authentication events to be logged, so that I can review security-related activity.

#### Acceptance Criteria

1. WHEN any of the following auth events occur, THE Auth_Service SHALL log the event with timestamp, user_id, tenant_id, ip_address, user_agent, and event result (success/failure): login, logout, register, verify-email, forgot-password, reset-password, change-password, user-created, user-deactivated, user-deleted, impersonate-start, impersonate-stop
2. THE Auth_Service SHALL store audit logs in a structured format that can be queried by tenant_id, user_id, event_type, and date range
3. THE Auth_Service SHALL NOT log sensitive data (passwords, tokens, password hashes) in audit log entries
