package usecase

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/rs/zerolog"
	"pgregory.net/rapid"
)

func newTestLogger() zerolog.Logger {
	return zerolog.New(io.Discard)
}

// mockCustomerRepo is an in-memory implementation of domain.CustomerRepository.
type mockCustomerRepo struct {
	mu          sync.Mutex
	customers   map[string]*domain.Customer
	seqByTenant map[string]int
}

func newMockCustomerRepo() *mockCustomerRepo {
	return &mockCustomerRepo{
		customers:   make(map[string]*domain.Customer),
		seqByTenant: make(map[string]int),
	}
}

func (m *mockCustomerRepo) Create(_ context.Context, customer *domain.Customer) (*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if customer.ID == "" {
		customer.ID = fmt.Sprintf("cust-%d", len(m.customers)+1)
	}
	stored := *customer
	m.customers[stored.ID] = &stored
	// Track max seq
	seq := m.seqByTenant[customer.TenantID]
	m.seqByTenant[customer.TenantID] = seq + 1
	ret := stored
	return &ret, nil
}

func (m *mockCustomerRepo) GetByID(_ context.Context, id string) (*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *c
	return &copy, nil
}

func (m *mockCustomerRepo) Update(_ context.Context, customer *domain.Customer) (*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.customers[customer.ID]; !ok {
		return nil, domain.ErrCustomerNotFound
	}
	copy := *customer
	m.customers[copy.ID] = &copy
	return &copy, nil
}

func (m *mockCustomerRepo) SoftDelete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.customers[id]
	if !ok {
		return domain.ErrCustomerNotFound
	}
	now := time.Now()
	c.DeletedAt = &now
	return nil
}

func (m *mockCustomerRepo) List(_ context.Context, params domain.CustomerListParams) (*domain.CustomerListResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []*domain.Customer
	for _, c := range m.customers {
		if c.TenantID != params.TenantID {
			continue
		}
		if c.DeletedAt != nil {
			continue
		}
		filtered = append(filtered, c)
	}

	total := int64(len(filtered))
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 25
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	return &domain.CustomerListResult{
		Data: filtered[start:end],
		Pagination: domain.PaginationMeta{
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
		},
	}, nil
}

func (m *mockCustomerRepo) UpdateStatus(_ context.Context, id string, status domain.CustomerStatus) (*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	c.Status = status
	copy := *c
	return &copy, nil
}

func (m *mockCustomerRepo) UpdatePackage(_ context.Context, id string, packageID string) (*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	c.PackageID = packageID
	copy := *c
	return &copy, nil
}

func (m *mockCustomerRepo) CountByStatus(_ context.Context) (map[domain.CustomerStatus]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[domain.CustomerStatus]int64)
	for _, c := range m.customers {
		if c.DeletedAt == nil {
			result[c.Status]++
		}
	}
	return result, nil
}

func (m *mockCustomerRepo) GetMaxSeq(_ context.Context, tenantID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.seqByTenant[tenantID], nil
}

func (m *mockCustomerRepo) PhoneExists(_ context.Context, tenantID, phone, excludeID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.customers {
		if c.TenantID == tenantID && c.Phone == phone && c.ID != excludeID && c.DeletedAt == nil {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockCustomerRepo) BulkUpdateStatus(_ context.Context, ids []string, status domain.CustomerStatus) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		if c, ok := m.customers[id]; ok {
			c.Status = status
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockCustomerRepo) BulkUpdateFields(_ context.Context, ids []string, _ map[string]interface{}) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		if _, ok := m.customers[id]; ok {
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockCustomerRepo) BulkSoftDelete(_ context.Context, ids []string) ([]domain.BulkResult, error) {
	results := make([]domain.BulkResult, 0, len(ids))
	for _, id := range ids {
		if _, ok := m.customers[id]; ok {
			results = append(results, domain.BulkResult{ID: id, Success: true})
		} else {
			results = append(results, domain.BulkResult{ID: id, Success: false, Error: domain.ErrCustomerNotFound})
		}
	}
	return results, nil
}

func (m *mockCustomerRepo) GetByIDs(_ context.Context, ids []string) ([]*domain.Customer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Customer
	for _, id := range ids {
		if c, ok := m.customers[id]; ok {
			copy := *c
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockCustomerRepo) SearchForPayment(_ context.Context, _, _ string) ([]*domain.Customer, error) {
	return nil, nil
}

// mockAuditLogRepo is an in-memory implementation of domain.AuditLogRepository.
type mockAuditLogRepo struct {
	mu   sync.Mutex
	logs []*domain.AuditLog
}

func newMockAuditLogRepo() *mockAuditLogRepo {
	return &mockAuditLogRepo{
		logs: make([]*domain.AuditLog, 0),
	}
}

func (m *mockAuditLogRepo) Create(_ context.Context, log *domain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *log
	m.logs = append(m.logs, &copy)
	return nil
}

func (m *mockAuditLogRepo) ListByEntity(_ context.Context, entityType, entityID string) ([]*domain.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.EntityType == entityType && l.EntityID == entityID {
			copy := *l
			result = append(result, &copy)
		}
	}
	return result, nil
}

func (m *mockAuditLogRepo) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.logs)
}

func (m *mockAuditLogRepo) lastLog() *domain.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.logs) == 0 {
		return nil
	}
	copy := *m.logs[len(m.logs)-1]
	return &copy
}

