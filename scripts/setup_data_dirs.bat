@echo off
chcp 65001 >nul
echo 🚀 开始创建数据目录...
echo.

REM 创建主数据目录
if not exist "data" mkdir data

REM 创建各服务的数据目录
if not exist "data\mongo" mkdir data\mongo
if not exist "data\redis" mkdir data\redis
if not exist "data\judge" mkdir data\judge

REM 创建日志目录
if not exist "logs" mkdir logs
if not exist "logs\nginx" mkdir logs\nginx

echo ✅ 数据目录创建完成！
echo.
echo 📁 创建的目录结构：
echo ├── data\
echo │   ├── mongo\     # MongoDB 数据存储
echo │   ├── redis\     # Redis 数据存储
echo │   └── judge\     # 判题服务数据
echo └── logs\
echo     └── nginx\     # Nginx 日志
echo.
echo 🔧 使用方法：
echo 1. 运行此脚本: scripts\setup_data_dirs.bat
echo 2. 启动服务: docker-compose up -d
echo.
echo 💡 提示：这些目录会被挂载到 Docker 容器中，数据将持久化保存在主机上
pause
