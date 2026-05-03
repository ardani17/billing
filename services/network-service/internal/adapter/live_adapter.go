// Package adapter — LiveAdapter untuk koneksi ke router MikroTik via RouterOS API.
// Menggunakan library go-routeros/routeros/v3 untuk komunikasi TCP/TLS.
package adapter

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/go-routeros/routeros/v3"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

// LiveAdapter mengimplementasikan RouterOSAdapter dengan koneksi ke router fisik.
type LiveAdapter struct {
	mu     sync.Mutex
	client *routeros.Client
}

// NewLiveAdapter membuat instance LiveAdapter baru.
func NewLiveAdapter() *LiveAdapter {
	return &LiveAdapter{}
}

// Connect membuka koneksi TCP/TLS ke router MikroTik.
// Menggunakan timeout dari ConnectionConfig untuk batas waktu koneksi.
func (a *LiveAdapter) Connect(_ context.Context, cfg ConnectionConfig) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	var (
		client *routeros.Client
		err    error
	)

	if cfg.UseSSL {
		client, err = routeros.DialTLSTimeout(
			addr, cfg.Username, cfg.Password, nil, cfg.ConnectTimeout,
		)
	} else {
		client, err = routeros.DialTimeout(
			addr, cfg.Username, cfg.Password, cfg.ConnectTimeout,
		)
	}

	if err != nil {
		if isTimeoutError(err) {
			return fmt.Errorf("%w: %s", domain.ErrConnectionTimeout, err.Error())
		}
		return fmt.Errorf("%w: %s", domain.ErrConnectionFailed, err.Error())
	}

	a.client = client
	return nil
}

// Close menutup koneksi ke router.
func (a *LiveAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client != nil {
		a.client.Close()
		a.client = nil
	}
	return nil
}

// Execute menjalankan perintah RouterOS dan mengembalikan hasil sebagai slice map.
// Setiap elemen map merepresentasikan satu baris response dari router.
func (a *LiveAdapter) Execute(ctx context.Context, command string, params map[string]string) ([]map[string]string, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()

	if client == nil {
		return nil, domain.ErrConnectionFailed
	}

	// Bangun argumen: command + parameter dalam format =key=value.
	// Sebagian command builder lama sudah menyimpan key dengan prefix "=".
	args := []string{command}
	for k, v := range params {
		if strings.HasPrefix(k, "=") {
			args = append(args, k+"="+v)
			continue
		}
		args = append(args, "="+k+"="+v)
	}

	reply, err := client.RunArgsContext(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domain.ErrConnectionFailed, err.Error())
	}

	// Konversi proto.Sentence ke map[string]string
	results := make([]map[string]string, 0, len(reply.Re))
	for _, sen := range reply.Re {
		results = append(results, sen.Map)
	}
	return results, nil
}

// GetSystemResource mengambil informasi sistem dari router.
// Menjalankan "/system/resource/print" dan parse hasilnya ke SystemResource.
func (a *LiveAdapter) GetSystemResource(ctx context.Context) (*SystemResource, error) {
	rows, err := a.Execute(ctx, "/system/resource/print", nil)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: response kosong dari /system/resource/print", domain.ErrConnectionFailed)
	}

	data := rows[0]

	// Ambil identity dari command terpisah
	identity := ""
	idRows, err := a.Execute(ctx, "/system/identity/print", nil)
	if err == nil && len(idRows) > 0 {
		identity = idRows[0]["name"]
	}

	cpuCount, _ := strconv.Atoi(data["cpu-count"])
	cpuLoad, _ := strconv.Atoi(data["cpu-load"])
	totalRAM, _ := strconv.ParseInt(data["total-memory"], 10, 64)
	freeRAM, _ := strconv.ParseInt(data["free-memory"], 10, 64)
	uptime := parseUptimeToSeconds(data["uptime"])

	return &SystemResource{
		Version:      data["version"],
		BoardName:    data["board-name"],
		CPUCount:     cpuCount,
		CPULoad:      cpuLoad,
		TotalRAM:     totalRAM,
		FreeRAM:      freeRAM,
		Uptime:       uptime,
		Architecture: data["architecture-name"],
		Identity:     identity,
	}, nil
}

// Ping memeriksa apakah koneksi ke router masih aktif.
// Menggunakan "/system/identity/print" sebagai health check ringan.
func (a *LiveAdapter) Ping(ctx context.Context) error {
	_, err := a.Execute(ctx, "/system/identity/print", nil)
	return err
}

// parseUptimeToSeconds mengkonversi format uptime RouterOS ke detik.
// Format: "45d00:00:00", "1w2d03:04:05", "00:05:30", "3d12:00:00".
func parseUptimeToSeconds(uptime string) int64 {
	var totalSeconds int64

	// Parse minggu (w)
	if idx := strings.Index(uptime, "w"); idx >= 0 {
		weeks, _ := strconv.ParseInt(uptime[:idx], 10, 64)
		totalSeconds += weeks * 7 * 24 * 3600
		uptime = uptime[idx+1:]
	}

	// Parse hari (d)
	if idx := strings.Index(uptime, "d"); idx >= 0 {
		days, _ := strconv.ParseInt(uptime[:idx], 10, 64)
		totalSeconds += days * 24 * 3600
		uptime = uptime[idx+1:]
	}

	// Parse jam:menit:detik (HH:MM:SS)
	parts := strings.Split(uptime, ":")
	if len(parts) == 3 {
		hours, _ := strconv.ParseInt(parts[0], 10, 64)
		minutes, _ := strconv.ParseInt(parts[1], 10, 64)
		seconds, _ := strconv.ParseInt(parts[2], 10, 64)
		totalSeconds += hours*3600 + minutes*60 + seconds
	}

	return totalSeconds
}

// isTimeoutError memeriksa apakah error adalah timeout.
func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "deadline exceeded")
}
