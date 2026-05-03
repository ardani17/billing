# Implementation Plan: Authentication & Role-Based Access Control (ISPBoss)

## Overview

Implementasi sistem autentikasi dan RBAC di dalam billing-api service menggunakan clean architecture (domain → repository → usecase → middleware → handler → router). Pendekatan bottom-up: mulai dari database schema, lalu domain entities, repository (sqlc), usecase, middleware, handler, dan terakhir wiring di router. Semua kode ditulis dalam Go, menggunakan library yang sudah ada di codebase (Fiber, pgx, zerolog, viper) ditambah bcrypt, validator, google-auth, dan rapid untuk testing.

## Tasks

- [x] 1. Database migrations dan sqlc queries
  - [x] 1.1 Create migration 000003_init_users (up & down)
    - Create `services/billing-api/migrations/000003_init_users.up.sql` with users table, RLS policies, unique constraint `(tenant_id, email)`, and indexes
    - Create `services/billing-api/migrations/000003_init_users.down.sql`
    - _Requirements: 1.1, 1.5, 1.6, 1.7_

  - [x] 1.2 Create migration 000004_init_sessions (up & down)
    - Create `services/billing-api/migrations/000004_init_sessions.up.sql` with sessions table, indexes on user_id, token_hash, expires_at
    - Create `services/billing-api/migrations/000004_init_sessions.down.sql`
    - _Requirements: 1.2, 1.7_

  - [x] 1.3 Create migration 000005_init_auth_tokens (up & down)
    - Create `services/billing-api/migrations/000005_init_auth_tokens.up.sql` with password_resets and email_verifications tables, indexes on token_hash and user_id
    - Create `services/billing-api/migrations/000005_init_auth_tokens.down.sql`
    - _Requirements: 1.3, 1.4, 1.7_

  - [x] 1.4 Create sqlc query files for auth tables
    - Create `services/billing-api/queries/users.sql` with all user queries (CreateUser, GetByID, GetByEmail, GetByTenantAndEmail, GetByGoogleID, UpdateUser, UpdateLastLogin, UpdatePasswordHash, UpdateUserStatus, LinkGoogleID, SetEmailVerified, DeleteUser, ListUsersByTenant, EmailExistsGlobal)
    - Create `services/billing-api/queries/sessions.sql` with all session queries (CreateSession, GetSessionByTokenHash, ListSessionsByUserID, DeleteSessionByID, DeleteSessionByTokenHash, DeleteSessionsByUserID, DeleteOtherSessions, DeleteExpiredSessions)
    - Create `services/billing-api/queries/auth_tokens.sql` with all token queries (CreatePasswordReset, GetPasswordResetByHash, MarkPasswordResetUsed, InvalidatePasswordResets, CreateEmailVerification, GetEmailVerificationByHash, MarkEmailVerificationUsed, InvalidateEmailVerifications)
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [x] 1.5 Run sqlc generate to produce repository code
    - Run `sqlc generate` in `services/billing-api/` to generate Go repository code from the new query files
    - Verify generated code compiles without errors
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2. Domain entities, DTOs, and error types
  - [x] 2.1 Create domain entity files
    - Create `services/billing-api/internal/domain/user.go` with User struct, UserRole constants (super_admin, tenant_admin, operator, teknisi, kasir, reseller), UserStatus constants (active, inactive)
    - Create `services/billing-api/internal/domain/session.go` with Session struct
    - Create `services/billing-api/internal/domain/token.go` with PasswordReset and EmailVerification structs
    - Create `services/billing-api/internal/domain/role.go` with role permission matrix, redirect path mapping, and RBAC configuration types (RBACConfig, AllowedRoles, MethodRestrictions)
    - _Requirements: 9.1, 4.2, 14.1_

  - [x] 2.2 Create auth DTOs and error types
    - Create `services/billing-api/internal/domain/auth.go` with all request/response DTOs: RegisterRequest, RegisterResponse, LoginRequest, LoginResponse, GoogleLoginRequest, ResetPasswordRequest, ChangePasswordRequest, TokenPair, CreateUserRequest, UpdateUserRequest, ImpersonateRequest, APIResponse, APIError, FieldError
    - Add domain error variables: ErrEmailAlreadyExists, ErrInvalidCredentials, ErrEmailNotVerified, ErrAccountDisabled, ErrAccountLocked, ErrTokenExpired, ErrTokenAlreadyUsed, ErrTokenNotFound, ErrUserNotFound, ErrForbidden, ErrCannotDeleteSelf, ErrCannotDeactivateSelf, ErrInvalidRole, ErrResendCooldown
    - Add ErrorResponse and SuccessResponse helper functions
    - _Requirements: 16.1, 16.2, 16.3, 16.4_

  - [x] 2.3 Add ImpersonatorID field to pkg/auth Claims
    - Update `pkg/auth/auth.go` Claims struct to add `ImpersonatorID string json:"impersonator_id,omitempty"`
    - _Requirements: 13.1_

  - [x] 2.4 Write property test for JWT token round-trip (Property 1)
    - **Property 1: JWT Token Round-Trip** — For any valid claims, GenerateToken then ValidateToken returns identical tenant_id, user_id, role
    - Add test in `pkg/auth/auth_test.go` using `pgregory.net/rapid`
    - **Validates: Requirements 8.6**

  - [x] 2.5 Write property test for API response format consistency (Property 18)
    - **Property 18: API Response Format Consistency** — Successful responses have `{"success": true, "data": {...}}`, error responses have `{"success": false, "error": {...}}`, never both data and error
    - Add test in `services/billing-api/internal/domain/auth_test.go` using rapid
    - **Validates: Requirements 16.1, 16.2**

