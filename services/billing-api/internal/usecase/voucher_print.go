// voucher_print.go berisi business logic untuk generate PDF voucher.
// Mengimplementasikan GeneratePDF pada VoucherPrintUsecase.
// Saat ini menggunakan implementasi placeholder yang menghasilkan PDF sederhana berbasis teks.
// Implementasi penuh dengan maroto/gofpdf akan ditambahkan kemudian.
package usecase

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// VoucherPrintUsecase mengimplementasikan business logic untuk generate PDF voucher.
type VoucherPrintUsecase struct {
	voucherRepo domain.VoucherRepository
	packageRepo domain.PackageRepository
	logger      zerolog.Logger
}

// NewVoucherPrintUsecase membuat instance baru VoucherPrintUsecase.
func NewVoucherPrintUsecase(
	voucherRepo domain.VoucherRepository,
	packageRepo domain.PackageRepository,
	logger zerolog.Logger,
) *VoucherPrintUsecase {
	return &VoucherPrintUsecase{
		voucherRepo: voucherRepo,
		packageRepo: packageRepo,
		logger:      logger,
	}
}

// voucherCardData berisi data yang ditampilkan pada setiap kartu voucher di PDF.
type voucherCardData struct {
	TenantName   string // nama tenant/ISP
	TenantPhone  string // kontak tenant
	VoucherCode  string // kode voucher
	PackageName  string // nama paket
	Bandwidth    string // bandwidth (download/upload)
	Duration     string // durasi paket
	SellPrice    int64  // harga jual
	ExpiryDate   string // tanggal kedaluwarsa (jika ada)
}

// GeneratePDF menghasilkan PDF berisi kartu-kartu voucher dalam layout grid.
// Setiap halaman A4 menampilkan 8-12 kartu voucher yang bisa dipotong.
// Setiap kartu menampilkan: nama tenant, kode voucher, nama paket, bandwidth,
// durasi, harga jual, tanggal kedaluwarsa, dan kontak tenant.
//
// Saat ini menggunakan implementasi placeholder yang menghasilkan dokumen teks sederhana.
// Implementasi penuh dengan maroto/gofpdf akan ditambahkan kemudian.
func (uc *VoucherPrintUsecase) GeneratePDF(ctx context.Context, voucherIDs []string, tenantName, tenantPhone string) ([]byte, error) {
	if len(voucherIDs) == 0 {
		return nil, fmt.Errorf("usecase: daftar voucher ID tidak boleh kosong")
	}

	// Ambil voucher berdasarkan IDs
	vouchers, err := uc.voucherRepo.GetByIDs(ctx, voucherIDs)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal mengambil voucher by IDs: %w", err)
	}

	if len(vouchers) == 0 {
		return nil, fmt.Errorf("usecase: tidak ada voucher ditemukan untuk IDs yang diberikan")
	}

	// Kumpulkan package_id unik untuk resolve info paket
	packageIDs := make(map[string]struct{})
	for _, v := range vouchers {
		packageIDs[v.PackageID] = struct{}{}
	}

	// Resolve info paket untuk setiap package_id unik
	packageMap := make(map[string]*domain.Package)
	for pkgID := range packageIDs {
		pkg, err := uc.packageRepo.GetByID(ctx, pkgID)
		if err != nil {
			uc.logger.Warn().Err(err).Str("package_id", pkgID).Msg("gagal mengambil info paket untuk PDF")
			continue
		}
		packageMap[pkgID] = pkg
	}

	// Siapkan data kartu voucher
	cards := make([]voucherCardData, 0, len(vouchers))
	for _, v := range vouchers {
		card := voucherCardData{
			TenantName:  tenantName,
			TenantPhone: tenantPhone,
			VoucherCode: v.Code,
		}

		// Resolve info paket
		if pkg, ok := packageMap[v.PackageID]; ok {
			card.PackageName = pkg.Name
			card.Bandwidth = fmt.Sprintf("%d/%d Mbps", pkg.DownloadMbps, pkg.UploadMbps)

			// Format durasi paket
			if pkg.DurationValue != nil && pkg.DurationUnit != "" {
				card.Duration = fmt.Sprintf("%d %s", *pkg.DurationValue, pkg.DurationUnit)
			}

			// Gunakan sell_price_snapshot jika voucher sudah dibeli, jika tidak gunakan harga paket saat ini
			if v.SellPriceSnapshot != nil {
				card.SellPrice = *v.SellPriceSnapshot
			} else if pkg.SellPrice != nil {
				card.SellPrice = *pkg.SellPrice
			}
		} else {
			// Fallback jika paket tidak ditemukan — gunakan data dari voucher
			card.PackageName = v.PackageName
			if v.SellPriceSnapshot != nil {
				card.SellPrice = *v.SellPriceSnapshot
			}
		}

		// Format tanggal kedaluwarsa
		if v.ExpiresAt != nil {
			card.ExpiryDate = v.ExpiresAt.Format("02 Jan 2006")
		}

		cards = append(cards, card)
	}

	// Generate PDF placeholder (teks sederhana)
	// TODO: Ganti dengan implementasi maroto/gofpdf untuk layout grid A4 yang sebenarnya
	pdfBytes, err := uc.generatePlaceholderPDF(cards)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal generate PDF: %w", err)
	}

	return pdfBytes, nil
}

