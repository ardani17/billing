package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/middleware"
	fiberhandler "github.com/ispboss/ispboss/services/network-service/internal/modules/fiber/handler"
	mikrotikhandler "github.com/ispboss/ispboss/services/network-service/internal/modules/mikrotik/handler"
	"github.com/rs/zerolog"
)

// RouterConfig berisi dependensi yang dibutuhkan untuk registrasi route.
type RouterConfig struct {
	// App adalah instance Fiber application
	App *fiber.App

	// HealthHandler adalah handler untuk health cek endpoint
	HealthHandler *HealthHandler

	// RouterHandler adalah handler untuk operasi CRUD router MikroTik
	RouterHandler *mikrotikhandler.RouterHandler

	// StatusHandler adalah handler untuk ringkasan status router
	StatusHandler *mikrotikhandler.StatusHandler

	// PPPoEHandler adalah handler untuk manajemen PPPoE user
	PPPoEHandler *mikrotikhandler.PPPoEHandler

	// SessionHandler adalah handler untuk manajemen PPPoE active sessions
	SessionHandler *mikrotikhandler.SessionHandler

	// VPNHandler adalah handler untuk manajemen VPN tunnel
	VPNHandler *mikrotikhandler.VPNHandler

	// OperationalHandler adalah handler untuk data operasional RouterOS on-demand
	OperationalHandler *mikrotikhandler.MikroTikOperationalHandler

	// DHCPHandler adalah handler untuk DHCP server/lease/static binding
	DHCPHandler *mikrotikhandler.DHCPHandler

	// StaticIPHandler adalah handler untuk pelanggan static IP
	StaticIPHandler *mikrotikhandler.StaticIPHandler

	// WalledGardenHandler adalah handler untuk rule isolir/walled garden
	WalledGardenHandler *mikrotikhandler.WalledGardenHandler

	// HotspotHandler adalah handler untuk user/profile/session Hotspot
	HotspotHandler *mikrotikhandler.HotspotHandler

	// TerminalHandler adalah handler untuk terminal hanya baca dan audit command
	TerminalHandler *mikrotikhandler.TerminalHandler

	// BackupHandler adalah handler untuk backup export dan firmware MikroTik
	BackupHandler *mikrotikhandler.BackupHandler

	// BulkHandler adalah handler untuk bulk action MikroTik on-demand
	BulkHandler *mikrotikhandler.MikroTikBulkHandler

	// OLTHandler adalah handler untuk manajemen OLT device
	OLTHandler *fiberhandler.OLTHandler

	// ODPHandler adalah handler untuk manajemen ODP/splitter
	ODPHandler *fiberhandler.ODPHandler

	// ProvisioningHandler adalah handler untuk provisioning ONT (single, bulk, decommission, reboot)
	ProvisioningHandler *fiberhandler.ProvisioningHandler

	// VLANHandler adalah handler untuk manajemen VLAN per OLT
	VLANHandler *fiberhandler.VLANHandler

	// ServiceProfileHandler adalah handler untuk manajemen service profile per OLT
	ServiceProfileHandler *fiberhandler.ServiceProfileHandler

	// MapNodeHandler adalah handler untuk manajemen map node (CRUD, foto, riwayat)
	MapNodeHandler *MapNodeHandler

	// CableRouteHandler adalah handler untuk manajemen cable route (CRUD)
	CableRouteHandler *CableRouteHandler

	// PencarianHandler adalah handler untuk pencarian map node
	SearchHandler *SearchHandler

	// ExportHandler adalah handler untuk export peta (KML, KMZ, GeoJSON, CSV)
	ExportHandler *ExportHandler

	// ImportHandler adalah handler untuk import peta (KML, KMZ, GeoJSON)
	ImportHandler *ImportHandler

	// GeocodingHandler adalah handler untuk reverse geocoding
	GeocodingHandler *GeocodingHandler

	// ShareHandler adalah handler untuk share link peta hanya baca
	ShareHandler *ShareHandler

	// LossCalcHandler adalah handler untuk kalkulasi optical loss budget
	LossCalcHandler *LossCalcHandler

	// LabelSettingsHandler adalah handler untuk konfigurasi label peta
	LabelSettingsHandler *LabelSettingsHandler

	// TrashHandler adalah handler untuk manajemen trash (hapus lunak)
	TrashHandler *TrashHandler

	// ModuleChecker memeriksa entitlement modul add-on per tenant
	ModuleChecker middleware.ModuleChecker

	// JWTSecret adalah secret key untuk validasi JWT token
	JWTSecret string

	// Logger adalah instance zerolog untuk permintaan logging
	Logger zerolog.Logger
}

