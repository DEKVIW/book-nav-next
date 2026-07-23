# BookNav detached start (Win32_Process.Create - survives agent Job timeout)
# Usage: .\scripts\start.ps1 [-Rebuild] [-ApiOnly] [-WebOnly] [-NoBuild]

param(
  [switch]$ApiOnly,
  [switch]$WebOnly,
  [switch]$Rebuild,
  [switch]$NoBuild
)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$DevDir = Join-Path $Root ".dev"
$LogDir = Join-Path $DevDir "logs"
$PidFile = Join-Path $DevDir "pids.json"
$ApiExe = Join-Path $Root "apps\server\bin\booknav.exe"
$DataDir = Join-Path $Root "data"
$WebDist = Join-Path $Root "apps\server\webdist"
$WebDir = Join-Path $Root "apps\web"
$ServerDir = Join-Path $Root "apps\server"

New-Item -ItemType Directory -Force -Path $DevDir, $LogDir, (Join-Path $ServerDir "bin") | Out-Null

$envExample = Join-Path $Root ".env.example"
$envFile = Join-Path $Root ".env"
if (-not (Test-Path $envFile) -and (Test-Path $envExample)) {
  Copy-Item $envExample $envFile
  Write-Host "[start] created .env from .env.example"
}

function Stop-Port([int]$Port) {
  Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue |
    ForEach-Object {
      if ($_.OwningProcess -gt 0) {
        Stop-Process -Id $_.OwningProcess -Force -ErrorAction SilentlyContinue
        Write-Host "[start] freed :$Port (pid $($_.OwningProcess))"
      }
    }
}

function Start-Breakaway {
  param([Parameter(Mandatory)][string]$CommandLine, [string]$WorkingDirectory = $Root)
  $r = Invoke-CimMethod -ClassName Win32_Process -MethodName Create -Arguments @{
    CommandLine      = $CommandLine
    CurrentDirectory = $WorkingDirectory
  }
  if ($null -eq $r -or $r.ReturnValue -ne 0) {
    $code = if ($r) { $r.ReturnValue } else { "null" }
    throw "Win32_Process.Create failed code=$code"
  }
  return [int]$r.ProcessId
}

function Wait-Http([string]$Url, [int]$Tries = 50, [int]$DelayMs = 300) {
  for ($i = 0; $i -lt $Tries; $i++) {
    try {
      $resp = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2
      if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 500) { return $true }
    } catch {}
    Start-Sleep -Milliseconds $DelayMs
  }
  return $false
}

function Escape-PsSingle([string]$s) {
  return $s.Replace("'", "''")
}

function Start-Api {
  if (-not $NoBuild -or $Rebuild -or -not (Test-Path $ApiExe)) {
    Write-Host "[start] building API..."
    Push-Location $ServerDir
    try {
      & go build -o bin/booknav.exe ./cmd/server
      if ($LASTEXITCODE -ne 0) { throw "go build failed" }
    } finally {
      Pop-Location
    }
  }

  if (-not $NoBuild -or $Rebuild) {
    $webDistSrc = Join-Path $WebDir "dist"
    if (Test-Path $webDistSrc) {
      New-Item -ItemType Directory -Force -Path $WebDist | Out-Null
      Get-ChildItem $WebDist -Force -ErrorAction SilentlyContinue |
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue
      Copy-Item -Recurse -Force (Join-Path $webDistSrc "*") $WebDist
      Write-Host "[start] synced web dist"
    }
  }

  Stop-Port 8080
  Get-Process -Name booknav -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

  $d = Escape-PsSingle $DataDir
  $s = Escape-PsSingle $WebDist
  $e = Escape-PsSingle $ApiExe
  $w = Escape-PsSingle $ServerDir

  # Nested Start-Process so the outer powershell can exit; booknav stays alive outside Job
  $inner = @(
    "`$env:BOOKNAV_ENV='development'"
    "`$env:BOOKNAV_HTTP_ADDR=':8080'"
    "`$env:BOOKNAV_DATA_DIR='$d'"
    "`$env:BOOKNAV_STATIC_DIR='$s'"
    "`$env:BOOKNAV_CORS_ORIGINS='http://localhost:5173,http://127.0.0.1:5173'"
    "Set-Location -LiteralPath '$w'"
    "Start-Process -FilePath '$e' -WorkingDirectory '$w' -WindowStyle Hidden"
  ) -join "; "

  $cmdLine = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -Command `"$inner`""
  $procId = Start-Breakaway -CommandLine $cmdLine -WorkingDirectory $ServerDir

  if (Wait-Http "http://127.0.0.1:8080/healthz") {
    Write-Host "[start] API OK  http://127.0.0.1:8080  (wmi pid $procId)"
  } else {
    Write-Host "[start] API not healthy yet"
  }
  return @{ kind = "api"; wmiPid = $procId }
}

function Start-Web {
  Stop-Port 5173
  if (-not (Test-Path (Join-Path $WebDir "node_modules"))) {
    Write-Host "[start] npm install..."
    Push-Location $WebDir
    npm install
    Pop-Location
  }

  $w = Escape-PsSingle $WebDir
  # Use npm.cmd run dev so node_modules/.bin is resolved
  $inner = @(
    "Set-Location -LiteralPath '$w'"
    "Start-Process -FilePath 'npm.cmd' -ArgumentList 'run','dev','--','--host','127.0.0.1','--port','5173' -WorkingDirectory '$w' -WindowStyle Hidden"
  ) -join "; "

  $cmdLine = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -Command `"$inner`""
  $procId = Start-Breakaway -CommandLine $cmdLine -WorkingDirectory $WebDir

  if (Wait-Http "http://127.0.0.1:5173/") {
    Write-Host "[start] Vite OK http://127.0.0.1:5173  (wmi pid $procId)"
  } else {
    Write-Host "[start] Vite starting..."
  }
  return @{ kind = "web"; wmiPid = $procId }
}

$state = @{
  startedAt = (Get-Date).ToString("o")
  method    = "Win32_Process.Create+Start-Process"
  api       = $null
  web       = $null
}
if (-not $WebOnly) { $state.api = Start-Api }
if (-not $ApiOnly) { $state.web = Start-Web }

$state | ConvertTo-Json -Depth 6 | Set-Content -Path $PidFile -Encoding UTF8

Write-Host ""
Write-Host "==== BookNav running (job-breakaway) ===="
Write-Host "  Frontend: http://127.0.0.1:5173"
Write-Host "  API:      http://127.0.0.1:8080"
Write-Host "  Logs:     $LogDir"
Write-Host "  Stop:     .\scripts\stop.ps1"
Write-Host "  Status:   .\scripts\status.ps1"
Write-Host "  Autostart:.\scripts\install-autostart.ps1"
Write-Host "========================================="
