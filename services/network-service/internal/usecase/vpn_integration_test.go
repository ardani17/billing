// vpn_integration_test.go — integration test untuk VPN Manager menggunakan mock.
package usecase

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// --- Mock: in-memory VPN tunnel repository ---
type vpnTunnelRepoMock struct {
	tunnels map[string]*domain.VPNTunnel; nextID int
}

func newVPNTunnelRepo() *vpnTunnelRepoMock { return &vpnTunnelRepoMock{tunnels: map[string]*domain.VPNTunnel{}} }
func (r *vpnTunnelRepoMock) Create(_ context.Context, t *domain.VPNTunnel) (*domain.VPNTunnel, error) {
	r.nextID++; t.ID = fmt.Sprintf("tun-%d", r.nextID); t.CreatedAt = time.Now(); t.UpdatedAt = t.CreatedAt; r.tunnels[t.ID] = t; return t, nil
}
func (r *vpnTunnelRepoMock) GetByID(_ context.Context, id string) (*domain.VPNTunnel, error) {
	if t, ok := r.tunnels[id]; ok { return t, nil }; return nil, domain.ErrVPNTunnelNotFound
}
func (r *vpnTunnelRepoMock) Update(_ context.Context, t *domain.VPNTunnel) (*domain.VPNTunnel, error) { r.tunnels[t.ID] = t; return t, nil }
func (r *vpnTunnelRepoMock) SoftDelete(_ context.Context, id string) error {
	if t, ok := r.tunnels[id]; ok { now := time.Now(); t.DeletedAt = &now; return nil }; return domain.ErrVPNTunnelNotFound
}
func (r *vpnTunnelRepoMock) List(_ context.Context, p domain.VPNTunnelListParams) (*domain.VPNTunnelListResult, error) {
	var d []*domain.VPNTunnelResponse
	for _, t := range r.tunnels {
		if t.DeletedAt == nil { d = append(d, &domain.VPNTunnelResponse{ID: t.ID, TunnelName: t.TunnelName, Protocol: t.Protocol, VPNIP: t.VPNIP, Status: t.Status}) }
	}
	s, e := (p.Page-1)*p.PageSize, (p.Page-1)*p.PageSize+p.PageSize
	if s > len(d) { s = len(d) }; if e > len(d) { e = len(d) }
	return &domain.VPNTunnelListResult{Data: d[s:e], Total: int64(len(d)), Page: p.Page, PageSize: p.PageSize}, nil
}
func (r *vpnTunnelRepoMock) GetByStatus(_ context.Context, s domain.TunnelStatus) ([]*domain.VPNTunnel, error) {
	var res []*domain.VPNTunnel; for _, t := range r.tunnels { if t.Status == s && t.DeletedAt == nil { res = append(res, t) } }; return res, nil
}
func (r *vpnTunnelRepoMock) CountByStatus(_ context.Context) (map[domain.TunnelStatus]int64, error) {
	c := map[domain.TunnelStatus]int64{}; for _, t := range r.tunnels { if t.DeletedAt == nil { c[t.Status]++ } }; return c, nil
}
func (r *vpnTunnelRepoMock) TunnelNameExists(context.Context, string, string, string) (bool, error) { return false, nil }
func (r *vpnTunnelRepoMock) VPNIPExists(context.Context, string, string) (bool, error)              { return false, nil }
func (r *vpnTunnelRepoMock) UpdateStatus(_ context.Context, id string, p domain.TunnelHealthUpdate) error {
	t, ok := r.tunnels[id]; if !ok { return domain.ErrVPNTunnelNotFound }
	if p.Status != nil { t.Status = *p.Status }; if p.LatencyMs != nil { t.LatencyMs = p.LatencyMs }; return nil
}
func (r *vpnTunnelRepoMock) GetConnectedTunnels(c context.Context) ([]*domain.VPNTunnel, error)    { return r.GetByStatus(c, domain.TunnelStatusConnected) }
func (r *vpnTunnelRepoMock) GetDisconnectedTunnels(c context.Context) ([]*domain.VPNTunnel, error) { return r.GetByStatus(c, domain.TunnelStatusDisconnected) }

