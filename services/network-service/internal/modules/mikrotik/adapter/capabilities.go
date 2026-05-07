package adapter

import "github.com/ispboss/ispboss/services/network-service/internal/domain"

type RouterOSCapabilities = domain.RouterOSCapabilities

func CapabilitiesFor(version string) RouterOSCapabilities {
	return domain.CapabilitiesForRouterOS(version)
}
