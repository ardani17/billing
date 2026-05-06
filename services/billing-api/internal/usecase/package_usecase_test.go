package usecase

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type mockPackageRepo struct {
	mu        sync.Mutex
	packages  map[string]*domain.Package
	deleteErr error
}

func newMockPackageRepo() *mockPackageRepo {
	return &mockPackageRepo{packages: make(map[string]*domain.Package)}
}

func (m *mockPackageRepo) Create(_ context.Context, pkg *domain.Package) (*domain.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if pkg.ID == "" {
		pkg.ID = "pkg-1"
	}
	copy := *pkg
	m.packages[pkg.ID] = &copy
	return &copy, nil
}

func (m *mockPackageRepo) GetByID(_ context.Context, id string) (*domain.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pkg, ok := m.packages[id]
	if !ok {
		return nil, domain.ErrPackageNotFound
	}
	copy := *pkg
	return &copy, nil
}

func (m *mockPackageRepo) Update(_ context.Context, pkg *domain.Package) (*domain.Package, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *pkg
	m.packages[pkg.ID] = &copy
	return &copy, nil
}

func (m *mockPackageRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.packages, id)
	return nil
}

func (m *mockPackageRepo) List(_ context.Context, _ domain.PackageListParams) (*domain.PackageListResult, error) {
	return &domain.PackageListResult{}, nil
}

func (m *mockPackageRepo) UpdateIsActive(ctx context.Context, id string, isActive bool) (*domain.Package, error) {
	pkg, err := m.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	pkg.IsActive = isActive
	return m.Update(ctx, pkg)
}

func (m *mockPackageRepo) NameExists(_ context.Context, tenantID, name, excludeID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pkg := range m.packages {
		if pkg.TenantID == tenantID && pkg.Name == name && pkg.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockPackageRepo) CustomerCount(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockPackageRepo) ListNamesByPrefix(_ context.Context, _, _ string) ([]string, error) {
	return nil, nil
}

func TestPackageUsecase_Create_MonthlyPackageWithoutMikrotikProfile(t *testing.T) {
	packageRepo := newMockPackageRepo()
	auditLogRepo := newMockAuditLogRepo()
	uc := NewPackageUsecase(packageRepo, auditLogRepo, nil, newTestLogger())

	monthlyPrice := int64(125000)
	installationFee := int64(0)
	sharedUsers := 1

	created, err := uc.Create(context.Background(), "tenant-1", domain.CreatePackageRequest{
		Type:            string(domain.PackageTypeMonthly),
		Name:            "Billing Only 20M",
		DownloadMbps:    20,
		UploadMbps:      10,
		BandwidthType:   "shared",
		QuotaType:       string(domain.QuotaUnlimited),
		MonthlyPrice:    &monthlyPrice,
		InstallationFee: &installationFee,
		SharedUsers:     &sharedUsers,
	}, domain.ActorInfo{ActorID: "actor-1", ActorName: "Test Actor"})
	if err != nil {
		t.Fatalf("create monthly package failed: %v", err)
	}

	if created.Type != domain.PackageTypeMonthly {
		t.Fatalf("expected monthly package type, got %q", created.Type)
	}
	if created.MikrotikProfileName != "" {
		t.Fatalf("monthly billing-only package should not auto-generate MikroTik profile, got %q", created.MikrotikProfileName)
	}
	if created.MonthlyPrice == nil || *created.MonthlyPrice != monthlyPrice {
		t.Fatalf("expected monthly price %d, got %#v", monthlyPrice, created.MonthlyPrice)
	}
}

func TestPackageUsecase_Delete_PackageWithVouchersReturnsConflictError(t *testing.T) {
	packageRepo := newMockPackageRepo()
	packageRepo.deleteErr = domain.ErrPackageHasVouchers
	auditLogRepo := newMockAuditLogRepo()
	uc := NewPackageUsecase(packageRepo, auditLogRepo, nil, newTestLogger())

	packageRepo.packages["pkg-voucher"] = &domain.Package{
		ID:       "pkg-voucher",
		TenantID: "tenant-1",
		Type:     domain.PackageTypeVoucher,
		Name:     "Voucher 1 Hari",
	}

	err := uc.Delete(context.Background(), "pkg-voucher", "Voucher 1 Hari", domain.ActorInfo{ActorID: "actor-1"})
	if !errors.Is(err, domain.ErrPackageHasVouchers) {
		t.Fatalf("expected ErrPackageHasVouchers, got %v", err)
	}
}
