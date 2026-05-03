// File ini berisi implementasi GetByID, Update, Delete, dan List untuk OLT Manager.
package usecase

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// GetByID mengambil detail OLT termasuk alarm count aktif.
// Menambahkan warning jika SNMP v2c digunakan tanpa VPN.
func (m *oltManager) GetByID(ctx context.Context, id string) (*domain.OLTDetailResponse, error) {
	olt, err := m.oltRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Hitung alarm aktif untuk OLT ini
	alarmCount, err := m.alarmRepo.CountActive(ctx, olt.ID)
	if err != nil {
		log.Warn().Err(err).Str("olt_id", id).Msg("gagal hitung alarm aktif")
		alarmCount = 0
	}

	resp := &domain.OLTDetailResponse{
		OLTResponse:      *oltToResponse(olt),
		SNMPVersion:      olt.SNMPVersion,
		CLIProtocol:      olt.CLIProtocol,
		CLIPort:          olt.CLIPort,
		ActiveAlarmCount: alarmCount,
	}

	// Warning jika SNMP v2c (community string tidak terenkripsi di jaringan)
	if olt.SNMPVersion == domain.SNMPv2c {
		resp.Warning = "SNMP v2c mengirim community string tanpa enkripsi, pertimbangkan gunakan VPN atau upgrade ke SNMPv3"
	}

	return resp, nil
}

// Update memperbarui data OLT. Encrypt kredensial baru jika diberikan.
func (m *oltManager) Update(ctx context.Context, id string, req domain.UpdateOLTRequest) (*domain.OLTResponse, error) {
	olt, err := m.oltRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi nama unik jika nama berubah
	if req.Name != "" && req.Name != olt.Name {
		exists, nameErr := m.oltRepo.NameExists(ctx, olt.TenantID, req.Name, olt.ID)
		if nameErr != nil {
			return nil, nameErr
		}
		if exists {
			return nil, domain.ErrOLTNameExists
		}
		olt.Name = req.Name
	}

	// Validasi transisi status jika diminta
	if req.Status != "" {
		target := domain.OLTStatus(req.Status)
		if !domain.CanTransitionOLT(olt.Status, target) {
			return nil, domain.ErrOLTInvalidStatusTransition
		}
		olt.Status = target
	}

	// Update field-field yang diberikan
	applyUpdateFields(olt, req)

	// Encrypt kredensial baru jika diberikan
	if err := m.encryptUpdateCredentials(olt, req); err != nil {
		return nil, err
	}

	updated, err := m.oltRepo.Update(ctx, olt)
	if err != nil {
		return nil, err
	}

	return oltToResponse(updated), nil
}

// applyUpdateFields menerapkan field non-credential dari UpdateOLTRequest ke OLT.
func applyUpdateFields(olt *domain.OLT, req domain.UpdateOLTRequest) {
	if req.Host != "" {
		olt.Host = req.Host
	}
	if req.SNMPVersion != "" {
		olt.SNMPVersion = domain.SNMPVersion(req.SNMPVersion)
	}
	if req.SNMPUsername != "" {
		olt.SNMPUsername = req.SNMPUsername
	}
	if req.SNMPAuthProtocol != "" {
		olt.SNMPAuthProtocol = req.SNMPAuthProtocol
	}
	if req.SNMPPrivProtocol != "" {
		olt.SNMPPrivProtocol = req.SNMPPrivProtocol
	}
	if req.CLIProtocol != "" {
		olt.CLIProtocol = domain.CLIProtocol(req.CLIProtocol)
	}
	if req.CLIPort != nil {
		olt.CLIPort = *req.CLIPort
	}
	if req.CLIUsername != "" {
		olt.CLIUsername = req.CLIUsername
	}
	if req.HealthCheckIntervalSec != nil {
		olt.HealthCheckIntervalSec = *req.HealthCheckIntervalSec
	}
	if req.Notes != "" {
		olt.Notes = req.Notes
	}
}

// encryptUpdateCredentials mengenkripsi kredensial baru dari UpdateOLTRequest.
func (m *oltManager) encryptUpdateCredentials(olt *domain.OLT, req domain.UpdateOLTRequest) error {
	if req.SNMPCommunity != "" {
		enc, err := m.encryptor.Encrypt(req.SNMPCommunity)
		if err != nil {
			return domain.ErrEncryptionFailed
		}
		olt.SNMPCommunityEncrypted = enc
	}
	if req.SNMPAuthPassword != "" {
		enc, err := m.encryptor.Encrypt(req.SNMPAuthPassword)
		if err != nil {
			return domain.ErrEncryptionFailed
		}
		olt.SNMPAuthPasswordEncrypted = enc
	}
	if req.SNMPPrivPassword != "" {
		enc, err := m.encryptor.Encrypt(req.SNMPPrivPassword)
		if err != nil {
			return domain.ErrEncryptionFailed
		}
		olt.SNMPPrivPasswordEncrypted = enc
	}
	if req.CLIPassword != "" {
		enc, err := m.encryptor.Encrypt(req.CLIPassword)
		if err != nil {
			return domain.ErrEncryptionFailed
		}
		olt.CLIPasswordEncrypted = enc
	}
	if req.CLIEnablePassword != "" {
		enc, err := m.encryptor.Encrypt(req.CLIEnablePassword)
		if err != nil {
			return domain.ErrEncryptionFailed
		}
		olt.CLIEnablePasswordEncrypted = enc
	}
	return nil
}

// Delete melakukan soft-delete OLT dan menghapus dari health checker.
func (m *oltManager) Delete(ctx context.Context, id string) error {
	if err := m.oltRepo.SoftDelete(ctx, id); err != nil {
		return err
	}
	if m.healthChecker != nil {
		m.healthChecker.RemoveOLT(id)
	}
	return nil
}

// List mengambil daftar OLT dengan paginasi dan filter.
func (m *oltManager) List(ctx context.Context, params domain.OLTListParams) (*domain.OLTListResult, error) {
	return m.oltRepo.List(ctx, params)
}
