package usecase

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const dhcpCommentPrefix = "ISPBoss:dhcp:"

type DHCPManager interface {
	ListServers(ctx context.Context, routerID string) ([]domain.DHCPServer, error)
	ListLeases(ctx context.Context, routerID string) ([]domain.DHCPLease, error)
	ListNetworks(ctx context.Context, routerID string) ([]domain.DHCPNetwork, error)
	ListBindings(ctx context.Context, routerID string, params domain.DHCPBindingListParams) (*domain.DHCPBindingListResult, error)
	CreateBinding(ctx context.Context, tenantID, routerID string, req domain.CreateDHCPBindingRequest) (*domain.DHCPBindingResponse, error)
	UpdateBinding(ctx context.Context, routerID, bindingID string, req domain.UpdateDHCPBindingRequest) (*domain.DHCPBindingResponse, error)
	DeleteBinding(ctx context.Context, routerID, bindingID string, req domain.DeleteDHCPBindingRequest) error
}

type dhcpManager struct {
	routerRepo     domain.RouterRepository
	bindingRepo    domain.DHCPBindingRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewDHCPManager(
	routerRepo domain.RouterRepository,
	bindingRepo domain.DHCPBindingRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) DHCPManager {
	return &dhcpManager{routerRepo: routerRepo, bindingRepo: bindingRepo, auditRepo: auditRepo, encryptor: encryptor, adapterFactory: adapterFactory}
}

func (m *dhcpManager) ListServers(ctx context.Context, routerID string) ([]domain.DHCPServer, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/dhcp-server/print", map[string]string{
		"=.proplist": ".id,name,interface,address-pool,lease-time,authoritative,disabled,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseDHCPServers(rows), nil
}

func (m *dhcpManager) ListLeases(ctx context.Context, routerID string) ([]domain.DHCPLease, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/dhcp-server/lease/print", map[string]string{
		"=.proplist": ".id,server,address,mac-address,host-name,client-id,status,dynamic,disabled,expires-after,last-seen,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseDHCPLeases(rows), nil
}

func (m *dhcpManager) ListNetworks(ctx context.Context, routerID string) ([]domain.DHCPNetwork, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	rows, err := adapter.Execute(ctx, "/ip/dhcp-server/network/print", map[string]string{
		"=.proplist": ".id,address,gateway,dns-server,domain,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseDHCPNetworks(rows), nil
}

func (m *dhcpManager) ListBindings(ctx context.Context, routerID string, params domain.DHCPBindingListParams) (*domain.DHCPBindingListResult, error) {
	params.RouterID = routerID
	result, err := m.bindingRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}
	leases, liveErr := m.ListLeases(ctx, routerID)
	if liveErr != nil {
		return result, nil
	}
	leaseByMAC := map[string]domain.DHCPLease{}
	for _, lease := range leases {
		leaseByMAC[normalizeMAC(lease.MACAddress)] = lease
	}
	for _, item := range result.Data {
		if lease, ok := leaseByMAC[normalizeMAC(item.MACAddress)]; ok {
			item.RouterLeaseID = lease.ID
			item.Disabled = lease.Disabled
			if item.SyncStatus == "synced" && item.IPAddress != lease.Address {
				item.SyncStatus = "out_of_sync"
			}
		}
	}
	return result, nil
}

func (m *dhcpManager) CreateBinding(ctx context.Context, tenantID, routerID string, req domain.CreateDHCPBindingRequest) (*domain.DHCPBindingResponse, error) {
	if err := validateDHCPBinding(req.MACAddress, req.IPAddress); err != nil {
		return nil, err
	}
	server := strings.TrimSpace(req.Server)
	if server == "" {
		server = "all"
	}
	binding := &domain.DHCPBinding{
		TenantID:   tenantID,
		RouterID:   routerID,
		CustomerID: strings.TrimSpace(req.CustomerID),
		Server:     server,
		MACAddress: normalizeMAC(req.MACAddress),
		IPAddress:  strings.TrimSpace(req.IPAddress),
		HostName:   strings.TrimSpace(req.HostName),
		Comment:    strings.TrimSpace(req.Comment),
		Disabled:   req.Disabled,
		Status:     statusFromDisabled(req.Disabled),
		SyncStatus: "pending_create",
	}
	created, err := m.bindingRepo.Create(ctx, binding)
	if err != nil {
		if !errors.Is(err, domain.ErrDHCPBindingExists) {
			return nil, err
		}
		existing, findErr := m.bindingRepo.GetByRouterAndMAC(ctx, routerID, binding.MACAddress)
		if findErr != nil {
			existing, findErr = m.bindingRepo.GetByRouterAndIP(ctx, routerID, binding.IPAddress)
		}
		if findErr != nil {
			return nil, err
		}
		existing.CustomerID = binding.CustomerID
		existing.Server = binding.Server
		existing.MACAddress = binding.MACAddress
		existing.IPAddress = binding.IPAddress
		existing.HostName = binding.HostName
		existing.Comment = binding.Comment
		existing.Disabled = binding.Disabled
		existing.Status = binding.Status
		existing.SyncStatus = "pending_update"
		created, err = m.bindingRepo.Update(ctx, existing)
		if err != nil {
			return nil, err
		}
	}
	if err := m.syncBindingToRouter(ctx, created); err != nil {
		_ = m.bindingRepo.UpdateSyncState(ctx, created.ID, created.RouterLeaseID, "error", nil)
		m.audit(ctx, created, "dhcp.binding.create", "/ip/dhcp-server/lease/add", "failed", err)
		return nil, err
	}
	m.audit(ctx, created, "dhcp.binding.create", "/ip/dhcp-server/lease/add", "success", nil)
	refreshed, err := m.bindingRepo.GetByID(ctx, created.ID)
	if err != nil {
		return created.ToResponse(), nil
	}
	return refreshed.ToResponse(), nil
}

func (m *dhcpManager) UpdateBinding(ctx context.Context, routerID, bindingID string, req domain.UpdateDHCPBindingRequest) (*domain.DHCPBindingResponse, error) {
	binding, err := m.bindingRepo.GetByID(ctx, bindingID)
	if err != nil {
		return nil, err
	}
	if binding.RouterID != routerID {
		return nil, domain.ErrDHCPBindingNotFound
	}
	if req.CustomerID != nil {
		binding.CustomerID = strings.TrimSpace(*req.CustomerID)
	}
	if req.Server != nil {
		binding.Server = strings.TrimSpace(*req.Server)
		if binding.Server == "" {
			binding.Server = "all"
		}
	}
	if req.MACAddress != nil {
		binding.MACAddress = normalizeMAC(*req.MACAddress)
	}
	if req.IPAddress != nil {
		binding.IPAddress = strings.TrimSpace(*req.IPAddress)
	}
	if err := validateDHCPBinding(binding.MACAddress, binding.IPAddress); err != nil {
		return nil, err
	}
	if req.HostName != nil {
		binding.HostName = strings.TrimSpace(*req.HostName)
	}
	if req.Comment != nil {
		binding.Comment = strings.TrimSpace(*req.Comment)
	}
	if req.Disabled != nil {
		binding.Disabled = *req.Disabled
		binding.Status = statusFromDisabled(*req.Disabled)
	}
	binding.SyncStatus = "pending_update"
	updated, err := m.bindingRepo.Update(ctx, binding)
	if err != nil {
		return nil, err
	}
	if err := m.syncBindingToRouter(ctx, updated); err != nil {
		_ = m.bindingRepo.UpdateSyncState(ctx, updated.ID, updated.RouterLeaseID, "error", nil)
		m.audit(ctx, updated, "dhcp.binding.update", "/ip/dhcp-server/lease/set", "failed", err)
		return nil, err
	}
	m.audit(ctx, updated, "dhcp.binding.update", "/ip/dhcp-server/lease/set", "success", nil)
	refreshed, err := m.bindingRepo.GetByID(ctx, updated.ID)
	if err != nil {
		return updated.ToResponse(), nil
	}
	return refreshed.ToResponse(), nil
}

func (m *dhcpManager) DeleteBinding(ctx context.Context, routerID, bindingID string, req domain.DeleteDHCPBindingRequest) error {
	binding, err := m.bindingRepo.GetByID(ctx, bindingID)
	if err != nil {
		return err
	}
	if binding.RouterID != routerID {
		return domain.ErrDHCPBindingNotFound
	}
	if normalizeMAC(req.ConfirmMAC) != normalizeMAC(binding.MACAddress) {
		return domain.ErrConfirmationMismatch
	}
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return err
	}
	defer closeFn()
	cmdBuilder, err := commandBuilderForRouter(ctx, m.routerRepo, routerID)
	if err != nil {
		return err
	}
	if leaseID, _ := findDHCPLeaseID(ctx, adapter, binding); leaseID != "" {
		cmd, args := cmdBuilder.RemoveDHCPLease(leaseID)
		if _, err := adapter.Execute(ctx, cmd, args); err != nil {
			m.audit(ctx, binding, "dhcp.binding.delete", cmd, "failed", err)
			return err
		}
	}
	m.audit(ctx, binding, "dhcp.binding.delete", "/ip/dhcp-server/lease/remove", "success", nil)
	return m.bindingRepo.SoftDelete(ctx, bindingID)
}

func (m *dhcpManager) audit(ctx context.Context, binding *domain.DHCPBinding, action, command, status string, err error) {
	if m.auditRepo == nil || binding == nil {
		return
	}
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_ = m.auditRepo.Create(ctx, domain.MikroTikCommandAuditLog{
		TenantID:     binding.TenantID,
		RouterID:     binding.RouterID,
		UserID:       actor.UserID,
		Action:       action,
		Command:      command,
		TargetType:   "dhcp_binding",
		TargetID:     binding.ID,
		Status:       status,
		ErrorMessage: msg,
		RemoteAddr:   actor.RemoteAddr,
	})
}

func (m *dhcpManager) syncBindingToRouter(ctx context.Context, binding *domain.DHCPBinding) error {
	adapter, closeFn, err := m.connect(ctx, binding.RouterID)
	if err != nil {
		return err
	}
	defer closeFn()
	leaseID, err := findDHCPLeaseID(ctx, adapter, binding)
	if err != nil {
		return err
	}
	cmdBuilder, err := commandBuilderForRouter(ctx, m.routerRepo, binding.RouterID)
	if err != nil {
		return err
	}
	params := map[string]string{
		"=mac-address": binding.MACAddress,
		"=address":     binding.IPAddress,
		"=comment":     managedDHCPComment(binding),
		"=disabled":    routerOSBool(binding.Disabled),
	}
	if binding.Server != "" && binding.Server != "all" {
		params["=server"] = binding.Server
	}
	if leaseID != "" {
		params["=numbers"] = leaseID
		cmd, args := cmdBuilder.SetDHCPLease(params)
		if _, err := adapter.Execute(ctx, cmd, args); err != nil {
			return err
		}
	} else {
		cmd, args := cmdBuilder.AddDHCPLease(params)
		if _, err := adapter.Execute(ctx, cmd, args); err != nil {
			return err
		}
		leaseID, _ = findDHCPLeaseID(ctx, adapter, binding)
	}
	now := time.Now()
	return m.bindingRepo.UpdateSyncState(ctx, binding.ID, leaseID, "synced", &now)
}

func (m *dhcpManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
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
		Host:           router.Host,
		Port:           router.Port,
		Username:       router.Username,
		Password:       password,
		UseSSL:         router.UseSSL,
		ConnectTimeout: 10 * time.Second,
		CommandTimeout: 10 * time.Second,
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

func findDHCPLeaseID(ctx context.Context, adapter domain.RouterOSAdapter, binding *domain.DHCPBinding) (string, error) {
	rows, err := adapter.Execute(ctx, "/ip/dhcp-server/lease/print", map[string]string{
		"=.proplist": ".id,mac-address,address,comment",
	})
	if err != nil {
		return "", err
	}
	mac := normalizeMAC(binding.MACAddress)
	for _, row := range rows {
		comment := row["comment"]
		if row[".id"] == binding.RouterLeaseID && row[".id"] != "" {
			return row[".id"], nil
		}
		if normalizeMAC(row["mac-address"]) == mac {
			return row[".id"], nil
		}
		if strings.HasPrefix(comment, dhcpCommentPrefix+binding.ID) {
			return row[".id"], nil
		}
	}
	return "", nil
}

func parseDHCPServers(rows []map[string]string) []domain.DHCPServer {
	items := make([]domain.DHCPServer, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.DHCPServer{
			ID:            row[".id"],
			Name:          row["name"],
			Interface:     row["interface"],
			AddressPool:   row["address-pool"],
			LeaseTime:     row["lease-time"],
			Authoritative: row["authoritative"],
			Disabled:      parseRouterOSBool(row["disabled"]),
			Comment:       row["comment"],
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func parseDHCPLeases(rows []map[string]string) []domain.DHCPLease {
	items := make([]domain.DHCPLease, 0, len(rows))
	for _, row := range rows {
		comment := row["comment"]
		items = append(items, domain.DHCPLease{
			ID:           row[".id"],
			Server:       row["server"],
			Address:      row["address"],
			MACAddress:   normalizeMAC(row["mac-address"]),
			HostName:     row["host-name"],
			ClientID:     row["client-id"],
			Status:       row["status"],
			Dynamic:      parseRouterOSBool(row["dynamic"]),
			Disabled:     parseRouterOSBool(row["disabled"]),
			ExpiresAfter: row["expires-after"],
			LastSeen:     row["last-seen"],
			Comment:      comment,
			Managed:      strings.HasPrefix(comment, dhcpCommentPrefix),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].MACAddress < items[j].MACAddress })
	return items
}

func parseDHCPNetworks(rows []map[string]string) []domain.DHCPNetwork {
	items := make([]domain.DHCPNetwork, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.DHCPNetwork{
			ID:        row[".id"],
			Address:   row["address"],
			Gateway:   row["gateway"],
			DNSServer: splitCSV(row["dns-server"]),
			Domain:    row["domain"],
			Comment:   row["comment"],
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Address < items[j].Address })
	return items
}

func validateDHCPBinding(macAddress, ipAddress string) error {
	if _, err := net.ParseMAC(macAddress); err != nil {
		return domain.ErrInvalidMACAddress
	}
	if net.ParseIP(ipAddress) == nil {
		return domain.ErrInvalidIPAddress
	}
	return nil
}

func normalizeMAC(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mac, err := net.ParseMAC(value)
	if err != nil {
		return strings.ToUpper(value)
	}
	return strings.ToUpper(mac.String())
}

func managedDHCPComment(binding *domain.DHCPBinding) string {
	comment := strings.TrimSpace(binding.Comment)
	if comment == "" {
		return dhcpCommentPrefix + binding.ID
	}
	return fmt.Sprintf("%s%s %s", dhcpCommentPrefix, binding.ID, comment)
}

func routerOSBool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func statusFromDisabled(disabled bool) string {
	if disabled {
		return "disabled"
	}
	return "active"
}
