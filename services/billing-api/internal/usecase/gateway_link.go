// gateway_link.go berisi method GatewayUsecase untuk generate, query, dan regenerasi payment link.
// Method walled garden dan sync ada di file terpisah (gateway_link_walled.go).
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
)

// GeneratePaymentLink membuat payment link baru via gateway untuk customer.
// Ambil config aktif, dekripsi API key, hitung total sisa, panggil adapter, simpan ke DB.
func (uc *GatewayUsecase) GeneratePaymentLink(ctx context.Context, req domain.GeneratePaymentLinkRequest) (*domain.PaymentLink, error) {
	configs, err := uc.configRepo.GetActiveByTenant(ctx, req.TenantID)
	if err != nil || len(configs) == 0 {
		return nil, domain.ErrNoActiveGateway
	}
	config := configs[0]
	// Dekripsi API key
	plainKey, err := gateway.DecryptAESGCM(config.APIKeyEncrypted, uc.masterKey)
	if err != nil {
		return nil, fmt.Errorf("%w: gagal dekripsi api_key", domain.ErrDecryptionFailed)
	}
	// Ambil invoice terbuka untuk customer
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice terbuka: %w", err)
	}
	if len(invoices) == 0 {
		return nil, fmt.Errorf("tidak ada invoice terbuka untuk customer")
	}

	// Filter invoice sesuai request (jika InvoiceIDs diisi)
	filtered := filterInvoicesByIDs(invoices, req.InvoiceIDs)
	if len(filtered) == 0 {
		return nil, fmt.Errorf("invoice yang diminta tidak ditemukan atau sudah lunas")
	}

	// Hitung total sisa pembayaran dan kumpulkan invoice IDs
	var totalAmount int64
	var invoiceIDs []string
	for _, inv := range filtered {
		remaining := inv.TotalAmount - inv.PaidAmount
		totalAmount += remaining
		invoiceIDs = append(invoiceIDs, inv.ID)
	}

	// Ambil data customer untuk deskripsi
	customer, err := uc.customerRepo.GetByID(ctx, req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data customer: %w", err)
	}

	// Buat deskripsi pembayaran
	var description string
	if len(filtered) == 1 {
		description = fmt.Sprintf("Pembayaran %s - %s", filtered[0].InvoiceNumber, customer.Name)
	} else {
		description = fmt.Sprintf("Pembayaran %d invoice - %s", len(filtered), customer.Name)
	}

	// Buat payment link ID dan adapter
	linkID := uuid.New().String()
	expiryDuration := time.Duration(config.PaymentLinkExpiryDays) * 24 * time.Hour
	adapter, err := gateway.NewAdapter(config.GatewayProvider, plainKey)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat adapter: %w", err)
	}

	linkReq := gateway.CreateLinkRequest{
		ExternalID:     linkID,
		Amount:         totalAmount,
		Description:    description,
		CustomerName:   customer.Name,
		CustomerEmail:  customer.Email,
		ExpiryDuration: expiryDuration,
		EnabledMethods: config.EnabledMethods,
	}

	resp, err := adapter.CreatePaymentLink(ctx, linkReq)
	if err != nil {
		uc.logger.Error().Err(err).Str("customer_id", req.CustomerID).Msg("gagal membuat payment link di gateway")
		return nil, fmt.Errorf("%w: %v", domain.ErrGatewayUnavailable, err)
	}

	// Simpan payment link + junction rows ke database
	link := &domain.PaymentLink{
		ID:              linkID,
		TenantID:        req.TenantID,
		CustomerID:      req.CustomerID,
		GatewayProvider: config.GatewayProvider,
		GatewayConfigID: config.ID,
		ExternalID:      resp.ExternalID,
		PaymentURL:      resp.PaymentURL,
		Amount:          totalAmount,
		Status:          domain.PaymentLinkActive,
		ExpiresAt:       resp.ExpiresAt,
	}

	created, err := uc.linkRepo.Create(ctx, link, invoiceIDs)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan payment link: %w", err)
	}

	uc.logger.Info().Str("link_id", created.ID).Str("customer_id", req.CustomerID).
		Int64("amount", totalAmount).Msg("payment link berhasil dibuat")

	return created, nil
}

