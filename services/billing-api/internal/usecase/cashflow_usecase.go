package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
	"github.com/ispboss/ispboss/services/billing-api/internal/repository"
	"github.com/rs/zerolog"
)

type CashflowUsecase struct {
	repo   *repository.CashflowRepo
	logger zerolog.Logger
}

func NewCashflowUsecase(repo *repository.CashflowRepo, logger zerolog.Logger) *CashflowUsecase {
	return &CashflowUsecase{repo: repo, logger: logger}
}

func (uc *CashflowUsecase) Summary(ctx context.Context, tenantID string, start, end time.Time) (*domain.CashflowSummary, error) {
	return uc.repo.Summary(ctx, tenantID, start, end)
}

func (uc *CashflowUsecase) Transactions(ctx context.Context, tenantID string, start, end time.Time, direction, source, category, search string) ([]domain.CashflowTransaction, error) {
	return uc.repo.Transactions(ctx, tenantID, start, end, direction, source, category, search)
}

func (uc *CashflowUsecase) Trend(ctx context.Context, tenantID string, start, end time.Time) ([]domain.CashflowTrendPoint, error) {
	return uc.repo.Trend(ctx, tenantID, start, end)
}

func (uc *CashflowUsecase) CreateManualTransaction(ctx context.Context, tenantID string, req domain.CreateManualCashflowRequest, actor domain.ActorInfo) (domain.CashflowTransaction, error) {
	return uc.repo.CreateManualTransaction(ctx, tenantID, req, actor)
}
