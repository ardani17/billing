package usecase

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const (
	staticIPActiveList    = "ISPBoss:static-active"
	staticIPIsolatedList  = "ISPBoss:static-isolated"
	staticIPCommentPrefix = "ISPBoss:static:"
)

type StaticIPManager interface {
	ListAssignments(ctx context.Context, routerID string, params domain.StaticIPAssignmentListParams) (*domain.StaticIPAssignmentListResult, error)
	CreateAssignment(ctx context.Context, tenantID, routerID string, req domain.CreateStaticIPAssignmentRequest) (*domain.StaticIPAssignmentResponse, error)
	UpdateAssignment(ctx context.Context, routerID, assignmentID string, req domain.UpdateStaticIPAssignmentRequest) (*domain.StaticIPAssignmentResponse, error)
	DeleteAssignment(ctx context.Context, routerID, assignmentID string, req domain.DeleteStaticIPAssignmentRequest) error
	IsolateAssignment(ctx context.Context, routerID, assignmentID string) (*domain.StaticIPAssignmentResponse, error)
	UnisolateAssignment(ctx context.Context, routerID, assignmentID string) (*domain.StaticIPAssignmentResponse, error)
}

type staticIPManager struct {
	routerRepo     domain.RouterRepository
	assignmentRepo domain.StaticIPAssignmentRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewStaticIPManager(
	routerRepo domain.RouterRepository,
	assignmentRepo domain.StaticIPAssignmentRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) StaticIPManager {
	return &staticIPManager{routerRepo: routerRepo, assignmentRepo: assignmentRepo, auditRepo: auditRepo, encryptor: encryptor, adapterFactory: adapterFactory}
}

func (m *staticIPManager) ListAssignments(ctx context.Context, routerID string, params domain.StaticIPAssignmentListParams) (*domain.StaticIPAssignmentListResult, error) {
	params.RouterID = routerID
	return m.assignmentRepo.List(ctx, params)
}

func (m *staticIPManager) CreateAssignment(ctx context.Context, tenantID, routerID string, req domain.CreateStaticIPAssignmentRequest) (*domain.StaticIPAssignmentResponse, error) {
	if net.ParseIP(req.IPAddress) == nil {
		return nil, domain.ErrInvalidIPAddress
	}
	item := &domain.StaticIPAssignment{
		TenantID:    tenantID,
		RouterID:    routerID,
		CustomerID:  strings.TrimSpace(req.CustomerID),
		IPAddress:   strings.TrimSpace(req.IPAddress),
		AddressList: staticIPActiveList,
		QueueName:   strings.TrimSpace(req.QueueName),
		RateLimit:   strings.TrimSpace(req.RateLimit),
		Comment:     strings.TrimSpace(req.Comment),
		Status:      domain.StaticIPStatusActive,
		SyncStatus:  "pending_create",
	}
	if item.Comment == "" {
		item.Comment = "Static IP customer"
	}
	if item.QueueName == "" && item.RateLimit != "" {
		item.QueueName = "ISPBoss-static-" + strings.ReplaceAll(item.IPAddress, ".", "-")
	}
	created, err := m.assignmentRepo.Create(ctx, item)
	if err != nil {
		if !errors.Is(err, domain.ErrStaticIPAssignmentExists) {
			return nil, err
		}
		existing, findErr := m.assignmentRepo.GetByRouterAndIP(ctx, routerID, item.IPAddress)
		if findErr != nil {
			return nil, err
		}
		existing.CustomerID = item.CustomerID
		existing.AddressList = item.AddressList
		existing.QueueName = item.QueueName
		existing.RateLimit = item.RateLimit
		existing.Comment = item.Comment
		existing.Status = item.Status
		existing.SyncStatus = "pending_update"
		created, err = m.assignmentRepo.Update(ctx, existing)
		if err != nil {
			return nil, err
		}
	}
	if err := m.syncAssignment(ctx, created); err != nil {
		_ = m.assignmentRepo.UpdateSyncState(ctx, created.ID, "error", nil)
		m.audit(ctx, created, "static_ip.create", "/ip/firewall/address-list/add", "failed", err)
		return nil, err
	}
	m.audit(ctx, created, "static_ip.create", "/ip/firewall/address-list/add", "success", nil)
	refreshed, err := m.assignmentRepo.GetByID(ctx, created.ID)
	if err != nil {
		return created.ToResponse(), nil
	}
	return refreshed.ToResponse(), nil
}

func (m *staticIPManager) UpdateAssignment(ctx context.Context, routerID, assignmentID string, req domain.UpdateStaticIPAssignmentRequest) (*domain.StaticIPAssignmentResponse, error) {
	item, err := m.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if item.RouterID != routerID {
		return nil, domain.ErrStaticIPAssignmentNotFound
	}
	if req.CustomerID != nil {
		item.CustomerID = strings.TrimSpace(*req.CustomerID)
	}
	if req.IPAddress != nil {
		if net.ParseIP(*req.IPAddress) == nil {
			return nil, domain.ErrInvalidIPAddress
		}
		item.IPAddress = strings.TrimSpace(*req.IPAddress)
	}
	if req.QueueName != nil {
		item.QueueName = strings.TrimSpace(*req.QueueName)
	}
	if req.RateLimit != nil {
		item.RateLimit = strings.TrimSpace(*req.RateLimit)
	}
	if req.Comment != nil {
		item.Comment = strings.TrimSpace(*req.Comment)
	}
	if req.Status != nil {
		item.Status = strings.TrimSpace(*req.Status)
		item.AddressList = addressListForStaticStatus(item.Status)
	}
	item.SyncStatus = "pending_update"
	updated, err := m.assignmentRepo.Update(ctx, item)
	if err != nil {
		return nil, err
	}
	if err := m.syncAssignment(ctx, updated); err != nil {
		_ = m.assignmentRepo.UpdateSyncState(ctx, updated.ID, "error", nil)
		m.audit(ctx, updated, "static_ip.update", "/ip/firewall/address-list/set", "failed", err)
		return nil, err
	}
	m.audit(ctx, updated, "static_ip.update", "/ip/firewall/address-list/set", "success", nil)
	refreshed, err := m.assignmentRepo.GetByID(ctx, updated.ID)
	if err != nil {
		return updated.ToResponse(), nil
	}
	return refreshed.ToResponse(), nil
}

func (m *staticIPManager) IsolateAssignment(ctx context.Context, routerID, assignmentID string) (*domain.StaticIPAssignmentResponse, error) {
	status := domain.StaticIPStatusIsolated
	return m.UpdateAssignment(ctx, routerID, assignmentID, domain.UpdateStaticIPAssignmentRequest{Status: &status})
}

func (m *staticIPManager) UnisolateAssignment(ctx context.Context, routerID, assignmentID string) (*domain.StaticIPAssignmentResponse, error) {
	status := domain.StaticIPStatusActive
	return m.UpdateAssignment(ctx, routerID, assignmentID, domain.UpdateStaticIPAssignmentRequest{Status: &status})
}

func (m *staticIPManager) DeleteAssignment(ctx context.Context, routerID, assignmentID string, req domain.DeleteStaticIPAssignmentRequest) error {
	item, err := m.assignmentRepo.GetByID(ctx, assignmentID)
	if err != nil {
		return err
	}
	if item.RouterID != routerID {
		return domain.ErrStaticIPAssignmentNotFound
	}
	if strings.TrimSpace(req.ConfirmIP) != item.IPAddress {
		return domain.ErrConfirmationMismatch
	}
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return err
	}
	defer closeFn()
	if listID, _ := findStaticAddressListID(ctx, adapter, item); listID != "" {
		if _, err := adapter.Execute(ctx, "/ip/firewall/address-list/remove", map[string]string{"=numbers": listID}); err != nil {
			m.audit(ctx, item, "static_ip.delete", "/ip/firewall/address-list/remove", "failed", err)
			return err
		}
	}
	if item.QueueName != "" {
		_ = removeStaticQueue(ctx, adapter, item)
	}
	m.audit(ctx, item, "static_ip.delete", "/ip/firewall/address-list/remove", "success", nil)
	return m.assignmentRepo.SoftDelete(ctx, assignmentID)
}

