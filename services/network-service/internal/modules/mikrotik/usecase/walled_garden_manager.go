package usecase

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const (
	walledGardenIsolatedList = "ISPBoss:walled-garden-isolated"
	walledGardenAllowedList  = "ISPBoss:walled-garden-allowed"
	walledGardenComment      = "ISPBoss:walled-garden:"
)

type WalledGardenManager interface {
	GetStatus(ctx context.Context, routerID string) (*domain.WalledGardenStatus, error)
	Apply(ctx context.Context, routerID string, req domain.ApplyWalledGardenRequest) (*domain.WalledGardenStatus, error)
	Remove(ctx context.Context, routerID string) (*domain.WalledGardenStatus, error)
}

type walledGardenManager struct {
	routerRepo     domain.RouterRepository
	auditRepo      domain.MikroTikCommandAuditRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewWalledGardenManager(
	routerRepo domain.RouterRepository,
	auditRepo domain.MikroTikCommandAuditRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) WalledGardenManager {
	return &walledGardenManager{
		routerRepo:     routerRepo,
		auditRepo:      auditRepo,
		encryptor:      encryptor,
		adapterFactory: adapterFactory,
	}
}

func (m *walledGardenManager) GetStatus(ctx context.Context, routerID string) (*domain.WalledGardenStatus, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()
	return readWalledGardenStatus(ctx, adapter, defaultWalledGardenConfig()), nil
}

func (m *walledGardenManager) Apply(ctx context.Context, routerID string, req domain.ApplyWalledGardenRequest) (*domain.WalledGardenStatus, error) {
	cfg := normalizeWalledGardenConfig(req)
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.apply", "connect", "failed", err)
		return nil, err
	}
	defer closeFn()
	cmdBuilder, err := commandBuilderForRouter(ctx, m.routerRepo, routerID)
	if err != nil {
		return nil, err
	}

	if err := ensureAllowedDestinations(ctx, adapter, cmdBuilder, cfg); err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.apply", "/ip/firewall/address-list", "failed", err)
		return nil, err
	}
	if err := pruneWalledGardenRules(ctx, adapter, cmdBuilder, cfg); err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.apply", "/ip/firewall/prune", "failed", err)
		return nil, err
	}
	if err := ensureWalledGardenRules(ctx, adapter, cmdBuilder, cfg); err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.apply", "/ip/firewall", "failed", err)
		return nil, err
	}
	m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.apply", "/ip/firewall", "success", nil)
	return readWalledGardenStatus(ctx, adapter, cfg), nil
}

func (m *walledGardenManager) Remove(ctx context.Context, routerID string) (*domain.WalledGardenStatus, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.remove", "connect", "failed", err)
		return nil, err
	}
	defer closeFn()
	cmdBuilder, err := commandBuilderForRouter(ctx, m.routerRepo, routerID)
	if err != nil {
		return nil, err
	}

	if err := removeWalledGardenManaged(ctx, adapter, cmdBuilder); err != nil {
		m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.remove", "/ip/firewall", "failed", err)
		return nil, err
	}
	m.audit(ctx, tenant.FromContext(ctx), routerID, "walled_garden.remove", "/ip/firewall", "success", nil)
	return readWalledGardenStatus(ctx, adapter, defaultWalledGardenConfig()), nil
}

