package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
)

type MikroTikBulkManager interface {
	CreateJob(ctx context.Context, req domain.CreateMikroTikBulkJobRequest) (*domain.MikroTikBulkJob, error)
	GetJob(ctx context.Context, id string) (*domain.MikroTikBulkJob, error)
	ListJobs(ctx context.Context, params domain.MikroTikBulkJobListParams) (*domain.MikroTikBulkJobListResult, error)
}

type mikrotikBulkManager struct {
	routerRepo    domain.RouterRepository
	jobRepo       domain.MikroTikBulkJobRepository
	backupManager BackupManager
	pppoeManager  PPPoEManager
}

func NewMikroTikBulkManager(
	routerRepo domain.RouterRepository,
	jobRepo domain.MikroTikBulkJobRepository,
	backupManager BackupManager,
	pppoeManager PPPoEManager,
) MikroTikBulkManager {
	return &mikrotikBulkManager{
		routerRepo: routerRepo, jobRepo: jobRepo, backupManager: backupManager, pppoeManager: pppoeManager,
	}
}

func (m *mikrotikBulkManager) CreateJob(ctx context.Context, req domain.CreateMikroTikBulkJobRequest) (*domain.MikroTikBulkJob, error) {
	action := domain.MikroTikBulkAction(strings.TrimSpace(string(req.Action)))
	if !isValidMikroTikBulkAction(action) {
		return nil, domain.ErrInvalidBulkAction
	}

	routers, err := m.resolveRouters(ctx, req)
	if err != nil {
		return nil, err
	}
	routerIDs := make([]string, 0, len(routers))
	for _, router := range routers {
		routerIDs = append(routerIDs, router.ID)
	}

	now := time.Now().UTC()
	actor, _ := ctx.Value(mikrotikAuditActorKey).(mikrotikAuditActor)
	job, err := m.jobRepo.Create(ctx, domain.CreateMikroTikBulkJobInput{
		TenantID: tenant.FromContext(ctx), Action: action, Status: domain.MikroTikBulkJobQueued,
		RouterIDs: routerIDs, TotalCount: len(routers), RequestedBy: actor.UserID,
	})
	if err != nil {
		return nil, err
	}
	if err := m.jobRepo.MarkRunning(ctx, job.ID, now); err != nil {
		return nil, err
	}

	results := make([]domain.MikroTikBulkJobResult, 0, len(routers))
	successCount := 0
	failedCount := 0
	for _, router := range routers {
		result := m.runRouterAction(ctx, action, router)
		if result.Status == "success" {
			successCount++
		} else {
			failedCount++
		}
		results = append(results, result)
	}

	finishedAt := time.Now().UTC()
	status := domain.MikroTikBulkJobSucceeded
	errorMessage := ""
	switch {
	case len(routers) == 0:
		status = domain.MikroTikBulkJobFailed
		errorMessage = "tidak ada router untuk diproses"
	case successCount == 0 && failedCount > 0:
		status = domain.MikroTikBulkJobFailed
		errorMessage = "semua router gagal diproses"
	case failedCount > 0:
		status = domain.MikroTikBulkJobPartialFailed
		errorMessage = fmt.Sprintf("%d dari %d router gagal", failedCount, len(routers))
	}

	return m.jobRepo.Complete(ctx, domain.UpdateMikroTikBulkJobResultInput{
		ID: job.ID, Status: status, SuccessCount: successCount, FailedCount: failedCount,
		Results: results, ErrorMessage: errorMessage, FinishedAt: &finishedAt,
	})
}

func (m *mikrotikBulkManager) GetJob(ctx context.Context, id string) (*domain.MikroTikBulkJob, error) {
	return m.jobRepo.GetByID(ctx, id)
}

func (m *mikrotikBulkManager) ListJobs(ctx context.Context, params domain.MikroTikBulkJobListParams) (*domain.MikroTikBulkJobListResult, error) {
	return m.jobRepo.List(ctx, params)
}

func (m *mikrotikBulkManager) resolveRouters(ctx context.Context, req domain.CreateMikroTikBulkJobRequest) ([]*domain.Router, error) {
	scope := strings.TrimSpace(req.Scope)
	if scope == "all_active" || len(req.RouterIDs) == 0 {
		return m.routerRepo.GetActiveRouters(ctx)
	}

	seen := map[string]struct{}{}
	routers := make([]*domain.Router, 0, len(req.RouterIDs))
	for _, rawID := range req.RouterIDs {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		router, err := m.routerRepo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		routers = append(routers, router)
	}
	return routers, nil
}

func (m *mikrotikBulkManager) runRouterAction(ctx context.Context, action domain.MikroTikBulkAction, router *domain.Router) domain.MikroTikBulkJobResult {
	startedAt := time.Now().UTC()
	result := domain.MikroTikBulkJobResult{
		RouterID: router.ID, RouterName: router.Name, Action: string(action),
		Status: "success", StartedAt: startedAt,
	}

	switch action {
	case domain.MikroTikBulkActionBackup:
		backup, err := m.backupManager.CreateBackup(ctx, router.ID)
		if err != nil {
			return failedBulkResult(result, err)
		}
		result.Message = "backup selesai"
		result.Details = map[string]any{
			"backup_id": backup.ID, "file_name": backup.FileName, "size_bytes": backup.SizeBytes,
		}
	case domain.MikroTikBulkActionFirmwareCheck:
		info, err := m.backupManager.GetFirmware(ctx, router.ID)
		if err != nil {
			return failedBulkResult(result, err)
		}
		result.Message = "firmware terbaca"
		result.Details = map[string]any{
			"routeros_version": info.RouterOSVersion,
			"architecture":     info.Architecture,
			"board_name":       info.BoardName,
			"package_count":    len(info.Packages),
			"outdated":         info.Outdated,
			"warning":          info.Warning,
		}
	case domain.MikroTikBulkActionPPPoESync:
		syncResult, err := m.pppoeManager.SyncRouter(ctx, router.ID)
		if err != nil {
			return failedBulkResult(result, err)
		}
		result.Message = "sync pppoe selesai"
		result.Details = map[string]any{
			"synced_count":      syncResult.SyncedCount,
			"orphan_count":      syncResult.OrphanCount,
			"missing_count":     syncResult.MissingCount,
			"out_of_sync_count": syncResult.OutOfSyncCount,
			"error_count":       syncResult.ErrorCount,
		}
	default:
		return failedBulkResult(result, domain.ErrInvalidBulkAction)
	}

	result.FinishedAt = time.Now().UTC()
	return result
}

func failedBulkResult(result domain.MikroTikBulkJobResult, err error) domain.MikroTikBulkJobResult {
	result.Status = "failed"
	if err != nil {
		result.Message = err.Error()
	}
	if errors.Is(err, domain.ErrRouterPermissionDenied) {
		result.Message = "permission user router tidak mencukupi"
	}
	result.FinishedAt = time.Now().UTC()
	return result
}

func isValidMikroTikBulkAction(action domain.MikroTikBulkAction) bool {
	switch action {
	case domain.MikroTikBulkActionBackup, domain.MikroTikBulkActionFirmwareCheck, domain.MikroTikBulkActionPPPoESync:
		return true
	default:
		return false
	}
}
