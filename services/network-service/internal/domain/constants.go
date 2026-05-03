package domain

import "strings"

// --- Router Status ---

// RouterStatus mendefinisikan status konektivitas router.
type RouterStatus string

const (
	// StatusOnline menandakan router aktif dan dapat dijangkau.
	StatusOnline RouterStatus = "online"

	// StatusOffline menandakan router tidak dapat dijangkau.
	StatusOffline RouterStatus = "offline"

	// StatusMaintenance menandakan router sedang dalam pemeliharaan.
	StatusMaintenance RouterStatus = "maintenance"
)

// ValidRouterTransitions mendefinisikan transisi status yang valid.
// Key: status asal, Value: daftar status tujuan yang diizinkan.
var ValidRouterTransitions = map[RouterStatus][]RouterStatus{
	StatusOffline:     {StatusOnline, StatusMaintenance},
	StatusOnline:      {StatusOffline, StatusMaintenance},
	StatusMaintenance: {StatusOnline, StatusOffline},
}

// CanTransitionRouter memeriksa apakah transisi status valid.
func CanTransitionRouter(current, target RouterStatus) bool {
	targets, ok := ValidRouterTransitions[current]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == target {
			return true
		}
	}
	return false
}

// --- Service Type ---

// ServiceType mendefinisikan tipe layanan yang didukung router.
type ServiceType string

const (
	// ServicePPPoE untuk layanan PPPoE.
	ServicePPPoE ServiceType = "pppoe"

	// ServiceHotspot untuk layanan hotspot.
	ServiceHotspot ServiceType = "hotspot"

	// ServiceDHCP untuk layanan DHCP binding.
	ServiceDHCP ServiceType = "dhcp_binding"

	// ServiceStatic untuk layanan IP static.
	ServiceStatic ServiceType = "static"
)

// --- Command Priority ---

// CommandPriority mendefinisikan prioritas perintah ke router.
type CommandPriority int

const (
	// PriorityHigh untuk perintah kritis: isolir, buka isolir, disconnect.
	PriorityHigh CommandPriority = 3

	// PriorityMedium untuk perintah operasional: CRUD user, update profile.
	PriorityMedium CommandPriority = 2

	// PriorityLow untuk perintah monitoring: sync, monitoring, backup.
	PriorityLow CommandPriority = 1
)

// --- RouterOS Version Helper ---

// IsRouterOSv7 memeriksa apakah versi RouterOS adalah v7.
// Digunakan untuk menentukan API path yang berbeda antara v6 dan v7.
func IsRouterOSv7(version string) bool {
	return strings.HasPrefix(version, "7")
}
