param(
  [string]$ContainerName = "ispboss-postgres",
  [string]$Database = "ispboss",
  [string]$User = "ispboss",
  [string]$OutputDir = "backups"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "Docker command tidak ditemukan"
}

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$backupDir = Join-Path (Get-Location) $OutputDir
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null

$fileName = "ispboss-$Database-$timestamp.dump"
$outputPath = Join-Path $backupDir $fileName
$containerPath = "/tmp/$fileName"

docker exec $ContainerName pg_dump -U $User -d $Database -Fc -f $containerPath
docker cp "${ContainerName}:$containerPath" $outputPath
docker exec $ContainerName rm -f $containerPath | Out-Null

Write-Host "Backup selesai: $outputPath"