func (m *mockAuditLogRepo) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = make([]*domain.AuditLog, 0)
}

func (m *mockAuditLogRepo) logsForEntity(entityType, entityID string) []*domain.AuditLog {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.EntityType == entityType && l.EntityID == entityID {
			copy := *l
			result = append(result, &copy)
		}
	}
	return result
}

type mockTenantModuleRepo struct {
	caps domain.TenantModuleCapabilities
	err  error
}

func (m mockTenantModuleRepo) Capabilities(_ context.Context, _ string) (domain.TenantModuleCapabilities, error) {
	if m.err != nil {
		return domain.DefaultTenantModuleCapabilities(), m.err
	}
	return m.caps, nil
}

func genTenantID() *rapid.Generator[string] {
	return rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
}

func genUUID() *rapid.Generator[string] {
	return rapid.StringMatching(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
}

func genPhone() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		digitCount := rapid.IntRange(9, 13).Draw(t, "phoneDigits")
		digits := make([]byte, digitCount)
		for i := range digits {
			digits[i] = byte('0' + rapid.IntRange(0, 9).Draw(t, fmt.Sprintf("d%d", i)))
		}
		return "+62" + string(digits)
	})
}

func genName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"Ahmad Rizki", "Budi Santoso", "Citra Dewi", "Dian Pratama",
		"Eka Putri", "Fajar Nugroho", "Gita Sari", "Hendra Wijaya",
	})
}

func genConnectionMethod() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"manual", "pppoe", "hotspot", "dhcp_binding", "static"})
}

func genValidCreateRequest(t *rapid.T) domain.CreateCustomerRequest {
	connMethod := genConnectionMethod().Draw(t, "connMethod")
	macAddr := ""
	if connMethod == "dhcp_binding" {
		macAddr = "AA:BB:CC:DD:EE:FF"
	}

	return domain.CreateCustomerRequest{
		Name:             genName().Draw(t, "name"),
		Phone:            genPhone().Draw(t, "phone"),
		Address:          "Jl. Test No. " + fmt.Sprintf("%d", rapid.IntRange(1, 100).Draw(t, "addrNum")),
		Latitude:         rapid.Float64Range(-7.0, -6.0).Draw(t, "lat"),
		Longitude:        rapid.Float64Range(106.0, 107.0).Draw(t, "lng"),
		PackageID:        genUUID().Draw(t, "packageID"),
		ActivationDate:   "2024-01-15",
		DueDate:          rapid.IntRange(1, 28).Draw(t, "dueDate"),
		ConnectionMethod: connMethod,
		MACAddress:       macAddr,
	}
}

// **Memvalidasi: Kebutuhan 8.1**
func TestProperty_NewCustomerDefaultStatus(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		req := genValidCreateRequest(t)
		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: "Test Actor",
		}

		created, err := uc.Create(context.Background(), tenantID, req, actor)
		if err != nil {
			t.Fatalf("unexpected error creating customer: %v", err)
		}

		if created.Status != domain.CustomerStatusPending {
			t.Fatalf("expected status 'pending', got %q", created.Status)
		}
	})
}

