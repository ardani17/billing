package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// --- Mock repositories for area usecase tests ---

type mockAreaRepo struct {
	areas          map[string]*domain.Area
	customerCounts map[string]int
}

func newMockAreaRepo() *mockAreaRepo {
	return &mockAreaRepo{
		areas:          make(map[string]*domain.Area),
		customerCounts: make(map[string]int),
	}
}

func (m *mockAreaRepo) Create(_ context.Context, area *domain.Area) (*domain.Area, error) {
	if area.ID == "" {
		area.ID = fmt.Sprintf("area-%d", len(m.areas)+1)
	}
	copy := *area
	m.areas[copy.ID] = &copy
	return &copy, nil
}

func (m *mockAreaRepo) GetByID(_ context.Context, id string) (*domain.Area, error) {
	a, ok := m.areas[id]
	if !ok {
		return nil, domain.ErrAreaNotFound
	}
	copy := *a
	return &copy, nil
}

func (m *mockAreaRepo) Update(_ context.Context, area *domain.Area) (*domain.Area, error) {
	if _, ok := m.areas[area.ID]; !ok {
		return nil, domain.ErrAreaNotFound
	}
	copy := *area
	m.areas[copy.ID] = &copy
	return &copy, nil
}

func (m *mockAreaRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.areas[id]; !ok {
		return domain.ErrAreaNotFound
	}
	delete(m.areas, id)
	return nil
}

func (m *mockAreaRepo) List(_ context.Context, tenantID string) ([]*domain.Area, error) {
	var result []*domain.Area
	for _, a := range m.areas {
		if a.TenantID == tenantID {
			copy := *a
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockAreaRepo) NameExists(_ context.Context, tenantID, name, excludeID string) (bool, error) {
	for _, a := range m.areas {
		if a.TenantID == tenantID && a.Name == name && a.ID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockAreaRepo) CustomerCount(_ context.Context, id string) (int, error) {
	return m.customerCounts[id], nil
}

// --- Unit Tests ---

func TestAreaUsecase_Create_Success(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	req := domain.CreateAreaRequest{
		Name:        "Area Sukamaju",
		Description: "Wilayah Sukamaju",
	}

	created, err := uc.Create(ctx, "test-tenant", req, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if created.Name != "Area Sukamaju" {
		t.Fatalf("expected name 'Area Sukamaju', got %q", created.Name)
	}
	if created.TenantID != "test-tenant" {
		t.Fatalf("expected tenant_id 'test-tenant', got %q", created.TenantID)
	}
}

func TestAreaUsecase_Create_DuplicateName(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	req := domain.CreateAreaRequest{
		Name: "Area Sukamaju",
	}

	// Create first area
	_, err := uc.Create(ctx, "test-tenant", req, actor)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	// Create second area with same name
	_, err = uc.Create(ctx, "test-tenant", req, actor)
	if err != domain.ErrAreaNameDuplicate {
		t.Fatalf("expected ErrAreaNameDuplicate, got %v", err)
	}
}

func TestAreaUsecase_Create_SameNameDifferentTenant(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	req := domain.CreateAreaRequest{
		Name: "Area Sukamaju",
	}

	// Create area for tenant A
	_, err := uc.Create(ctx, "tenant-a", req, actor)
	if err != nil {
		t.Fatalf("create for tenant A failed: %v", err)
	}

	// Create area with same name for tenant B should succeed
	_, err = uc.Create(ctx, "tenant-b", req, actor)
	if err != nil {
		t.Fatalf("create for tenant B should succeed, got: %v", err)
	}
}

func TestAreaUsecase_GetByID_NotFound(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()

	_, err := uc.GetByID(ctx, "nonexistent-id")
	if err != domain.ErrAreaNotFound {
		t.Fatalf("expected ErrAreaNotFound, got %v", err)
	}
}

func TestAreaUsecase_GetByID_Success(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	created, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area Test"}, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	fetched, err := uc.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if fetched.ID != created.ID {
		t.Fatalf("expected ID %q, got %q", created.ID, fetched.ID)
	}
}

func TestAreaUsecase_Update_Success(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	created, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area Old"}, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	updated, err := uc.Update(ctx, created.ID, domain.UpdateAreaRequest{Name: "Area New"}, actor)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	if updated.Name != "Area New" {
		t.Fatalf("expected name 'Area New', got %q", updated.Name)
	}
}

func TestAreaUsecase_Update_DuplicateName(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	// Create two areas
	_, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area A"}, actor)
	if err != nil {
		t.Fatalf("create A failed: %v", err)
	}

	createdB, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area B"}, actor)
	if err != nil {
		t.Fatalf("create B failed: %v", err)
	}

	// Try to rename B to A's name
	_, err = uc.Update(ctx, createdB.ID, domain.UpdateAreaRequest{Name: "Area A"}, actor)
	if err != domain.ErrAreaNameDuplicate {
		t.Fatalf("expected ErrAreaNameDuplicate, got %v", err)
	}
}

func TestAreaUsecase_Delete_Success(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	created, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area To Delete"}, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	err = uc.Delete(ctx, created.ID, actor)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify it's deleted
	_, err = uc.GetByID(ctx, created.ID)
	if err != domain.ErrAreaNotFound {
		t.Fatalf("expected ErrAreaNotFound after delete, got %v", err)
	}
}

func TestAreaUsecase_Delete_HasCustomers(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	created, err := uc.Create(ctx, "test-tenant", domain.CreateAreaRequest{Name: "Area With Customers"}, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Set customer count for this area
	areaRepo.customerCounts[created.ID] = 3

	err = uc.Delete(ctx, created.ID, actor)
	if err == nil {
		t.Fatal("expected error when deleting area with customers")
	}

	// Should contain ErrAreaHasCustomers
	if !contains(err.Error(), domain.ErrAreaHasCustomers.Error()) {
		t.Fatalf("expected error to contain %q, got %q", domain.ErrAreaHasCustomers.Error(), err.Error())
	}
}

func TestAreaUsecase_Delete_NotFound(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	err := uc.Delete(ctx, "nonexistent-id", actor)
	if err != domain.ErrAreaNotFound {
		t.Fatalf("expected ErrAreaNotFound, got %v", err)
	}
}

func TestAreaUsecase_List_Success(t *testing.T) {
	areaRepo := newMockAreaRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewAreaUsecase(areaRepo, auditLogRepo, logger)

	ctx := context.Background()
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}

	// Create areas for different tenants
	_, _ = uc.Create(ctx, "tenant-a", domain.CreateAreaRequest{Name: "Area A1"}, actor)
	_, _ = uc.Create(ctx, "tenant-a", domain.CreateAreaRequest{Name: "Area A2"}, actor)
	_, _ = uc.Create(ctx, "tenant-b", domain.CreateAreaRequest{Name: "Area B1"}, actor)

	// List for tenant A
	areas, err := uc.List(ctx, "tenant-a")
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(areas) != 2 {
		t.Fatalf("expected 2 areas for tenant-a, got %d", len(areas))
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
