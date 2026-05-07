package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type BackupManager interface {
	CreateBackup(ctx context.Context, routerID string) (*domain.RouterBackup, error)
	ListBackups(ctx context.Context, params domain.RouterBackupListParams) (*domain.RouterBackupListResult, error)
	GetBackup(ctx context.Context, backupID string) (*domain.RouterBackup, error)
	DeleteBackup(ctx context.Context, backupID string) error
	GetFirmware(ctx context.Context, routerID string) (*domain.RouterFirmwareInfo, error)
}

type backupManager struct {
	routerRepo     domain.RouterRepository
	backupRepo     domain.RouterBackupRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

const routerInventoryBackupCommand = "read-only inventory"

func NewBackupManager(
	routerRepo domain.RouterRepository,
	backupRepo domain.RouterBackupRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) BackupManager {
	return &backupManager{routerRepo: routerRepo, backupRepo: backupRepo, auditRepo: auditRepo, encryptor: encryptor, adapterFactory: adapterFactory}
}

func (m *backupManager) CreateBackup(ctx context.Context, routerID string) (*domain.RouterBackup, error) {
	router, adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		m.audit(ctx, routerID, "backup.create", "/export", "failed", err)
		return nil, err
	}
	defer closeFn()

	now := time.Now().UTC()
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	fileName := fmt.Sprintf("%s-%s.rsc", sanitizeBackupName(router.Name), now.Format("20060102-150405"))
	fileBase := strings.TrimSuffix(fileName, ".rsc")
	content, command, err := m.exportRouterConfig(ctx, adapter, fileBase)
	if err != nil {
		m.audit(ctx, routerID, "backup.create", command, "failed", err)
		return nil, err
	}
	if command == routerInventoryBackupCommand {
		fileName = strings.TrimSuffix(fileName, ".rsc") + "-inventory.rsc"
	}
	sum := sha256.Sum256([]byte(content))
	item, err := m.backupRepo.Create(ctx, domain.CreateRouterBackupInput{
		TenantID: tenant.FromContext(ctx), RouterID: routerID, FileName: fileName, Format: "rsc",
		SizeBytes: int64(len([]byte(content))), Checksum: hex.EncodeToString(sum[:]), Content: content, CreatedBy: actor.UserID,
	})
	if err != nil {
		m.audit(ctx, routerID, "backup.create", "/export", "failed", err)
		return nil, err
	}
	m.audit(ctx, routerID, "backup.create", "/export", "success", nil)
	return item, nil
}

func (m *backupManager) exportRouterConfig(ctx context.Context, adapter domain.RouterOSAdapter, fileBase string) (string, string, error) {
	rows, err := adapter.Execute(ctx, "/export", map[string]string{"=terse": "", "=hide-sensitive": ""})
	if err != nil {
		return "", "/export", err
	}
	if content := exportRowsToScript(rows); strings.TrimSpace(content) != "" {
		return content, "/export", nil
	}
	content, command, err := m.exportRouterConfigViaFile(ctx, adapter, fileBase)
	if err == nil {
		return content, command, nil
	}
	if errors.Is(err, domain.ErrRouterPermissionDenied) {
		return exportRouterInventorySnapshot(ctx, adapter, err)
	}
	return "", command, err
}

func (m *backupManager) exportRouterConfigViaFile(ctx context.Context, adapter domain.RouterOSAdapter, fileBase string) (string, string, error) {
	targetName := fileBase + ".rsc"
	cleanupFileID := ""
	cleanup := func() {
		if cleanupFileID != "" {
			_, _ = adapter.Execute(context.Background(), "/file/remove", map[string]string{"=numbers": cleanupFileID})
			return
		}
		if row, err := findRouterFile(context.Background(), adapter, targetName); err == nil {
			if id := routerFileID(row); id != "" {
				_, _ = adapter.Execute(context.Background(), "/file/remove", map[string]string{"=numbers": id})
			}
		}
	}
	defer cleanup()

	if _, err := adapter.Execute(ctx, "/export", map[string]string{"=file": fileBase, "=terse": "", "=hide-sensitive": ""}); err != nil {
		return "", "/export file", err
	}

	var lastRow map[string]string
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		if attempt > 0 {
			time.Sleep(250 * time.Millisecond)
		}
		row, err := findRouterFile(ctx, adapter, targetName)
		if err == nil {
			lastRow = row
			cleanupFileID = routerFileID(row)
			break
		}
		lastErr = err
	}
	if lastRow == nil {
		if lastErr != nil {
			return "", "/file/print", lastErr
		}
		return "", "/file/print", errors.New("file export RouterOS tidak ditemukan")
	}

	content, err := readRouterFileContents(ctx, adapter, lastRow)
	if err != nil {
		return "", "/file/get", err
	}
	if strings.TrimSpace(content) == "" {
		return "", "/file/get", errors.New("hasil export RouterOS kosong")
	}
	return content, "/export file", nil
}

