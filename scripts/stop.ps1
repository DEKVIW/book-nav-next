# 停止 BookNav 本地 API / Vite（按端口杀进程）
# 用法：.\scripts\stop.ps1

param(
  [switch]$ApiOnly,
  [switch]$WebOnly
)

$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$DevDir = Join-Path $Root ".dev"

function Stop-Port([int]$Port) {
  $conns = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  $killed = @()
  foreach ($c in $conns) {
    $pid = $c.OwningProcess
    if ($pid -gt 0 -and $killed -notcontains $pid) {
      try {
        $proc = Get-Process -Id $pid -ErrorAction SilentlyContinue
        Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue
        Write-Host "[stop] :$Port pid $pid ($($proc.ProcessName))"
        $killed += $pid
      } catch {
        Write-Host "[stop] failed to kill $pid : $_"
      }
    }
  }
  if (-not $killed.Count) {
    Write-Host "[stop] nothing listening on :$Port"
  }
}

# 也清掉我们启动的 launcher powershell（若仍在跑）
$pidFile = Join-Path $DevDir "pids.json"
if (Test-Path $pidFile) {
  try {
    $state = Get-Content $pidFile -Raw | ConvertFrom-Json
    foreach ($key in @("api", "web")) {
      $item = $state.$key
      if ($item -and $item.launcherPid) {
        Stop-Process -Id $item.launcherPid -Force -ErrorAction SilentlyContinue
      }
    }
  } catch {}
}

if (-not $WebOnly) { Stop-Port 8080 }
if (-not $ApiOnly) { Stop-Port 5173 }

# booknav.exe 兜底
Get-Process -Name "booknav" -ErrorAction SilentlyContinue | ForEach-Object {
  Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
  Write-Host "[stop] booknav.exe pid $($_.Id)"
}

Write-Host "[stop] done"
