-- Seed default billing settings for tenants created before billing settings existed.

INSERT INTO billing_settings (
    tenant_id,
    generate_days,
    grace_period_days,
    suspend_days,
    tax_enabled,
    tax_rate,
    penalty_enabled,
    penalty_type,
    penalty_amount,
    penalty_percentage,
    penalty_daily_amount,
    penalty_max_amount,
    invoice_prefix,
    new_customer_billing,
    timezone,
    auto_isolir,
    auto_open_isolir
)
SELECT
    t.id,
    5,
    7,
    30,
    FALSE,
    11.00,
    FALSE,
    'fixed',
    0,
    0,
    0,
    0,
    'INV',
    'prorate',
    'Asia/Jakarta',
    TRUE,
    TRUE
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1
    FROM billing_settings bs
    WHERE bs.tenant_id = t.id
);
