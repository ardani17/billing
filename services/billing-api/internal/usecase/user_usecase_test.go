package usecase

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"pgregory.net/rapid"
)

// --- In-memory mock repositories untuk testing ---

// mockUserRepo adalah implementasi in-memory dari domain.UserRepository.
type mockUserRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) CreateUser(_ context.Context, user *domain.User) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Generate simple ID jika kosong
	if user.ID == "" {
		user.ID = fmt.Sprintf("user-%d", len(m.users)+1)
	}
	copy := *user
	m.users[copy.ID] = &copy
	return &copy, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	copy := *u
	return &copy, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			copy := *u
			return &copy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByTenantAndEmail(_ context.Context, tenantID, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Email == email {
			copy := *u
			return &copy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) GetByGoogleID(_ context.Context, googleID string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepo) UpdateUser(_ context.Context, user *domain.User) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.users[user.ID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	// Update hanya name, phone, role — preserve tenant_id dan field lain
	existing.Name = user.Name
	existing.Phone = user.Phone
	existing.Role = user.Role
	copy := *existing
	return &copy, nil
}

func (m *mockUserRepo) UpdateLastLogin(_ context.Context, _ string) error { return nil }

func (m *mockUserRepo) UpdatePasswordHash(_ context.Context, userID, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.PasswordHash = hash
	return nil
}

func (m *mockUserRepo) UpdateStatus(_ context.Context, userID string, status domain.UserStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	u.Status = status
	return nil
}

func (m *mockUserRepo) LinkGoogleID(_ context.Context, _, _ string) error { return nil }
func (m *mockUserRepo) SetEmailVerified(_ context.Context, _ string) error { return nil }

func (m *mockUserRepo) DeleteUser(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.users, userID)
	return nil
}

func (m *mockUserRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.User
	for _, u := range m.users {
		if u.TenantID == tenantID {
			copy := *u
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockUserRepo) EmailExistsGlobal(_ context.Context, email string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

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

// mockTokenRepo adalah implementasi minimal dari domain.TokenRepository.
type mockTokenRepo struct {
	mu             sync.Mutex
	passwordResets map[string]*domain.PasswordReset
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{passwordResets: make(map[string]*domain.PasswordReset)}
}

func (m *mockTokenRepo) CreatePasswordReset(_ context.Context, pr *domain.PasswordReset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pr.ID == "" {
		pr.ID = fmt.Sprintf("pr-%d", len(m.passwordResets)+1)
	}
	copy := *pr
	m.passwordResets[copy.ID] = &copy
	return nil
}

func (m *mockTokenRepo) GetPasswordResetByHash(_ context.Context, tokenHash string) (*domain.PasswordReset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pr := range m.passwordResets {
		if pr.TokenHash == tokenHash {
			copy := *pr
			return &copy, nil
		}
	}
	return nil, domain.ErrTokenNotFound
}

func (m *mockTokenRepo) MarkPasswordResetUsed(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	pr, ok := m.passwordResets[id]
	if !ok {
		return domain.ErrTokenNotFound
	}
	pr.Used = true
	return nil
}

func (m *mockTokenRepo) InvalidatePasswordResets(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pr := range m.passwordResets {
		if pr.UserID == userID && !pr.Used {
			pr.Used = true
		}
	}
	return nil
}

func (m *mockTokenRepo) CreateEmailVerification(_ context.Context, _ *domain.EmailVerification) error {
	return nil
}
func (m *mockTokenRepo) GetEmailVerificationByHash(_ context.Context, _ string) (*domain.EmailVerification, error) {
	return nil, domain.ErrTokenNotFound
}
func (m *mockTokenRepo) MarkEmailVerificationUsed(_ context.Context, _ string) error { return nil }
func (m *mockTokenRepo) InvalidateEmailVerifications(_ context.Context, _ string) error {
	return nil
}

// --- Property Tests ---

// Feature: auth-rbac, Property 12: User Update Preserves Tenant ID Invariant
// **Validates: Requirements 10.2**
//
// For any user update operation, the user's tenant_id SHALL remain unchanged
// regardless of the request payload. Only name, phone, and role fields SHALL
// be modifiable.
func TestProperty_UserUpdatePreservesTenantID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup: buat user repo dan usecase
		userRepo := newMockUserRepo()
		sessionRepo := newMockSessionRepo()
		tokenRepo := newMockTokenRepo()

		uc := NewUserManagementUsecase(UserManagementUsecaseConfig{
			UserRepo:    userRepo,
			SessionRepo: sessionRepo,
			TokenRepo:   tokenRepo,
			BcryptCost:  4, // cost rendah untuk testing
		})

		// Generate tenant_id dan user data secara random
		originalTenantID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "tenantID")
		originalName := rapid.StringMatching(`[A-Za-z ]{3,50}`).Draw(t, "originalName")
		originalEmail := rapid.StringMatching(`[a-z]{3,10}@example\.com`).Draw(t, "email")
		originalPhone := "+62" + rapid.StringMatching(`[0-9]{9,12}`).Draw(t, "phone")
		originalRole := rapid.SampledFrom([]domain.UserRole{
			domain.RoleOperator, domain.RoleTeknisi, domain.RoleKasir, domain.RoleReseller,
		}).Draw(t, "originalRole")

		// Buat user di repo
		user, err := userRepo.CreateUser(context.Background(), &domain.User{
			TenantID:      originalTenantID,
			Name:          originalName,
			Email:         originalEmail,
			Phone:         originalPhone,
			PasswordHash:  "hashed",
			Role:          originalRole,
			EmailVerified: true,
			Status:        domain.UserStatusActive,
		})
		if err != nil {
			t.Fatalf("gagal membuat user: %v", err)
		}

		// Generate update request dengan field random
		newName := rapid.StringMatching(`[A-Za-z ]{3,50}`).Draw(t, "newName")
		newPhone := "+62" + rapid.StringMatching(`[0-9]{9,12}`).Draw(t, "newPhone")
		newRole := rapid.SampledFrom([]domain.UserRole{
			domain.RoleOperator, domain.RoleTeknisi, domain.RoleKasir, domain.RoleReseller,
		}).Draw(t, "newRole")

		updateReq := domain.UpdateUserRequest{
			Name:  newName,
			Phone: newPhone,
			Role:  newRole,
		}

		// Lakukan update
		updated, err := uc.UpdateUser(context.Background(), user.ID, updateReq)
		if err != nil {
			t.Fatalf("gagal update user: %v", err)
		}

		// Property: tenant_id HARUS tetap sama setelah update
		if updated.TenantID != originalTenantID {
			t.Errorf("tenant_id berubah setelah update: got %q, want %q", updated.TenantID, originalTenantID)
		}

		// Verifikasi field yang diupdate berubah sesuai request
		if updated.Name != newName {
			t.Errorf("name tidak terupdate: got %q, want %q", updated.Name, newName)
		}
		if updated.Phone != newPhone {
			t.Errorf("phone tidak terupdate: got %q, want %q", updated.Phone, newPhone)
		}
		if updated.Role != newRole {
			t.Errorf("role tidak terupdate: got %q, want %q", updated.Role, newRole)
		}

		// Verifikasi juga dari repo langsung bahwa tenant_id tidak berubah
		fromRepo, err := userRepo.GetByID(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("gagal mengambil user dari repo: %v", err)
		}
		if fromRepo.TenantID != originalTenantID {
			t.Errorf("tenant_id di repo berubah: got %q, want %q", fromRepo.TenantID, originalTenantID)
		}
	})
}

