package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type mockPaymentLinkRepo struct {
	links    map[string]*domain.PaymentLink
	junction map[string][]string // linkID -> invoiceIDs
}

func newMockPaymentLinkRepo() *mockPaymentLinkRepo {
	return &mockPaymentLinkRepo{
		links:    make(map[string]*domain.PaymentLink),
		junction: make(map[string][]string),
	}
}

func (m *mockPaymentLinkRepo) Create(_ context.Context, l *domain.PaymentLink, invoiceIDs []string) (*domain.PaymentLink, error) {
	l.CreatedAt = time.Now()
	l.UpdatedAt = time.Now()
	cp := *l
	m.links[cp.ID] = &cp
	m.junction[cp.ID] = invoiceIDs
	return &cp, nil
}

func (m *mockPaymentLinkRepo) GetByID(_ context.Context, id string) (*domain.PaymentLink, error) {
	l, ok := m.links[id]
	if !ok {
		return nil, domain.ErrPaymentLinkNotFound
	}
	cp := *l
	return &cp, nil
}

func (m *mockPaymentLinkRepo) GetByExternalID(_ context.Context, extID string) (*domain.PaymentLink, error) {
	for _, l := range m.links {
		if l.ExternalID == extID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, domain.ErrPaymentLinkNotFound
}

func (m *mockPaymentLinkRepo) GetActiveByCustomer(_ context.Context, customerID string) (*domain.PaymentLink, error) {
	for _, l := range m.links {
		if l.CustomerID == customerID && l.Status == domain.PaymentLinkActive {
			cp := *l
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockPaymentLinkRepo) GetInvoiceIDsByLinkID(_ context.Context, linkID string) ([]string, error) {
	ids, ok := m.junction[linkID]
	if !ok {
		return nil, nil
	}
	return ids, nil
}

func (m *mockPaymentLinkRepo) UpdateStatus(_ context.Context, id string, status domain.PaymentLinkStatus) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = status
	return nil
}

func (m *mockPaymentLinkRepo) UpdateStatusPaid(_ context.Context, id string, method string, paidAt time.Time) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = domain.PaymentLinkPaid
	l.PaidMethod = method
	l.PaidAt = &paidAt
	return nil
}

func (m *mockPaymentLinkRepo) ListByInvoice(_ context.Context, invoiceID string) ([]*domain.PaymentLink, error) {
	var result []*domain.PaymentLink
	for linkID, invIDs := range m.junction {
		for _, id := range invIDs {
			if id == invoiceID {
				if l, ok := m.links[linkID]; ok {
					cp := *l
					result = append(result, &cp)
				}
			}
		}
	}
	return result, nil
}

func (m *mockPaymentLinkRepo) FindExpired(_ context.Context, _ int) ([]*domain.PaymentLink, error) {
	return nil, nil
}

func (m *mockPaymentLinkRepo) ExpireByID(_ context.Context, id string) error {
	l, ok := m.links[id]
	if !ok {
		return domain.ErrPaymentLinkNotFound
	}
	l.Status = domain.PaymentLinkExpired
	return nil
}

type mockWebhookLogRepo struct {
	logs map[string]*domain.WebhookLog
}

func newMockWebhookLogRepo() *mockWebhookLogRepo {
	return &mockWebhookLogRepo{logs: make(map[string]*domain.WebhookLog)}
}

func (m *mockWebhookLogRepo) Create(_ context.Context, l *domain.WebhookLog) (*domain.WebhookLog, error) {
	if l.ID == "" {
		l.ID = fmt.Sprintf("wh-%d", len(m.logs)+1)
	}
	l.CreatedAt = time.Now()
	cp := *l
	m.logs[cp.ID] = &cp
	return &cp, nil
}

func (m *mockWebhookLogRepo) GetByID(_ context.Context, id string) (*domain.WebhookLog, error) {
	l, ok := m.logs[id]
	if !ok {
		return nil, fmt.Errorf("webhook log tidak ditemukan")
	}
	cp := *l
	return &cp, nil
}

func (m *mockWebhookLogRepo) UpdateStatus(_ context.Context, id string, status domain.WebhookProcessingStatus, errMsg string) error {
	l, ok := m.logs[id]
	if !ok {
		return fmt.Errorf("webhook log tidak ditemukan")
	}
	l.ProcessingStatus = status
	l.ErrorMessage = errMsg
	return nil
}

func (m *mockWebhookLogRepo) UpdateSignatureValid(_ context.Context, id string, valid bool) error {
	l, ok := m.logs[id]
	if !ok {
		return fmt.Errorf("webhook log tidak ditemukan")
	}
	l.SignatureValid = &valid
	return nil
}

func (m *mockWebhookLogRepo) IsAlreadyProcessed(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockWebhookLogRepo) ListByPaymentLink(_ context.Context, extID string) ([]*domain.WebhookLog, error) {
	var result []*domain.WebhookLog
	for _, l := range m.logs {
		if l.ExternalID == extID {
			cp := *l
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockWebhookLogRepo) DeleteOlderThan(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}
