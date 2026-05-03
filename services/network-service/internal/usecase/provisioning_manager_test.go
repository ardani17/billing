// Package usecase — Unit tests untuk ProvisionONT: happy path, validation errors, CLI failure.
package usecase

import (
	"context"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Mock implementations untuk testing Provisioning Manager
// =============================================================================

// mockONTRepo adalah mock implementasi domain.ONTRepository.
type mockONTRepo struct {
	onts             map[string]*domain.ONT
	snExists         bool
	posExists        bool
	createErr        error
	getErr           error
	updateErr        error
	customerONT      *domain.ONT
	listResult       *domain.ONTListResult
	listByStatusResult []*domain.ONT
}

func newMockONTRepo() *mockONTRepo {
	return &mockONTRepo{onts: make(map[string]*domain.ONT)}
}

func (r *mockONTRepo) Create(_ context.Context, ont *domain.ONT) (*domain.ONT, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.onts[ont.ID] = ont
	return ont, nil
}

func (r *mockONTRepo) GetByID(_ context.Context, id string) (*domain.ONT, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	ont, ok := r.onts[id]
	if !ok {
		return nil, domain.ErrONTNotFound
	}
	return ont, nil
}

func (r *mockONTRepo) GetBySerialNumber(_ context.Context, _, _ string) (*domain.ONT, error) {
	return nil, domain.ErrONTNotFound
}

func (r *mockONTRepo) Update(_ context.Context, ont *domain.ONT) (*domain.ONT, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	r.onts[ont.ID] = ont
	return ont, nil
}

func (r *mockONTRepo) SoftDelete(_ context.Context, id string) error {
	delete(r.onts, id)
	return nil
}

func (r *mockONTRepo) List(_ context.Context, _ domain.ONTListParams) (*domain.ONTListResult, error) {
	if r.listResult != nil {
		return r.listResult, nil
	}
	return &domain.ONTListResult{Data: []*domain.ONTResponse{}, Total: 0, Page: 1, PageSize: 20}, nil
}

func (r *mockONTRepo) ListByOLTAndStatus(_ context.Context, _, _ string) ([]*domain.ONT, error) {
	return r.listByStatusResult, nil
}

func (r *mockONTRepo) GetByCustomerID(_ context.Context, _ string) (*domain.ONT, error) {
	if r.customerONT != nil {
		return r.customerONT, nil
	}
	return nil, domain.ErrONTNotFound
}

func (r *mockONTRepo) SerialNumberExists(_ context.Context, _, _, _ string) (bool, error) {
	return r.snExists, nil
}

func (r *mockONTRepo) PositionExists(_ context.Context, _ string, _, _ int, _ string) (bool, error) {
	return r.posExists, nil
}

func (r *mockONTRepo) UpdateStatus(_ context.Context, id, status, state string) error {
	if ont, ok := r.onts[id]; ok {
		ont.Status = domain.ONTStatus(status)
		ont.ProvisioningState = domain.ProvisioningState(state)
	}
	return nil
}

func (r *mockONTRepo) UpdatePortMigration(_ context.Context, _ string, _, _ int) error {
	return nil
}

func (r *mockONTRepo) DeleteUnregisteredByOLT(_ context.Context, _ string, _ []string) (int64, error) {
	return 0, nil
}

// mockVLANRepo adalah mock implementasi domain.VLANRepository.
type mockVLANRepo struct {
	vlans map[string]*domain.VLAN
}

func newMockVLANRepo() *mockVLANRepo {
	return &mockVLANRepo{vlans: make(map[string]*domain.VLAN)}
}

func (r *mockVLANRepo) Create(_ context.Context, v *domain.VLAN) (*domain.VLAN, error) {
	r.vlans[v.ID] = v
	return v, nil
}

func (r *mockVLANRepo) GetByID(_ context.Context, id string) (*domain.VLAN, error) {
	v, ok := r.vlans[id]
	if !ok {
		return nil, domain.ErrVLANNotFound
	}
	return v, nil
}

func (r *mockVLANRepo) Update(_ context.Context, v *domain.VLAN) (*domain.VLAN, error) {
	r.vlans[v.ID] = v
	return v, nil
}

func (r *mockVLANRepo) SoftDelete(_ context.Context, id string) error {
	delete(r.vlans, id)
	return nil
}

func (r *mockVLANRepo) List(_ context.Context, _ string, _ domain.VLANListParams) (*domain.VLANListResult, error) {
	return &domain.VLANListResult{}, nil
}

func (r *mockVLANRepo) GetByOLTAndVLANID(_ context.Context, _ string, _ int) (*domain.VLAN, error) {
	return nil, domain.ErrVLANNotFound
}

func (r *mockVLANRepo) GetDefaultVLAN(_ context.Context, _ string) (*domain.VLAN, error) {
	for _, v := range r.vlans {
		return v, nil
	}
	return nil, domain.ErrVLANNotFound
}

func (r *mockVLANRepo) VLANIDExists(_ context.Context, _ string, _ int, _ string) (bool, error) {
	return false, nil
}

func (r *mockVLANRepo) CountActiveONTs(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// mockServiceProfileRepo adalah mock implementasi domain.ServiceProfileRepository.
type mockServiceProfileRepo struct {
	profiles map[string]*domain.ServiceProfile
}

func newMockServiceProfileRepo() *mockServiceProfileRepo {
	return &mockServiceProfileRepo{profiles: make(map[string]*domain.ServiceProfile)}
}

func (r *mockServiceProfileRepo) Create(_ context.Context, p *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	r.profiles[p.ID] = p
	return p, nil
}

func (r *mockServiceProfileRepo) GetByID(_ context.Context, id string) (*domain.ServiceProfile, error) {
	p, ok := r.profiles[id]
	if !ok {
		return nil, domain.ErrServiceProfileNotFound
	}
	return p, nil
}

func (r *mockServiceProfileRepo) Update(_ context.Context, p *domain.ServiceProfile) (*domain.ServiceProfile, error) {
	r.profiles[p.ID] = p
	return p, nil
}

func (r *mockServiceProfileRepo) SoftDelete(_ context.Context, id string) error {
	delete(r.profiles, id)
	return nil
}

func (r *mockServiceProfileRepo) List(_ context.Context, _ string, _ domain.ServiceProfileListParams) (*domain.ServiceProfileListResult, error) {
	return &domain.ServiceProfileListResult{}, nil
}

func (r *mockServiceProfileRepo) GetByPackageAndOLT(_ context.Context, _, _ string) (*domain.ServiceProfile, error) {
	return nil, domain.ErrNoProfileMapping
}

func (r *mockServiceProfileRepo) ProfileExists(_ context.Context, _ string, _, _ int, _ string) (bool, error) {
	return false, nil
}

func (r *mockServiceProfileRepo) CountActiveONTs(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

// mockAuditLogRepo adalah mock implementasi domain.AuditLogRepository.
type mockAuditLogRepo struct {
	logs []*domain.ProvisioningAuditLog
}

func (r *mockAuditLogRepo) Create(_ context.Context, l *domain.ProvisioningAuditLog) (*domain.ProvisioningAuditLog, error) {
	r.logs = append(r.logs, l)
	return l, nil
}

func (r *mockAuditLogRepo) List(_ context.Context, _ domain.AuditLogListParams) (*domain.AuditLogListResult, error) {
	return &domain.AuditLogListResult{Data: r.logs, Total: int64(len(r.logs)), Page: 1, PageSize: 20}, nil
}

// mockSettingsRepo adalah mock implementasi domain.ProvisioningSettingsRepository.
type mockSettingsRepo struct {
	settings *domain.ProvisioningSettings
}

func (r *mockSettingsRepo) GetByTenantID(_ context.Context, tenantID string) (*domain.ProvisioningSettings, error) {
	if r.settings != nil {
		return r.settings, nil
	}
	return nil, domain.ErrONTNotFound
}

func (r *mockSettingsRepo) Upsert(_ context.Context, s *domain.ProvisioningSettings) (*domain.ProvisioningSettings, error) {
	r.settings = s
	return s, nil
}

// mockProvisioningAdapter adalah mock OLTAdapter dengan kontrol granular untuk provisioning tests.
type mockProvisioningAdapter struct {
	mockOLTAdapter
	addONTErr        error
	addONTResult     *domain.ProvisioningResult
	addSPErr         error
	addSPResult      *domain.ProvisioningResult
	rebootErr        error
	rebootResult     *domain.ProvisioningResult
	removeONTErr     error
	removeONTResult  *domain.ProvisioningResult
	removeSPErr      error
	removeSPResult   *domain.ProvisioningResult
}

func (a *mockProvisioningAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	if a.addONTErr != nil {
		return a.addONTResult, a.addONTErr
	}
	if a.addONTResult != nil {
		return a.addONTResult, nil
	}
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu add"}, Responses: []string{"ok"}}, nil
}

func (a *mockProvisioningAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	if a.addSPErr != nil {
		return a.addSPResult, a.addSPErr
	}
	if a.addSPResult != nil {
		return a.addSPResult, nil
	}
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"service-port add"}, Responses: []string{"ok"}}, nil
}

func (a *mockProvisioningAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	if a.rebootErr != nil {
		return a.rebootResult, a.rebootErr
	}
	if a.rebootResult != nil {
		return a.rebootResult, nil
	}
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu reset"}, Responses: []string{"ok"}}, nil
}

func (a *mockProvisioningAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	if a.removeONTErr != nil {
		return a.removeONTResult, a.removeONTErr
	}
	if a.removeONTResult != nil {
		return a.removeONTResult, nil
	}
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"onu delete"}, Responses: []string{"ok"}}, nil
}

