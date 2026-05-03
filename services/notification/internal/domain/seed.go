package domain

// =============================================================================
// Default Templates — template bawaan yang di-seed saat tenant pertama kali
// mengaktifkan konfigurasi notifikasi. Semua template memiliki is_default=true
// dan is_active=true, serta mendukung channel WhatsApp dan SMS.
// =============================================================================

// DefaultTemplates berisi daftar template bawaan untuk seeding.
// Template ini akan dibuat otomatis saat tenant pertama kali menyimpan
// konfigurasi notifikasi. Admin bisa mengedit isi template, tapi tidak
// bisa menghapus template default.
var DefaultTemplates = []NotificationTemplate{
	// Template: invoice_new — notifikasi invoice baru dibuat
	{
		Slug:      "invoice_new",
		Name:      "Invoice Baru",
		Category:  CategoryTransactional,
		EventType: "invoice.created",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, invoice baru #{no_invoice} untuk paket {paket} " +
			"periode {periode} sebesar {total_tagihan} telah dibuat. " +
			"Jatuh tempo: {jatuh_tempo}. Bayar via: {link_bayar_short}",
		BodySMS: "ISP {nama_isp}: Invoice #{no_invoice} {paket} {periode} " +
			"Rp{total_tagihan}. JT: {jatuh_tempo}. Bayar: {link_bayar_short}",
		Variables: []string{
			"nama", "no_invoice", "paket", "periode",
			"total_tagihan", "jatuh_tempo", "link_bayar_short", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: reminder_h1 — pengingat tagihan H-1 jatuh tempo
	{
		Slug:      "reminder_h1",
		Name:      "Pengingat H-1 Jatuh Tempo",
		Category:  CategoryReminder,
		EventType: "invoice.reminder",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, tagihan #{no_invoice} sebesar {total_tagihan} " +
			"jatuh tempo besok ({jatuh_tempo}). Segera bayar untuk menghindari " +
			"isolir. Bayar via: {link_bayar_short}",
		BodySMS: "ISP {nama_isp}: Tagihan #{no_invoice} {total_tagihan} " +
			"JT besok. Bayar: {link_bayar_short}",
		Variables: []string{
			"nama", "no_invoice", "total_tagihan",
			"jatuh_tempo", "link_bayar_short", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: payment_confirm — konfirmasi pembayaran diterima
	{
		Slug:      "payment_confirm",
		Name:      "Konfirmasi Pembayaran",
		Category:  CategoryTransactional,
		EventType: "payment.online.received",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, pembayaran sebesar {jumlah_bayar} untuk " +
			"invoice #{no_invoice} telah diterima pada {tanggal_bayar} " +
			"via {metode_bayar}. Terima kasih!",
		BodySMS: "ISP {nama_isp}: Pembayaran {jumlah_bayar} invoice " +
			"#{no_invoice} diterima. Terima kasih!",
		Variables: []string{
			"nama", "jumlah_bayar", "no_invoice",
			"tanggal_bayar", "metode_bayar", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: isolir_notice — pemberitahuan layanan diisolir
	{
		Slug:      "isolir_notice",
		Name:      "Pemberitahuan Isolir",
		Category:  CategoryInformation,
		EventType: "notification.isolir",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, layanan internet Anda telah diisolir karena " +
			"tagihan belum dibayar. Silakan bayar tagihan untuk mengaktifkan " +
			"kembali. Info: {telepon_isp}",
		BodySMS: "ISP {nama_isp}: Layanan diisolir. Bayar tagihan " +
			"untuk aktivasi. Info: {telepon_isp}",
		Variables: []string{
			"nama", "telepon_isp", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: suspend_notice — pemberitahuan layanan disuspend
	{
		Slug:      "suspend_notice",
		Name:      "Pemberitahuan Suspend",
		Category:  CategoryInformation,
		EventType: "notification.suspend",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, layanan internet Anda telah disuspend. " +
			"Hubungi admin untuk informasi lebih lanjut: {telepon_isp}",
		BodySMS: "ISP {nama_isp}: Layanan disuspend. Hubungi: {telepon_isp}",
		Variables: []string{
			"nama", "telepon_isp", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: unblock_notice — pemberitahuan layanan diaktifkan kembali (un-isolir)
	{
		Slug:      "unblock_notice",
		Name:      "Pemberitahuan Un-Isolir",
		Category:  CategoryInformation,
		EventType: "notification.un_isolir",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, layanan internet Anda telah diaktifkan kembali. " +
			"Terima kasih telah melakukan pembayaran. Selamat berselancar!",
		BodySMS: "ISP {nama_isp}: Layanan aktif kembali. Terima kasih!",
		Variables: []string{
			"nama", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},

	// Template: reactivated_notice — pemberitahuan layanan direaktivasi
	{
		Slug:      "reactivated_notice",
		Name:      "Pemberitahuan Reaktivasi",
		Category:  CategoryInformation,
		EventType: "notification.reactivated",
		Channels:  []Channel{ChannelWhatsApp, ChannelSMS},
		BodyWhatsApp: "Halo {nama}, layanan internet Anda telah direaktivasi. " +
			"Selamat berselancar kembali!",
		BodySMS: "ISP {nama_isp}: Layanan direaktivasi. Selamat berselancar!",
		Variables: []string{
			"nama", "nama_isp",
		},
		IsActive:  true,
		IsDefault: true,
	},
}