func (m *staticIPManager) syncAssignment(ctx context.Context, item *domain.StaticIPAssignment) error {
	adapter, closeFn, err := m.connect(ctx, item.RouterID)
	if err != nil {
		return err
	}
	defer closeFn()
	listID, err := findStaticAddressListID(ctx, adapter, item)
	if err != nil {
		return err
	}
	params := map[string]string{
		"=list":    item.AddressList,
		"=address": item.IPAddress,
		"=comment": staticIPComment(item),
	}
	if listID != "" {
		params["=numbers"] = listID
		if _, err := adapter.Execute(ctx, "/ip/firewall/address-list/set", params); err != nil {
			return err
		}
	} else if _, err := adapter.Execute(ctx, "/ip/firewall/address-list/add", params); err != nil {
		return err
	}
	if item.QueueName != "" && item.RateLimit != "" {
		if err := upsertStaticQueue(ctx, adapter, item); err != nil {
			return err
		}
	}
	now := time.Now()
	return m.assignmentRepo.UpdateSyncState(ctx, item.ID, "synced", &now)
}

func (m *staticIPManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
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

func findStaticAddressListID(ctx context.Context, adapter domain.RouterOSAdapter, item *domain.StaticIPAssignment) (string, error) {
	rows, err := adapter.Execute(ctx, "/ip/firewall/address-list/print", map[string]string{"=.proplist": ".id,list,address,comment"})
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if strings.HasPrefix(row["comment"], staticIPCommentPrefix+item.ID) || row["address"] == item.IPAddress {
			return row[".id"], nil
		}
	}
	return "", nil
}