func (a *mockProvisioningAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	if a.removeSPErr != nil {
		return a.removeSPResult, a.removeSPErr
	}
	if a.removeSPResult != nil {
		return a.removeSPResult, nil
	}
	return &domain.ProvisioningResult{Success: true, CommandsSent: []string{"service-port delete"}, Responses: []string{"ok"}}, nil
}

// =============================================================================
// Helper untuk membuat Provisioning Manager dengan mock dependencies
// =============================================================================

func newTestProvisioningManager() (*provisioningManager, *mockONTRepo, *mockVLANRepo, *mockServiceProfileRepo, *mockAuditLogRepo, *mockOLTEventPublisher, *mockProvisioningAdapter) {
	ontRepo := newMockONTRepo()
	vlanRepo := newMockVLANRepo()
	profileRepo := newMockServiceProfileRepo()
	auditRepo := &mockAuditLogRepo{}
	settingsRepo := &mockSettingsRepo{}
	oltRepo := newMockOLTRepo()
	adapter := &mockProvisioningAdapter{}
	factory := &mockOLTAdapterFactory{adapter: adapter}
	encryptor := &mockEncryptor{}
	eventPub := &mockOLTEventPublisher{}

	mgr := NewProvisioningManager(
		ontRepo, vlanRepo, profileRepo, auditRepo, settingsRepo,
		oltRepo, factory, encryptor, eventPub, nil, nil,
	).(*provisioningManager)

	// Siapkan OLT di repo
	oltRepo.olts["olt-001"] = &domain.OLT{
		ID:                     "olt-001",
		TenantID:               "tenant-001",
		Name:                   "OLT-Test",
		Host:                   "192.168.1.100",
		Brand:                  domain.BrandZTE,
		SNMPVersion:            domain.SNMPv2c,
		SNMPCommunityEncrypted: "enc:public",
		CLIProtocol:            domain.CLIProtocolSSH,
		CLIPort:                22,
		CLIUsername:             "admin",
		CLIPasswordEncrypted:   "enc:secret",
		Status:                 domain.OLTStatusOnline,
	}

	// Siapkan VLAN
	vlanRepo.vlans["vlan-001"] = &domain.VLAN{
		ID:       "vlan-001",
		TenantID: "tenant-001",
		OLTID:    "olt-001",
		VLANID:   100,
		Name:     "VLAN-Data",
		VLANType: domain.VLANTypeData,
	}

	// Siapkan service profile
	profileRepo.profiles["profile-001"] = &domain.ServiceProfile{
		ID:               "profile-001",
		TenantID:         "tenant-001",
		OLTID:            "olt-001",
		Name:             "Profile-10M",
		LineProfileID:    1,
		ServiceProfileID: 1,
	}

	return mgr, ontRepo, vlanRepo, profileRepo, auditRepo, eventPub, adapter
}

