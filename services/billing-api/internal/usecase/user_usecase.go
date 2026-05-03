package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog/log"

	"github.com/ispboss/ispboss/pkg/queue"
	"github.com/ispboss/ispboss/services/billing-api/internal/domain"
)

// UserManagementUsecaseConfig berisi semua dependensi yang dibutuhkan UserManagementUsecase.
type UserManagementUsecaseConfig struct {
	UserRepo    domain.UserRepository
	SessionRepo domain.SessionRepository
	TokenRepo   domain.TokenRepository
	QueueClient *asynq.Client
	BcryptCost  int
}

// UserManagementUsecase mengimplementasikan business logic manajemen user oleh tenant admin.
type UserManagementUsecase struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	tokenRepo   domain.TokenRepository
	queueClient *asynq.Client
	bcryptCost  int
}

// NewUserManagementUsecase membuat instance baru UserManagementUsecase.
func NewUserManagementUsecase(cfg UserManagementUsecaseConfig) *UserManagementUsecase {
	return &UserManagementUsecase{
		userRepo:    cfg.UserRepo,
		sessionRepo: cfg.SessionRepo,
		tokenRepo:   cfg.TokenRepo,
		queueClient: cfg.QueueClient,
		bcryptCost:  cfg.BcryptCost,
	}
}

// CreateUser membuat user baru dalam tenant.
// Admin-created users langsung email_verified=true (skip verifikasi email).
// Role super_admin tidak boleh dibuat oleh tenant admin.
func (uc *UserManagementUsecase) CreateUser(ctx context.Context, tenantID string, req domain.CreateUserRequest) (*domain.User, error) {
	// Tolak pembuatan user dengan role super_admin
	if req.Role == domain.RoleSuperAdmin {
		return nil, domain.ErrInvalidRole
	}

	// Cek apakah email sudah terdaftar di tenant ini
	_, err := uc.userRepo.GetByTenantAndEmail(ctx, tenantID, req.Email)
	if err == nil {
		// User sudah ada dengan email tersebut
		return nil, domain.ErrEmailAlreadyExists
	}
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("gagal mengecek email: %w", err)
	}

	// Hash password dengan bcrypt
	hashedPassword, err := HashPassword(req.Password, uc.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("gagal hash password: %w", err)
	}

	// Buat user baru (email_verified=true untuk user yang dibuat admin)
	user, err := uc.userRepo.CreateUser(ctx, &domain.User{
		TenantID:      tenantID,
		Name:          req.Name,
		Email:         req.Email,
		Phone:         req.Phone,
		PasswordHash:  hashedPassword,
		Role:          req.Role,
		EmailVerified: true,
		Status:        domain.UserStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat user: %w", err)
	}

	return user, nil
}

// UpdateUser memperbarui data user (name, phone, role saja).
// tenant_id tidak boleh diubah.
func (uc *UserManagementUsecase) UpdateUser(ctx context.Context, userID string, req domain.UpdateUserRequest) (*domain.User, error) {
	// Ambil user yang akan diupdate
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Update field yang diberikan (preserve tenant_id)
	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Role != "" {
		user.Role = req.Role
	}

	// Simpan perubahan
	updated, err := uc.userRepo.UpdateUser(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("gagal update user: %w", err)
	}

	return updated, nil
}

// DeactivateUser menonaktifkan user dan menghapus semua session aktif.
// User tidak boleh menonaktifkan akun sendiri.
func (uc *UserManagementUsecase) DeactivateUser(ctx context.Context, userID string, callerID string) error {
	// Tolak self-deactivation
	if userID == callerID {
		return domain.ErrCannotDeactivateSelf
	}

	// Set status user menjadi inactive
	if err := uc.userRepo.UpdateStatus(ctx, userID, domain.UserStatusInactive); err != nil {
		return fmt.Errorf("gagal menonaktifkan user: %w", err)
	}

	// Hapus semua session aktif user
	if err := uc.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		log.Error().Err(err).Str("user_id", userID).Msg("gagal menghapus session user yang dinonaktifkan")
	}

	return nil
}

// ActivateUser mengaktifkan kembali user yang dinonaktifkan.
func (uc *UserManagementUsecase) ActivateUser(ctx context.Context, userID string) error {
	if err := uc.userRepo.UpdateStatus(ctx, userID, domain.UserStatusActive); err != nil {
		return fmt.Errorf("gagal mengaktifkan user: %w", err)
	}
	return nil
}

// DeleteUser menghapus user secara permanen.
// User tidak boleh menghapus akun sendiri.
// confirmName harus cocok dengan nama user sebagai konfirmasi.
func (uc *UserManagementUsecase) DeleteUser(ctx context.Context, userID string, callerID string, confirmName string) error {
	// Tolak self-deletion
	if userID == callerID {
		return domain.ErrCannotDeleteSelf
	}

	// Ambil user untuk verifikasi confirmName
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Verifikasi confirmName cocok dengan nama user
	if confirmName != user.Name {
		return fmt.Errorf("nama konfirmasi tidak cocok")
	}

	// Hapus user (cascade akan menghapus sessions, password_resets, email_verifications)
	if err := uc.userRepo.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("gagal menghapus user: %w", err)
	}

	return nil
}

// ResetUserPassword mengirim email reset password ke user target.
// Generate token reset dan enqueue email via queue.
func (uc *UserManagementUsecase) ResetUserPassword(ctx context.Context, userID string) error {
	// Ambil data user
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("gagal mengambil user: %w", err)
	}

	// Invalidate semua token reset lama
	if err := uc.tokenRepo.InvalidatePasswordResets(ctx, userID); err != nil {
		return fmt.Errorf("gagal invalidate token lama: %w", err)
	}

	// Generate token reset password baru
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		return fmt.Errorf("gagal generate token: %w", err)
	}

	// Simpan hash token di database (berlaku 1 jam)
	err = uc.tokenRepo.CreatePasswordReset(ctx, &domain.PasswordReset{
		UserID:    userID,
		TokenHash: tokenHash,
	})
	if err != nil {
		return fmt.Errorf("gagal menyimpan token reset: %w", err)
	}

	// Enqueue email reset password
	uc.enqueueResetEmail(user.TenantID, userID, user.Email, user.Name, plainToken)

	return nil
}

// ListUsers mengambil daftar semua user dalam satu tenant.
func (uc *UserManagementUsecase) ListUsers(ctx context.Context, tenantID string) ([]*domain.User, error) {
	users, err := uc.userRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar user: %w", err)
	}
	return users, nil
}

// GetUser mengambil detail user berdasarkan ID.
func (uc *UserManagementUsecase) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil user: %w", err)
	}
	return user, nil
}

// enqueueResetEmail mengirim task email reset password ke queue.
func (uc *UserManagementUsecase) enqueueResetEmail(tenantID, userID, email, name, token string) {
	if uc.queueClient == nil {
		log.Warn().Msg("queue client tidak tersedia, skip enqueue reset email")
		return
	}

	payload, _ := json.Marshal(map[string]string{
		"user_id": userID,
		"email":   email,
		"name":    name,
		"token":   token,
	})

	err := queue.EnqueueTask(uc.queueClient, queue.TaskEnvelope{
		EventType: "email.password_reset",
		TenantID:  tenantID,
		Payload:   payload,
	})
	if err != nil {
		log.Error().Err(err).Str("email", email).Msg("gagal enqueue email reset password")
	}
}