// Feature: auth-rbac, Property 13: User Deactivation Invalidates All Sessions
// **Validates: Requirements 10.3**
//
// For any active user with one or more active sessions, deactivating the user
// SHALL set status to 'inactive' AND delete all session records for that user,
// leaving zero active sessions.
func TestProperty_UserDeactivationInvalidatesAllSessions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Setup: buat repos dan usecase
		userRepo := newMockUserRepo()
		sessionRepo := newMockSessionRepo()
		tokenRepo := newMockTokenRepo()

		uc := NewUserManagementUsecase(UserManagementUsecaseConfig{
			UserRepo:    userRepo,
			SessionRepo: sessionRepo,
			TokenRepo:   tokenRepo,
			BcryptCost:  4,
		})

		// Buat user aktif
		tenantID := rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).Draw(t, "tenantID")
		user, err := userRepo.CreateUser(context.Background(), &domain.User{
			TenantID:      tenantID,
			Name:          "Test User",
			Email:         rapid.StringMatching(`[a-z]{5,10}@test\.com`).Draw(t, "email"),
			PasswordHash:  "hashed",
			Role:          domain.RoleOperator,
			EmailVerified: true,
			Status:        domain.UserStatusActive,
		})
		if err != nil {
			t.Fatalf("gagal membuat user: %v", err)
		}

		// Buat caller (admin) yang berbeda dari user target
		caller, err := userRepo.CreateUser(context.Background(), &domain.User{
			TenantID:      tenantID,
			Name:          "Admin User",
			Email:         "admin@test.com",
			PasswordHash:  "hashed",
			Role:          domain.RoleTenantAdmin,
			EmailVerified: true,
			Status:        domain.UserStatusActive,
		})
		if err != nil {
			t.Fatalf("gagal membuat caller: %v", err)
		}

		// Buat N session untuk user (1 sampai 10)
		numSessions := rapid.IntRange(1, 10).Draw(t, "numSessions")
		for i := 0; i < numSessions; i++ {
			_, err := sessionRepo.CreateSession(context.Background(), &domain.Session{
				UserID:    user.ID,
				TokenHash: fmt.Sprintf("token-hash-%d", i),
			})
			if err != nil {
				t.Fatalf("gagal membuat session %d: %v", i, err)
			}
		}

		// Verifikasi session ada sebelum deactivation
		beforeCount := sessionRepo.countSessionsByUserID(user.ID)
		if beforeCount != numSessions {
			t.Fatalf("expected %d sessions before deactivation, got %d", numSessions, beforeCount)
		}

		// Deactivate user
		err = uc.DeactivateUser(context.Background(), user.ID, caller.ID)
		if err != nil {
			t.Fatalf("gagal deactivate user: %v", err)
		}

		// Property 1: status harus menjadi inactive
		deactivated, err := userRepo.GetByID(context.Background(), user.ID)
		if err != nil {
			t.Fatalf("gagal mengambil user setelah deactivation: %v", err)
		}
		if deactivated.Status != domain.UserStatusInactive {
			t.Errorf("status setelah deactivation: got %q, want %q", deactivated.Status, domain.UserStatusInactive)
		}

		// Property 2: semua session harus dihapus (zero sessions)
		afterCount := sessionRepo.countSessionsByUserID(user.ID)
		if afterCount != 0 {
			t.Errorf("sessions setelah deactivation: got %d, want 0", afterCount)
		}
	})
}