- [x] 3. Checkpoint — Ensure migrations, sqlc generation, and domain entities compile
  - Ensure all tests pass, ask the user if questions arise.

- [x] 4. Config updates and utility functions
  - [x] 4.1 Update AppConfig with auth settings
    - Add GoogleClientID, JWTRefreshExpiry, BcryptCost, LoginMaxAttempts, LoginLockDuration fields to `services/billing-api/internal/config/config.go`
    - Add defaults: BcryptCost=10, LoginMaxAttempts=5, LoginLockDuration=15m, JWTRefreshExpiry=720h (30 days)
    - Add validation for GoogleClientID (optional but warn if empty)
    - _Requirements: 5.1, 5.2, 6.1, 11.1_

  - [x] 4.2 Create token utility functions
    - Create `services/billing-api/internal/usecase/token_util.go` with GenerateSecureToken (crypto/rand 32 bytes → hex + SHA-256 hash) and HashToken (SHA-256 only)
    - _Requirements: 12.1, 12.2, 12.3_

  - [x] 4.3 Write property test for token hash round-trip (Property 4)
    - **Property 4: Token Hash Round-Trip** — For any generated token, SHA-256 of plaintext matches the stored hash
    - Add test in `services/billing-api/internal/usecase/token_util_test.go` using rapid
    - **Validates: Requirements 12.5, 12.2, 12.3**

  - [x] 4.3.1 Create password utility functions
    - Create `services/billing-api/internal/usecase/password_util.go` with HashPassword (bcrypt, configurable cost) and VerifyPassword (bcrypt.CompareHashAndPassword)
    - _Requirements: 11.1, 11.2, 2.3_

  - [x] 4.4 Write property tests for bcrypt password (Properties 2 & 3)
    - **Property 2: Bcrypt Password Round-Trip** — For any valid password (>= 8 chars), hash then compare returns match
    - **Property 3: Bcrypt Collision Resistance** — For any two distinct passwords, hashes are different
    - Add tests in `services/billing-api/internal/usecase/password_util_test.go` using rapid
    - **Validates: Requirements 11.5, 11.6, 11.1, 2.3**

- [x] 5. Update go.mod dependencies
  - Add `golang.org/x/crypto` (bcrypt), `github.com/go-playground/validator/v10`, `google.golang.org/api` (Google OAuth id_token verification), `pgregory.net/rapid` (property testing) to `services/billing-api/go.mod`
  - Add `pgregory.net/rapid` to `pkg/auth/go.mod` for JWT property tests
  - Run `go mod tidy` in both modules
  - _Requirements: 11.1, 2.2, 6.1_

