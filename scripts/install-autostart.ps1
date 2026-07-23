# 注册 Windows 登录自启 + 看门狗（彻底解决「进程被会话杀掉」）
# 需要当前用户权限即可（不要求管理员）
#
#   .\scripts\install-autostart.ps1
#   .\scripts\install-autostart.ps1 -Remove

param([switch]$Remove)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$StartScript = Join-Path $PSScriptRoot "start.ps1"
$WatchScript = Join-Path $PSScriptRoot "watchdog.ps1"
$TaskStart = "BookNav-Start"
$TaskWatch = "BookNav-Watchdog"

if ($Remove) {
  Unregister-ScheduledTask -TaskName $TaskStart -Confirm:$false -ErrorAction SilentlyContinue
  Unregister-ScheduledTask -TaskName $TaskWatch -Confirm:$false -ErrorAction SilentlyContinue
  Write-Host "removed tasks: $TaskStart, $TaskWatch"
  exit 0
}

# 登录后启动服务
$ps = "powershell.exe"
$startArgs = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$StartScript`" -NoBuild"
$action1 = New-ScheduledTaskAction -Execute $ps -Argument $startArgs
$trigger1 = New-ScheduledTaskTrigger -AtLogOn
$settings = New-ScheduledTaskSettingsSet `
  -AllowStartIfOnBatteries `
  -DontStopIfGoingOnBatteries `
  -StartWhenAvailable `
  -RestartCount 5 `
  -RestartInterval (New-TimeSpan -Minutes 1) `
  -ExecutionTimeLimit (New-TimeSpan -Days 0) # 无超时

Register-ScheduledTask -TaskName $TaskStart -Action $action1 -Trigger $trigger1 `
  -Settings $settings -Description "BookNav API+Vite at logon" -Force | Out-Null

# 看门狗（登录后延迟 1 分钟开始循环）
$watchArgs = "-NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$WatchScript`""
$action2 = New-ScheduledTaskAction -Execute $ps -Argument $watchArgs
$trigger2 = New-ScheduledTaskTrigger -AtLogOn
# 延迟一点，等 start 先跑
$trigger2.Delay = "PT1M"
Register-ScheduledTask -TaskName $TaskWatch -Action $action2 -Trigger $trigger2 `
  -Settings $settings -Description "BookNav watchdog: restart if down" -Force | Out-Null

Write-Host "OK registered:"
Write-Host "  $TaskStart  — 登录时启动 BookNav"
Write-Host "  $TaskWatch  — 登录后看门狗（挂了自动拉起）"
Write-Host ""
Write-Host "立即启动一次:"
Write-Host "  .\scripts\start.ps1"
Write-Host "取消自启:"
Write-Host "  .\scripts\install-autostart.ps1 -Remove"
