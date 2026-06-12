param(
  [Parameter(Mandatory=$true)] [string]$PackageUrl,
  [string]$PackageSha256Url,
  [string]$ExpectedPackageSha256,
  [string]$InstallDir = (Split-Path -Parent $MyInvocation.MyCommand.Path),
  [switch]$Restart
)

$ErrorActionPreference = "Stop"
$PackageMaxBytes = 1GB

function Test-GardenerAdmin {
  try {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  } catch {
    return $false
  }
}

function Unblock-GardenerPath([string]$Path) {
  if (-not $Path -or -not (Test-Path $Path)) { return }
  try {
    if ((Get-Item $Path) -is [IO.DirectoryInfo]) {
      Get-ChildItem -Path $Path -Recurse -Force -ErrorAction SilentlyContinue | Unblock-File -ErrorAction SilentlyContinue
    } else {
      Unblock-File -Path $Path -ErrorAction SilentlyContinue
    }
  } catch {
    Write-Host "Warning: could not unblock a Gardener file." -ForegroundColor Yellow
  }
}



function Get-GardenerSha256FromText([string]$Text) {
  $m = [regex]::Match($Text, '(?i)\b[0-9a-f]{64}\b')
  if (-not $m.Success) { throw "Package SHA256 digest is missing or invalid." }
  return $m.Value.ToLowerInvariant()
}

function Test-GardenerPackageHash([string]$Path, [string]$Sha256Url, [string]$ExpectedSha256) {
  $expected = $ExpectedSha256
  if (-not $expected -and $Sha256Url) {
    Write-Host "Loading Gardener package checksum..." -ForegroundColor Green
    $expected = Get-GardenerSha256FromText (Invoke-WebRequest -Uri $Sha256Url).Content
  }
  if (-not $expected) { return }
  $expected = Get-GardenerSha256FromText $expected
  $actual = (Get-FileHash -Algorithm SHA256 -Path $Path).Hash.ToLowerInvariant()
  if ($actual -ne $expected) { throw "Package SHA256 mismatch: expected $expected but got $actual" }
  Write-Host "Package checksum verified." -ForegroundColor Green
}

