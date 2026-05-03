// File ini berisi implementasi Create untuk OLT Manager
// beserta helper enkripsi kredensial dan auto-detect.
package usecase

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Create membuat OLT baru, encrypt credentials, simpan ke DB,
// test SNMP (best-effort), auto-detect brand/model/firmware, dan return OLTResponse.
func (m *oltManager) Create(ctx context.Context, tenantID string, req domain.CreateOLTRequest) (*domain.OLTResponse, error) {
	// Validasi nama unik di tenant
	exists, err := m.oltRepo.NameExists(ctx, tenantID, req.Name, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrOLTNameExists
	}

	// Enkripsi semua kredensial
	creds, err := m.encryptCredentials(req)
	if err != nil {
		return nil, err
	}

	// Set default values
	snmpPort := req.SNMPPort
	if snmpPort == 0 {
		snmpPort = 161
	}
	interval := req.HealthCheckIntervalSec
	if interval == 0 {
		interval = 300
	}

	olt := &domain.OLT{
		TenantID:                   tenantID,
		Name:                       req.Name,
		Host:                       req.Host,
		SNMPVersion:                domain.SNMPVersion(req.SNMPVersion),
		SNMPPort:                   snmpPort,
		SNMPCommunityEncrypted:     creds.community,
		SNMPUsername:               req.SNMPUsername,
		SNMPAuthProtocol:           req.SNMPAuthProtocol,
		SNMPAuthPasswordEncrypted:  creds.authPass,
		SNMPPrivProtocol:           req.SNMPPrivProtocol,
		SNMPPrivPasswordEncrypted:  creds.privPass,
		CLIProtocol:                domain.CLIProtocol(req.CLIProtocol),
		CLIPort:                    req.CLIPort,
		CLIUsername:                req.CLIUsername,
		CLIPasswordEncrypted:       creds.cliPass,
		CLIEnablePasswordEncrypted: creds.enablePass,
		Status:                     domain.OLTStatusOffline,
		HealthCheckIntervalSec:     interval,
		Notes:                      req.Notes,
	}

	// Simpan OLT ke database (status awal: offline)
	created, err := m.oltRepo.Create(ctx, olt)
	if err != nil {
		return nil, err
	}

	// Best-effort: test SNMP dan auto-detect brand/model/firmware
	m.tryAutoDetect(ctx, created)

	// Tambahkan ke health checker jika tersedia
	if m.healthChecker != nil {
		m.healthChecker.AddOLT(created)
	}

	return oltToResponse(created), nil
}

// tryAutoDetect mencoba koneksi SNMP dan auto-detect info OLT.
// Jika gagal, OLT tetap disimpan sebagai offline tanpa error.
func (m *oltManager) tryAutoDetect(ctx context.Context, olt *domain.OLT) {
	snmpCfg, err := m.buildSNMPConfig(olt)
	if err != nil {
		log.Warn().Err(err).Str("olt_id", olt.ID).Msg("gagal build SNMP config untuk auto-detect")
		return
	}

	cliCfg, _ := m.buildCLIConfig(olt)
	adapter, err := m.factory.CreateAdapter("", snmpCfg, cliCfg)
	if err != nil {
		log.Warn().Err(err).Str("olt_id", olt.ID).Msg("gagal buat adapter untuk auto-detect")
		return
	}

	sysInfo, err := adapter.GetSystemInfo(ctx)
	if err != nil {
		log.Warn().Err(err).Str("olt_id", olt.ID).Msg("gagal auto-detect OLT, tetap disimpan sebagai offline")
		return
	}

	// Update OLT dengan info yang terdeteksi
	olt.Brand = sysInfo.Brand
	olt.Model = sysInfo.Model
	olt.FirmwareVersion = sysInfo.FirmwareVersion
	olt.PONPortCount = sysInfo.PONPortCount
	olt.TotalONTCount = sysInfo.TotalONTCount
	olt.Status = domain.OLTStatusOnline
	now := time.Now()
	olt.LastOnlineAt = &now

	updated, updateErr := m.oltRepo.Update(ctx, olt)
	if updateErr != nil {
		log.Error().Err(updateErr).Str("olt_id", olt.ID).Msg("gagal update OLT setelah auto-detect")
		return
	}
	*olt = *updated
}

// encryptedCreds menyimpan hasil enkripsi semua kredensial.
type encryptedCreds struct {
	community  string
	authPass   string
	privPass   string
	cliPass    string
	enablePass string
}

// encryptCredentials mengenkripsi semua kredensial dari CreateOLTRequest.
func (m *oltManager) encryptCredentials(req domain.CreateOLTRequest) (encryptedCreds, error) {
	var creds encryptedCreds
	var err error

	if req.SNMPCommunity != "" {
		creds.community, err = m.encryptor.Encrypt(req.SNMPCommunity)
		if err != nil {
			log.Error().Err(err).Msg("gagal enkripsi SNMP community")
			return creds, domain.ErrEncryptionFailed
		}
	}
	if req.SNMPAuthPassword != "" {
		creds.authPass, err = m.encryptor.Encrypt(req.SNMPAuthPassword)
		if err != nil {
			log.Error().Err(err).Msg("gagal enkripsi SNMP auth password")
			return creds, domain.ErrEncryptionFailed
		}
	}
	if req.SNMPPrivPassword != "" {
		creds.privPass, err = m.encryptor.Encrypt(req.SNMPPrivPassword)
		if err != nil {
			log.Error().Err(err).Msg("gagal enkripsi SNMP priv password")
			return creds, domain.ErrEncryptionFailed
		}
	}
	creds.cliPass, err = m.encryptor.Encrypt(req.CLIPassword)
	if err != nil {
		log.Error().Err(err).Msg("gagal enkripsi CLI password")
		return creds, domain.ErrEncryptionFailed
	}
	if req.CLIEnablePassword != "" {
		creds.enablePass, err = m.encryptor.Encrypt(req.CLIEnablePassword)
		if err != nil {
			log.Error().Err(err).Msg("gagal enkripsi CLI enable password")
			return creds, domain.ErrEncryptionFailed
		}
	}

	return creds, nil
}
