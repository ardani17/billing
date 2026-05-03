package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Feature: notification-service, Property 13: Credential masking preserves last 4 characters
// **Validates: Requirements 13.6**
//
// Untuk setiap string credential dengan panjang >= 4, MaskCredential menghasilkan
// output yang diakhiri 4 karakter terakhir dari string asli, dan semua karakter
// sebelumnya diganti dengan "•". Untuk string < 4 karakter, seluruh string
// di-mask menjadi "••••••••".
func TestProperty_CredentialMaskingPreservesLast4(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate string credential acak (panjang 0-100, karakter printable)
		value := rapid.StringMatching(`[a-zA-Z0-9!@#$%^&*()_+\-=]{0,100}`).Draw(t, "credential")

		result := MaskCredential(value)

		if len(value) >= 4 {
			// Verifikasi: output diakhiri 4 karakter terakhir dari string asli
			last4 := value[len(value)-4:]
			if !strings.HasSuffix(result, last4) {
				t.Fatalf(
					"MaskCredential(%q) = %q, tidak diakhiri dengan 4 karakter terakhir %q",
					value, result, last4,
				)
			}

			// Verifikasi: bagian sebelum 4 karakter terakhir hanya berisi "•"
			maskedPart := result[:len(result)-4]
			expectedMasked := strings.Repeat("•", len(value)-4)
			if maskedPart != expectedMasked {
				t.Fatalf(
					"MaskCredential(%q) = %q, bagian mask %q tidak sesuai expected %q",
					value, result, maskedPart, expectedMasked,
				)
			}

			// Verifikasi: panjang output sama dengan panjang input
			if len(result) != len(value)+(len(value)-4)*(len("•")-1) {
				// Karena "•" adalah multi-byte, kita cek jumlah rune
				expectedRuneLen := len(value) - 4 + 4
				resultRunes := []rune(result)
				if len(resultRunes) != expectedRuneLen {
					t.Fatalf(
						"MaskCredential(%q) rune length = %d, expected %d",
						value, len(resultRunes), expectedRuneLen,
					)
				}
			}
		} else {
			// Verifikasi: string < 4 karakter di-mask seluruhnya
			if result != "••••••••" {
				t.Fatalf(
					"MaskCredential(%q) = %q, expected %q untuk string < 4 karakter",
					value, result, "••••••••",
				)
			}
		}
	})
}

