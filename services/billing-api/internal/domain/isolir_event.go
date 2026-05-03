package domain

// =============================================================================
// Task Type Constants — konstanta tipe task untuk modul isolir
// =============================================================================

const (
	// Cron dan background task
	TaskAutoIsolirCron       = "isolir.auto_isolir_cron"
	TaskSuspendCron          = "isolir.suspend_cron"
	TaskPeriodicSync         = "isolir.periodic_sync"
	TaskPaymentOnlineReceived = "payment.online.received"
	TaskPaymentRecorded      = "payment.recorded"
	TaskPaymentVoidedReIsolir = "payment.voided.re_isolir"

	// Notifikasi task
	TaskNotifIsolir           = "notification.isolir"
	TaskNotifUnIsolir         = "notification.un_isolir"
	TaskNotifSuspend          = "notification.suspend"
	TaskNotifReactivated      = "notification.reactivated"
	TaskNotifPendingSyncFailed = "notification.pending_sync_failed"

	// Event tipe untuk sinkronisasi router
	TaskCustomerIsolir      = "customer.isolir"
	TaskCustomerUnIsolir    = "customer.un_isolir"
	TaskCustomerSuspend     = "customer.suspend"
	TaskInvoicePenaltyAdded = "invoice.penalty_added"
)

// =============================================================================
// Event Payloads — struct payload event untuk modul isolir
// =============================================================================

// CustomerIsolirPayload adalah payload event customer.isolir.
// Dikirim saat pelanggan di-isolir karena invoice terlambat.
type CustomerIsolirPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id,omitempty"`
	PPPoEUsername    string `json:"pppoe_username,omitempty"`
	ConnectionMethod string `json:"connection_method"`
	Reason           string `json:"reason"`
	OverdueDays      int    `json:"overdue_days"`
}

// CustomerUnIsolirPayload adalah payload event customer.un_isolir.
// Dikirim saat pelanggan dibuka isolirnya setelah pembayaran diterima.
type CustomerUnIsolirPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id,omitempty"`
	PPPoEUsername    string `json:"pppoe_username,omitempty"`
	ConnectionMethod string `json:"connection_method"`
	Trigger          string `json:"trigger"` // "payment_received" atau "admin_manual"
}

// CustomerSuspendPayload adalah payload event customer.suspend.
// Dikirim saat pelanggan di-suspend karena melewati batas toleransi.
type CustomerSuspendPayload struct {
	CustomerID       string `json:"customer_id"`
	TenantID         string `json:"tenant_id"`
	CustomerName     string `json:"customer_name"`
	RouterID         string `json:"router_id,omitempty"`
	PPPoEUsername    string `json:"pppoe_username,omitempty"`
	ConnectionMethod string `json:"connection_method"`
	OverdueDays      int    `json:"overdue_days"`
}

// PenaltyAddedPayload adalah payload event invoice.penalty_added.
// Dikirim saat denda ditambahkan atau dihapus dari invoice.
type PenaltyAddedPayload struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	PenaltyAmount int64  `json:"penalty_amount"`
	PenaltyType   string `json:"penalty_type"`
	InvoiceNumber string `json:"invoice_number"`
}
