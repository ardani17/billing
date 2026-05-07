// Package usecase berisi implementasi business logic untuk network-service.
// File ini berisi helper methods yang digunakan oleh beberapa handler pppoeManager.
package usecase

import (
	"context"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// disconnectActiveSessionByUsername mencari active session berdasarkan username
// dan memutus koneksi jika ditemukan. Mengembalikan nil jika tidak ada session aktif.
func (m *pppoeManager) disconnectActiveSessionByUsername(
	ctx context.Context,
	adapter domain.RouterOSAdapter,
	cmdBuilder domain.CommandBuilder,
	username string,
	log zerolog.Logger,
) error {
	// Ambil semua active sessions
	cmd, args := cmdBuilder.PrintActiveSessions()
	sessions, err := adapter.Execute(ctx, cmd, args)
	if err != nil {
		return err
	}

	// Cari session berdasarkan username
	for _, session := range sessions {
		if session["name"] == username {
			sessionID := session[".id"]
			if sessionID == "" {
				log.Warn().Str("username", username).Msg("session ditemukan tapi tidak ada .id")
				continue
			}

			removeCmd, removeArgs := cmdBuilder.RemoveActiveSession(sessionID)
			log.Info().Str("session_id", sessionID).Msg("disconnect active session")
			_, removeErr := adapter.Execute(ctx, removeCmd, removeArgs)
			if removeErr != nil {
				return removeErr
			}
			return nil
		}
	}

	log.Debug().Str("username", username).Msg("tidak ada active session untuk user ini")
	return nil
}
