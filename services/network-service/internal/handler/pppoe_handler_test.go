// pppoe_handler_test.go — unit test untuk PPPoEHandler.
// Menggunakan mock PPPoEManager dan Fiber app.Test() untuk HTTP testing.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/pkg/tenant"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/usecase"
)

type mockPPPoEMgr struct {
	createFn func(context.Context, string, domain.CreatePPPoEUserRequest) (*domain.PPPoEUser, error)
	syncFn   func(context.Context, string) (*domain.SyncResult, error)
}

func (m *mockPPPoEMgr) HandleCustomerActivated(context.Context, domain.CustomerActivatedPayload) error {
	return nil
}
func (m *mockPPPoEMgr) HandleIsolir(context.Context, domain.CustomerIsolirPayload) error { return nil }
func (m *mockPPPoEMgr) HandleUnIsolir(context.Context, domain.CustomerUnIsolirPayload) error {
	return nil
}
func (m *mockPPPoEMgr) HandleSuspend(context.Context, domain.CustomerSuspendPayload) error {
	return nil
}
func (m *mockPPPoEMgr) HandlePackageChanged(context.Context, domain.PackageChangedPayload) error {
	return nil
}
func (m *mockPPPoEMgr) SyncProfile(context.Context, *domain.PPPoEProfile) error { return nil }
func (m *mockPPPoEMgr) ListUsers(context.Context, string, domain.PPPoEUserListParams) (*domain.PPPoEUserListResult, error) {
	return nil, nil
}
func (m *mockPPPoEMgr) DeleteUser(context.Context, string, string) error { return nil }
func (m *mockPPPoEMgr) UpdateUser(context.Context, string, string, domain.UpdatePPPoEUserRequest) (*domain.PPPoEUser, error) {
	return nil, nil
}
func (m *mockPPPoEMgr) GetSyncStatus(context.Context, string) (*domain.SyncStatusSummary, error) {
	return nil, nil
}
func (m *mockPPPoEMgr) GetActiveSessions(context.Context, string) ([]domain.PPPoESession, error) {
	return nil, nil
}
func (m *mockPPPoEMgr) DisconnectSession(context.Context, string, string) error { return nil }
func (m *mockPPPoEMgr) DisconnectUser(context.Context, string, string) error    { return nil }
func (m *mockPPPoEMgr) GetSessionCount(context.Context, string) (int, error)    { return 0, nil }
func (m *mockPPPoEMgr) CreateUser(ctx context.Context, rid string, req domain.CreatePPPoEUserRequest) (*domain.PPPoEUser, error) {
	if m.createFn != nil {
		return m.createFn(ctx, rid, req)
	}
	return nil, nil
}
func (m *mockPPPoEMgr) SyncRouter(ctx context.Context, rid string) (*domain.SyncResult, error) {
	if m.syncFn != nil {
		return m.syncFn(ctx, rid)
	}
	return nil, nil
}

var _ usecase.PPPoEManager = (*mockPPPoEMgr)(nil)

func setupPPPoETestApp(mgr usecase.PPPoEManager) *fiber.App {
	app := fiber.New()
	h := NewPPPoEHandler(mgr, zerolog.Nop())
	app.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(tenant.SetForTest(c.UserContext(), testTenantID))
		return c.Next()
	})
	g := app.Group("/api/v1/mikrotik/routers/:id/pppoe")
	g.Post("/users", h.CreateUser)
	g.Post("/sync", h.TriggerSync)
	return app
}

