// Package usecase berisi business logic untuk billing-api.
// AuthUsecase mengimplementasikan semua operasi autentikasi:
// register, login, Google OAuth, verifikasi email, reset password,
// refresh token, logout, dan ubah password.
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/idtoken"

	"github.com/ispboss/ispboss/pkg/auth"
	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/middleware"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
)

// AuthUsecaseConfig berisi semua dependensi yang dibutuhkan AuthUsecase.
type AuthUsecaseConfig struct {
	UserRepo         domain.UserRepository
	SessionRepo      domain.SessionRepository
	TokenRepo        domain.TokenRepository
	RateLimiter      *middleware.LoginRateLimiter
	QueueClient      *asynq.Client
	Pool             *pgxpool.Pool
	RedisClient      *redis.Client
	JWTSecret        string
	JWTExpiry        time.Duration
	JWTRefreshExpiry time.Duration
	BcryptCost       int
	GoogleClientID   string
}

// AuthUsecase mengimplementasikan business logic autentikasi.
type AuthUsecase struct {
	userRepo         domain.UserRepository
	sessionRepo      domain.SessionRepository
	tokenRepo        domain.TokenRepository
	rateLimiter      *middleware.LoginRateLimiter
	queueClient      *asynq.Client
	pool             *pgxpool.Pool
	redisClient      *redis.Client
	jwtSecret        string
	jwtExpiry        time.Duration
	jwtRefreshExpiry time.Duration
	bcryptCost       int
	googleClientID   string
}

// NewAuthUsecase membuat instance baru AuthUsecase dengan konfigurasi yang diberikan.
func NewAuthUsecase(cfg AuthUsecaseConfig) *AuthUsecase {
	return &AuthUsecase{
		userRepo:         cfg.UserRepo,
		sessionRepo:      cfg.SessionRepo,
		tokenRepo:        cfg.TokenRepo,
		rateLimiter:      cfg.RateLimiter,
		queueClient:      cfg.QueueClient,
		pool:             cfg.Pool,
		redisClient:      cfg.RedisClient,
		jwtSecret:        cfg.JWTSecret,
		jwtExpiry:        cfg.JWTExpiry,
		jwtRefreshExpiry: cfg.JWTRefreshExpiry,
		bcryptCost:       cfg.BcryptCost,
		googleClientID:   cfg.GoogleClientID,
	}
}

// resendCooldownKey mengembalikan Redis key untuk cooldown resend verification.
func resendCooldownKey(email string) string {
	return fmt.Sprintf("cooldown:resend:%s", email)
}

// --- Task 11.1: Register ---

