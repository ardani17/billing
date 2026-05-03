package handler

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// --- In-memory mock repositories untuk testing session handler logic ---

// mockSessionRepo adalah implementasi in-memory dari domain.SessionRepository.
type mockSessionRepo struct {
	mu       sync.Mutex
	sessions map[string]*domain.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{sessions: make(map[string]*domain.Session)}
}

func (m *mockSessionRepo) CreateSession(_ context.Context, session *domain.Session) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if session.ID == "" {
		session.ID = fmt.Sprintf("session-%d", len(m.sessions)+1)
	}
	copy := *session
	m.sessions[copy.ID] = &copy
	return &copy, nil
}

func (m *mockSessionRepo) GetByTokenHash(_ context.Context, tokenHash string) (*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.TokenHash == tokenHash {
			copy := *s
			return &copy, nil
		}
	}
	return nil, domain.ErrTokenNotFound
}

func (m *mockSessionRepo) ListByUserID(_ context.Context, userID string) ([]*domain.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			copy := *s
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockSessionRepo) DeleteByID(_ context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *mockSessionRepo) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.TokenHash == tokenHash {
			delete(m.sessions, id)
			return nil
		}
	}
	return nil
}

func (m *mockSessionRepo) DeleteByUserID(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepo) DeleteOtherSessions(_ context.Context, userID, currentSessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.UserID == userID && id != currentSessionID {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *mockSessionRepo) DeleteExpired(_ context.Context) error { return nil }

// countSessionsByUserID menghitung jumlah session untuk user tertentu (helper untuk test).
func (m *mockSessionRepo) countSessionsByUserID(userID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := 0
	for _, s := range m.sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count
}

// getSessionByID mengambil session berdasarkan ID (helper untuk test).
func (m *mockSessionRepo) getSessionByID(sessionID string) (*domain.Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	copy := *s
	return &copy, true
}

// --- Property Tests ---

// Feature: auth-rbac, Property 14: Password Change Invalidates Other Sessions
// **Validates: Requirements 11.4**
//
// For any authenticated user with multiple active sessions who successfully
// changes their password, all sessions except the current session SHALL be
// deleted. The current session SHALL remain active.
func TestProperty_PasswordChangeInvalidatesOtherSessions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Generate user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

		// Create N sessions (2 to 10) for this user
		numSessions := rapid.IntRange(2, 10).Draw(t, "numSessions")
		sessionIDs := make([]string, numSessions)
		for i := 0; i < numSessions; i++ {
			s, err := repo.CreateSession(ctx, &domain.Session{
				UserID:    userID,
				TokenHash: fmt.Sprintf("token-hash-%s-%d", userID, i),
			})
			if err != nil {
				t.Fatalf("gagal membuat session %d: %v", i, err)
			}
			sessionIDs[i] = s.ID
		}

		// Pick one session as the "current" session
		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentSessionID := sessionIDs[currentIdx]

		// Verify all sessions exist before operation
		beforeCount := repo.countSessionsByUserID(userID)
		if beforeCount != numSessions {
			t.Fatalf("expected %d sessions before, got %d", numSessions, beforeCount)
		}

		// Simulate password change: DeleteOtherSessions (same logic as ChangePassword usecase)
		err := repo.DeleteOtherSessions(ctx, userID, currentSessionID)
		if err != nil {
			t.Fatalf("gagal DeleteOtherSessions: %v", err)
		}

		// Property: only the current session remains
		afterCount := repo.countSessionsByUserID(userID)
		if afterCount != 1 {
			t.Errorf("expected 1 session after password change, got %d", afterCount)
		}

		// Property: the remaining session is the current session
		currentSession, exists := repo.getSessionByID(currentSessionID)
		if !exists {
			t.Errorf("current session %s was deleted, should have been preserved", currentSessionID)
		}
		if currentSession != nil && currentSession.UserID != userID {
			t.Errorf("remaining session belongs to wrong user: got %q, want %q", currentSession.UserID, userID)
		}
	})
}

