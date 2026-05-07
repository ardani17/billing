// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi HandleIsolir untuk pppoeManager.
// Sequence isolir: disable user -> disconnect session -> add firewall redirect.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// HandleIsolir menjalankan sequence isolir: disable user, disconnect session, add firewall.
// Menggunakan PriorityHigh karena operasi billing-related harus cepat.
// Jika connection_method bukan "pppoe", event di-skip (mengembalikan nil).
func (m *pppoeManager) HandleIsolir(ctx context.Context, payload domain.CustomerIsolirPayload) error {
	// Skip event yang bukan PPPoE
	if payload.ConnectionMethod != "pppoe" {
		m.logger.Debug().
			Str("customer_id", payload.CustomerID).
			Str("connection_method", payload.ConnectionMethod).
			Msg("skip event isolir: connection_method bukan pppoe")
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

	log.Info().Str("isolir_method", payload.IsolirMethod).Msg("memulai sequence isolir PPPoE user")

	// Ambil router dan koneksi dari pool dengan PriorityHigh
	router, pool, adapter, err := m.getRouterAndPool(ctx, payload.RouterID, domain.PriorityHigh)
	if err != nil {
		log.Error().Err(err).Msg("gagal mendapatkan router dan koneksi pool")
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)

	// Step 1: Disable PPPoE user -> /ppp/secret/atur disabled=yes
	cmd, args := cmdBuilder.SetSecret(payload.PPPoEUsername, map[string]string{"disabled": "yes"})
	log.Info().Str("command", cmd).Msg("step 1: disable PPPoE user")

	_, execErr := adapter.Execute(ctx, cmd, args)
	if execErr != nil {
		log.Error().Err(execErr).Msg("gagal disable PPPoE user di router")
		m.publishIsolirResult(ctx, correlationID, payload, startTime, execErr)
		return fmt.Errorf("gagal disable PPPoE user di router %s: %w", payload.RouterID, execErr)
	}

	// Step 2: Disconnect active session - cari session by username, lalu remove
	execErr = m.disconnectActiveSessionByUsername(ctx, adapter, cmdBuilder, payload.PPPoEUsername, log)
	if execErr != nil {
		log.Warn().Err(execErr).Msg("gagal disconnect active session (mungkin tidak ada session aktif)")
		// Lanjut ke step berikutnya, disconnect gagal bukan fatal
	}

	// Step 3: Add firewall redirect berdasarkan isolir_method
	// Ambil remote_address dari PPPoE user di DB untuk src-address NAT rule
	pppoeUser, userErr := m.userRepo.GetByCustomerID(ctx, payload.CustomerID)
	srcAddress := ""
	if userErr == nil && pppoeUser != nil {
		srcAddress = pppoeUser.RemoteAddress
	}

	if payload.IsolirMethod == "" {
		log.Warn().Msg("isolir_method kosong; hanya disable dan disconnect PPPoE user")
	} else {
		execErr = m.addIsolirFirewallRule(ctx, adapter, cmdBuilder, payload, srcAddress, log)
		if execErr != nil {
			log.Error().Err(execErr).Msg("gagal menambahkan firewall rule isolir")
			m.publishIsolirResult(ctx, correlationID, payload, startTime, execErr)
			return fmt.Errorf("gagal menambahkan firewall rule isolir: %w", execErr)
		}
	}

	// Perbarui PPPoE user di DB: atur disabled=true
	if pppoeUser != nil {
		pppoeUser.Disabled = true
		now := time.Now()
		pppoeUser.UpdatedAt = now
		if _, updateErr := m.userRepo.Update(ctx, pppoeUser); updateErr != nil {
			log.Error().Err(updateErr).Msg("gagal update status disabled PPPoE user di DB")
		}
	}

	// Terbitkan command_result event sukses
	m.publishIsolirResult(ctx, correlationID, payload, startTime, nil)
	log.Info().Msg("sequence isolir PPPoE user berhasil")
	return nil
}

// addIsolirFirewallRule menambahkan NAT rule berdasarkan isolir_method.
func (m *pppoeManager) addIsolirFirewallRule(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	payload domain.CustomerIsolirPayload,
	srcAddress string,
	log zerolog.Logger,
) error {
	switch payload.IsolirMethod {
	case "firewall_nat_redirect":
		params := domain.NATRuleParams{
			Chain:      "dstnat",
			SrcAddress: srcAddress,
			Protocol:   "tcp",
			DstPort:    "80",
			Action:     "dst-nat",
			ToAddress:  payload.WalledGardenIP,
			Comment:    fmt.Sprintf("ISPBoss:isolir:%s", payload.CustomerID),
		}
		cmd, args := cmdBuilder.CreateNATRule(params)
		log.Info().Str("command", cmd).Msg("step 3: add firewall NAT redirect rule")
		_, err := adapter.Execute(ctx, cmd, args)
		return err

	case "dns_redirect":
		params := domain.NATRuleParams{
			Chain:      "dstnat",
			SrcAddress: srcAddress,
			Protocol:   "udp",
			DstPort:    "53",
			Action:     "dst-nat",
			ToAddress:  payload.DNSServerIP,
			Comment:    fmt.Sprintf("ISPBoss:dns-redirect:%s", payload.CustomerID),
		}
		cmd, args := cmdBuilder.CreateNATRule(params)
		log.Info().Str("command", cmd).Msg("step 3: add DNS redirect rule")
		_, err := adapter.Execute(ctx, cmd, args)
		return err

	default:
		return domain.ErrInvalidIsolirMethod
	}
}

// publishIsolirResult mempublikasikan command_result event untuk operasi isolir.
func (m *pppoeManager) publishIsolirResult(
	ctx context.Context,
	correlationID string,
	payload domain.CustomerIsolirPayload,
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
		Operation:     "isolir",
		Status:        status,
		ErrorMessage:  errMsg,
		ExecutedAt:    executedAt,
		DurationMs:    executedAt.Sub(startTime).Milliseconds(),
	})
	if publishErr != nil {
		m.logger.Error().Err(publishErr).Str("correlation_id", correlationID).
			Msg("gagal publish command_result event isolir")
	}
}
