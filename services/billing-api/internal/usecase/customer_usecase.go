// customer_usecase.go berisi business logic untuk manajemen pelanggan.
// Mengimplementasikan Buat, GetByID, Perbarui, SoftDelete, List, dan Stats.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// ActorInfo berisi informasi aktor yang melakukan operasi.
// Diambil dari JWT claims dan user data oleh handler.
type ActorInfo struct {
	ID   string
	Name string
}

// CustomerUsecase mengimplementasikan business logic untuk manajemen pelanggan.
type CustomerUsecase struct {
	customerRepo domain.CustomerRepository
	packageRepo  domain.PackageRepository
	moduleRepo   TenantModuleRepository
	auditLogRepo domain.AuditLogRepository
	queueClient  *asynq.Client
	logger       zerolog.Logger
}

// NewCustomerUsecase membuat instance baru CustomerUsecase.
func NewCustomerUsecase(
	customerRepo domain.CustomerRepository,
	auditLogRepo domain.AuditLogRepository,
	queueClient *asynq.Client,
	logger zerolog.Logger,
) *CustomerUsecase {
	return &CustomerUsecase{
		customerRepo: customerRepo,
		auditLogRepo: auditLogRepo,
		queueClient:  queueClient,
		logger:       logger,
	}
}

// SetPackageRepository memasang package repositori opsional untuk memperkaya event jaringan.
// Constructor lama dipertahankan agar test dan modul lain tetap kompatibel.
func (uc *CustomerUsecase) SetPackageRepository(packageRepo domain.PackageRepository) {
	uc.packageRepo = packageRepo
}

// SetTenantModuleRepository memasang repositori entitlement agar event teknis
// jaringan hanya dikirim saat add-on terkait aktif.
func (uc *CustomerUsecase) SetTenantModuleRepository(moduleRepo TenantModuleRepository) {
	uc.moduleRepo = moduleRepo
}

// Buat membuat pelanggan baru.
// Alur: validasi -> cek phone duplicate -> get max seq -> buat customer ID ->
// tulis audit log -> terbitkan customer.created event.
func (uc *CustomerUsecase) Create(ctx context.Context, tenantID string, req domain.CreateCustomerRequest, actor ActorInfo) (*domain.Customer, error) {
	// Periksa phone duplicate
	exists, err := uc.customerRepo.PhoneExists(ctx, tenantID, req.Phone, "")
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal cek phone duplicate: %w", err)
	}
	if exists {
		return nil, domain.ErrPhoneDuplicate
	}

	// Get max sequence number untuk this tenant
	maxSeq, err := uc.customerRepo.GetMaxSeq(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal ambil max seq: %w", err)
	}

	// Buat customer ID
	customerIDSeq := domain.GenerateCustomerID(maxSeq)

	// Parsing activation date
	activationDate, err := time.Parse("2006-01-02", req.ActivationDate)
	if err != nil {
		return nil, fmt.Errorf("usecase: format activation_date tidak valid: %w", err)
	}

	// Auto-buat PPPoE credentials jika needed
	pppoeUsername := req.PPPoEUsername
	pppoePassword := req.PPPoEPassword
	if domain.ConnectionMethod(req.ConnectionMethod) == domain.ConnectionPPPoE {
		if pppoeUsername == "" {
			pppoeUsername = domain.GeneratePPPoEUsername(req.Name, customerIDSeq)
		}
		if pppoePassword == "" {
			pppoePassword = domain.GeneratePPPoEPassword()
		}
	}

	// Bangun customer entity
	customer := &domain.Customer{
		TenantID:         tenantID,
		CustomerIDSeq:    customerIDSeq,
		Name:             req.Name,
		Phone:            req.Phone,
		Email:            req.Email,
		Address:          req.Address,
		AreaID:           req.AreaID,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		PackageID:        req.PackageID,
		ActivationDate:   activationDate,
		DueDate:          req.DueDate,
		ConnectionMethod: domain.ConnectionMethod(req.ConnectionMethod),
		PPPoEUsername:    pppoeUsername,
		PPPoEPassword:    pppoePassword,
		MACAddress:       req.MACAddress,
		RouterID:         req.RouterID,
		ODPPort:          req.ODPPort,
		Notes:            req.Notes,
		Status:           domain.CustomerStatusPending,
	}

	// Buat customer in database
	created, err := uc.customerRepo.Create(ctx, customer)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal membuat customer: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, tenantID, created.ID, "customer.created", actor, nil)

	if uc.mikrotikEnabled(ctx, tenantID) {
		uc.publishEvent(tenantID, "customer.created", domain.CustomerCreatedPayload{
			CustomerID:       created.ID,
			Name:             created.Name,
			PackageID:        created.PackageID,
			ConnectionMethod: string(created.ConnectionMethod),
			RouterID:         created.RouterID,
		})
	}

	return created, nil
}

