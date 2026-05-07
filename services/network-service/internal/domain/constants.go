package domain

import (
	"strconv"
	"strings"
	"unicode"
)

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

	// PriorityMedium untuk perintah operasional: CRUD user, perbarui profile.
	PriorityMedium CommandPriority = 2

	// PriorityLow untuk perintah pemantauan: sync, pemantauan, backup.
	PriorityLow CommandPriority = 1
)

// --- RouterOS Version Fungsi bantu ---

type RouterOSMajor int

const (
	RouterOSUnknown RouterOSMajor = 0
	RouterOSv6      RouterOSMajor = 6
	RouterOSv7      RouterOSMajor = 7
)

func NormalizeRouterOSVersion(version string) string {
	return strings.TrimSpace(version)
}

func ParseRouterOSMajor(version string) RouterOSMajor {
	version = NormalizeRouterOSVersion(version)
	if version == "" {
		return RouterOSUnknown
	}
	for i, r := range version {
		if !unicode.IsDigit(r) {
			continue
		}
		start := i
		end := i + len(string(r))
		for end < len(version) {
			next, width := rune(version[end]), 1
			if next >= 0x80 {
				next = []rune(version[end:])[0]
				width = len(string(next))
			}
			if !unicode.IsDigit(next) {
				break
			}
			end += width
		}
		major, err := strconv.Atoi(version[start:end])
		if err != nil {
			return RouterOSUnknown
		}
		switch major {
		case int(RouterOSv6):
			return RouterOSv6
		case int(RouterOSv7):
			return RouterOSv7
		default:
			return RouterOSUnknown
		}
	}
	return RouterOSUnknown
}

type RouterOSCapabilities struct {
	Major             RouterOSMajor
	SupportsWireGuard bool
}

func CapabilitiesForRouterOS(version string) RouterOSCapabilities {
	major := ParseRouterOSMajor(version)
	return RouterOSCapabilities{
		Major:             major,
		SupportsWireGuard: major == RouterOSv7,
	}
}

// IsRouterOSv7 memeriksa apakah versi RouterOS adalah v7.
// Digunakan untuk menentukan API path yang berbeda antara v6 dan v7.
func IsRouterOSv7(version string) bool {
	return ParseRouterOSMajor(version) == RouterOSv7
}
