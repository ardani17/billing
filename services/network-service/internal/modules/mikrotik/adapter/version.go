package adapter

import "github.com/ispboss/ispboss/services/network-service/internal/domain"

type RouterOSMajor = domain.RouterOSMajor

const (
	RouterOSUnknown = domain.RouterOSUnknown
	RouterOSv6      = domain.RouterOSv6
	RouterOSv7      = domain.RouterOSv7
)

func NormalizeRouterOSVersion(version string) string {
	return domain.NormalizeRouterOSVersion(version)
}

func ParseRouterOSMajor(version string) RouterOSMajor {
	return domain.ParseRouterOSMajor(version)
}
