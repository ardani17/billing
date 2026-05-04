package usecase

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const hotspotCommentPrefix = "ISPBoss:hotspot:"

type HotspotManager interface {
	ListUsers(ctx context.Context, routerID string) ([]domain.HotspotUser, error)
	CreateUser(ctx context.Context, routerID string, req domain.CreateHotspotUserRequest) (*domain.HotspotUser, error)
	UpdateUser(ctx context.Context, routerID, userID string, req domain.UpdateHotspotUserRequest) (*domain.HotspotUser, error)
	DeleteUser(ctx context.Context, routerID, userID string) error
	ListProfiles(ctx context.Context, routerID string) ([]domain.HotspotProfile, error)
	ListActive(ctx context.Context, routerID string) ([]domain.HotspotActiveSession, error)
	GenerateLoginTemplate(ctx context.Context, routerID string, req domain.HotspotLoginTemplateRequest) (*domain.HotspotLoginTemplate, error)
}

type hotspotManager struct {
	routerRepo     domain.RouterRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewHotspotManager(
	routerRepo domain.RouterRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) HotspotManager {
	return &hotspotManager{routerRepo: routerRepo, auditRepo: auditRepo, encryptor: encryptor, adapterFactory: adapterFactory}
}

func (m *hotspotManager) ListUsers(ctx context.Context, routerID string) ([]domain.HotspotUser, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/hotspot/user/print", map[string]string{
		"=.proplist": ".id,name,password,profile,limit-uptime,uptime,bytes-in,bytes-out,disabled,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseHotspotUsers(rows), nil
}

func (m *hotspotManager) CreateUser(ctx context.Context, routerID string, req domain.CreateHotspotUserRequest) (*domain.HotspotUser, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	comment := strings.TrimSpace(req.Comment)
	if comment == "" {
		comment = hotspotCommentPrefix + strings.TrimSpace(req.Name)
	} else if !strings.HasPrefix(comment, hotspotCommentPrefix) {
		comment = hotspotCommentPrefix + comment
	}
	params := map[string]string{
		"=name":     strings.TrimSpace(req.Name),
		"=password": strings.TrimSpace(req.Password),
		"=profile":  strings.TrimSpace(req.Profile),
		"=comment":  comment,
	}
	if value := strings.TrimSpace(req.LimitUptime); value != "" {
		params["=limit-uptime"] = value
	}
	if _, err := adapter.Execute(ctx, "/ip/hotspot/user/add", params); err != nil {
		m.audit(ctx, routerID, "hotspot.user.create", "/ip/hotspot/user/add", "failed", err)
		return nil, err
	}
	m.audit(ctx, routerID, "hotspot.user.create", "/ip/hotspot/user/add", "success", nil)
	return findHotspotUserByName(ctx, adapter, req.Name)
}

func (m *hotspotManager) UpdateUser(ctx context.Context, routerID, userID string, req domain.UpdateHotspotUserRequest) (*domain.HotspotUser, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	params := map[string]string{"=numbers": userID}
	if req.Password != nil {
		params["=password"] = strings.TrimSpace(*req.Password)
	}
	if req.Profile != nil {
		params["=profile"] = strings.TrimSpace(*req.Profile)
	}
	if req.LimitUptime != nil {
		params["=limit-uptime"] = strings.TrimSpace(*req.LimitUptime)
	}
	if req.Disabled != nil {
		params["=disabled"] = boolToRouterOS(*req.Disabled)
	}
	if req.Comment != nil {
		comment := strings.TrimSpace(*req.Comment)
		if comment != "" && !strings.HasPrefix(comment, hotspotCommentPrefix) {
			comment = hotspotCommentPrefix + comment
		}
		params["=comment"] = comment
	}
	if len(params) <= 1 {
		return findHotspotUserByID(ctx, adapter, userID)
	}
	if _, err := adapter.Execute(ctx, "/ip/hotspot/user/set", params); err != nil {
		m.audit(ctx, routerID, "hotspot.user.update", "/ip/hotspot/user/set", "failed", err)
		return nil, err
	}
	m.audit(ctx, routerID, "hotspot.user.update", "/ip/hotspot/user/set", "success", nil)
	return findHotspotUserByID(ctx, adapter, userID)
}

func (m *hotspotManager) DeleteUser(ctx context.Context, routerID, userID string) error {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return err
	}
	defer closeFn()
	if _, err := adapter.Execute(ctx, "/ip/hotspot/user/remove", map[string]string{"=numbers": userID}); err != nil {
		m.audit(ctx, routerID, "hotspot.user.delete", "/ip/hotspot/user/remove", "failed", err)
		return err
	}
	m.audit(ctx, routerID, "hotspot.user.delete", "/ip/hotspot/user/remove", "success", nil)
	return nil
}

