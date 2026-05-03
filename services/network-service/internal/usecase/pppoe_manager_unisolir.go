// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi HandleUnIsolir untuk pppoeManager.
// Sequence buka isolir: enable user → remove firewall rules → reset queue counters.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandleUnIsolir menjalankan sequence buka isolir: enable user, remove firewall, reset queue.
// Menggunakan PriorityHigh karena operasi billing-related harus cepat.
// Jika connection_method bukan "pppoe", event di-skip (return nil).
func (m *pppoeManager) HandleUnIsolir(ctx context.Context, payload domain.CustomerUnIsolirPayload) error {
	// Skip event yang bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		m.logger.Debug().
			Str("customer_id", payload.CustomerID).
			Str("connection_method", payload.ConnectionMethod).
			Msg("skip event un-isolir: connection_method bukan pppoe")
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

	log.Info().Msg("memulai sequence buka isolir PPPoE user")

	// Ambil router dan koneksi dari pool dengan PriorityHigh
	router, pool, adapter, err := m.getRouterAndPool(ctx, payload.RouterID, domain.PriorityHigh)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Step 1: Enable PPPoE user → /ppp/secret/set disabled=no
	cmd, args := cmdBuilder.SetSecret(payload.PPPoEUsername, map[string]string{"disabled": "no"})
	log.Info().Str("command", cmd).Msg("step 1: enable PPPoE user")

	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		log.Error().Err(execErr).Msg("gagal enable PPPoE user di router")
		m.publishUnIsolirResult(ctx, correlationID, payload, startTime, execErr)
		return fmt.Errorf("gagal enable PPPoE user di router %s: %w", payload.RouterID, execErr)
	}

	// Step 2: Remove firewall NAT rules by comment
	// Hapus rule isolir (firewall_nat_redirect)
	isolirComment := fmt.Sprintf("ISPBoss:isolir:%s", payload.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, isolirComment, log)

	// Hapus rule dns-redirect
	dnsComment := fmt.Sprintf("ISPBoss:dns-redirect:%s", payload.CustomerID)
	m.removeNATRuleSafe(ctx, adapter, cmdBuilder, dnsComment, log)

	// Step 3: Reset simple queue counters jika use_simple_queue enabled
	pppoeUser, userErr := m.userRepo.GetByCustomerID(ctx, payload.CustomerID)
	if userErr == nil && pppoeUser != nil && pppoeUser.UseSimpleQueue {
		resetCmd, resetArgs := cmdBuilder.ResetSimpleQueueCounters(payload.PPPoEUsername)
		log.Info().Str("command", resetCmd).Msg("step 3: reset simple queue counters")
		if _, resetErr := adapter.Execute(ctx, resetCmd, resetArgs); resetErr != nil {
			log.Warn().Err(resetErr).Msg("gagal reset simple queue counters (mungkin queue tidak ada)")
		}
	}

	// Update PPPoE user di DB: set disabled=false
	if pppoeUser != nil {
		pppoeUser.Disabled = false
		pppoeUser.UpdatedAt = time.Now()
		if _, updateErr := m.userRepo.Update(ctx, pppoeUser); updateErr != nil {
			log.Error().Err(updateErr).Msg("gagal update status disabled PPPoE user di DB")
		}
	}

	// Publish command_result event sukses
	m.publishUnIsolirResult(ctx, correlationID, payload, startTime, nil)
	log.Info().Msg("sequence buka isolir PPPoE user berhasil")
	return nil
}

// removeNATRuleSafe menghapus NAT rule berdasarkan comment.
// Jika rule tidak ditemukan, log warning dan lanjut tanpa error.
func (m *pppoeManager) removeNATRuleSafe(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	comment string,
	log zerolog.Logger,
) {
	cmd, args := cmdBuilder.RemoveNATRuleByComment(comment)
	log.Info().Str("command", cmd).Str("comment", comment).Msg("remove NAT rule by comment")

	_, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		log.Warn().Err(err).Str("comment", comment).
			Msg("gagal hapus NAT rule (mungkin sudah dihapus manual)")
	}
}

// publishUnIsolirResult mempublikasikan command_result event untuk operasi buka isolir.
func (m *pppoeManager) publishUnIsolirResult(
	ctx context.Context,
	correlationID string,
	payload domain.CustomerUnIsolirPayload,
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
		Operation:     "un_isolir",
		Status:        status,
		ErrorMessage:  errMsg,
		ExecutedAt:    executedAt,
		DurationMs:    executedAt.Sub(startTime).Milliseconds(),
	})
	if publishErr != nil {
		m.logger.Error().Err(publishErr).Str("correlation_id", correlationID).
			Msg("gagal publish command_result event un-isolir")
	}
}