func exportRouterInventorySnapshot(ctx context.Context, adapter domain.RouterOSAdapter, exportErr error) (string, string, error) {
	sections := []string{
		"/system/identity/print",
		"/system/resource/print",
		"/interface/print",
		"/ip/address/print",
		"/ip/route/print",
		"/ip/dns/print",
		"/ip/pool/print",
		"/ip/dhcp-server/print",
		"/ip/dhcp-server/lease/print",
		"/ppp/profile/print",
		"/ppp/secret/print",
		"/ppp/active/print",
		"/ip/hotspot/user/profile/print",
		"/ip/hotspot/user/print",
		"/ip/hotspot/active/print",
		"/ip/firewall/nat/print",
		"/ip/firewall/filter/print",
		"/ip/firewall/address-list/print",
		"/queue/simple/print",
	}

	lines := []string{
		"# ISPBoss MikroTik read-only inventory backup",
		"# This file is generated from live RouterOS read-only API commands.",
		"# Full importable /export file was denied by RouterOS permissions.",
		"# Grant the API user write/ftp policy to enable full RouterOS export backup.",
		"# RouterOS error: " + exportErr.Error(),
		"# Generated at: " + time.Now().UTC().Format(time.RFC3339),
	}

	successCount := 0
	for _, command := range sections {
		rows, err := adapter.Execute(ctx, command, nil)
		lines = append(lines, "", "# "+command)
		if err != nil {
			lines = append(lines, "# error: "+err.Error())
			continue
		}
		successCount++
		if len(rows) == 0 {
			lines = append(lines, "# no rows")
			continue
		}
		for _, row := range rows {
			lines = append(lines, "# "+formatRouterRowForInventory(row))
		}
	}
	if successCount == 0 {
		return "", "/export file", exportErr
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), routerInventoryBackupCommand, nil
}

func (m *backupManager) ListBackups(ctx context.Context, params domain.RouterBackupListParams) (*domain.RouterBackupListResult, error) {
	return m.backupRepo.List(ctx, params)
}

func (m *backupManager) GetBackup(ctx context.Context, backupID string) (*domain.RouterBackup, error) {
	return m.backupRepo.GetByID(ctx, backupID)
}

func (m *backupManager) DeleteBackup(ctx context.Context, backupID string) error {
	item, err := m.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return err
	}
	if err := m.backupRepo.Delete(ctx, backupID); err != nil {
		return err
	}
	m.audit(ctx, item.RouterID, "backup.delete", "router_backups/delete", "success", nil)
	return nil
}

func (m *backupManager) GetFirmware(ctx context.Context, routerID string) (*domain.RouterFirmwareInfo, error) {
	_, adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	resourceRows, err := adapter.Execute(ctx, "/system/resource/print", nil)
	if err != nil {
		return nil, err
	}
	packageRows, err := adapter.Execute(ctx, "/system/package/print", map[string]string{"=.proplist": "name,version,disabled,scheduled"})
	if err != nil {
		return nil, err
	}
	boardRows, _ := adapter.Execute(ctx, "/system/routerboard/print", nil)
	info := &domain.RouterFirmwareInfo{}
	if len(resourceRows) > 0 {
		info.RouterOSVersion = resourceRows[0]["version"]
		info.Architecture = resourceRows[0]["architecture-name"]
		info.BoardName = resourceRows[0]["board-name"]
	}
	if len(boardRows) > 0 {
		info.FactoryFirmware = boardRows[0]["factory-firmware"]
		info.CurrentFirmware = boardRows[0]["current-firmware"]
		info.UpgradeFirmware = boardRows[0]["upgrade-firmware"]
		info.Outdated = info.UpgradeFirmware != "" && info.CurrentFirmware != "" && info.UpgradeFirmware != info.CurrentFirmware
		if info.Outdated {
			info.Warning = "firmware routerboard belum sama dengan upgrade firmware"
		}
	}
	info.Packages = make([]domain.RouterOSPackage, 0, len(packageRows))
	for _, row := range packageRows {
		info.Packages = append(info.Packages, domain.RouterOSPackage{
			Name: row["name"], Version: row["version"], Disabled: parseRouterOSBool(row["disabled"]), Scheduled: row["scheduled"],
		})
	}
	m.audit(ctx, routerID, "firmware.read", "/system/package/print", "success", nil)
	return info, nil
}