// Register mendaftarkan tenant baru beserta user tenant_admin.
// Proses: validasi input -> cek email unik global -> buat tenant (dalam transaksi) ->
// hash password -> buat user -> buat token verifikasi email -> antrekan email.
func (uc *AuthUsecase) Register(ctx context.Context, req domain.RegisterRequest) (*domain.RegisterResponse, error) {
	// Validasi agree_terms harus true
	if !req.AgreeTerms {
		return nil, fmt.Errorf("agree_terms harus disetujui")
	}

	// Cek apakah email sudah terdaftar di tenant manapun (bypass RLS)
	exists, err := uc.userRepo.EmailExistsGlobal(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("gagal mengecek email: %w", err)
	}
	if exists {
		return nil, domain.ErrEmailAlreadyExists
	}

	// Hash password dengan bcrypt
	hashedPassword, err := HashPassword(req.Password, uc.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("gagal hash password: %w", err)
	}

	// Mulai transaksi database untuk membuat tenant + user secara atomik
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Buat tenant baru menggunakan sqlc queries dalam transaksi
	queries := repository.New(tx)
	tenantRow, err := queries.CreateTenant(ctx, repository.CreateTenantParams{
		Name:   req.CompanyName,
		Plan:   "starter",
		Status: "trial",
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tenant: %w", err)
	}

	tenantID := pgUUIDToString(tenantRow.ID)

	// Set RLS context agar INSERT ke tabel users berhasil
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
	if err != nil {
		return nil, fmt.Errorf("gagal set tenant context: %w", err)
	}

	// Buat user tenant_admin dalam transaksi yang sama
	userQueries := repository.New(tx)
	userRow, err := userQueries.CreateUser(ctx, repository.CreateUserParams{
		TenantID:      tenantRow.ID,
		Name:          req.Name,
		Email:         req.Email,
		Phone:         toPgText(req.Phone),
		PasswordHash:  toPgText(hashedPassword),
		Role:          string(domain.RoleTenantAdmin),
		EmailVerified: false,
		Status:        string(domain.UserStatusActive),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}

	userID := pgUUIDToString(userRow.ID)

	// Commit transaksi
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	// Buat token verifikasi email
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("gagal generate token verifikasi: %w", err)
	}

	// Simpan hash token di database
	err = uc.tokenRepo.CreateEmailVerification(ctx, &domain.EmailVerification{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		log.Error().Err(err).Msg("gagal menyimpan token verifikasi email")
	}

	// Enqueue email verifikasi via queue
	uc.enqueueVerificationEmail(tenantID, userID, req.Email, req.Name, plainToken)

	return &domain.RegisterResponse{
		UserID:   userID,
		TenantID: tenantID,
	}, nil
}

// --- Task 11.2: Login ---

// Login memverifikasi credential dan mengembalikan JWT + refresh token.
// Proses: cek rate limiter -> ambil user by email -> verifikasi password ->
// cek email_verified -> cek status -> buat JWT -> buat session -> perbarui last_login.
func (uc *AuthUsecase) Login(ctx context.Context, req domain.LoginRequest, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Cek rate limiter sebelum proses login
	allowed, remainingSec, err := uc.rateLimiter.Check(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("gagal cek rate limiter: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("%w: coba lagi dalam %d detik", domain.ErrAccountLocked, remainingSec)
	}

	// Ambil user berdasarkan email (lintas tenant, bypass RLS)
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Increment rate limiter meskipun email tidak ditemukan (prevent enumeration)
			_ = uc.rateLimiter.Increment(ctx, req.Email)
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Verifikasi password dengan bcrypt
	if err := VerifyPassword(user.PasswordHash, req.Password); err != nil {
		_ = uc.rateLimiter.Increment(ctx, req.Email)
		return nil, domain.ErrInvalidCredentials
	}

	// Cek apakah email sudah diverifikasi
	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	// Cek apakah akun aktif
	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrAccountDisabled
	}

	// Tentukan expiry JWT berdasarkan remember_me
	jwtExpiry := uc.jwtExpiry
	if req.RememberMe {
		jwtExpiry = 7 * 24 * time.Hour // 7 hari jika remember_me
	}

	// Buat JWT token
	accessToken, err := uc.generateJWT(user, jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat refresh token dan buat session
	refreshToken, _, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	// Perbarui last_login
	if err := uc.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("gagal update last_login")
	}

	// Reset rate limiter setelah login berhasil
	if err := uc.rateLimiter.Reset(ctx, req.Email); err != nil {
		log.Error().Err(err).Str("email", req.Email).Msg("gagal reset rate limiter")
	}

	// Tentukan redirect path berdasarkan role
	redirectPath := domain.RedirectPathMap[user.Role]

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(jwtExpiry.Seconds()),
		User:         user,
		RedirectPath: redirectPath,
	}, nil
}

// --- Task 11.3: Google OAuth ---

// googleClaims berisi data yang diekstrak dari Google id_token.
type googleClaims struct {
	Email    string
	Name     string
	GoogleID string
}

// GoogleIDTokenValidator adalah fungsi untuk memverifikasi Google id_token.
// Bisa di-override untuk testing.
var GoogleIDTokenValidator = defaultGoogleIDTokenValidator

// defaultGoogleIDTokenValidator memverifikasi Google id_token menggunakan library resmi.
func defaultGoogleIDTokenValidator(ctx context.Context, idTokenStr, clientID string) (map[string]interface{}, error) {
	payload, err := idtoken.Validate(ctx, idTokenStr, clientID)
	if err != nil {
		return nil, err
	}
	return payload.Claims, nil
}

// verifyGoogleIDToken memverifikasi Google id_token dan mengekstrak claims.
func (uc *AuthUsecase) verifyGoogleIDToken(ctx context.Context, idTokenStr string) (*googleClaims, error) {
	claimsMap, err := GoogleIDTokenValidator(ctx, idTokenStr, uc.googleClientID)
	if err != nil {
		return nil, fmt.Errorf("gagal verifikasi Google token: %w", err)
	}

	email, _ := claimsMap["email"].(string)
	name, _ := claimsMap["name"].(string)
	sub, _ := claimsMap["sub"].(string)

	if email == "" || sub == "" {
		return nil, fmt.Errorf("Google token tidak mengandung email atau sub")
	}

	if name == "" {
		name = email // Cadangan ke email jika nama kosong
	}

	return &googleClaims{
		Email:    email,
		Name:     name,
		GoogleID: sub,
	}, nil
}

// LoginWithGoogle memverifikasi Google id_token dan login/register user.
// Menangani 3 kasus: user baru, user existing dengan google_id, user existing tanpa google_id.
func (uc *AuthUsecase) LoginWithGoogle(ctx context.Context, req domain.GoogleLoginRequest, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Verifikasi Google id_token
	claims, err := uc.verifyGoogleIDToken(ctx, req.IDToken)
	if err != nil {
		return nil, fmt.Errorf("Google token tidak valid: %w", err)
	}

	// Cek apakah user sudah ada berdasarkan google_id
	user, err := uc.userRepo.GetByGoogleID(ctx, claims.GoogleID)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("gagal mengambil user by Google ID: %w", err)
	}

	if user != nil {
		// Kasus 2: User existing dengan google_id -> langsung login
		return uc.loginExistingUser(ctx, user, deviceInfo, ipAddress)
	}

	// Cek apakah user sudah ada berdasarkan email
	user, err = uc.userRepo.GetByEmail(ctx, claims.Email)
	if err != nil && !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("gagal mengambil user by email: %w", err)
	}

	if user != nil {
		// Kasus 3: User existing tanpa google_id -> link Google account lalu login
		if err := uc.userRepo.LinkGoogleID(ctx, user.ID, claims.GoogleID); err != nil {
			return nil, fmt.Errorf("gagal menautkan Google ID: %w", err)
		}
		return uc.loginExistingUser(ctx, user, deviceInfo, ipAddress)
	}

	// Kasus 1: User baru -> buat tenant + user baru
	return uc.registerGoogleUser(ctx, claims, deviceInfo, ipAddress)
}

