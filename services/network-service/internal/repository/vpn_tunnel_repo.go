package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// VPNTunnelRepo mengimplementasikan domain.VPNTunnelRepository dengan membungkus
// sqlc-generated Queries dan memetakan tipe database ke domain.VPNTunnel.
type VPNTunnelRepo struct {
	queries *Queries
}

// NewVPNTunnelRepo membuat instance baru VPNTunnelRepo.
func NewVPNTunnelRepo(queries *Queries) *VPNTunnelRepo {
	return &VPNTunnelRepo{queries: queries}
}

// --- Helper konversi nullable UUID untuk router_id ---

// uuidToStringPtr mengkonversi pgtype.UUID ke *string. NULL → nil.
func uuidToStringPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuidToString(u)
	return &s
}

// stringPtrToUUID mengkonversi *string ke pgtype.UUID. nil → NULL.
func stringPtrToUUID(s *string) pgtype.UUID {
	if s == nil || *s == "" {
		return pgtype.UUID{}
	}
	return stringToUUID(*s)
}

// int4ToIntPtr mengkonversi pgtype.Int4 ke *int. NULL → nil.
func int4ToIntPtr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// intPtrToInt4 mengkonversi *int ke pgtype.Int4. nil → NULL.
func intPtrToInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// --- Mapping sqlc VpnTunnel → domain.VPNTunnel ---

// mapVPNTunnelRow memetakan VpnTunnel (sqlc model) ke domain.VPNTunnel.
func mapVPNTunnelRow(row VpnTunnel) *domain.VPNTunnel {
	return &domain.VPNTunnel{
		ID:                        uuidToString(row.ID),
		TenantID:                  uuidToString(row.TenantID),
		RouterID:                  uuidToStringPtr(row.RouterID),
		TunnelName:                row.TunnelName,
		Protocol:                  domain.VPNProtocol(row.Protocol),
		VPNIP:                     row.VpnIp,
		ServerEndpoint:            row.ServerEndpoint,
		ServerPublicKey:           textToString(row.ServerPublicKey),
		ClientPublicKey:           textToString(row.ClientPublicKey),
		ClientPrivateKeyEncrypted: textToString(row.ClientPrivateKeyEncrypted),
		PreSharedKeyEncrypted:     textToString(row.PreSharedKeyEncrypted),
		L2TPUsername:              textToString(row.L2tpUsername),
		L2TPPasswordEncrypted:     textToString(row.L2tpPasswordEncrypted),
		Status:                    domain.TunnelStatus(row.Status),
		ListenPort:                int(row.ListenPort),
		AllowedAddresses:          row.AllowedAddresses,
		PersistentKeepalive:       int(row.PersistentKeepalive),
		LastHandshakeAt:           timestamptzToTimePtr(row.LastHandshakeAt),
		LatencyMs:                 int4ToIntPtr(row.LatencyMs),
		BandwidthCapMbps:          int4ToIntPtr(row.BandwidthCapMbps),
		RateLimitPps:              int(row.RateLimitPps),
		ActiveEndpoint:            textToString(row.ActiveEndpoint),
		Notes:                     textToString(row.Notes),
		CreatedAt:                 timestamptzToTime(row.CreatedAt),
		UpdatedAt:                 timestamptzToTime(row.UpdatedAt),
		DeletedAt:                 timestamptzToTimePtr(row.DeletedAt),
	}
}