func (m *backupManager) connect(ctx context.Context, routerID string) (*domain.Router, domain.RouterOSAdapter, func(), error) {
	router, err := m.routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, nil, nil, err
	}
	password, err := m.encryptor.Decrypt(router.PasswordEncrypted)
	if err != nil {
		return nil, nil, nil, domain.ErrDecryptionFailed
	}
	adapter := m.adapterFactory()
	cfg := domain.ConnectionConfig{
		Host: router.Host, Port: router.Port, Username: router.Username, Password: password,
		UseSSL: router.UseSSL, ConnectTimeout: 10 * time.Second, CommandTimeout: 20 * time.Second,
	}
	if err := adapter.Connect(ctx, cfg); err != nil {
		_ = adapter.Close()
		if errors.Is(err, domain.ErrConnectionTimeout) {
			return nil, nil, nil, domain.ErrConnectionTimeout
		}
		return nil, nil, nil, domain.ErrConnectionFailed
	}
	return router, adapter, func() { _ = adapter.Close() }, nil
}

func exportRowsToScript(rows []map[string]string) string {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		if ret := strings.TrimSpace(row["ret"]); ret != "" {
			lines = append(lines, ret)
			continue
		}
		if line := strings.TrimSpace(row["line"]); line != "" {
			lines = append(lines, line)
			continue
		}
		if len(row) == 0 {
			continue
		}
		keys := make([]string, 0, len(row))
		for key := range row {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%s", key, row[key]))
		}
		lines = append(lines, strings.Join(parts, " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func findRouterFile(ctx context.Context, adapter domain.RouterOSAdapter, name string) (map[string]string, error) {
	rows, err := adapter.Execute(ctx, "/file/print", map[string]string{
		"?name":      name,
		"=.proplist": ".id,name,size,contents",
	})
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row["name"] == name {
			return row, nil
		}
	}
	return nil, errors.New("file export RouterOS tidak ditemukan")
}

func readRouterFileContents(ctx context.Context, adapter domain.RouterOSAdapter, row map[string]string) (string, error) {
	if content := strings.TrimSpace(row["contents"]); content != "" {
		return content, nil
	}
	id := routerFileID(row)
	if id == "" {
		return "", errors.New("file export RouterOS tidak memiliki id")
	}
	rows, err := adapter.Execute(ctx, "/file/get", map[string]string{"=numbers": id, "=value-name": "contents"})
	if err != nil {
		return "", err
	}
	return exportRowsToScript(rows), nil
}

func routerFileID(row map[string]string) string {
	if id := strings.TrimSpace(row[".id"]); id != "" {
		return id
	}
	return strings.TrimSpace(row["id"])
}

func formatRouterRowForInventory(row map[string]string) string {
	if len(row) == 0 {
		return "(empty)"
	}
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := row[key]
		if isSensitiveRouterKey(key) {
			value = "[redacted]"
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, " ")
}

func isSensitiveRouterKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") ||
		strings.Contains(key, "secret") ||
		strings.Contains(key, "private") ||
		strings.Contains(key, "key") ||
		strings.Contains(key, "certificate")
}

func sanitizeBackupName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "router"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-_")
	if name == "" {
		return "router"
	}
	return name
}

func (m *backupManager) audit(ctx context.Context, routerID, action, command, status string, err error) {
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
		Action: action, Command: command, TargetType: "router_backup",
		Status: status, ErrorMessage: msg, RemoteAddr: actor.RemoteAddr,
	})
}
