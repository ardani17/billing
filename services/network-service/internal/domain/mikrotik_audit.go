package domain

type MikroTikCommandAuditLog struct {
	TenantID     string
	RouterID     string
	UserID       string
	Action       string
	Command      string
	TargetType   string
	TargetID     string
	Status       string
	ErrorMessage string
	RemoteAddr   string
}