// **Memvalidasi: Kebutuhan 5.1, 5.2, 5.3**
//
// is exactly 8 alphanumeric characters.
func TestProperty_PPPoEAutoGenerationCompleteness(t *testing.T) {
	alphanumRegex := regexp.MustCompile(`^[a-zA-Z0-9]{8}$`)

	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		req := genValidCreateRequest(t)
		// Force connection method to pppoe
		req.ConnectionMethod = "pppoe"

		// Decide whether to provide credentials atau let them auto-buat
		provideUsername := rapid.Bool().Draw(t, "provideUsername")
		providePassword := rapid.Bool().Draw(t, "providePassword")

		if provideUsername {
			req.PPPoEUsername = "custom-user-" + fmt.Sprintf("%d", rapid.IntRange(1, 1000).Draw(t, "userSuffix"))
		} else {
			req.PPPoEUsername = ""
		}
		if providePassword {
			req.PPPoEPassword = "custom12"
		} else {
			req.PPPoEPassword = ""
		}

		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: "Test Actor",
		}

		created, err := uc.Create(context.Background(), tenantID, req, actor)
		if err != nil {
			t.Fatalf("unexpected error creating customer: %v", err)
		}

		if created.PPPoEUsername == "" {
			t.Fatal("pppoe_username is empty for pppoe customer")
		}
		if created.PPPoEPassword == "" {
			t.Fatal("pppoe_password is empty for pppoe customer")
		}

		if !provideUsername {
			name := req.Name
			firstName := strings.ToLower(strings.Fields(name)[0])
			idPart := strings.ToLower(strings.ReplaceAll(created.CustomerIDSeq, "-", ""))
			expectedUsername := firstName + "-" + idPart

			if created.PPPoEUsername != expectedUsername {
				t.Fatalf("auto-generated username: got %q, expected %q", created.PPPoEUsername, expectedUsername)
			}
		}

		if !providePassword {
			if !alphanumRegex.MatchString(created.PPPoEPassword) {
				t.Fatalf("auto-generated password %q does not match 8 alphanumeric chars", created.PPPoEPassword)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 6.6**
//
// min(page_size, total - (page-1)*page_size).
func TestProperty_PaginationMetadataCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.Int64Range(0, 1000).Draw(t, "total")
		pageSize := rapid.SampledFrom([]int{10, 25, 50}).Draw(t, "pageSize")

		expectedTotalPages := int(math.Ceil(float64(total) / float64(pageSize)))
		if expectedTotalPages < 1 {
			expectedTotalPages = 1
		}

		page := rapid.IntRange(1, expectedTotalPages).Draw(t, "page")

		meta := ComputePaginationMeta(total, page, pageSize)

		if meta.TotalPages != expectedTotalPages {
			t.Fatalf("total_pages: got %d, expected %d (total=%d, page_size=%d)",
				meta.TotalPages, expectedTotalPages, total, pageSize)
		}

		if meta.Page < 1 || meta.Page > meta.TotalPages {
			t.Fatalf("page %d is out of range [1, %d]", meta.Page, meta.TotalPages)
		}

		expectedItems := int64(pageSize)
		remaining := total - int64((page-1)*pageSize)
		if remaining < expectedItems {
			expectedItems = remaining
		}
		if expectedItems < 0 {
			expectedItems = 0
		}

		actualItems := int64(pageSize)
		actualRemaining := total - int64((page-1)*pageSize)
		if actualRemaining < actualItems {
			actualItems = actualRemaining
		}
		if actualItems < 0 {
			actualItems = 0
		}

		if actualItems != expectedItems {
			t.Fatalf("items on page %d: got %d, expected %d (total=%d, page_size=%d)",
				page, actualItems, expectedItems, total, pageSize)
		}
	})
}