// loginExistingUser membuat JWT dan session untuk user yang sudah ada.
func (uc *AuthUsecase) loginExistingUser(ctx context.Context, user *domain.User, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Cek apakah akun aktif
	if user.Status != domain.UserStatusActive {
		return nil, domain.ErrAccountDisabled
	}

	// Buat JWT token
	accessToken, err := uc.generateJWT(user, uc.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat refresh token dan buat session
	refreshToken, _, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	// Perbarui last_login
	if err := uc.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		log.Error().Err(err).Str("user_id", user.ID).Msg("gagal update last_login")
	}

	redirectPath := domain.RedirectPathMap[user.Role]

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
		User:         user,
		RedirectPath: redirectPath,
	}, nil
}

// registerGoogleUser membuat tenant dan user baru dari Google OAuth.
func (uc *AuthUsecase) registerGoogleUser(ctx context.Context, claims *googleClaims, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Mulai transaksi untuk membuat tenant + user
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Buat tenant baru
	queries := repository.New(tx)
	tenantRow, err := queries.CreateTenant(ctx, repository.CreateTenantParams{
		Name:   claims.Name,
		Plan:   "starter",
		Status: "trial",
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tenant: %w", err)
	}

	tenantID := pgUUIDToString(tenantRow.ID)

	// Set RLS context untuk INSERT user
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID))
	if err != nil {
		return nil, fmt.Errorf("gagal set tenant context: %w", err)
	}

	// Buat user dengan Google OAuth (email_verified=true, password_hash kosong)
	userQueries := repository.New(tx)
	userRow, err := userQueries.CreateUser(ctx, repository.CreateUserParams{
		TenantID:      tenantRow.ID,
		Name:          claims.Name,
		Email:         claims.Email,
		Role:          string(domain.RoleTenantAdmin),
		EmailVerified: true,
		GoogleID:      toPgText(claims.GoogleID),
		Status:        string(domain.UserStatusActive),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	userID := pgUUIDToString(userRow.ID)

	// Buat domain.User dari row data
	user := &domain.User{
		ID:            userID,
		TenantID:      tenantID,
		Name:          claims.Name,
		Email:         claims.Email,
		Role:          domain.RoleTenantAdmin,
		EmailVerified: true,
		GoogleID:      claims.GoogleID,
		Status:        domain.UserStatusActive,
		CreatedAt:     userRow.CreatedAt.Time,
		UpdatedAt:     userRow.UpdatedAt.Time,
	}

	// Buat JWT token
	accessToken, err := uc.generateJWT(user, uc.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat refresh token dan buat session
	refreshToken, _, err := uc.createSession(ctx, userID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	redirectPath := domain.RedirectPathMap[user.Role]

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
		User:         user,
		RedirectPath: redirectPath,
	}, nil
}

// --- Task 11.4: Email Verification ---

// VerifyEmail memverifikasi email dengan token.
// Proses: hash token -> lookup di email_verifications -> cek expiry -> cek used ->
// atur email_verified=true -> mark token used -> buat JWT + refresh -> buat session.
func (uc *AuthUsecase) VerifyEmail(ctx context.Context, token string, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Hash token untuk lookup di database
	tokenHash := HashToken(token)

	// Cari token di database
	verification, err := uc.tokenRepo.GetEmailVerificationByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("gagal mengambil token verifikasi: %w", err)
	}

	// Cek apakah token sudah digunakan
	if verification.Used {
		return nil, domain.ErrTokenAlreadyUsed
	}

	// Cek apakah token sudah kedaluwarsa
	if time.Now().After(verification.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	// Set email_verified=true pada user
	if err := uc.userRepo.SetEmailVerified(ctx, verification.UserID); err != nil {
		return nil, fmt.Errorf("gagal set email_verified: %w", err)
	}

	// Mark token sebagai sudah digunakan
	if err := uc.tokenRepo.MarkEmailVerificationUsed(ctx, verification.ID); err != nil {
		return nil, fmt.Errorf("gagal mark token used: %w", err)
	}

	// Ambil data user untuk buat JWT
	user, err := uc.userRepo.GetByID(ctx, verification.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Buat JWT token
	accessToken, err := uc.generateJWT(user, uc.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat refresh token dan buat session
	refreshToken, _, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	redirectPath := domain.RedirectPathMap[user.Role]

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
		User:         user,
		RedirectPath: redirectPath,
	}, nil
}

// ResendVerification mengirim ulang email verifikasi.
// Proses: cek cooldown (Redis 60s) -> ambil user -> invalidate token lama ->
// buat token baru -> antrekan email.
func (uc *AuthUsecase) ResendVerification(ctx context.Context, email string) error {
	// Cek cooldown di Redis (60 detik antar resend)
	cooldownKey := resendCooldownKey(email)
	ttl, err := uc.redisClient.TTL(ctx, cooldownKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("gagal cek cooldown: %w", err)
	}
	if ttl > 0 {
		return fmt.Errorf("%w: %d detik", domain.ErrResendCooldown, int(ttl.Seconds()))
	}

	// Ambil user berdasarkan email
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			// Jangan bocorkan informasi apakah email terdaftar
			return nil
		}
		return fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Jika email sudah terverifikasi, tidak perlu kirim ulang
	if user.EmailVerified {
		return nil
	}

	// Invalidate semua token verifikasi lama
	if err := uc.tokenRepo.InvalidateEmailVerifications(ctx, user.ID); err != nil {
		return fmt.Errorf("gagal invalidate token lama: %w", err)
	}

	// Buat token verifikasi baru
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("gagal generate token: %w", err)
	}

	// Simpan hash token di database
	err = uc.tokenRepo.CreateEmailVerification(ctx, &domain.EmailVerification{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	if err != nil {
		return fmt.Errorf("gagal menyimpan token verifikasi: %w", err)
	}

	// Atur cooldown di Redis (60 detik)
	if err := uc.redisClient.Set(ctx, cooldownKey, "1", 60*time.Second).Err(); err != nil {
		log.Error().Err(err).Str("email", email).Msg("gagal set cooldown resend")
	}

	// Enqueue email verifikasi
	uc.enqueueVerificationEmail(user.TenantID, user.ID, user.Email, user.Name, plainToken)

	return nil
}

// --- Task 11.5: Forgot/Reset Password ---

// ForgotPassword mengirim email reset password.
// Selalu mengembalikan nil (bahkan jika email tidak ditemukan) untuk mencegah email enumeration.
func (uc *AuthUsecase) ForgotPassword(ctx context.Context, email string) error {
	// Ambil user berdasarkan email
	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil // Mencegah email enumeration
		}
		return fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Invalidate semua token reset lama
	if err := uc.tokenRepo.InvalidatePasswordResets(ctx, user.ID); err != nil {
		return fmt.Errorf("gagal invalidate token lama: %w", err)
	}

	// Buat token reset password baru
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("gagal generate token: %w", err)
	}

	// Simpan hash token di database (berlaku 1 jam)
	err = uc.tokenRepo.CreatePasswordReset(ctx, &domain.PasswordReset{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	if err != nil {
		return fmt.Errorf("gagal menyimpan token reset: %w", err)
	}

	// Enqueue email reset password
	uc.enqueuePasswordResetEmail(user.TenantID, user.ID, user.Email, user.Name, plainToken)

	return nil
}

// ResetPassword mereset password dengan token.
// Proses: hash token -> lookup -> cek expiry/used -> hash password baru ->
// perbarui password_hash -> mark token used -> invalidate semua session ->
// buat JWT + refresh baru -> buat session.
func (uc *AuthUsecase) ResetPassword(ctx context.Context, req domain.ResetPasswordRequest, deviceInfo, ipAddress string) (*domain.LoginResponse, error) {
	// Hash token untuk lookup di database
	tokenHash := HashToken(req.Token)

	// Cari token di database
	resetToken, err := uc.tokenRepo.GetPasswordResetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("gagal mengambil token reset: %w", err)
	}

	// Cek apakah token sudah digunakan
	if resetToken.Used {
		return nil, domain.ErrTokenAlreadyUsed
	}

	// Cek apakah token sudah kedaluwarsa
	if time.Now().After(resetToken.ExpiresAt) {
		return nil, domain.ErrTokenExpired
	}

	// Hash password baru
	hashedPassword, err := HashPassword(req.Password, uc.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("gagal hash password: %w", err)
	}

	// Perbarui password_hash di database
	if err := uc.userRepo.UpdatePasswordHash(ctx, resetToken.UserID, hashedPassword); err != nil {
		return nil, fmt.Errorf("gagal update password: %w", err)
	}

	// Mark token sebagai sudah digunakan
	if err := uc.tokenRepo.MarkPasswordResetUsed(ctx, resetToken.ID); err != nil {
		return nil, fmt.Errorf("gagal mark token used: %w", err)
	}

	// Invalidate semua session yang ada (force logout dari semua device)
	if err := uc.sessionRepo.DeleteByUserID(ctx, resetToken.UserID); err != nil {
		log.Error().Err(err).Str("user_id", resetToken.UserID).Msg("gagal invalidate sessions")
	}

	// Ambil data user untuk buat JWT
	user, err := uc.userRepo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Buat JWT token
	accessToken, err := uc.generateJWT(user, uc.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat refresh token dan buat session baru
	refreshToken, _, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	redirectPath := domain.RedirectPathMap[user.Role]

	return &domain.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
		User:         user,
		RedirectPath: redirectPath,
	}, nil
}

// --- Tugas 11.6: Refresh token, logout, ambil pengguna saat ini, ubah password ---

// RefreshToken memperpanjang JWT dengan refresh token.
// Proses: hash refresh token -> lookup session -> cek user aktif ->
// rotate token (hapus session lama, buat baru) -> buat JWT baru.
func (uc *AuthUsecase) RefreshToken(ctx context.Context, refreshToken string, deviceInfo, ipAddress string) (*domain.TokenPair, error) {
	// Hash refresh token untuk lookup di database
	tokenHash := HashToken(refreshToken)

	// Cari session berdasarkan token hash (kueri sudah filter expires_at > NOW())
	session, err := uc.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, domain.ErrTokenNotFound) {
			return nil, fmt.Errorf("refresh token tidak valid")
		}
		return nil, fmt.Errorf("gagal mengambil session: %w", err)
	}

	// Ambil data user dan cek apakah masih aktif
	user, err := uc.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}
	if user.Status != domain.UserStatusActive {
		_ = uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, domain.ErrAccountDisabled
	}

	// Rotate token: hapus session lama
	if err := uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("gagal menghapus session lama: %w", err)
	}

	// Buat session baru dengan token baru
	newRefreshToken, _, err := uc.createSession(ctx, user.ID, deviceInfo, ipAddress)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session baru: %w", err)
	}

	// Buat JWT baru
	accessToken, err := uc.generateJWT(user, uc.jwtExpiry)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
	}, nil
}

