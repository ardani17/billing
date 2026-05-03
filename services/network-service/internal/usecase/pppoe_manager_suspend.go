// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi HandleSuspend untuk pppoeManager.
// Sequence suspend: disconnect → remove secret → remove queue → remove firewall → soft-delete DB.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandleSuspend menjalankan sequence suspend: disconnect, remove user, remove queue, remove firewall.
// Menggunakan PriorityMedium untuk operasi suspend/terminate.
// Jika connection_method bukan "pppoe", event di-skip (return nil).
func (m *pppoeManager) HandleSuspend(ctx context.Context, payload domain.CustomerSuspendPayload) error {
	// Skip event yang bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		m.logger.Debug().
			Str("customer_id", payload.CustomerID).
			Str("connection_method", payload.ConnectionMethod).
			Msg("skip event suspend: connection_method bukan pppoe")
		return nil
	}

	startTime := time.Now()
	correlationID := uuid.New().String()

	log := m.logger.With().
		Str("customer_id", payload.CustomerID).
		Str("tenant_id", payload.TenantID).
		Str("router_id", payload.RouterID).
		Str("pppoe_username", payload.PPPoEUsername).
		Str("correlation_id", correlationID).
		Logger()

	log.Info().Msg("memulai sequence suspend PPPoE user")

	// Ambil router dan koneksi dari pool dengan PriorityMedium
	router, pool, adapter, err := m.getRouterAndPool(ctx, payload.RouterID, domain.PriorityMedium)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Step 1: Disconnect active session
	log.Info().Msg("step 1: disconnect active session")
	disconnectErr := m.disconnectActiveSessionByUsername(ctx, adapter, cmdBuilder, payload.PPPoEUsername, log)
	if disconnectErr != nil {
		log.Warn().Err(disconnectErr).Msg("gagal disconnect active session (mungkin tidak ada session aktif)")
	}

	// Step 2: Remove PPPoE secret
	cmd, args := cmdBuilder.RemoveSecret(payload.PPPoEUsername)
	log.Info().Str("command", cmd).Msg("step 2: remove PPPoE secret")

	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		log.Error().Err(execErr).Msg("gagal remove PPPoE secret di router")
		m.publishSuspendResult(ctx, correlationID, payload, startTime, execErr)
		return fmt.Errorf("gagal remove PPPoE secret di router %s: %w", payload.RouterID, execErr)
	}

	// Step 3: Remove simple queue jika ada
	removeQueueCmd, removeQueueArgs := cmdBuilder.RemoveSimpleQueue(payload.PPPoEUsername)
	log.Info().Str("command", removeQueueCmd).Msg("step 3: remove simple queue")
	if _, queueErr := adapter.Execute(ctx, removeQueueCmd, removeQueueArgs); queueErr != nil {
		log.Warn().Err(queueErr).Msg("gagal hapus simple queue (mungkin tidak ada)")
	}

	// Step 4: Remove firewall rules (isolir dan dns-redirect)
	isolirComment := fmt.Sprintf("ISPBoss:isolir:%s", payload.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, isolirComment, log)

	dnsComment := fmt.Sprintf("ISPBoss:dns-redirect:%s", payload.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, dnsComment, log)

	// Soft-delete PPPoE user dari DB
	pppoeUser, userErr := m.userRepo.GetByCustomerID(ctx, payload.CustomerID)
	if userErr == nil && pppoeUser != nil {
		if deleteErr := m.userRepo.SoftDelete(ctx, pppoeUser.ID); deleteErr != nil {
			log.Error().Err(deleteErr).Msg("gagal soft-delete PPPoE user di DB")
		}
	} else if userErr != nil {
		log.Warn().Err(userErr).Msg("PPPoE user tidak ditemukan di DB untuk soft-delete")
	}

	// Publish command_result event
	m.publishSuspendResult(ctx, correlationID, payload, startTime, nil)
	log.Info().Msg("sequence suspend PPPoE user berhasil")
	return nil
}

// publishSuspendResult mempublikasikan command_result event untuk operasi suspend.
func (m *pppoeManager) publishSuspendResult(
	ctx context.Context,
	correlationID string,
	payload domain.CustomerSuspendPayload,
	startTime time.Time,
	execErr error,
) {
	executedAt := time.Now()
	status := "success"
	var errMsg string
	if execErr != nil {
		status = "failed"
		errMsg = execErr.Error()
	}

	publishErr := m.eventPub.PublishCommandResult(ctx, domain.CommandResultPayload{
		CorrelationID: correlationID,
		CustomerID:    payload.CustomerID,
		RouterID:      payload.RouterID,
		TenantID:      payload.TenantID,
		Operation:     "suspend",
		Status:        status,
		ErrorMessage:  errMsg,
		ExecutedAt:    executedAt,
		DurationMs:    executedAt.Sub(startTime).Milliseconds(),
	})
	if publishErr != nil {
		m.logger.Error().Err(publishErr).Str("correlation_id", correlationID).
			Msg("gagal publish command_result event suspend")
	}
}
