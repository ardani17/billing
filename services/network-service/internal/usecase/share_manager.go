// Package usecase berisi implementasi business logic untuk network-service.
// File ini mendefinisikan ShareManager: manajemen share link peta hanya baca.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// shareBaseURL adalah base URL untuk share link (dikonfigurasi saat deployment).
const shareBaseURL = "/api/v1/network-map"

// Compile-time cek: shareManager harus mengimplementasikan domain.ShareManager.
var _ domain.ShareManager = (*shareManager)(nil)

// shareManager mengimplementasikan domain.ShareManager.
// Mengelola pembuatan, validasi, dan penghapusan share link hanya baca.
type shareManager struct {
	shareLinkRepo  domain.ShareLinkRepository
	mapNodeRepo    domain.MapNodeRepository
	cableRouteRepo domain.CableRouteRepository
}

// NewShareManager membuat instance ShareManager baru dengan dependensi repositori.
func NewShareManager(
	shareLinkRepo domain.ShareLinkRepository,
	mapNodeRepo domain.MapNodeRepository,
	cableRouteRepo domain.CableRouteRepository,
) domain.ShareManager {
	return &shareManager{
		shareLinkRepo:  shareLinkRepo,
		mapNodeRepo:    mapNodeRepo,
		cableRouteRepo: cableRouteRepo,
	}
}

// CreateShareLink membuat share link baru dengan opsi expiry dan password.
// Token di-buat secara kriptografis aman (32 bytes hex).
// Password di-hash menggunakan bcrypt sebelum disimpan.
func (m *shareManager) CreateShareLink(ctx context.Context, tenantID, createdBy string, req domain.CreateShareLinkRequest) (*domain.ShareLinkResponse, error) {
	// Buat token unik
	token, err := domain.GenerateShareToken()
	if err != nil {
		return nil, fmt.Errorf("gagal generate token: %w", err)
	}

	// Hash password jika diberikan
	var passwordHash *string
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("gagal hash password: %w", err)
		}
		hashStr := string(hash)
		passwordHash = &hashStr
	}

	// Hitung expiry jika diberikan
	var expiresAt *time.Time
	if req.ExpiryDays != nil && *req.ExpiryDays > 0 {
		exp := time.Now().AddDate(0, 0, *req.ExpiryDays)
		expiresAt = &exp
	}

	link := &domain.MapShareLink{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Token:         token,
		VisibleLayers: req.VisibleLayers,
		ExpiresAt:     expiresAt,
		PasswordHash:  passwordHash,
		AccessCount:   0,
		CreatedBy:     createdBy,
	}

	created, err := m.shareLinkRepo.Create(ctx, link)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat share link: %w", err)
	}

	return domain.ToShareLinkResponse(created, shareBaseURL), nil
}

// GetSharedMap mengambil data peta berdasarkan token share link.
// Validasi: token harus ada, belum expired, password benar (jika ada).
// Data difilter berdasarkan visible_layers yang ditentukan saat pembuatan link.
func (m *shareManager) GetSharedMap(ctx context.Context, token, password string) (*domain.SharedMapData, error) {
	// Ambil share link berdasarkan token
	link, err := m.shareLinkRepo.GetByToken(ctx, token)
	if err != nil {
		return nil, domain.ErrShareLinkNotFound
	}

	// Validasi expiry
	if link.IsExpired() {
		return nil, domain.ErrShareLinkExpired
	}

	// Validasi password jika link dilindungi password
	if link.PasswordHash != nil {
		if password == "" {
			return nil, domain.ErrShareLinkPassword
		}
		if err := bcrypt.CompareHashAndPassword([]byte(*link.PasswordHash), []byte(password)); err != nil {
			return nil, domain.ErrShareLinkPassword
		}
	}

	// Increment access count
	if err := m.shareLinkRepo.IncrementAccessCount(ctx, token); err != nil {
		// Log tapi jangan gagalkan operasi
		_ = err
	}

	// Kueri data peta berdasarkan visible_layers
	nodes, cables, err := m.querySharedData(ctx, link.TenantID, link.VisibleLayers)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data peta: %w", err)
	}

	// Konversi ke respons
	nodeResponses := make([]domain.MapNodeWithRefResponse, 0, len(nodes))
	for _, n := range nodes {
		nodeResponses = append(nodeResponses, *domain.ToMapNodeWithRefResponse(n))
	}

	cableResponses := make([]domain.CableRouteResponse, 0, len(cables))
	for _, c := range cables {
		cableResponses = append(cableResponses, *domain.ToCableRouteResponse(c))
	}

	return &domain.SharedMapData{
		Nodes:         nodeResponses,
		Cables:        cableResponses,
		VisibleLayers: link.VisibleLayers,
	}, nil
}

// DeleteShareLink menghapus share link berdasarkan token.
func (m *shareManager) DeleteShareLink(ctx context.Context, token string) error {
	if err := m.shareLinkRepo.Delete(ctx, token); err != nil {
		return fmt.Errorf("gagal menghapus share link: %w", err)
	}
	return nil
}

// ListShareLinks mengambil daftar share link untuk satu tenant.
func (m *shareManager) ListShareLinks(ctx context.Context, tenantID string) ([]*domain.ShareLinkResponse, error) {
	links, err := m.shareLinkRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar share link: %w", err)
	}

	responses := make([]*domain.ShareLinkResponse, 0, len(links))
	for _, l := range links {
		responses = append(responses, domain.ToShareLinkResponse(l, shareBaseURL))
	}

	return responses, nil
}

// kueriSharedData mengambil data node dan cable berdasarkan visible_layers.
func (m *shareManager) querySharedData(
	ctx context.Context,
	tenantID string,
	visibleLayers json.RawMessage,
) ([]*domain.MapNodeWithRef, []*domain.CableRoute, error) {
	// Parsing visible layers
	var layers []string
	if err := json.Unmarshal(visibleLayers, &layers); err != nil {
		// Jika gagal parsing, tampilkan semua layer
		layers = append(domain.ValidNodeTypes, domain.ValidRouteTypes...)
	}

	var allNodes []*domain.MapNodeWithRef
	var allCables []*domain.CableRoute

	// Kueri node berdasarkan layer yang visible
	for _, layer := range layers {
		if domain.IsValidNodeType(layer) {
			params := domain.MapNodeListParams{
				TenantID: tenantID,
				NodeType: layer,
				MinLat:   -90,
				MaxLat:   90,
				MinLng:   -180,
				MaxLng:   180,
			}
			nodes, err := m.mapNodeRepo.ListByBounds(ctx, params)
			if err != nil {
				return nil, nil, err
			}
			allNodes = append(allNodes, nodes...)
		}
	}

	// Kueri cable berdasarkan layer yang visible
	for _, layer := range layers {
		if domain.IsValidRouteType(layer) {
			params := domain.CableRouteListParams{
				TenantID:  tenantID,
				RouteType: layer,
				MinLat:    -90,
				MaxLat:    90,
				MinLng:    -180,
				MaxLng:    180,
			}
			cables, err := m.cableRouteRepo.ListByBounds(ctx, params)
			if err != nil {
				return nil, nil, err
			}
			allCables = append(allCables, cables...)
		}
	}

	return allNodes, allCables, nil
}