// Logout menghapus session aktif berdasarkan refresh token.
func (uc *AuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := HashToken(refreshToken)
	if err := uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("gagal menghapus session: %w", err)
	}
	return nil
}

// GetCurrentUser mengambil data user berdasarkan ID dari JWT claims.
func (uc *AuthUsecase) GetCurrentUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}
	return user, nil
}

// ChangePassword mengubah password user yang sedang login.
// Proses: verifikasi password lama -> hash password baru -> perbarui -> invalidate session lain.
func (uc *AuthUsecase) ChangePassword(ctx context.Context, userID string, req domain.ChangePasswordRequest, currentRefreshToken string) error {
	// Ambil data user
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Verifikasi password lama
	if err := VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return domain.ErrInvalidCredentials
	}

	// Hash password baru
	hashedPassword, err := HashPassword(req.NewPassword, uc.bcryptCost)
	if err != nil {
		return fmt.Errorf("gagal hash password: %w", err)
	}

	// Perbarui password_hash di database
	if err := uc.userRepo.UpdatePasswordHash(ctx, userID, hashedPassword); err != nil {
		return fmt.Errorf("gagal update password: %w", err)
	}

	// Invalidate semua session lain (kecuali session saat ini)
	if currentRefreshToken != "" {
		currentTokenHash := HashToken(currentRefreshToken)
		currentSession, err := uc.sessionRepo.GetByTokenHash(ctx, currentTokenHash)
		if err == nil && currentSession != nil {
			if err := uc.sessionRepo.DeleteOtherSessions(ctx, userID, currentSession.ID); err != nil {
				log.Error().Err(err).Str("user_id", userID).Msg("gagal invalidate session lain")
			}
			return nil
		}
	}

	// Cadangan: hapus semua session jika tidak bisa identifikasi session saat ini
	if err := uc.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gagal invalidate semua session")
	}

	return nil
}