// --- Mock: in-memory VPN subnet repository ---
type vpnSubnetRepoMock struct{ subnets map[string]*domain.VPNSubnet; nextSeq int }

func newVPNSubnetRepo() *vpnSubnetRepoMock { return &vpnSubnetRepoMock{subnets: map[string]*domain.VPNSubnet{}, nextSeq: 1} }
func (r *vpnSubnetRepoMock) GetByTenantID(_ context.Context, tid string) (*domain.VPNSubnet, error) {
	if s, ok := r.subnets[tid]; ok { return s, nil }; return nil, fmt.Errorf("not found")
}
func (r *vpnSubnetRepoMock) Create(_ context.Context, s *domain.VPNSubnet) (*domain.VPNSubnet, error) { r.subnets[s.TenantID] = s; return s, nil }
func (r *vpnSubnetRepoMock) GetNextTenantSeq(context.Context) (int, error) { seq := r.nextSeq; r.nextSeq++; return seq, nil }
func (r *vpnSubnetRepoMock) IncrementNextClientIPSeq(_ context.Context, tid string) (int, error) {
	s, ok := r.subnets[tid]; if !ok { return 0, fmt.Errorf("not found") }; seq := s.NextClientIPSeq; s.NextClientIPSeq++; return seq, nil
}

// --- Mock: encryptor (base64), event publisher, bandwidth store, command builder ---
type vpnEncMock struct{}
func (*vpnEncMock) Encrypt(p string) (string, error) { return base64.StdEncoding.EncodeToString([]byte(p)), nil }
func (*vpnEncMock) Decrypt(c string) (string, error) { b, e := base64.StdEncoding.DecodeString(c); return string(b), e }

type vpnEvtMock struct{ created []domain.VPNTunnelCreatedPayload }
func (p *vpnEvtMock) PublishTunnelDown(context.Context, domain.VPNTunnelDownPayload) error     { return nil }
func (p *vpnEvtMock) PublishTunnelUp(context.Context, domain.VPNTunnelUpPayload) error         { return nil }
func (p *vpnEvtMock) PublishTunnelCreated(_ context.Context, pl domain.VPNTunnelCreatedPayload) error { p.created = append(p.created, pl); return nil }
func (*vpnEvtMock) PublishServerBandwidthHigh(context.Context, domain.VPNServerBandwidthHighPayload) error     { return nil }
func (*vpnEvtMock) PublishServerBandwidthNormal(context.Context, domain.VPNServerBandwidthNormalPayload) error { return nil }
func (*vpnEvtMock) PublishMaintenanceScheduled(context.Context, domain.VPNMaintenanceScheduledPayload) error   { return nil }

type vpnBwMock struct{}
func (*vpnBwMock) Store(context.Context, string, domain.VPNBandwidthMetrics) error                         { return nil }
func (*vpnBwMock) Query(context.Context, string, time.Time, time.Time) ([]domain.VPNBandwidthPoint, error) { return nil, nil }
func (*vpnBwMock) GetLatest(context.Context, string) (*domain.VPNBandwidthPoint, error)                    { return nil, nil }

