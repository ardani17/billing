package adapter

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// sysUpTimeOID adalah OID standar sysUpTime untuk cek konektivitas SNMP.
const sysUpTimeOID = "1.3.6.1.2.1.1.3.0"

// Default timeout dan port untuk koneksi SNMP.
const (
	defaultSNMPPort       = 161
	defaultConnectTimeout = 5 * time.Second
	defaultRequestTimeout = 10 * time.Second
)

// snmpConnector mengimplementasikan domain.SNMPConnector menggunakan gosnmp.
type snmpConnector struct{}

// NewSNMPConnector membuat instance baru SNMPConnector.
func NewSNMPConnector() domain.SNMPConnector {
	return &snmpConnector{}
}

// Get melakukan SNMP GET untuk satu atau lebih OID.
func (c *snmpConnector) Get(ctx context.Context, cfg domain.SNMPConfig, oids []string) ([]domain.SNMPResult, error) {
	client, err := c.createClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.ConnectIPv4(); err != nil {
		return nil, classifySNMPError(err)
	}
	defer client.Conn.Close()

	result, err := client.Get(oids)
	if err != nil {
		return nil, classifySNMPError(err)
	}
	return convertPDUs(result.Variables), nil
}

// Walk melakukan SNMP WALK pada subtree OID.
func (c *snmpConnector) Walk(ctx context.Context, cfg domain.SNMPConfig, rootOID string) ([]domain.SNMPResult, error) {
	client, err := c.createClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.ConnectIPv4(); err != nil {
		return nil, classifySNMPError(err)
	}
	defer client.Conn.Close()

	pdus, err := client.WalkAll(rootOID)
	if err != nil {
		return nil, classifySNMPError(err)
	}
	return convertPDUs(pdus), nil
}

// GetBulk melakukan SNMP GETBULK untuk efisiensi pada tabel besar.
func (c *snmpConnector) GetBulk(ctx context.Context, cfg domain.SNMPConfig, oids []string, maxRepetitions int) ([]domain.SNMPResult, error) {
	client, err := c.createClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.ConnectIPv4(); err != nil {
		return nil, classifySNMPError(err)
	}
	defer client.Conn.Close()

	result, err := client.GetBulk(oids, 0, uint32(maxRepetitions))
	if err != nil {
		return nil, classifySNMPError(err)
	}
	return convertPDUs(result.Variables), nil
}

// Ping melakukan SNMP GET sysUpTime untuk cek konektivitas.
func (c *snmpConnector) Ping(ctx context.Context, cfg domain.SNMPConfig) error {
	_, err := c.Get(ctx, cfg, []string{sysUpTimeOID})
	return err
}

// createClient membuat gosnmp.GoSNMP dari domain.SNMPConfig.
func (c *snmpConnector) createClient(cfg domain.SNMPConfig) (*gosnmp.GoSNMP, error) {
	port := cfg.Port
	if port == 0 {
		port = defaultSNMPPort
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultRequestTimeout
	}

	client := &gosnmp.GoSNMP{
		Target:  cfg.Host,
		Port:    uint16(port),
		Timeout: timeout,
	}

	switch cfg.Version {
	case domain.SNMPv2c:
		client.Version = gosnmp.Version2c
		client.Community = cfg.Community
	case domain.SNMPv3:
		client.Version = gosnmp.Version3
		client.SecurityModel = gosnmp.UserSecurityModel
		usmParams := &gosnmp.UsmSecurityParameters{
			UserName: cfg.Username,
		}
		// Tentukan level keamanan berdasarkan konfigurasi
		switch {
		case cfg.AuthProtocol != "" && cfg.PrivProtocol != "":
			client.MsgFlags = gosnmp.AuthPriv
			usmParams.AuthenticationProtocol = mapAuthProtocol(cfg.AuthProtocol)
			usmParams.AuthenticationPassphrase = cfg.AuthPassword
			usmParams.PrivacyProtocol = mapPrivProtocol(cfg.PrivProtocol)
			usmParams.PrivacyPassphrase = cfg.PrivPassword
		case cfg.AuthProtocol != "":
			client.MsgFlags = gosnmp.AuthNoPriv
			usmParams.AuthenticationProtocol = mapAuthProtocol(cfg.AuthProtocol)
			usmParams.AuthenticationPassphrase = cfg.AuthPassword
		default:
			client.MsgFlags = gosnmp.NoAuthNoPriv
		}
		client.SecurityParameters = usmParams
	default:
		return nil, fmt.Errorf("versi SNMP tidak didukung: %s", cfg.Version)
	}

	return client, nil
}

// mapAuthProtocol mengkonversi string auth protocol ke gosnmp.SnmpV3AuthProtocol.
func mapAuthProtocol(proto string) gosnmp.SnmpV3AuthProtocol {
	switch strings.ToUpper(proto) {
	case "SHA":
		return gosnmp.SHA
	default:
		return gosnmp.MD5
	}
}

// mapPrivProtocol mengkonversi string priv protocol ke gosnmp.SnmpV3PrivProtocol.
func mapPrivProtocol(proto string) gosnmp.SnmpV3PrivProtocol {
	switch strings.ToUpper(proto) {
	case "AES":
		return gosnmp.AES
	default:
		return gosnmp.DES
	}
}

// convertPDUs mengkonversi slice gosnmp.SnmpPDU ke slice domain.SNMPResult.
func convertPDUs(pdus []gosnmp.SnmpPDU) []domain.SNMPResult {
	results := make([]domain.SNMPResult, 0, len(pdus))
	for _, pdu := range pdus {
		results = append(results, convertPDU(pdu))
	}
	return results
}

// convertPDU mengkonversi satu gosnmp.SnmpPDU ke domain.SNMPResult.
func convertPDU(pdu gosnmp.SnmpPDU) domain.SNMPResult {
	result := domain.SNMPResult{OID: pdu.Name}
	switch pdu.Type {
	case gosnmp.Integer:
		result.Type = domain.SNMPValueInteger
		result.Value = gosnmp.ToBigInt(pdu.Value).Int64()
	case gosnmp.OctetString:
		result.Type = domain.SNMPValueString
		result.Value = string(pdu.Value.([]byte))
	case gosnmp.Counter32:
		result.Type = domain.SNMPValueCounter32
		result.Value = gosnmp.ToBigInt(pdu.Value).Int64()
	case gosnmp.Counter64:
		result.Type = domain.SNMPValueCounter64
		result.Value = gosnmp.ToBigInt(pdu.Value).Int64()
	case gosnmp.Gauge32:
		result.Type = domain.SNMPValueGauge32
		result.Value = gosnmp.ToBigInt(pdu.Value).Int64()
	case gosnmp.TimeTicks:
		result.Type = domain.SNMPValueTimeTicks
		result.Value = gosnmp.ToBigInt(pdu.Value).Int64()
	default:
		result.Type = domain.SNMPValueString
		result.Value = fmt.Sprintf("%v", pdu.Value)
	}
	return result
}

// classifySNMPError mengklasifikasikan error SNMP ke domain error yang sesuai.
func classifySNMPError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := strings.ToLower(err.Error())

	// Deteksi timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return domain.ErrSNMPTimeout
	}
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
		return domain.ErrSNMPTimeout
	}

	// Deteksi error autentikasi
	if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "security") ||
		strings.Contains(errMsg, "unknown user") || strings.Contains(errMsg, "wrong digest") {
		return domain.ErrSNMPAuthFailed
	}

	// Default: koneksi gagal
	return domain.ErrSNMPConnectionFailed
}
