//go:build integration

// map_integration_test.go — integration test end-to-end untuk FTTH Mapping.
// Menguji alur lengkap: buat node → buat cable route → verifikasi kalkulasi jarak →
// update lokasi node → verifikasi riwayat perubahan → soft delete → verifikasi trash → restore.
// Menggunakan mock in-memory repository (sama seperti map_node_manager_test.go).
// Semua komentar dalam Bahasa Indonesia.
package usecase

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Integration Test: Alur Lengkap FTTH Mapping
// =============================================================================

// TestIntegration_FullMapWorkflow menguji alur end-to-end:
// 1. Buat map node (OLT dan ODP)
// 2. Buat cable route antara kedua node
// 3. Verifikasi kalkulasi jarak otomatis
// 4. Update lokasi node
// 5. Verifikasi riwayat perubahan (created + location_moved)
// 6. Soft delete node
// 7. Verifikasi node masuk trash
// 8. Restore node dari trash
// 9. Verifikasi node kembali aktif
func TestIntegration_FullMapWorkflow(t *testing.T) {
	// --- Inisialisasi semua mock repository ---
	nodeRepo := newMockMapNodeRepo()
	photoRepo := newMockNodePhotoRepo()
	historyRepo := newMockChangeHistoryRepo()
	labelRepo := newMockLabelSettingsRepo()
	cableRepo := newMockCableRouteRepo()

	// Buat usecase manager
	nodeMgr := NewMapNodeManager(nodeRepo, photoRepo, historyRepo, labelRepo)
	cableMgr := NewCableRouteManager(cableRepo, nodeRepo)

	ctx := context.Background()
	tenantID := "tenant-integ-1"

	// =========================================================================
	// Langkah 1: Buat node OLT
	// =========================================================================
	t.Log("Langkah 1: Membuat node OLT")
	oltReq := domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeOLT,
		ReferenceID: "olt-ref-integ-001",
		Latitude:    -6.2088,
		Longitude:   106.8456,
	}
	oltNode, err := nodeMgr.CreateNode(ctx, tenantID, oltReq)
	if err != nil {
		t.Fatalf("gagal membuat node OLT: %v", err)
	}
	if oltNode.ID == "" {
		t.Fatal("ID node OLT seharusnya tidak kosong")
	}
	if oltNode.NodeType != domain.NodeTypeOLT {
		t.Errorf("NodeType OLT: got %q, want %q", oltNode.NodeType, domain.NodeTypeOLT)
	}

	// =========================================================================
	// Langkah 2: Buat node ODP
	// =========================================================================
	t.Log("Langkah 2: Membuat node ODP")
	odpReq := domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeODP,
		ReferenceID: "odp-ref-integ-001",
		Latitude:    -6.9175,
		Longitude:   107.6191,
	}
	odpNode, err := nodeMgr.CreateNode(ctx, tenantID, odpReq)
	if err != nil {
		t.Fatalf("gagal membuat node ODP: %v", err)
	}
	if odpNode.ID == "" {
		t.Fatal("ID node ODP seharusnya tidak kosong")
	}

	// =========================================================================
	// Langkah 3: Buat cable route antara OLT dan ODP
	// =========================================================================
	t.Log("Langkah 3: Membuat cable route OLT → ODP")
	coords := [][2]float64{
		{-6.2088, 106.8456}, // Lokasi OLT (Jakarta)
		{-6.5000, 107.2000}, // Titik tengah
		{-6.9175, 107.6191}, // Lokasi ODP (Bandung)
	}
	coordsJSON, _ := json.Marshal(coords)

	cableReq := domain.CreateCableRouteRequest{
		FromNodeID:  oltNode.ID,
		ToNodeID:    odpNode.ID,
		RouteType:   domain.RouteTypeBackbone,
		Coordinates: coordsJSON,
	}
	cable, err := cableMgr.CreateRoute(ctx, tenantID, cableReq)
	if err != nil {
		t.Fatalf("gagal membuat cable route: %v", err)
	}
	if cable.ID == "" {
		t.Fatal("ID cable route seharusnya tidak kosong")
	}

	// =========================================================================
	// Langkah 4: Verifikasi kalkulasi jarak otomatis
	// =========================================================================
	t.Log("Langkah 4: Verifikasi kalkulasi jarak otomatis via Haversine")
	expectedDistance := domain.CalculateRouteDistance(coords)
	if cable.DistanceMeters != expectedDistance {
		t.Errorf("DistanceMeters: got %.2f, want %.2f", cable.DistanceMeters, expectedDistance)
	}
	// Jarak Jakarta-Bandung via titik tengah harus > 0
	if cable.DistanceMeters <= 0 {
		t.Error("DistanceMeters seharusnya > 0")
	}

	// =========================================================================
	// Langkah 5: Update lokasi node ODP
	// =========================================================================
	t.Log("Langkah 5: Update lokasi node ODP")
	newLat := -6.9200
	newLng := 107.6200
	updateReq := domain.UpdateMapNodeRequest{
		Latitude:  &newLat,
		Longitude: &newLng,
	}
	updatedODP, err := nodeMgr.UpdateNode(ctx, odpNode.ID, updateReq)
	if err != nil {
		t.Fatalf("gagal update lokasi ODP: %v", err)
	}
	if updatedODP.Latitude != newLat {
		t.Errorf("Latitude setelah update: got %f, want %f", updatedODP.Latitude, newLat)
	}
	if updatedODP.Longitude != newLng {
		t.Errorf("Longitude setelah update: got %f, want %f", updatedODP.Longitude, newLng)
	}

	// =========================================================================
	// Langkah 6: Verifikasi riwayat perubahan
	// =========================================================================
	t.Log("Langkah 6: Verifikasi riwayat perubahan node ODP")
	history, err := nodeMgr.GetHistory(ctx, odpNode.ID, 50, 0)
	if err != nil {
		t.Fatalf("gagal mengambil riwayat: %v", err)
	}
	// Harus ada minimal 2 entri: "created" dan "location_moved"
	if len(history) < 2 {
		t.Fatalf("jumlah riwayat: got %d, want >= 2", len(history))
	}

	// Verifikasi aksi riwayat yang tercatat
	actionSet := make(map[string]bool)
	for _, h := range history {
		actionSet[h.Action] = true
	}
	if !actionSet[domain.ChangeActionCreated] {
		t.Error("riwayat seharusnya mengandung aksi 'created'")
	}
	if !actionSet[domain.ChangeActionLocationMoved] {
		t.Error("riwayat seharusnya mengandung aksi 'location_moved'")
	}

	// =========================================================================
	// Langkah 7: Soft delete node ODP
	// =========================================================================
	t.Log("Langkah 7: Soft delete node ODP")
	err = nodeMgr.DeleteNode(ctx, odpNode.ID, "admin-integ")
	if err != nil {
		t.Fatalf("gagal menghapus node ODP: %v", err)
	}

	// =========================================================================
	// Langkah 8: Verifikasi node masuk trash
	// =========================================================================
	t.Log("Langkah 8: Verifikasi node ODP masuk trash")
	trashed, err := nodeMgr.ListTrashed(ctx, tenantID)
	if err != nil {
		t.Fatalf("gagal mengambil daftar trash: %v", err)
	}
	foundInTrash := false
	for _, n := range trashed {
		if n.ID == odpNode.ID {
			foundInTrash = true
			break
		}
	}
	if !foundInTrash {
		t.Error("node ODP seharusnya ditemukan di trash setelah soft delete")
	}

	// Verifikasi riwayat "deleted" tercatat
	historyAfterDelete, err := nodeMgr.GetHistory(ctx, odpNode.ID, 50, 0)
	if err != nil {
		t.Fatalf("gagal mengambil riwayat setelah delete: %v", err)
	}
	deletedActionFound := false
	for _, h := range historyAfterDelete {
		if h.Action == domain.ChangeActionDeleted {
			deletedActionFound = true
			break
		}
	}
	if !deletedActionFound {
		t.Error("riwayat seharusnya mengandung aksi 'deleted' setelah soft delete")
	}

	// =========================================================================
	// Langkah 9: Restore node dari trash
	// =========================================================================
	t.Log("Langkah 9: Restore node ODP dari trash")
	err = nodeMgr.RestoreNode(ctx, odpNode.ID, "admin-integ")
	if err != nil {
		t.Fatalf("gagal restore node ODP: %v", err)
	}

	// Verifikasi node tidak lagi di trash
	trashedAfterRestore, err := nodeMgr.ListTrashed(ctx, tenantID)
	if err != nil {
		t.Fatalf("gagal mengambil daftar trash setelah restore: %v", err)
	}
	stillInTrash := false
	for _, n := range trashedAfterRestore {
		if n.ID == odpNode.ID {
			stillInTrash = true
			break
		}
	}
	if stillInTrash {
		t.Error("node ODP seharusnya tidak ada di trash setelah restore")
	}

	// Verifikasi node bisa diakses kembali via GetNode
	detail, err := nodeMgr.GetNode(ctx, odpNode.ID)
	if err != nil {
		t.Fatalf("gagal mengambil detail node setelah restore: %v", err)
	}
	if detail.ID != odpNode.ID {
		t.Errorf("ID node setelah restore: got %q, want %q", detail.ID, odpNode.ID)
	}

	// Verifikasi riwayat "restored" tercatat
	historyAfterRestore, err := nodeMgr.GetHistory(ctx, odpNode.ID, 50, 0)
	if err != nil {
		t.Fatalf("gagal mengambil riwayat setelah restore: %v", err)
	}
	restoredActionFound := false
	for _, h := range historyAfterRestore {
		if h.Action == domain.ChangeActionRestored {
			restoredActionFound = true
			break
		}
	}
	if !restoredActionFound {
		t.Error("riwayat seharusnya mengandung aksi 'restored' setelah restore")
	}

	t.Log("Integration test alur lengkap FTTH Mapping berhasil!")
}

