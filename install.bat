@echo off
chcp 65001 >nul

set "current_dir=%~dp0"
set "current_dir=%current_dir:~0,-1%"

if not exist "%current_dir%\dredger.exe" (
    echo 未找到 dredger.exe
    pause
    exit /b 1
)

if not exist "%current_dir%\logs" (
    md "%current_dir%\logs"
)

:: 使用绝对路径调用 nssm.exe
"%current_dir%\nssm.exe" install DredgerService "%current_dir%\dredger.exe"
"%current_dir%\nssm.exe" set DredgerService AppDirectory "%current_dir%"
"%current_dir%\nssm.exe" set DredgerService Start SERVICE_AUTO_START
"%current_dir%\nssm.exe" set DredgerService AppStdout "%current_dir%\logs\service.log"
"%current_dir%\nssm.exe" set DredgerService AppStderr "%current_dir%\logs\error.log"
"%current_dir%\nssm.exe" start DredgerService

echo 服务已注册并启动，日志输出到 logs\service.log 和 logs\error.log

echo.
echo 按任意键退出...
pause >nul