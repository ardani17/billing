package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Konstanta Bawaan Label - nilai bawaan label per tipe node
// =============================================================================

// DefaultMinZoomLevel adalah level zoom minimum untuk menampilkan label di peta.
// Label hanya ditampilkan saat zoom level >= 15.
const DefaultMinZoomLevel = 15

// DefaultOLTLabels berisi label bawaan untuk node OLT: name, brand_model, ont_count.
var DefaultOLTLabels = json.RawMessage(`["name","brand_model","ont_count"]`)

// DefaultODPLabels berisi label bawaan untuk node ODP: name, splitter_type, capacity.
var DefaultODPLabels = json.RawMessage(`["name","splitter_type","capacity"]`)

// DefaultONTLabels berisi label bawaan untuk node ONT: customer_name, package.
var DefaultONTLabels = json.RawMessage(`["customer_name","package"]`)

// =============================================================================
// MapLabelSettings Entitas - konfigurasi label per tenant
// =============================================================================

// MapLabelSettings merepresentasikan konfigurasi label per tenant.
// Menentukan informasi apa yang tampil di label node di peta.
// Setiap tenant memiliki satu record label settings (UNIQUE tenant_id).
type MapLabelSettings struct {
	ID           string          `json:"id"`
	TenantID     string          `json:"tenant_id"`
	OLTLabels    json.RawMessage `json:"olt_labels"`
	ODPLabels    json.RawMessage `json:"odp_labels"`
	ONTLabels    json.RawMessage `json:"ont_labels"`
	MinZoomLevel int             `json:"min_zoom_level"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// NewDefaultLabelSettings membuat MapLabelSettings baru dengan nilai bawaan.
// Digunakan saat tenant belum memiliki konfigurasi label.
func NewDefaultLabelSettings(tenantID string) MapLabelSettings {
	now := time.Now()
	return MapLabelSettings{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		OLTLabels:    cloneRawMessage(DefaultOLTLabels),
		ODPLabels:    cloneRawMessage(DefaultODPLabels),
		ONTLabels:    cloneRawMessage(DefaultONTLabels),
		MinZoomLevel: DefaultMinZoomLevel,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// cloneRawMessage membuat salinan json.RawMessage agar tidak berbagi slice.
func cloneRawMessage(src json.RawMessage) json.RawMessage {
	if src == nil {
		return nil
	}
	dst := make(json.RawMessage, len(src))
	copy(dst, src)
	return dst
}
