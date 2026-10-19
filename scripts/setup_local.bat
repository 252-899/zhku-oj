@echo off
REM ZK Code Arena 本地开发环境配置脚本 (Windows)

echo ========================================
echo ZK Code Arena 本地开发环境配置
echo ========================================
echo.

REM 检查 Docker 是否运行
docker version >nul 2>&1
if errorlevel 1 (
    echo [错误] Docker 未运行，请先启动 Docker Desktop
    pause
    exit /b 1
)

echo [1/5] 创建数据目录...
if not exist "data\mongo" mkdir data\mongo
if not exist "data\redis" mkdir data\redis
if not exist "data\judge" mkdir data\judge
if not exist "logs" mkdir logs
echo ✓ 数据目录创建完成

echo.
echo [2/5] 启动 MongoDB...
docker ps -a | findstr zk-mongo >nul
if errorlevel 1 (
    docker run -d --name zk-mongo -p 27017:27017 -v %cd%\data\mongo:/data/db mongo:7.0
    echo ✓ MongoDB 启动成功
) else (
    docker start zk-mongo >nul 2>&1
    echo ✓ MongoDB 已在运行
)

echo.
echo [3/5] 启动 Redis...
docker ps -a | findstr zk-redis >nul
if errorlevel 1 (
    docker run -d --name zk-redis -p 6379:6379 -v %cd%\data\redis:/data redis:7.2-alpine
    echo ✓ Redis 启动成功
) else (
    docker start zk-redis >nul 2>&1
    echo ✓ Redis 已在运行
)

echo.
echo [4/5] 启动 go-judge 评测服务...
docker ps -a | findstr zk-judge >nul
if errorlevel 1 (
    docker run -d --name zk-judge -p 5050:5050 --privileged criyle/go-judge:latest
    echo ✓ go-judge 启动成功
) else (
    docker start zk-judge >nul 2>&1
    echo ✓ go-judge 已在运行
)

echo.
echo [5/5] 创建本地配置文件...
if not exist "conf\config.local.yaml" (
    (
        echo App:
        echo   Host: "0.0.0.0"
        echo   Port: "8080"
        echo   Mode: "debug"
        echo   Env: "development"
        echo.
        echo Mongo:
        echo   Uri: mongodb://localhost:27017
        echo   DbName: zk_code_arena
        echo.
        echo Judge:
        echo   SandboxURL: "http://localhost:5050"
        echo   Workers: 2
        echo   QueueSize: 50
        echo.
        echo Redis:
        echo   Host: "localhost"
        echo   Port: 6379
        echo   Password: ""
        echo   DB: 0
        echo.
        echo RateLimit:
        echo   Enabled: false
        echo.
        echo Log:
        echo   Level: "debug"
        echo   Format: "text"
        echo   Output: "stdout"
    ) > conf\config.local.yaml
    echo ✓ 本地配置文件创建完成
) else (
    echo ✓ 本地配置文件已存在
)

echo.
echo ========================================
echo 环境配置完成！
echo ========================================
echo.
echo 服务状态:
docker ps --filter "name=zk-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo.
echo 下一步:
echo   1. 运行 'air' 启动应用（热更新）
echo   2. 或运行 'go run cmd/*.go run' 直接启动
echo   3. 访问 http://localhost:8080/health 检查服务
echo.
echo 查看详细文档: LOCAL_SETUP.md
echo ========================================
pause