// =============================================================================
// Integration Test: Isolasi Antar Tenant
// =============================================================================

// TestIntegration_CrossTenantIsolation memverifikasi bahwa data antar tenant
// terisolasi — node dari tenant-A tidak terlihat oleh tenant-B.
func TestIntegration_CrossTenantIsolation(t *testing.T) {
	// --- Inisialisasi mock repository ---
	nodeRepo := newMockMapNodeRepo()
	photoRepo := newMockNodePhotoRepo()
	historyRepo := newMockChangeHistoryRepo()
	labelRepo := newMockLabelSettingsRepo()

	nodeMgr := NewMapNodeManager(nodeRepo, photoRepo, historyRepo, labelRepo)
	ctx := context.Background()

	// =========================================================================
	// Buat node untuk tenant-A
	// =========================================================================
	_, err := nodeMgr.CreateNode(ctx, "tenant-A", domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeOLT,
		ReferenceID: "olt-ref-tenant-a",
		Latitude:    -6.2088,
		Longitude:   106.8456,
	})
	if err != nil {
		t.Fatalf("gagal membuat node tenant-A: %v", err)
	}

	// =========================================================================
	// Buat node untuk tenant-B
	// =========================================================================
	_, err = nodeMgr.CreateNode(ctx, "tenant-B", domain.CreateMapNodeRequest{
		NodeType:    domain.NodeTypeODP,
		ReferenceID: "odp-ref-tenant-b",
		Latitude:    -7.2575,
		Longitude:   112.7521,
	})
	if err != nil {
		t.Fatalf("gagal membuat node tenant-B: %v", err)
	}

	// =========================================================================
	// Verifikasi isolasi: ListNodes tenant-A hanya mengembalikan node tenant-A
	// =========================================================================
	paramsA := domain.MapNodeListParams{
		TenantID: "tenant-A",
		MinLat:   -90,
		MaxLat:   90,
		MinLng:   -180,
		MaxLng:   180,
	}
	nodesA, err := nodeMgr.ListNodes(ctx, paramsA)
	if err != nil {
		t.Fatalf("gagal list nodes tenant-A: %v", err)
	}
	if len(nodesA) != 1 {
		t.Errorf("jumlah node tenant-A: got %d, want 1", len(nodesA))
	}

	// =========================================================================
	// Verifikasi isolasi: ListNodes tenant-B hanya mengembalikan node tenant-B
	// =========================================================================
	paramsB := domain.MapNodeListParams{
		TenantID: "tenant-B",
		MinLat:   -90,
		MaxLat:   90,
		MinLng:   -180,
		MaxLng:   180,
	}
	nodesB, err := nodeMgr.ListNodes(ctx, paramsB)
	if err != nil {
		t.Fatalf("gagal list nodes tenant-B: %v", err)
	}
	if len(nodesB) != 1 {
		t.Errorf("jumlah node tenant-B: got %d, want 1", len(nodesB))
	}

	// =========================================================================
	// Verifikasi isolasi: Search tenant-A tidak mengembalikan data tenant-B
	// =========================================================================
	searchA, err := nodeMgr.Search(ctx, "tenant-A", "olt")
	if err != nil {
		t.Fatalf("gagal search tenant-A: %v", err)
	}
	for _, r := range searchA {
		// Semua hasil pencarian harus bertipe OLT (milik tenant-A)
		if r.Type != domain.NodeTypeOLT {
			t.Errorf("search tenant-A mengembalikan tipe %q, seharusnya hanya OLT", r.Type)
		}
	}

	// =========================================================================
	// Verifikasi isolasi: ListTrashed tenant-A kosong (belum ada yang dihapus)
	// =========================================================================
	trashedA, err := nodeMgr.ListTrashed(ctx, "tenant-A")
	if err != nil {
		t.Fatalf("gagal list trashed tenant-A: %v", err)
	}
	if len(trashedA) != 0 {
		t.Errorf("trash tenant-A seharusnya kosong: got %d", len(trashedA))
	}

	t.Log("Integration test isolasi antar tenant berhasil!")
}
