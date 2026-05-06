package usecase

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

type billingSettingsMockRepo struct {
	settings *domain.BillingSettings
	err      error
	saved    *domain.BillingSettings
}

func (m *billingSettingsMockRepo) GetByTenantID(_ context.Context, tenantID string) (*domain.BillingSettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.settings == nil {
		return nil, domain.ErrBillingSettingsNotFound
	}
	m.settings.TenantID = tenantID
	return m.settings, nil
}

func (m *billingSettingsMockRepo) Upsert(_ context.Context, settings *domain.BillingSettings) (*domain.BillingSettings, error) {
	m.saved = settings
	return settings, nil
}

func (m *billingSettingsMockRepo) ListAll(_ context.Context) ([]*domain.BillingSettings, error) {
	return nil, nil
}

func TestBillingSettingsUsecaseGetReturnsDefaultWhenMissing(t *testing.T) {
	uc := NewBillingSettingsUsecase(&billingSettingsMockRepo{err: domain.ErrBillingSettingsNotFound}, zerolog.New(io.Discard))

	settings, err := uc.Get(context.Background(), "tenant-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if settings.TenantID != "tenant-1" {
		t.Fatalf("TenantID = %q, want tenant-1", settings.TenantID)
	}
	if settings.InvoicePrefix != "INV" || settings.Timezone != "Asia/Jakarta" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
}

func TestBillingSettingsUsecaseUpdateNormalizesAndSaves(t *testing.T) {
	repo := &billingSettingsMockRepo{}
	uc := NewBillingSettingsUsecase(repo, zerolog.New(io.Discard))

	_, err := uc.Update(context.Background(), "tenant-1", domain.UpdateBillingSettingsRequest{
		GenerateDays:       1,
		GracePeriodDays:    3,
		SuspendDays:        30,
		TaxEnabled:         true,
		TaxRate:            11,
		PenaltyEnabled:     true,
		PenaltyType:        domain.PenaltyFixed,
		PenaltyAmount:      5000,
		InvoicePrefix:      " inv ",
		NewCustomerBilling: "prorate",
		Timezone:           "Asia/Jakarta",
		AutoOpenIsolir:     true,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if repo.saved == nil {
		t.Fatal("expected settings to be saved")
	}
	if repo.saved.InvoicePrefix != "INV" {
		t.Fatalf("InvoicePrefix = %q, want INV", repo.saved.InvoicePrefix)
	}
	if repo.saved.TaxRate != 11 {
		t.Fatalf("TaxRate = %v, want 11", repo.saved.TaxRate)
	}
}

func TestBillingSettingsUsecaseUpdateRejectsInvalidPenalty(t *testing.T) {
	uc := NewBillingSettingsUsecase(&billingSettingsMockRepo{}, zerolog.New(io.Discard))

	_, err := uc.Update(context.Background(), "tenant-1", domain.UpdateBillingSettingsRequest{
		GenerateDays:       1,
		GracePeriodDays:    3,
		SuspendDays:        30,
		PenaltyEnabled:     true,
		PenaltyType:        domain.PenaltyFixed,
		InvoicePrefix:      "INV",
		NewCustomerBilling: "prorate",
		Timezone:           "Asia/Jakarta",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestBillingSettingsUsecaseGetPropagatesRepositoryError(t *testing.T) {
	expected := errors.New("db down")
	uc := NewBillingSettingsUsecase(&billingSettingsMockRepo{err: expected}, zerolog.New(io.Discard))

	_, err := uc.Get(context.Background(), "tenant-1")
	if !errors.Is(err, expected) {
		t.Fatalf("err = %v, want %v", err, expected)
	}
}