// GetByID mengambil detail pelanggan berdasarkan ID.
// Jika includeAudit true, audit logs juga disertakan.
func (uc *CustomerUsecase) GetByID(ctx context.Context, id string, includeAudit bool) (*domain.CustomerDetail, error) {
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Periksa jika hapus lunak
	if customer.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	detail := &domain.CustomerDetail{
		Customer: customer,
	}

	// Ambil audit log jika diminta
	if includeAudit {
		logs, err := uc.auditLogRepo.ListByEntity(ctx, "customer", id)
		if err != nil {
			uc.logger.Error().Err(err).Str("customer_id", id).Msg("gagal mengambil audit logs")
			// Don't fail the permintaan, just skip audit logs
		} else {
			detail.AuditLogs = logs
		}
	}

	return detail, nil
}

// Perbarui memperbarui data pelanggan.
// Alur: validasi -> cek phone duplicate -> perbarui -> compute changed field ->
// tulis audit log dengan nilai lama/baru.
func (uc *CustomerUsecase) Update(ctx context.Context, id string, req domain.UpdateCustomerRequest, actor ActorInfo) (*domain.Customer, error) {
	// Ambil existing customer
	existing, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if existing.DeletedAt != nil {
		return nil, domain.ErrCustomerNotFound
	}

	// Periksa phone duplicate jika phone is being changed
	if req.Phone != "" && req.Phone != existing.Phone {
		exists, err := uc.customerRepo.PhoneExists(ctx, existing.TenantID, req.Phone, id)
		if err != nil {
			return nil, fmt.Errorf("usecase: gagal cek phone duplicate: %w", err)
		}
		if exists {
			return nil, domain.ErrPhoneDuplicate
		}
	}

	updated := applyCustomerUpdates(existing, req)

	// Simpan to database
	result, err := uc.customerRepo.Update(ctx, updated)
	if err != nil {
		return nil, fmt.Errorf("usecase: gagal memperbarui customer: %w", err)
	}

	// Compute changed field untuk audit log
	changes := computeChanges(existing, result)

	// Tulis audit log dengan nilai lama/baru
	if len(changes) > 0 {
		uc.writeAuditLog(ctx, existing.TenantID, id, "customer.updated", actor, changes)
	}

	return result, nil
}

// SoftDelete menghapus pelanggan secara hapus lunak.
// tulis audit log -> terbitkan customer.terminated event.
func (uc *CustomerUsecase) SoftDelete(ctx context.Context, id string, confirmationName string, actor ActorInfo) error {
	// Ambil existing customer
	customer, err := uc.customerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if customer.DeletedAt != nil {
		return domain.ErrCustomerNotFound
	}

	// Verify confirmation name matches (case-sensitive)
	if confirmationName != customer.Name {
		return domain.ErrConfirmationMismatch
	}

	// Soft hapus
	if err := uc.customerRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("usecase: gagal soft-delete customer: %w", err)
	}

	// Tulis audit log
	uc.writeAuditLog(ctx, customer.TenantID, id, "customer.deleted", actor, nil)

	if uc.mikrotikEnabled(ctx, customer.TenantID) {
		uc.publishEvent(customer.TenantID, "customer.terminated", domain.CustomerTerminatedPayload{
			CustomerID:       customer.ID,
			TenantID:         customer.TenantID,
			Name:             customer.Name,
			RouterID:         customer.RouterID,
			PPPoEUsername:    customer.PPPoEUsername,
			ConnectionMethod: string(customer.ConnectionMethod),
		})
	}

	return nil
}