// generatePlaceholderPDF menghasilkan dokumen teks sederhana sebagai placeholder PDF.
// Menampilkan kartu voucher dalam format teks dengan layout grid simulasi.
// Setiap "halaman" menampilkan hingga 12 kartu voucher.
//
// TODO: Ganti dengan implementasi maroto/gofpdf yang menghasilkan PDF A4 sebenarnya
// dengan grid layout 8-12 kartu per halaman yang bisa dipotong.
func (uc *VoucherPrintUsecase) generatePlaceholderPDF(cards []voucherCardData) ([]byte, error) {
	var buf bytes.Buffer
	cardsPerPage := 12
	separator := strings.Repeat("=", 50)
	cardSeparator := strings.Repeat("-", 50)

	for i, card := range cards {
		// Header halaman baru setiap 12 kartu
		if i%cardsPerPage == 0 {
			pageNum := (i / cardsPerPage) + 1
			if i > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(fmt.Sprintf("%s\n", separator))
			buf.WriteString(fmt.Sprintf("  VOUCHER PRINT — Halaman %d\n", pageNum))
			buf.WriteString(fmt.Sprintf("%s\n", separator))
		}

		// Render kartu voucher
		buf.WriteString(fmt.Sprintf("\n%s\n", cardSeparator))
		buf.WriteString(fmt.Sprintf("  %s\n", card.TenantName))
		buf.WriteString(fmt.Sprintf("  Kode    : %s\n", card.VoucherCode))
		buf.WriteString(fmt.Sprintf("  Paket   : %s\n", card.PackageName))
		if card.Bandwidth != "" {
			buf.WriteString(fmt.Sprintf("  Bandwidth: %s\n", card.Bandwidth))
		}
		if card.Duration != "" {
			buf.WriteString(fmt.Sprintf("  Durasi  : %s\n", card.Duration))
		}
		if card.SellPrice > 0 {
			buf.WriteString(fmt.Sprintf("  Harga   : Rp %d\n", card.SellPrice))
		}
		if card.ExpiryDate != "" {
			buf.WriteString(fmt.Sprintf("  Berlaku s/d: %s\n", card.ExpiryDate))
		}
		if card.TenantPhone != "" {
			buf.WriteString(fmt.Sprintf("  Kontak  : %s\n", card.TenantPhone))
		}
		buf.WriteString(fmt.Sprintf("%s\n", cardSeparator))
	}

	return buf.Bytes(), nil
}
