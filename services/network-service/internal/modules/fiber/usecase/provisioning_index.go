package usecase

import (
	"context"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

const defaultMaxONTIndex = 128

func (pm *provisioningManager) resolveAvailableONTIndex(ctx context.Context, oltID string, ponPortIndex int) (int, error) {
	if ponPortIndex < 0 {
		return 0, fmt.Errorf("pon port tidak valid: %d", ponPortIndex)
	}
	for ontIndex := 1; ontIndex <= defaultMaxONTIndex; ontIndex++ {
		exists, err := pm.ontRepo.PositionExists(ctx, oltID, ponPortIndex, ontIndex, "")
		if err != nil {
			return 0, err
		}
		if !exists {
			return ontIndex, nil
		}
	}
	return 0, domain.ErrONTPositionExists
}