func (uc *CustomerUsecase) packageNetworkFields(ctx context.Context, packageID string) (profileName string, downloadMbps int, uploadMbps int, addressPool string) {
	if uc.packageRepo == nil || packageID == "" {
		return "", 0, 0, ""
	}
	pkg, err := uc.packageRepo.GetByID(ctx, packageID)
	if err != nil {
		uc.logger.Warn().Err(err).Str("package_id", packageID).Msg("gagal mengambil paket untuk payload network")
		return "", 0, 0, ""
	}
	return pkg.MikrotikProfileName, pkg.DownloadMbps, pkg.UploadMbps, pkg.AddressPool
}

func (uc *CustomerUsecase) mikrotikEnabled(ctx context.Context, tenantID string) bool {
	if uc.moduleRepo == nil {
		return true
	}
	caps, err := uc.moduleRepo.Capabilities(ctx, tenantID)
	if err != nil {
		uc.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("gagal cek entitlement MikroTik")
		return false
	}
	return caps.MikroTik
}

// List mengambil daftar pelanggan dengan paginasi, filter, dan pengurutan.
// Menerapkan bawaan: page=1, page_size=25.
func (uc *CustomerUsecase) List(ctx context.Context, params domain.CustomerListParams) (*domain.CustomerListResult, error) {
	// Terapkan defaults
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}

	return uc.customerRepo.List(ctx, params)
}

// Stats mengembalikan jumlah pelanggan per status.
func (uc *CustomerUsecase) Stats(ctx context.Context) (map[domain.CustomerStatus]int64, error) {
	return uc.customerRepo.CountByStatus(ctx)
}

// --- Fungsi bantu functions ---

// writeAuditLog menulis audit log entry. Tidak mengembalikan error agar
// operasi utama tidak gagal karena audit log.
func (uc *CustomerUsecase) writeAuditLog(ctx context.Context, tenantID, entityID, action string, actor ActorInfo, changes map[string]interface{}) {
	log := &domain.AuditLog{
		TenantID:   tenantID,
		EntityType: "customer",
		EntityID:   entityID,
		Action:     action,
		ActorID:    actor.ID,
		ActorName:  actor.Name,
		Changes:    changes,
	}

	if err := uc.auditLogRepo.Create(ctx, log); err != nil {
		uc.logger.Error().Err(err).
			Str("entity_id", entityID).
			Str("action", action).
			Msg("gagal menulis audit log")
	}
}

// publishEvent mempublikasikan event ke Redis queue.
// Tidak mengembalikan error agar operasi utama tidak gagal.
func (uc *CustomerUsecase) publishEvent(tenantID, eventType string, payload interface{}) {
	if uc.queueClient == nil {
		return
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal marshal event payload")
		return
	}

	envelope := queue.TaskEnvelope{
		EventType: eventType,
		TenantID:  tenantID,
		Payload:   payloadJSON,
	}

	if err := queue.EnqueueTaskWithOptions(uc.queueClient, envelope, customerEventQueueOptions(eventType)...); err != nil {
		uc.logger.Error().Err(err).Str("event_type", eventType).Msg("gagal publish event")
	}
}

func customerEventQueueOptions(eventType string) []asynq.Option {
	switch eventType {
	case "customer.activated",
		domain.TaskCustomerIsolir,
		domain.TaskCustomerUnIsolir,
		domain.TaskCustomerSuspend,
		"customer.terminated",
		"package.changed":
		return []asynq.Option{asynq.Queue("critical")}
	default:
		return nil
	}
}

