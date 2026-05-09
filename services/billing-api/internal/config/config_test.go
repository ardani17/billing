package config

import (
	"testing"
)

// TestIsValidHex64 memverifikasi validasi format hex 64 karakter.
func TestIsValidHex64(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid 64 hex lowercase",
			input: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			want:  true,
		},
		{
			name:  "valid 64 hex uppercase",
			input: "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF",
			want:  true,
		},
		{
			name:  "valid 64 hex mixed case",
			input: "0123456789abCDEF0123456789abCDEF0123456789abCDEF0123456789abCDEF",
			want:  true,
		},
		{
			name:  "terlalu pendek",
			input: "0123456789abcdef",
			want:  false,
		},
		{
			name:  "terlalu panjang",
			input: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef00",
			want:  false,
		},
		{
			name:  "karakter non-hex",
			input: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg",
			want:  false,
		},
		{
			name:  "string kosong",
			input: "",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidHex64(tt.input)
			if got != tt.want {
				t.Errorf("isValidHex64(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestSplitAndTrim memverifikasi pemecahan string IP yang dipisahkan koma.
func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "string kosong",
			input: "",
			want:  nil,
		},
		{
			name:  "satu IP",
			input: "10.0.0.1",
			want:  []string{"10.0.0.1"},
		},
		{
			name:  "beberapa IP",
			input: "10.0.0.1,10.0.0.2,10.0.0.3",
			want:  []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		},
		{
			name:  "dengan spasi",
			input: " 10.0.0.1 , 10.0.0.2 , 10.0.0.3 ",
			want:  []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"},
		},
		{
			name:  "koma trailing diabaikan",
			input: "10.0.0.1,,10.0.0.2,",
			want:  []string{"10.0.0.1", "10.0.0.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("splitAndTrim(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitAndTrim(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestParseWebhookIPs memverifikasi parsing IP whitelist dari konfigurasi.
func TestParseWebhookIPs(t *testing.T) {
	cfg := &AppConfig{
		XenditWebhookIPs:   "10.0.0.1, 10.0.0.2",
		MidtransWebhookIPs: "192.168.1.1",
	}

	xenditIPs, midtransIPs := cfg.ParseWebhookIPs()

	if len(xenditIPs) != 2 {
		t.Fatalf("xenditIPs len = %d, want 2", len(xenditIPs))
	}
	if xenditIPs[0] != "10.0.0.1" || xenditIPs[1] != "10.0.0.2" {
		t.Errorf("xenditIPs = %v, want [10.0.0.1 10.0.0.2]", xenditIPs)
	}
	if len(midtransIPs) != 1 || midtransIPs[0] != "192.168.1.1" {
		t.Errorf("midtransIPs = %v, want [192.168.1.1]", midtransIPs)
	}
}

// TestParseWebhookIPsKosong memverifikasi bahwa IP kosong menghasilkan slice nil.
func TestParseWebhookIPsKosong(t *testing.T) {
	cfg := &AppConfig{}
	xenditIPs, midtransIPs := cfg.ParseWebhookIPs()
	if xenditIPs != nil {
		t.Errorf("xenditIPs = %v, want nil", xenditIPs)
	}
	if midtransIPs != nil {
		t.Errorf("midtransIPs = %v, want nil", midtransIPs)
	}
}

// TestMasterKeyBytes memverifikasi dekode hex master key menjadi 32 bytes.
func TestMasterKeyBytes(t *testing.T) {
	// 64 karakter hex = 32 bytes
	validKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	cfg := &AppConfig{GatewayMasterKey: validKey}

	key, err := cfg.MasterKeyBytes()
	if err != nil {
		t.Fatalf("MasterKeyBytes() error = %v", err)
	}
	if len(key) != 32 {
		t.Errorf("MasterKeyBytes() len = %d, want 32", len(key))
	}
}

func TestMasterKeyBytesKosong(t *testing.T) {
	cfg := &AppConfig{}
	_, err := cfg.MasterKeyBytes()
	if err == nil {
		t.Fatal("MasterKeyBytes() harus error jika key kosong")
	}
}

func TestMasterKeyBytesInvalid(t *testing.T) {
	cfg := &AppConfig{GatewayMasterKey: "bukan-hex-valid"}
	_, err := cfg.MasterKeyBytes()
	if err == nil {
		t.Fatal("MasterKeyBytes() harus error jika key bukan hex valid")
	}
}

func TestMasterKeyBytesSalahPanjang(t *testing.T) {
	// 32 karakter hex = 16 bytes (terlalu pendek)
	cfg := &AppConfig{GatewayMasterKey: "0123456789abcdef0123456789abcdef"}
	_, err := cfg.MasterKeyBytes()
	if err == nil {
		t.Fatal("MasterKeyBytes() harus error jika key bukan 32 bytes")
	}
}

func TestValidateGatewayMasterKeyValid(t *testing.T) {
	cfg := validConfig()
	cfg.GatewayMasterKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, seharusnya nil", err)
	}
}

func TestValidateGatewayMasterKeyInvalid(t *testing.T) {
	cfg := validConfig()
	cfg.GatewayMasterKey = "bukan-hex"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() seharusnya error untuk GATEWAY_MASTER_KEY invalid")
	}
}

// TestValidateGatewayMasterKeyKosong memverifikasi validasi lolos jika key kosong (opsional).
func TestValidateGatewayMasterKeyKosong(t *testing.T) {
	cfg := validConfig()
	cfg.GatewayMasterKey = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, seharusnya nil (key opsional)", err)
	}
}

func TestValidateProductionRejectsDevelopmentSecrets(t *testing.T) {
	cfg := validConfig()
	cfg.AppEnv = "production"
	cfg.DBSSLMode = "require"
	cfg.CORSAllowOrigins = "https://app.example.com"
	cfg.JWTSecret = developmentJWTSecret
	cfg.DBPassword = developmentDBPassword

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() harus error untuk secret development pada production")
	}
}

func TestValidateProductionAcceptsStrongSecrets(t *testing.T) {
	cfg := validConfig()
	cfg.AppEnv = "production"
	cfg.DBSSLMode = "require"
	cfg.CORSAllowOrigins = "https://app.example.com"
	cfg.JWTSecret = "production-jwt-secret-minimum-32-chars"
	cfg.DBPassword = "production-db-password"

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() error = %v, seharusnya nil", err)
	}
}

// validConfig mengembalikan AppConfig dengan semua field wajib terisi.
func validConfig() *AppConfig {
	return &AppConfig{
		AppName:    "test",
		DBHost:     "localhost",
		DBUser:     "user",
		DBPassword: "pass",
		DBName:     "db",
		RedisHost:  "localhost",
		JWTSecret:  "secret",
	}
}
