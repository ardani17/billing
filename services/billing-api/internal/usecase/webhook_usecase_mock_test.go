// untuk unit test WebhookUsecase.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/gateway"
	"github.com/rs/zerolog"
)

type whMockWebhookLogRepo struct {
	logs             map[string]*domain.WebhookLog
	alreadyProcessed map[string]bool // key: "externalID|eventType"
}

func newWhMockWebhookLogRepo() *whMockWebhookLogRepo {
	return &whMockWebhookLogRepo{
		logs:             make(map[string]*domain.WebhookLog),
		alreadyProcessed: make(map[string]bool),
	}
}

func (m *whMockWebhookLogRepo) Create(_ context.Context, l *domain.WebhookLog) (*domain.WebhookLog, error) {
	if l.ID == "" {
		l.ID = fmt.Sprintf("wh-%d", len(m.logs)+1)
	}
	l.CreatedAt = time.Now()
	cp := *l
	m.logs[cp.ID] = &cp
	return &cp, nil
}

func (m *whMockWebhookLogRepo) GetByID(_ context.Context, id string) (*domain.WebhookLog, error) {
	l, ok := m.logs[id]
	if !ok {
		return nil, fmt.Errorf("webhook log tidak ditemukan")
	}
	cp := *l
	return &cp, nil
}

func (m *whMockWebhookLogRepo) UpdateStatus(_ context.Context, id string, status domain.WebhookProcessingStatus, errMsg string) error {
	l, ok := m.logs[id]
	if !ok {
		return fmt.Errorf("webhook log tidak ditemukan")
	}
	l.ProcessingStatus = status
	l.ErrorMessage = errMsg
	return nil
}

func (m *whMockWebhookLogRepo) UpdateSignatureValid(_ context.Context, id string, valid bool) error {
	l, ok := m.logs[id]
	if !ok {
		return fmt.Errorf("webhook log tidak ditemukan")
	}
	l.SignatureValid = &valid
	return nil
}

func (m *whMockWebhookLogRepo) IsAlreadyProcessed(_ context.Context, extID, eventType string) (bool, error) {
	return m.alreadyProcessed[extID+"|"+eventType], nil
}

func (m *whMockWebhookLogRepo) ListByPaymentLink(_ context.Context, extID string) ([]*domain.WebhookLog, error) {
	var result []*domain.WebhookLog
	for _, l := range m.logs {
		if l.ExternalID == extID {
			cp := *l
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *whMockWebhookLogRepo) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type whMockPaymentRepo struct{ payments []*domain.InvoicePayment }

func (m *whMockPaymentRepo) Create(_ context.Context, p *domain.InvoicePayment) (*domain.InvoicePayment, error) {
	cp := *p
	m.payments = append(m.payments, &cp)
	return &cp, nil
}
func (m *whMockPaymentRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoicePayment, error) {
	return nil, nil
}
func (m *whMockPaymentRepo) VoidPayment(_ context.Context, _, _, _ string) error { return nil }
func (m *whMockPaymentRepo) GetByID(_ context.Context, _ string) (*domain.InvoicePayment, error) {
	return nil, nil
}
func (m *whMockPaymentRepo) ListWithFilters(_ context.Context, _ domain.PaymentListParams) (*domain.PaymentListResult, error) {
	return nil, nil
}
func (m *whMockPaymentRepo) GetSummary(_ context.Context, _, _ string, _, _ *int) (*domain.PaymentSummary, error) {
	return nil, nil
}
func (m *whMockPaymentRepo) FindDuplicate(_ context.Context, _ string, _ int64, _ string, _ time.Time) (bool, error) {
	return false, nil
}

type whMockAuditRepo struct{ logs []*domain.InvoiceAuditLog }

func (m *whMockAuditRepo) Create(_ context.Context, l *domain.InvoiceAuditLog) error {
	cp := *l
	m.logs = append(m.logs, &cp)
	return nil
}
func (m *whMockAuditRepo) ListByInvoice(_ context.Context, _ string) ([]*domain.InvoiceAuditLog, error) {
	return nil, nil
}

type whMockReceiptSeqRepo struct{ seq int }

func (m *whMockReceiptSeqRepo) NextSequence(_ context.Context, _ string, _, _ int) (int, error) {
	m.seq++
	return m.seq, nil
}

type whTestSetup struct {
	uc           *WebhookUsecase
	webhookRepo  *whMockWebhookLogRepo
	linkRepo     *gwMockLinkRepo
	invoiceRepo  *gwMockInvoiceRepo
	paymentRepo  *whMockPaymentRepo
	auditRepo    *whMockAuditRepo
	configRepo   *gwMockConfigRepo
	customerRepo *gwMockCustomerRepo
}

// pool dan queueClient nil - hanya early-exit path dan event sederhana yang ditest.
func setupWebhookUsecase() *whTestSetup {
	webhookRepo := newWhMockWebhookLogRepo()
	linkRepo := newGwMockLinkRepo()
	invoiceRepo := newGwMockInvoiceRepo()
	paymentRepo := &whMockPaymentRepo{}
	auditRepo := &whMockAuditRepo{}
	customerRepo := newGwMockCustomerRepo()
	configRepo := newGwMockConfigRepo()
	logger := zerolog.New(io.Discard)
	uc := NewWebhookUsecase(
		webhookRepo, linkRepo, invoiceRepo, paymentRepo,
		auditRepo, &whMockReceiptSeqRepo{}, customerRepo, configRepo,
		nil, nil, testMasterKey, logger,
	)
	return &whTestSetup{
		uc: uc, webhookRepo: webhookRepo, linkRepo: linkRepo,
		invoiceRepo: invoiceRepo, paymentRepo: paymentRepo,
		auditRepo: auditRepo, configRepo: configRepo, customerRepo: customerRepo,
	}
}

// seedWebhookConfig menyiapkan gateway config Xendit terenkripsi untuk test.
func seedWebhookConfig(s *whTestSetup) {
	enc, _ := gateway.EncryptAESGCM("xnd_production_test_key_12345", testMasterKey)
	secEnc, _ := gateway.EncryptAESGCM("whsec_callback_token_12345", testMasterKey)
	s.configRepo.configs["cfg-1"] = &domain.GatewayConfig{
		ID: "cfg-1", TenantID: "tenant-1",
		GatewayProvider: domain.GatewayXendit, IsActive: true,
		APIKeyEncrypted: enc, WebhookSecretEncrypted: secEnc,
		EnabledMethods: []string{"va_bca"}, PaymentLinkExpiryDays: 7,
	}
}

func seedWebhookLog(s *whTestSetup, extID string, body map[string]interface{}) string {
	raw, _ := json.Marshal(body)
	log := &domain.WebhookLog{
		ID: "wlog-1", GatewayProvider: domain.GatewayXendit,
		EventType: "invoice.paid", ExternalID: extID,
		RequestBody: raw, SourceIP: "1.2.3.4",
		ProcessingStatus: domain.WebhookReceived,
	}
	s.webhookRepo.logs[log.ID] = log
	return log.ID
}