// GetCustomerPaymentLink mengambil payment link aktif untuk customer. Nil jika tidak ada.
func (uc *GatewayUsecase) GetCustomerPaymentLink(ctx context.Context, customerID string) (*domain.CustomerPaymentLinkResponse, error) {
	link, err := uc.linkRepo.GetActiveByCustomer(ctx, customerID)
	if err != nil || link == nil {
		return nil, nil
	}

	// Ambil invoice terbuka untuk menghitung total arrears
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil invoice terbuka: %w", err)
	}

	items, totalArrears := buildOpenInvoiceItems(invoices)

	return &domain.CustomerPaymentLinkResponse{
		PaymentLink:  link,
		Invoices:     items,
		TotalArrears: totalArrears,
	}, nil
}

// RegeneratePaymentLink meng-expire link aktif lama dan membuat link baru dengan jumlah terbaru.
func (uc *GatewayUsecase) RegeneratePaymentLink(ctx context.Context, customerID string) (*domain.PaymentLink, error) {
	// Expire link aktif yang ada (jika ada)
	if err := uc.expireActiveLink(ctx, customerID); err != nil {
		uc.logger.Warn().Err(err).Str("customer_id", customerID).Msg("gagal expire link lama, lanjut generate baru")
	}

	// Ambil invoice terbuka untuk customer
	invoices, err := uc.invoiceRepo.FindOpenByCustomer(ctx, customerID)
	if err != nil || len(invoices) == 0 {
		return nil, fmt.Errorf("tidak ada invoice terbuka untuk customer")
	}

	// Ambil tenant_id dari invoice pertama
	tenantID := invoices[0].TenantID
	var invoiceIDs []string
	for _, inv := range invoices {
		invoiceIDs = append(invoiceIDs, inv.ID)
	}

	return uc.GeneratePaymentLink(ctx, domain.GeneratePaymentLinkRequest{
		TenantID:   tenantID,
		CustomerID: customerID,
		InvoiceIDs: invoiceIDs,
	})
}

// GetInvoicePaymentLinks mengambil semua payment links untuk invoice tertentu.
func (uc *GatewayUsecase) GetInvoicePaymentLinks(ctx context.Context, invoiceID string) ([]*domain.PaymentLink, error) {
	return uc.linkRepo.ListByInvoice(ctx, invoiceID)
}

// expireActiveLink meng-expire payment link aktif untuk customer via adapter dan repo.
func (uc *GatewayUsecase) expireActiveLink(ctx context.Context, customerID string) error {
	link, err := uc.linkRepo.GetActiveByCustomer(ctx, customerID)
	if err != nil || link == nil {
		return nil
	}
	// Expire di gateway via adapter (best effort)
	config, err := uc.configRepo.GetByID(ctx, link.GatewayConfigID)
	if err == nil && config != nil {
		plainKey, decErr := gateway.DecryptAESGCM(config.APIKeyEncrypted, uc.masterKey)
		if decErr == nil {
			adapter, adErr := gateway.NewAdapter(config.GatewayProvider, plainKey)
			if adErr == nil {
				_ = adapter.ExpirePaymentLink(ctx, link.ExternalID)
			}
		}
	}

	return uc.linkRepo.ExpireByID(ctx, link.ID)
}

// filterInvoicesByIDs memfilter invoice berdasarkan daftar ID yang diminta.
func filterInvoicesByIDs(invoices []*domain.Invoice, ids []string) []*domain.Invoice {
	idSet := make(map[string]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	var result []*domain.Invoice
	for _, inv := range invoices {
		if idSet[inv.ID] {
			result = append(result, inv)
		}
	}
	return result
}
// buildOpenInvoiceItems mengkonversi daftar invoice menjadi OpenInvoiceItem dan total arrears.
func buildOpenInvoiceItems(invoices []*domain.Invoice) ([]domain.OpenInvoiceItem, int64) {
	var items []domain.OpenInvoiceItem
	var totalArrears int64
	for _, inv := range invoices {
		remaining := inv.TotalAmount - inv.PaidAmount
		totalArrears += remaining
		items = append(items, domain.OpenInvoiceItem{
			ID:              inv.ID,
			InvoiceNumber:   inv.InvoiceNumber,
			PeriodMonth:     inv.PeriodMonth,
			PeriodYear:      inv.PeriodYear,
			TotalAmount:     inv.TotalAmount,
			PaidAmount:      inv.PaidAmount,
			RemainingAmount: remaining,
			Status:          inv.Status,
			DueDate:         inv.DueDate,
		})
	}
	if items == nil {
		items = []domain.OpenInvoiceItem{}
	}
	return items, totalArrears
}
