// cable_route_manager_test.go - unit test untuk CableRouteManager.
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

// mockCableRouteRepo adalah implementasi in-memory dari domain.CableRouteRepository.
type mockCableRouteRepo struct {
	mu     sync.Mutex
	routes map[string]*domain.CableRoute // key: ID
}

func newMockCableRouteRepo() *mockCableRouteRepo {
	return &mockCableRouteRepo{routes: make(map[string]*domain.CableRoute)}
}

func (r *mockCableRouteRepo) Create(_ context.Context, route *domain.CableRoute) (*domain.CableRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	route.CreatedAt = now
	route.UpdatedAt = now
	r.routes[route.ID] = route
	return route, nil
}

func (r *mockCableRouteRepo) GetByID(_ context.Context, id string) (*domain.CableRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	route, ok := r.routes[id]
	if !ok {
		return nil, domain.ErrCableRouteNotFound
	}
	return route, nil
}

func (r *mockCableRouteRepo) Update(_ context.Context, route *domain.CableRoute) (*domain.CableRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.routes[route.ID]; !ok {
		return nil, domain.ErrCableRouteNotFound
	}
	route.UpdatedAt = time.Now()
	r.routes[route.ID] = route
	return route, nil
}

func (r *mockCableRouteRepo) SoftDelete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	route, ok := r.routes[id]
	if !ok {
		return domain.ErrCableRouteNotFound
	}
	now := time.Now()
	route.DeletedAt = &now
	return nil
}

// ListByBounds mengembalikan cable route yang terhubung ke node dalam bounding box.
func (r *mockCableRouteRepo) ListByBounds(_ context.Context, params domain.CableRouteListParams) ([]*domain.CableRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []*domain.CableRoute
	for _, route := range r.routes {
		if route.DeletedAt != nil {
			continue
		}
		// Filter berdasarkan tenant
		if params.TenantID != "" && route.TenantID != params.TenantID {
			continue
		}
		// Filter berdasarkan route_type jika diberikan
		if params.RouteType != "" && route.RouteType != params.RouteType {
			continue
		}
		results = append(results, route)
	}
	return results, nil
}

// ListByNode mengembalikan cable route yang terhubung ke node tertentu.
func (r *mockCableRouteRepo) ListByNode(_ context.Context, nodeID string) ([]*domain.CableRoute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var results []*domain.CableRoute
	for _, route := range r.routes {
		if route.DeletedAt != nil {
			continue
		}
		if route.FromNodeID == nodeID || route.ToNodeID == nodeID {
			results = append(results, route)
		}
	}
	return results, nil
}

// =============================================================================
// =============================================================================

func newTestCableRouteManager() (domain.CableRouteManager, *mockCableRouteRepo, *mockMapNodeRepo) {
	cableRepo := newMockCableRouteRepo()
	nodeRepo := newMockMapNodeRepo()
	mgr := NewCableRouteManager(cableRepo, nodeRepo)
	return mgr, cableRepo, nodeRepo
}

