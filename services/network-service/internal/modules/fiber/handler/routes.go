package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"

	"github.com/ispboss/ispboss/services/network-service/internal/domain"
	"github.com/ispboss/ispboss/services/network-service/internal/middleware"
)

// RouterConfig berisi handler dan dependensi route khusus modul fiber network.
type RouterConfig struct {
	OLTHandler            *OLTHandler
	ODPHandler            *ODPHandler
	ProvisioningHandler   *ProvisioningHandler
	VLANHandler           *VLANHandler
	ServiceProfileHandler *ServiceProfileHandler
	ModuleChecker         middleware.ModuleChecker
	Logger                zerolog.Logger
}

// RegisterRoutes mendaftarkan route OLT/ODP/provisioning tanpa mengubah path publik lama.
func RegisterRoutes(api fiber.Router, cfg RouterConfig) {
	fiberNetworkGuard := middleware.RequireModule(domain.ModuleFiberNetwork, cfg.ModuleChecker, cfg.Logger)
	fiberNetwork := api.Group("", fiberNetworkGuard)

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
	oltDevices.Get("/:id/pon-ports/:port/onts/:ont/signal", cfg.OLTHandler.GetSignal)
	oltDevices.Get("/:id/alarms", cfg.OLTHandler.GetAlarms)
	oltDevices.Get("/:id/sfp", cfg.OLTHandler.GetSFP)
	oltDevices.Get("/:id/capacity", cfg.OLTHandler.GetCapacity)

	odp := olt.Group("/odp")
	odp.Post("/", cfg.ODPHandler.CreateODP)
	odp.Get("/", cfg.ODPHandler.ListODPs)
	odp.Get("/:id", cfg.ODPHandler.GetODP)
	odp.Put("/:id", cfg.ODPHandler.UpdateODP)
	odp.Delete("/:id", cfg.ODPHandler.DeleteODP)

	prov := olt.Group("/provisioning")
	prov.Post("/ont", cfg.ProvisioningHandler.ProvisionONT)
	prov.Post("/ont/preview", cfg.ProvisioningHandler.PreviewProvisionONT)
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

	oltDevices.Post("/:id/vlans", cfg.VLANHandler.CreateVLAN)
	oltDevices.Get("/:id/vlans", cfg.VLANHandler.ListVLANs)

	oltDevices.Post("/:id/service-profiles", cfg.ServiceProfileHandler.CreateServiceProfile)
	oltDevices.Get("/:id/service-profiles", cfg.ServiceProfileHandler.ListServiceProfiles)

	oltDevices.Get("/:id/unregistered-onts", cfg.ProvisioningHandler.GetUnregisteredONTs)

	vlans := olt.Group("/vlans")
	vlans.Put("/:id", cfg.VLANHandler.UpdateVLAN)
	vlans.Delete("/:id", cfg.VLANHandler.DeleteVLAN)

	serviceProfiles := olt.Group("/service-profiles")
	serviceProfiles.Put("/:id", cfg.ServiceProfileHandler.UpdateServiceProfile)
	serviceProfiles.Delete("/:id", cfg.ServiceProfileHandler.DeleteServiceProfile)

	olt.Get("/summary", cfg.OLTHandler.GetSummary)
}