function Test-GardenerZipEntries([string]$ZipPath) {
  Add-Type -AssemblyName System.IO.Compression.FileSystem
  $archive = [System.IO.Compression.ZipFile]::OpenRead($ZipPath)
  try {
    foreach ($entry in $archive.Entries) {
      $name = $entry.FullName.Replace('\', '/')
      if ([string]::IsNullOrWhiteSpace($name)) { continue }
      if ($name.StartsWith('/') -or $name -match '(^|/)\.\.($|/)') {
        throw "Unsafe archive path: $name"
      }
    }
  } finally {
    $archive.Dispose()
  }
}

function Unblock-GardenerPackageFiles([string]$Dir) {
  foreach ($name in @("gardener.exe", "frpc.exe", "start-gardener.ps1", "start-gardener.bat", "update-gardener.ps1", "install-gardener.ps1", "README-Windows.txt", "gardener.config.example.ps1", "frpc.example.toml")) {
    $path = Join-Path $Dir $name
    if (Test-Path -LiteralPath $path -PathType Leaf) {
      Unblock-GardenerPath -Path $path
    }
  }
}


function Stop-GardenerInstalledProcess([string]$ProcessName, [string]$ExePath) {
  $fullPath = [IO.Path]::GetFullPath($ExePath)
  Get-CimInstance Win32_Process -Filter "name = '$ProcessName'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -and ([IO.Path]::GetFullPath([string]$_.ExecutablePath) -eq $fullPath) } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
}

function Set-GardenerFirewallPolicy([string]$ExePath) {
  if (-not (Test-Path $ExePath)) { return }
  if (-not (Test-GardenerAdmin)) {
    Write-Host "Firewall: skipped rule setup because PowerShell is not running as Administrator. Gardener listens on 127.0.0.1 by default, so Windows should not need an inbound network prompt." -ForegroundColor Yellow
    return
  }
  try {
    $ruleName = "Gardener local service - block external inbound"
    Get-NetFirewallRule -DisplayName $ruleName -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue
    New-NetFirewallRule -DisplayName $ruleName -Direction Inbound -Program $ExePath -Action Block -Profile Any -Enabled True | Out-Null
    Write-Host "Firewall: external inbound access to gardener.exe is blocked; local browser and relay access still work." -ForegroundColor Green
  } catch {
    Write-Host "Warning: could not configure Windows Firewall rule." -ForegroundColor Yellow
  }
}

function Remove-OldGardenerBackups([string]$Dir, [int]$Keep = 5) {
  Get-ChildItem -Path $Dir -Directory -Filter "backup-*" -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending |
    Select-Object -Skip $Keep |
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
}

$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$Temp = Join-Path $env:TEMP ("gardener-update-" + [guid]::NewGuid().ToString("N"))
$Zip = Join-Path $Temp "Gardener-Windows.zip"
$Extract = Join-Path $Temp "extract"
$Backup = Join-Path $InstallDir ("backup-" + (Get-Date -Format "yyyyMMdd-HHmmss"))

trap {
  Remove-Item -Path $Temp -Recurse -Force -ErrorAction SilentlyContinue
  throw
}

New-Item -ItemType Directory -Force -Path $Temp, $Extract | Out-Null
Write-Host "Downloading Gardener package..." -ForegroundColor Green
Invoke-WebRequest -Uri $PackageUrl -OutFile $Zip
if ((Get-Item $Zip).Length -gt $PackageMaxBytes) {
  throw "Gardener package archive is too large."
}
Unblock-GardenerPath -Path $Zip
Test-GardenerPackageHash -Path $Zip -Sha256Url $PackageSha256Url -ExpectedSha256 $ExpectedPackageSha256
Test-GardenerZipEntries -ZipPath $Zip
Expand-Archive -Path $Zip -DestinationPath $Extract -Force
Unblock-GardenerPackageFiles -Dir $Extract

$Source = Get-ChildItem -Path $Extract -Directory | Where-Object { $_.Name -eq "Gardener-Windows" } | Select-Object -First 1
if ($null -eq $Source) { throw "Package is missing expected Gardener-Windows directory" }

Write-Host "Stopping running Gardener processes for this install directory if any..."
Stop-GardenerInstalledProcess -ProcessName "gardener.exe" -ExePath (Join-Path $InstallDir "gardener.exe")
Stop-GardenerInstalledProcess -ProcessName "frpc.exe" -ExePath (Join-Path $InstallDir "frpc.exe")
Start-Sleep -Seconds 1

New-Item -ItemType Directory -Force -Path $Backup | Out-Null
foreach ($name in @("gardener.exe", "frpc.exe", "web", "start-gardener.bat", "start-gardener.ps1", "README-Windows.txt")) {
  $target = Join-Path $InstallDir $name
  if (Test-Path $target) { Move-Item -Path $target -Destination (Join-Path $Backup $name) -Force }
}

foreach ($name in @("gardener.exe", "frpc.exe", "web", "start-gardener.bat", "start-gardener.ps1", "README-Windows.txt")) {
  $src = Join-Path $Source.FullName $name
  if (Test-Path $src) { Copy-Item -Path $src -Destination (Join-Path $InstallDir $name) -Recurse -Force }
}

# Preserve local relay configuration, user config and data. Copy scripts/examples only if absent or safe to replace.
foreach ($name in @("gardener.config.example.ps1", "frpc.example.toml", "update-gardener.ps1", "install-gardener.ps1")) {
  $src = Join-Path $Source.FullName $name
  if (Test-Path $src) { Copy-Item -Path $src -Destination (Join-Path $InstallDir $name) -Force }
}

Unblock-GardenerPackageFiles -Dir $InstallDir
Set-GardenerFirewallPolicy -ExePath (Join-Path $InstallDir "gardener.exe")

Remove-Item -Path $Temp -Recurse -Force -ErrorAction SilentlyContinue
Remove-OldGardenerBackups -Dir $InstallDir
Write-Host "Gardener updated successfully." -ForegroundColor Green
Write-Host "Backup: created under the Gardener install directory."

if ($Restart) {
  Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$(Join-Path $InstallDir 'start-gardener.ps1')`""
}
