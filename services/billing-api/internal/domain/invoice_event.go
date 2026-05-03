package domain

// InvoiceCreatedPayload adalah payload event invoice.created.
// Dikirim saat invoice berhasil dibuat (manual maupun otomatis via cron).
type InvoiceCreatedPayload struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	InvoiceNumber string `json:"invoice_number"`
	TotalAmount   int64  `json:"total_amount"`
	DueDate       string `json:"due_date"`
}

// InvoiceOverduePayload adalah payload event invoice.overdue.
// Dikirim saat invoice berubah status menjadi terlambat oleh cron harian.
type InvoiceOverduePayload struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	InvoiceNumber string `json:"invoice_number"`
	TotalAmount   int64  `json:"total_amount"`
	DaysOverdue   int    `json:"days_overdue"`
}

// InvoiceCancelledPayload adalah payload event invoice.cancelled.
// Dikirim saat invoice dibatalkan oleh admin tenant.
type InvoiceCancelledPayload struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	InvoiceNumber string `json:"invoice_number"`
	Reason        string `json:"reason"`
}

// InvoiceReminderPayload adalah payload event invoice.reminder.
// Dikirim saat pengingat pembayaran dikirim ke pelanggan (bulk reminder).
type InvoiceReminderPayload struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	CustomerID    string `json:"customer_id"`
	InvoiceNumber string `json:"invoice_number"`
	TotalAmount   int64  `json:"total_amount"`
	DueDate       string `json:"due_date"`
}
