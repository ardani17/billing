package adapter

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// Bawaan timeout untuk koneksi CLI.
const (
	defaultCLIConnectTimeout = 10 * time.Second
	defaultCLICommandTimeout = 30 * time.Second
	defaultSSHPort           = 22
	defaultTelnetPort        = 23
)

// cliConnector mengimplementasikan domain.CLIConnector.
// Connect-on-demand: buka session -> kirim command -> tutup session.
type cliConnector struct{}

// NewCLIConnector membuat instance baru CLIConnector.
func NewCLIConnector() domain.CLIConnector {
	return &cliConnector{}
}

// Execute membuka session, mengirim satu command, dan mengembalikan output.
func (c *cliConnector) Execute(ctx context.Context, cfg domain.CLIConfig, command string) (string, error) {
	cfg = applyDefaults(cfg)

	switch cfg.Protocol {
	case domain.CLIProtocolSSH:
		return c.executeSSH(ctx, cfg, command)
	case domain.CLIProtocolTelnet:
		return c.executeTelnet(ctx, cfg, command)
	default:
		return "", fmt.Errorf("protokol CLI tidak didukung: %s", cfg.Protocol)
	}
}

// ExecuteMultiple mengirim beberapa command dalam satu session.
func (c *cliConnector) ExecuteMultiple(ctx context.Context, cfg domain.CLIConfig, commands []string) ([]string, error) {
	cfg = applyDefaults(cfg)

	switch cfg.Protocol {
	case domain.CLIProtocolSSH:
		return c.executeMultipleSSH(ctx, cfg, commands)
	case domain.CLIProtocolTelnet:
		return c.executeMultipleTelnet(ctx, cfg, commands)
	default:
		return nil, fmt.Errorf("protokol CLI tidak didukung: %s", cfg.Protocol)
	}
}

// TestConnection menguji koneksi CLI dan mengembalikan banner/prompt.
func (c *cliConnector) TestConnection(ctx context.Context, cfg domain.CLIConfig) (string, error) {
	cfg = applyDefaults(cfg)

	switch cfg.Protocol {
	case domain.CLIProtocolSSH:
		return c.testSSH(ctx, cfg)
	case domain.CLIProtocolTelnet:
		return c.testTelnet(ctx, cfg)
	default:
		return "", fmt.Errorf("protokol CLI tidak didukung: %s", cfg.Protocol)
	}
}

// applyDefaults mengisi nilai bawaan pada CLIConfig jika belum diset.
func applyDefaults(cfg domain.CLIConfig) domain.CLIConfig {
	if cfg.ConnTimeout == 0 {
		cfg.ConnTimeout = defaultCLIConnectTimeout
	}
	if cfg.CmdTimeout == 0 {
		cfg.CmdTimeout = defaultCLICommandTimeout
	}
	if cfg.Port == 0 {
		switch cfg.Protocol {
		case domain.CLIProtocolSSH:
			cfg.Port = defaultSSHPort
		case domain.CLIProtocolTelnet:
			cfg.Port = defaultTelnetPort
		}
	}
	return cfg
}

// buildSSHConfig membuat ssh.ClientConfig dari domain.CLIConfig.
func buildSSHConfig(cfg domain.CLIConfig) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: cfg.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		// OLT device biasanya tidak punya known host key
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         cfg.ConnTimeout,
	}
}

// executeSSH menjalankan satu command via SSH.
func (c *cliConnector) executeSSH(ctx context.Context, cfg domain.CLIConfig, command string) (string, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, buildSSHConfig(cfg))
	if err != nil {
		return "", classifyCLIError(err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", classifyCLIError(err)
	}
	defer session.Close()

	// Gunakan command timeout via context
	var output bytes.Buffer
	session.Stdout = &output

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case err := <-done:
		if err != nil {
			// SSH exit status non-zero bukan selalu error fatal
			// OLT CLI kadang kembalikan exit code 1 tapi output tetap valid
			if output.Len() > 0 {
				return strings.TrimSpace(output.String()), nil
			}
			return "", classifyCLIError(err)
		}
		return strings.TrimSpace(output.String()), nil
	case <-time.After(cfg.CmdTimeout):
		return "", domain.ErrCLITimeout
	case <-ctx.Done():
		return "", domain.ErrCLITimeout
	}
}

// executeMultipleSSH menjalankan beberapa command dalam satu SSH session.
func (c *cliConnector) executeMultipleSSH(ctx context.Context, cfg domain.CLIConfig, commands []string) ([]string, error) {
	results := make([]string, 0, len(commands))
	for _, cmd := range commands {
		output, err := c.executeSSH(ctx, cfg, cmd)
		if err != nil {
			return results, err
		}
		results = append(results, output)
	}
	return results, nil
}

// testSSH menguji koneksi SSH dan mengembalikan server version.
func (c *cliConnector) testSSH(ctx context.Context, cfg domain.CLIConfig) (string, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	client, err := ssh.Dial("tcp", addr, buildSSHConfig(cfg))
	if err != nil {
		return "", classifyCLIError(err)
	}
	defer client.Close()

	// Ambil server version sebagai banner
	banner := string(client.ServerVersion())
	return banner, nil
}

// classifyCLIError mengklasifikasikan error CLI ke domain error yang sesuai.
func classifyCLIError(err error) error {
	if err == nil {
		return nil
	}
	errMsg := strings.ToLower(err.Error())

	// Deteksi timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return domain.ErrCLITimeout
	}
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline") {
		return domain.ErrCLITimeout
	}

	// Deteksi error autentikasi
	if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "password") ||
		strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "handshake") {
		return domain.ErrCLIAuthFailed
	}

	// Bawaan: koneksi gagal
	return domain.ErrCLIConnectionFailed
}
