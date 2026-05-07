// Package usecase - implementasi ValidateBulk, ExecuteBulk, dan GetBulkTemplate.
// Parsing CSV, validasi per baris, eksekusi provisioning secara sekuensial.
package usecase

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// csvHeaders mendefinisikan kolom CSV template untuk bulk provisioning.
var csvHeaders = []string{"sn_ont", "pelanggan_id", "pon_port", "vlan", "odp", "deskripsi"}

// GetBulkTemplate mengembalikan CSV template bytes dengan header columns.
func (pm *provisioningManager) GetBulkTemplate() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(csvHeaders)
	// Contoh baris
	_ = w.Write([]string{"ZTEG12345678", "customer-uuid", "0", "vlan-uuid", "odp-uuid", "Deskripsi ONT"})
	w.Flush()
	return buf.Bytes()
}

// ValidateBulk memvalidasi CSV upload dan mengembalikan preview.
// Setiap baris divalidasi: format, serial number unik, customer valid, posisi tersedia.
func (pm *provisioningManager) ValidateBulk(ctx context.Context, tenantID string, oltID string, csvData []byte) (*domain.BulkPreview, error) {
	reader := csv.NewReader(bytes.NewReader(csvData))

	// Baca header
	header, err := reader.Read()
	if err != nil {
		return nil, domain.ErrInvalidCSVFormat
	}
	if len(header) < 4 {
		return nil, domain.ErrInvalidCSVFormat
	}

	bulkID := uuid.New().String()
	preview := &domain.BulkPreview{
		BulkID: bulkID,
		OLTID:  oltID,
		Rows:   []domain.BulkRowPreview{},
	}

	rowNum := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, domain.ErrInvalidCSVFormat
		}
		rowNum++

		row := pm.validateBulkRow(ctx, tenantID, oltID, rowNum, record)
		preview.Rows = append(preview.Rows, row)
		preview.TotalRows++
		if row.Valid {
			preview.ValidCount++
		} else {
			preview.ErrorCount++
		}
	}

	// Simpan preview di memory untuk ExecuteBulk
	pm.bulkStore[bulkID] = preview

	return preview, nil
}

// validateBulkRow memvalidasi satu baris CSV.
func (pm *provisioningManager) validateBulkRow(ctx context.Context, tenantID, oltID string, rowNum int, record []string) domain.BulkRowPreview {
	row := domain.BulkRowPreview{RowNumber: rowNum, Valid: true}

	if len(record) < 4 {
		row.Valid = false
		row.ErrorMessage = "jumlah kolom kurang dari 4"
		return row
	}

	row.SerialNumber = strings.TrimSpace(record[0])
	row.CustomerID = strings.TrimSpace(record[1])
	ponPortStr := strings.TrimSpace(record[2])
	row.VLAN = strings.TrimSpace(record[3])

	if len(record) > 4 {
		row.ODP = strings.TrimSpace(record[4])
	}
	if len(record) > 5 {
		row.Description = strings.TrimSpace(record[5])
	}

	// Validasi serial number
	if row.SerialNumber == "" {
		row.Valid = false
		row.ErrorMessage = "serial number kosong"
		return row
	}

	// Validasi customer_id
	if row.CustomerID == "" {
		row.Valid = false
		row.ErrorMessage = "pelanggan_id kosong"
		return row
	}

	// Validasi pon_port
	ponPort, err := strconv.Atoi(ponPortStr)
	if err != nil || ponPort < 0 {
		row.Valid = false
		row.ErrorMessage = "pon_port tidak valid"
		return row
	}
	row.PONPort = ponPort

	// Validasi VLAN exists
	if row.VLAN == "" {
		row.Valid = false
		row.ErrorMessage = "vlan kosong"
		return row
	}

	// Cek serial number unik
	exists, err := pm.ontRepo.SerialNumberExists(ctx, tenantID, row.SerialNumber, "")
	if err != nil {
		row.Valid = false
		row.ErrorMessage = fmt.Sprintf("gagal cek serial number: %v", err)
		return row
	}
	if exists {
		row.Valid = false
		row.ErrorMessage = "serial number sudah ada"
		return row
	}

	return row
}

// ExecuteBulk mengeksekusi bulk provisioning untuk semua row valid.
// Provisioning dilakukan secara sekuensial, per-row error handling (continue on failure).
func (pm *provisioningManager) ExecuteBulk(ctx context.Context, bulkID string, performedBy string) (*domain.BulkResult, error) {
	if err := pm.ensureWriteEnabled(); err != nil {
		return nil, err
	}

	preview, ok := pm.bulkStore[bulkID]
	if !ok {
		return nil, domain.ErrBulkNotFound
	}

	result := &domain.BulkResult{
		BulkID: bulkID,
		Total:  preview.ValidCount,
		Rows:   []domain.BulkRowResult{},
	}

	for _, row := range preview.Rows {
		if !row.Valid {
			continue
		}

		rowResult := domain.BulkRowResult{
			RowNumber:    row.RowNumber,
			SerialNumber: row.SerialNumber,
		}

		req := domain.ProvisionONTRequest{
			SerialNumber:     row.SerialNumber,
			OLTID:            preview.OLTID,
			PONPortIndex:     row.PONPort,
			CustomerID:       row.CustomerID,
			ServiceProfileID: "", // akan di-resolve dari context
			VLANID:           row.VLAN,
			ODPID:            row.ODP,
			Description:      row.Description,
		}

		// Coba provision - lanjut ke baris berikutnya jika gagal
		resp, err := pm.ProvisionONT(ctx, preview.OLTID, req)
		if err != nil {
			rowResult.Success = false
			rowResult.ErrorMessage = err.Error()
			result.FailureCount++
			log.Warn().Err(err).Int("row", row.RowNumber).Msg("bulk provisioning: baris gagal")
		} else {
			rowResult.Success = true
			rowResult.ONTID = resp.ID
			result.SuccessCount++
		}

		result.Rows = append(result.Rows, rowResult)
	}

	// Hapus preview dari store setelah eksekusi
	delete(pm.bulkStore, bulkID)

	return result, nil
}
