package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/middleware"
	"github.com/rs/zerolog"
)

// RouterConfig berisi dependensi yang dibutuhkan untuk registrasi route.
type RouterConfig struct {
	// App adalah instance Fiber application
	App *fiber.App

	// HealthHandler adalah handler untuk health check endpoint
	HealthHandler *HealthHandler

	// RouterHandler adalah handler untuk operasi CRUD router MikroTik
	RouterHandler *RouterHandler

	// StatusHandler adalah handler untuk ringkasan status router
	StatusHandler *StatusHandler

	// PPPoEHandler adalah handler untuk manajemen PPPoE user
	PPPoEHandler *PPPoEHandler

	// SessionHandler adalah handler untuk manajemen PPPoE active sessions
	SessionHandler *SessionHandler

	// VPNHandler adalah handler untuk manajemen VPN tunnel
	VPNHandler *VPNHandler

	// OperationalHandler adalah handler untuk data operasional RouterOS on-demand
	OperationalHandler *MikroTikOperationalHandler

	// DHCPHandler adalah handler untuk DHCP server/lease/static binding
	DHCPHandler *DHCPHandler

	// StaticIPHandler adalah handler untuk pelanggan static IP
	StaticIPHandler *StaticIPHandler

	// WalledGardenHandler adalah handler untuk rule isolir/walled garden
	WalledGardenHandler *WalledGardenHandler

	// HotspotHandler adalah handler untuk user/profile/session Hotspot
	HotspotHandler *HotspotHandler

	// TerminalHandler adalah handler untuk terminal read-only dan audit command
	TerminalHandler *TerminalHandler

	// BackupHandler adalah handler untuk backup export dan firmware MikroTik
	BackupHandler *BackupHandler

	// BulkHandler adalah handler untuk bulk action MikroTik on-demand
	BulkHandler *MikroTikBulkHandler

	// OLTHandler adalah handler untuk manajemen OLT device
	OLTHandler *OLTHandler

	// ODPHandler adalah handler untuk manajemen ODP/splitter
	ODPHandler *ODPHandler

	// ProvisioningHandler adalah handler untuk provisioning ONT (single, bulk, decommission, reboot)
	ProvisioningHandler *ProvisioningHandler

	// VLANHandler adalah handler untuk manajemen VLAN per OLT
	VLANHandler *VLANHandler

	// ServiceProfileHandler adalah handler untuk manajemen service profile per OLT
	ServiceProfileHandler *ServiceProfileHandler

	// MapNodeHandler adalah handler untuk manajemen map node (CRUD, foto, riwayat)
	MapNodeHandler *MapNodeHandler

	// CableRouteHandler adalah handler untuk manajemen cable route (CRUD)
	CableRouteHandler *CableRouteHandler

	// SearchHandler adalah handler untuk pencarian map node
	SearchHandler *SearchHandler

	// ExportHandler adalah handler untuk export peta (KML, KMZ, GeoJSON, CSV)
	ExportHandler *ExportHandler

	// ImportHandler adalah handler untuk import peta (KML, KMZ, GeoJSON)
	ImportHandler *ImportHandler

	// GeocodingHandler adalah handler untuk reverse geocoding
	GeocodingHandler *GeocodingHandler

	// ShareHandler adalah handler untuk share link peta read-only
	ShareHandler *ShareHandler

	// LossCalcHandler adalah handler untuk kalkulasi optical loss budget
	LossCalcHandler *LossCalcHandler

	// LabelSettingsHandler adalah handler untuk konfigurasi label peta
	LabelSettingsHandler *LabelSettingsHandler

	// TrashHandler adalah handler untuk manajemen trash (soft-delete)
	TrashHandler *TrashHandler

	// ModuleChecker memeriksa entitlement modul add-on per tenant
	ModuleChecker middleware.ModuleChecker

	// JWTSecret adalah secret key untuk validasi JWT token
	JWTSecret string

	// Logger adalah instance zerolog untuk request logging
	Logger zerolog.Logger
}

