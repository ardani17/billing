// pppoe_handlers.go berisi implementasi handler functions untuk PPPoEEventWorker.
// Setiap handler: decode TaskEnvelope, validate payload, filter connection_method,
// delegate ke PPPoEManager, dan handle permanent failure setelah max retries.
package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// handleCustomerActivated memproses event customer.activated.
// Decode payload, validasi, dan delegate ke PPPoEManager.HandleCustomerActivated.
func (w *PPPoEEventWorker) handleCustomerActivated(ctx context.Context, task *asynq.Task) error {
	var payload domain.CustomerActivatedPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	// Skip jika bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		w.logger.Debug().
			Str("connection_method", payload.ConnectionMethod).
			Str("customer_id", payload.CustomerID).
			Msg("skip customer.activated: bukan pppoe")
		return nil
	}

	// Validasi field wajib
	if payload.CustomerID == "" || payload.RouterID == "" || payload.PPPoEUsername == "" {
		w.logger.Error().Msg("payload customer.activated tidak lengkap")
		return fmt.Errorf("worker: payload customer.activated tidak lengkap")
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("router_id", payload.RouterID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses customer.activated")

	if err := w.manager.HandleCustomerActivated(ctx, payload); err != nil {
		return w.handleRetryOrFail(ctx, envelope, "create", payload.CustomerID,
			payload.RouterID, payload.TenantID, err)
	}

	return nil
}

// handleVoucherActivated memproses event voucher.activated.
// Event ini membuat atau mengaktifkan ulang user Hotspot sesuai kode voucher.
func (w *PPPoEEventWorker) handleVoucherActivated(ctx context.Context, task *asynq.Task) error {
	var payload domain.VoucherActivatedPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	if payload.Code == "" {
		return fmt.Errorf("worker: payload voucher.activated tidak lengkap")
	}
	routerID, err := w.resolveHotspotRouter(ctx, payload.RouterID)
	if err != nil {
		return w.handleRetryOrFail(ctx, envelope, "hotspot_voucher_activate", payload.VoucherID, payload.RouterID, payload.TenantID, err)
	}

	profile := strings.TrimSpace(payload.HotspotProfileName)
	if profile == "" {
		profile = "default"
	}
	comment := fmt.Sprintf("voucher:%s", payload.VoucherID)
	req := domain.CreateHotspotUserRequest{
		Name:        strings.TrimSpace(payload.Code),
		Password:    strings.TrimSpace(payload.Code),
		Profile:     profile,
		LimitUptime: strings.TrimSpace(payload.LimitUptime),
		Comment:     comment,
	}

	users, listErr := w.hotspotManager.ListUsers(ctx, routerID)
	if listErr == nil {
		for _, user := range users {
			if user.Name == req.Name {
				disabled := false
				_, updateErr := w.hotspotManager.UpdateUser(ctx, routerID, user.ID, domain.UpdateHotspotUserRequest{
					Password:    &req.Password,
					Profile:     &req.Profile,
					LimitUptime: &req.LimitUptime,
					Disabled:    &disabled,
					Comment:     &req.Comment,
				})
				if updateErr != nil {
					return w.handleRetryOrFail(ctx, envelope, "hotspot_voucher_update", payload.VoucherID, routerID, payload.TenantID, updateErr)
				}
				return nil
			}
		}
	}

	if _, err := w.hotspotManager.CreateUser(ctx, routerID, req); err != nil {
		return w.handleRetryOrFail(ctx, envelope, "hotspot_voucher_create", payload.VoucherID, routerID, payload.TenantID, err)
	}
	return nil
}

func (w *PPPoEEventWorker) resolveHotspotRouter(ctx context.Context, routerID string) (string, error) {
	if strings.TrimSpace(routerID) != "" {
		return strings.TrimSpace(routerID), nil
	}
	routers, err := w.routerRepo.GetActiveRouters(ctx)
	if err != nil {
		return "", err
	}
	for _, router := range routers {
		if router.Status == domain.StatusOnline && routerHasService(router.ServiceTypes, "hotspot") {
			return router.ID, nil
		}
	}
	return "", domain.ErrRouterNotFound
}

func routerHasService(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

// handleIsolir memproses event customer.isolir.
func (w *PPPoEEventWorker) handleIsolir(ctx context.Context, task *asynq.Task) error {
	var payload domain.CustomerIsolirPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	if payload.ConnectionMethod != "pppoe" {
		w.logger.Debug().
			Str("connection_method", payload.ConnectionMethod).
			Str("customer_id", payload.CustomerID).
			Msg("skip customer.isolir: bukan pppoe")
		return nil
	}

	if payload.CustomerID == "" || payload.RouterID == "" || payload.PPPoEUsername == "" {
		w.logger.Error().Msg("payload customer.isolir tidak lengkap")
		return fmt.Errorf("worker: payload customer.isolir tidak lengkap")
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("router_id", payload.RouterID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses customer.isolir")

	if err := w.manager.HandleIsolir(ctx, payload); err != nil {
		return w.handleRetryOrFail(ctx, envelope, "isolir", payload.CustomerID,
			payload.RouterID, payload.TenantID, err)
	}

	return nil
}

// handleUnIsolir memproses event customer.un_isolir.
func (w *PPPoEEventWorker) handleUnIsolir(ctx context.Context, task *asynq.Task) error {
	var payload domain.CustomerUnIsolirPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	if payload.ConnectionMethod != "pppoe" {
		w.logger.Debug().
			Str("connection_method", payload.ConnectionMethod).
			Str("customer_id", payload.CustomerID).
			Msg("skip customer.un_isolir: bukan pppoe")
		return nil
	}

	if payload.CustomerID == "" || payload.RouterID == "" || payload.PPPoEUsername == "" {
		w.logger.Error().Msg("payload customer.un_isolir tidak lengkap")
		return fmt.Errorf("worker: payload customer.un_isolir tidak lengkap")
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("router_id", payload.RouterID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses customer.un_isolir")

	if err := w.manager.HandleUnIsolir(ctx, payload); err != nil {
		return w.handleRetryOrFail(ctx, envelope, "un_isolir", payload.CustomerID,
			payload.RouterID, payload.TenantID, err)
	}

	return nil
}

// handleSuspend memproses event customer.suspend dan customer.terminated.
// Kedua event menjalankan removal sequence yang sama.
func (w *PPPoEEventWorker) handleSuspend(ctx context.Context, task *asynq.Task) error {
	var payload domain.CustomerSuspendPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	if payload.ConnectionMethod != "pppoe" {
		w.logger.Debug().
			Str("connection_method", payload.ConnectionMethod).
			Str("customer_id", payload.CustomerID).
			Msg("skip suspend/terminated: bukan pppoe")
		return nil
	}

	if payload.CustomerID == "" || payload.RouterID == "" || payload.PPPoEUsername == "" {
		w.logger.Error().Msg("payload suspend/terminated tidak lengkap")
		return fmt.Errorf("worker: payload suspend/terminated tidak lengkap")
	}

	// Tentukan operation name berdasarkan event type
	operation := "suspend"
	if task.Type() == EventCustomerTerminated {
		operation = "terminate"
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("router_id", payload.RouterID).
		Str("operation", operation).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses suspend/terminated")

	if err := w.manager.HandleSuspend(ctx, payload); err != nil {
		return w.handleRetryOrFail(ctx, envelope, operation, payload.CustomerID,
			payload.RouterID, payload.TenantID, err)
	}

	return nil
}

// handlePackageChanged memproses event package.changed.
func (w *PPPoEEventWorker) handlePackageChanged(ctx context.Context, task *asynq.Task) error {
	var payload domain.PackageChangedPayload
	envelope, err := w.decodePayload(task, &payload)
	if err != nil {
		return err
	}
	payload.TenantID = envelope.TenantID
	ctx = tenant.SetForTest(ctx, envelope.TenantID)
	if ok, err := w.canProcessMikroTik(ctx, envelope.TenantID, task.Type()); err != nil || !ok {
		return err
	}

	if payload.ConnectionMethod != "pppoe" {
		w.logger.Debug().
			Str("connection_method", payload.ConnectionMethod).
			Str("customer_id", payload.CustomerID).
			Msg("skip package.changed: bukan pppoe")
		return nil
	}

	if payload.CustomerID == "" || payload.RouterID == "" || payload.NewPackageID == "" {
		w.logger.Error().Msg("payload package.changed tidak lengkap")
		return fmt.Errorf("worker: payload package.changed tidak lengkap")
	}

	w.logger.Info().
		Str("customer_id", payload.CustomerID).
		Str("router_id", payload.RouterID).
		Str("correlation_id", envelope.CorrelationID).
		Msg("memproses package.changed")

	if err := w.manager.HandlePackageChanged(ctx, payload); err != nil {
		return w.handleRetryOrFail(ctx, envelope, "package_change", payload.CustomerID,
			payload.RouterID, payload.TenantID, err)
	}

	return nil
}
