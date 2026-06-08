param(
  [string]$PackageUrl,
  [string]$InstallDir = "$env:LOCALAPPDATA\Gardener",
  [string]$RelayBaseUrl,
  [string]$SetupKey,
  [string]$User,
  [string]$ProvisionUrl,
  [switch]$StartMenuShortcut,
  [switch]$DesktopShortcut,
  [switch]$StartAfterInstall
)

$ErrorActionPreference = "Stop"

if (-not $RelayBaseUrl -and $env:GARDENER_RELAY_BASE_URL) { $RelayBaseUrl = $env:GARDENER_RELAY_BASE_URL }
if (-not $RelayBaseUrl) { $RelayBaseUrl = "http://YOUR_RELAY_SERVER" }

if ($SetupKey -and ($SetupKey -notmatch '^[A-Za-z0-9_-]{20,}$')) {
  throw "Setup key format is invalid."
}

function Test-GardenerPlaceholderUrl([string]$Url) {
  return [string]::IsNullOrWhiteSpace($Url) -or $Url -match 'YOUR_RELAY_SERVER|YOUR_SERVER_IP|example\.com'
}

function Invoke-GardenerDownload($Uri, $OutFile) {
  Write-Host "Downloading Gardener file..." -ForegroundColor Green
  Invoke-WebRequest -Uri $Uri -OutFile $OutFile
  Unblock-GardenerPath -Path $OutFile
}

function ConvertTo-PlainJson($Value) {
  return ($Value | ConvertTo-Json -Depth 10)
}

function Test-GardenerAdmin {
  try {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
  } catch {
    return $false
  }
}


