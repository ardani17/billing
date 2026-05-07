// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi implementasi GetActiveSessions, DisconnectSession,
// dan GetSessionCount untuk pppoeManager.
package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetActiveSessions mengambil active PPPoE sessions dari router.
// Menggunakan PriorityLow karena operasi pemantauan.
func (m *pppoeManager) GetActiveSessions(ctx context.Context, routerID string) ([]domain.PPPoESession, error) {
	log := m.logger.With().Str("router_id", routerID).Logger()

	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityLow)
	if err != nil {
		return nil, err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	cmd, args := cmdBuilder.PrintActiveSessions()

	results, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		log.Error().Err(err).Msg("gagal ambil active sessions dari router")
		return nil, fmt.Errorf("gagal ambil active sessions: %w", err)
	}

	sessions := make([]domain.PPPoESession, 0, len(results))
	for _, r := range results {
		session := domain.PPPoESession{
			ID:       r[".id"],
			Username: r["name"],
			CallerID: r["caller-id"],
			Address:  r["address"],
			Uptime:   r["uptime"],
			BytesIn:  parseInt64(r["bytes-in"]),
			BytesOut: parseInt64(r["bytes-out"]),
			Service:  r["service"],
			Encoding: r["encoding"],
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// parseInt64 mengurai string ke int64, mengembalikan 0 jika gagal.
func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return v
}

// DisconnectSession memutus satu active session di router berdasarkan session ID.
// Menggunakan PriorityHigh karena operasi disconnect bersifat kritis.
func (m *pppoeManager) DisconnectSession(ctx context.Context, routerID, sessionID string) error {
	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityHigh)
	if err != nil {
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	cmd, args := cmdBuilder.RemoveActiveSession(sessionID)

	_, err = adapter.Execute(ctx, cmd, args)
	if err != nil {
		m.logger.Error().Err(err).
			Str("router_id", routerID).
			Str("session_id", sessionID).
			Msg("gagal disconnect session di router")
		return fmt.Errorf("gagal disconnect session %s: %w", sessionID, err)
	}

	return nil
}

// DisconnectUser memutus active session milik satu PPPoE user terkelola.
// User ID dipakai untuk lookup username lebih dulu agar tidak memutus session pelanggan lain.
func (m *pppoeManager) DisconnectUser(ctx context.Context, routerID, userID string) error {
	pppoeUser, err := m.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("gagal mengambil PPPoE user: %w", err)
	}
	if pppoeUser.RouterID != routerID {
		return domain.ErrPPPoEUserNotFound
	}

	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityHigh)
	if err != nil {
		return err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	cmd, args := cmdBuilder.PrintActiveSessions()

	results, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		m.logger.Error().Err(err).
			Str("router_id", routerID).
			Str("user_id", userID).
			Str("username", pppoeUser.Username).
			Msg("gagal ambil active sessions untuk disconnect user")
		return fmt.Errorf("gagal ambil active sessions: %w", err)
	}

	for _, session := range results {
		if session["name"] != pppoeUser.Username {
			continue
		}
		sessionID := session[".id"]
		if sessionID == "" {
			continue
		}

		removeCmd, removeArgs := cmdBuilder.RemoveActiveSession(sessionID)
		if _, err := adapter.Execute(ctx, removeCmd, removeArgs); err != nil {
			m.logger.Error().Err(err).
				Str("router_id", routerID).
				Str("user_id", userID).
				Str("session_id", sessionID).
				Msg("gagal disconnect session PPPoE user")
			return fmt.Errorf("gagal disconnect session %s: %w", sessionID, err)
		}

		return nil
	}

	return domain.ErrSessionNotFound
}

// GetSessionCount mengambil jumlah active PPPoE sessions di router.
// Menggunakan PriorityLow karena operasi pemantauan.
func (m *pppoeManager) GetSessionCount(ctx context.Context, routerID string) (int, error) {
	router, pool, adapter, err := m.getRouterAndPool(ctx, routerID, domain.PriorityLow)
	if err != nil {
		return 0, err
	}
	defer pool.Put(adapter)

	cmdBuilder := m.buildCommandBuilder(router)
	cmd, args := cmdBuilder.PrintActiveSessions()

	results, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		m.logger.Error().Err(err).Str("router_id", routerID).Msg("gagal ambil session count")
		return 0, fmt.Errorf("gagal ambil session count: %w", err)
	}

	return len(results), nil
}
