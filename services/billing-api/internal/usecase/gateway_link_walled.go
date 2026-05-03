// gateway_link_walled.go berisi method GatewayUsecase untuk walled garden dan sinkronisasi amount.
// Endpoint publik untuk halaman captive portal pelanggan yang diisolir.
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// GetWalledGardenPaymentInfo mengambil info pembayaran untuk walled garden.
// Jika tidak ada link aktif atau link sudah expired, generate link baru on-demand.
// Mengembalikan URL pembayaran, total tunggakan, dan detail invoice.
func (uc *GatewayUsecase) GetWalledGardenPaymentInfo(ctx context.Context, customerID string) (*domain.WalledGardenPaymentInfo, error) {
	// Ambil data customer
	customer, err := uc.customerRepo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data customer: %w", err)
	}

	// Ambil invoice terbuka untuk customer
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice terbuka: %w", err)
	}
	if len(invoices) == 0 {
		return &domain.WalledGardenPaymentInfo{
			TotalArrears: 0,
			Invoices:     []domain.OpenInvoiceItem{},
			CustomerName: customer.Name,
		}, nil
	}

	items, totalArrears := buildOpenInvoiceItems(invoices)

	// Cek apakah ada link aktif
	link, _ := uc.linkRepo.GetActiveByCustomer(ctx, customerID)

	// Jika link ada tapi sudah expired, expire dan set nil agar di-generate ulang
	if link != nil && link.ExpiresAt.Before(time.Now()) {
		_ = uc.expireActiveLink(ctx, customerID)
		link = nil
	}

	// Jika tidak ada link aktif, generate on-demand
	if link == nil {
		link, err = uc.generateWalledGardenLink(ctx, customer, invoices)
		if err != nil {
			uc.logger.Error().Err(err).
				Str("customer_id", customerID).
				Msg("gagal generate payment link untuk walled garden")
			// Tetap kembalikan info tanpa payment URL
			return &domain.WalledGardenPaymentInfo{
				TotalArrears: totalArrears,
				Invoices:     items,
				CustomerName: customer.Name,
			}, nil
		}
	}

	return &domain.WalledGardenPaymentInfo{
		PaymentURL:   link.PaymentURL,
		TotalArrears: totalArrears,
		Invoices:     items,
		CustomerName: customer.Name,
	}, nil
}

// generateWalledGardenLink membuat payment link baru untuk walled garden.
// Menggunakan semua invoice terbuka customer.
func (uc *GatewayUsecase) generateWalledGardenLink(
	ctx context.Context,
	customer *domain.Customer,
	invoices []*domain.Invoice,
) (*domain.PaymentLink, error) {
	var invoiceIDs []string
	for _, inv := range invoices {
		invoiceIDs = append(invoiceIDs, inv.ID)
	}

	return uc.GeneratePaymentLink(ctx, domain.GeneratePaymentLinkRequest{
		TenantID:   customer.TenantID,
		CustomerID: customer.ID,
		InvoiceIDs: invoiceIDs,
	})
}

// SyncPaymentLinkAmount menyinkronkan jumlah payment link setelah invoice berubah.
// Dipanggil saat ada event invoice.payment_recorded atau invoice.penalty_added.
// Expire link lama, generate link baru dengan jumlah terbaru.
func (uc *GatewayUsecase) SyncPaymentLinkAmount(ctx context.Context, invoiceID string) error {
	// Cari payment link aktif yang mencakup invoice ini
	links, err := uc.linkRepo.ListByInvoice(ctx, invoiceID)
	if err != nil {
		return fmt.Errorf("gagal mencari payment links: %w", err)
	}

	// Cari link aktif
	var activeLink *domain.PaymentLink
	for _, l := range links {
		if l.Status == domain.PaymentLinkActive {
			activeLink = l
			break
		}
	}
	if activeLink == nil {
		return nil // Tidak ada link aktif, tidak perlu sync
	}

	// Expire link aktif
	if err := uc.expireActiveLink(ctx, activeLink.CustomerID); err != nil {
		return fmt.Errorf("gagal expire link aktif: %w", err)
	}

	// Generate link baru dengan jumlah terbaru
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, activeLink.CustomerID)
	if err != nil || len(invoices) == 0 {
		return nil // Tidak ada invoice terbuka, tidak perlu link baru
	}

	var invoiceIDs []string
	for _, inv := range invoices {
		invoiceIDs = append(invoiceIDs, inv.ID)
	}

	_, err = uc.GeneratePaymentLink(ctx, domain.GeneratePaymentLinkRequest{
		TenantID:   activeLink.TenantID,
		CustomerID: activeLink.CustomerID,
		InvoiceIDs: invoiceIDs,
	})
	return err
}