func upsertStaticQueue(ctx context.Context, adapter domain.RouterOSAdapter, item *domain.StaticIPAssignment) error {
	rows, err := adapter.Execute(ctx, "/queue/simple/print", map[string]string{"=.proplist": ".id,name,comment"})
	if err != nil {
		return err
	}
	params := map[string]string{
		"=name":      item.QueueName,
		"=target":    item.IPAddress + "/32",
		"=max-limit": item.RateLimit,
		"=comment":   staticIPComment(item),
	}
	for _, row := range rows {
		if row["name"] == item.QueueName || strings.HasPrefix(row["comment"], staticIPCommentPrefix+item.ID) {
			params["=numbers"] = row[".id"]
			_, err := adapter.Execute(ctx, "/queue/simple/set", params)
			return err
		}
	}
	_, err = adapter.Execute(ctx, "/queue/simple/add", params)
	return err
}

func removeStaticQueue(ctx context.Context, adapter domain.RouterOSAdapter, item *domain.StaticIPAssignment) error {
	rows, err := adapter.Execute(ctx, "/queue/simple/print", map[string]string{"=.proplist": ".id,name,comment"})
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row["name"] == item.QueueName || strings.HasPrefix(row["comment"], staticIPCommentPrefix+item.ID) {
			_, err := adapter.Execute(ctx, "/queue/simple/remove", map[string]string{"=numbers": row[".id"]})
			return err
		}
	}
	return nil
}

func staticIPComment(item *domain.StaticIPAssignment) string {
	comment := strings.TrimSpace(item.Comment)
	if comment == "" {
		return staticIPCommentPrefix + item.ID
	}
	return fmt.Sprintf("%s%s %s", staticIPCommentPrefix, item.ID, comment)
}

func addressListForStaticStatus(status string) string {
	if status == domain.StaticIPStatusIsolated {
		return staticIPIsolatedList
	}
	return staticIPActiveList
}

func (m *staticIPManager) audit(ctx context.Context, item *domain.StaticIPAssignment, action, command, status string, err error) {
	if m.auditRepo == nil || item == nil {
		return
	}
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_ = m.auditRepo.Create(ctx, domain.MikroTikCommandAuditLog{
		TenantID:     item.TenantID,
		RouterID:     item.RouterID,
		UserID:       actor.UserID,
		Action:       action,
		Command:      command,
		TargetType:   "static_ip_assignment",
		TargetID:     item.ID,
		Status:       status,
		ErrorMessage: msg,
		RemoteAddr:   actor.RemoteAddr,
	})
}
