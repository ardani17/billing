package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type fakeImpersonationUserRepo struct {
	users map[string]*domain.User
}

func (r fakeImpersonationUserRepo) CreateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	return user, nil
}

func (r fakeImpersonationUserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (r fakeImpersonationUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r fakeImpersonationUserRepo) GetByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r fakeImpersonationUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	return nil, domain.ErrUserNotFound
}

func (r fakeImpersonationUserRepo) UpdateUser(ctx context.Context, user *domain.User) (*domain.User, error) {
	return user, nil
}

func (r fakeImpersonationUserRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	return nil
}

func (r fakeImpersonationUserRepo) UpdatePasswordHash(ctx context.Context, userID, hash string) error {
	return nil
}

func (r fakeImpersonationUserRepo) UpdateStatus(ctx context.Context, userID string, status domain.UserStatus) error {
	return nil
}

func (r fakeImpersonationUserRepo) LinkGoogleID(ctx context.Context, userID, googleID string) error {
	return nil
}

func (r fakeImpersonationUserRepo) SetEmailVerified(ctx context.Context, userID string) error {
	return nil
}

func (r fakeImpersonationUserRepo) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (r fakeImpersonationUserRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.User, error) {
	return nil, nil
}

func (r fakeImpersonationUserRepo) EmailExistsGlobal(ctx context.Context, email string) (bool, error) {
	return false, nil
}

func newImpersonationUsecase(users map[string]*domain.User) *ImpersonationUsecase {
	return NewImpersonationUsecase(ImpersonationUsecaseConfig{
		UserRepo:  fakeImpersonationUserRepo{users: users},
		JWTSecret: "test-secret-for-impersonation-usecase",
		JWTExpiry: time.Hour,
	})
}

func TestImpersonationUsecaseRejectsForbiddenTargets(t *testing.T) {
	uc := newImpersonationUsecase(map[string]*domain.User{
		"super":    {ID: "super", TenantID: "platform", Role: domain.RoleSuperAdmin},
		"operator": {ID: "operator", TenantID: "tenant-1", Role: domain.RoleOperator},
	})

	for _, userID := range []string{"super", "operator"} {
		_, err := uc.StartImpersonation(context.Background(), "owner", domain.ImpersonateRequest{
			TenantID: "tenant-1",
			UserID:   userID,
			Reason:   "Support smoke test",
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("expected forbidden for %s, got %v", userID, err)
		}
	}
}

func TestImpersonationUsecaseStartsAndStopsTenantAdmin(t *testing.T) {
	uc := newImpersonationUsecase(map[string]*domain.User{
		"owner": {ID: "owner", TenantID: "platform", Role: domain.RoleSuperAdmin},
		"admin": {ID: "admin", TenantID: "tenant-1", Role: domain.RoleTenantAdmin},
	})

	started, err := uc.StartImpersonation(context.Background(), "owner", domain.ImpersonateRequest{
		TenantID: "tenant-1",
		UserID:   "admin",
		Reason:   "Support smoke test",
	})
	if err != nil {
		t.Fatalf("start impersonation failed: %v", err)
	}
	if started.AccessToken == "" {
		t.Fatal("expected access token for impersonated tenant admin")
	}

	stopped, err := uc.StopImpersonation(context.Background(), "owner")
	if err != nil {
		t.Fatalf("stop impersonation failed: %v", err)
	}
	if stopped.AccessToken == "" {
		t.Fatal("expected access token for restored super admin")
	}
}
