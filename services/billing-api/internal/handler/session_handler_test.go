package handler

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

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

// **Memvalidasi: Kebutuhan 11.4**
func TestProperty_PasswordChangeInvalidatesOtherSessions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Buat user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

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

		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentSessionID := sessionIDs[currentIdx]

		beforeCount := repo.countSessionsByUserID(userID)
		if beforeCount != numSessions {
			t.Fatalf("expected %d sessions before, got %d", numSessions, beforeCount)
		}

		err := repo.DeleteOtherSessions(ctx, userID, currentSessionID)
		if err != nil {
			t.Fatalf("gagal DeleteOtherSessions: %v", err)
		}

		afterCount := repo.countSessionsByUserID(userID)
		if afterCount != 1 {
			t.Errorf("expected 1 session after password change, got %d", afterCount)
		}

		currentSession, exists := repo.getSessionByID(currentSessionID)
		if !exists {
			t.Errorf("current session %s was deleted, should have been preserved", currentSessionID)
		}
		if currentSession != nil && currentSession.UserID != userID {
			t.Errorf("remaining session belongs to wrong user: got %q, want %q", currentSession.UserID, userID)
		}
	})
}

// **Memvalidasi: Kebutuhan 15.1**
func TestProperty_SessionListingIdentifiesCurrentSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Buat user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

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

		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentTokenHash := tokenHashes[currentIdx]

		sessions, err := repo.ListByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("gagal ListByUserID: %v", err)
		}

		if len(sessions) != numSessions {
			t.Errorf("expected %d sessions, got %d", numSessions, len(sessions))
		}

		for _, s := range sessions {
			if currentTokenHash != "" && s.TokenHash == currentTokenHash {
				s.IsCurrent = true
			}
		}

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

// **Memvalidasi: Kebutuhan 15.2, 8.3**
func TestProperty_SessionRevocationRespectsOwnership(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Buat two distinct user IDs
		userA := rapid.StringMatching(`a[0-9a-f]{7}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userA")
		userB := rapid.StringMatching(`b[0-9a-f]{7}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userB")

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

		// User A tries to revoke one of user B's sessions
		targetSessionID := sessionIDsB[rapid.IntRange(0, numSessionsB-1).Draw(t, "targetIdx")]

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

		if owned {
			t.Errorf("user A should not own session %s belonging to user B", targetSessionID)
		}

		ownSessionID := sessionIDsA[rapid.IntRange(0, numSessionsA-1).Draw(t, "ownIdx")]

		ownedOwn := false
		for _, s := range sessionsA {
			if s.ID == ownSessionID {
				ownedOwn = true
				break
			}
		}

		if !ownedOwn {
			t.Errorf("user A should own session %s", ownSessionID)
		}

		err = repo.DeleteByID(ctx, ownSessionID)
		if err != nil {
			t.Fatalf("gagal DeleteByID: %v", err)
		}

		_, exists := repo.getSessionByID(ownSessionID)
		if exists {
			t.Errorf("session %s should have been deleted", ownSessionID)
		}

		afterCountB := repo.countSessionsByUserID(userB)
		if afterCountB != numSessionsB {
			t.Errorf("user B sessions changed: expected %d, got %d", numSessionsB, afterCountB)
		}
	})
}

// **Memvalidasi: Kebutuhan 15.3**
func TestProperty_RevokeOthersPreservesCurrentSession(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := newMockSessionRepo()
		ctx := context.Background()

		// Buat user ID
		userID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "userID")

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

		currentIdx := rapid.IntRange(0, numSessions-1).Draw(t, "currentIdx")
		currentSessionID := sessionIDs[currentIdx]
		currentTokenHash := tokenHashes[currentIdx]

		currentSession, err := repo.GetByTokenHash(ctx, currentTokenHash)
		if err != nil {
			t.Fatalf("gagal GetByTokenHash: %v", err)
		}
		if currentSession.ID != currentSessionID {
			t.Fatalf("GetByTokenHash returned wrong session: got %q, want %q", currentSession.ID, currentSessionID)
		}

		err = repo.DeleteOtherSessions(ctx, userID, currentSession.ID)
		if err != nil {
			t.Fatalf("gagal DeleteOtherSessions: %v", err)
		}

		afterCount := repo.countSessionsByUserID(userID)
		if afterCount != 1 {
			t.Errorf("expected 1 session after revoke-others, got %d", afterCount)
		}

		remaining, exists := repo.getSessionByID(currentSessionID)
		if !exists {
			t.Errorf("current session %s was deleted, should have been preserved", currentSessionID)
		}
		if remaining != nil && remaining.TokenHash != currentTokenHash {
			t.Errorf("remaining session has wrong token hash: got %q, want %q", remaining.TokenHash, currentTokenHash)
		}

		if remaining != nil && remaining.UserID != userID {
			t.Errorf("remaining session belongs to wrong user: got %q, want %q", remaining.UserID, userID)
		}
	})
}