type vpnCmdMock struct{}
func (*vpnCmdMock) CreateWireGuardInterface(domain.WireGuardInterfaceParams) (string, map[string]string) { return "", nil }
func (*vpnCmdMock) AddWireGuardPeer(domain.WireGuardPeerParams) (string, map[string]string)              { return "", nil }
func (*vpnCmdMock) RemoveWireGuardInterface(string) (string, map[string]string)                          { return "", nil }
func (*vpnCmdMock) RemoveWireGuardPeer(string) (string, map[string]string)                               { return "", nil }
func (*vpnCmdMock) CreateL2TPClient(domain.L2TPClientParams) (string, map[string]string)                 { return "", nil }
func (*vpnCmdMock) RemoveL2TPClient(string) (string, map[string]string)                                  { return "", nil }
func (*vpnCmdMock) CreateIPSecProfile(domain.IPSecProfileParams) (string, map[string]string)             { return "", nil }
func (*vpnCmdMock) CreateIPSecProposal(domain.IPSecProposalParams) (string, map[string]string)           { return "", nil }
func (*vpnCmdMock) CreatePPTPClient(domain.PPTPClientParams) (string, map[string]string)                 { return "", nil }
func (*vpnCmdMock) RemovePPTPClient(string) (string, map[string]string)                                  { return "", nil }
func (*vpnCmdMock) CreateSSTPClient(domain.SSTPClientParams) (string, map[string]string)                 { return "", nil }
func (*vpnCmdMock) RemoveSSTPClient(string) (string, map[string]string)                                  { return "", nil }
func (*vpnCmdMock) CreateOpenVPNClient(domain.OpenVPNClientParams) (string, map[string]string)           { return "", nil }
func (*vpnCmdMock) RemoveOpenVPNClient(string) (string, map[string]string)                               { return "", nil }
func (*vpnCmdMock) AddIPAddress(domain.IPAddressParams) (string, map[string]string)                      { return "", nil }
func (*vpnCmdMock) RemoveIPAddressByInterface(string) (string, map[string]string)                        { return "", nil }
func (*vpnCmdMock) AddIPRoute(domain.IPRouteParams) (string, map[string]string)                          { return "", nil }
func (*vpnCmdMock) AddFirewallFilter(domain.FirewallFilterParams) (string, map[string]string)            { return "", nil }

// newVPNIntegMgr membuat VPN manager dengan semua mock.
func newVPNIntegMgr() (domain.VPNManager, *vpnTunnelRepoMock, *vpnEvtMock) {
	tr, sr, ep := newVPNTunnelRepo(), newVPNSubnetRepo(), &vpnEvtMock{}
	r := &domain.Router{ID: "r1", TenantID: "t1", Host: "10.0.0.1", Port: 8728, Username: "admin", PasswordEncrypted: "secret", RouterOSVersion: "7.12", Status: domain.StatusOnline}
	cfg := VPNServerConfig{PrimaryEndpoint: "vpn.ispboss.id", SecondaryEndpoint: "vpn2.ispboss.id", ServerPublicKey: "srv-pub", SecondaryServerPublicKey: "srv-pub-2", ListenPort: 51820}
	sg := NewVPNScriptGenerator(VPNScriptConfig{PrimaryEndpoint: cfg.PrimaryEndpoint, SecondaryEndpoint: cfg.SecondaryEndpoint, ServerPublicKey: cfg.ServerPublicKey, SecondaryServerPublicKey: cfg.SecondaryServerPublicKey})
	return NewVPNManager(tr, sr, &integRouterRepo{router: r}, &integPoolManager{pool: &integConnPool{adapter: &integAdapter{}}}, &vpnEncMock{}, NewVPNKeyGenerator(), sg, ep, &vpnCmdMock{}, &vpnBwMock{}, cfg, zerolog.Nop()), tr, ep
}

// TestVPNInteg_CreateAndVerify — table-driven: create tunnel per protokol, verifikasi field.
func TestVPNInteg_CreateAndVerify(t *testing.T) {
	for _, tc := range []struct{ name, proto string; wg, l2 bool }{
		{"WireGuard", "wireguard", true, false}, {"L2TP", "l2tp_ipsec", false, true}, {"PPTP", "pptp", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mgr, tr, ep := newVPNIntegMgr()
			resp, err := mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: "t-" + tc.proto, Protocol: tc.proto})
			if err != nil { t.Fatalf("CreateTunnel: %v", err) }
			if !strings.HasPrefix(resp.VPNIP, "10.99.") { t.Errorf("VPNIP=%s, want 10.99.*", resp.VPNIP) }
			s := tr.tunnels[resp.ID]
			if tc.wg && (resp.ClientPublicKey == "" || s.ClientPrivateKeyEncrypted == "" || s.PreSharedKeyEncrypted == "") { t.Error("WireGuard keys harus ada") }
			if tc.l2 && (s.L2TPUsername == "" || s.L2TPPasswordEncrypted == "") { t.Error("L2TP credential harus ada") }
			if len(ep.created) != 1 || ep.created[0].Protocol != tc.proto { t.Errorf("event=%v, want proto=%s", ep.created, tc.proto) }
		})
	}
}

