param(
  [string]$BaseUrl = "http://localhost:3000"
)

$ErrorActionPreference = "Stop"

$routes = @(
  "/",
  "/dashboard",
  "/customers",
  "/customers/new",
  "/customers/areas",
  "/packages",
  "/packages/new",
  "/invoices",
  "/payments",
  "/resellers",
  "/reseller",
  "/vouchers",
  "/expenses",
  "/inventory",
  "/cashflow",
  "/reports",
  "/reports/reconciliation",
  "/notifications",
  "/settings",
  "/settings/billing",
  "/settings/payment",
  "/settings/notifications",
  "/settings/security",
  "/settings/users",
  "/settings/subscription",
  "/super-admin",
  "/super-admin/tenants",
  "/super-admin/subscriptions",
  "/super-admin/upgrade-requests",
  "/super-admin/support",
  "/super-admin/health",
  "/super-admin/audit",
  "/super-admin/settings"
)

$failures = @()

foreach ($route in $routes) {
  $url = "$BaseUrl$route"
  try {
    $response = Invoke-WebRequest -Uri $url -Method GET -UseBasicParsing -TimeoutSec 20
    if ($response.StatusCode -lt 200 -or $response.StatusCode -ge 400) {
      $failures += "$route -> HTTP $($response.StatusCode)"
    } else {
      Write-Host "OK $route ($($response.StatusCode))"
    }
  } catch {
    $failures += "$route -> $($_.Exception.Message)"
  }
}

if ($failures.Count -gt 0) {
  Write-Host ""
  Write-Host "Smoke route gagal:"
  $failures | ForEach-Object { Write-Host "- $_" }
  exit 1
}

Write-Host ""
Write-Host "Semua smoke route berhasil."
