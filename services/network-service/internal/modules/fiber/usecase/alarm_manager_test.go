// Package usecase - unit tests untuk alarm manager.
// Menguji PollAlarms, GetAlarms, PurgeOldAlarms, dan trap parsing.
package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type amAlarmRepo struct {
	mu           sync.Mutex
	created      []*domain.OLTAlarmRecord
	listResult   *domain.AlarmListResult
	purgeCount   int64
	purgedBefore *time.Time
	clearedIDs   []string
}

func (r *amAlarmRepo) Create(_ context.Context, alarm *domain.OLTAlarmRecord) (*domain.OLTAlarmRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created = append(r.created, alarm)
	return alarm, nil
}

func (r *amAlarmRepo) List(_ context.Context, _ string, _ domain.AlarmListParams) (*domain.AlarmListResult, error) {
	if r.listResult != nil {
		return r.listResult, nil
	}
	return &domain.AlarmListResult{Data: []*domain.OLTAlarmRecord{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (r *amAlarmRepo) CountActive(_ context.Context, _ string) (int64, error) { return 0, nil }

func (r *amAlarmRepo) CountActiveByTenant(_ context.Context) (int64, error) { return 0, nil }

func (r *amAlarmRepo) ClearAlarm(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clearedIDs = append(r.clearedIDs, id)
	return nil
}

func (r *amAlarmRepo) PurgeOlderThan(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.purgedBefore = &before
	return r.purgeCount, nil
}

// amOLTRepo menyimpan OLT untuk lookup di PollAlarms.
type amOLTRepo struct {
	olts map[string]*domain.OLT
}

func (r *amOLTRepo) Create(_ context.Context, o *domain.OLT) (*domain.OLT, error) { return o, nil }
func (r *amOLTRepo) GetByID(_ context.Context, id string) (*domain.OLT, error) {
	if o, ok := r.olts[id]; ok {
		return o, nil
	}
	return nil, domain.ErrOLTNotFound
}
func (r *amOLTRepo) Update(_ context.Context, o *domain.OLT) (*domain.OLT, error) { return o, nil }
func (r *amOLTRepo) SoftDelete(_ context.Context, _ string) error                 { return nil }
func (r *amOLTRepo) List(_ context.Context, _ domain.OLTListParams) (*domain.OLTListResult, error) {
	return nil, nil
}
func (r *amOLTRepo) CountByStatus(_ context.Context) (map[domain.OLTStatus]int64, error) {
	return nil, nil
}
func (r *amOLTRepo) GetActiveOLTs(_ context.Context) ([]*domain.OLT, error) {
	result := make([]*domain.OLT, 0, len(r.olts))
	for _, olt := range r.olts {
		result = append(result, olt)
	}
	return result, nil
}
func (r *amOLTRepo) GetOnlineOLTs(_ context.Context) ([]*domain.OLT, error) { return nil, nil }
func (r *amOLTRepo) GetByHost(_ context.Context, host string) (*domain.OLT, error) {
	for _, olt := range r.olts {
		if olt.Host == host {
			return olt, nil
		}
	}
	return nil, domain.ErrOLTNotFound
}
func (r *amOLTRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) { return false, nil }
func (r *amOLTRepo) UpdateHealthCheck(_ context.Context, _ string, _ domain.OLTHealthCheckUpdate) error {
	return nil
}
func (r *amOLTRepo) UpdateONTCounts(_ context.Context, _ string, _ int) error { return nil }

// amEventPub merekam panggilan PublishAlarm.
type amEventPub struct {
	mu     sync.Mutex
	alarms []domain.OLTAlarmPayload
}

func (p *amEventPub) PublishDeviceOffline(_ context.Context, _ domain.OLTDeviceOfflinePayload) error {
	return nil
}
func (p *amEventPub) PublishDeviceOnline(_ context.Context, _ domain.OLTDeviceOnlinePayload) error {
	return nil
}
func (p *amEventPub) PublishAlarm(_ context.Context, payload domain.OLTAlarmPayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alarms = append(p.alarms, payload)
	return nil
}

// --- Provisioning event stubs (diperlukan oleh interface OLTEventPublisher) ---
func (p *amEventPub) PublishONTProvisioned(_ context.Context, _ domain.ONTProvisionedPayload) error {
	return nil
}
func (p *amEventPub) PublishONTDecommissioned(_ context.Context, _ domain.ONTDecommissionedPayload) error {
	return nil
}
func (p *amEventPub) PublishONTAutoProvisioned(_ context.Context, _ domain.ONTAutoProvisionedPayload) error {
	return nil
}
func (p *amEventPub) PublishONTAutoProvisionFailed(_ context.Context, _ domain.ONTAutoProvisionFailedPayload) error {
	return nil
}
func (p *amEventPub) PublishONTPortMigrated(_ context.Context, _ domain.ONTPortMigratedPayload) error {
	return nil
}

type amEncryptor struct{}

func (e *amEncryptor) Encrypt(plaintext string) (string, error)  { return "enc:" + plaintext, nil }
func (e *amEncryptor) Decrypt(ciphertext string) (string, error) { return ciphertext, nil }

type amAdapter struct {
	alarms    []domain.OLTAlarm
	alarmsErr error
}

func (a *amAdapter) GetSystemInfo(_ context.Context) (*domain.OLTSystemInfo, error) { return nil, nil }
func (a *amAdapter) GetPONPortStatus(_ context.Context, _ int) (*domain.PONPortStatus, error) {
	return nil, nil
}
func (a *amAdapter) GetAllPONPorts(_ context.Context) ([]domain.PONPortStatus, error) {
	return nil, nil
}
func (a *amAdapter) GetONTList(_ context.Context, _ int) ([]domain.ONTPortStatus, error) {
	return nil, nil
}
func (a *amAdapter) GetONTSignal(_ context.Context, _, _ int) (*domain.ONTSignalInfo, error) {
	return nil, nil
}
func (a *amAdapter) GetAlarms(_ context.Context) ([]domain.OLTAlarm, error) {
	if a.alarmsErr != nil {
		return nil, a.alarmsErr
	}
	return a.alarms, nil
}
func (a *amAdapter) GetSFPInfo(_ context.Context, _ int) (*domain.SFPInfo, error) { return nil, nil }
func (a *amAdapter) GetTrafficStats(_ context.Context, _ int) (*domain.PONTrafficStats, error) {
	return nil, nil
}
func (a *amAdapter) Ping(_ context.Context) error { return nil }

// --- Provisioning method stubs (diperlukan oleh interface OLTAdapter) ---
func (a *amAdapter) AddONT(_ context.Context, _ domain.AddONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *amAdapter) RemoveONT(_ context.Context, _ domain.RemoveONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *amAdapter) AddServicePort(_ context.Context, _ domain.AddServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *amAdapter) RemoveServicePort(_ context.Context, _ domain.RemoveServicePortParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *amAdapter) RebootONT(_ context.Context, _ domain.RebootONTParams) (*domain.ProvisioningResult, error) {
	return nil, nil
}
func (a *amAdapter) GetUnregisteredONTs(_ context.Context) ([]domain.UnregisteredONT, error) {
	return nil, nil
}

type amFactory struct{ adapter domain.OLTAdapter }

func (f *amFactory) CreateAdapter(_ domain.OLTBrand, _ domain.SNMPConfig, _ domain.CLIConfig) (domain.OLTAdapter, error) {
	return f.adapter, nil
}

// =============================================================================
// =============================================================================

func newTestAlarmManager(adapter *amAdapter) (*alarmManager, *amAlarmRepo, *amEventPub) {
	alarmRepo := &amAlarmRepo{purgeCount: 5}
	oltRepo := &amOLTRepo{
		olts: map[string]*domain.OLT{
			"olt-001": {
				ID: "olt-001", TenantID: "tenant-001", Name: "OLT-Test",
				Host: "192.168.1.100", SNMPVersion: domain.SNMPv2c, SNMPPort: 161,
				SNMPCommunityEncrypted: "enc:public", CLIProtocol: domain.CLIProtocolSSH,
				CLIPort: 22, CLIUsername: "admin", CLIPasswordEncrypted: "enc:password",
				Brand: domain.BrandZTE, Status: domain.OLTStatusOnline,
			},
		},
	}
	eventPub := &amEventPub{}
	am := &alarmManager{
		alarmRepo: alarmRepo, oltRepo: oltRepo,
		factory: &amFactory{adapter: adapter}, encryptor: &amEncryptor{},
		eventPub: eventPub, trapPort: 16200, stopChan: make(chan struct{}),
	}
	return am, alarmRepo, eventPub
}