// =============================================================================
// Test Cases — ProvisionONT
// =============================================================================

// TestProvisionONT_HappyPath memverifikasi provisioning ONT berhasil end-to-end.
func TestProvisionONT_HappyPath(t *testing.T) {
	mgr, ontRepo, _, _, auditRepo, eventPub, _ := newTestProvisioningManager()
	ctx := context.Background()

	req := domain.ProvisionONTRequest{
		SerialNumber:     "ZTEG12345678",
		OLTID:            "olt-001",
		PONPortIndex:     0,
		CustomerID:       "customer-001",
		ServiceProfileID: "profile-001",
		VLANID:           "vlan-001",
		Description:      "ONT pelanggan test",
	}

	resp, err := mgr.ProvisionONT(ctx, "tenant-001", req)
	if err != nil {
		t.Fatalf("ProvisionONT gagal: %v", err)
	}

	if resp.SerialNumber != "ZTEG12345678" {
		t.Errorf("serial number salah: got %q", resp.SerialNumber)
	}
	if resp.Status != domain.ONTStatusProvisioned {
		t.Errorf("status salah: got %q, want provisioned", resp.Status)
	}
	if resp.ProvisioningState != domain.ProvisioningStateCompleted {
		t.Errorf("provisioning state salah: got %q, want completed", resp.ProvisioningState)
	}

	// Verifikasi ONT tersimpan di repo
	if len(ontRepo.onts) != 1 {
		t.Errorf("jumlah ONT di repo: got %d, want 1", len(ontRepo.onts))
	}

	// Verifikasi audit log dibuat
	if len(auditRepo.logs) == 0 {
		t.Error("audit log tidak dibuat")
	}

	// Verifikasi event dipublish
	if len(eventPub.provisionedEvents) != 1 {
		t.Errorf("jumlah event provisioned: got %d, want 1", len(eventPub.provisionedEvents))
	}
}

