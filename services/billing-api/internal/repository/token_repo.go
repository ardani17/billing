package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
)

// TokenRepo mengimplementasikan domain.TokenRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.PasswordReset
// dan domain.EmailVerification.
type TokenRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi token.
	queries *Queries
}

// NewTokenRepo membuat instance baru TokenRepo.
func NewTokenRepo(queries *Queries) *TokenRepo {
	return &TokenRepo{
		queries: queries,
	}
}

// --- Helper functions untuk mapping sqlc row -> domain types ---

// mapGetPasswordResetByHashRow memetakan PasswordReset (sqlc model) ke domain.PasswordReset.
func mapGetPasswordResetByHashRow(row PasswordReset) *domain.PasswordReset {
	return &domain.PasswordReset{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		TokenHash: row.TokenHash,
		ExpiresAt: timestamptzToTime(row.ExpiresAt),
		Used:      row.Used,
		CreatedAt: timestamptzToTime(row.CreatedAt),
	}
}

// mapGetEmailVerificationByHashRow memetakan EmailVerification (sqlc model) ke domain.EmailVerification.
func mapGetEmailVerificationByHashRow(row EmailVerification) *domain.EmailVerification {
	return &domain.EmailVerification{
		ID:        uuidToString(row.ID),
		UserID:    uuidToString(row.UserID),
		TokenHash: row.TokenHash,
		ExpiresAt: timestamptzToTime(row.ExpiresAt),
		Used:      row.Used,
		CreatedAt: timestamptzToTime(row.CreatedAt),
	}
}

// --- Implementasi domain.TokenRepository ---

// CreatePasswordReset membuat token reset password baru.
func (r *TokenRepo) CreatePasswordReset(ctx context.Context, pr *domain.PasswordReset) error {
	err := r.queries.CreatePasswordReset(ctx, CreatePasswordResetParams{
		UserID:    stringToUUID(pr.UserID),
		TokenHash: pr.TokenHash,
		ExpiresAt: timeToTimestamptz(pr.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal membuat password reset token: %w", err)
	}
	return nil
}

// GetPasswordResetByHash mengambil password reset berdasarkan token hash.
// Mengembalikan domain.ErrTokenNotFound jika token tidak ditemukan.
func (r *TokenRepo) GetPasswordResetByHash(ctx context.Context, tokenHash string) (*domain.PasswordReset, error) {
	row, err := r.queries.GetPasswordResetByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil password reset by hash: %w", err)
	}
	return mapGetPasswordResetByHashRow(row), nil
}

// MarkPasswordResetUsed menandai token reset password sebagai sudah digunakan.
func (r *TokenRepo) MarkPasswordResetUsed(ctx context.Context, id string) error {
	err := r.queries.MarkPasswordResetUsed(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menandai password reset sebagai used: %w", err)
	}
	return nil
}

// InvalidatePasswordResets menandai semua token reset yang belum dipakai untuk user tertentu.
func (r *TokenRepo) InvalidatePasswordResets(ctx context.Context, userID string) error {
	err := r.queries.InvalidatePasswordResets(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal menginvalidasi password resets: %w", err)
	}
	return nil
}

// CreateEmailVerification membuat token verifikasi email baru.
func (r *TokenRepo) CreateEmailVerification(ctx context.Context, ev *domain.EmailVerification) error {
	err := r.queries.CreateEmailVerification(ctx, CreateEmailVerificationParams{
		UserID:    stringToUUID(ev.UserID),
		TokenHash: ev.TokenHash,
		ExpiresAt: timeToTimestamptz(ev.ExpiresAt),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal membuat email verification token: %w", err)
	}
	return nil
}

// GetEmailVerificationByHash mengambil verifikasi email berdasarkan token hash.
// Mengembalikan domain.ErrTokenNotFound jika token tidak ditemukan.
func (r *TokenRepo) GetEmailVerificationByHash(ctx context.Context, tokenHash string) (*domain.EmailVerification, error) {
	row, err := r.queries.GetEmailVerificationByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil email verification by hash: %w", err)
	}
	return mapGetEmailVerificationByHashRow(row), nil
}

// MarkEmailVerificationUsed menandai token verifikasi email sebagai sudah digunakan.
func (r *TokenRepo) MarkEmailVerificationUsed(ctx context.Context, id string) error {
	err := r.queries.MarkEmailVerificationUsed(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal menandai email verification sebagai used: %w", err)
	}
	return nil
}

// InvalidateEmailVerifications menandai semua token verifikasi yang belum dipakai untuk user tertentu.
func (r *TokenRepo) InvalidateEmailVerifications(ctx context.Context, userID string) error {
	err := r.queries.InvalidateEmailVerifications(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal menginvalidasi email verifications: %w", err)
	}
	return nil
}
