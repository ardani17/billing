package usecase

import (
	"context"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	mikrotikadapter "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/adapter"
)

func commandBuilderForRouter(ctx context.Context, routerRepo domain.RouterRepository, routerID string) (domain.CommandBuilder, error) {
	router, err := routerRepo.GetByID(ctx, routerID)
	if err != nil {
		return nil, err
	}
	return mikrotikadapter.NewCommandBuilder(router.RouterOSVersion), nil
}
