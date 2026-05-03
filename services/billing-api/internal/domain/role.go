package domain

// RedirectPathMap memetakan setiap role ke path redirect setelah login berhasil.
// Frontend menggunakan nilai ini untuk mengarahkan user ke halaman yang sesuai.
var RedirectPathMap = map[UserRole]string{
	RoleSuperAdmin:  "/super-admin",
	RoleTenantAdmin: "/dashboard",
	RoleOperator:    "/dashboard",
	RoleTeknisi:     "/network",
	RoleKasir:       "/payments",
	RoleReseller:    "/reseller/dashboard",
}

// ValidRoles berisi semua role yang valid di sistem.
var ValidRoles = []UserRole{
	RoleSuperAdmin,
	RoleTenantAdmin,
	RoleOperator,
	RoleTeknisi,
	RoleKasir,
	RoleReseller,
}

// IsValidRole memeriksa apakah role yang diberikan valid.
func IsValidRole(role UserRole) bool {
	for _, r := range ValidRoles {
		if r == role {
			return true
		}
	}
	return false
}

// RBACConfig mendefinisikan konfigurasi akses per endpoint.
type RBACConfig struct {
	// AllowedRoles adalah daftar role yang boleh mengakses endpoint.
	AllowedRoles []UserRole

	// MethodRestrictions membatasi HTTP method per role.
	// Key: role, Value: daftar method yang diizinkan.
	// Jika role tidak ada di map, semua method diizinkan.
	MethodRestrictions map[UserRole][]string
}
