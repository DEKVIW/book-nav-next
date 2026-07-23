# 查看 BookNav 本地服务状态
$ErrorActionPreference = "Continue"

function Port-Info([int]$Port) {
  $c = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
  if (-not $c) { return "DOWN" }
  $p = Get-Process -Id $c.OwningProcess -ErrorAction SilentlyContinue
  return "UP pid=$($c.OwningProcess) name=$($p.ProcessName)"
}

function Http-Ok([string]$Url) {
  try {
    $r = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec 2
    return "HTTP $($r.StatusCode)"
  } catch {
    return "HTTP fail"
  }
}

Write-Host "8080  : $(Port-Info 8080)  | $(Http-Ok 'http://127.0.0.1:8080/healthz')"
Write-Host "5173  : $(Port-Info 5173)  | $(Http-Ok 'http://127.0.0.1:5173/')"
Write-Host "proxy : $(Http-Ok 'http://127.0.0.1:5173/api/v1/portal/home')"

$Root = Split-Path -Parent $PSScriptRoot
$LogDir = Join-Path $Root ".dev\logs"
if (Test-Path $LogDir) {
  Write-Host "logs  : $LogDir"
  Get-ChildItem $LogDir -Filter "*.log" -ErrorAction SilentlyContinue | ForEach-Object {
    Write-Host "  $($_.Name)  $([math]::Round($_.Length/1KB,1)) KB  $($_.LastWriteTime)"
  }
}
