// Package worker berisi asynq worker untuk memproses event dari Billing API.
// PPPoEEventWorker menangani enam jenis event:
// 1. customer.activated — buat PPPoE user di router
// 2. customer.isolir — disable user, disconnect, add firewall
// 3. customer.un_isolir — enable user, remove firewall
// 4. customer.suspend — disconnect, remove user, remove queue, remove firewall
// 5. customer.terminated — sama dengan suspend
// 6. package.changed — update profile assignment, reconnect
package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/usecase"
)

// Event type constants untuk PPPoE events dari Billing API.
const (
	EventCustomerActivated  = "customer.activated"
	EventCustomerIsolir     = "customer.isolir"
	EventCustomerIsolated   = "customer.isolated"
	EventCustomerUnIsolir   = "customer.un_isolir"
	EventCustomerUnblocked  = "customer.unblocked"
	EventCustomerSuspend    = "customer.suspend"
	EventCustomerTerminated = "customer.terminated"
	EventPackageChanged     = "package.changed"
	EventVoucherActivated   = "voucher.activated"
)

// maxRetries adalah jumlah maksimal retry sebelum ditandai failed_permanent.
const maxRetries = 5

// PPPoERetryDelays adalah jadwal delay retry dengan exponential backoff.
// Diekspor agar bisa ditest dari luar package.
var PPPoERetryDelays = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	120 * time.Second,
	300 * time.Second,
	600 * time.Second,
}

// PPPoEEventWorker memproses event PPPoE dari Billing API via asynq.
// Mendaftarkan handler untuk semua event lifecycle pelanggan.
type PPPoEEventWorker struct {
	manager        usecase.PPPoEManager
	hotspotManager usecase.HotspotManager
	routerRepo     domain.RouterRepository
	moduleChecker  interface {
		IsEnabled(ctx context.Context, tenantID, moduleCode string) (bool, error)
	}
	eventPub domain.PPPoEEventPublisher
	logger   zerolog.Logger
}

// NewPPPoEEventWorker membuat instance baru PPPoEEventWorker.
func NewPPPoEEventWorker(
	manager usecase.PPPoEManager,
	eventPub domain.PPPoEEventPublisher,
	logger zerolog.Logger,
) *PPPoEEventWorker {
	return &PPPoEEventWorker{
		manager:  manager,
		eventPub: eventPub,
		logger:   logger,
	}
}

// SetHotspotDependencies menambahkan handler event voucher Hotspot tanpa mengubah constructor lama.
func (w *PPPoEEventWorker) SetHotspotDependencies(manager usecase.HotspotManager, routerRepo domain.RouterRepository) {
	w.hotspotManager = manager
	w.routerRepo = routerRepo
}

func (w *PPPoEEventWorker) SetModuleChecker(checker interface {
	IsEnabled(ctx context.Context, tenantID, moduleCode string) (bool, error)
}) {
	w.moduleChecker = checker
}

// RegisterHandlers mendaftarkan semua handler task ke asynq ServeMux.
func (w *PPPoEEventWorker) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(EventCustomerActivated, w.handleCustomerActivated)
	mux.HandleFunc(EventCustomerIsolir, w.handleIsolir)
	mux.HandleFunc(EventCustomerIsolated, w.handleIsolir)
	mux.HandleFunc(EventCustomerUnIsolir, w.handleUnIsolir)
	mux.HandleFunc(EventCustomerUnblocked, w.handleUnIsolir)
	mux.HandleFunc(EventCustomerSuspend, w.handleSuspend)
	mux.HandleFunc(EventCustomerTerminated, w.handleSuspend)
	mux.HandleFunc(EventPackageChanged, w.handlePackageChanged)
	if w.hotspotManager != nil {
		mux.HandleFunc(EventVoucherActivated, w.handleVoucherActivated)
	}
}

// PPPoERetryDelay menghitung delay retry berdasarkan nomor percobaan.
// Digunakan sebagai asynq.RetryDelayFunc.
func PPPoERetryDelay(n int, err error, task *asynq.Task) time.Duration {
	if n < len(PPPoERetryDelays) {
		return PPPoERetryDelays[n]
	}
	return PPPoERetryDelays[len(PPPoERetryDelays)-1]
}
