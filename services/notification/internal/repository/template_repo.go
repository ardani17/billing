package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ispboss/ispboss/services/notification/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// TemplateRepo mengimplementasikan domain.TemplateRepository menggunakan sqlc Queries.
type TemplateRepo struct{ queries *Queries }

// NewTemplateRepo membuat instance baru TemplateRepo.
func NewTemplateRepo(q *Queries) *TemplateRepo { return &TemplateRepo{queries: q} }

func textToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

func stringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// mapTemplateRow memetakan sqlc model ke domain.NotificationTemplate.
func mapTemplateRow(row NotificationTemplate) (*domain.NotificationTemplate, error) {
	var channels []domain.Channel
	if len(row.Channels) > 0 {
		if err := json.Unmarshal(row.Channels, &channels); err != nil {
			return nil, fmt.Errorf("repository: gagal unmarshal channels: %w", err)
		}
	}
	var variables []string
	if len(row.Variables) > 0 {
		if err := json.Unmarshal(row.Variables, &variables); err != nil {
			return nil, fmt.Errorf("repository: gagal unmarshal variables: %w", err)
		}
	}
	return &domain.NotificationTemplate{
		ID: uuidToString(row.ID), TenantID: uuidToString(row.TenantID),
		Slug: row.Slug, Name: row.Name, Category: domain.TemplateCategory(row.Category),
		EventType: textToString(row.EventType), Channels: channels,
		BodyWhatsApp: textToString(row.BodyWhatsapp), BodySMS: textToString(row.BodySms),
		BodyEmailSubject: textToString(row.BodyEmailSubject),
		BodyEmailHTML:    textToString(row.BodyEmailHtml),
		Variables:        variables, IsActive: row.IsActive, IsDefault: row.IsDefault,
		CreatedAt: timestamptzToTime(row.CreatedAt), UpdatedAt: timestamptzToTime(row.UpdatedAt),
	}, nil
}

// buildCreateParams menyiapkan JSONB channels dan variables untuk insert.
func buildCreateParams(t *domain.NotificationTemplate) (chJSON, varJSON []byte, err error) {
	if chJSON, err = json.Marshal(t.Channels); err != nil {
		return nil, nil, fmt.Errorf("repository: gagal marshal channels: %w", err)
	}
	if varJSON, err = json.Marshal(t.Variables); err != nil {
		return nil, nil, fmt.Errorf("repository: gagal marshal variables: %w", err)
	}
	return chJSON, varJSON, nil
}

