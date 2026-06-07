param(
  [string]$Addr = "127.0.0.1:8080",
  [switch]$NoBrowser,
  [switch]$NoRelay,
  [switch]$NoRestart
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Exe = Join-Path $Root "gardener.exe"
$Config = Join-Path $Root "gardener.config.ps1"
$RelayJson = Join-Path $Root "gardener.relay.json"
$FrpcExe = Join-Path $Root "frpc.exe"
$FrpcConfig = Join-Path $Root "frpc.toml"
$FrpcOutLog = Join-Path $Root "frpc.out.log"
$FrpcErrLog = Join-Path $Root "frpc.err.log"

function Show-GardenerPowerWarning {
  try {
    $bad = @()
    foreach ($item in @(@('STANDBYIDLE','sleep'), @('HIBERNATEIDLE','hibernate'))) {
      $alias = $item[0]
      $label = $item[1]
      $out = powercfg /query SCHEME_CURRENT SUB_SLEEP $alias 2>$null | Out-String
      $ac = [regex]::Match($out, 'Current AC Power Setting Index:\s*0x([0-9a-fA-F]+)')
      $dc = [regex]::Match($out, 'Current DC Power Setting Index:\s*0x([0-9a-fA-F]+)')
      if ($ac.Success -and ([Convert]::ToInt64($ac.Groups[1].Value,16) -gt 0)) { $bad += "AC $label timeout is not Never" }
      if ($dc.Success -and ([Convert]::ToInt64($dc.Groups[1].Value,16) -gt 0)) { $bad += "Battery $label timeout is not Never" }
    }
    $lid = powercfg /query SCHEME_CURRENT SUB_BUTTONS LIDACTION 2>$null | Out-String
    $lac = [regex]::Match($lid, 'Current AC Power Setting Index:\s*0x([0-9a-fA-F]+)')
    $ldc = [regex]::Match($lid, 'Current DC Power Setting Index:\s*0x([0-9a-fA-F]+)')
    if (($lac.Success -and ([Convert]::ToInt64($lac.Groups[1].Value,16) -ne 0)) -or ($ldc.Success -and ([Convert]::ToInt64($ldc.Groups[1].Value,16) -ne 0))) { $bad += "lid close action may sleep/hibernate/shutdown" }
    if ($bad.Count -gt 0) {
      Write-Host ""
      Write-Host "WARNING: Gardener remote access requires this computer to stay awake, online, and powered on." -ForegroundColor Yellow
      $bad | ForEach-Object { Write-Host " - $_" -ForegroundColor Yellow }
      Write-Host "Set Windows Settings > System > Power & battery > Screen and sleep to Never. Do not shut down or close the lid during remote tasks." -ForegroundColor Yellow
      Write-Host "Optional admin command: powercfg /change standby-timeout-ac 0; powercfg /change standby-timeout-dc 0; powercfg /change hibernate-timeout-ac 0; powercfg /change hibernate-timeout-dc 0" -ForegroundColor Yellow
      Write-Host ""
    }
  } catch {
    Write-Host "Warning: could not check Windows power settings. Please make sure the computer never sleeps and is not shut down during remote tasks." -ForegroundColor Yellow
  }
}


if (Test-Path $Config) { . $Config }

if (-not (Test-Path $Exe)) {
  Write-Host "gardener.exe not found in $Root" -ForegroundColor Red
  Read-Host "Press Enter to exit"
  exit 1
}

try {
  Get-ChildItem -Path $Root -Recurse -Force -ErrorAction SilentlyContinue | Unblock-File -ErrorAction SilentlyContinue
} catch {
  Write-Host "Warning: could not unblock downloaded Gardener files: $_" -ForegroundColor Yellow
}

$env:PATH = "$env:APPDATA\npm;$env:ProgramFiles\nodejs;${env:ProgramFiles(x86)}\nodejs;$env:PATH"
if (-not $env:AUTO_GARDENER_ADDR) { $env:AUTO_GARDENER_ADDR = $Addr }
if (-not $env:AUTO_GARDENER_STATIC) { $env:AUTO_GARDENER_STATIC = Join-Path $Root "web\static" }
if (-not $env:AUTO_GARDENER_DATA) { $env:AUTO_GARDENER_DATA = Join-Path ([Environment]::GetFolderPath("Desktop")) "forest_data" }

$displayAddr = $env:AUTO_GARDENER_ADDR
if ($displayAddr.StartsWith(":")) { $displayAddr = "localhost$displayAddr" }
$localUrl = "http://$displayAddr"
$openUrl = $localUrl
$MaxOpenUrlLength = 2048
$Relay = $null
if (Test-Path $RelayJson) {
  try {
    $Relay = Get-Content $RelayJson -Raw | ConvertFrom-Json
    if ($Relay.publicUrl) {
      $candidateUrl = [string]$Relay.publicUrl
      if ($candidateUrl.Length -le $MaxOpenUrlLength) {
        $openUrl = $candidateUrl
      } else {
        Write-Host "Warning: remote URL is too long; opening local Gardener instead." -ForegroundColor Yellow
      }
    }
  } catch {
    Write-Host "Warning: cannot parse gardener.relay.json: $_" -ForegroundColor Yellow
  }
}

Write-Host "Starting Gardener..." -ForegroundColor Green
Write-Host "Local URL: $localUrl"
if ($Relay -and $Relay.publicUrl) {
  Write-Host "Remote URL: $($Relay.publicUrl)" -ForegroundColor Cyan
  Write-Host "Remote login: $($Relay.webUsername) / $($Relay.webPassword)" -ForegroundColor Cyan
}
Write-Host "Data: $env:AUTO_GARDENER_DATA"
Write-Host "Static: $env:AUTO_GARDENER_STATIC"
Show-GardenerPowerWarning

$frpcProc = $null
if (-not $NoRelay -and (Test-Path $FrpcConfig)) {
  if (Test-Path $FrpcExe) {
    Write-Host "Starting relay tunnel..." -ForegroundColor Green
    $escapedConfig = $FrpcConfig.Replace('''', '''''')
    Get-CimInstance Win32_Process -Filter "name = 'frpc.exe'" -ErrorAction SilentlyContinue |
      Where-Object { $_.CommandLine -like "*$escapedConfig*" -or $_.CommandLine -like "*$FrpcConfig*" } |
      ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
    $frpcProc = Start-Process -FilePath $FrpcExe -ArgumentList @("-c", $FrpcConfig) -WorkingDirectory $Root -PassThru -WindowStyle Hidden -RedirectStandardOutput $FrpcOutLog -RedirectStandardError $FrpcErrLog
  } else {
    Write-Host "Relay config exists but frpc.exe was not found; remote URL will be offline until frpc.exe is installed." -ForegroundColor Yellow
  }
}

if (-not $NoBrowser) {
  Start-Job -ScriptBlock { param($u) Start-Sleep -Seconds 3; Start-Process $u } -ArgumentList $openUrl | Out-Null
}

try {
  while ($true) {
    $startedAt = Get-Date
    try {
      & $Exe
      $exitCode = $LASTEXITCODE
      Write-Host "Gardener exited with code $exitCode." -ForegroundColor Yellow
    } catch {
      Write-Host "Gardener stopped unexpectedly: $_" -ForegroundColor Red
    }
    if ($NoRestart -or $env:AUTO_GARDENER_NO_RESTART -eq "1") { break }
    $ranFor = (Get-Date) - $startedAt
    if ($ranFor.TotalSeconds -lt 3) { Start-Sleep -Seconds 5 } else { Start-Sleep -Seconds 2 }
    Write-Host "Restarting Gardener automatically. Keep this window open for remote access; close it only when you want Gardener offline." -ForegroundColor Yellow
  }
} finally {
  if ($frpcProc -and -not $frpcProc.HasExited) {
    Stop-Process -Id $frpcProc.Id -Force -ErrorAction SilentlyContinue
  }
}
