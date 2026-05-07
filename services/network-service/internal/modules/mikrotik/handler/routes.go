package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/middleware"
)

// RouterConfig berisi handler dan dependensi route khusus modul MikroTik.
type RouterConfig struct {
	RouterHandler       *RouterHandler
	StatusHandler       *StatusHandler
	PPPoEHandler        *PPPoEHandler
	SessionHandler      *SessionHandler
	VPNHandler          *VPNHandler
	OperationalHandler  *MikroTikOperationalHandler
	DHCPHandler         *DHCPHandler
	StaticIPHandler     *StaticIPHandler
	WalledGardenHandler *WalledGardenHandler
	HotspotHandler      *HotspotHandler
	TerminalHandler     *TerminalHandler
	BackupHandler       *BackupHandler
	BulkHandler         *MikroTikBulkHandler
	ModuleChecker       middleware.ModuleChecker
	Logger              zerolog.Logger
}

// RegisterRoutes mendaftarkan semua route publik modul MikroTik tanpa mengubah path lama.
func RegisterRoutes(api fiber.Router, cfg RouterConfig) {
	mikrotikGuard := middleware.RequireModule(domain.ModuleMikroTik, cfg.ModuleChecker, cfg.Logger)

	// Route CRUD router MikroTik.
	mikrotik := api.Group("/mikrotik", mikrotikGuard)
	routers := mikrotik.Group("/routers")
	routers.Post("/", cfg.RouterHandler.Create)
	routers.Get("/", cfg.RouterHandler.List)
	routers.Get("/:id", cfg.RouterHandler.GetByID)
	routers.Put("/:id", cfg.RouterHandler.Update)
	routers.Delete("/:id", cfg.RouterHandler.Delete)
	routers.Post("/:id/test", cfg.RouterHandler.TestConnection)
	routers.Post("/:id/reboot", cfg.RouterHandler.Reboot)

	// Route manajemen PPPoE user dan sessions.
	pppoe := routers.Group("/:id/pppoe")
	pppoe.Get("/users", cfg.PPPoEHandler.ListUsers)
	pppoe.Post("/users", cfg.PPPoEHandler.CreateUser)
	pppoe.Put("/users/:user_id", cfg.PPPoEHandler.UpdateUser)
	pppoe.Delete("/users/:user_id", cfg.PPPoEHandler.DeleteUser)
	pppoe.Post("/users/:user_id/disconnect", cfg.PPPoEHandler.DisconnectUser)
	pppoe.Get("/sync-status", cfg.PPPoEHandler.GetSyncStatus)
	pppoe.Post("/sync", cfg.PPPoEHandler.TriggerSync)

	// Route PPPoE active sessions.
	pppoe.Get("/sessions", cfg.SessionHandler.GetSessions)
	pppoe.Post("/sessions/:session_id/disconnect", cfg.SessionHandler.DisconnectSession)
	pppoe.Get("/sessions/count", cfg.SessionHandler.GetSessionCount)

	// Route operasional RouterOS. Semua dibaca manual/on-demand dari UI/API.
	routers.Get("/:id/interfaces", cfg.OperationalHandler.ListInterfaces)
	routers.Get("/:id/traffic", cfg.OperationalHandler.GetTraffic)
	routers.Get("/:id/ip-pools", cfg.OperationalHandler.ListIPPools)
	routers.Get("/:id/firewall/managed", cfg.OperationalHandler.ListManagedFirewall)
	routers.Get("/:id/logs", cfg.OperationalHandler.ListLogs)

	// Route DHCP server, leases, static bindings, dan networks.
	dhcp := routers.Group("/:id/dhcp")
	dhcp.Get("/servers", cfg.DHCPHandler.ListServers)
	dhcp.Get("/leases", cfg.DHCPHandler.ListLeases)
	dhcp.Get("/bindings", cfg.DHCPHandler.ListBindings)
	dhcp.Post("/bindings", cfg.DHCPHandler.CreateBinding)
	dhcp.Put("/bindings/:binding_id", cfg.DHCPHandler.UpdateBinding)
	dhcp.Delete("/bindings/:binding_id", cfg.DHCPHandler.DeleteBinding)
	dhcp.Get("/networks", cfg.DHCPHandler.ListNetworks)

	// Route static IP customer management.
	staticIP := routers.Group("/:id/static-ip")
	staticIP.Get("/assignments", cfg.StaticIPHandler.ListAssignments)
	staticIP.Post("/assignments", cfg.StaticIPHandler.CreateAssignment)
	staticIP.Put("/assignments/:assignment_id", cfg.StaticIPHandler.UpdateAssignment)
	staticIP.Delete("/assignments/:assignment_id", cfg.StaticIPHandler.DeleteAssignment)
	staticIP.Post("/assignments/:assignment_id/isolate", cfg.StaticIPHandler.IsolateAssignment)
	staticIP.Post("/assignments/:assignment_id/unisolate", cfg.StaticIPHandler.UnisolateAssignment)

	// Route walled garden isolir. Semua write manual dan idempotent by comment prefix.
	walledGarden := routers.Group("/:id/walled-garden")
	walledGarden.Get("", cfg.WalledGardenHandler.GetStatus)
	walledGarden.Post("/apply", cfg.WalledGardenHandler.Apply)
	walledGarden.Post("/remove", cfg.WalledGardenHandler.Remove)

	// Route Hotspot voucher users, profiles, active sessions, dan template login.
	hotspot := routers.Group("/:id/hotspot")
	hotspot.Get("/users", cfg.HotspotHandler.ListUsers)
	hotspot.Post("/users", cfg.HotspotHandler.CreateUser)
	hotspot.Put("/users/:user_id", cfg.HotspotHandler.UpdateUser)
	hotspot.Delete("/users/:user_id", cfg.HotspotHandler.DeleteUser)
	hotspot.Get("/profiles", cfg.HotspotHandler.ListProfiles)
	hotspot.Get("/active", cfg.HotspotHandler.ListActive)
	hotspot.Post("/login-template/generate", cfg.HotspotHandler.GenerateLoginTemplate)

	// Route terminal hanya baca dan audit command MikroTik.
	terminal := routers.Group("/:id/terminal")
	terminal.Post("/execute", cfg.TerminalHandler.Execute)
	terminal.Get("/audit", cfg.TerminalHandler.ListAudit)

	// Route backup export dan firmware hanya baca MikroTik.
	backups := routers.Group("/:id/backups")
	backups.Get("/", cfg.BackupHandler.List)
	backups.Post("/", cfg.BackupHandler.Create)
	backups.Get("/:backup_id/download", cfg.BackupHandler.Download)
	backups.Delete("/:backup_id", cfg.BackupHandler.Delete)
	routers.Get("/:id/firmware", cfg.BackupHandler.Firmware)

	// Route bulk action MikroTik. Semua action berjalan manual/on-demand.
	bulkJobs := mikrotik.Group("/bulk-jobs")
	bulkJobs.Get("/", cfg.BulkHandler.List)
	bulkJobs.Post("/", cfg.BulkHandler.Create)
	bulkJobs.Get("/:id", cfg.BulkHandler.Get)

	// Route status summary router MikroTik.
	status := mikrotik.Group("/status")
	status.Get("/summary", cfg.StatusHandler.GetSummary)

	// Route manajemen VPN tunnel.
	vpn := mikrotik.Group("/vpn")
	vpn.Get("/tunnels", cfg.VPNHandler.ListTunnels)
	vpn.Post("/tunnels", cfg.VPNHandler.CreateTunnel)
	vpn.Get("/tunnels/:id", cfg.VPNHandler.GetTunnel)
	vpn.Put("/tunnels/:id", cfg.VPNHandler.UpdateTunnel)
	vpn.Delete("/tunnels/:id", cfg.VPNHandler.DeleteTunnel)
	vpn.Post("/tunnels/:id/test", cfg.VPNHandler.TestConnection)
	vpn.Post("/tunnels/:id/auto-configure", cfg.VPNHandler.AutoConfigure)
	vpn.Get("/tunnels/:id/script", cfg.VPNHandler.GenerateScript)
	vpn.Get("/tunnels/:id/bandwidth", cfg.VPNHandler.GetBandwidth)
	vpn.Get("/summary", cfg.VPNHandler.GetSummary)
	vpn.Get("/maintenance", cfg.VPNHandler.GetUpcomingMaintenance)

	// Route admin VPN (maintenance scheduling).
	admin := api.Group("/admin/vpn", mikrotikGuard)
	admin.Post("/maintenance", cfg.VPNHandler.ScheduleMaintenance)
}