func (m *walledGardenManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
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

func defaultWalledGardenConfig() domain.WalledGardenConfig {
	return domain.WalledGardenConfig{
		Method:              domain.WalledGardenMethodDNSRedirect,
		WalledGardenIP:      "10.255.255.1",
		DNSServerIP:         "10.255.255.1",
		IsolatedAddressList: walledGardenIsolatedList,
		AllowedAddressList:  walledGardenAllowedList,
		AllowedDestinations: []string{"payment.ispboss.local"},
	}
}

func normalizeWalledGardenConfig(req domain.ApplyWalledGardenRequest) domain.WalledGardenConfig {
	cfg := defaultWalledGardenConfig()
	if value := strings.TrimSpace(req.Method); value != "" {
		cfg.Method = value
	}
	if value := strings.TrimSpace(req.WalledGardenIP); value != "" {
		cfg.WalledGardenIP = value
	}
	if value := strings.TrimSpace(req.DNSServerIP); value != "" {
		cfg.DNSServerIP = value
	}
	if value := strings.TrimSpace(req.IsolatedAddressList); value != "" {
		cfg.IsolatedAddressList = value
	}
	if value := strings.TrimSpace(req.AllowedAddressList); value != "" {
		cfg.AllowedAddressList = value
	}
	if len(req.AllowedDestinations) > 0 {
		cfg.AllowedDestinations = normalizeDestinations(req.AllowedDestinations)
	}
	return cfg
}

func normalizeDestinations(items []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func ensureAllowedDestinations(ctx context.Context, adapter domain.RouterOSAdapter, cmdBuilder domain.CommandBuilder, cfg domain.WalledGardenConfig) error {
	rows, err := adapter.Execute(ctx, "/ip/firewall/address-list/print", map[string]string{
		"=.proplist": ".id,list,address,comment,disabled",
	})
	if err != nil {
		return err
	}
	for _, destination := range cfg.AllowedDestinations {
		if destination == "" {
			continue
		}
		params := map[string]string{
			"=list":    cfg.AllowedAddressList,
			"=address": destination,
			"=comment": walledGardenComment + "allow:" + destination,
		}
		if id := findAddressListID(rows, cfg.AllowedAddressList, destination); id != "" {
			params["=numbers"] = id
			cmd, args := cmdBuilder.SetAddressList(params)
			if _, err := adapter.Execute(ctx, cmd, args); err != nil {
				return err
			}
		} else {
			cmd, args := cmdBuilder.AddAddressList(params)
			if _, err := adapter.Execute(ctx, cmd, args); err != nil {
				return err
			}
		}
		if ok, err := addressListExists(ctx, adapter, cfg.AllowedAddressList, destination); err != nil || !ok {
			if err != nil {
				return err
			}
			return fmt.Errorf("address-list %s:%s tidak tersimpan di router", cfg.AllowedAddressList, destination)
		}
	}
	return nil
}

func ensureWalledGardenRules(ctx context.Context, adapter domain.RouterOSAdapter, cmdBuilder domain.CommandBuilder, cfg domain.WalledGardenConfig) error {
	rules := rulesForWalledGarden(cfg)
	for _, rule := range rules {
		if err := upsertFirewallRule(ctx, adapter, cmdBuilder, rule); err != nil {
			return err
		}
	}
	return nil
}

func pruneWalledGardenRules(ctx context.Context, adapter domain.RouterOSAdapter, cmdBuilder domain.CommandBuilder, cfg domain.WalledGardenConfig) error {
	keep := map[string]bool{}
	for _, rule := range rulesForWalledGarden(cfg) {
		keep[rule.Comment] = true
	}
	for _, target := range []string{"/ip/firewall/nat", "/ip/firewall/filter"} {
		rows, err := adapter.Execute(ctx, target+"/print", map[string]string{
			"=.proplist": ".id,comment",
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			comment := row["comment"]
			if strings.HasPrefix(comment, walledGardenComment) && !keep[comment] {
				kind := strings.TrimPrefix(target, "/ip/firewall/")
				cmd, args := cmdBuilder.RemoveFirewallRule(kind, row[".id"])
				if _, err := adapter.Execute(ctx, cmd, args); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type firewallRuleSpec struct {
	Kind    string
	Comment string
	Params  map[string]string
}

func rulesForWalledGarden(cfg domain.WalledGardenConfig) []firewallRuleSpec {
	baseSrc := map[string]string{"=src-address-list": cfg.IsolatedAddressList}
	switch cfg.Method {
	case domain.WalledGardenMethodHTTPRedirect:
		return []firewallRuleSpec{{
			Kind:    "nat",
			Comment: walledGardenComment + "http",
			Params: mergeParams(baseSrc, map[string]string{
				"=chain": "dstnat", "=protocol": "tcp", "=dst-port": "80",
				"=action": "dst-nat", "=to-addresses": cfg.WalledGardenIP, "=to-ports": "80",
			}),
		}}
	case domain.WalledGardenMethodBlockAllWhitelist:
		return []firewallRuleSpec{
			{
				Kind:    "filter",
				Comment: walledGardenComment + "allow",
				Params: mergeParams(baseSrc, map[string]string{
					"=chain": "forward", "=dst-address-list": cfg.AllowedAddressList, "=action": "accept",
				}),
			},
			{
				Kind:    "filter",
				Comment: walledGardenComment + "drop",
				Params: mergeParams(baseSrc, map[string]string{
					"=chain": "forward", "=action": "drop",
				}),
			},
		}
	default:
		return []firewallRuleSpec{
			{
				Kind:    "nat",
				Comment: walledGardenComment + "dns-udp",
				Params: mergeParams(baseSrc, map[string]string{
					"=chain": "dstnat", "=protocol": "udp", "=dst-port": "53",
					"=action": "dst-nat", "=to-addresses": cfg.DNSServerIP, "=to-ports": "53",
				}),
			},
			{
				Kind:    "nat",
				Comment: walledGardenComment + "dns-tcp",
				Params: mergeParams(baseSrc, map[string]string{
					"=chain": "dstnat", "=protocol": "tcp", "=dst-port": "53",
					"=action": "dst-nat", "=to-addresses": cfg.DNSServerIP, "=to-ports": "53",
				}),
			},
		}
	}
}

func mergeParams(a, b map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range a {
		result[key] = value
	}
	for key, value := range b {
		result[key] = value
	}
	return result
}

func upsertFirewallRule(ctx context.Context, adapter domain.RouterOSAdapter, cmdBuilder domain.CommandBuilder, spec firewallRuleSpec) error {
	commandPrefix := "/ip/firewall/" + spec.Kind
	rows, err := adapter.Execute(ctx, commandPrefix+"/print", map[string]string{
		"=.proplist": ".id,comment",
	})
	if err != nil {
		return err
	}
	params := map[string]string{"=comment": spec.Comment}
	for key, value := range spec.Params {
		params[key] = value
	}
	if id := findCommentID(rows, spec.Comment); id != "" {
		params["=numbers"] = id
		cmd, args := cmdBuilder.SetFirewallRule(spec.Kind, params)
		if _, err = adapter.Execute(ctx, cmd, args); err != nil {
			return err
		}
	} else {
		cmd, args := cmdBuilder.AddFirewallRule(spec.Kind, params)
		if _, err = adapter.Execute(ctx, cmd, args); err != nil {
			return err
		}
	}
	if ok, err := firewallRuleExists(ctx, adapter, commandPrefix, spec.Comment); err != nil || !ok {
		if err != nil {
			return err
		}
		return fmt.Errorf("firewall rule %s tidak tersimpan di router", spec.Comment)
	}
	return nil
}

func readWalledGardenStatus(ctx context.Context, adapter domain.RouterOSAdapter, cfg domain.WalledGardenConfig) *domain.WalledGardenStatus {
	rules := make([]domain.RouterFirewallRule, 0)
	isolatedCount := 0
	allowedCount := 0
	for _, source := range []struct {
		kind     string
		command  string
		proplist string
	}{
		{"nat", "/ip/firewall/nat/print", ".id,chain,action,disabled,comment"},
		{"filter", "/ip/firewall/filter/print", ".id,chain,action,disabled,comment"},
		{"address_list", "/ip/firewall/address-list/print", ".id,list,address,disabled,comment"},
	} {
		rows, err := adapter.Execute(ctx, source.command, map[string]string{
			"=.proplist": source.proplist,
		})
		if err != nil {
			continue
		}
		for _, row := range rows {
			comment := row["comment"]
			list := row["list"]
			if source.kind == "address_list" {
				if list == cfg.IsolatedAddressList {
					isolatedCount++
				}
				if list == cfg.AllowedAddressList {
					allowedCount++
				}
			}
			if !strings.HasPrefix(comment, walledGardenComment) && list != cfg.IsolatedAddressList && list != cfg.AllowedAddressList {
				continue
			}
			rules = append(rules, domain.RouterFirewallRule{
				ID: row[".id"], Kind: source.kind, Chain: row["chain"], Action: row["action"],
				List: list, Address: row["address"], Disabled: parseRouterOSBool(row["disabled"]), Comment: comment,
			})
		}
	}
	return &domain.WalledGardenStatus{
		Config: cfg, Rules: rules, IsolatedCount: isolatedCount,
		AllowedCount: allowedCount, Applied: hasWalledGardenRule(rules),
	}
}

func removeWalledGardenManaged(ctx context.Context, adapter domain.RouterOSAdapter, cmdBuilder domain.CommandBuilder) error {
	for _, target := range []struct {
		kind    string
		command string
	}{
		{"nat", "/ip/firewall/nat"},
		{"filter", "/ip/firewall/filter"},
		{"address_list", "/ip/firewall/address-list"},
	} {
		rows, err := adapter.Execute(ctx, target.command+"/print", map[string]string{
			"=.proplist": ".id,comment",
		})
		if err != nil {
			return err
		}
		for _, row := range rows {
			comment := row["comment"]
			if !strings.HasPrefix(comment, walledGardenComment) {
				continue
			}
			var cmd string
			var args map[string]string
			if target.kind == "address_list" {
				cmd, args = cmdBuilder.RemoveAddressList(row[".id"])
			} else {
				cmd, args = cmdBuilder.RemoveFirewallRule(target.kind, row[".id"])
			}
			if _, err := adapter.Execute(ctx, cmd, args); err != nil {
				return err
			}
		}
	}
	return nil
}

func findCommentID(rows []map[string]string, comment string) string {
	for _, row := range rows {
		if row["comment"] == comment {
			return row[".id"]
		}
	}
	return ""
}

func findAddressListID(rows []map[string]string, list, address string) string {
	for _, row := range rows {
		if row["list"] == list && row["address"] == address {
			return row[".id"]
		}
	}
	return ""
}

func addressListExists(ctx context.Context, adapter domain.RouterOSAdapter, list, address string) (bool, error) {
	rows, err := adapter.Execute(ctx, "/ip/firewall/address-list/print", map[string]string{
		"=.proplist": ".id,list,address",
	})
	if err != nil {
		return false, err
	}
	return findAddressListID(rows, list, address) != "", nil
}

func firewallRuleExists(ctx context.Context, adapter domain.RouterOSAdapter, commandPrefix, comment string) (bool, error) {
	rows, err := adapter.Execute(ctx, commandPrefix+"/print", map[string]string{
		"=.proplist": ".id,comment",
	})
	if err != nil {
		return false, err
	}
	return findCommentID(rows, comment) != "", nil
}

func hasWalledGardenRule(rules []domain.RouterFirewallRule) bool {
	for _, rule := range rules {
		if strings.HasPrefix(rule.Comment, walledGardenComment) && rule.Kind != "address_list" {
			return true
		}
	}
	return false
}

func (m *walledGardenManager) audit(ctx context.Context, tenantID, routerID, action, command, status string, err error) {
	if m.auditRepo == nil {
		return
	}
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_ = m.auditRepo.Create(ctx, domain.MikroTikCommandAuditLog{
		TenantID: tenantID, RouterID: routerID, UserID: actor.UserID,
		Action: action, Command: command, TargetType: "walled_garden",
		Status: status, ErrorMessage: msg, RemoteAddr: actor.RemoteAddr,
	})
}

func validWalledGardenMethod(method string) bool {
	switch method {
	case domain.WalledGardenMethodDNSRedirect, domain.WalledGardenMethodHTTPRedirect, domain.WalledGardenMethodBlockAllWhitelist:
		return true
	default:
		return false
	}
}

func validOptionalIP(value string) bool {
	return strings.TrimSpace(value) == "" || net.ParseIP(value) != nil
}
