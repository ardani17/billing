package handler

import (
	"github.com/gofiber/fiber/v2"
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

	// Route CRUD router MikroTik
	routers := api.Group("/mikrotik/routers")
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

	// Route status summary router MikroTik
	status := api.Group("/mikrotik/status")
	status.Get("/summary", cfg.StatusHandler.GetSummary)

	// Route manajemen VPN tunnel
	vpn := api.Group("/mikrotik/vpn")
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
	admin := api.Group("/admin/vpn")
	admin.Post("/maintenance", cfg.VPNHandler.ScheduleMaintenance)

	// Route CRUD dan monitoring OLT device
	oltDevices := api.Group("/olt/devices")
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
	odp := api.Group("/olt/odp")
	odp.Post("/", cfg.ODPHandler.CreateODP)
	odp.Get("/", cfg.ODPHandler.ListODPs)
	odp.Get("/:id", cfg.ODPHandler.GetODP)
	odp.Put("/:id", cfg.ODPHandler.UpdateODP)
	odp.Delete("/:id", cfg.ODPHandler.DeleteODP)

	// Route provisioning ONT (single, bulk, decommission, reboot, audit, settings)
	prov := api.Group("/olt/provisioning")
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
	vlans := api.Group("/olt/vlans")
	vlans.Put("/:id", cfg.VLANHandler.UpdateVLAN)
	vlans.Delete("/:id", cfg.VLANHandler.DeleteVLAN)

	// Route update/delete service profile by ID
	serviceProfiles := api.Group("/olt/service-profiles")
	serviceProfiles.Put("/:id", cfg.ServiceProfileHandler.UpdateServiceProfile)
	serviceProfiles.Delete("/:id", cfg.ServiceProfileHandler.DeleteServiceProfile)

	// Route ringkasan status OLT
	api.Get("/olt/summary", cfg.OLTHandler.GetSummary)

	// --- Route FTTH Visual Mapping ---

	// Route publik — akses peta read-only via share token (tanpa auth)
	cfg.App.Get("/api/v1/network-map/share/:token", cfg.ShareHandler.GetSharedMap)

	// Grup route network-map yang dilindungi auth + tenant middleware
	networkMap := api.Group("/network-map")

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
