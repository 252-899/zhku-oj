#!/bin/bash
# 简单启动脚本 - 直接运行，无需构建
# 在 WSL2 中使用

set -e

PROJECT_DIR="/mnt/g/code-oj/ZKCodeArenaServer"

echo "========================================"
echo "ZK Code Arena 简单启动"
echo "========================================"
echo ""

# 进入项目目录
cd "$PROJECT_DIR"

# 检查 Docker 服务
echo "[1/2] 检查 Docker 服务..."
if docker ps | grep -E 'zk-mongo|zk-redis' > /dev/null 2>&1; then
    echo "✓ Docker 服务已运行"
else
    echo "[提示] Docker 服务未运行，正在启动..."
    docker start zk-mongo zk-redis 2>/dev/null || {
        echo "[提示] 容器不存在，正在创建..."
        ./scripts/setup_wsl2.sh
    }
fi

echo ""
echo "[2/2] 启动应用..."
echo ""
echo "========================================"
echo "应用正在启动..."
echo "访问: http://localhost:8080/health"
echo "文档: http://localhost:8080/swagger/index.html"
echo "按 Ctrl+C 停止应用"
echo "========================================"
echo ""

# 直接运行，不构建
go run cmd/*.go run