// Feature: notification-service, Property 14: Config validation — credentials required when enabled
// **Validates: Requirements 13.3**
//
// Untuk setiap NotificationConfig dengan is_enabled=true, ValidateCredentials
// gagal jika ada field credential yang kosong, dan berhasil jika semua field
// required terisi non-empty.
func TestProperty_ConfigValidationCredentialsRequired(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Pilih channel secara acak
		channels := []Channel{ChannelWhatsApp, ChannelSMS, ChannelEmail}
		channelIdx := rapid.IntRange(0, len(channels)-1).Draw(t, "channelIdx")
		channel := channels[channelIdx]

		// Generate credential yang valid (semua field non-empty)
		allFilled := rapid.Bool().Draw(t, "allFilled")

		var creds json.RawMessage
		var err error

		switch channel {
		case ChannelWhatsApp:
			c := WhatsAppCredentials{}
			if allFilled {
				c.APIToken = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "apiToken")
				c.SenderNumber = rapid.StringMatching(`\+62[0-9]{9,12}`).Draw(t, "senderNumber")
			} else {
				// Setidaknya satu field kosong
				leaveTokenEmpty := rapid.Bool().Draw(t, "leaveTokenEmpty")
				if leaveTokenEmpty {
					c.APIToken = ""
					c.SenderNumber = rapid.StringMatching(`\+62[0-9]{9,12}`).Draw(t, "senderNumber")
				} else {
					c.APIToken = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "apiToken")
					c.SenderNumber = ""
				}
			}
			creds, err = json.Marshal(c)
			if err != nil {
				t.Fatalf("gagal marshal WhatsAppCredentials: %v", err)
			}

		case ChannelSMS:
			c := SMSCredentials{}
			if allFilled {
				c.APIKey = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "apiKey")
				c.UserKey = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "userKey")
			} else {
				leaveAPIKeyEmpty := rapid.Bool().Draw(t, "leaveAPIKeyEmpty")
				if leaveAPIKeyEmpty {
					c.APIKey = ""
					c.UserKey = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "userKey")
				} else {
					c.APIKey = rapid.StringMatching(`[a-zA-Z0-9]{8,32}`).Draw(t, "apiKey")
					c.UserKey = ""
				}
			}
			creds, err = json.Marshal(c)
			if err != nil {
				t.Fatalf("gagal marshal SMSCredentials: %v", err)
			}

		case ChannelEmail:
			c := EmailCredentials{}
			if allFilled {
				c.SMTPHost = rapid.StringMatching(`[a-z]{3,10}\.[a-z]{2,5}`).Draw(t, "smtpHost")
				c.SMTPPort = rapid.IntRange(1, 65535).Draw(t, "smtpPort")
				c.Username = rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "username")
				c.Password = rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "password")
				c.FromName = rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "fromName")
				c.FromEmail = rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,10}\.[a-z]{2,5}`).Draw(t, "fromEmail")
			} else {
				// Kosongkan salah satu field secara acak
				fieldToEmpty := rapid.IntRange(0, 5).Draw(t, "fieldToEmpty")
				c.SMTPHost = rapid.StringMatching(`[a-z]{3,10}\.[a-z]{2,5}`).Draw(t, "smtpHost")
				c.SMTPPort = rapid.IntRange(1, 65535).Draw(t, "smtpPort")
				c.Username = rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "username")
				c.Password = rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "password")
				c.FromName = rapid.StringMatching(`[A-Za-z ]{3,20}`).Draw(t, "fromName")
				c.FromEmail = rapid.StringMatching(`[a-z]{3,10}@[a-z]{3,10}\.[a-z]{2,5}`).Draw(t, "fromEmail")
				switch fieldToEmpty {
				case 0:
					c.SMTPHost = ""
				case 1:
					c.SMTPPort = 0
				case 2:
					c.Username = ""
				case 3:
					c.Password = ""
				case 4:
					c.FromName = ""
				case 5:
					c.FromEmail = ""
				}
			}
			creds, err = json.Marshal(c)
			if err != nil {
				t.Fatalf("gagal marshal EmailCredentials: %v", err)
			}
		}

		validationErr := ValidateCredentials(channel, creds)

		if allFilled {
			// Semua field terisi → validasi harus berhasil
			if validationErr != nil {
				t.Fatalf(
					"ValidateCredentials(%s, %s) gagal padahal semua field terisi: %v",
					channel, string(creds), validationErr,
				)
			}
		} else {
			// Ada field kosong → validasi harus gagal
			if validationErr == nil {
				t.Fatalf(
					"ValidateCredentials(%s, %s) berhasil padahal ada field kosong",
					channel, string(creds),
				)
			}
		}
	})
}

// Feature: notification-service, Property 15: Template validation — at least one channel body required
// **Validates: Requirements 14.5**
//
// Untuk setiap kombinasi body template, ValidateTemplateBody gagal jika semua
// body kosong, dan berhasil jika minimal satu body channel terisi.
func TestProperty_TemplateBodyValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 4 body field — masing-masing bisa kosong atau terisi
		allEmpty := rapid.Bool().Draw(t, "allEmpty")

		var bodyWA, bodySMS, bodyEmailSubject, bodyEmailHTML string

		if allEmpty {
			// Semua body kosong
			bodyWA = ""
			bodySMS = ""
			bodyEmailSubject = ""
			bodyEmailHTML = ""
		} else {
			// Minimal satu body terisi — generate secara acak
			bodyWA = rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "bodyWA")
			bodySMS = rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "bodySMS")
			bodyEmailSubject = rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "bodyEmailSubject")
			bodyEmailHTML = rapid.StringMatching(`[a-zA-Z0-9 ]{0,50}`).Draw(t, "bodyEmailHTML")

			// Jika secara kebetulan semua kosong, paksa minimal satu terisi
			if bodyWA == "" && bodySMS == "" && bodyEmailSubject == "" && bodyEmailHTML == "" {
				bodyWA = rapid.StringMatching(`[a-zA-Z0-9 ]{1,50}`).Draw(t, "bodyWAForced")
			}
		}

		err := ValidateTemplateBody(bodyWA, bodySMS, bodyEmailSubject, bodyEmailHTML)

		if allEmpty {
			if err == nil {
				t.Fatal("ValidateTemplateBody harus gagal jika semua body kosong")
			}
		} else {
			if err != nil {
				t.Fatalf(
					"ValidateTemplateBody(%q, %q, %q, %q) gagal padahal ada body terisi: %v",
					bodyWA, bodySMS, bodyEmailSubject, bodyEmailHTML, err,
				)
			}
		}
	})
}

// Feature: notification-service, Property 16: Settings range validation
// **Validates: Requirements 20.5, 20.6**
//
// Untuk setiap integer daily_limit, validasi berhasil jika dan hanya jika
// nilainya dalam [1, 20]. Untuk setiap integer cooldown_minutes, validasi
// berhasil jika dan hanya jika nilainya dalam [5, 120].
func TestProperty_SettingsRangeValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dailyLimit := rapid.IntRange(-10, 30).Draw(t, "dailyLimit")
		cooldown := rapid.IntRange(-10, 200).Draw(t, "cooldown")

		settings := ConfigSettings{
			DailyLimitPerCust: dailyLimit,
			CooldownMinutes:   cooldown,
		}

		err := ValidateSettings(settings)

		dailyValid := dailyLimit >= 1 && dailyLimit <= 20
		cooldownValid := cooldown >= 5 && cooldown <= 120

		if dailyValid && cooldownValid {
			// Kedua nilai dalam range → validasi harus berhasil
			if err != nil {
				t.Fatalf(
					"ValidateSettings(daily=%d, cooldown=%d) gagal padahal dalam range: %v",
					dailyLimit, cooldown, err,
				)
			}
		} else {
			// Salah satu di luar range → validasi harus gagal
			if err == nil {
				t.Fatalf(
					"ValidateSettings(daily=%d, cooldown=%d) berhasil padahal di luar range",
					dailyLimit, cooldown,
				)
			}
		}
	})
}

// Feature: notification-service, Property 17: Quiet hours time validation
// **Validates: Requirements 20.3**
//
// Untuk setiap pasangan string HH:MM yang valid, ValidateQuietHours berhasil
// jika dan hanya jika start secara kronologis sebelum end dalam hari yang sama.
func TestProperty_QuietHoursTimeValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate jam dan menit yang valid
		startHour := rapid.IntRange(0, 23).Draw(t, "startHour")
		startMin := rapid.IntRange(0, 59).Draw(t, "startMin")
		endHour := rapid.IntRange(0, 23).Draw(t, "endHour")
		endMin := rapid.IntRange(0, 59).Draw(t, "endMin")

		start := fmt.Sprintf("%02d:%02d", startHour, startMin)
		end := fmt.Sprintf("%02d:%02d", endHour, endMin)

		err := ValidateQuietHours(start, end)

		startTotal := startHour*60 + startMin
		endTotal := endHour*60 + endMin

		if startTotal < endTotal {
			// start sebelum end → validasi harus berhasil
			if err != nil {
				t.Fatalf(
					"ValidateQuietHours(%q, %q) gagal padahal start < end: %v",
					start, end, err,
				)
			}
		} else {
			// start >= end → validasi harus gagal
			if err == nil {
				t.Fatalf(
					"ValidateQuietHours(%q, %q) berhasil padahal start >= end",
					start, end,
				)
			}
		}
	})
}

// Feature: notification-service, Property 18: Page size normalization
// **Validates: Requirements 12.3**
//
// Untuk setiap integer page_size, NormalizePageSize mengembalikan nilai apa adanya
// jika termasuk {10, 25, 50}, dan mengembalikan 25 untuk nilai lainnya.
func TestProperty_PageSizeNormalization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pageSize := rapid.IntRange(-100, 200).Draw(t, "pageSize")

		result := NormalizePageSize(pageSize)

		validSizes := map[int]bool{10: true, 25: true, 50: true}

		if validSizes[pageSize] {
			// Nilai valid → harus dikembalikan apa adanya
			if result != pageSize {
				t.Fatalf(
					"NormalizePageSize(%d) = %d, expected %d (nilai valid)",
					pageSize, result, pageSize,
				)
			}
		} else {
			// Nilai tidak valid → harus default ke 25
			if result != 25 {
				t.Fatalf(
					"NormalizePageSize(%d) = %d, expected 25 (default)",
					pageSize, result,
				)
			}
		}
	})
}
