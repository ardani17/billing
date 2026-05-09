// reseller_auth.go berisi business logic untuk autentikasi reseller.
// Mengimplementasikan Login, Logout, RefreshToken pada ResellerAuthUsecase.
// Reseller menggunakan phone+password untuk login, terpisah dari admin auth.
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/auth"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/middleware"
)

// ResellerAuthUsecaseConfig berisi semua dependensi yang dibutuhkan ResellerAuthUsecase.
type ResellerAuthUsecaseConfig struct {
	ResellerRepo domain.ResellerRepository
	SessionRepo  domain.ResellerSessionRepository
	RateLimiter  *middleware.LoginRateLimiter
	JWTSecret    string
	JWTExpiry    time.Duration
	// SessionExpiry adalah durasi session reseller (bawaan 24 jam).
	SessionExpiry time.Duration
}

// ResellerAuthUsecase mengimplementasikan business logic autentikasi reseller.
type ResellerAuthUsecase struct {
	resellerRepo  domain.ResellerRepository
	sessionRepo   domain.ResellerSessionRepository
	rateLimiter   *middleware.LoginRateLimiter
	jwtSecret     string
	jwtExpiry     time.Duration
	sessionExpiry time.Duration
	logger        zerolog.Logger
}

// NewResellerAuthUsecase membuat instance baru ResellerAuthUsecase dengan konfigurasi yang diberikan.
func NewResellerAuthUsecase(cfg ResellerAuthUsecaseConfig, logger zerolog.Logger) *ResellerAuthUsecase {
	sessionExpiry := cfg.SessionExpiry
	if sessionExpiry == 0 {
		sessionExpiry = 24 * time.Hour // bawaan 24 jam untuk reseller
	}

	return &ResellerAuthUsecase{
		resellerRepo:  cfg.ResellerRepo,
		sessionRepo:   cfg.SessionRepo,
		rateLimiter:   cfg.RateLimiter,
		jwtSecret:     cfg.JWTSecret,
		jwtExpiry:     cfg.JWTExpiry,
		sessionExpiry: sessionExpiry,
		logger:        logger,
	}
}

// Login memverifikasi credential reseller dan mengembalikan JWT + refresh token.
// Alur: cek rate limiter (phone-based) -> ambil reseller by phone -> verifikasi status aktif ->
// verifikasi password -> buat session (24h expiry) -> buat JWT dengan claims reseller ->
// perbarui last_login -> reset rate limiter -> kembalikan tokens + reseller.
func (uc *ResellerAuthUsecase) Login(ctx context.Context, req domain.ResellerLoginRequest) (*domain.ResellerLoginResponse, error) {
	// Cek rate limiter sebelum proses login (menggunakan phone sebagai key)
	allowed, remainingSec, err := uc.rateLimiter.Check(ctx, req.Phone)
	if err != nil {
		return nil, fmt.Errorf("gagal cek rate limiter: %w", err)
	}
	if !allowed {
		return nil, fmt.Errorf("%w: coba lagi dalam %d detik", domain.ErrResellerAccountLocked, remainingSec)
	}

	// Ambil reseller berdasarkan nomor telepon (lintas tenant, bypass RLS).
	// Login reseller bersifat publik tanpa konteks tenant, sehingga menggunakan
	// GetByPhoneGlobal yang mencari berdasarkan phone saja.
	reseller, err := uc.resellerRepo.GetByPhoneGlobal(ctx, req.Phone)
	if err != nil {
		if errors.Is(err, domain.ErrResellerNotFound) {
			// Increment rate limiter meskipun phone tidak ditemukan (cegah enumerasi)
			_ = uc.rateLimiter.Increment(ctx, req.Phone)
			return nil, domain.ErrResellerInvalidCredentials
		}
		return nil, fmt.Errorf("gagal mengambil reseller: %w", err)
	}

	// Verifikasi status reseller harus aktif
	if reseller.Status != domain.ResellerStatusAktif {
		return nil, domain.ErrResellerAccountDisabled
	}

	// Verifikasi password dengan bcrypt
	if err := VerifyPassword(reseller.PasswordHash, req.Password); err != nil {
		// Increment rate limiter saat password salah
		_ = uc.rateLimiter.Increment(ctx, req.Phone)
		return nil, domain.ErrResellerInvalidCredentials
	}

	// Buat JWT token dengan claims reseller
	accessToken, err := uc.generateResellerJWT(reseller)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	// Buat session baru dengan refresh token (expiry 24 jam)
	refreshToken, _, err := uc.createResellerSession(ctx, reseller.ID)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	// Perbarui last_login pada reseller
	if err := uc.resellerRepo.UpdateLastLogin(ctx, reseller.ID); err != nil {
		uc.logger.Error().Err(err).Str("reseller_id", reseller.ID).Msg("gagal update last_login reseller")
	}

	// Reset rate limiter setelah login berhasil
	if err := uc.rateLimiter.Reset(ctx, req.Phone); err != nil {
		uc.logger.Error().Err(err).Str("phone", req.Phone).Msg("gagal reset rate limiter reseller")
	}

	return &domain.ResellerLoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
		Reseller:     reseller,
	}, nil
}

