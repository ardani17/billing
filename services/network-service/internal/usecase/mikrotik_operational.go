package usecase

import (
	"context"
	"errors"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type MikroTikOperationalManager interface {
	ListInterfaces(ctx context.Context, routerID string) ([]domain.RouterInterface, error)
	GetTraffic(ctx context.Context, routerID string, interfaces []string) ([]domain.RouterTrafficSample, error)
	ListIPPools(ctx context.Context, routerID string) ([]domain.RouterIPPoolUsage, error)
	ListManagedFirewall(ctx context.Context, routerID string) ([]domain.RouterFirewallRule, error)
	ListLogs(ctx context.Context, routerID string, filter domain.RouterLogFilter) ([]domain.RouterLogEntry, error)
}

type mikrotikOperationalManager struct {
	routerRepo     domain.RouterRepository
	encryptor      domain.CredentialEncryptor
	adapterFactory AdapterFactory
}

func NewMikroTikOperationalManager(
	routerRepo domain.RouterRepository,
	encryptor domain.CredentialEncryptor,
	adapterFactory AdapterFactory,
) MikroTikOperationalManager {
	return &mikrotikOperationalManager{
		routerRepo:     routerRepo,
		encryptor:      encryptor,
		adapterFactory: adapterFactory,
	}
}

func (m *mikrotikOperationalManager) ListInterfaces(ctx context.Context, routerID string) ([]domain.RouterInterface, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	rows, err := adapter.Execute(ctx, "/interface/print", map[string]string{
		"=.proplist": ".id,name,type,mtu,mac-address,running,disabled,rx-byte,tx-byte,rx-packet,tx-packet,comment",
	})
	if err != nil {
		return nil, err
	}
	return parseRouterInterfaces(rows), nil
}

func (m *mikrotikOperationalManager) GetTraffic(ctx context.Context, routerID string, interfaces []string) ([]domain.RouterTrafficSample, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	if len(interfaces) == 0 {
		rows, listErr := adapter.Execute(ctx, "/interface/print", map[string]string{
			"=.proplist": "name,running,disabled",
		})
		if listErr != nil {
			return nil, listErr
		}
		for _, row := range rows {
			if parseRouterOSBool(row["running"]) && !parseRouterOSBool(row["disabled"]) {
				interfaces = append(interfaces, row["name"])
			}
		}
	}

	samples := make([]domain.RouterTrafficSample, 0, len(interfaces))
	for _, name := range interfaces {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		rows, trafficErr := adapter.Execute(ctx, "/interface/monitor-traffic", map[string]string{
			"=interface": name,
			"=once":      "",
		})
		if trafficErr != nil {
			return nil, trafficErr
		}
		if len(rows) == 0 {
			continue
		}
		samples = append(samples, parseTrafficSample(name, rows[0]))
	}
	return samples, nil
}

func (m *mikrotikOperationalManager) ListIPPools(ctx context.Context, routerID string) ([]domain.RouterIPPoolUsage, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	poolRows, err := adapter.Execute(ctx, "/ip/pool/print", map[string]string{
		"=.proplist": ".id,name,ranges",
	})
	if err != nil {
		return nil, err
	}
	usedRows, err := adapter.Execute(ctx, "/ip/pool/used/print", map[string]string{
		"=.proplist": "pool,address",
	})
	if err != nil {
		log.Warn().Err(err).Str("router_id", routerID).Msg("gagal membaca pool used; lanjutkan dengan used=0")
		usedRows = nil
	}
	return parsePoolUsage(poolRows, usedRows), nil
}

func (m *mikrotikOperationalManager) ListManagedFirewall(ctx context.Context, routerID string) ([]domain.RouterFirewallRule, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	rules := make([]domain.RouterFirewallRule, 0)
	for _, source := range []struct {
		kind    string
		command string
	}{
		{"nat", "/ip/firewall/nat/print"},
		{"filter", "/ip/firewall/filter/print"},
		{"address_list", "/ip/firewall/address-list/print"},
	} {
		rows, readErr := adapter.Execute(ctx, source.command, map[string]string{
			"=.proplist": ".id,chain,action,list,address,disabled,comment",
		})
		if readErr != nil {
			return nil, readErr
		}
		rules = append(rules, parseManagedFirewallRows(source.kind, rows)...)
	}
	return rules, nil
}

func (m *mikrotikOperationalManager) ListLogs(ctx context.Context, routerID string, filter domain.RouterLogFilter) ([]domain.RouterLogEntry, error) {
	adapter, closeFn, err := m.connect(ctx, routerID)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 300 {
		limit = 300
	}
	rows, err := adapter.Execute(ctx, "/log/print", map[string]string{
		"=.proplist": ".id,time,topics,message",
	})
	if err != nil {
		return nil, err
	}
	return parseLogs(rows, filter.Topic, filter.Search, limit), nil
}

func (m *mikrotikOperationalManager) connect(ctx context.Context, routerID string) (domain.RouterOSAdapter, func(), error) {
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

func parseRouterInterfaces(rows []map[string]string) []domain.RouterInterface {
	items := make([]domain.RouterInterface, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.RouterInterface{
			ID:       row[".id"],
			Name:     row["name"],
			Type:     row["type"],
			MTU:      parseRouterOSInt(row["mtu"]),
			MAC:      row["mac-address"],
			Running:  parseRouterOSBool(row["running"]),
			Disabled: parseRouterOSBool(row["disabled"]),
			RXByte:   parseRouterOSInt64(row["rx-byte"]),
			TXByte:   parseRouterOSInt64(row["tx-byte"]),
			RXPacket: parseRouterOSInt64(row["rx-packet"]),
			TXPacket: parseRouterOSInt64(row["tx-packet"]),
			Comment:  row["comment"],
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func parseTrafficSample(name string, row map[string]string) domain.RouterTrafficSample {
	if value := row["name"]; value != "" {
		name = value
	}
	return domain.RouterTrafficSample{
		Interface: name,
		RXBps:     parseRouterOSInt64(row["rx-bits-per-second"]),
		TXBps:     parseRouterOSInt64(row["tx-bits-per-second"]),
		RXPackets: parseRouterOSInt64(row["rx-packets-per-second"]),
		TXPackets: parseRouterOSInt64(row["tx-packets-per-second"]),
	}
}

func parsePoolUsage(poolRows, usedRows []map[string]string) []domain.RouterIPPoolUsage {
	usedByPool := map[string]int{}
	for _, row := range usedRows {
		if pool := row["pool"]; pool != "" {
			usedByPool[pool]++
		}
	}
	items := make([]domain.RouterIPPoolUsage, 0, len(poolRows))
	for _, row := range poolRows {
		ranges := splitCSV(row["ranges"])
		total := 0
		for _, item := range ranges {
			total += countIPRange(item)
		}
		used := usedByPool[row["name"]]
		available := total - used
		if available < 0 {
			available = 0
		}
		percent := 0
		if total > 0 {
			percent = int(float64(used) / float64(total) * 100)
		}
		items = append(items, domain.RouterIPPoolUsage{
			Name:         row["name"],
			Ranges:       ranges,
			Used:         used,
			Total:        total,
			Available:    available,
			UsagePercent: percent,
			WarningLevel: poolWarning(percent),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func parseManagedFirewallRows(kind string, rows []map[string]string) []domain.RouterFirewallRule {
	rules := make([]domain.RouterFirewallRule, 0)
	for _, row := range rows {
		comment := row["comment"]
		list := row["list"]
		if !strings.HasPrefix(comment, "ISPBoss:") && !strings.HasPrefix(list, "ISPBoss:") &&
			!strings.HasPrefix(list, "walled-garden") && !strings.HasPrefix(list, "isolated") &&
			!strings.HasPrefix(list, "active-") {
			continue
		}
		rules = append(rules, domain.RouterFirewallRule{
			ID:       row[".id"],
			Kind:     kind,
			Chain:    row["chain"],
			Action:   row["action"],
			List:     list,
			Address:  row["address"],
			Disabled: parseRouterOSBool(row["disabled"]),
			Comment:  comment,
		})
	}
	return rules
}

func parseLogs(rows []map[string]string, topic, search string, limit int) []domain.RouterLogEntry {
	topic = strings.ToLower(strings.TrimSpace(topic))
	search = strings.ToLower(strings.TrimSpace(search))
	items := make([]domain.RouterLogEntry, 0, len(rows))
	for _, row := range rows {
		topics := row["topics"]
		message := row["message"]
		if topic != "" && !strings.Contains(strings.ToLower(topics), topic) {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(message+" "+topics), search) {
			continue
		}
		items = append(items, domain.RouterLogEntry{
			ID:      row[".id"],
			Time:    row["time"],
			Topics:  topics,
			Message: message,
		})
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			items = append(items, part)
		}
	}
	return items
}

func countIPRange(value string) int {
	parts := strings.Split(value, "-")
	if len(parts) == 1 {
		if net.ParseIP(strings.TrimSpace(parts[0])) == nil {
			return 0
		}
		return 1
	}
	start := ipToUint32(net.ParseIP(strings.TrimSpace(parts[0])).To4())
	end := ipToUint32(net.ParseIP(strings.TrimSpace(parts[1])).To4())
	if start == 0 || end == 0 || end < start {
		return 0
	}
	return int(end-start) + 1
}

func ipToUint32(ip net.IP) uint32 {
	if len(ip) != 4 {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func poolWarning(percent int) string {
	switch {
	case percent >= 90:
		return "critical"
	case percent >= 80:
		return "warning"
	default:
		return "normal"
	}
}

func parseRouterOSBool(value string) bool {
	return value == "true" || value == "yes"
}

func parseRouterOSInt(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}

func parseRouterOSInt64(value string) int64 {
	n, _ := strconv.ParseInt(value, 10, 64)
	return n
}