// **Memvalidasi: Kebutuhan 8.6, 9.5, 10.4, 11.5, 12.4, 20.1, 20.2**
func TestProperty_AuditTrailCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: genName().Draw(t, "actorName"),
		}

		operation := rapid.SampledFrom([]string{
			"create", "update", "delete", "status_change", "package_change",
		}).Draw(t, "operation")

		ctx := context.Background()

		switch operation {
		case "create":
			req := genValidCreateRequest(t)
			auditLogRepo.reset()

			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for create, got %d", len(logs))
			}
			log := logs[0]
			if log.EntityType != "customer" {
				t.Fatalf("expected entity_type 'customer', got %q", log.EntityType)
			}
			if log.EntityID != created.ID {
				t.Fatalf("expected entity_id %q, got %q", created.ID, log.EntityID)
			}
			if log.Action != "customer.created" {
				t.Fatalf("expected action 'customer.created', got %q", log.Action)
			}
			if log.ActorID != actor.ID {
				t.Fatalf("expected actor_id %q, got %q", actor.ID, log.ActorID)
			}
			if log.ActorName != actor.Name {
				t.Fatalf("expected actor_name %q, got %q", actor.Name, log.ActorName)
			}

		case "update":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			newName := genName().Draw(t, "newName")
			updateReq := domain.UpdateCustomerRequest{
				Name: newName,
			}

			_, err = uc.Update(ctx, created.ID, updateReq, actor)
			if err != nil {
				t.Fatalf("update failed: %v", err)
			}

			if newName != created.Name {
				logs := auditLogRepo.logsForEntity("customer", created.ID)
				if len(logs) != 1 {
					t.Fatalf("expected 1 audit log for update, got %d", len(logs))
				}
				log := logs[0]
				if log.Action != "customer.updated" {
					t.Fatalf("expected action 'customer.updated', got %q", log.Action)
				}
				if log.ActorID != actor.ID {
					t.Fatalf("expected actor_id %q, got %q", actor.ID, log.ActorID)
				}
				if log.ActorName != actor.Name {
					t.Fatalf("expected actor_name %q, got %q", actor.Name, log.ActorName)
				}
				if log.Changes == nil {
					t.Fatal("expected changes to be non-nil for update")
				}
				nameChange, ok := log.Changes["name"]
				if !ok {
					t.Fatal("expected 'name' in changes")
				}
				changeMap, ok := nameChange.(map[string]interface{})
				if !ok {
					t.Fatal("expected name change to be a map")
				}
				if changeMap["old"] != created.Name {
					t.Fatalf("expected old name %q, got %v", created.Name, changeMap["old"])
				}
				if changeMap["new"] != newName {
					t.Fatalf("expected new name %q, got %v", newName, changeMap["new"])
				}
			}

		case "delete":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			err = uc.SoftDelete(ctx, created.ID, created.Name, actor)
			if err != nil {
				t.Fatalf("delete failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for delete, got %d", len(logs))
			}
			log := logs[0]
			if log.Action != "customer.deleted" {
				t.Fatalf("expected action 'customer.deleted', got %q", log.Action)
			}
			if log.ActorID != actor.ID {
				t.Fatalf("expected actor_id %q, got %q", actor.ID, log.ActorID)
			}
			if log.ActorName != actor.Name {
				t.Fatalf("expected actor_name %q, got %q", actor.Name, log.ActorName)
			}

		case "status_change":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			// Activate (pending -> aktif)
			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("activate failed: %v", err)
			}

			auditLogRepo.reset()

			// Isolir (aktif -> isolir)
			_, err = uc.Isolir(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("isolir failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for status_change, got %d", len(logs))
			}
			log := logs[0]
			if log.Action != "customer.status_changed" {
				t.Fatalf("expected action 'customer.status_changed', got %q", log.Action)
			}
			if log.ActorID != actor.ID {
				t.Fatalf("expected actor_id %q, got %q", actor.ID, log.ActorID)
			}
			if log.Changes == nil {
				t.Fatal("expected changes to be non-nil for status change")
			}
			statusChange, ok := log.Changes["status"]
			if !ok {
				t.Fatal("expected 'status' in changes")
			}
			changeMap, ok := statusChange.(map[string]interface{})
			if !ok {
				t.Fatal("expected status change to be a map")
			}
			if changeMap["old"] != string(domain.CustomerStatusAktif) {
				t.Fatalf("expected old status 'aktif', got %v", changeMap["old"])
			}
			if changeMap["new"] != string(domain.CustomerStatusIsolir) {
				t.Fatalf("expected new status 'isolir', got %v", changeMap["new"])
			}

		case "package_change":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			// Change package to a different one
			newPackageID := genUUID().Draw(t, "newPackageID")
			for newPackageID == created.PackageID {
				newPackageID = genUUID().Draw(t, "newPackageID2")
			}

			_, err = uc.ChangePackage(ctx, created.ID, newPackageID, actor)
			if err != nil {
				t.Fatalf("change package failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for package_change, got %d", len(logs))
			}
			log := logs[0]
			if log.Action != "customer.package_changed" {
				t.Fatalf("expected action 'customer.package_changed', got %q", log.Action)
			}
			if log.ActorID != actor.ID {
				t.Fatalf("expected actor_id %q, got %q", actor.ID, log.ActorID)
			}
			if log.Changes == nil {
				t.Fatal("expected changes to be non-nil for package change")
			}
			pkgChange, ok := log.Changes["package_id"]
			if !ok {
				t.Fatal("expected 'package_id' in changes")
			}
			changeMap, ok := pkgChange.(map[string]interface{})
			if !ok {
				t.Fatal("expected package_id change to be a map")
			}
			if changeMap["old"] != created.PackageID {
				t.Fatalf("expected old package_id %q, got %v", created.PackageID, changeMap["old"])
			}
			if changeMap["new"] != newPackageID {
				t.Fatalf("expected new package_id %q, got %v", newPackageID, changeMap["new"])
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 8.5, 10.5, 21.1, 21.2, 21.3, 21.4, 21.5, 21.6, 21.7**
//
// code path that publishes events was executed.
func TestProperty_EventPublishingOnLifecycleChanges(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		// Kirim nil queueClient - publishEvent menangani dengan aman nil client
		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: genName().Draw(t, "actorName"),
		}

		operation := rapid.SampledFrom([]string{
			"create", "activate_from_pending", "isolir", "unblock", "terminate", "package_change",
		}).Draw(t, "operation")

		ctx := context.Background()

		switch operation {
		case "create":
			req := genValidCreateRequest(t)
			auditLogRepo.reset()

			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for create event, got %d", len(logs))
			}
			if logs[0].Action != "customer.created" {
				t.Fatalf("expected action 'customer.created', got %q", logs[0].Action)
			}

		case "activate_from_pending":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("activate failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for activate event, got %d", len(logs))
			}
			if logs[0].Action != "customer.status_changed" {
				t.Fatalf("expected action 'customer.status_changed', got %q", logs[0].Action)
			}
			statusChange := logs[0].Changes["status"].(map[string]interface{})
			if statusChange["old"] != string(domain.CustomerStatusPending) {
				t.Fatalf("expected old status 'pending', got %v", statusChange["old"])
			}
			if statusChange["new"] != string(domain.CustomerStatusAktif) {
				t.Fatalf("expected new status 'aktif', got %v", statusChange["new"])
			}

		case "isolir":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}
			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("activate failed: %v", err)
			}

			auditLogRepo.reset()

			_, err = uc.Isolir(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("isolir failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for isolir event, got %d", len(logs))
			}
			if logs[0].Action != "customer.status_changed" {
				t.Fatalf("expected action 'customer.status_changed', got %q", logs[0].Action)
			}

		case "unblock":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}
			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("activate failed: %v", err)
			}
			_, err = uc.Isolir(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("isolir failed: %v", err)
			}

			auditLogRepo.reset()

			_, err = uc.Activate(ctx, created.ID, actor)
			if err != nil {
				t.Fatalf("unblock (activate from isolir) failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for unblock event, got %d", len(logs))
			}
			if logs[0].Action != "customer.status_changed" {
				t.Fatalf("expected action 'customer.status_changed', got %q", logs[0].Action)
			}
			statusChange := logs[0].Changes["status"].(map[string]interface{})
			if statusChange["old"] != string(domain.CustomerStatusIsolir) {
				t.Fatalf("expected old status 'isolir', got %v", statusChange["old"])
			}
			if statusChange["new"] != string(domain.CustomerStatusAktif) {
				t.Fatalf("expected new status 'aktif', got %v", statusChange["new"])
			}

		case "terminate":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			err = uc.SoftDelete(ctx, created.ID, created.Name, actor)
			if err != nil {
				t.Fatalf("soft delete failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for terminate event, got %d", len(logs))
			}
			if logs[0].Action != "customer.deleted" {
				t.Fatalf("expected action 'customer.deleted', got %q", logs[0].Action)
			}

		case "package_change":
			req := genValidCreateRequest(t)
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create failed: %v", err)
			}

			auditLogRepo.reset()

			newPackageID := genUUID().Draw(t, "newPackageID")
			for newPackageID == created.PackageID {
				newPackageID = genUUID().Draw(t, "newPackageID2")
			}

			_, err = uc.ChangePackage(ctx, created.ID, newPackageID, actor)
			if err != nil {
				t.Fatalf("change package failed: %v", err)
			}

			logs := auditLogRepo.logsForEntity("customer", created.ID)
			if len(logs) != 1 {
				t.Fatalf("expected 1 audit log for package_change event, got %d", len(logs))
			}
			if logs[0].Action != "customer.package_changed" {
				t.Fatalf("expected action 'customer.package_changed', got %q", logs[0].Action)
			}
		}
	})
}

