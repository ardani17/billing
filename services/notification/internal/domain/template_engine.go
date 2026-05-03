package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// variablePattern mencocokkan pola {nama_variabel} dalam template body.
	variablePattern = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)
	// validTimezones berisi daftar timezone yang diizinkan.
	validTimezones = map[string]bool{
		"Asia/Jakarta": true, "Asia/Makassar": true, "Asia/Jayapura": true,
	}
	// validPageSizes berisi ukuran halaman yang diizinkan.
	validPageSizes = map[int]bool{10: true, 25: true, 50: true}
	// indonesianMonths memetakan nomor bulan ke nama bulan Indonesia.
	indonesianMonths = [...]string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
)

// TemplateEngine melakukan substitusi variabel pada template body.
type TemplateEngine struct{}

// NewTemplateEngine membuat instance baru TemplateEngine.
func NewTemplateEngine() *TemplateEngine { return &TemplateEngine{} }

// Render mengganti semua variabel {nama_var} dengan nilai dari data map.
func (e *TemplateEngine) Render(body string, data map[string]string) string {
	vars := e.ExtractVariables(body)
	result := body
	for _, v := range vars {
		val := data[v]
		result = strings.ReplaceAll(result, "{"+v+"}", val)
	}
	return result
}

// ExtractVariables mengekstrak daftar nama variabel dari template body.
// Mengembalikan slice nama variabel tanpa kurung kurawal.
func (e *TemplateEngine) ExtractVariables(body string) []string {
	matches := variablePattern.FindAllStringSubmatch(body, -1)
	seen := make(map[string]bool, len(matches))
	vars := make([]string, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if !seen[name] {
			seen[name] = true
			vars = append(vars, name)
		}
	}
	return vars
}

// FormatMoney memformat angka ke format Rupiah: "Rp 388.500".
func FormatMoney(amount int64) string {
	prefix := "Rp "
	if amount < 0 {
		prefix = "-Rp "
		amount = -amount
	}
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return prefix + s
	}
	var b strings.Builder
	rem := n % 3
	if rem > 0 {
		b.WriteString(s[:rem])
	}
	for i := rem; i < n; i += 3 {
		if b.Len() > 0 {
			b.WriteByte('.')
		}
		b.WriteString(s[i : i+3])
	}
	return prefix + b.String()
}

// FormatDateID memformat time.Time ke format Indonesia: "5 April 2026".
func FormatDateID(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), indonesianMonths[t.Month()], t.Year())
}

// MaskCredential menyembunyikan credential, menampilkan 4 karakter terakhir.
func MaskCredential(value string) string {
	if len(value) < 4 {
		return "••••••••"
	}
	masked := strings.Repeat("•", len(value)-4)
	return masked + value[len(value)-4:]
}

// ValidateQuietHours memvalidasi format dan urutan jam tenang (HH:MM).
func ValidateQuietHours(start, end string) error {
	sh, sm, err := parseHHMM(start)
	if err != nil {
		return ErrInvalidQuietHours
	}
	eh, em, err := parseHHMM(end)
	if err != nil {
		return ErrInvalidQuietHours
	}
	startMin := sh*60 + sm
	endMin := eh*60 + em
	if startMin >= endMin {
		return ErrInvalidQuietHours
	}
	return nil
}

// parseHHMM mem-parse string format "HH:MM" menjadi jam dan menit.
func parseHHMM(s string) (int, int, error) {
	if len(s) != 5 || s[2] != ':' {
		return 0, 0, fmt.Errorf("format tidak valid")
	}
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil {
		return 0, 0, err
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("jam/menit di luar jangkauan")
	}
	return h, m, nil
}

// ValidateTimezone memvalidasi timezone yang diizinkan.
// Hanya menerima: Asia/Jakarta, Asia/Makassar, Asia/Jayapura.
func ValidateTimezone(tz string) error {
	if !validTimezones[tz] {
		return ErrInvalidTimezone
	}
	return nil
}

// ValidateSettings memvalidasi pengaturan umum notifikasi.
func ValidateSettings(s ConfigSettings) error {
	if s.DailyLimitPerCust < 1 || s.DailyLimitPerCust > 20 {
		return fmt.Errorf("daily_limit_per_customer harus antara 1 dan 20")
	}
	if s.CooldownMinutes < 5 || s.CooldownMinutes > 120 {
		return fmt.Errorf("cooldown_minutes harus antara 5 dan 120")
	}
	if s.Timezone != "" {
		if err := ValidateTimezone(s.Timezone); err != nil {
			return err
		}
	}
	if s.QuietHoursStart != "" && s.QuietHoursEnd != "" {
		if err := ValidateQuietHours(s.QuietHoursStart, s.QuietHoursEnd); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTemplateBody memvalidasi bahwa minimal satu body channel terisi.
func ValidateTemplateBody(bodyWA, bodySMS, bodyEmailSubject, bodyEmailHTML string) error {
	if bodyWA == "" && bodySMS == "" && bodyEmailSubject == "" && bodyEmailHTML == "" {
		return fmt.Errorf("minimal satu body channel harus diisi")
	}
	return nil
}

// ValidateCredentials memvalidasi kelengkapan credential berdasarkan channel.
func ValidateCredentials(channel Channel, creds json.RawMessage) error {
	switch channel {
	case ChannelWhatsApp:
		var c WhatsAppCredentials
		if err := json.Unmarshal(creds, &c); err != nil {
			return ErrInvalidCredentials
		}
		if c.APIToken == "" || c.SenderNumber == "" {
			return ErrInvalidCredentials
		}
	case ChannelSMS:
		var c SMSCredentials
		if err := json.Unmarshal(creds, &c); err != nil {
			return ErrInvalidCredentials
		}
		if c.APIKey == "" || c.UserKey == "" {
			return ErrInvalidCredentials
		}
	case ChannelEmail:
		var c EmailCredentials
		if err := json.Unmarshal(creds, &c); err != nil {
			return ErrInvalidCredentials
		}
		if c.SMTPHost == "" || c.SMTPPort == 0 || c.Username == "" ||
			c.Password == "" || c.FromName == "" || c.FromEmail == "" {
			return ErrInvalidCredentials
		}
	default:
		return ErrInvalidCredentials
	}
	return nil
}

// NormalizePageSize menormalisasi ukuran halaman (valid: 10, 25, 50; default: 25).
func NormalizePageSize(pageSize int) int {
	if validPageSizes[pageSize] {
		return pageSize
	}
	return 25
}
