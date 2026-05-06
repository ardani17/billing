param(
  [string]$EnvPath = "docker/.env.production"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $EnvPath)) {
  throw "Env file tidak ditemukan: $EnvPath"
}

$required = @(
  "APP_ENV",
  "DB_HOST",
  "DB_USER",
  "DB_PASSWORD",
  "DB_NAME",
  "DB_SSL_MODE",
  "REDIS_HOST",
  "JWT_SECRET",
  "CORS_ALLOW_ORIGINS",
  "BILLING_API_URL",
  "ISPBOSS_ENABLE_DEV_AUTH"
)

$unsafeValues = @(
  "change-me-to-a-strong-secret",
  "ispboss_secret",
  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

$envMap = @{}
Get-Content -LiteralPath $EnvPath | ForEach-Object {
  $line = $_.Trim()
  if ($line -eq "" -or $line.StartsWith("#")) { return }
  $parts = $line.Split("=", 2)
  if ($parts.Count -eq 2) {
    $envMap[$parts[0]] = $parts[1]
  }
}

function Get-EnvValue {
  param([string]$Key)
  if ($envMap.ContainsKey($Key)) {
    return $envMap[$Key]
  }
  return ""
}

$errors = @()

foreach ($key in $required) {
  if (-not $envMap.ContainsKey($key) -or [string]::IsNullOrWhiteSpace($envMap[$key])) {
    $errors += "$key wajib diisi"
  }
}

if ((Get-EnvValue "APP_ENV") -ne "production") {
  $errors += "APP_ENV harus production"
}

if ((Get-EnvValue "ISPBOSS_ENABLE_DEV_AUTH") -ne "false") {
  $errors += "ISPBOSS_ENABLE_DEV_AUTH harus false"
}

if ((Get-EnvValue "DB_SSL_MODE") -eq "disable") {
  $errors += "DB_SSL_MODE production tidak boleh disable"
}

foreach ($key in $envMap.Keys) {
  if ($unsafeValues -contains $envMap[$key]) {
    $errors += "$key masih memakai value development"
  }
  if ($envMap[$key] -like "replace-with-*") {
    $errors += "$key masih memakai placeholder"
  }
}

if ((Get-EnvValue "JWT_SECRET").Length -lt 32) {
  $errors += "JWT_SECRET minimal 32 karakter"
}

if ((Get-EnvValue "CORS_ALLOW_ORIGINS").Contains("*")) {
  $errors += "CORS_ALLOW_ORIGINS tidak boleh wildcard"
}

if ($errors.Count -gt 0) {
  Write-Host "Production env belum aman:"
  $errors | ForEach-Object { Write-Host "- $_" }
  exit 1
}

Write-Host "Production env preflight lolos: $EnvPath"