// tunnelToResponse mengkonversi domain.VPNTunnel ke domain.VPNTunnelResponse untuk list.
func tunnelToResponse(t *domain.VPNTunnel) *domain.VPNTunnelResponse {
	return &domain.VPNTunnelResponse{
		ID:                  t.ID,
		TunnelName:          t.TunnelName,
		RouterID:            t.RouterID,
		Protocol:            t.Protocol,
		VPNIP:               t.VPNIP,
		ServerEndpoint:      t.ServerEndpoint,
		ServerPublicKey:     t.ServerPublicKey,
		ClientPublicKey:     t.ClientPublicKey,
		Status:              t.Status,
		ListenPort:          t.ListenPort,
		AllowedAddresses:    t.AllowedAddresses,
		PersistentKeepalive: t.PersistentKeepalive,
		LatencyMs:           t.LatencyMs,
		BandwidthCapMbps:    t.BandwidthCapMbps,
		LastHandshakeAt:     t.LastHandshakeAt,
		Notes:               t.Notes,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
}

// --- Implementasi CRUD domain.VPNTunnelRepository ---

// Create membuat record VPN tunnel baru.
func (r *VPNTunnelRepo) Create(ctx context.Context, tunnel *domain.VPNTunnel) (*domain.VPNTunnel, error) {
	row, err := r.queries.CreateVPNTunnel(ctx, CreateVPNTunnelParams{
		TenantID:                  stringToUUID(tunnel.TenantID),
		RouterID:                  stringPtrToUUID(tunnel.RouterID),
		TunnelName:                tunnel.TunnelName,
		Protocol:                  string(tunnel.Protocol),
		VpnIp:                     tunnel.VPNIP,
		ServerEndpoint:            tunnel.ServerEndpoint,
		ServerPublicKey:           stringToText(tunnel.ServerPublicKey),
		ClientPublicKey:           stringToText(tunnel.ClientPublicKey),
		ClientPrivateKeyEncrypted: stringToText(tunnel.ClientPrivateKeyEncrypted),
		PreSharedKeyEncrypted:     stringToText(tunnel.PreSharedKeyEncrypted),
		L2tpUsername:              stringToText(tunnel.L2TPUsername),
		L2tpPasswordEncrypted:     stringToText(tunnel.L2TPPasswordEncrypted),
		Status:                    string(tunnel.Status),
		ListenPort:                int32(tunnel.ListenPort),
		AllowedAddresses:          tunnel.AllowedAddresses,
		PersistentKeepalive:       int32(tunnel.PersistentKeepalive),
		BandwidthCapMbps:          intPtrToInt4(tunnel.BandwidthCapMbps),
		RateLimitPps:              int32(tunnel.RateLimitPps),
		ActiveEndpoint:            stringToText(tunnel.ActiveEndpoint),
		Notes:                     stringToText(tunnel.Notes),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, domain.ErrVPNTunnelNameExists
		}
		return nil, fmt.Errorf("repository: gagal membuat vpn tunnel: %w", err)
	}
	return mapVPNTunnelRow(row), nil
}

// GetByID mengambil VPN tunnel berdasarkan ID (tenant-scoped via RLS).
func (r *VPNTunnelRepo) GetByID(ctx context.Context, id string) (*domain.VPNTunnel, error) {
	row, err := r.queries.GetVPNTunnelByID(ctx, stringToUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVPNTunnelNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil vpn tunnel by ID: %w", err)
	}
	return mapVPNTunnelRow(row), nil
}

// Update memperbarui record VPN tunnel (field yang diizinkan saja).
func (r *VPNTunnelRepo) Update(ctx context.Context, tunnel *domain.VPNTunnel) (*domain.VPNTunnel, error) {
	row, err := r.queries.UpdateVPNTunnel(ctx, UpdateVPNTunnelParams{
		ID:                  stringToUUID(tunnel.ID),
		TunnelName:          tunnel.TunnelName,
		RouterID:            stringPtrToUUID(tunnel.RouterID),
		Notes:               stringToText(tunnel.Notes),
		PersistentKeepalive: int32(tunnel.PersistentKeepalive),
		AllowedAddresses:    tunnel.AllowedAddresses,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrVPNTunnelNotFound
		}
		if isUniqueViolation(err) {
			return nil, domain.ErrVPNTunnelNameExists
		}
		return nil, fmt.Errorf("repository: gagal memperbarui vpn tunnel: %w", err)
	}
	return mapVPNTunnelRow(row), nil
}

// SoftDelete melakukan soft-delete VPN tunnel (set deleted_at).
func (r *VPNTunnelRepo) SoftDelete(ctx context.Context, id string) error {
	err := r.queries.SoftDeleteVPNTunnel(ctx, stringToUUID(id))
	if err != nil {
		return fmt.Errorf("repository: gagal soft-delete vpn tunnel: %w", err)
	}
	return nil
}
