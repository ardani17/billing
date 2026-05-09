package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/jackc/pgx/v5"
)

// SessionRepo mengimplementasikan domain.SessionRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.Session.
type SessionRepo struct {
	// queries adalah sqlc-generated Queries untuk operasi session.
	queries *Queries
}

// NewSessionRepo membuat instance baru SessionRepo.
func NewSessionRepo(queries *Queries) *SessionRepo {
	return &SessionRepo{
		queries: queries,
	}
}

// --- Helper functions untuk mapping sqlc row -> domain.Session ---

// mapCreateSessionRow memetakan Session (sqlc model) ke domain.Session.
func mapCreateSessionRow(row Session) *domain.Session {
	return &domain.Session{
		ID:         uuidToString(row.ID),
		UserID:     uuidToString(row.UserID),
		TokenHash:  row.TokenHash,
		DeviceInfo: textToString(row.DeviceInfo),
		IPAddress:  textToString(row.IpAddress),
		ExpiresAt:  timestamptzToTime(row.ExpiresAt),
		CreatedAt:  timestamptzToTime(row.CreatedAt),
	}
}

// mapGetSessionByTokenHashRow memetakan Session (sqlc model) ke domain.Session.
func mapGetSessionByTokenHashRow(row Session) *domain.Session {
	return mapCreateSessionRow(row)
}

// mapListSessionsByUserIDRow memetakan Session (sqlc model) ke domain.Session.
func mapListSessionsByUserIDRow(row Session) *domain.Session {
	return mapCreateSessionRow(row)
}

// --- Implementasi domain.SessionRepository ---

// CreateSession membuat session baru dan mengembalikan session yang dibuat.
func (r *SessionRepo) CreateSession(ctx context.Context, session *domain.Session) (*domain.Session, error) {
	row, err := r.queries.CreateSession(ctx, CreateSessionParams{
		UserID:     stringToUUID(session.UserID),
		TokenHash:  session.TokenHash,
		DeviceInfo: stringToText(session.DeviceInfo),
		IpAddress:  stringToText(session.IPAddress),
		ExpiresAt:  timeToTimestamptz(session.ExpiresAt),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat session: %w", err)
	}
	return mapCreateSessionRow(row), nil
}

// GetByTokenHash mengambil session berdasarkan hash refresh token.
// Hanya mengembalikan session yang belum expired.
func (r *SessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Session, error) {
	row, err := r.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil session by token hash: %w", err)
	}
	return mapGetSessionByTokenHashRow(row), nil
}

// ListByUserID mengambil semua session aktif (belum expired) untuk user tertentu.
func (r *SessionRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	rows, err := r.queries.ListSessionsByUserID(ctx, stringToUUID(userID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar session: %w", err)
	}

	sessions := make([]*domain.Session, 0, len(rows))
	for _, row := range rows {
		sessions = append(sessions, mapListSessionsByUserIDRow(row))
	}
	return sessions, nil
}

// DeleteByID menghapus session berdasarkan ID.
func (r *SessionRepo) DeleteByID(ctx context.Context, sessionID string) error {
	err := r.queries.DeleteSessionByID(ctx, stringToUUID(sessionID))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus session by ID: %w", err)
	}
	return nil
}

// DeleteByTokenHash menghapus session berdasarkan token hash.
func (r *SessionRepo) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	err := r.queries.DeleteSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus session by token hash: %w", err)
	}
	return nil
}

// DeleteByUserID menghapus semua session untuk user tertentu.
func (r *SessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	err := r.queries.DeleteSessionsByUserID(ctx, stringToUUID(userID))
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus semua session user: %w", err)
	}
	return nil
}

// DeleteOtherSessions menghapus semua session kecuali session yang diberikan.
// Digunakan untuk fitur "logout dari semua device lain".
func (r *SessionRepo) DeleteOtherSessions(ctx context.Context, userID, currentSessionID string) error {
	err := r.queries.DeleteOtherSessions(ctx, DeleteOtherSessionsParams{
		UserID: stringToUUID(userID),
		ID:     stringToUUID(currentSessionID),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus session lain: %w", err)
	}
	return nil
}

// DeleteExpired menghapus semua session yang sudah expired.
// Digunakan untuk cleanup berkala.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	err := r.queries.DeleteExpiredSessions(ctx)
	if err != nil {
		return fmt.Errorf("repository: gagal menghapus session expired: %w", err)
	}
	return nil
}