func (m *hotspotManager) ListProfiles(ctx context.Context, routerID string) ([]domain.HotspotProfile, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/hotspot/user/profile/print", map[string]string{
		"=.proplist": ".id,name,rate-limit,shared-users,address-pool,transparent-proxy,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseHotspotProfiles(rows), nil
}

func (m *hotspotManager) ListActive(ctx context.Context, routerID string) ([]domain.HotspotActiveSession, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/hotspot/active/print", map[string]string{
		"=.proplist": ".id,user,address,mac-address,uptime,bytes-in,bytes-out,server",
	})
	if err != nil {
		return nil, err
	}
	return parseHotspotActive(rows), nil
}

func (m *hotspotManager) GenerateLoginTemplate(_ context.Context, _ string, req domain.HotspotLoginTemplateRequest) (*domain.HotspotLoginTemplate, error) {
	brand := strings.TrimSpace(req.BrandName)
	if brand == "" {
		brand = "ISPBoss Hotspot"
	}
	color := strings.TrimSpace(req.PrimaryColor)
	if color == "" {
		color = "#2563eb"
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = "Masukkan voucher Anda untuk mulai menggunakan internet."
	}
	data := struct {
		Brand   string
		Color   string
		Phone   string
		Message string
	}{
		Brand: template.HTMLEscapeString(brand), Color: template.HTMLEscapeString(color),
		Phone: template.HTMLEscapeString(strings.TrimSpace(req.SupportPhone)), Message: template.HTMLEscapeString(message),
	}
	html := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s Hotspot</title><style>body{margin:0;font-family:Arial,sans-serif;background:#f8fafc;color:#0f172a}.wrap{min-height:100vh;display:grid;place-items:center;padding:24px}.box{width:min(420px,100%%);background:white;border:1px solid #e2e8f0;border-radius:14px;padding:28px;box-shadow:0 24px 80px rgba(15,23,42,.12)}h1{margin:0 0 8px;font-size:24px}.msg{margin:0 0 22px;color:#475569;line-height:1.55}.field{width:100%%;height:44px;border:1px solid #cbd5e1;border-radius:10px;padding:0 12px;margin-bottom:12px}button{width:100%%;height:44px;border:0;border-radius:10px;background:%s;color:white;font-weight:700}.help{margin-top:16px;font-size:13px;color:#64748b}</style></head>
<body><main class="wrap"><form class="box" name="login" action="$(link-login-only)" method="post">
<input type="hidden" name="dst" value="$(link-orig)"><input type="hidden" name="popup" value="true">
<h1>%s</h1><p class="msg">%s</p><input class="field" name="username" placeholder="Kode voucher" autocomplete="username">
<input class="field" name="password" type="password" placeholder="Password"><button type="submit">Masuk</button>
<div class="help">%s</div></form></main></body></html>`, data.Brand, data.Color, data.Brand, data.Message, data.Phone)
	return &domain.HotspotLoginTemplate{FileName: "login.html", HTML: html}, nil
}

func (m *hotspotManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
	router, err := m.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, nil, err
	}
	password, err := m.encryptor.Decrypt(router.PasswordEncrypted)
	if err != nil {
		return nil, nil, domain.ErrDecryptionFailed
	}
	adapter := m.adapterFactory()
	cfg := domain.ConnectionConfig{Host: router.Host, Port: router.Port, Username: router.Username, Password: password, UseSSL: router.UseSSL, ConnectTimeout: 10 * time.Second, CommandTimeout: 10 * time.Second}
	if err := adapter.Connect(ctx, cfg); err != nil {
		_ = adapter.Close()
		if errors.Is(err, domain.ErrConnectionTimeout) {
			return nil, nil, domain.ErrConnectionTimeout
		}
		return nil, nil, domain.ErrConnectionFailed
	}
	return adapter, func() { _ = adapter.Close() }, nil
}

func parseHotspotUsers(rows []map[string]string) []domain.HotspotUser {
	items := make([]domain.HotspotUser, 0, len(rows))
	for _, row := range rows {
		comment := row["comment"]
		items = append(items, domain.HotspotUser{
			ID: row[".id"], Name: row["name"], Password: row["password"], Profile: row["profile"],
			LimitUptime: row["limit-uptime"], Uptime: row["uptime"], BytesIn: parseHotspotInt64(row["bytes-in"]),
			BytesOut: parseHotspotInt64(row["bytes-out"]), Disabled: parseRouterOSBool(row["disabled"]),
			Comment: comment, Managed: strings.HasPrefix(comment, hotspotCommentPrefix),
		})
	}
	return items
}

func parseHotspotProfiles(rows []map[string]string) []domain.HotspotProfile {
	items := make([]domain.HotspotProfile, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.HotspotProfile{
			ID: row[".id"], Name: row["name"], RateLimit: row["rate-limit"],
			SharedUsers: row["shared-users"], AddressPool: row["address-pool"],
			Transparent: parseRouterOSBool(row["transparent-proxy"]), Comment: row["comment"],
		})
	}
	return items
}

func parseHotspotActive(rows []map[string]string) []domain.HotspotActiveSession {
	items := make([]domain.HotspotActiveSession, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.HotspotActiveSession{
			ID: row[".id"], User: row["user"], Address: row["address"], MACAddress: row["mac-address"],
			Uptime: row["uptime"], BytesIn: parseHotspotInt64(row["bytes-in"]), BytesOut: parseHotspotInt64(row["bytes-out"]),
			Server: row["server"],
		})
	}
	return items
}

func findHotspotUserByName(ctx context.Context, adapter domain.RouterOSAdapter, name string) (*domain.HotspotUser, error) {
	rows, err := adapter.Execute(ctx, "/ip/hotspot/user/print", map[string]string{"=.proplist": ".id,name,password,profile,limit-uptime,uptime,bytes-in,bytes-out,disabled,comment"})
	if err != nil {
		return nil, err
	}
	for _, user := range parseHotspotUsers(rows) {
		if user.Name == strings.TrimSpace(name) {
			return &user, nil
		}
	}
	return nil, domain.ErrHotspotUserNotFound
}

func findHotspotUserByID(ctx context.Context, adapter domain.RouterOSAdapter, id string) (*domain.HotspotUser, error) {
	rows, err := adapter.Execute(ctx, "/ip/hotspot/user/print", map[string]string{"=.proplist": ".id,name,password,profile,limit-uptime,uptime,bytes-in,bytes-out,disabled,comment"})
	if err != nil {
		return nil, err
	}
	for _, user := range parseHotspotUsers(rows) {
		if user.ID == id || user.Name == id {
			return &user, nil
		}
	}
	return nil, domain.ErrHotspotUserNotFound
}

func parseHotspotInt64(value string) int64 {
	n, _ := strconv.ParseInt(value, 10, 64)
	return n
}

func boolToRouterOS(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func (m *hotspotManager) audit(ctx context.Context, routerID, action, command, status string, err error) {
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
		Action: action, Command: command, TargetType: "hotspot_user",
		Status: status, ErrorMessage: msg, RemoteAddr: actor.RemoteAddr,
	})
}