func seedMapNode(ctx context.Context, nodeRepo *mockMapNodeRepo, id, tenantID string) {
	nodeRepo.mu.Lock()
	defer nodeRepo.mu.Unlock()
	nodeRepo.nodes[id] = &domain.MapNode{
		ID:          id,
		TenantID:    tenantID,
		NodeType:    domain.NodeTypeODP,
		ReferenceID: "ref-" + id,
		Latitude:    -6.2088,
		Longitude:   106.8456,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// makeCoordinatesJSON membuat json.RawMessage dari array koordinat [][2]float64.
func makeCoordinatesJSON(coords [][2]float64) json.RawMessage {
	data, _ := json.Marshal(coords)
	return data
}

// =============================================================================
// =============================================================================

// mengembalikan CableRouteResponse dengan jarak yang dihitung otomatis.
func TestCreateRoute_HappyPath(t *testing.T) {
	mgr, _, nodeRepo := newTestCableRouteManager()
	ctx := context.Background()

	// Seed node from dan to
	seedMapNode(ctx, nodeRepo, "node-from-1", "tenant-1")
	seedMapNode(ctx, nodeRepo, "node-to-1", "tenant-1")

	// Koordinat Jakarta -> Bandung (sekitar 120km)
	coords := [][2]float64{
		{-6.2088, 106.8456}, // Jakarta
		{-6.9175, 107.6191}, // Bandung
	}

	req := domain.CreateCableRouteRequest{
		FromNodeID:  "node-from-1",
		ToNodeID:    "node-to-1",
		RouteType:   domain.RouteTypeBackbone,
		Coordinates: makeCoordinatesJSON(coords),
	}

	resp, err := mgr.CreateRoute(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateRoute gagal: %v", err)
	}

	if resp.ID == "" {
		t.Error("ID seharusnya tidak kosong")
	}
	if resp.FromNodeID != "node-from-1" {
		t.Errorf("FromNodeID: got %q, want %q", resp.FromNodeID, "node-from-1")
	}
	if resp.ToNodeID != "node-to-1" {
		t.Errorf("ToNodeID: got %q, want %q", resp.ToNodeID, "node-to-1")
	}
	if resp.RouteType != domain.RouteTypeBackbone {
		t.Errorf("RouteType: got %q, want %q", resp.RouteType, domain.RouteTypeBackbone)
	}

	// Verifikasi jarak dihitung otomatis (Jakarta-Bandung ≈ 100-150km)
	if resp.DistanceMeters < 100000 || resp.DistanceMeters > 150000 {
		t.Errorf("DistanceMeters: got %.2f, want antara 100000-150000", resp.DistanceMeters)
	}
}

// =============================================================================
// Unit Tes 2: TestCreateRoute_InvalidCoordinates - kurang dari 2 titik
// =============================================================================

// TestCreateRoute_InvalidCoordinates memverifikasi bahwa CreateRoute mengembalikan
// ErrInvalidCoordArray saat koordinat kurang dari 2 titik.
func TestCreateRoute_InvalidCoordinates(t *testing.T) {
	mgr, _, nodeRepo := newTestCableRouteManager()
	ctx := context.Background()

	// Seed node
	seedMapNode(ctx, nodeRepo, "node-from-2", "tenant-1")
	seedMapNode(ctx, nodeRepo, "node-to-2", "tenant-1")

	// Hanya 1 titik koordinat - harus gagal
	coords := [][2]float64{
		{-6.2088, 106.8456},
	}

	req := domain.CreateCableRouteRequest{
		FromNodeID:  "node-from-2",
		ToNodeID:    "node-to-2",
		RouteType:   domain.RouteTypeDrop,
		Coordinates: makeCoordinatesJSON(coords),
	}

	_, err := mgr.CreateRoute(ctx, "tenant-1", req)
	if err == nil {
		t.Fatal("CreateRoute seharusnya mengembalikan error untuk koordinat < 2 titik")
	}

	if !containsString(err.Error(), domain.ErrInvalidCoordArray.Error()) {
		t.Errorf("error: got %v, want error yang mengandung ErrInvalidCoordArray", err)
	}
}

// =============================================================================
// Unit Tes 3: TestCreateRoute_AutoDistanceCalculation - verifikasi kalkulasi jarak
// =============================================================================

// TestCreateRoute_AutoDistanceCalculation memverifikasi bahwa jarak dihitung
// secara otomatis dari koordinat menggunakan formula Haversine.
func TestCreateRoute_AutoDistanceCalculation(t *testing.T) {
	mgr, _, nodeRepo := newTestCableRouteManager()
	ctx := context.Background()

	seedMapNode(ctx, nodeRepo, "node-from-3", "tenant-1")
	seedMapNode(ctx, nodeRepo, "node-to-3", "tenant-1")

	// Koordinat multi-segment: A -> B -> C
	coords := [][2]float64{
		{-6.2088, 106.8456}, // Titik A
		{-6.3000, 106.9000}, // Titik B
		{-6.4000, 107.0000}, // Titik C
	}

	req := domain.CreateCableRouteRequest{
		FromNodeID:  "node-from-3",
		ToNodeID:    "node-to-3",
		RouteType:   domain.RouteTypeBackbone,
		Coordinates: makeCoordinatesJSON(coords),
	}

	resp, err := mgr.CreateRoute(ctx, "tenant-1", req)
	if err != nil {
		t.Fatalf("CreateRoute gagal: %v", err)
	}

	// Hitung jarak yang diharapkan secara manual
	expectedDistance := domain.CalculateRouteDistance(coords)

	// Verifikasi jarak sesuai dengan kalkulasi Haversine
	if math.Abs(resp.DistanceMeters-expectedDistance) > 0.01 {
		t.Errorf("DistanceMeters: got %.6f, want %.6f", resp.DistanceMeters, expectedDistance)
	}

	// Verifikasi jarak > 0 untuk multi-segment
	if resp.DistanceMeters <= 0 {
		t.Error("DistanceMeters seharusnya > 0 untuk koordinat multi-segment")
	}
}

// =============================================================================
// =============================================================================

// TestUpdateRoute_RecalculateDistance memverifikasi bahwa UpdateRoute menghitung
// ulang jarak saat koordinat diperbarui.
func TestUpdateRoute_RecalculateDistance(t *testing.T) {
	mgr, _, nodeRepo := newTestCableRouteManager()
	ctx := context.Background()

	seedMapNode(ctx, nodeRepo, "node-from-4", "tenant-1")
	seedMapNode(ctx, nodeRepo, "node-to-4", "tenant-1")

	// Buat route awal dengan koordinat pendek
	initialCoords := [][2]float64{
		{-6.2088, 106.8456},
		{-6.2100, 106.8470},
	}

	createReq := domain.CreateCableRouteRequest{
		FromNodeID:  "node-from-4",
		ToNodeID:    "node-to-4",
		RouteType:   domain.RouteTypeDrop,
		Coordinates: makeCoordinatesJSON(initialCoords),
	}

	created, err := mgr.CreateRoute(ctx, "tenant-1", createReq)
	if err != nil {
		t.Fatalf("CreateRoute gagal: %v", err)
	}

	initialDistance := created.DistanceMeters

	newCoords := [][2]float64{
		{-6.2088, 106.8456},
		{-6.5000, 107.0000},
		{-6.9175, 107.6191},
	}
	newCoordsJSON := makeCoordinatesJSON(newCoords)

	updateReq := domain.UpdateCableRouteRequest{
		Coordinates: newCoordsJSON,
	}

	updated, err := mgr.UpdateRoute(ctx, created.ID, updateReq)
	if err != nil {
		t.Fatalf("UpdateRoute gagal: %v", err)
	}

	// Verifikasi jarak berubah (route baru lebih panjang)
	if updated.DistanceMeters <= initialDistance {
		t.Errorf("jarak seharusnya bertambah: initial=%.2f, updated=%.2f",
			initialDistance, updated.DistanceMeters)
	}

	// Verifikasi jarak sesuai dengan kalkulasi Haversine dari koordinat baru
	expectedDistance := domain.CalculateRouteDistance(newCoords)
	if math.Abs(updated.DistanceMeters-expectedDistance) > 0.01 {
		t.Errorf("DistanceMeters: got %.6f, want %.6f", updated.DistanceMeters, expectedDistance)
	}
}