// RegisterRoutes mendaftarkan semua route pada Fiber app.
// Health cek endpoint bersifat publik (tanpa auth).
// Route lainnya dilindungi oleh auth dan tenant middleware.
func RegisterRoutes(cfg RouterConfig) {
	// Middleware logging untuk semua permintaan
	cfg.App.Use(middleware.RequestLogger(cfg.Logger))

	// Route publik - health cek (tanpa autentikasi)
	cfg.App.Get("/healthz", cfg.HealthHandler.Healthz)
	cfg.App.Get("/readyz", cfg.HealthHandler.Readyz)

	// Grup route yang dilindungi oleh auth dan tenant middleware
	api := cfg.App.Group("/api/v1")
	api.Use(middleware.Auth(cfg.JWTSecret))
	api.Use(middleware.TenantContext(cfg.JWTSecret))

	mikrotikhandler.RegisterRoutes(api, mikrotikhandler.RouterConfig{
		RouterHandler:       cfg.RouterHandler,
		StatusHandler:       cfg.StatusHandler,
		PPPoEHandler:        cfg.PPPoEHandler,
		SessionHandler:      cfg.SessionHandler,
		VPNHandler:          cfg.VPNHandler,
		OperationalHandler:  cfg.OperationalHandler,
		DHCPHandler:         cfg.DHCPHandler,
		StaticIPHandler:     cfg.StaticIPHandler,
		WalledGardenHandler: cfg.WalledGardenHandler,
		HotspotHandler:      cfg.HotspotHandler,
		TerminalHandler:     cfg.TerminalHandler,
		BackupHandler:       cfg.BackupHandler,
		BulkHandler:         cfg.BulkHandler,
		ModuleChecker:       cfg.ModuleChecker,
		Logger:              cfg.Logger,
	})

	fiberhandler.RegisterRoutes(api, fiberhandler.RouterConfig{
		OLTHandler:            cfg.OLTHandler,
		ODPHandler:            cfg.ODPHandler,
		ProvisioningHandler:   cfg.ProvisioningHandler,
		VLANHandler:           cfg.VLANHandler,
		ServiceProfileHandler: cfg.ServiceProfileHandler,
		ModuleChecker:         cfg.ModuleChecker,
		Logger:                cfg.Logger,
	})

	fiberNetworkGuard := middleware.RequireModule(domain.ModuleFiberNetwork, cfg.ModuleChecker, cfg.Logger)
	fiberNetwork := api.Group("", fiberNetworkGuard)

	// --- Route FTTH Visual Mapping ---

	// Route publik - akses peta hanya baca via share token (tanpa auth)
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

	// Route share link (CRUD - authenticated)
	networkMap.Post("/share", cfg.ShareHandler.CreateShareLink)
	networkMap.Get("/share", cfg.ShareHandler.ListShareLinks)
	networkMap.Delete("/share/:token", cfg.ShareHandler.DeleteShareLink)

	// Route loss calculator
	networkMap.Post("/loss-calculator", cfg.LossCalcHandler.CalculateLoss)

	// Route konfigurasi label peta
	networkMap.Get("/settings/labels", cfg.LabelSettingsHandler.GetLabelSettings)
	networkMap.Put("/settings/labels", cfg.LabelSettingsHandler.UpdateLabelSettings)

	// Route trash management (hapus lunak)
	networkMap.Get("/trash", cfg.TrashHandler.ListTrashed)
	networkMap.Post("/trash/:id/restore", cfg.TrashHandler.RestoreNode)
}