- [x] 6. Checkpoint — Ensure config, utilities, and dependencies compile
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. Repository layer (wrapping sqlc-generated code)
  - [x] 7.1 Create user repository wrapper
    - Create `services/billing-api/internal/repository/user_repo.go` implementing domain.UserRepository interface, wrapping sqlc-generated queries
    - Handle RLS bypass for global email check (use separate connection without tenant context)
    - Map sqlc-generated types to domain.User
    - _Requirements: 1.1, 1.5, 1.6_

  - [x] 7.2 Create session repository wrapper
    - Create `services/billing-api/internal/repository/session_repo.go` implementing domain.SessionRepository interface
    - Map sqlc-generated types to domain.Session
    - _Requirements: 1.2_

  - [x] 7.3 Create token repository wrapper
    - Create `services/billing-api/internal/repository/token_repo.go` implementing domain.TokenRepository interface
    - Map sqlc-generated types to domain.PasswordReset and domain.EmailVerification
    - _Requirements: 1.3, 1.4_

- [x] 8. Rate limiter middleware
  - [x] 8.1 Implement Redis-based login rate limiter
    - Create `services/billing-api/internal/middleware/rate_limiter.go` with LoginRateLimiter struct
    - Implement Check (GET counter, compare threshold), Increment (INCR with TTL), Reset (DEL key)
    - Key format: `rate:login:{email}`, TTL: 15 minutes, threshold: 5 attempts
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [x] 8.2 Write property test for rate limiter (Property 9)
    - **Property 9: Rate Limiter Enforces Lockout Correctly** — After N < 5 failures allow next, after 5 failures block, after success reset to zero
    - Add test in `services/billing-api/internal/middleware/rate_limiter_test.go` using rapid + miniredis
    - **Validates: Requirements 5.1, 5.2, 5.3**

