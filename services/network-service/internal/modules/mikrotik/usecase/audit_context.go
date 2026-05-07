package usecase

import "context"

type mikrotikAuditContextKey string

const mikrotikAuditActorKey mikrotikAuditContextKey = "mikrotik_audit_actor"

type mikrotikAuditActor struct {
	UserID     string
	RemoteAddr string
}

func WithMikroTikAuditActor(ctx context.Context, userID, remoteAddr string) context.Context {
	return context.WithValue(ctx, mikrotikAuditActorKey, mikrotikAuditActor{UserID: userID, RemoteAddr: remoteAddr})
}
