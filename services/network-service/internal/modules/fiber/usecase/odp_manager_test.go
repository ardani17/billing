package usecase

import (
	"context"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// =============================================================================

type mockODPRepo struct {
	odps       map[string]*domain.ODP
	nameExists bool
	createErr  error
	getErr     error
	updateErr  error
	deleteErr  error
	listResult *domain.ODPListResult
}

func newMockODPRepo() *mockODPRepo {
	return &mockODPRepo{odps: make(map[string]*domain.ODP)}
}

func (r *mockODPRepo) Create(_ context.Context, odp *domain.ODP) (*domain.ODP, error) {
	if r.createErr != nil {
		return nil, r.createErr
	}
	odp.ID = "odp-test-001"
	odp.CreatedAt = time.Now()
	odp.UpdatedAt = time.Now()
	r.odps[odp.ID] = odp
	return odp, nil
}

func (r *mockODPRepo) GetByID(_ context.Context, id string) (*domain.ODP, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	odp, ok := r.odps[id]
	if !ok {
		return nil, domain.ErrODPNotFound
	}
	return odp, nil
}

func (r *mockODPRepo) Update(_ context.Context, odp *domain.ODP) (*domain.ODP, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	odp.UpdatedAt = time.Now()
	r.odps[odp.ID] = odp
	return odp, nil
}

func (r *mockODPRepo) SoftDelete(_ context.Context, id string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.odps, id)
	return nil
}

func (r *mockODPRepo) List(_ context.Context, _ domain.ODPListParams) (*domain.ODPListResult, error) {
	if r.listResult != nil {
		return r.listResult, nil
	}
	return &domain.ODPListResult{Data: []*domain.ODPResponse{}, Total: 0, Page: 1, PageSize: 20, TotalPages: 0}, nil
}

func (r *mockODPRepo) NameExists(_ context.Context, _, _, _ string) (bool, error) {
	return r.nameExists, nil
}

func (r *mockODPRepo) GetByOLTAndPort(_ context.Context, _ string, _ int) ([]*domain.ODP, error) {
	return nil, nil
}

// =============================================================================
// =============================================================================

func newTestODPManager() (*odpManager, *mockODPRepo) {
	repo := newMockODPRepo()
	mgr := NewODPManager(repo, nil).(*odpManager)
	return mgr, repo
}

// createTestODPRequest membuat CreateODPRequest standar untuk testing.
func createTestODPRequest() domain.CreateODPRequest {
	return domain.CreateODPRequest{
		OLTID:        "olt-001",
		PONPortIndex: 0,
		Name:         "ODP-Test-01",
		SplitterType: domain.SplitterType1x8,
		Address:      "Jl. Merdeka No. 10",
	}
}