- [x] 9. RBAC middleware
  - [x] 9.1 Implement RBAC middleware
    - Create `services/billing-api/internal/middleware/rbac.go` with RBAC function returning fiber.Handler
    - Extract role from JWT claims (c.Locals), check against AllowedRoles, check MethodRestrictions per role
    - Super_admin bypasses all checks; reseller restricted to `/v1/reseller/*` and `/v1/auth/*`
    - Return 403 FORBIDDEN if role not allowed or method restricted
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 14.2_

  - [x] 9.2 Write property test for RBAC access rules (Property 10)
    - **Property 10: RBAC Enforces Endpoint-Role-Method Access Rules** — For any (role, path, method) combination, RBAC allows iff role in allowed roles AND method permitted. Super_admin always allowed. Reseller only /v1/reseller/* and /v1/auth/*
    - Add test in `services/billing-api/internal/middleware/rbac_test.go` using rapid
    - **Validates: Requirements 9.2, 9.3, 9.4, 9.6, 14.2**

- [x] 10. Checkpoint — Ensure repository wrappers and middleware compile
  - Ensure all tests pass, ask the user if questions arise.

- [x] 11. Auth usecase
  - [x] 11.1 Implement AuthUsecase — Register
    - Create `services/billing-api/internal/usecase/auth_usecase.go` with AuthUsecase struct and constructor
    - Implement Register: validate input (including agree_terms must be true), check email uniqueness (global), create tenant (plan=starter, status=trial), hash password (bcrypt), create user (role=tenant_admin, email_verified=false), generate email verification token, enqueue verification email via pkg/queue
    - Use database transaction for tenant + user creation
    - Reject registration if agree_terms is false with validation error
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

  - [x] 11.2 Implement AuthUsecase — Login
    - Implement Login: check rate limiter, get user by email, verify bcrypt password, check email_verified, check status=active, generate JWT (with remember_me expiry logic), generate refresh token, create session, update last_login, reset rate limiter, return tokens + redirect_path based on role
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.3_

  - [x] 11.3 Implement AuthUsecase — Google OAuth
    - Implement LoginWithGoogle: verify Google id_token with Google public keys, extract email/name/google_id, handle 3 cases (new user → create tenant+user, existing with google_id → login, existing without google_id → link account), generate tokens
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x] 11.4 Implement AuthUsecase — Email verification
    - Implement VerifyEmail: hash token, lookup in email_verifications, check expiry, check used flag, set email_verified=true, mark token used, generate JWT+refresh, create session
    - Implement ResendVerification: check cooldown (Redis, 60s), invalidate old tokens, generate new token, enqueue email
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6_

  - [x] 11.5 Implement AuthUsecase — Forgot/Reset password
    - Implement ForgotPassword: generate token, store hash (expires 1h), invalidate old tokens, enqueue email; return 200 even if email not found (prevent enumeration)
    - Implement ResetPassword: hash token, lookup, check expiry/used, hash new password, update password_hash, mark token used, invalidate all sessions, generate new tokens+session
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6_

  - [x] 11.6 Implement AuthUsecase — Token refresh, logout, get current user, change password
    - Implement RefreshToken: hash refresh token, lookup session, check expiry, check user active, rotate token (delete old session, create new), generate new JWT
    - Implement Logout: hash refresh token, delete session
    - Implement GetCurrentUser: get user by ID from claims
    - Implement ChangePassword: verify current password, hash new password, update, invalidate other sessions
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 11.4_

  - [x] 11.7 Write property tests for auth usecase (Properties 5, 7, 8)
    - **Property 5: Token Single-Use Enforcement** — After token consumed once, second use rejected
    - **Property 7: Input Validation Rejects Invalid Data** — Invalid fields return 422 with field-level errors
    - **Property 8: Login Returns Correct Redirect Path Per Role** — Each role maps to correct redirect_path
    - Add tests in `services/billing-api/internal/usecase/auth_usecase_test.go` using rapid + mocks
    - **Validates: Requirements 12.4, 3.1, 2.2, 11.2, 16.3, 4.1, 4.2, 14.1**

- [x] 12. User management usecase
  - [x] 12.1 Implement UserManagementUsecase
    - Create `services/billing-api/internal/usecase/user_usecase.go` with UserManagementUsecase struct
    - Implement CreateUser: validate input, reject super_admin role, hash password, create user (email_verified=true for admin-created), enqueue no verification email
    - Implement UpdateUser: update name/phone/role only, preserve tenant_id
    - Implement DeactivateUser: reject self-deactivation, set status=inactive, delete all sessions
    - Implement ActivateUser: set status=active
    - Implement DeleteUser: reject self-deletion, require confirmName match, delete user (cascade)
    - Implement ResetUserPassword: generate token, enqueue reset email
    - Implement ListUsers, GetUser
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6, 10.7, 10.8_

  - [x] 12.2 Write property tests for user management (Properties 12, 13)
    - **Property 12: User Update Preserves Tenant ID Invariant** — Update never changes tenant_id
    - **Property 13: User Deactivation Invalidates All Sessions** — Deactivate sets status=inactive AND deletes all sessions
    - Add tests in `services/billing-api/internal/usecase/user_usecase_test.go` using rapid + mocks
    - **Validates: Requirements 10.2, 10.3**

- [x] 13. Impersonation usecase
  - [x] 13.1 Implement ImpersonationUsecase
    - Create `services/billing-api/internal/usecase/impersonation_usecase.go`
    - Implement StartImpersonation: verify target is tenant_admin (reject super_admin targets), generate JWT with target's claims + impersonator_id in Claims
    - Implement StopImpersonation: generate JWT with super_admin's original claims
    - _Requirements: 13.1, 13.2, 13.3, 13.4_

- [x] 14. Checkpoint — Ensure all usecases compile and pass tests
  - Ensure all tests pass, ask the user if questions arise.

- [x] 15. Audit logging utility
  - [x] 15.1 Create audit logger
    - Create `services/billing-api/internal/usecase/audit.go` with AuditLogger struct using zerolog
    - Implement LogEvent(eventType, userID, tenantID, ipAddress, userAgent, result, metadata) — structured JSON log
    - Supported events: login, logout, register, verify-email, forgot-password, reset-password, change-password, user-created, user-deactivated, user-deleted, impersonate-start, impersonate-stop
    - Ensure passwords, tokens, and hashes are NEVER logged
    - _Requirements: 17.1, 17.2, 17.3_

  - [x] 15.2 Write property test for audit log completeness (Property 19)
    - **Property 19: Audit Log Completeness** — For any auth event, log entry contains timestamp, user_id, tenant_id, ip_address, user_agent, result; never contains passwords/tokens/hashes
    - Add test in `services/billing-api/internal/usecase/audit_test.go` using rapid
    - **Validates: Requirements 17.1, 17.3**

- [x] 16. HTTP handlers
  - [x] 16.1 Implement AuthHandler
    - Create `services/billing-api/internal/handler/auth_handler.go`
    - Implement Register, Login, LoginWithGoogle, VerifyEmail, ResendVerification, ForgotPassword, ResetPassword, RefreshToken, Logout, GetMe, ChangePassword
    - Parse request body, validate with go-playground/validator, call usecase, map domain errors to HTTP error codes using ErrorResponse/SuccessResponse helpers
    - Extract device_info from User-Agent header, ip_address from c.IP()
    - _Requirements: 2.1, 2.6, 3.1, 4.1, 6.1, 7.1, 7.3, 8.1, 8.2, 8.3, 8.4, 11.4, 16.1, 16.2, 16.3, 16.4_

  - [x] 16.2 Implement UserHandler
    - Create `services/billing-api/internal/handler/user_handler.go`
    - Implement List, Create, Get, Update, Delete, Deactivate, Activate, ResetPassword
    - Extract tenant_id from JWT claims for scoping
    - _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_

  - [x] 16.3 Implement SessionHandler
    - Create `services/billing-api/internal/handler/session_handler.go`
    - Implement List (with is_current flag), Revoke (single session), RevokeOthers (all except current)
    - _Requirements: 15.1, 15.2, 15.3_

  - [x] 16.4 Implement AdminHandler (impersonation)
    - Create `services/billing-api/internal/handler/admin_handler.go`
    - Implement Start (POST /v1/admin/impersonate), Stop (POST /v1/admin/stop-impersonate)
    - _Requirements: 13.1, 13.4_

  - [x] 16.5 Write property tests for session handlers (Properties 14, 15, 16, 17)
    - **Property 14: Password Change Invalidates Other Sessions** — Change password deletes all sessions except current
    - **Property 15: Session Listing Correctly Identifies Current Session** — Exactly one session has is_current=true
    - **Property 16: Session Revocation Respects Ownership** — Can only delete own sessions, 403 for others
    - **Property 17: Revoke-Others Preserves Current Session** — Revoke others deletes N-1 sessions, keeps current
    - Add tests in `services/billing-api/internal/handler/session_handler_test.go` using rapid + mocks
    - **Validates: Requirements 11.4, 15.1, 15.2, 15.3**

- [x] 17. Router wiring and middleware integration
  - [x] 17.1 Update router.go with all auth routes
    - Update `services/billing-api/internal/handler/router.go`:
    - Add RouterConfig fields for all new handlers (AuthHandler, UserHandler, SessionHandler, AdminHandler) and RateLimiter
    - Register public auth routes (register, login, google, verify-email, resend-verification, forgot-password, reset-password, refresh) — no auth middleware
    - Register protected auth routes (me, logout, sessions) — auth middleware only
    - Register settings routes (change-password, users CRUD) — auth + RBAC middleware (tenant_admin for users)
    - Register admin routes (impersonate, stop-impersonate) — auth + RBAC middleware (super_admin)
    - Apply rate limiter middleware to login endpoint
    - Apply RBAC middleware with correct role configurations per endpoint group
    - _Requirements: 9.2, 9.3, 9.6, 14.2_

  - [x] 17.2 Update main.go to wire all dependencies
    - Update `services/billing-api/cmd/main.go`:
    - Instantiate repositories (UserRepo, SessionRepo, TokenRepo)
    - Instantiate rate limiter with Redis client
    - Instantiate usecases (AuthUsecase, UserManagementUsecase, ImpersonationUsecase) with dependencies
    - Instantiate handlers (AuthHandler, UserHandler, SessionHandler, AdminHandler)
    - Pass all handlers and rate limiter to RouterConfig
    - _Requirements: all_

- [x] 18. Final checkpoint — Ensure full build compiles and all tests pass
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation after major milestones
- Property tests validate universal correctness properties from the design document using `pgregory.net/rapid`
- Unit tests validate specific examples and edge cases using `testify`
- Bottom-up approach: database → domain → repository → usecase → middleware → handler → router
- sqlc generates repository code from SQL queries — manual wrappers map to domain types
