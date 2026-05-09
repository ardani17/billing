package domain

import (
	"encoding/json"
	"time"
)

// =============================================================================
// Map Node DTO permintaan - payload dari HTTP permintaan untuk operasi map node
// =============================================================================

// CreateMapNodeRequest adalah payload untuk POST /api/v1/network-map/nodes.
// Digunakan untuk membuat map node baru yang merujuk ke entitas OLT/ODP/ONT.
type CreateMapNodeRequest struct {
	NodeType     string          `json:"node_type" validate:"required,oneof=olt odp ont"`
	ReferenceID  string          `json:"reference_id" validate:"required,uuid"`
	Latitude     float64         `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude    float64         `json:"longitude" validate:"required,min=-180,max=180"`
	CustomFields json.RawMessage `json:"custom_fields,omitempty"`
}

// UpdateMapNodeRequest adalah payload untuk PUT /api/v1/network-map/nodes/:id.
// Semua field bersifat opsional - hanya field yang dikirim yang akan diupdate.
type UpdateMapNodeRequest struct {
	Latitude     *float64        `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude    *float64        `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	CustomFields json.RawMessage `json:"custom_fields,omitempty"`
}

// =============================================================================
// =============================================================================

// MapNodeResponse adalah respons dasar untuk operasi buat/perbarui map node.
// Berisi data map node tanpa informasi join dari entitas referensi.
type MapNodeResponse struct {
	ID           string          `json:"id"`
	NodeType     string          `json:"node_type"`
	ReferenceID  string          `json:"reference_id"`
	Latitude     float64         `json:"latitude"`
	Longitude    float64         `json:"longitude"`
	CustomFields json.RawMessage `json:"custom_fields,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// MapNodeDetailResponse adalah respons untuk GET /api/v1/network-map/nodes/:id.
// Menyertakan foto, riwayat perubahan, dan data join dari entitas referensi
// (OLT/ODP/ONT) untuk tampilan detail lengkap di panel samping.
// NodePhotoResponse dan MapChangeHistoryResponse didefinisikan di map_share_dto.go.
type MapNodeDetailResponse struct {
	MapNodeResponse
	Photos  []NodePhotoResponse        `json:"photos"`
	History []MapChangeHistoryResponse `json:"history"`
	RefData map[string]interface{}     `json:"ref_data,omitempty"`
}

// MapNodeWithRefResponse adalah respons untuk GET /api/v1/network-map/nodes (list).
// Berisi data map node yang sudah di-join dengan informasi dari entitas referensi
// (OLT/ODP/ONT) untuk ditampilkan sebagai marker di peta.
type MapNodeWithRefResponse struct {
	ID            string          `json:"id"`
	NodeType      string          `json:"node_type"`
	ReferenceID   string          `json:"reference_id"`
	Latitude      float64         `json:"latitude"`
	Longitude     float64         `json:"longitude"`
	CustomFields  json.RawMessage `json:"custom_fields,omitempty"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	Signal        *float64        `json:"signal,omitempty"`
	CustomerName  *string         `json:"customer_name,omitempty"`
	CustomerID    *string         `json:"customer_id,omitempty"`
	PackageName   *string         `json:"package_name,omitempty"`
	SerialNumber  *string         `json:"serial_number,omitempty"`
	SplitterType  *string         `json:"splitter_type,omitempty"`
	Capacity      *int            `json:"capacity,omitempty"`
	UsedPorts     *int            `json:"used_ports,omitempty"`
	Address       *string         `json:"address,omitempty"`
	BillingStatus *string         `json:"billing_status,omitempty"`
	PackageID     *string         `json:"package_id,omitempty"`
	AreaID        *string         `json:"area_id,omitempty"`
	ODPID         *string         `json:"odp_id,omitempty"`
}

// =============================================================================
// Label Settings DTOs - permintaan/respons untuk konfigurasi label di peta
// =============================================================================

// UpdateLabelSettingsRequest adalah payload untuk PUT /api/v1/network-map/settings/labels.
// Menentukan informasi apa yang tampil di label node per tipe di peta.
// Semua field bersifat opsional - hanya field yang dikirim yang akan diupdate.
type UpdateLabelSettingsRequest struct {
	OLTLabels    json.RawMessage `json:"olt_labels,omitempty"`
	ODPLabels    json.RawMessage `json:"odp_labels,omitempty"`
	ONTLabels    json.RawMessage `json:"ont_labels,omitempty"`
	MinZoomLevel *int            `json:"min_zoom_level,omitempty" validate:"omitempty,min=1,max=20"`
}

// MapLabelSettingsResponse adalah respons untuk GET/PUT /api/v1/network-map/settings/labels.
// Berisi konfigurasi label lengkap per tipe node untuk tenant.
type MapLabelSettingsResponse struct {
	OLTLabels    json.RawMessage `json:"olt_labels"`
	ODPLabels    json.RawMessage `json:"odp_labels"`
	ONTLabels    json.RawMessage `json:"ont_labels"`
	MinZoomLevel int             `json:"min_zoom_level"`
}

// =============================================================================
// Fungsi bantu Functions - konversi entity ke respons DTO
// =============================================================================

// ToMapNodeResponse mengkonversi MapNode entity ke MapNodeResponse DTO.
func ToMapNodeResponse(node *MapNode) *MapNodeResponse {
	return &MapNodeResponse{
		ID:           node.ID,
		NodeType:     node.NodeType,
		ReferenceID:  node.ReferenceID,
		Latitude:     node.Latitude,
		Longitude:    node.Longitude,
		CustomFields: node.CustomFields,
		CreatedAt:    node.CreatedAt,
		UpdatedAt:    node.UpdatedAt,
	}
}

// ToMapNodeWithRefResponse mengkonversi MapNodeWithRef ke MapNodeWithRefResponse DTO.
func ToMapNodeWithRefResponse(n *MapNodeWithRef) *MapNodeWithRefResponse {
	return &MapNodeWithRefResponse{
		ID:            n.ID,
		NodeType:      n.NodeType,
		ReferenceID:   n.ReferenceID,
		Latitude:      n.Latitude,
		Longitude:     n.Longitude,
		CustomFields:  n.CustomFields,
		Name:          n.Name,
		Status:        n.Status,
		Signal:        n.Signal,
		CustomerName:  n.CustomerName,
		CustomerID:    n.CustomerID,
		PackageName:   n.PackageName,
		SerialNumber:  n.SerialNumber,
		SplitterType:  n.SplitterType,
		Capacity:      n.Capacity,
		UsedPorts:     n.UsedPorts,
		Address:       n.Address,
		BillingStatus: n.BillingStatus,
		PackageID:     n.PackageID,
		AreaID:        n.AreaID,
		ODPID:         n.ODPID,
	}
}

// ToMapLabelSettingsResponse mengkonversi MapLabelSettings entity ke respons DTO.
func ToMapLabelSettingsResponse(s *MapLabelSettings) *MapLabelSettingsResponse {
	return &MapLabelSettingsResponse{
		OLTLabels:    s.OLTLabels,
		ODPLabels:    s.ODPLabels,
		ONTLabels:    s.ONTLabels,
		MinZoomLevel: s.MinZoomLevel,
	}
}