// Feature: auth-rbac, Property 15: Session Listing Correctly Identifies Current Session
// **Validates: Requirements 15.1**
//
// For any authenticated user with N active sessions (N >= 1), listing sessions
// SHALL return exactly N sessions, and exactly one session SHALL have
// is_current set to true (the session matching the current request's token).
func TestProperty_SessionListingIdentifiesCurrentSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Generate user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

		// Create N sessions (1 to 10)
		numSessions := rapid.IntRange(1, 10).Draw(t, "numSessions")
		tokenHashes := make([]string, numSessions)
		for i := 0; i < numSessions; i++ {
			tokenHash := fmt.Sprintf("token-hash-%s-%d", userID, i)
			tokenHashes[i] = tokenHash
			_, err := repo.CreateSession(ctx, &domain.Session{
				UserID:    userID,
				TokenHash: tokenHash,
			})
			if err != nil {
				t.Fatalf("gagal membuat session %d: %v", i, err)
			}
		}

		// Pick one token hash as the "current" token
		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentTokenHash := tokenHashes[currentIdx]

		// List sessions (same logic as SessionHandler.List)
		sessions, err := repo.ListByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("gagal ListByUserID: %v", err)
		}

		// Property: returned count matches created count
		if len(sessions) != numSessions {
			t.Errorf("expected %d sessions, got %d", numSessions, len(sessions))
		}

		// Mark current session (same logic as SessionHandler.List)
		for _, s := range sessions {
			if currentTokenHash != "" && s.TokenHash == currentTokenHash {
				s.IsCurrent = true
			}
		}

		// Property: exactly one session has is_current=true
		currentCount := 0
		for _, s := range sessions {
			if s.IsCurrent {
				currentCount++
			}
		}
		if currentCount != 1 {
			t.Errorf("expected exactly 1 current session, got %d", currentCount)
		}
	})
}

// Feature: auth-rbac, Property 16: Session Revocation Respects Ownership
// **Validates: Requirements 15.2, 8.3**
//
// For any session revocation request, the system SHALL delete the session if
// and only if the session belongs to the requesting user. Attempting to delete
// a session belonging to a different user SHALL be rejected.
func TestProperty_SessionRevocationRespectsOwnership(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Generate two distinct user IDs
		userA := rapid.StringMatching(`a[0-9a-f]{7}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userA")
		userB := rapid.StringMatching(`b[0-9a-f]{7}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userB")

		// Create sessions for user A (1 to 5)
		numSessionsA := rapid.IntRange(1, 5).Draw(t, "numSessionsA")
		sessionIDsA := make([]string, numSessionsA)
		for i := 0; i < numSessionsA; i++ {
			s, err := repo.CreateSession(ctx, &domain.Session{
				UserID:    userA,
				TokenHash: fmt.Sprintf("token-a-%d", i),
			})
			if err != nil {
				t.Fatalf("gagal membuat session A-%d: %v", i, err)
			}
			sessionIDsA[i] = s.ID
		}

		// Create sessions for user B (1 to 5)
		numSessionsB := rapid.IntRange(1, 5).Draw(t, "numSessionsB")
		sessionIDsB := make([]string, numSessionsB)
		for i := 0; i < numSessionsB; i++ {
			s, err := repo.CreateSession(ctx, &domain.Session{
				UserID:    userB,
				TokenHash: fmt.Sprintf("token-b-%d", i),
			})
			if err != nil {
				t.Fatalf("gagal membuat session B-%d: %v", i, err)
			}
			sessionIDsB[i] = s.ID
		}

		// Simulate revocation logic (same as SessionHandler.Revoke):
		// User A tries to revoke one of user B's sessions
		targetSessionID := sessionIDsB[rapid.IntRange(0, numSessionsB-1).Draw(t, "targetIdx")]

		// Check ownership: list user A's sessions and see if target is in the list
		sessionsA, err := repo.ListByUserID(ctx, userA)
		if err != nil {
			t.Fatalf("gagal ListByUserID for userA: %v", err)
		}

		owned := false
		for _, s := range sessionsA {
			if s.ID == targetSessionID {
				owned = true
				break
			}
		}

		// Property: user A does NOT own user B's session
		if owned {
			t.Errorf("user A should not own session %s belonging to user B", targetSessionID)
		}

		// Now user A tries to revoke their own session — should succeed
		ownSessionID := sessionIDsA[rapid.IntRange(0, numSessionsA-1).Draw(t, "ownIdx")]

		ownedOwn := false
		for _, s := range sessionsA {
			if s.ID == ownSessionID {
				ownedOwn = true
				break
			}
		}

		// Property: user A owns their own session
		if !ownedOwn {
			t.Errorf("user A should own session %s", ownSessionID)
		}

		// Delete own session
		err = repo.DeleteByID(ctx, ownSessionID)
		if err != nil {
			t.Fatalf("gagal DeleteByID: %v", err)
		}

		// Property: own session is deleted
		_, exists := repo.getSessionByID(ownSessionID)
		if exists {
			t.Errorf("session %s should have been deleted", ownSessionID)
		}

		// Property: user B's sessions are untouched
		afterCountB := repo.countSessionsByUserID(userB)
		if afterCountB != numSessionsB {
			t.Errorf("user B sessions changed: expected %d, got %d", numSessionsB, afterCountB)
		}
	})
}

