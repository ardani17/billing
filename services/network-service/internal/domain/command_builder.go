package domain

// =============================================================================
// CommandBuilder — abstraksi perintah RouterOS untuk v6 dan v7
// =============================================================================

// CommandBuilder membangun perintah RouterOS yang kompatibel dengan v6 dan v7.
// Mengabstraksi perbedaan API path dan parameter antar versi.
type CommandBuilder interface {
	// CreateSecret membangun perintah /ppp/secret/add.
	CreateSecret(params PPPoESecretParams) (command string, args map[string]string)

	// SetSecret membangun perintah /ppp/secret/set.
	SetSecret(username string, params map[string]string) (command string, args map[string]string)

	// RemoveSecret membangun perintah /ppp/secret/remove.
	RemoveSecret(username string) (command string, args map[string]string)

	// PrintSecrets membangun perintah /ppp/secret/print.
	PrintSecrets() (command string, args map[string]string)

	// RemoveActiveSession membangun perintah /ppp/active/remove.
	RemoveActiveSession(sessionID string) (command string, args map[string]string)

	// PrintActiveSessions membangun perintah /ppp/active/print.
	PrintActiveSessions() (command string, args map[string]string)

	// CreateProfile membangun perintah /ppp/profile/add.
	CreateProfile(params PPPoEProfileParams) (command string, args map[string]string)

	// SetProfile membangun perintah /ppp/profile/set.
	SetProfile(name string, params map[string]string) (command string, args map[string]string)

	// CreateNATRule membangun perintah /ip/firewall/nat/add.
	CreateNATRule(params NATRuleParams) (command string, args map[string]string)

	// RemoveNATRuleByComment membangun perintah /ip/firewall/nat/remove dengan find by comment.
	RemoveNATRuleByComment(comment string) (command string, args map[string]string)

	// CreateSimpleQueue membangun perintah /queue/simple/add.
	CreateSimpleQueue(params SimpleQueueParams) (command string, args map[string]string)

	// SetSimpleQueue membangun perintah /queue/simple/set.
	SetSimpleQueue(name string, params map[string]string) (command string, args map[string]string)

	// RemoveSimpleQueue membangun perintah /queue/simple/remove.
	RemoveSimpleQueue(name string) (command string, args map[string]string)

	// ResetSimpleQueueCounters membangun perintah /queue/simple/reset-counters.
	// Digunakan saat buka isolir untuk reset traffic counter.
	ResetSimpleQueueCounters(name string) (command string, args map[string]string)
}
