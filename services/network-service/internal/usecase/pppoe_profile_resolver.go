package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

func (m *pppoeManager) resolveProfileForPackage(ctx context.Context, payload domain.PackageProfilePayload) (*domain.PPPoEProfile, error) {
	profile, err := m.profileRepo.GetByPackageID(ctx, payload.PackageID)
	if err == nil && profile != nil {
		return profile, nil
	}
	if err == nil && profile == nil {
		err = domain.ErrPPPoEProfileNotFound
	}
	if !errors.Is(err, domain.ErrPPPoEProfileNotFound) {
		return nil, err
	}
	if payload.MikrotikProfileName == "" {
		return nil, err
	}

	profile = &domain.PPPoEProfile{
		TenantID:      payload.TenantID,
		PackageID:     payload.PackageID,
		ProfileName:   payload.MikrotikProfileName,
		DownloadLimit: mbpsLimit(payload.DownloadMbps),
		UploadLimit:   mbpsLimit(payload.UploadMbps),
		AddressPool:   payload.AddressPool,
		OnlyOne:       true,
	}

	if profile.DownloadLimit == "" || profile.UploadLimit == "" {
		return profile, nil
	}

	created, createErr := m.profileRepo.Create(ctx, profile)
	if createErr == nil {
		return created, nil
	}

	existing, getErr := m.profileRepo.GetByProfileName(ctx, payload.TenantID, payload.MikrotikProfileName)
	if getErr == nil {
		return existing, nil
	}

	return profile, createErr
}

func mbpsLimit(mbps int) string {
	if mbps <= 0 {
		return ""
	}
	return fmt.Sprintf("%dM", mbps)
}