// TestVPNInteg_GenerateScript — create → generate script → verifikasi isi.
func TestVPNInteg_GenerateScript(t *testing.T) {
	mgr, _, _ := newVPNIntegMgr()
	resp, _ := mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: "scr", Protocol: "wireguard"})
	script, err := mgr.GenerateScript(context.Background(), resp.ID)
	if err != nil { t.Fatalf("GenerateScript: %v", err) }
	for _, p := range []string{"/interface/wireguard", "ispboss-vpn", resp.VPNIP} {
		if !strings.Contains(script, p) { t.Errorf("script harus mengandung %q", p) }
	}
}

// TestVPNInteg_GetDetailMasked — create → get detail → private key di-mask.
func TestVPNInteg_GetDetailMasked(t *testing.T) {
	mgr, _, _ := newVPNIntegMgr()
	resp, _ := mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: "det", Protocol: "wireguard"})
	d, err := mgr.GetTunnel(context.Background(), resp.ID)
	if err != nil { t.Fatalf("GetTunnel: %v", err) }
	if d.ClientPrivateKeyMasked != "********" { t.Errorf("private key harus ********") }
	if d.PreSharedKeyMasked != "********" { t.Errorf("PSK harus ********") }
}

// TestVPNInteg_UpdateTunnel — create → update nama → verifikasi.
func TestVPNInteg_UpdateTunnel(t *testing.T) {
	mgr, _, _ := newVPNIntegMgr()
	resp, _ := mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: "old", Protocol: "wireguard"})
	upd, err := mgr.UpdateTunnel(context.Background(), resp.ID, domain.UpdateVPNTunnelRequest{TunnelName: "new"})
	if err != nil { t.Fatalf("UpdateTunnel: %v", err) }
	if upd.TunnelName != "new" { t.Errorf("TunnelName=%s, want new", upd.TunnelName) }
}

// TestVPNInteg_DeleteTunnel — create → delete → verifikasi soft-deleted.
func TestVPNInteg_DeleteTunnel(t *testing.T) {
	mgr, tr, _ := newVPNIntegMgr()
	resp, _ := mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: "del", Protocol: "l2tp_ipsec"})
	if err := mgr.DeleteTunnel(context.Background(), resp.ID); err != nil { t.Fatalf("DeleteTunnel: %v", err) }
	if tr.tunnels[resp.ID].DeletedAt == nil { t.Error("tunnel harus soft-deleted") }
}

// TestVPNInteg_ListTunnels — create 3 → list page=1 size=2 → verifikasi paginasi.
func TestVPNInteg_ListTunnels(t *testing.T) {
	mgr, _, _ := newVPNIntegMgr()
	for i := 0; i < 3; i++ { mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: fmt.Sprintf("l-%d", i), Protocol: "wireguard"}) }
	res, err := mgr.ListTunnels(context.Background(), domain.VPNTunnelListParams{TenantID: "t1", Page: 1, PageSize: 2})
	if err != nil { t.Fatalf("ListTunnels: %v", err) }
	if res.Total != 3 { t.Errorf("Total=%d, want 3", res.Total) }
	if len(res.Data) != 2 { t.Errorf("len(Data)=%d, want 2", len(res.Data)) }
}

// TestVPNInteg_GetSummary — create 2, ubah 1 ke connected → verifikasi counts.
func TestVPNInteg_GetSummary(t *testing.T) {
	mgr, tr, _ := newVPNIntegMgr()
	for i := 0; i < 2; i++ { mgr.CreateTunnel(context.Background(), "t1", domain.CreateVPNTunnelRequest{TunnelName: fmt.Sprintf("s-%d", i), Protocol: "wireguard"}) }
	for _, tun := range tr.tunnels { tun.Status = domain.TunnelStatusConnected; break }
	sum, err := mgr.GetSummary(context.Background())
	if err != nil { t.Fatalf("GetSummary: %v", err) }
	if sum.TotalTunnels != 2 { t.Errorf("Total=%d, want 2", sum.TotalTunnels) }
	if sum.ConnectedCount != 1 { t.Errorf("Connected=%d, want 1", sum.ConnectedCount) }
	if sum.PendingCount != 1 { t.Errorf("Pending=%d, want 1", sum.PendingCount) }
}
