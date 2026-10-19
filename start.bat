@echo off
REM 一键启动脚本 - 在 WSL2 中配置并启动项目
REM 项目路径: G:\code-oj\ZKCodeArenaServer

echo ========================================
echo ZK Code Arena 一键启动
echo ========================================
echo.

REM 检查是否在项目目录
if not exist "go.mod" (
    echo [错误] 请在项目根目录运行此脚本
    echo 当前目录: %cd%
    echo 期望目录: G:\code-oj\ZKCodeArenaServer
    pause
    exit /b 1
)

echo [1/3] 检查 WSL2...
wsl --list --running >nul 2>&1
if errorlevel 1 (
    echo [错误] WSL2 未运行，请先启动 WSL2
    pause
    exit /b 1
)
echo ✓ WSL2 已运行

echo.
echo [2/3] 在 WSL2 中配置环境...
echo 提示: 这将在 WSL2 中启动 MongoDB、Redis 等服务
echo.

REM 在 WSL2 中运行配置脚本
wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && chmod +x scripts/setup_wsl2.sh && ./scripts/setup_wsl2.sh"

if errorlevel 1 (
    echo.
    echo [错误] 环境配置失败
    echo 请检查 WSL2 中的错误信息
    pause
    exit /b 1
)

echo.
echo [3/3] 选择启动方式...
echo.
echo 请选择在哪里运行应用:
echo   1. 在 WSL2 中运行 (推荐)
echo   2. 在 Windows 中运行
echo   3. 退出
echo.
set /p choice="请输入选项 (1-3): "

if "%choice%"=="1" (
    echo.
    echo 正在 WSL2 中启动应用...
    echo 提示: 按 Ctrl+C 可以停止应用
    echo.
    wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && go run cmd/*.go run"
) else if "%choice%"=="2" (
    echo.
    echo 正在 Windows 中启动应用...
    echo 提示: 按 Ctrl+C 可以停止应用
    echo.
    go run cmd/*.go run
) else (
    echo.
    echo 已取消启动
    echo.
    echo 手动启动方式:
    echo   WSL2:    wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && go run cmd/*.go run"
    echo   Windows: go run cmd/*.go run
    echo.
)

pause
