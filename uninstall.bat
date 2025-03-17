@echo off
chcp 65001 >nul

set "current_dir=%~dp0"
set "current_dir=%current_dir:~0,-1%"

echo echo 正在删除服务......
:: 停止并删除服务
"%current_dir%\nssm.exe" status DredgerService >nul 2>&1
if %errorlevel% equ 0 (
    echo 正在停止服务...
    "%current_dir%\nssm.exe" stop DredgerService >nul 2>&1
    timeout /t 2 >nul

    echo 正在删除服务...
    "%current_dir%\nssm.exe" remove DredgerService confirm >nul 2>&1
    if %errorlevel% equ 0 (
        echo 服务已成功卸载
    ) else (
        echo 错误：服务删除失败，请手动检查
    )
) else (
    echo 服务不存在，无需操作
)

echo.
echo 按任意键退出...
pause >nul