// Buat membuat template notifikasi baru dan mengembalikan template yang dibuat.
func (r *TemplateRepo) Create(ctx context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	chJSON, varJSON, err := buildCreateParams(t)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateTemplate(ctx, CreateTemplateParams{
		TenantID: parseUUID(t.TenantID), Slug: t.Slug, Name: t.Name,
		Category: string(t.Category), EventType: stringToText(t.EventType),
		Channels: chJSON, BodyWhatsapp: stringToText(t.BodyWhatsApp),
		BodySms: stringToText(t.BodySMS), BodyEmailSubject: stringToText(t.BodyEmailSubject),
		BodyEmailHtml: stringToText(t.BodyEmailHTML), Variables: varJSON,
		IsActive: t.IsActive, IsDefault: t.IsDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("repository: gagal membuat template: %w", err)
	}
	return mapTemplateRow(row)
}

// GetByID mengambil template berdasarkan ID.
func (r *TemplateRepo) GetByID(ctx context.Context, id string) (*domain.NotificationTemplate, error) {
	row, err := r.queries.GetTemplateByID(ctx, parseUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil template by ID: %w", err)
	}
	return mapTemplateRow(row)
}

// GetBySlug mengambil template berdasarkan tenant_id dan slug.
func (r *TemplateRepo) GetBySlug(ctx context.Context, tenantID, slug string) (*domain.NotificationTemplate, error) {
	row, err := r.queries.GetTemplateBySlug(ctx, GetTemplateBySlugParams{
		TenantID: parseUUID(tenantID), Slug: slug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil template by slug: %w", err)
	}
	return mapTemplateRow(row)
}

// GetByEventType mengambil template berdasarkan tenant_id dan event_type (untuk delivery pipeline).
func (r *TemplateRepo) GetByEventType(ctx context.Context, tenantID, eventType string) (*domain.NotificationTemplate, error) {
	row, err := r.queries.GetTemplateByEventType(ctx, GetTemplateByEventTypeParams{
		TenantID: parseUUID(tenantID), EventType: stringToText(eventType),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("repository: gagal mengambil template by event_type: %w", err)
	}
	return mapTemplateRow(row)
}

// Perbarui memperbarui template dan mengembalikan template yang diperbarui.
func (r *TemplateRepo) Update(ctx context.Context, t *domain.NotificationTemplate) (*domain.NotificationTemplate, error) {
	chJSON, err := json.Marshal(t.Channels)
	if err != nil {
		return nil, fmt.Errorf("repository: gagal marshal channels: %w", err)
	}
	row, err := r.queries.UpdateTemplate(ctx, UpdateTemplateParams{
		ID: parseUUID(t.ID), Name: t.Name, Channels: chJSON,
		BodyWhatsapp: stringToText(t.BodyWhatsApp), BodySms: stringToText(t.BodySMS),
		BodyEmailSubject: stringToText(t.BodyEmailSubject),
		BodyEmailHtml:    stringToText(t.BodyEmailHTML), IsActive: t.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, fmt.Errorf("repository: gagal memperbarui template: %w", err)
	}
	return mapTemplateRow(row)
}

// SoftDelete menonaktifkan template (is_active=false).
func (r *TemplateRepo) SoftDelete(ctx context.Context, id string) error {
	if err := r.queries.SoftDeleteTemplate(ctx, parseUUID(id)); err != nil {
		return fmt.Errorf("repository: gagal soft-delete template: %w", err)
	}
	return nil
}

// ListByTenant mengambil semua template untuk tenant, diurutkan created_at ASC.
func (r *TemplateRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.NotificationTemplate, error) {
	rows, err := r.queries.ListTemplatesByTenant(ctx, parseUUID(tenantID))
	if err != nil {
		return nil, fmt.Errorf("repository: gagal mengambil template by tenant: %w", err)
	}
	result := make([]*domain.NotificationTemplate, 0, len(rows))
	for _, row := range rows {
		t, err := mapTemplateRow(row)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

// BulkCreate membuat beberapa template sekaligus (untuk seeding bawaan templates).
func (r *TemplateRepo) BulkCreate(ctx context.Context, templates []*domain.NotificationTemplate) error {
	for _, t := range templates {
		chJSON, varJSON, err := buildCreateParams(t)
		if err != nil {
			return err
		}
		_, err = r.queries.BulkCreateTemplates(ctx, BulkCreateTemplatesParams{
			TenantID: parseUUID(t.TenantID), Slug: t.Slug, Name: t.Name,
			Category: string(t.Category), EventType: stringToText(t.EventType),
			Channels: chJSON, BodyWhatsapp: stringToText(t.BodyWhatsApp),
			BodySms: stringToText(t.BodySMS), BodyEmailSubject: stringToText(t.BodyEmailSubject),
			BodyEmailHtml: stringToText(t.BodyEmailHTML), Variables: varJSON,
			IsActive: t.IsActive, IsDefault: t.IsDefault,
		})
		if err != nil {
			return fmt.Errorf("repository: gagal bulk-create template '%s': %w", t.Slug, err)
		}
	}
	return nil
}

// SlugExists mengecek apakah slug sudah ada di tenant (exclude ID untuk perbarui).
func (r *TemplateRepo) SlugExists(ctx context.Context, tenantID, slug, excludeID string) (bool, error) {
	exists, err := r.queries.SlugExists(ctx, SlugExistsParams{
		TenantID: parseUUID(tenantID), Slug: slug, ID: parseUUID(excludeID),
	})
	if err != nil {
		return false, fmt.Errorf("repository: gagal mengecek slug exists: %w", err)
	}
	return exists, nil
}
