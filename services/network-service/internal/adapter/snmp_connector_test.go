package adapter

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Test: Konfigurasi SNMP v2c
// =============================================================================

// TestCreateClient_V2c memverifikasi bahwa konfigurasi v2c menghasilkan
// client gosnmp dengan community string, port, dan timeout yang benar.
func TestCreateClient_V2c(t *testing.T) {
	c := &snmpConnector{}
	cfg := domain.SNMPConfig{
		Host:      "192.168.1.1",
		Port:      161,
		Version:   domain.SNMPv2c,
		Community: "public",
		Timeout:   10 * time.Second,
	}

	client, err := c.createClient(cfg)
	if err != nil {
		t.Fatalf("createClient gagal: %v", err)
	}
	if client.Version != gosnmp.Version2c {
		t.Errorf("version: got %v, want Version2c", client.Version)
	}
	if client.Community != "public" {
		t.Errorf("community: got %q, want %q", client.Community, "public")
	}
	if client.Port != 161 {
		t.Errorf("port: got %d, want 161", client.Port)
	}
	if client.Timeout != 10*time.Second {
		t.Errorf("timeout: got %v, want 10s", client.Timeout)
	}
	if client.Target != "192.168.1.1" {
		t.Errorf("target: got %q, want %q", client.Target, "192.168.1.1")
	}
}

// TestCreateClient_V2c_DefaultPort memverifikasi bahwa port default 161
// digunakan jika port tidak diset.
func TestCreateClient_V2c_DefaultPort(t *testing.T) {
	c := &snmpConnector{}
	cfg := domain.SNMPConfig{
		Host:      "10.0.0.1",
		Version:   domain.SNMPv2c,
		Community: "private",
	}

	client, err := c.createClient(cfg)
	if err != nil {
		t.Fatalf("createClient gagal: %v", err)
	}
	if client.Port != defaultSNMPPort {
		t.Errorf("port: got %d, want %d", client.Port, defaultSNMPPort)
	}
	if client.Timeout != defaultRequestTimeout {
		t.Errorf("timeout: got %v, want %v", client.Timeout, defaultRequestTimeout)
	}
}

// =============================================================================
// Test: Konfigurasi SNMP v3
// =============================================================================

// TestCreateClient_V3_AuthPriv memverifikasi konfigurasi v3 dengan
// autentikasi dan privasi (AuthPriv).
func TestCreateClient_V3_AuthPriv(t *testing.T) {
	c := &snmpConnector{}
	cfg := domain.SNMPConfig{
		Host:         "192.168.1.1",
		Port:         161,
		Version:      domain.SNMPv3,
		Username:     "admin",
		AuthProtocol: "SHA",
		AuthPassword: "authpass123",
		PrivProtocol: "AES",
		PrivPassword: "privpass123",
		Timeout:      5 * time.Second,
	}

	client, err := c.createClient(cfg)
	if err != nil {
		t.Fatalf("createClient gagal: %v", err)
	}
	if client.Version != gosnmp.Version3 {
		t.Errorf("version: got %v, want Version3", client.Version)
	}
	if client.MsgFlags != gosnmp.AuthPriv {
		t.Errorf("msgFlags: got %v, want AuthPriv", client.MsgFlags)
	}
	usm, ok := client.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok {
		t.Fatal("SecurityParameters bukan UsmSecurityParameters")
	}
	if usm.UserName != "admin" {
		t.Errorf("username: got %q, want %q", usm.UserName, "admin")
	}
	if usm.AuthenticationProtocol != gosnmp.SHA {
		t.Errorf("authProto: got %v, want SHA", usm.AuthenticationProtocol)
	}
	if usm.AuthenticationPassphrase != "authpass123" {
		t.Errorf("authPass: got %q, want %q", usm.AuthenticationPassphrase, "authpass123")
	}
	if usm.PrivacyProtocol != gosnmp.AES {
		t.Errorf("privProto: got %v, want AES", usm.PrivacyProtocol)
	}
	if usm.PrivacyPassphrase != "privpass123" {
		t.Errorf("privPass: got %q, want %q", usm.PrivacyPassphrase, "privpass123")
	}
}