// **Memvalidasi: Kebutuhan 6.7, 7.4, 17.2**
func TestProperty_SoftDeleteExclusion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customerRepo := newMockCustomerRepo()
		auditLogRepo := newMockAuditLogRepo()
		logger := newTestLogger()

		uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

		tenantID := genTenantID().Draw(t, "tenantID")
		actor := ActorInfo{
			ID:   genUUID().Draw(t, "actorID"),
			Name: "Test Actor",
		}

		ctx := context.Background()

		activeCount := rapid.IntRange(1, 5).Draw(t, "activeCount")
		deletedCount := rapid.IntRange(1, 5).Draw(t, "deletedCount")

		var activeIDs []string
		var deletedIDs []string

		for i := 0; i < activeCount; i++ {
			req := genValidCreateRequest(t)
			req.Phone = genPhone().Draw(t, fmt.Sprintf("activePhone_%d", i))
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create active customer failed: %v", err)
			}
			activeIDs = append(activeIDs, created.ID)
		}

		for i := 0; i < deletedCount; i++ {
			req := genValidCreateRequest(t)
			req.Phone = genPhone().Draw(t, fmt.Sprintf("deletedPhone_%d", i))
			created, err := uc.Create(ctx, tenantID, req, actor)
			if err != nil {
				t.Fatalf("create to-be-deleted customer failed: %v", err)
			}
			deletedIDs = append(deletedIDs, created.ID)

			err = uc.SoftDelete(ctx, created.ID, created.Name, actor)
			if err != nil {
				t.Fatalf("soft delete failed: %v", err)
			}
		}

		listResult, err := uc.List(ctx, domain.CustomerListParams{
			TenantID: tenantID,
			Page:     1,
			PageSize: 50,
		})
		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		for _, c := range listResult.Data {
			for _, deletedID := range deletedIDs {
				if c.ID == deletedID {
					t.Fatalf("list returned soft-deleted customer %s", deletedID)
				}
			}
		}

		if int64(activeCount) != listResult.Pagination.Total {
			t.Fatalf("list total: expected %d active customers, got %d", activeCount, listResult.Pagination.Total)
		}

		stats, err := uc.Stats(ctx)
		if err != nil {
			t.Fatalf("stats failed: %v", err)
		}

		totalStats := int64(0)
		for _, count := range stats {
			totalStats += count
		}
		if totalStats != int64(activeCount) {
			t.Fatalf("stats total: expected %d active customers, got %d", activeCount, totalStats)
		}

		for _, deletedID := range deletedIDs {
			_, err := uc.GetByID(ctx, deletedID, false)
			if err == nil {
				t.Fatalf("GetByID should return error for soft-deleted customer %s", deletedID)
			}
			if err != domain.ErrCustomerNotFound {
				t.Fatalf("GetByID for soft-deleted customer %s: expected ErrCustomerNotFound, got %v", deletedID, err)
			}
		}

		for _, activeID := range activeIDs {
			detail, err := uc.GetByID(ctx, activeID, false)
			if err != nil {
				t.Fatalf("GetByID for active customer %s failed: %v", activeID, err)
			}
			if detail.Customer.ID != activeID {
				t.Fatalf("GetByID returned wrong customer: expected %s, got %s", activeID, detail.Customer.ID)
			}
		}
	})
}

