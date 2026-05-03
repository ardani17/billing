package adapter

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// telnetPromptChars berisi karakter-karakter yang menandakan prompt OLT.
// OLT biasanya menggunakan #, >, atau $ sebagai prompt.
var telnetPromptChars = []string{"#", ">", "$"}

// executeTelnet menjalankan satu command via Telnet.
func (c *cliConnector) executeTelnet(ctx context.Context, cfg domain.CLIConfig, command string) (string, error) {
	conn, err := c.dialTelnet(cfg)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Login jika username/password tersedia
	if err := c.telnetLogin(conn, cfg); err != nil {
		return "", err
	}

	// Kirim enable password jika diperlukan
	if cfg.EnablePassword != "" {
		if err := c.telnetEnable(conn, cfg); err != nil {
			return "", err
		}
	}

	// Kirim command dan baca response
	output, err := c.telnetSendCommand(conn, cfg, command)
	if err != nil {
		return "", err
	}
	return output, nil
}

// executeMultipleTelnet menjalankan beberapa command dalam satu Telnet session.
func (c *cliConnector) executeMultipleTelnet(ctx context.Context, cfg domain.CLIConfig, commands []string) ([]string, error) {
	conn, err := c.dialTelnet(cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Login
	if err := c.telnetLogin(conn, cfg); err != nil {
		return nil, err
	}

	// Enable mode jika diperlukan
	if cfg.EnablePassword != "" {
		if err := c.telnetEnable(conn, cfg); err != nil {
			return nil, err
		}
	}

	results := make([]string, 0, len(commands))
	for _, cmd := range commands {
		output, err := c.telnetSendCommand(conn, cfg, cmd)
		if err != nil {
			return results, err
		}
		results = append(results, output)
	}
	return results, nil
}

// testTelnet menguji koneksi Telnet dan mengembalikan banner.
func (c *cliConnector) testTelnet(ctx context.Context, cfg domain.CLIConfig) (string, error) {
	conn, err := c.dialTelnet(cfg)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Baca banner awal dari OLT
	banner, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout)
	if err != nil {
		return "", classifyCLIError(err)
	}
	return banner, nil
}

// dialTelnet membuka koneksi TCP ke OLT via Telnet.
func (c *cliConnector) dialTelnet(cfg domain.CLIConfig) (net.Conn, error) {
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	conn, err := net.DialTimeout("tcp", addr, cfg.ConnTimeout)
	if err != nil {
		return nil, classifyCLIError(err)
	}
	return conn, nil
}

// telnetLogin melakukan login ke OLT via Telnet.
func (c *cliConnector) telnetLogin(conn net.Conn, cfg domain.CLIConfig) error {
	// Baca sampai prompt username
	if _, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout); err != nil {
		return classifyCLIError(err)
	}

	// Kirim username
	if _, err := fmt.Fprintf(conn, "%s\n", cfg.Username); err != nil {
		return classifyCLIError(err)
	}

	// Baca sampai prompt password
	if _, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout); err != nil {
		return classifyCLIError(err)
	}

	// Kirim password
	if _, err := fmt.Fprintf(conn, "%s\n", cfg.Password); err != nil {
		return classifyCLIError(err)
	}

	// Baca sampai prompt CLI
	if _, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout); err != nil {
		return domain.ErrCLIAuthFailed
	}
	return nil
}

// telnetEnable mengirim enable password untuk masuk privileged mode.
func (c *cliConnector) telnetEnable(conn net.Conn, cfg domain.CLIConfig) error {
	if _, err := fmt.Fprintf(conn, "enable\n"); err != nil {
		return classifyCLIError(err)
	}
	// Baca sampai prompt password
	if _, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout); err != nil {
		return classifyCLIError(err)
	}
	// Kirim enable password
	if _, err := fmt.Fprintf(conn, "%s\n", cfg.EnablePassword); err != nil {
		return classifyCLIError(err)
	}
	// Baca sampai prompt privileged
	if _, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout); err != nil {
		return domain.ErrCLIAuthFailed
	}
	return nil
}

// telnetSendCommand mengirim command dan membaca response sampai prompt.
func (c *cliConnector) telnetSendCommand(conn net.Conn, cfg domain.CLIConfig, command string) (string, error) {
	if _, err := fmt.Fprintf(conn, "%s\n", command); err != nil {
		return "", classifyCLIError(err)
	}
	output, err := c.telnetReadUntilPrompt(conn, cfg.CmdTimeout)
	if err != nil {
		return "", classifyCLIError(err)
	}
	return strings.TrimSpace(output), nil
}

// telnetReadUntilPrompt membaca dari koneksi sampai menemukan prompt.
func (c *cliConnector) telnetReadUntilPrompt(conn net.Conn, timeout time.Duration) (string, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}

	var result strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		result.WriteString(line)
		result.WriteString("\n")
		// Cek apakah baris terakhir mengandung prompt
		trimmed := strings.TrimSpace(line)
		for _, p := range telnetPromptChars {
			if strings.HasSuffix(trimmed, p) {
				return result.String(), nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return result.String(), err
	}
	return result.String(), nil
}
