# BookNav 看门狗：检测 8080/5173，挂了就拉起
# 推荐在「独立 PowerShell 窗口」里长期运行（不要只在 AI 会话里跑）
#   .\scripts\watchdog.ps1
# 注册为开机任务（可选）：
#   .\scripts\watchdog.ps1 -RegisterTask
# 取消：
#   .\scripts\watchdog.ps1 -UnregisterTask

param(
  [int]$IntervalSec = 20,
  [switch]$RegisterTask,
  [switch]$UnregisterTask,
  [switch]$Once
)

$ErrorActionPreference = "Continue"
$Root = Split-Path -Parent $PSScriptRoot
$StartScript = Join-Path $PSScriptRoot "start.ps1"
$TaskName = "BookNav-Watchdog"

if ($UnregisterTask) {
  Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction SilentlyContinue
  Write-Host "unregistered task $TaskName"
  exit 0
}

if ($RegisterTask) {
  $action = New-ScheduledTaskAction -Execute "powershell.exe" `
    -Argument "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$PSCommandPath`""
  $trigger = New-ScheduledTaskTrigger -AtLogOn
  $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 3 -RestartInterval (New-TimeSpan -Minutes 1)
  Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Settings $settings -Force | Out-Null
  Write-Host "registered logon task: $TaskName"
  Write-Host "it will keep BookNav up after you log in"
  exit 0
}

function Port-Up([int]$Port) {
  $c = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  return $null -ne $c
}

function Health-Api {
  try {
    $r = Invoke-WebRequest -Uri "http://127.0.0.1:8080/healthz" -UseBasicParsing -TimeoutSec 2
    return $r.StatusCode -eq 200
  } catch {
    return $false
  }
}

Write-Host "[watchdog] interval ${IntervalSec}s  root=$Root"
Write-Host "[watchdog] Ctrl+C to stop this loop (services keep running)"

while ($true) {
  $needApi = -not (Health-Api)
  $needWeb = -not (Port-Up 5173)
  if ($needApi -or $needWeb) {
    $ts = Get-Date -Format "HH:mm:ss"
    if ($needApi) { Write-Host "[$ts] API down -> restart" }
    if ($needWeb) { Write-Host "[$ts] Vite down -> restart" }
    $args = @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $StartScript, "-NoBuild")
    if ($needApi -and -not $needWeb) { $args += "-ApiOnly" }
    if ($needWeb -and -not $needApi) { $args += "-WebOnly" }
    Start-Process -FilePath "powershell.exe" -ArgumentList $args -WindowStyle Hidden | Out-Null
  }
  if ($Once) { break }
  Start-Sleep -Seconds $IntervalSec
}
