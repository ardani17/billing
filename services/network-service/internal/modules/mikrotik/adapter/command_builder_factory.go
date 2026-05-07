package adapter

import "github.com/ispboss/ispboss/services/network-service/internal/domain"

// NewCommandBuilder membuat CommandBuilder sesuai versi RouterOS.
// Menggunakan domain.IsRouterOSv7() untuk menentukan versi.
// Jika versi dimulai dengan "7", menggunakan commandBuilderV7.
// Untuk versi lainnya (termasuk v6), menggunakan commandBuilderV6.
func NewCommandBuilder(routerOSVersion string) domain.CommandBuilder {
	if ParseRouterOSMajor(routerOSVersion) == RouterOSv7 {
		return &commandBuilderV7{}
	}
	return &commandBuilderV6{}
}