// TestPPPoEErrorMapping — domain errors → HTTP status codes
func TestPPPoEErrorMapping(t *testing.T) {
	tt := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"PPPoEUserNotFound→404", domain.ErrPPPoEUserNotFound, 404, "PPPOE_USER_NOT_FOUND"},
		{"UsernameExists→409", domain.ErrPPPoEUsernameExists, 409, "PPPOE_USERNAME_EXISTS"},
		{"RouterNotFound→404", domain.ErrRouterNotFound, 404, "ROUTER_NOT_FOUND"},
		{"RouterOffline→503", domain.ErrRouterOffline, 503, "ROUTER_OFFLINE"},
		{"SyncInProgress→409", domain.ErrSyncInProgress, 409, "SYNC_IN_PROGRESS"},
		{"SessionNotFound→404", domain.ErrSessionNotFound, 404, "SESSION_NOT_FOUND"},
		{"ConnectionFailed→502", domain.ErrConnectionFailed, 502, "CONNECTION_FAILED"},
		{"ConnectionTimeout→504", domain.ErrConnectionTimeout, 504, "CONNECTION_TIMEOUT"},
		{"UnknownError→500", errors.New("boom"), 500, "INTERNAL_ERROR"},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			mgr := &mockPPPoEMgr{syncFn: func(context.Context, string) (*domain.SyncResult, error) {
				return nil, tc.err
			}}
			app := setupPPPoETestApp(mgr)
			req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/r1/pppoe/sync", nil)
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("request gagal: %v", err)
			}
			if resp.StatusCode != tc.status {
				t.Fatalf("expected %d, got %d", tc.status, resp.StatusCode)
			}
			ar := parseAPIResponse(t, resp.Body)
			if ar.Error == nil || ar.Error.Code != tc.code {
				t.Fatalf("expected code %s, got %v", tc.code, ar.Error)
			}
		})
	}
}

// TestCreateUser_Validation — field required menghasilkan 422
func TestCreateUser_Validation(t *testing.T) {
	uid := "550e8400-e29b-41d4-a716-446655440000"
	tt := []struct {
		name string
		body map[string]string
	}{
		{"MissingCustomerID", map[string]string{"username": "u1", "password": "p1", "profile_name": "10m"}},
		{"MissingUsername", map[string]string{"customer_id": uid, "password": "p1", "profile_name": "10m"}},
		{"MissingPassword", map[string]string{"customer_id": uid, "username": "u1", "profile_name": "10m"}},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			app := setupPPPoETestApp(&mockPPPoEMgr{})
			b, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/r1/pppoe/users", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				t.Fatalf("request gagal: %v", err)
			}
			if resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d", resp.StatusCode)
			}
			ar := parseAPIResponse(t, resp.Body)
			if ar.Error == nil || ar.Error.Code != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR, got %v", ar.Error)
			}
		})
	}
}

// TestCreateUser_SuccessFormat — respons sukses mengikuti format APIResponse
func TestCreateUser_SuccessFormat(t *testing.T) {
	mgr := &mockPPPoEMgr{createFn: func(context.Context, string, domain.CreatePPPoEUserRequest) (*domain.PPPoEUser, error) {
		return &domain.PPPoEUser{ID: "u1", Username: "user1", ProfileName: "10mbps"}, nil
	}}
	app := setupPPPoETestApp(mgr)
	b, _ := json.Marshal(domain.CreatePPPoEUserRequest{
		CustomerID: "550e8400-e29b-41d4-a716-446655440000",
		Username:   "user1", Password: "pass1", ProfileName: "10mbps",
	})
	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/r1/pppoe/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	ar := parseAPIResponse(t, resp.Body)
	if !ar.Success {
		t.Fatal("expected success=true")
	}
	if ar.Data == nil {
		t.Fatal("expected data tidak nil")
	}
}

// TestPPPoEErrorResponseFormat — respons error mengikuti format APIResponse
func TestPPPoEErrorResponseFormat(t *testing.T) {
	mgr := &mockPPPoEMgr{syncFn: func(context.Context, string) (*domain.SyncResult, error) {
		return nil, domain.ErrRouterNotFound
	}}
	app := setupPPPoETestApp(mgr)
	req := httptest.NewRequest("POST", "/api/v1/mikrotik/routers/r1/pppoe/sync", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request gagal: %v", err)
	}
	ar := parseAPIResponse(t, resp.Body)
	if ar.Success {
		t.Fatal("expected success=false")
	}
	if ar.Error == nil {
		t.Fatal("expected error tidak nil")
	}
	if ar.Error.Code == "" || ar.Error.Message == "" {
		t.Fatal("expected error code dan message tidak kosong")
	}
}
