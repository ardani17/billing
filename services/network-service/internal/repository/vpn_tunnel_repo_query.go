package repository

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// --- Implementasi query dan health methods domain.VPNTunnelRepository ---

// List mengambil daftar VPN tunnel dengan paginasi dan filter.
func (r *VPNTunnelRepo) List(ctx context.Context, params domain.VPNTunnelListParams) (*domain.VPNTunnelListResult, error) {
	offset := (params.Page - 1) * params.PageSize

	// Hitung total untuk paginasi
	total, err := r.queries.CountVPNTunnels(ctx, CountVPNTunnelsParams{
		Status:   stringToText(params.Status),
		Protocol: stringToText(params.Protocol),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung total vpn tunnels: %w", err)
	}

	// Ambil data tunnel
	rows, err := r.queries.ListVPNTunnels(ctx, ListVPNTunnelsParams{
		Limit:    int32(params.PageSize),
		Offset:   int32(offset),
		Status:   stringToText(params.Status),
		Protocol: stringToText(params.Protocol),
		Search:   stringToText(params.Search),
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil daftar vpn tunnels: %w", err)
	}

	responses := make([]*domain.VPNTunnelResponse, 0, len(rows))
	for _, row := range rows {
		responses = append(responses, tunnelToResponse(mapVPNTunnelRow(row)))
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &domain.VPNTunnelListResult{
		Data:       responses,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByStatus mengambil semua tunnel dengan status tertentu.
func (r *VPNTunnelRepo) GetByStatus(ctx context.Context, status domain.TunnelStatus) ([]*domain.VPNTunnel, error) {
	rows, err := r.queries.GetVPNTunnelsByStatus(ctx, string(status))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil vpn tunnels by status: %w", err)
	}

	tunnels := make([]*domain.VPNTunnel, 0, len(rows))
	for _, row := range rows {
		tunnels = append(tunnels, mapVPNTunnelRow(row))
	}
	return tunnels, nil
}

// CountByStatus menghitung jumlah tunnel per status untuk tenant.
func (r *VPNTunnelRepo) CountByStatus(ctx context.Context) (map[domain.TunnelStatus]int64, error) {
	rows, err := r.queries.CountVPNTunnelsByStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal menghitung vpn tunnels per status: %w", err)
	}

	result := make(map[domain.TunnelStatus]int64)
	for _, row := range rows {
		result[domain.TunnelStatus(row.Status)] = row.Count
	}
	return result, nil
}

// TunnelNameExists mengecek apakah tunnel_name sudah ada di tenant.
func (r *VPNTunnelRepo) TunnelNameExists(ctx context.Context, tenantID, name, excludeID string) (bool, error) {
	// Gunakan UUID nil jika excludeID kosong agar tidak mengecualikan siapapun
	exID := excludeID
	if exID == "" {
		exID = "00000000-0000-0000-0000-000000000000"
	}

	exists, err := r.queries.VPNTunnelNameExists(ctx, VPNTunnelNameExistsParams{
		TenantID:   stringToUUID(tenantID),
		TunnelName: name,
		ID:         stringToUUID(exID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek nama vpn tunnel: %w", err)
	}
	return exists, nil
}

// VPNIPExists mengecek apakah vpn_ip sudah digunakan di tenant.
func (r *VPNTunnelRepo) VPNIPExists(ctx context.Context, tenantID, vpnIP string) (bool, error) {
	exists, err := r.queries.VPNIPExists(ctx, VPNIPExistsParams{
		TenantID: stringToUUID(tenantID),
		VpnIp:    vpnIP,
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek vpn ip: %w", err)
	}
	return exists, nil
}

// UpdateStatus memperbarui status tunnel dan field terkait health check.
func (r *VPNTunnelRepo) UpdateStatus(ctx context.Context, id string, params domain.TunnelHealthUpdate) error {
	status := ""
	if params.Status != nil {
		status = string(*params.Status)
	}

	err := r.queries.UpdateVPNTunnelStatus(ctx, UpdateVPNTunnelStatusParams{
		ID:              stringToUUID(id),
		Status:          status,
		LastHandshakeAt: timePtrToTimestamptz(params.LastHandshakeAt),
		LatencyMs:       intPtrToInt4(params.LatencyMs),
		ActiveEndpoint:  stringToText(params.ActiveEndpoint),
	})
	if err != nil {
		return fmt.Errorf("repository: gagal memperbarui status vpn tunnel: %w", err)
	}
	return nil
}

// GetConnectedTunnels mengambil semua tunnel dengan status "connected" (cross-tenant).
func (r *VPNTunnelRepo) GetConnectedTunnels(ctx context.Context) ([]*domain.VPNTunnel, error) {
	rows, err := r.queries.GetConnectedVPNTunnels(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil connected vpn tunnels: %w", err)
	}

	tunnels := make([]*domain.VPNTunnel, 0, len(rows))
	for _, row := range rows {
		tunnels = append(tunnels, mapVPNTunnelRow(row))
	}
	return tunnels, nil
}

// GetDisconnectedTunnels mengambil semua tunnel dengan status "disconnected" (cross-tenant).
func (r *VPNTunnelRepo) GetDisconnectedTunnels(ctx context.Context) ([]*domain.VPNTunnel, error) {
	rows, err := r.queries.GetDisconnectedVPNTunnels(ctx)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil disconnected vpn tunnels: %w", err)
	}

	tunnels := make([]*domain.VPNTunnel, 0, len(rows))
	for _, row := range rows {
		tunnels = append(tunnels, mapVPNTunnelRow(row))
	}
	return tunnels, nil
}