// TestCreateClient_V3_AuthNoPriv memverifikasi konfigurasi v3 dengan
// autentikasi tanpa privasi (AuthNoPriv).
func TestCreateClient_V3_AuthNoPriv(t *testing.T) {
	c := &snmpConnector{}
	cfg := domain.SNMPConfig{
		Host:         "192.168.1.1",
		Version:      domain.SNMPv3,
		Username:     "monitor",
		AuthProtocol: "MD5",
		AuthPassword: "authonly",
	}

	client, err := c.createClient(cfg)
	if err != nil {
		t.Fatalf("createClient gagal: %v", err)
	}
	if client.MsgFlags != gosnmp.AuthNoPriv {
		t.Errorf("msgFlags: got %v, want AuthNoPriv", client.MsgFlags)
	}
	usm := client.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if usm.AuthenticationProtocol != gosnmp.MD5 {
		t.Errorf("authProto: got %v, want MD5", usm.AuthenticationProtocol)
	}
}

// TestCreateClient_V3_NoAuthNoPriv memverifikasi konfigurasi v3 tanpa
// autentikasi dan tanpa privasi (NoAuthNoPriv).
func TestCreateClient_V3_NoAuthNoPriv(t *testing.T) {
	c := &snmpConnector{}
	cfg := domain.SNMPConfig{
		Host:     "192.168.1.1",
		Version:  domain.SNMPv3,
		Username: "readonly",
	}

	client, err := c.createClient(cfg)
	if err != nil {
		t.Fatalf("createClient gagal: %v", err)
	}
	if client.MsgFlags != gosnmp.NoAuthNoPriv {
		t.Errorf("msgFlags: got %v, want NoAuthNoPriv", client.MsgFlags)
	}
}

// =============================================================================
// Test: Klasifikasi Error SNMP
// =============================================================================

// timeoutError mengimplementasikan net.Error dengan Timeout() = true.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// Pastikan timeoutError mengimplementasikan net.Error.
var _ net.Error = (*timeoutError)(nil)

// TestClassifySNMPError_Timeout memverifikasi bahwa net.Error timeout
// diklasifikasikan sebagai ErrSNMPTimeout.
func TestClassifySNMPError_Timeout(t *testing.T) {
	err := classifySNMPError(&timeoutError{})
	if !errors.Is(err, domain.ErrSNMPTimeout) {
		t.Errorf("got %v, want ErrSNMPTimeout", err)
	}
}

// TestClassifySNMPError_TimeoutString memverifikasi bahwa error dengan
// pesan "timeout" diklasifikasikan sebagai ErrSNMPTimeout.
func TestClassifySNMPError_TimeoutString(t *testing.T) {
	err := classifySNMPError(errors.New("request timeout exceeded"))
	if !errors.Is(err, domain.ErrSNMPTimeout) {
		t.Errorf("got %v, want ErrSNMPTimeout", err)
	}
}

// TestClassifySNMPError_Auth memverifikasi bahwa error autentikasi
// diklasifikasikan sebagai ErrSNMPAuthFailed.
func TestClassifySNMPError_Auth(t *testing.T) {
	cases := []string{
		"authentication failure",
		"security name not found",
		"unknown user name",
		"wrong digest value",
	}
	for _, msg := range cases {
		err := classifySNMPError(errors.New(msg))
		if !errors.Is(err, domain.ErrSNMPAuthFailed) {
			t.Errorf("msg=%q: got %v, want ErrSNMPAuthFailed", msg, err)
		}
	}
}

// TestClassifySNMPError_Connection memverifikasi bahwa error umum
// diklasifikasikan sebagai ErrSNMPConnectionFailed.
func TestClassifySNMPError_Connection(t *testing.T) {
	err := classifySNMPError(errors.New("connection refused"))
	if !errors.Is(err, domain.ErrSNMPConnectionFailed) {
		t.Errorf("got %v, want ErrSNMPConnectionFailed", err)
	}
}

// TestClassifySNMPError_Nil memverifikasi bahwa nil error tetap nil.
func TestClassifySNMPError_Nil(t *testing.T) {
	err := classifySNMPError(nil)
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

// =============================================================================
// Test: Konversi PDU ke SNMPResult
// =============================================================================

// TestConvertPDU_Integer memverifikasi konversi PDU tipe Integer.
func TestConvertPDU_Integer(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.1.7.0",
		Type:  gosnmp.Integer,
		Value: 72,
	}
	result := convertPDU(pdu)
	if result.OID != pdu.Name {
		t.Errorf("OID: got %q, want %q", result.OID, pdu.Name)
	}
	if result.Type != domain.SNMPValueInteger {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueInteger)
	}
}

// TestConvertPDU_OctetString memverifikasi konversi PDU tipe OctetString.
func TestConvertPDU_OctetString(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.1.1.0",
		Type:  gosnmp.OctetString,
		Value: []byte("ZTE ZXA10 C320"),
	}
	result := convertPDU(pdu)
	if result.Type != domain.SNMPValueString {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueString)
	}
	if result.Value != "ZTE ZXA10 C320" {
		t.Errorf("Value: got %q, want %q", result.Value, "ZTE ZXA10 C320")
	}
}