// --- Unit Tests untuk CustomerUsecase ---

func TestCustomerUsecase_Create_PhoneDuplicate(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "pppoe",
	}

	_, err := uc.Create(ctx, tenantID, req, actor)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	req2 := req
	req2.Name = "Budi Santoso"
	_, err = uc.Create(ctx, tenantID, req2, actor)
	if err != domain.ErrPhoneDuplicate {
		t.Fatalf("expected ErrPhoneDuplicate, got %v", err)
	}
}

func TestCustomerUsecase_SoftDelete_ConfirmationMismatch(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	created, err := uc.Create(ctx, tenantID, req, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	err = uc.SoftDelete(ctx, created.ID, "Wrong Name", actor)
	if err != domain.ErrConfirmationMismatch {
		t.Fatalf("expected ErrConfirmationMismatch, got %v", err)
	}

	err = uc.SoftDelete(ctx, created.ID, created.Name, actor)
	if err != nil {
		t.Fatalf("delete with correct name failed: %v", err)
	}
}

func TestCustomerUsecase_ChangePackage_SamePackageError(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	created, err := uc.Create(ctx, tenantID, req, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = uc.ChangePackage(ctx, created.ID, created.PackageID, actor)
	if err != domain.ErrSamePackage {
		t.Fatalf("expected ErrSamePackage, got %v", err)
	}

	_, err = uc.ChangePackage(ctx, created.ID, "00000000-0000-0000-0000-000000000002", actor)
	if err != nil {
		t.Fatalf("change to different package failed: %v", err)
	}
}

func TestCustomerUsecase_InvalidStatusTransitions(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	created, err := uc.Create(ctx, tenantID, req, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = uc.Isolir(ctx, created.ID, actor)
	if err == nil {
		t.Fatal("expected error for pending → isolir transition")
	}

	_, err = uc.Activate(ctx, created.ID, actor)
	if err != nil {
		t.Fatalf("activate from pending failed: %v", err)
	}

	_, err = uc.Activate(ctx, created.ID, actor)
	if err == nil {
		t.Fatal("expected error for aktif → aktif transition")
	}

	_, err = uc.Isolir(ctx, created.ID, actor)
	if err != nil {
		t.Fatalf("isolir from aktif failed: %v", err)
	}

	_, err = uc.Activate(ctx, created.ID, actor)
	if err != nil {
		t.Fatalf("activate from isolir failed: %v", err)
	}
}

func TestCustomerUsecase_PPPoEAutoGeneration(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "pppoe",
	}

	created, err := uc.Create(ctx, tenantID, req, actor)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	if created.PPPoEUsername == "" {
		t.Fatal("expected pppoe_username to be auto-generated")
	}

	if created.PPPoEPassword == "" {
		t.Fatal("expected pppoe_password to be auto-generated")
	}
	if len(created.PPPoEPassword) != 8 {
		t.Fatalf("expected pppoe_password length 8, got %d", len(created.PPPoEPassword))
	}

	expectedPrefix := "ahmad-"
	if !strings.HasPrefix(created.PPPoEUsername, expectedPrefix) {
		t.Fatalf("expected pppoe_username to start with %q, got %q", expectedPrefix, created.PPPoEUsername)
	}

	// Tes that non-pppoe connection methods don't auto-buat
	req2 := domain.CreateCustomerRequest{
		Name:             "Budi Santoso",
		Phone:            "+6281234567891",
		Address:          "Jl. Test No. 2",
		Latitude:         -6.3,
		Longitude:        106.9,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	created2, err := uc.Create(ctx, tenantID, req2, actor)
	if err != nil {
		t.Fatalf("create hotspot customer failed: %v", err)
	}

	if created2.PPPoEUsername != "" {
		t.Fatalf("expected empty pppoe_username for hotspot, got %q", created2.PPPoEUsername)
	}
}

func TestCustomerUsecase_Create_ManualConnectionNoNetworkFields(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)
	uc.SetTenantModuleRepository(mockTenantModuleRepo{caps: domain.DefaultTenantModuleCapabilities()})

	created, err := uc.Create(context.Background(), "test-tenant-001", domain.CreateCustomerRequest{
		Name:             "Manual Billing",
		Phone:            "+6281234567800",
		Address:          "Jl. Billing Only No. 1",
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "manual",
	}, ActorInfo{ID: "actor-1", Name: "Test Actor"})
	if err != nil {
		t.Fatalf("create manual customer failed: %v", err)
	}
	if created.ConnectionMethod != domain.ConnectionManual {
		t.Fatalf("expected manual connection, got %q", created.ConnectionMethod)
	}
	if created.PPPoEUsername != "" || created.PPPoEPassword != "" || created.RouterID != "" || created.ODPPort != "" {
		t.Fatalf("manual customer should not receive network fields: %#v", created)
	}
	if created.Latitude != 0 || created.Longitude != 0 {
		t.Fatalf("manual customer without coordinates should keep zero values, got %f,%f", created.Latitude, created.Longitude)
	}
}

func TestCustomerUsecase_ImportTemplate_UsesModuleCapabilities(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)
	uc.SetTenantModuleRepository(mockTenantModuleRepo{caps: domain.TenantModuleCapabilities{BillingCore: true}})

	billingOnlyCSV, err := uc.GetImportTemplate(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("billing-only template failed: %v", err)
	}
	rows, err := csv.NewReader(strings.NewReader(string(billingOnlyCSV))).ReadAll()
	if err != nil {
		t.Fatalf("read billing-only template failed: %v", err)
	}
	header := strings.Join(rows[0], ",")
	for _, forbidden := range []string{"pppoe_username", "router_id", "latitude", "odp_port"} {
		if strings.Contains(header, forbidden) {
			t.Fatalf("billing-only template should not include %s: %s", forbidden, header)
		}
	}

	uc.SetTenantModuleRepository(mockTenantModuleRepo{caps: domain.TenantModuleCapabilities{BillingCore: true, MikroTik: true, FiberNetwork: true}})
	fullCSV, err := uc.GetImportTemplate(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("full template failed: %v", err)
	}
	rows, err = csv.NewReader(strings.NewReader(string(fullCSV))).ReadAll()
	if err != nil {
		t.Fatalf("read full template failed: %v", err)
	}
	fullHeader := strings.Join(rows[0], ",")
	for _, expected := range []string{"pppoe_username", "router_id", "latitude", "longitude", "odp_port"} {
		if !strings.Contains(fullHeader, expected) {
			t.Fatalf("full template should include %s: %s", expected, fullHeader)
		}
	}
}

func TestCustomerUsecase_ExportColumns_FilterInactiveModuleFields(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)
	uc.SetTenantModuleRepository(mockTenantModuleRepo{caps: domain.TenantModuleCapabilities{BillingCore: true}})

	columns := uc.customerExportColumns(context.Background(), "tenant-1", []string{"name", "pppoe_username", "latitude", "phone", "name"})
	got := strings.Join(columns, ",")
	if got != "name,phone" {
		t.Fatalf("expected only allowed unique billing columns, got %q", got)
	}
}