// --- Fungsi bantu Methods ---

// generateJWT membuat JWT token untuk user dengan expiry yang ditentukan.
func (uc *AuthUsecase) generateJWT(user *domain.User, expiry time.Duration) (string, error) {
	tokenCfg := auth.TokenConfig{
		Secret: uc.jwtSecret,
		Expiry: expiry,
		Issuer: "ispboss",
	}

	claims := auth.Claims{
		TenantID: user.TenantID,
		UserID:   user.ID,
		Role:     string(user.Role),
	}

	return auth.GenerateToken(tokenCfg, claims)
}

// createSession membuat session baru dengan refresh token.
// Mengembalikan plaintext refresh token dan session yang dibuat.
func (uc *AuthUsecase) createSession(ctx context.Context, userID, deviceInfo, ipAddress string) (string, *domain.Session, error) {
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return "", nil, fmt.Errorf("gagal generate refresh token: %w", err)
	}

	session, err := uc.sessionRepo.CreateSession(ctx, &domain.Session{
		UserID:     userID,
		TokenHash:  tokenHash,
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		ExpiresAt:  time.Now().Add(uc.jwtRefreshExpiry),
	})
	if err != nil {
		return "", nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	return plainToken, session, nil
}

// enqueueVerificationEmail mengirim task email verifikasi ke queue.
func (uc *AuthUsecase) enqueueVerificationEmail(tenantID, userID, email, name, token string) {
	if uc.queueClient == nil {
		log.Warn().Msg("queue client tidak tersedia, skip enqueue verification email")
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"email":   email,
		"name":    name,
		"token":   token,
	})

	err := queue.EnqueueTask(uc.queueClient, queue.TaskEnvelope{
		EventType: "email.verification",
		TenantID:  tenantID,
		Payload:   payload,
	})
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("gagal enqueue email verifikasi")
	}
}

// enqueuePasswordResetEmail mengirim task email reset password ke queue.
func (uc *AuthUsecase) enqueuePasswordResetEmail(tenantID, userID, email, name, token string) {
	if uc.queueClient == nil {
		log.Warn().Msg("queue client tidak tersedia, skip enqueue password reset email")
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"email":   email,
		"name":    name,
		"token":   token,
	})

	err := queue.EnqueueTask(uc.queueClient, queue.TaskEnvelope{
		EventType: "email.password_reset",
		TenantID:  tenantID,
		Payload:   payload,
	})
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("gagal enqueue email reset password")
	}
}

// --- Type Conversion Fungsi bantus ---

// pgUUIDToString mengkonversi pgtype.UUID ke string format UUID.
func pgUUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

// toPgText mengkonversi string ke pgtype.Text.
// String kosong dikonversi ke NULL (Valid=false).
func toPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
