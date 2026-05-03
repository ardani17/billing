package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/notification/internal/domain"
)

// TemplateHandler menangani HTTP request untuk manajemen template notifikasi.
// Menyediakan endpoint CRUD: list, create, update, dan soft-delete template.
type TemplateHandler struct {
	templateRepo domain.TemplateRepository
}

// NewTemplateHandler membuat instance TemplateHandler baru dengan dependensi TemplateRepository.
func NewTemplateHandler(templateRepo domain.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{templateRepo: templateRepo}
}

// List menangani GET /api/v1/notifications/templates.
// Mengembalikan semua template notifikasi untuk tenant yang terautentikasi.
func (h *TemplateHandler) List(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-set oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Panggil repository untuk mengambil semua template tenant
	templates, err := h.templateRepo.ListByTenant(c.UserContext(), tenantID)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil data template")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, templates)
}

// Create menangani POST /api/v1/notifications/templates.
// Memvalidasi request, cek keunikan slug, dan membuat template baru.
func (h *TemplateHandler) Create(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-set oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Parse request body
	var req domain.CreateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}

	// Validasi slug tidak boleh kosong
	if req.Slug == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "slug tidak boleh kosong")
	}

	// Validasi name tidak boleh kosong
	if req.Name == "" {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", "name tidak boleh kosong")
	}

	// Validasi minimal satu body channel harus diisi
	if err := domain.ValidateTemplateBody(req.BodyWhatsApp, req.BodySMS, req.BodyEmailSubject, req.BodyEmailHTML); err != nil {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	}

	// Cek keunikan slug di tenant (excludeID kosong karena ini template baru)
	exists, err := h.templateRepo.SlugExists(c.UserContext(), tenantID, req.Slug, "")
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memeriksa slug")
	}
	if exists {
		return domain.ErrorResponse(c, fiber.StatusConflict, "TEMPLATE_SLUG_EXISTS", "slug template sudah ada")
	}

	// Buat template baru dengan tenant_id, is_default=false, is_active=true
	template := &domain.NotificationTemplate{
		TenantID:         tenantID,
		Slug:             req.Slug,
		Name:             req.Name,
		Category:         req.Category,
		EventType:        req.EventType,
		Channels:         req.Channels,
		BodyWhatsApp:     req.BodyWhatsApp,
		BodySMS:          req.BodySMS,
		BodyEmailSubject: req.BodyEmailSubject,
		BodyEmailHTML:    req.BodyEmailHTML,
		Variables:        req.Variables,
		IsActive:         true,
		IsDefault:        false,
	}

	// Simpan template ke database
	created, err := h.templateRepo.Create(c.UserContext(), template)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal membuat template")
	}

	return domain.SuccessResponse(c, fiber.StatusCreated, created)
}

// Update menangani PUT /api/v1/notifications/templates/:id.
// Memvalidasi request dan memperbarui template yang sudah ada.
func (h *TemplateHandler) Update(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-set oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Ambil template ID dari URL parameter
	templateID := c.Params("id")
	if templateID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "id template tidak boleh kosong")
	}

	// Parse request body
	var req domain.UpdateTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "format request tidak valid")
	}

	// Ambil template yang sudah ada berdasarkan ID
	existing, err := h.templateRepo.GetByID(c.UserContext(), templateID)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "template tidak ditemukan")
		}
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil template")
	}

	// Pastikan template milik tenant yang sama
	if existing.TenantID != tenantID {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "template tidak ditemukan")
	}

	// Terapkan perubahan dari request ke template yang ada
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Channels != nil {
		existing.Channels = req.Channels
	}
	if req.BodyWhatsApp != "" {
		existing.BodyWhatsApp = req.BodyWhatsApp
	}
	if req.BodySMS != "" {
		existing.BodySMS = req.BodySMS
	}
	if req.BodyEmailSubject != "" {
		existing.BodyEmailSubject = req.BodyEmailSubject
	}
	if req.BodyEmailHTML != "" {
		existing.BodyEmailHTML = req.BodyEmailHTML
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	// Simpan perubahan ke database
	updated, err := h.templateRepo.Update(c.UserContext(), existing)
	if err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal memperbarui template")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, updated)
}

// Delete menangani DELETE /api/v1/notifications/templates/:id.
// Memeriksa apakah template default, lalu melakukan soft-delete.
func (h *TemplateHandler) Delete(c *fiber.Ctx) error {
	// Ambil tenant_id dari Fiber locals (di-set oleh auth middleware)
	tenantID, _ := c.Locals("tenant_id").(string)
	if tenantID == "" {
		return domain.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant_id tidak ditemukan")
	}

	// Ambil template ID dari URL parameter
	templateID := c.Params("id")
	if templateID == "" {
		return domain.ErrorResponse(c, fiber.StatusBadRequest, "BAD_REQUEST", "id template tidak boleh kosong")
	}

	// Ambil template berdasarkan ID untuk validasi
	template, err := h.templateRepo.GetByID(c.UserContext(), templateID)
	if err != nil {
		if errors.Is(err, domain.ErrTemplateNotFound) {
			return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "template tidak ditemukan")
		}
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal mengambil template")
	}

	// Pastikan template milik tenant yang sama
	if template.TenantID != tenantID {
		return domain.ErrorResponse(c, fiber.StatusNotFound, "TEMPLATE_NOT_FOUND", "template tidak ditemukan")
	}

	// Template default tidak boleh dihapus
	if template.IsDefault {
		return domain.ErrorResponse(c, fiber.StatusUnprocessableEntity, "TEMPLATE_NOT_DELETABLE", "template default tidak bisa dihapus")
	}

	// Soft-delete template (set is_active=false)
	if err := h.templateRepo.SoftDelete(c.UserContext(), templateID); err != nil {
		return domain.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", "gagal menghapus template")
	}

	return domain.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"message": "template berhasil dihapus",
	})
}
