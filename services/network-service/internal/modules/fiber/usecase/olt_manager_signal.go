package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

func (m *oltManager) GetSignal(ctx context.Context, oltID string, portIndex int, ontIndex int, from, to time.Time) ([]domain.ONTSignalPoint, error) {
	if _, err := m.oltRepo.GetByID(ctx, oltID); err != nil {
		return nil, err
	}
	if portIndex < 0 {
		return nil, fmt.Errorf("port index tidak valid")
	}
	if ontIndex <= 0 {
		return nil, fmt.Errorf("ont index tidak valid")
	}
	if m.signalStore == nil {
		return []domain.ONTSignalPoint{}, nil
	}
	return m.signalStore.Query(ctx, oltID, portIndex, ontIndex, from, to)
}