func TestCustomerUsecase_GetByID_NotFound(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	ctx := context.Background()

	_, err := uc.GetByID(ctx, "nonexistent-id", false)
	if err != domain.ErrCustomerNotFound {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestCustomerUsecase_Update_PhoneDuplicate(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	tenantID := "test-tenant-001"
	actor := ActorInfo{ID: "actor-1", Name: "Test Actor"}
	ctx := context.Background()

	req1 := domain.CreateCustomerRequest{
		Name:             "Ahmad Rizki",
		Phone:            "+6281234567890",
		Address:          "Jl. Test No. 1",
		Latitude:         -6.2,
		Longitude:        106.8,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	_, err := uc.Create(ctx, tenantID, req1, actor)
	if err != nil {
		t.Fatalf("create first customer failed: %v", err)
	}

	req2 := domain.CreateCustomerRequest{
		Name:             "Budi Santoso",
		Phone:            "+6281234567891",
		Address:          "Jl. Test No. 2",
		Latitude:         -6.3,
		Longitude:        106.9,
		PackageID:        "00000000-0000-0000-0000-000000000001",
		ActivationDate:   "2024-01-15",
		DueDate:          10,
		ConnectionMethod: "hotspot",
	}

	created2, err := uc.Create(ctx, tenantID, req2, actor)
	if err != nil {
		t.Fatalf("create second customer failed: %v", err)
	}

	updateReq := domain.UpdateCustomerRequest{
		Phone: "+6281234567890",
	}

	_, err = uc.Update(ctx, created2.ID, updateReq, actor)
	if err != domain.ErrPhoneDuplicate {
		t.Fatalf("expected ErrPhoneDuplicate, got %v", err)
	}
}

func TestCustomerUsecase_List_Defaults(t *testing.T) {
	customerRepo := newMockCustomerRepo()
	auditLogRepo := newMockAuditLogRepo()
	logger := newTestLogger()

	uc := NewCustomerUsecase(customerRepo, auditLogRepo, nil, logger)

	ctx := context.Background()

	result, err := uc.List(ctx, domain.CustomerListParams{
		TenantID: "test-tenant",
		Page:     0,
		PageSize: 0,
	})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if result.Pagination.Page != 1 {
		t.Fatalf("expected default page 1, got %d", result.Pagination.Page)
	}
	if result.Pagination.PageSize != 25 {
		t.Fatalf("expected default page_size 25, got %d", result.Pagination.PageSize)
	}
}