function Protect-GardenerSecretFile([string]$Path) {
  if (-not $Path -or -not (Test-Path $Path)) { return }
  try {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent().Name
    $acl = Get-Acl -Path $Path
    $acl.SetAccessRuleProtection($true, $false)
    $rule = New-Object Security.AccessControl.FileSystemAccessRule($identity, "FullControl", "Allow")
    $acl.SetAccessRule($rule)
    Set-Acl -Path $Path -AclObject $acl
  } catch {
    Write-Host "Warning: could not restrict permissions on $Path: $_" -ForegroundColor Yellow
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

function Set-GardenerFirewallPolicy([string]$ExePath) {
  if (-not (Test-Path $ExePath)) { return }
  if (-not (Test-GardenerAdmin)) {
    Write-Host "Firewall: skipped rule setup because PowerShell is not running as Administrator. Gardener now listens on 127.0.0.1 only, so Windows should not need an inbound network prompt." -ForegroundColor Yellow
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

$Provision = $null
if (-not $ProvisionUrl -and $SetupKey) {
  if (Test-GardenerPlaceholderUrl $RelayBaseUrl) {
    throw "RelayBaseUrl is not configured. Re-run with -RelayBaseUrl http://YOUR_SERVER or set GARDENER_RELAY_BASE_URL."
  }
  $RelayBaseUrl = $RelayBaseUrl.TrimEnd('/')
  $ProvisionUrl = "$RelayBaseUrl/downloads/provision/$SetupKey/gardener.provision.json"
}

if ($ProvisionUrl) {
  Write-Host "Loading Gardener relay provision..." -ForegroundColor Green
  $Provision = Invoke-RestMethod -Uri $ProvisionUrl
  if ($User -and $Provision.user -and ($User.ToLowerInvariant() -ne [string]($Provision.user).ToLowerInvariant())) {
    throw "Provision user mismatch: expected $User but got $($Provision.user)"
  }
  if (-not $PackageUrl -and $Provision.packageUrl) { $PackageUrl = [string]$Provision.packageUrl }
}

if (-not $PackageUrl) {
  if (Test-GardenerPlaceholderUrl $RelayBaseUrl) {
    throw "PackageUrl is not configured. Re-run with -PackageUrl http://YOUR_SERVER/downloads/Gardener-Windows.zip or set GARDENER_RELAY_BASE_URL."
  }
  $PackageUrl = "$($RelayBaseUrl.TrimEnd('/'))/downloads/Gardener-Windows.zip"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Updater = Join-Path $InstallDir "update-gardener.ps1"
if (-not (Test-Path $Updater)) {
  $LocalUpdater = Join-Path $ScriptDir "update-gardener.ps1"
  if (Test-Path $LocalUpdater) {
    Copy-Item -Path $LocalUpdater -Destination $Updater -Force
    Unblock-GardenerPath -Path $Updater
  } else {
    $UpdaterUrl = "$($RelayBaseUrl.TrimEnd('/'))/downloads/update-gardener.ps1"
    Invoke-GardenerDownload -Uri $UpdaterUrl -OutFile $Updater
  }
}

& $Updater -PackageUrl $PackageUrl -InstallDir $InstallDir
Unblock-GardenerPath -Path $InstallDir

$FrpcExe = Join-Path $InstallDir "frpc.exe"
if (-not (Test-Path $FrpcExe)) {
  $FrpcUrl = "$($RelayBaseUrl.TrimEnd('/'))/downloads/frpc.exe"
  try {
    Invoke-GardenerDownload -Uri $FrpcUrl -OutFile $FrpcExe
  } catch {
    Write-Host "Warning: could not download frpc.exe automatically: $_" -ForegroundColor Yellow
  }
}

$ConfigText = @"
# Generated by install-gardener.ps1. You usually do not need to edit this file.
# Local-only binding avoids Windows Defender Firewall prompts and prevents LAN exposure.
`$env:AUTO_GARDENER_ADDR = "127.0.0.1:8080"
`$env:AUTO_GARDENER_STATIC = "`$PSScriptRoot\web\static"
`$env:AUTO_GARDENER_DATA = "`$([Environment]::GetFolderPath('Desktop'))\forest_data"
"@

if ($Provision) {
  Write-Host "Writing relay configuration..." -ForegroundColor Green
  if (-not $Provision.frpcToml) { throw "Provision is missing frpcToml" }
  $FrpcConfigPath = Join-Path $InstallDir "frpc.toml"
  [IO.File]::WriteAllText($FrpcConfigPath, [string]$Provision.frpcToml, [Text.UTF8Encoding]::new($false))
  Protect-GardenerSecretFile -Path $FrpcConfigPath

  $RelayConfig = [ordered]@{
    schemaVersion = 1
    user = [string]$Provision.user
    publicUrl = [string]$Provision.publicUrl
    webUsername = [string]$Provision.webUsername
    webPassword = [string]$Provision.webPassword
    installedAt = (Get-Date).ToString("o")
  }
  $RelayJsonPath = Join-Path $InstallDir "gardener.relay.json"
  [IO.File]::WriteAllText($RelayJsonPath, (ConvertTo-PlainJson $RelayConfig), [Text.UTF8Encoding]::new($false))
  Protect-GardenerSecretFile -Path $RelayJsonPath
}

[IO.File]::WriteAllText((Join-Path $InstallDir "gardener.config.ps1"), $ConfigText, [Text.UTF8Encoding]::new($false))
Unblock-GardenerPath -Path $InstallDir
Set-GardenerFirewallPolicy -ExePath (Join-Path $InstallDir "gardener.exe")

$StartScript = Join-Path $InstallDir "start-gardener.ps1"
if ($DesktopShortcut -or $StartMenuShortcut) {
  $Wsh = New-Object -ComObject WScript.Shell
  $targets = @()
  if ($DesktopShortcut) { $targets += Join-Path ([Environment]::GetFolderPath("Desktop")) "Gardener.lnk" }
  if ($StartMenuShortcut) {
    $dir = Join-Path ([Environment]::GetFolderPath("Programs")) "Gardener"
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    $targets += Join-Path $dir "Gardener.lnk"
  }
  foreach ($lnk in $targets) {
    $Shortcut = $Wsh.CreateShortcut($lnk)
    $Shortcut.TargetPath = "powershell.exe"
    $Shortcut.Arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$StartScript`""
    $Shortcut.WorkingDirectory = $InstallDir
    $Shortcut.Save()
  }
}

Write-Host "Gardener installed to $InstallDir" -ForegroundColor Green
if ($Provision) {
  Write-Host "Remote URL: $($Provision.publicUrl)" -ForegroundColor Cyan
  Write-Host "Login user: $($Provision.webUsername)" -ForegroundColor Cyan
  Write-Host "Login password: saved in gardener.relay.json; keep that file private." -ForegroundColor Cyan
}
if ($StartAfterInstall) { Start-Process powershell -ArgumentList "-NoProfile -ExecutionPolicy Bypass -File `"$StartScript`"" }
