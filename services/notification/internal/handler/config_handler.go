package handler

import (
	"encoding/json"
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// ConfigHandler menangani HTTP permintaan untuk konfigurasi notifikasi.
// Menyediakan endpoint untuk melihat dan memperbarui konfigurasi provider per channel.
type ConfigHandler struct {
	configRepo   domain.ConfigRepository
	templateRepo domain.TemplateRepository
}

// NewConfigHandler membuat instance ConfigHandler baru dengan dependensi repositori.
func NewConfigHandler(configRepo domain.ConfigRepository, templateRepo domain.TemplateRepository) *ConfigHandler {
	return &ConfigHandler{configRepo: configRepo, templateRepo: templateRepo}
}

// Get menangani GET /api/v1/notifications/config.
// Mengembalikan konfigurasi notifikasi tenant dengan credential yang di-mask.
func (h *ConfigHandler) Get(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-atur oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Ambil semua konfigurasi channel untuk tenant
	configs, err := h.configRepo.GetByTenant(c.UserContext(), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil konfigurasi")
	}

	// Ambil pengaturan umum notifikasi untuk tenant
	settings, err := h.configRepo.GetSettings(c.UserContext(), tenantID)
	if err != nil {
		if errors.Is(err, domain.ErrConfigNotFound) {
			settings = &domain.ConfigSettings{}
		} else {
			return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil pengaturan")
		}
	}

	// Mask credential pada setiap konfigurasi sebelum dikembalikan
	maskedConfigs := make([]map[string]interface{}, 0, len(configs))
	for _, cfg := range configs {
		masked := map[string]interface{}{
			"id":         cfg.ID,
			"tenant_id":  cfg.TenantID,
			"channel":    cfg.Channel,
			"provider":   cfg.Provider,
			"is_enabled": cfg.IsEnabled,
			"priority":   cfg.Priority,
			"created_at": cfg.CreatedAt,
			"updated_at": cfg.UpdatedAt,
		}

		// Parsing dan mask setiap field credential
		masked["credentials"] = maskCredentials(cfg.Credentials)
		maskedConfigs = append(maskedConfigs, masked)
	}

	// Kembalikan respons dengan konfigurasi dan pengaturan
	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"configs":  maskedConfigs,
		"settings": settings,
	})
}

// Perbarui menangani PUT /api/v1/notifications/config.
// Memvalidasi dan menyimpan konfigurasi provider, serta seed template bawaan
// jika ini adalah konfigurasi pertama untuk tenant.
func (h *ConfigHandler) Update(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-atur oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Parsing permintaan body
	var req domain.UpdateConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}

	// Validasi channel
	if !domain.IsValidChannel(req.Channel) {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "channel tidak valid")
	}

	// Jika channel diaktifkan, validasi kelengkapan credential
	if req.IsEnabled {
		if err := domain.ValidateCredentials(req.Channel, req.Credentials); err != nil {
			return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		}
	}

	// Cek konfigurasi yang sudah ada sebelum upsert (untuk deteksi konfigurasi pertama)
	existingConfigs, err := h.configRepo.GetByTenant(c.UserContext(), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil konfigurasi")
	}

	// Buat atau perbarui konfigurasi
	cfg := &domain.NotificationConfig{
		TenantID:    tenantID,
		Channel:     req.Channel,
		Provider:    req.Provider,
		Credentials: req.Credentials,
		IsEnabled:   req.IsEnabled,
		Priority:    req.Priority,
	}

	updated, err := h.configRepo.Upsert(c.UserContext(), cfg)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan konfigurasi")
	}

	// Jika ini konfigurasi pertama untuk tenant, seed template bawaan
	if len(existingConfigs) == 0 {
		h.seedDefaultTemplates(c, tenantID)
	}

	return domain.SuccessResponse(c, fiber.StatusOK, updated)
}

// UpdateSettings menangani PUT /api/v1/notifications/config/settings.
// Memvalidasi dan menyimpan pengaturan umum notifikasi.
func (h *ConfigHandler) UpdateSettings(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-atur oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Parsing permintaan body
	var settings domain.ConfigSettings
	if err := c.BodyParser(&settings); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}

	// Validasi pengaturan
	if err := domain.ValidateSettings(settings); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}

	// Simpan pengaturan
	if err := h.configRepo.UpdateSettings(c.UserContext(), tenantID, settings); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menyimpan pengaturan")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, settings)
}

// seedDefaultTemplates membuat template bawaan untuk tenant baru.
// Dipanggil saat konfigurasi notifikasi pertama kali dibuat.
func (h *ConfigHandler) seedDefaultTemplates(c *fiber.Ctx, tenantID string) {
	templates := make([]*domain.NotificationTemplate, len(domain.DefaultTemplates))
	for i := range domain.DefaultTemplates {
		t := domain.DefaultTemplates[i]
		t.TenantID = tenantID
		templates[i] = &t
	}
	// Seed template, abaikan error (best-effort, tidak memblokir respons utama)
	_ = h.templateRepo.BulkCreate(c.UserContext(), templates)
}

// maskCredentials mem-parsing credential JSON dan menyembunyikan nilai setiap field.
// Menampilkan hanya 4 karakter terakhir dari setiap nilai string.
func maskCredentials(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return raw
	}

	for key, val := range fields {
		if strVal, ok := val.(string); ok {
			fields[key] = domain.MaskCredential(strVal)
		}
	}

	masked, err := json.Marshal(fields)
	if err != nil {
		return raw
	}
	return masked
}
