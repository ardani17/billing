package adapter

import "github.com/ispboss/ispboss/services/network-service/internal/domain"

func NewVPNCommandBuilder() domain.VPNCommandBuilder {
	return NewVPNCommandBuilderForVersion("")
}

func NewVPNCommandBuilderForVersion(routerOSVersion string) domain.VPNCommandBuilder {
	if ParseRouterOSMajor(routerOSVersion) == RouterOSv7 {
		return &vpnCommandBuilderV7{}
	}
	return &vpnCommandBuilderV6{}
}
