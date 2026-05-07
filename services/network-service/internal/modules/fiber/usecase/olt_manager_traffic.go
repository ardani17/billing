package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetTraffic mengambil data traffic PON port dari TrafficStore.
func (m *oltManager) GetTraffic(ctx context.Context, oltID string, portIndex int, from, to time.Time) ([]domain.PONTrafficPoint, error) {
	if _, err := m.oltRepo.GetByID(ctx, oltID); err != nil {
		return nil, err
	}
	if portIndex < 0 {
		return nil, fmt.Errorf("port index tidak valid: %d", portIndex)
	}
	if m.trafficStore == nil {
		return []domain.PONTrafficPoint{}, nil
	}
	return m.trafficStore.Query(ctx, oltID, portIndex, from, to)
}