// Logout menghapus session reseller berdasarkan refresh token.
func (uc *ResellerAuthUsecase) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := HashToken(refreshToken)
	if err := uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return fmt.Errorf("gagal menghapus session: %w", err)
	}
	return nil
}

// RefreshToken memperpanjang JWT reseller dengan refresh token.
// Alur: hash refresh token -> lookup session -> verifikasi belum expired ->
// rotate token (hapus session lama, buat baru) -> buat JWT baru.
func (uc *ResellerAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
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

	// Verifikasi session belum expired
	if time.Now().After(session.ExpiresAt) {
		// Hapus session yang sudah expired
		_ = uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, fmt.Errorf("refresh token sudah kedaluwarsa")
	}

	// Ambil data reseller dan verifikasi masih aktif
	reseller, err := uc.resellerRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil reseller: %w", err)
	}
	if reseller.Status != domain.ResellerStatusAktif {
		// Hapus session jika reseller tidak aktif
		_ = uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash)
		return nil, domain.ErrResellerAccountDisabled
	}

	// Rotate token: hapus session lama
	if err := uc.sessionRepo.DeleteByTokenHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("gagal menghapus session lama: %w", err)
	}

	// Buat session baru dengan token baru
	newRefreshToken, _, err := uc.createResellerSession(ctx, reseller.ID)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat session baru: %w", err)
	}

	// Buat JWT baru
	accessToken, err := uc.generateResellerJWT(reseller)
	if err != nil {
		return nil, fmt.Errorf("gagal generate JWT: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(uc.jwtExpiry.Seconds()),
	}, nil
}

// --- Fungsi bantu Methods ---

// generateResellerJWT membuat JWT token untuk reseller dengan claims khusus reseller.
// Claims: reseller_id (sebagai UserID), tenant_id, name (tidak digunakan di claims standar),
// role=reseller.
func (uc *ResellerAuthUsecase) generateResellerJWT(reseller *domain.Reseller) (string, error) {
	tokenCfg := auth.TokenConfig{
		Secret: uc.jwtSecret,
		Expiry: uc.jwtExpiry,
		Issuer: "ispboss",
	}

	// Gunakan reseller_id sebagai UserID dalam claims standar,
	// dan role=reseller untuk membedakan dari admin JWT.
	claims := auth.Claims{
		TenantID: reseller.TenantID,
		UserID:   reseller.ID,
		Role:     "reseller",
	}

	return auth.GenerateToken(tokenCfg, claims)
}

// createResellerSession membuat session baru untuk reseller dengan refresh token.
// Session expiry diset ke 24 jam (konfigurasi bawaan reseller).
// Mengembalikan plaintext refresh token dan session yang dibuat.
func (uc *ResellerAuthUsecase) createResellerSession(ctx context.Context, resellerID string) (string, *domain.Session, error) {
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return "", nil, fmt.Errorf("gagal generate refresh token: %w", err)
	}

	session, err := uc.sessionRepo.CreateSession(ctx, &domain.Session{
		UserID:    resellerID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(uc.sessionExpiry),
	})
	if err != nil {
		return "", nil, fmt.Errorf("gagal membuat session: %w", err)
	}

	return plainToken, session, nil
}
