param(
  [Parameter(Mandatory=$true)] [string]$PackageUrl,
  [string]$InstallDir = (Split-Path -Parent $MyInvocation.MyCommand.Path),
  [switch]$Restart
)

$ErrorActionPreference = "Stop"

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
    Write-Host "Warning: could not unblock $Path: $_" -ForegroundColor Yellow
  }
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
    Write-Host "Warning: could not configure Windows Firewall rule: $_" -ForegroundColor Yellow
  }
}

$InstallDir = [IO.Path]::GetFullPath($InstallDir)
$Temp = Join-Path $env:TEMP ("gardener-update-" + [guid]::NewGuid().ToString("N"))
$Zip = Join-Path $Temp "Gardener-Windows.zip"
$Extract = Join-Path $Temp "extract"
$Backup = Join-Path $InstallDir ("backup-" + (Get-Date -Format "yyyyMMdd-HHmmss"))

New-Item -ItemType Directory -Force -Path $Temp, $Extract | Out-Null
Write-Host "Downloading $PackageUrl" -ForegroundColor Green
Invoke-WebRequest -Uri $PackageUrl -OutFile $Zip
Unblock-GardenerPath -Path $Zip
Test-GardenerZipEntries -ZipPath $Zip
Expand-Archive -Path $Zip -DestinationPath $Extract -Force
Unblock-GardenerPath -Path $Extract

$Source = Get-ChildItem -Path $Extract -Directory | Select-Object -First 1
if ($null -eq $Source) { $Source = Get-Item $Extract }

Write-Host "Stopping running gardener.exe/frpc.exe if any..."
Get-Process gardener -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Get-Process frpc -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
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

Unblock-GardenerPath -Path $InstallDir
Set-GardenerFirewallPolicy -ExePath (Join-Path $InstallDir "gardener.exe")

Remove-Item -Path $Temp -Recurse -Force -ErrorAction SilentlyContinue
Write-Host "Gardener updated successfully." -ForegroundColor Green
Write-Host "Backup: $Backup"

if ($Restart) {
  Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$(Join-Path $InstallDir 'start-gardener.ps1')`""
}
