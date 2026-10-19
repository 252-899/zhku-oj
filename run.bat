@echo off
REM 简单启动脚本 - 直接在 WSL2 中运行，无需构建
REM 项目路径: G:\code-oj\ZKCodeArenaServer

echo ========================================
echo ZK Code Arena 简单启动
echo ========================================
echo.

REM 检查是否在项目目录
if not exist "go.mod" (
    echo [错误] 请在项目根目录运行此脚本
    echo 当前目录: %cd%
    pause
    exit /b 1
)

echo [提示] 将在 WSL2 中直接运行应用（无需构建）
echo.

REM 检查 WSL2
wsl --list --running >nul 2>&1
if errorlevel 1 (
    echo [错误] WSL2 未运行
    echo 正在启动 WSL2...
    wsl echo WSL2 已启动
)

echo [1/2] 检查 Docker 服务...
wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && docker ps | grep -E 'zk-mongo|zk-redis' > /dev/null 2>&1"
if errorlevel 1 (
    echo [提示] Docker 服务未运行，正在启动...
    wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && docker start zk-mongo zk-redis 2>/dev/null || ./scripts/setup_wsl2.sh"
) else (
    echo ✓ Docker 服务已运行
)

echo.
echo [2/2] 启动应用...
echo.
echo ========================================
echo 应用正在启动...
echo 访问: http://localhost:8080/health
echo 文档: http://localhost:8080/swagger/index.html
echo 按 Ctrl+C 停止应用
echo ========================================
echo.

REM 在 WSL2 中直接运行，不构建
wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && go run cmd/*.go run"

echo.
echo 应用已停止
pause