// TestConvertPDU_Counter32 memverifikasi konversi PDU tipe Counter32.
func TestConvertPDU_Counter32(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.2.2.1.10.1",
		Type:  gosnmp.Counter32,
		Value: uint(123456),
	}
	result := convertPDU(pdu)
	if result.Type != domain.SNMPValueCounter32 {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueCounter32)
	}
}

// TestConvertPDU_Counter64 memverifikasi konversi PDU tipe Counter64.
func TestConvertPDU_Counter64(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.31.1.1.1.6.1",
		Type:  gosnmp.Counter64,
		Value: uint64(9876543210),
	}
	result := convertPDU(pdu)
	if result.Type != domain.SNMPValueCounter64 {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueCounter64)
	}
}

// TestConvertPDU_Gauge32 memverifikasi konversi PDU tipe Gauge32.
func TestConvertPDU_Gauge32(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.2.2.1.5.1",
		Type:  gosnmp.Gauge32,
		Value: uint(1000000000),
	}
	result := convertPDU(pdu)
	if result.Type != domain.SNMPValueGauge32 {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueGauge32)
	}
}

// TestConvertPDU_TimeTicks memverifikasi konversi PDU tipe TimeTicks.
func TestConvertPDU_TimeTicks(t *testing.T) {
	pdu := gosnmp.SnmpPDU{
		Name:  ".1.3.6.1.2.1.1.3.0",
		Type:  gosnmp.TimeTicks,
		Value: uint32(123456789),
	}
	result := convertPDU(pdu)
	if result.Type != domain.SNMPValueTimeTicks {
		t.Errorf("Type: got %q, want %q", result.Type, domain.SNMPValueTimeTicks)
	}
}

// TestConvertPDUs_Multiple memverifikasi konversi beberapa PDU sekaligus.
func TestConvertPDUs_Multiple(t *testing.T) {
	pdus := []gosnmp.SnmpPDU{
		{Name: ".1.3.6.1.2.1.1.1.0", Type: gosnmp.OctetString, Value: []byte("test")},
		{Name: ".1.3.6.1.2.1.1.3.0", Type: gosnmp.TimeTicks, Value: uint32(100)},
	}
	results := convertPDUs(pdus)
	if len(results) != 2 {
		t.Fatalf("len: got %d, want 2", len(results))
	}
	if results[0].Type != domain.SNMPValueString {
		t.Errorf("[0] Type: got %q, want %q", results[0].Type, domain.SNMPValueString)
	}
	if results[1].Type != domain.SNMPValueTimeTicks {
		t.Errorf("[1] Type: got %q, want %q", results[1].Type, domain.SNMPValueTimeTicks)
	}
}

// TestConvertPDUs_Empty memverifikasi konversi slice PDU kosong.
func TestConvertPDUs_Empty(t *testing.T) {
	results := convertPDUs(nil)
	if len(results) != 0 {
		t.Errorf("len: got %d, want 0", len(results))
	}
}

// =============================================================================
// Test: Mapping Auth/Priv Protocol
// =============================================================================

// TestMapAuthProtocol memverifikasi mapping string ke gosnmp auth protocol.
func TestMapAuthProtocol(t *testing.T) {
	if mapAuthProtocol("SHA") != gosnmp.SHA {
		t.Error("SHA tidak di-map dengan benar")
	}
	if mapAuthProtocol("sha") != gosnmp.SHA {
		t.Error("sha (lowercase) tidak di-map dengan benar")
	}
	if mapAuthProtocol("MD5") != gosnmp.MD5 {
		t.Error("MD5 tidak di-map dengan benar")
	}
	if mapAuthProtocol("unknown") != gosnmp.MD5 {
		t.Error("unknown harus default ke MD5")
	}
}

// TestMapPrivProtocol memverifikasi mapping string ke gosnmp priv protocol.
func TestMapPrivProtocol(t *testing.T) {
	if mapPrivProtocol("AES") != gosnmp.AES {
		t.Error("AES tidak di-map dengan benar")
	}
	if mapPrivProtocol("aes") != gosnmp.AES {
		t.Error("aes (lowercase) tidak di-map dengan benar")
	}
	if mapPrivProtocol("DES") != gosnmp.DES {
		t.Error("DES tidak di-map dengan benar")
	}
	if mapPrivProtocol("unknown") != gosnmp.DES {
		t.Error("unknown harus default ke DES")
	}
}
