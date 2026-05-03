package domain

import "errors"

// --- Domain Errors ---
// Semua error domain untuk network-service.
// Digunakan oleh usecase, repository, adapter, dan handler layer.

var (
	// ErrRouterNotFound dikembalikan saat router tidak ditemukan atau milik tenant lain.
	ErrRouterNotFound = errors.New("router tidak ditemukan")

	// ErrRouterNameExists dikembalikan saat nama router sudah ada di tenant yang sama.
	ErrRouterNameExists = errors.New("nama router sudah ada")

	// ErrInvalidStatusTransition dikembalikan saat transisi status tidak valid.
	ErrInvalidStatusTransition = errors.New("transisi status tidak valid")

	// ErrConfirmationMismatch dikembalikan saat nama konfirmasi reboot tidak cocok.
	ErrConfirmationMismatch = errors.New("nama konfirmasi tidak cocok")

	// ErrRouterOffline dikembalikan saat operasi membutuhkan router online.
	ErrRouterOffline = errors.New("router sedang offline")

	// ErrConnectionFailed dikembalikan saat koneksi ke router gagal.
	ErrConnectionFailed = errors.New("gagal terhubung ke router")

	// ErrConnectionTimeout dikembalikan saat koneksi ke router timeout.
	ErrConnectionTimeout = errors.New("koneksi ke router timeout")

	// ErrPoolExhausted dikembalikan saat pool koneksi penuh dan timeout menunggu.
	ErrPoolExhausted = errors.New("pool koneksi penuh")

	// ErrRateLimited dikembalikan saat rate limit per router terlampaui.
	ErrRateLimited = errors.New("rate limit terlampaui")

	// ErrEncryptionFailed dikembalikan saat enkripsi password gagal.
	ErrEncryptionFailed = errors.New("gagal mengenkripsi password")

	// ErrDecryptionFailed dikembalikan saat dekripsi password gagal.
	ErrDecryptionFailed = errors.New("gagal mendekripsi password")

	// ErrInvalidEncryptionKey dikembalikan saat ENCRYPTION_KEY tidak valid.
	ErrInvalidEncryptionKey = errors.New("ENCRYPTION_KEY harus 32 bytes")

	// ErrRouterDeleted dikembalikan saat router sudah di-soft-delete.
	ErrRouterDeleted = errors.New("router sudah dihapus")

	// --- PPPoE Domain Errors ---

	// ErrPPPoEUserNotFound dikembalikan saat PPPoE user tidak ditemukan.
	ErrPPPoEUserNotFound = errors.New("pppoe user tidak ditemukan")

	// ErrPPPoEUsernameExists dikembalikan saat username sudah ada di router.
	ErrPPPoEUsernameExists = errors.New("username pppoe sudah ada di router ini")

	// ErrPPPoEProfileNotFound dikembalikan saat profile tidak ditemukan.
	ErrPPPoEProfileNotFound = errors.New("pppoe profile tidak ditemukan")

	// ErrProfileNameExists dikembalikan saat profile name sudah ada di tenant.
	ErrProfileNameExists = errors.New("nama profile sudah ada")

	// ErrInvalidConnectionMethod dikembalikan saat connection_method bukan "pppoe".
	ErrInvalidConnectionMethod = errors.New("connection method bukan pppoe")

	// ErrInvalidIsolirMethod dikembalikan saat isolir method tidak valid.
	ErrInvalidIsolirMethod = errors.New("isolir method tidak valid, gunakan firewall_nat_redirect atau dns_redirect")

	// ErrInvalidCommentFormat dikembalikan saat comment field tidak sesuai format.
	ErrInvalidCommentFormat = errors.New("format comment tidak valid, harus ISPBoss:{customer_id}:{tenant_id}")

	// ErrSyncInProgress dikembalikan saat sync sudah berjalan untuk router ini.
	ErrSyncInProgress = errors.New("sinkronisasi sedang berjalan untuk router ini")

	// ErrMaxRetriesExhausted dikembalikan saat semua retry gagal.
	ErrMaxRetriesExhausted = errors.New("semua retry gagal, operasi ditandai sebagai failed_permanent")

	// ErrSessionNotFound dikembalikan saat session PPPoE tidak ditemukan.
	ErrSessionNotFound = errors.New("session pppoe tidak ditemukan")

	// --- VPN Domain Errors ---

	// ErrVPNTunnelNotFound dikembalikan saat VPN tunnel tidak ditemukan.
	ErrVPNTunnelNotFound = errors.New("vpn tunnel tidak ditemukan")

	// ErrVPNTunnelNameExists dikembalikan saat tunnel_name sudah ada di tenant.
	ErrVPNTunnelNameExists = errors.New("nama vpn tunnel sudah ada")

	// ErrVPNIPExists dikembalikan saat vpn_ip sudah digunakan di tenant.
	ErrVPNIPExists = errors.New("vpn ip sudah digunakan")

	// ErrVPNSubnetExhausted dikembalikan saat subnet /24 sudah penuh (253 client).
	ErrVPNSubnetExhausted = errors.New("subnet vpn sudah penuh, maksimal 253 tunnel per tenant")

	// ErrInvalidVPNProtocol dikembalikan saat protokol VPN tidak valid.
	ErrInvalidVPNProtocol = errors.New("protokol vpn tidak valid, gunakan wireguard, l2tp_ipsec, pptp, atau sstp")

	// ErrWireGuardRequiresV7 dikembalikan saat WireGuard dipilih untuk router v6.
	ErrWireGuardRequiresV7 = errors.New("wireguard membutuhkan RouterOS v7 atau lebih baru")

	// ErrInvalidTunnelTransition dikembalikan saat transisi status tunnel tidak valid.
	ErrInvalidTunnelTransition = errors.New("transisi status tunnel tidak valid")

	// ErrTunnelImmutableField dikembalikan saat mencoba update field yang tidak boleh diubah.
	ErrTunnelImmutableField = errors.New("vpn_ip, protocol, dan key pairs tidak dapat diubah setelah dibuat")

	// ErrVPNConnectionFailed dikembalikan saat test koneksi VPN gagal.
	ErrVPNConnectionFailed = errors.New("koneksi vpn gagal")

	// ErrVPNHandshakeTimeout dikembalikan saat handshake VPN timeout.
	ErrVPNHandshakeTimeout = errors.New("vpn handshake timeout")

	// ErrVPNAuthFailure dikembalikan saat autentikasi VPN gagal.
	ErrVPNAuthFailure = errors.New("autentikasi vpn gagal")

	// ErrRouterNotOnline dikembalikan saat auto-configure membutuhkan router online.
	ErrRouterNotOnline = errors.New("router harus online untuk auto-configure vpn")

	// ErrAutoConfigFailed dikembalikan saat auto-configure gagal di router.
	ErrAutoConfigFailed = errors.New("auto-configure vpn gagal, gunakan metode script manual")

	// ErrKeyGenerationFailed dikembalikan saat generate key pair gagal.
	ErrKeyGenerationFailed = errors.New("gagal generate key pair vpn")

	// ErrVPNIPUpdateFailed dikembalikan saat update router host ke VPN IP gagal.
	ErrVPNIPUpdateFailed = errors.New("gagal update router host ke vpn ip, koneksi via vpn tidak berhasil")

	// ErrTunnelDeleteWarning dikembalikan sebagai warning saat router menggunakan VPN IP.
	ErrTunnelDeleteWarning = errors.New("router menggunakan vpn ip sebagai host, menghapus tunnel akan membuat router tidak dapat dijangkau via vpn")

	// --- OLT Domain Errors ---

	// ErrOLTNotFound dikembalikan saat OLT tidak ditemukan atau milik tenant lain.
	ErrOLTNotFound = errors.New("olt tidak ditemukan")

	// ErrOLTNameExists dikembalikan saat nama OLT sudah ada di tenant yang sama.
	ErrOLTNameExists = errors.New("nama olt sudah ada")

	// ErrOLTInvalidStatusTransition dikembalikan saat transisi status OLT tidak valid.
	ErrOLTInvalidStatusTransition = errors.New("transisi status olt tidak valid")

	// ErrOLTOffline dikembalikan saat operasi membutuhkan OLT online.
	ErrOLTOffline = errors.New("olt sedang offline")

	// ErrOLTDeleted dikembalikan saat OLT sudah di-soft-delete.
	ErrOLTDeleted = errors.New("olt sudah dihapus")

	// ErrSNMPConnectionFailed dikembalikan saat koneksi SNMP ke OLT gagal.
	ErrSNMPConnectionFailed = errors.New("gagal koneksi snmp ke olt")

	// ErrSNMPTimeout dikembalikan saat operasi SNMP timeout.
	ErrSNMPTimeout = errors.New("snmp timeout")

	// ErrSNMPAuthFailed dikembalikan saat autentikasi SNMP gagal.
	ErrSNMPAuthFailed = errors.New("autentikasi snmp gagal")

	// ErrCLIConnectionFailed dikembalikan saat koneksi CLI ke OLT gagal.
	ErrCLIConnectionFailed = errors.New("gagal koneksi cli ke olt")

	// ErrCLITimeout dikembalikan saat CLI command timeout.
	ErrCLITimeout = errors.New("cli command timeout")

	// ErrCLIAuthFailed dikembalikan saat autentikasi CLI gagal.
	ErrCLIAuthFailed = errors.New("autentikasi cli gagal")

	// ErrUnsupportedBrand dikembalikan saat brand OLT tidak didukung.
	ErrUnsupportedBrand = errors.New("brand olt tidak didukung")

	// ErrBrandDetectionFailed dikembalikan saat gagal mendeteksi brand OLT dari sysDescr.
	ErrBrandDetectionFailed = errors.New("gagal mendeteksi brand olt")

	// --- ODP Domain Errors ---

	// ErrODPNotFound dikembalikan saat ODP tidak ditemukan atau milik tenant lain.
	ErrODPNotFound = errors.New("odp tidak ditemukan")

	// ErrODPNameExists dikembalikan saat nama ODP sudah ada di tenant yang sama.
	ErrODPNameExists = errors.New("nama odp sudah ada")

	// ErrODPFull dikembalikan saat ODP sudah mencapai kapasitas maksimal.
	ErrODPFull = errors.New("odp sudah penuh")

	// ErrInvalidSplitterType dikembalikan saat tipe splitter tidak valid.
	ErrInvalidSplitterType = errors.New("tipe splitter tidak valid")

	// --- Alarm Domain Errors ---

	// ErrAlarmNotFound dikembalikan saat alarm tidak ditemukan.
	ErrAlarmNotFound = errors.New("alarm tidak ditemukan")

	// ErrTrapReceiverFailed dikembalikan saat gagal menjalankan SNMP trap receiver.
	ErrTrapReceiverFailed = errors.New("gagal menjalankan trap receiver")

	// --- ONT Provisioning Errors ---

	// ErrONTNotFound dikembalikan saat ONT tidak ditemukan atau milik tenant lain.
	ErrONTNotFound = errors.New("ont tidak ditemukan")

	// ErrONTSerialNumberExists dikembalikan saat serial number sudah ada di tenant.
	ErrONTSerialNumberExists = errors.New("serial number ont sudah ada")

	// ErrONTPositionExists dikembalikan saat posisi (olt_id, pon_port, ont_index) sudah terisi.
	ErrONTPositionExists = errors.New("posisi ont sudah terisi pada port ini")

	// ErrONTAlreadyProvisioned dikembalikan saat ONT sudah dalam status provisioned.
	ErrONTAlreadyProvisioned = errors.New("ont sudah di-provision")

	// ErrONTNotProvisioned dikembalikan saat operasi membutuhkan ONT provisioned (misal reboot).
	ErrONTNotProvisioned = errors.New("ont belum di-provision")

	// ErrCustomerHasActiveONT dikembalikan saat customer sudah punya ONT aktif.
	ErrCustomerHasActiveONT = errors.New("pelanggan sudah memiliki ont aktif")

	// ErrProvisioningInProgress dikembalikan saat provisioning sedang berjalan untuk ONT ini.
	ErrProvisioningInProgress = errors.New("provisioning sedang berjalan untuk ont ini")

	// ErrProvisioningFailed dikembalikan saat CLI command gagal saat provisioning.
	ErrProvisioningFailed = errors.New("provisioning gagal, periksa audit log untuk detail")

	// ErrDecommissionFailed dikembalikan saat CLI command gagal saat decommission.
	ErrDecommissionFailed = errors.New("decommission gagal, periksa audit log untuk detail")

	// ErrRebootFailed dikembalikan saat CLI command gagal saat reboot.
	ErrRebootFailed = errors.New("reboot ont gagal")

	// --- VLAN Errors ---

	// ErrVLANNotFound dikembalikan saat VLAN tidak ditemukan.
	ErrVLANNotFound = errors.New("vlan tidak ditemukan")

	// ErrVLANIDExists dikembalikan saat VLAN ID sudah ada pada OLT yang sama.
	ErrVLANIDExists = errors.New("vlan id sudah ada pada olt ini")

	// ErrVLANInUse dikembalikan saat VLAN masih digunakan oleh ONT aktif.
	ErrVLANInUse = errors.New("vlan masih digunakan oleh ont aktif")

	// ErrVLANResolutionFailed dikembalikan saat resolusi VLAN gagal berdasarkan strategy.
	ErrVLANResolutionFailed = errors.New("gagal menentukan vlan berdasarkan strategi")

	// --- Service Profile Errors ---

	// ErrServiceProfileNotFound dikembalikan saat service profile tidak ditemukan.
	ErrServiceProfileNotFound = errors.New("service profile tidak ditemukan")

	// ErrServiceProfileExists dikembalikan saat kombinasi profile sudah ada pada OLT.
	ErrServiceProfileExists = errors.New("kombinasi line/service profile sudah ada pada olt ini")

	// ErrServiceProfileInUse dikembalikan saat profile masih digunakan oleh ONT aktif.
	ErrServiceProfileInUse = errors.New("service profile masih digunakan oleh ont aktif")

	// ErrNoProfileMapping dikembalikan saat paket pelanggan tidak punya mapping ke OLT profile.
	ErrNoProfileMapping = errors.New("paket pelanggan tidak memiliki mapping service profile pada olt ini")

	// --- Bulk Provisioning Errors ---

	// ErrBulkNotFound dikembalikan saat bulk_id tidak ditemukan.
	ErrBulkNotFound = errors.New("bulk provisioning tidak ditemukan")

	// ErrInvalidCSVFormat dikembalikan saat format CSV tidak valid.
	ErrInvalidCSVFormat = errors.New("format csv tidak valid, gunakan template yang disediakan")

	// ErrBulkAlreadyExecuted dikembalikan saat bulk sudah dieksekusi.
	ErrBulkAlreadyExecuted = errors.New("bulk provisioning sudah dieksekusi")

	// --- Settings Errors ---

	// ErrInvalidVLANStrategy dikembalikan saat VLAN strategy tidak valid.
	ErrInvalidVLANStrategy = errors.New("vlan strategy tidak valid")

	// --- Map Domain Errors ---

	// ErrMapNodeNotFound dikembalikan saat node peta tidak ditemukan.
	ErrMapNodeNotFound = errors.New("node peta tidak ditemukan")

	// ErrMapNodeDuplicate dikembalikan saat node peta duplikat
	// (tenant_id, node_type, reference_id sudah ada).
	ErrMapNodeDuplicate = errors.New("node peta duplikat")

	// ErrMapNodeDeleted dikembalikan saat node peta sudah dihapus.
	ErrMapNodeDeleted = errors.New("node peta sudah dihapus")

	// ErrInvalidNodeType dikembalikan saat tipe node tidak valid.
	ErrInvalidNodeType = errors.New("tipe node tidak valid")

	// ErrInvalidCoordinates dikembalikan saat koordinat GPS di luar range valid
	// (latitude: -90 sampai 90, longitude: -180 sampai 180).
	ErrInvalidCoordinates = errors.New("koordinat tidak valid")

	// ErrReferenceNotFound dikembalikan saat entitas referensi (OLT/ODP/ONT) tidak ditemukan.
	ErrReferenceNotFound = errors.New("entitas referensi tidak ditemukan")

	// ErrCableRouteNotFound dikembalikan saat cable route tidak ditemukan.
	ErrCableRouteNotFound = errors.New("cable route tidak ditemukan")

	// ErrInvalidRouteType dikembalikan saat tipe route tidak valid.
	ErrInvalidRouteType = errors.New("tipe route tidak valid")

	// ErrInvalidCoordArray dikembalikan saat array koordinat tidak valid (kurang dari 2 titik).
	ErrInvalidCoordArray = errors.New("array koordinat tidak valid")

	// ErrNodeNotFound dikembalikan saat node tidak ditemukan (generic).
	ErrNodeNotFound = errors.New("node tidak ditemukan")

	// ErrPhotoLimitReached dikembalikan saat batas foto per node tercapai (max 5).
	ErrPhotoLimitReached = errors.New("batas foto per node tercapai")

	// ErrInvalidFileType dikembalikan saat tipe file tidak diizinkan.
	ErrInvalidFileType = errors.New("tipe file tidak diizinkan")

	// ErrFileTooLarge dikembalikan saat ukuran file melebihi batas.
	ErrFileTooLarge = errors.New("ukuran file melebihi batas")

	// ErrPhotoNotFound dikembalikan saat foto tidak ditemukan.
	ErrPhotoNotFound = errors.New("foto tidak ditemukan")

	// ErrUnsupportedFormat dikembalikan saat format export/import tidak didukung.
	ErrUnsupportedFormat = errors.New("format tidak didukung")

	// ErrInvalidImportFile dikembalikan saat file import tidak valid.
	ErrInvalidImportFile = errors.New("file import tidak valid")

	// ErrImportNotFound dikembalikan saat import job tidak ditemukan.
	ErrImportNotFound = errors.New("import job tidak ditemukan")

	// ErrExportNotFound dikembalikan saat export job tidak ditemukan.
	ErrExportNotFound = errors.New("export job tidak ditemukan")

	// ErrShareLinkNotFound dikembalikan saat share link tidak ditemukan.
	ErrShareLinkNotFound = errors.New("share link tidak ditemukan")

	// ErrShareLinkExpired dikembalikan saat share link sudah kedaluwarsa.
	ErrShareLinkExpired = errors.New("share link sudah kedaluwarsa")

	// ErrShareLinkPassword dikembalikan saat password share link salah.
	ErrShareLinkPassword = errors.New("password share link salah")

	// ErrGeocodingFailed dikembalikan saat reverse geocoding gagal.
	ErrGeocodingFailed = errors.New("reverse geocoding gagal")

	// ErrGeocodingRateLimit dikembalikan saat rate limit geocoding tercapai.
	ErrGeocodingRateLimit = errors.New("rate limit geocoding tercapai")

	// ErrInvalidLossInput dikembalikan saat input loss calculator tidak valid
	// (splitter type tidak dikenal, jarak negatif, atau count negatif).
	ErrInvalidLossInput = errors.New("input loss calculator tidak valid")
)
