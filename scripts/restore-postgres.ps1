param(
  [Parameter(Mandatory = $true)]
  [string]$BackupPath,
  [string]$ContainerName = "ispboss-postgres",
  [string]$Database = "ispboss_restore",
  [string]$User = "ispboss"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $BackupPath)) {
  throw "File backup tidak ditemukan: $BackupPath"
}

$fileName = Split-Path -Leaf $BackupPath
$containerPath = "/tmp/$fileName"

docker cp $BackupPath "${ContainerName}:$containerPath"
docker exec $ContainerName dropdb -U $User --if-exists $Database
docker exec $ContainerName createdb -U $User $Database
docker exec $ContainerName pg_restore -U $User -d $Database --clean --if-exists $containerPath

Write-Host "Restore selesai ke database: $Database"
