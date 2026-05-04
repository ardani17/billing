package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type TerminalManager interface {
	Execute(ctx context.Context, routerID string, req domain.TerminalExecuteRequest) (*domain.TerminalExecuteResult, error)
	ListAudit(ctx context.Context, params domain.MikroTikCommandAuditListParams) (*domain.MikroTikCommandAuditListResult, error)
}

type terminalManager struct {
	routerRepo     domain.RouterRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewTerminalManager(
	routerRepo domain.RouterRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) TerminalManager {
	return &terminalManager{routerRepo: routerRepo, auditRepo: auditRepo, encryptor: encryptor, adapterFactory: adapterFactory}
}

var (
	terminalCommandPattern = regexp.MustCompile(`^/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*$`)
	terminalParamKey       = regexp.MustCompile(`^[=?][A-Za-z0-9_.-]+$`)
	terminalAllowed        = map[string]struct{}{
		"/system/resource/print":          {},
		"/system/identity/print":          {},
		"/interface/print":                {},
		"/interface/monitor-traffic":      {},
		"/ip/address/print":               {},
		"/ip/route/print":                 {},
		"/ip/dns/print":                   {},
		"/ip/pool/print":                  {},
		"/ip/pool/used/print":             {},
		"/ip/firewall/nat/print":          {},
		"/ip/firewall/filter/print":       {},
		"/ip/firewall/address-list/print": {},
		"/log/print":                      {},
		"/ip/dhcp-server/print":           {},
		"/ip/dhcp-server/lease/print":     {},
		"/ip/dhcp-server/network/print":   {},
		"/ppp/secret/print":               {},
		"/ppp/profile/print":              {},
		"/ppp/active/print":               {},
		"/ip/hotspot/user/print":          {},
		"/ip/hotspot/user/profile/print":  {},
		"/ip/hotspot/active/print":        {},
		"/queue/simple/print":             {},
	}
)

func (m *terminalManager) Execute(ctx context.Context, routerID string, req domain.TerminalExecuteRequest) (*domain.TerminalExecuteResult, error) {
	command, params, err := validateTerminalRequest(req)
	if err != nil {
		m.audit(ctx, routerID, commandForAudit(req.Command), "denied", err)
		return nil, fmt.Errorf("%w: %s", domain.ErrTerminalCommandDenied, err.Error())
	}

	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		m.audit(ctx, routerID, command, "failed", err)
		return nil, err
	}
	defer closeFn()

	rows, err := adapter.Execute(ctx, command, params)
	if err != nil {
		m.audit(ctx, routerID, command, "failed", err)
		return nil, err
	}
	m.audit(ctx, routerID, command, "success", nil)
	return &domain.TerminalExecuteResult{Command: command, Rows: rows}, nil
}

func (m *terminalManager) ListAudit(ctx context.Context, params domain.MikroTikCommandAuditListParams) (*domain.MikroTikCommandAuditListResult, error) {
	if m.auditRepo == nil {
		return &domain.MikroTikCommandAuditListResult{Data: []domain.MikroTikCommandAuditLog{}, Page: 1, PageSize: 20}, nil
	}
	return m.auditRepo.List(ctx, params)
}

func (m *terminalManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
	router, err := m.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, nil, err
	}
	password, err := m.encryptor.Decrypt(router.PasswordEncrypted)
	if err != nil {
		return nil, nil, domain.ErrDecryptionFailed
	}
	adapter := m.adapterFactory()
	cfg := domain.ConnectionConfig{
		Host: router.Host, Port: router.Port, Username: router.Username, Password: password,
		UseSSL: router.UseSSL, ConnectTimeout: 10 * time.Second, CommandTimeout: 10 * time.Second,
	}
	if err := adapter.Connect(ctx, cfg); err != nil {
		_ = adapter.Close()
		if errors.Is(err, domain.ErrConnectionTimeout) {
			return nil, nil, domain.ErrConnectionTimeout
		}
		return nil, nil, domain.ErrConnectionFailed
	}
	return adapter, func() { _ = adapter.Close() }, nil
}

func validateTerminalRequest(req domain.TerminalExecuteRequest) (string, map[string]string, error) {
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return "", nil, errors.New("command wajib diisi")
	}
	if strings.ContainsAny(command, " \t\r\n;|") || strings.Contains(command, "&&") || strings.Contains(command, "..") {
		return command, nil, errors.New("command harus path RouterOS tunggal tanpa argumen inline")
	}
	if !terminalCommandPattern.MatchString(command) {
		return command, nil, errors.New("format command RouterOS tidak valid")
	}
	if _, ok := terminalAllowed[command]; !ok {
		return command, nil, errors.New("hanya command read-only yang diizinkan")
	}

	params := make(map[string]string, len(req.Params))
	for rawKey, rawValue := range req.Params {
		key := strings.TrimSpace(rawKey)
		value := strings.TrimSpace(rawValue)
		if key == "" {
			return command, nil, errors.New("parameter key tidak boleh kosong")
		}
		if len(key) > 80 || len(value) > 240 {
			return command, nil, errors.New("parameter terlalu panjang")
		}
		if !terminalParamKey.MatchString(key) {
			return command, nil, errors.New("parameter hanya boleh memakai prefix = atau ?")
		}
		if strings.ContainsAny(value, "\r\n;|") || strings.Contains(value, "&&") {
			return command, nil, errors.New("parameter mengandung karakter berbahaya")
		}
		params[key] = value
	}
	if len(params) == 0 {
		return command, nil, nil
	}
	return command, params, nil
}

func commandForAudit(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return "(empty)"
	}
	if len(command) > 160 {
		return command[:160]
	}
	return command
}

func (m *terminalManager) audit(ctx context.Context, routerID, command, status string, err error) {
	if m.auditRepo == nil {
		return
	}
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_ = m.auditRepo.Create(ctx, domain.MikroTikCommandAuditLog{
		TenantID: tenant.FromContext(ctx), RouterID: routerID, UserID: actor.UserID,
		Action: "terminal.execute", Command: command, TargetType: "terminal",
		Status: status, ErrorMessage: msg, RemoteAddr: actor.RemoteAddr,
	})
}