// TestProvisionONT_SerialNumberExists memverifikasi error saat SN sudah ada.
func TestProvisionONT_SerialNumberExists(t *testing.T) {
	mgr, ontRepo, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	ontRepo.snExists = true

	req := domain.ProvisionONTRequest{
		SerialNumber:     "ZTEG12345678",
		OLTID:            "olt-001",
		CustomerID:       "customer-001",
		ServiceProfileID: "profile-001",
		VLANID:           "vlan-001",
	}

	_, err := mgr.ProvisionONT(ctx, "tenant-001", req)
	if err != domain.ErrONTSerialNumberExists {
		t.Errorf("expected ErrONTSerialNumberExists, got: %v", err)
	}
}

// TestProvisionONT_CustomerHasActiveONT memverifikasi error saat customer sudah punya ONT aktif.
func TestProvisionONT_CustomerHasActiveONT(t *testing.T) {
	mgr, ontRepo, _, _, _, _, _ := newTestProvisioningManager()
	ctx := context.Background()

	ontRepo.customerONT = &domain.ONT{
		ID:       "existing-ont",
		Status:   domain.ONTStatusProvisioned,
	}

	req := domain.ProvisionONTRequest{
		SerialNumber:     "ZTEG12345678",
		OLTID:            "olt-001",
		CustomerID:       "customer-001",
		ServiceProfileID: "profile-001",
		VLANID:           "vlan-001",
	}

	_, err := mgr.ProvisionONT(ctx, "tenant-001", req)
	if err != domain.ErrCustomerHasActiveONT {
		t.Errorf("expected ErrCustomerHasActiveONT, got: %v", err)
	}
}

// TestProvisionONT_CLIFailure memverifikasi handling saat CLI command gagal.
func TestProvisionONT_CLIFailure(t *testing.T) {
	mgr, _, _, _, auditRepo, _, adapter := newTestProvisioningManager()
	ctx := context.Background()

	// Simulasikan AddONT gagal
	adapter.addONTResult = &domain.ProvisioningResult{
		Success:      false,
		CommandsSent: []string{"onu add sn ZTEG12345678"},
		Responses:    []string{"Error: command failed"},
		ErrorMessage: "CLI command gagal",
	}

	req := domain.ProvisionONTRequest{
		SerialNumber:     "ZTEG12345678",
		OLTID:            "olt-001",
		PONPortIndex:     0,
		CustomerID:       "customer-001",
		ServiceProfileID: "profile-001",
		VLANID:           "vlan-001",
	}

	_, err := mgr.ProvisionONT(ctx, "tenant-001", req)
	if err != domain.ErrProvisioningFailed {
		t.Errorf("expected ErrProvisioningFailed, got: %v", err)
	}

	// Verifikasi audit log mencatat kegagalan
	if len(auditRepo.logs) == 0 {
		t.Error("audit log harus dibuat meski provisioning gagal")
	}
	if auditRepo.logs[0].Status != "failed" {
		t.Errorf("audit log status salah: got %q, want failed", auditRepo.logs[0].Status)
	}
}