// applyCustomerUpdates menerapkan perubahan dari UpdateCustomerRequest ke Customer.
// Hanya field yang non-zero/non-empty yang diperbarui.
func applyCustomerUpdates(existing *domain.Customer, req domain.UpdateCustomerRequest) *domain.Customer {
	updated := *existing // copy

	if req.Name != "" {
		updated.Name = req.Name
	}
	if req.Phone != "" {
		updated.Phone = req.Phone
	}
	if req.Email != "" {
		updated.Email = req.Email
	}
	if req.Address != "" {
		updated.Address = req.Address
	}
	if req.AreaID != "" {
		updated.AreaID = req.AreaID
	}
	if req.Latitude != nil {
		updated.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		updated.Longitude = *req.Longitude
	}
	if req.PackageID != "" {
		updated.PackageID = req.PackageID
	}
	if req.ActivationDate != "" {
		if t, err := time.Parse("2006-01-02", req.ActivationDate); err == nil {
			updated.ActivationDate = t
		}
	}
	if req.DueDate != nil {
		updated.DueDate = *req.DueDate
	}
	if req.ConnectionMethod != "" {
		updated.ConnectionMethod = domain.ConnectionMethod(req.ConnectionMethod)
	}
	if req.PPPoEUsername != "" {
		updated.PPPoEUsername = req.PPPoEUsername
	}
	if req.PPPoEPassword != "" {
		updated.PPPoEPassword = req.PPPoEPassword
	}
	if req.MACAddress != "" {
		updated.MACAddress = req.MACAddress
	}
	if req.RouterID != "" {
		updated.RouterID = req.RouterID
	}
	if req.ODPPort != "" {
		updated.ODPPort = req.ODPPort
	}
	if req.Notes != "" {
		updated.Notes = req.Notes
	}

	return &updated
}

// computeChanges menghitung field yang berubah antara old dan new customer.
// Mengembalikan map dengan format {"field": {"old": oldVal, "new": newVal}}.
func computeChanges(old, new *domain.Customer) map[string]interface{} {
	changes := make(map[string]interface{})

	if old.Name != new.Name {
		changes["name"] = map[string]interface{}{"old": old.Name, "new": new.Name}
	}
	if old.Phone != new.Phone {
		changes["phone"] = map[string]interface{}{"old": old.Phone, "new": new.Phone}
	}
	if old.Email != new.Email {
		changes["email"] = map[string]interface{}{"old": old.Email, "new": new.Email}
	}
	if old.Address != new.Address {
		changes["address"] = map[string]interface{}{"old": old.Address, "new": new.Address}
	}
	if old.AreaID != new.AreaID {
		changes["area_id"] = map[string]interface{}{"old": old.AreaID, "new": new.AreaID}
	}
	if old.Latitude != new.Latitude {
		changes["latitude"] = map[string]interface{}{"old": old.Latitude, "new": new.Latitude}
	}
	if old.Longitude != new.Longitude {
		changes["longitude"] = map[string]interface{}{"old": old.Longitude, "new": new.Longitude}
	}
	if old.PackageID != new.PackageID {
		changes["package_id"] = map[string]interface{}{"old": old.PackageID, "new": new.PackageID}
	}
	if !old.ActivationDate.Equal(new.ActivationDate) {
		changes["activation_date"] = map[string]interface{}{"old": old.ActivationDate, "new": new.ActivationDate}
	}
	if old.DueDate != new.DueDate {
		changes["due_date"] = map[string]interface{}{"old": old.DueDate, "new": new.DueDate}
	}
	if old.ConnectionMethod != new.ConnectionMethod {
		changes["connection_method"] = map[string]interface{}{"old": old.ConnectionMethod, "new": new.ConnectionMethod}
	}
	if old.PPPoEUsername != new.PPPoEUsername {
		changes["pppoe_username"] = map[string]interface{}{"old": old.PPPoEUsername, "new": new.PPPoEUsername}
	}
	if old.PPPoEPassword != new.PPPoEPassword {
		changes["pppoe_password"] = map[string]interface{}{"old": "***", "new": "***"}
	}
	if old.MACAddress != new.MACAddress {
		changes["mac_address"] = map[string]interface{}{"old": old.MACAddress, "new": new.MACAddress}
	}
	if old.RouterID != new.RouterID {
		changes["router_id"] = map[string]interface{}{"old": old.RouterID, "new": new.RouterID}
	}
	if old.ODPPort != new.ODPPort {
		changes["odp_port"] = map[string]interface{}{"old": old.ODPPort, "new": new.ODPPort}
	}
	if old.Notes != new.Notes {
		changes["notes"] = map[string]interface{}{"old": old.Notes, "new": new.Notes}
	}
	if old.Status != new.Status {
		changes["status"] = map[string]interface{}{"old": old.Status, "new": new.Status}
	}

	return changes
}

// ComputePaginationMeta menghitung metadata paginasi.
// Digunakan oleh usecase dan test.
func ComputePaginationMeta(total int64, page, pageSize int) domain.PaginationMeta {
	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	if totalPages < 1 {
		totalPages = 1
	}

	return domain.PaginationMeta{
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
