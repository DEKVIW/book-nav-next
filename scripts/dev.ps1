# BookNav 本地开发入口（推荐）
# 会以「分离进程」启动 API + Vite，关闭本窗口也不会杀服务
# 用法：
#   .\scripts\dev.ps1
#   .\scripts\dev.ps1 -Rebuild
# 停止：
#   .\scripts\stop.ps1

param([switch]$Rebuild)

$ErrorActionPreference = "Stop"
& "$PSScriptRoot\start.ps1" @PSBoundParameters
& "$PSScriptRoot\status.ps1"
