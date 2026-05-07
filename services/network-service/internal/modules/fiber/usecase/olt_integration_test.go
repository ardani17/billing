// olt_integration_test.go - integration test untuk OLT Management Layer.
// Menguji interaksi antar komponen: OLTManager, HealthChecker, AlarmManager, SyncEngine.
package usecase

import (
	"context"
	"sync"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type integOLTRepo struct {
	mu           sync.Mutex
	olts         map[string]*domain.OLT
	nameExists   bool
	statusCounts map[domain.OLTStatus]int64
	nextID       int
}

func newIntegOLTRepo() *integOLTRepo {
	return &integOLTRepo{olts: make(map[string]*domain.OLT), statusCounts: make(map[domain.OLTStatus]int64)}
}

func (r *integOLTRepo) Create(_ context.Context, olt *domain.OLT) (*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	olt.ID = "olt-integ-" + string(rune('0'+r.nextID))
	olt.CreatedAt = time.Now()
	olt.UpdatedAt = time.Now()
	r.olts[olt.ID] = olt
	return olt, nil
}

func (r *integOLTRepo) GetByID(_ context.Context, id string) (*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.olts[id]; ok {
		return o, nil
	}
	return nil, domain.ErrOLTNotFound
}

func (r *integOLTRepo) Update(_ context.Context, olt *domain.OLT) (*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	olt.UpdatedAt = time.Now()
	r.olts[olt.ID] = olt
	return olt, nil
}

func (r *integOLTRepo) SoftDelete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.olts, id)
	return nil
}

func (r *integOLTRepo) List(_ context.Context, _ domain.OLTListParams) (*domain.OLTListResult, error) {
	return &domain.OLTListResult{Data: []*domain.OLTResponse{}, Total: 0, Page: 1, PageSize: 20}, nil
}

func (r *integOLTRepo) CountByStatus(_ context.Context) (map[domain.OLTStatus]int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	counts := make(map[domain.OLTStatus]int64)
	for _, o := range r.olts {
		counts[o.Status]++
	}
	return counts, nil
}

func (r *integOLTRepo) GetActiveOLTs(_ context.Context) ([]*domain.OLT, error) { return nil, nil }

func (r *integOLTRepo) GetOnlineOLTs(_ context.Context) ([]*domain.OLT, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []*domain.OLT
	for _, o := range r.olts {
		if o.Status == domain.OLTStatusOnline {
			res = append(res, o)
		}
	}
	return res, nil
}

func (r *integOLTRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return r.nameExists, nil
}

func (r *integOLTRepo) UpdateHealthCheck(_ context.Context, id string, p domain.OLTHealthCheckUpdate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	olt, ok := r.olts[id]
	if !ok {
		return domain.ErrOLTNotFound
	}
	olt.FailureCount = p.FailureCount
	if p.LastCheckedAt != nil {
		olt.LastCheckedAt = p.LastCheckedAt
	}
	if p.LastOnlineAt != nil {
		olt.LastOnlineAt = p.LastOnlineAt
	}
	if p.Status != nil {
		olt.Status = *p.Status
	}
	return nil
}

func (r *integOLTRepo) UpdateONTCounts(_ context.Context, id string, total int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if o, ok := r.olts[id]; ok {
		o.TotalONTCount = total
	}
	return nil
}

// oltIntegEventPub merekam semua event yang dipublikasikan lintas komponen.
type oltIntegEventPub struct {
	mu      sync.Mutex
	offline []domain.OLTDeviceOfflinePayload
	online  []domain.OLTDeviceOnlinePayload
	alarms  []domain.OLTAlarmPayload
}

func (p *oltIntegEventPub) PublishDeviceOffline(_ context.Context, pl domain.OLTDeviceOfflinePayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.offline = append(p.offline, pl)
	return nil
}

func (p *oltIntegEventPub) PublishDeviceOnline(_ context.Context, pl domain.OLTDeviceOnlinePayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.online = append(p.online, pl)
	return nil
}

func (p *oltIntegEventPub) PublishAlarm(_ context.Context, pl domain.OLTAlarmPayload) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.alarms = append(p.alarms, pl)
	return nil
}

// --- Provisioning event stubs (diperlukan oleh interface OLTEventPublisher) ---
func (p *oltIntegEventPub) PublishONTProvisioned(_ context.Context, _ domain.ONTProvisionedPayload) error {
	return nil
}
func (p *oltIntegEventPub) PublishONTDecommissioned(_ context.Context, _ domain.ONTDecommissionedPayload) error {
	return nil
}
func (p *oltIntegEventPub) PublishONTAutoProvisioned(_ context.Context, _ domain.ONTAutoProvisionedPayload) error {
	return nil
}
func (p *oltIntegEventPub) PublishONTAutoProvisionFailed(_ context.Context, _ domain.ONTAutoProvisionFailedPayload) error {
	return nil
}
func (p *oltIntegEventPub) PublishONTPortMigrated(_ context.Context, _ domain.ONTPortMigratedPayload) error {
	return nil
}

// integAlarmRepo merekam alarm yang disimpan.
type integAlarmRepo struct {
	mu      sync.Mutex
	records []*domain.OLTAlarmRecord
}

func (r *integAlarmRepo) Create(_ context.Context, a *domain.OLTAlarmRecord) (*domain.OLTAlarmRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, a)
	return a, nil
}

func (r *integAlarmRepo) List(_ context.Context, _ string, _ domain.AlarmListParams) (*domain.AlarmListResult, error) {
	return &domain.AlarmListResult{Data: []*domain.OLTAlarmRecord{}, Total: 0, Page: 1, PageSize: 20}, nil
}

func (r *integAlarmRepo) CountActive(_ context.Context, _ string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.records)), nil
}

func (r *integAlarmRepo) CountActiveByTenant(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.records)), nil
}

func (r *integAlarmRepo) ClearAlarm(_ context.Context, _ string) error { return nil }

func (r *integAlarmRepo) PurgeOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
