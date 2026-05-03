package adapter

import (
	"errors"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// =============================================================================
// Test: Konfigurasi Default CLI
// =============================================================================

// TestApplyDefaults_SSH memverifikasi default values untuk SSH config.
func TestApplyDefaults_SSH(t *testing.T) {
	cfg := applyDefaults(domain.CLIConfig{
		Host:     "192.168.1.1",
		Protocol: domain.CLIProtocolSSH,
		Username: "admin",
		Password: "secret",
	})

	if cfg.Port != defaultSSHPort {
		t.Errorf("port: got %d, want %d", cfg.Port, defaultSSHPort)
	}
	if cfg.ConnTimeout != defaultCLIConnectTimeout {
		t.Errorf("connTimeout: got %v, want %v", cfg.ConnTimeout, defaultCLIConnectTimeout)
	}
	if cfg.CmdTimeout != defaultCLICommandTimeout {
		t.Errorf("cmdTimeout: got %v, want %v", cfg.CmdTimeout, defaultCLICommandTimeout)
	}
}

// TestApplyDefaults_Telnet memverifikasi default values untuk Telnet config.
func TestApplyDefaults_Telnet(t *testing.T) {
	cfg := applyDefaults(domain.CLIConfig{
		Host:     "192.168.1.1",
		Protocol: domain.CLIProtocolTelnet,
		Username: "admin",
		Password: "secret",
	})

	if cfg.Port != defaultTelnetPort {
		t.Errorf("port: got %d, want %d", cfg.Port, defaultTelnetPort)
	}
	if cfg.ConnTimeout != defaultCLIConnectTimeout {
		t.Errorf("connTimeout: got %v, want %v", cfg.ConnTimeout, defaultCLIConnectTimeout)
	}
	if cfg.CmdTimeout != defaultCLICommandTimeout {
		t.Errorf("cmdTimeout: got %v, want %v", cfg.CmdTimeout, defaultCLICommandTimeout)
	}
}

// TestApplyDefaults_CustomValues memverifikasi bahwa custom values tidak di-override.
func TestApplyDefaults_CustomValues(t *testing.T) {
	cfg := applyDefaults(domain.CLIConfig{
		Host:        "10.0.0.1",
		Port:        2222,
		Protocol:    domain.CLIProtocolSSH,
		Username:    "admin",
		Password:    "secret",
		ConnTimeout: 5 * time.Second,
		CmdTimeout:  15 * time.Second,
	})

	if cfg.Port != 2222 {
		t.Errorf("port: got %d, want 2222", cfg.Port)
	}
	if cfg.ConnTimeout != 5*time.Second {
		t.Errorf("connTimeout: got %v, want 5s", cfg.ConnTimeout)
	}
	if cfg.CmdTimeout != 15*time.Second {
		t.Errorf("cmdTimeout: got %v, want 15s", cfg.CmdTimeout)
	}
}

// =============================================================================
// Test: SSH Config Building
// =============================================================================

// TestBuildSSHConfig memverifikasi bahwa ssh.ClientConfig dibangun dengan benar.
func TestBuildSSHConfig(t *testing.T) {
	cfg := domain.CLIConfig{
		Host:        "192.168.1.1",
		Port:        22,
		Protocol:    domain.CLIProtocolSSH,
		Username:    "admin",
		Password:    "secret123",
		ConnTimeout: 10 * time.Second,
	}

	sshCfg := buildSSHConfig(cfg)

	if sshCfg.User != "admin" {
		t.Errorf("user: got %q, want %q", sshCfg.User, "admin")
	}
	if sshCfg.Timeout != 10*time.Second {
		t.Errorf("timeout: got %v, want 10s", sshCfg.Timeout)
	}
	if len(sshCfg.Auth) != 1 {
		t.Fatalf("auth methods: got %d, want 1", len(sshCfg.Auth))
	}
	// HostKeyCallback harus diset (InsecureIgnoreHostKey)
	if sshCfg.HostKeyCallback == nil {
		t.Error("hostKeyCallback: harus diset (InsecureIgnoreHostKey)")
	}
}

// TestBuildSSHConfig_HostKeyCallback memverifikasi InsecureIgnoreHostKey menerima semua key.
func TestBuildSSHConfig_HostKeyCallback(t *testing.T) {
	cfg := domain.CLIConfig{
		Host:        "192.168.1.1",
		Username:    "admin",
		Password:    "secret",
		ConnTimeout: 5 * time.Second,
	}

	sshCfg := buildSSHConfig(cfg)

	// InsecureIgnoreHostKey harus menerima key apapun tanpa error
	err := sshCfg.HostKeyCallback("192.168.1.1:22", nil, &fakePublicKey{})
	if err != nil {
		t.Errorf("hostKeyCallback harus menerima semua key, got error: %v", err)
	}
}

// fakePublicKey mengimplementasikan ssh.PublicKey untuk testing.
type fakePublicKey struct{}

func (k *fakePublicKey) Type() string          { return "ssh-rsa" }
func (k *fakePublicKey) Marshal() []byte       { return []byte("fake-key") }
func (k *fakePublicKey) Verify([]byte, *ssh.Signature) error { return nil }

// =============================================================================
// Test: Enable Password Handling
// =============================================================================

// TestApplyDefaults_EnablePassword memverifikasi enable password dipertahankan.
func TestApplyDefaults_EnablePassword(t *testing.T) {
	cfg := applyDefaults(domain.CLIConfig{
		Host:           "192.168.1.1",
		Protocol:       domain.CLIProtocolSSH,
		Username:       "admin",
		Password:       "secret",
		EnablePassword: "enable123",
	})

	if cfg.EnablePassword != "enable123" {
		t.Errorf("enablePassword: got %q, want %q", cfg.EnablePassword, "enable123")
	}
}

// TestApplyDefaults_EmptyEnablePassword memverifikasi enable password kosong.
func TestApplyDefaults_EmptyEnablePassword(t *testing.T) {
	cfg := applyDefaults(domain.CLIConfig{
		Host:     "192.168.1.1",
		Protocol: domain.CLIProtocolTelnet,
		Username: "admin",
		Password: "secret",
	})

	if cfg.EnablePassword != "" {
		t.Errorf("enablePassword: got %q, want empty", cfg.EnablePassword)
	}
}

// =============================================================================
// Test: Klasifikasi Error CLI
// =============================================================================

// cliTimeoutError mengimplementasikan net.Error dengan Timeout() = true.
type cliTimeoutError struct{}

func (e *cliTimeoutError) Error() string   { return "i/o timeout" }
func (e *cliTimeoutError) Timeout() bool   { return true }
func (e *cliTimeoutError) Temporary() bool { return true }

// Pastikan cliTimeoutError mengimplementasikan net.Error.
var _ net.Error = (*cliTimeoutError)(nil)

// TestClassifyCLIError_Timeout memverifikasi net.Error timeout → ErrCLITimeout.
func TestClassifyCLIError_Timeout(t *testing.T) {
	err := classifyCLIError(&cliTimeoutError{})
	if !errors.Is(err, domain.ErrCLITimeout) {
		t.Errorf("got %v, want ErrCLITimeout", err)
	}
}

// TestClassifyCLIError_TimeoutString memverifikasi error "timeout" → ErrCLITimeout.
func TestClassifyCLIError_TimeoutString(t *testing.T) {
	err := classifyCLIError(errors.New("connection timeout exceeded"))
	if !errors.Is(err, domain.ErrCLITimeout) {
		t.Errorf("got %v, want ErrCLITimeout", err)
	}
}

// TestClassifyCLIError_DeadlineString memverifikasi error "deadline" → ErrCLITimeout.
func TestClassifyCLIError_DeadlineString(t *testing.T) {
	err := classifyCLIError(errors.New("context deadline exceeded"))
	if !errors.Is(err, domain.ErrCLITimeout) {
		t.Errorf("got %v, want ErrCLITimeout", err)
	}
}

// TestClassifyCLIError_Auth memverifikasi error autentikasi → ErrCLIAuthFailed.
func TestClassifyCLIError_Auth(t *testing.T) {
	cases := []string{
		"ssh: authentication failed",
		"invalid password",
		"permission denied (publickey,password)",
		"ssh: handshake failed",
	}
	for _, msg := range cases {
		err := classifyCLIError(errors.New(msg))
		if !errors.Is(err, domain.ErrCLIAuthFailed) {
			t.Errorf("msg=%q: got %v, want ErrCLIAuthFailed", msg, err)
		}
	}
}

// TestClassifyCLIError_Connection memverifikasi error umum → ErrCLIConnectionFailed.
func TestClassifyCLIError_Connection(t *testing.T) {
	err := classifyCLIError(errors.New("connection refused"))
	if !errors.Is(err, domain.ErrCLIConnectionFailed) {
		t.Errorf("got %v, want ErrCLIConnectionFailed", err)
	}
}

// TestClassifyCLIError_Nil memverifikasi nil error tetap nil.
func TestClassifyCLIError_Nil(t *testing.T) {
	err := classifyCLIError(nil)
	if err != nil {
		t.Errorf("got %v, want nil", err)
	}
}

// =============================================================================
// Test: NewCLIConnector
// =============================================================================

// TestNewCLIConnector memverifikasi constructor mengembalikan instance valid.
func TestNewCLIConnector(t *testing.T) {
	connector := NewCLIConnector()
	if connector == nil {
		t.Fatal("NewCLIConnector mengembalikan nil")
	}
}

// =============================================================================
// Test: Telnet Prompt Detection
// =============================================================================

// TestTelnetPromptChars memverifikasi prompt chars yang didukung.
func TestTelnetPromptChars(t *testing.T) {
	expected := []string{"#", ">", "$"}
	if len(telnetPromptChars) != len(expected) {
		t.Fatalf("telnetPromptChars: got %d items, want %d", len(telnetPromptChars), len(expected))
	}
	for i, p := range expected {
		if telnetPromptChars[i] != p {
			t.Errorf("telnetPromptChars[%d]: got %q, want %q", i, telnetPromptChars[i], p)
		}
	}
}