// RegisterRoutes mendaftarkan semua route pada Fiber app.
// Health check endpoint bersifat publik (tanpa auth).
// Route lainnya dilindungi oleh auth dan tenant middleware.
func RegisterRoutes(cfg RouterConfig) {
	// Middleware logging untuk semua request
	cfg.App.Use(middleware.RequestLogger(cfg.Logger))

	// Route publik — health check (tanpa autentikasi)
	cfg.App.Get("/healthz", cfg.HealthHandler.Healthz)
	cfg.App.Get("/readyz", cfg.HealthHandler.Readyz)

	// Grup route yang dilindungi oleh auth dan tenant middleware
	api := cfg.App.Group("/api/v1")
	api.Use(middleware.Auth(cfg.JWTSecret))
	api.Use(middleware.TenantContext(cfg.JWTSecret))

	mikrotikGuard := middleware.RequireModule(domain.ModuleMikroTik, cfg.ModuleChecker, cfg.Logger)
	fiberNetworkGuard := middleware.RequireModule(domain.ModuleFiberNetwork, cfg.ModuleChecker, cfg.Logger)

	// Route CRUD router MikroTik
	mikrotik := api.Group("/mikrotik", mikrotikGuard)
	routers := mikrotik.Group("/routers")
	routers.Post("/", cfg.RouterHandler.Create)
	routers.Get("/", cfg.RouterHandler.List)
	routers.Get("/:id", cfg.RouterHandler.GetByID)
	routers.Put("/:id", cfg.RouterHandler.Update)
	routers.Delete("/:id", cfg.RouterHandler.Delete)
	routers.Post("/:id/test", cfg.RouterHandler.TestConnection)
	routers.Post("/:id/reboot", cfg.RouterHandler.Reboot)

	// Route manajemen PPPoE user dan sessions
	pppoe := routers.Group("/:id/pppoe")
	pppoe.Get("/users", cfg.PPPoEHandler.ListUsers)
	pppoe.Post("/users", cfg.PPPoEHandler.CreateUser)
	pppoe.Put("/users/:user_id", cfg.PPPoEHandler.UpdateUser)
	pppoe.Delete("/users/:user_id", cfg.PPPoEHandler.DeleteUser)
	pppoe.Post("/users/:user_id/disconnect", cfg.PPPoEHandler.DisconnectUser)
	pppoe.Get("/sync-status", cfg.PPPoEHandler.GetSyncStatus)
	pppoe.Post("/sync", cfg.PPPoEHandler.TriggerSync)

	// Route PPPoE active sessions
	pppoe.Get("/sessions", cfg.SessionHandler.GetSessions)
	pppoe.Post("/sessions/:session_id/disconnect", cfg.SessionHandler.DisconnectSession)
	pppoe.Get("/sessions/count", cfg.SessionHandler.GetSessionCount)

	// Route operasional RouterOS. Semua dibaca manual/on-demand dari UI/API.
	routers.Get("/:id/interfaces", cfg.OperationalHandler.ListInterfaces)
	routers.Get("/:id/traffic", cfg.OperationalHandler.GetTraffic)
	routers.Get("/:id/ip-pools", cfg.OperationalHandler.ListIPPools)
	routers.Get("/:id/firewall/managed", cfg.OperationalHandler.ListManagedFirewall)
	routers.Get("/:id/logs", cfg.OperationalHandler.ListLogs)

	// Route DHCP server, leases, static bindings, dan networks
	dhcp := routers.Group("/:id/dhcp")
	dhcp.Get("/servers", cfg.DHCPHandler.ListServers)
	dhcp.Get("/leases", cfg.DHCPHandler.ListLeases)
	dhcp.Get("/bindings", cfg.DHCPHandler.ListBindings)
	dhcp.Post("/bindings", cfg.DHCPHandler.CreateBinding)
	dhcp.Put("/bindings/:binding_id", cfg.DHCPHandler.UpdateBinding)
	dhcp.Delete("/bindings/:binding_id", cfg.DHCPHandler.DeleteBinding)
	dhcp.Get("/networks", cfg.DHCPHandler.ListNetworks)

	// Route static IP customer management
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

	// Route terminal read-only dan audit command MikroTik.
	terminal := routers.Group("/:id/terminal")
	terminal.Post("/execute", cfg.TerminalHandler.Execute)
	terminal.Get("/audit", cfg.TerminalHandler.ListAudit)

	// Route backup export dan firmware read-only MikroTik.
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

	// Route status summary router MikroTik
	status := mikrotik.Group("/status")
	status.Get("/summary", cfg.StatusHandler.GetSummary)

	// Route manajemen VPN tunnel
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

	// Route admin VPN (maintenance scheduling)
	admin := api.Group("/admin/vpn", mikrotikGuard)
	admin.Post("/maintenance", cfg.VPNHandler.ScheduleMaintenance)

	fiberNetwork := api.Group("", fiberNetworkGuard)

	// Route CRUD dan monitoring OLT device
	olt := fiberNetwork.Group("/olt")
	oltDevices := olt.Group("/devices")
	oltDevices.Post("/", cfg.OLTHandler.CreateOLT)
	oltDevices.Get("/", cfg.OLTHandler.ListOLTs)
	oltDevices.Get("/:id", cfg.OLTHandler.GetOLT)
	oltDevices.Put("/:id", cfg.OLTHandler.UpdateOLT)
	oltDevices.Delete("/:id", cfg.OLTHandler.DeleteOLT)
	oltDevices.Post("/:id/test-snmp", cfg.OLTHandler.TestSNMP)
	oltDevices.Post("/:id/test-cli", cfg.OLTHandler.TestCLI)
	oltDevices.Get("/:id/pon-ports", cfg.OLTHandler.GetPONPorts)
	oltDevices.Get("/:id/pon-ports/:port/onts", cfg.OLTHandler.GetONTList)
	oltDevices.Get("/:id/pon-ports/:port/traffic", cfg.OLTHandler.GetTraffic)
	oltDevices.Get("/:id/alarms", cfg.OLTHandler.GetAlarms)
	oltDevices.Get("/:id/sfp", cfg.OLTHandler.GetSFP)
	oltDevices.Get("/:id/capacity", cfg.OLTHandler.GetCapacity)

	// Route CRUD ODP/splitter
	odp := olt.Group("/odp")
	odp.Post("/", cfg.ODPHandler.CreateODP)
	odp.Get("/", cfg.ODPHandler.ListODPs)
	odp.Get("/:id", cfg.ODPHandler.GetODP)
	odp.Put("/:id", cfg.ODPHandler.UpdateODP)
	odp.Delete("/:id", cfg.ODPHandler.DeleteODP)

	// Route provisioning ONT (single, bulk, decommission, reboot, audit, settings)
	prov := olt.Group("/provisioning")
	prov.Post("/ont", cfg.ProvisioningHandler.ProvisionONT)
	prov.Get("/onts", cfg.ProvisioningHandler.ListONTs)
	prov.Get("/onts/:id", cfg.ProvisioningHandler.GetONT)
	prov.Post("/ont/:id/decommission", cfg.ProvisioningHandler.DecommissionONT)
	prov.Post("/ont/:id/reboot", cfg.ProvisioningHandler.RebootONT)
	prov.Post("/ont/:id/confirm-migration", cfg.ProvisioningHandler.ConfirmMigration)
	prov.Post("/bulk", cfg.ProvisioningHandler.BulkUpload)
	prov.Post("/bulk/execute", cfg.ProvisioningHandler.BulkExecute)
	prov.Get("/bulk/template", cfg.ProvisioningHandler.BulkTemplate)
	prov.Get("/audit-logs", cfg.ProvisioningHandler.GetAuditLogs)
	prov.Get("/settings", cfg.ProvisioningHandler.GetSettings)
	prov.Put("/settings", cfg.ProvisioningHandler.UpdateSettings)

	// Route VLAN per OLT device
	oltDevices.Post("/:id/vlans", cfg.VLANHandler.CreateVLAN)
	oltDevices.Get("/:id/vlans", cfg.VLANHandler.ListVLANs)

	// Route service profile per OLT device
	oltDevices.Post("/:id/service-profiles", cfg.ServiceProfileHandler.CreateServiceProfile)
	oltDevices.Get("/:id/service-profiles", cfg.ServiceProfileHandler.ListServiceProfiles)

	// Route unregistered ONTs per OLT device
	oltDevices.Get("/:id/unregistered-onts", cfg.ProvisioningHandler.GetUnregisteredONTs)

	// Route update/delete VLAN by ID
	vlans := olt.Group("/vlans")
	vlans.Put("/:id", cfg.VLANHandler.UpdateVLAN)
	vlans.Delete("/:id", cfg.VLANHandler.DeleteVLAN)

	// Route update/delete service profile by ID
	serviceProfiles := olt.Group("/service-profiles")
	serviceProfiles.Put("/:id", cfg.ServiceProfileHandler.UpdateServiceProfile)
	serviceProfiles.Delete("/:id", cfg.ServiceProfileHandler.DeleteServiceProfile)

	// Route ringkasan status OLT
	olt.Get("/summary", cfg.OLTHandler.GetSummary)

	// --- Route FTTH Visual Mapping ---

	// Route publik — akses peta read-only via share token (tanpa auth)
	cfg.App.Get("/api/v1/network-map/share/:token", cfg.ShareHandler.GetSharedMap)

	// Grup route network-map yang dilindungi auth + tenant middleware
	networkMap := fiberNetwork.Group("/network-map")

	// Route CRUD map node (OLT, ODP, ONT)
	networkMap.Get("/nodes", cfg.MapNodeHandler.ListNodes)
	networkMap.Post("/nodes", cfg.MapNodeHandler.CreateNode)
	networkMap.Get("/nodes/:id", cfg.MapNodeHandler.GetNode)
	networkMap.Put("/nodes/:id", cfg.MapNodeHandler.UpdateNode)
	networkMap.Delete("/nodes/:id", cfg.MapNodeHandler.DeleteNode)
	networkMap.Get("/nodes/:id/photos", cfg.MapNodeHandler.ListPhotos)
	networkMap.Post("/nodes/:id/photos", cfg.MapNodeHandler.UploadPhoto)
	networkMap.Delete("/nodes/:id/photos/:photo_id", cfg.MapNodeHandler.DeletePhoto)
	networkMap.Get("/nodes/:id/history", cfg.MapNodeHandler.GetHistory)

	// Route CRUD cable route (backbone, drop)
	networkMap.Get("/cables", cfg.CableRouteHandler.ListRoutes)
	networkMap.Post("/cables", cfg.CableRouteHandler.CreateRoute)
	networkMap.Get("/cables/:id", cfg.CableRouteHandler.GetRoute)
	networkMap.Put("/cables/:id", cfg.CableRouteHandler.UpdateRoute)
	networkMap.Delete("/cables/:id", cfg.CableRouteHandler.DeleteRoute)

	// Route pencarian map node
	networkMap.Get("/search", cfg.SearchHandler.Search)

	// Route export peta (KML, KMZ, GeoJSON, CSV)
	networkMap.Post("/export", cfg.ExportHandler.Export)
	networkMap.Get("/export/status/:job_id", cfg.ExportHandler.GetExportStatus)

	// Route import peta (KML, KMZ, GeoJSON)
	networkMap.Post("/import", cfg.ImportHandler.Preview)
	networkMap.Post("/import/execute", cfg.ImportHandler.Execute)
	networkMap.Get("/import/status/:job_id", cfg.ImportHandler.GetImportStatus)

	// Route reverse geocoding
	networkMap.Get("/geocode/reverse", cfg.GeocodingHandler.ReverseGeocode)

	// Route share link (CRUD — authenticated)
	networkMap.Post("/share", cfg.ShareHandler.CreateShareLink)
	networkMap.Get("/share", cfg.ShareHandler.ListShareLinks)
	networkMap.Delete("/share/:token", cfg.ShareHandler.DeleteShareLink)

	// Route loss calculator
	networkMap.Post("/loss-calculator", cfg.LossCalcHandler.CalculateLoss)

	// Route konfigurasi label peta
	networkMap.Get("/settings/labels", cfg.LabelSettingsHandler.GetLabelSettings)
	networkMap.Put("/settings/labels", cfg.LabelSettingsHandler.UpdateLabelSettings)

	// Route trash management (soft-delete)
	networkMap.Get("/trash", cfg.TrashHandler.ListTrashed)
	networkMap.Post("/trash/:id/restore", cfg.TrashHandler.RestoreNode)
}