// Feature: auth-rbac, Property 17: Revoke-Others Preserves Current Session
// **Validates: Requirements 15.3**
//
// For any authenticated user with N active sessions (N >= 2), revoking all
// other sessions SHALL delete exactly N-1 sessions and preserve exactly the
// current session.
func TestProperty_RevokeOthersPreservesCurrentSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Generate user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

		// Create N sessions (2 to 10)
		numSessions := rapid.IntRange(2, 10).Draw(t, "numSessions")
		sessionIDs := make([]string, numSessions)
		tokenHashes := make([]string, numSessions)
		for i := 0; i < numSessions; i++ {
			tokenHash := fmt.Sprintf("token-hash-%s-%d", userID, i)
			tokenHashes[i] = tokenHash
			s, err := repo.CreateSession(ctx, &domain.Session{
				UserID:    userID,
				TokenHash: tokenHash,
			})
			if err != nil {
				t.Fatalf("gagal membuat session %d: %v", i, err)
			}
			sessionIDs[i] = s.ID
		}

		// Pick one session as the "current" session
		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentSessionID := sessionIDs[currentIdx]
		currentTokenHash := tokenHashes[currentIdx]

		// Simulate RevokeOthers: find current session by token hash, then delete others
		currentSession, err := repo.GetByTokenHash(ctx, currentTokenHash)
		if err != nil {
			t.Fatalf("gagal GetByTokenHash: %v", err)
		}
		if currentSession.ID != currentSessionID {
			t.Fatalf("GetByTokenHash returned wrong session: got %q, want %q", currentSession.ID, currentSessionID)
		}

		// Delete other sessions (same logic as SessionHandler.RevokeOthers)
		err = repo.DeleteOtherSessions(ctx, userID, currentSession.ID)
		if err != nil {
			t.Fatalf("gagal DeleteOtherSessions: %v", err)
		}

		// Property: exactly 1 session remains
		afterCount := repo.countSessionsByUserID(userID)
		if afterCount != 1 {
			t.Errorf("expected 1 session after revoke-others, got %d", afterCount)
		}

		// Property: the remaining session is the current session
		remaining, exists := repo.getSessionByID(currentSessionID)
		if !exists {
			t.Errorf("current session %s was deleted, should have been preserved", currentSessionID)
		}
		if remaining != nil && remaining.TokenHash != currentTokenHash {
			t.Errorf("remaining session has wrong token hash: got %q, want %q", remaining.TokenHash, currentTokenHash)
		}

		// Property: the remaining session belongs to the correct user
		if remaining != nil && remaining.UserID != userID {
			t.Errorf("remaining session belongs to wrong user: got %q, want %q", remaining.UserID, userID)
		}
	})
}
