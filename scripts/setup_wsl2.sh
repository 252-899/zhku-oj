#!/bin/bash
# WSL2 环境配置脚本

set -e

echo "========================================"
echo "WSL2 环境配置检查"
echo "========================================"
echo ""

# 检查是否在 WSL2 中运行
if grep -qi microsoft /proc/version; then
    echo "✓ 检测到 WSL2 环境"
else
    echo "⚠ 警告: 似乎不在 WSL2 环境中"
fi
echo ""

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "[错误] Docker 未安装"
    echo "请在 WSL2 中安装 Docker 或使用 Docker Desktop for Windows"
    exit 1
fi

# 检查判题服务
echo "[1/4] 检查判题服务..."
if curl -s http://localhost:5050/version > /dev/null 2>&1; then
    echo "✓ 判题服务已运行在 localhost:5050"
    JUDGE_EXISTS=true
    JUDGE_INFO=$(curl -s http://localhost:5050/version 2>&1 | head -n 1)
    echo "  版本信息: $JUDGE_INFO"
elif docker ps | grep -q judge; then
    echo "✓ 检测到 Docker 中的判题服务"
    JUDGE_EXISTS=true
else
    echo "⚠ 未检测到判题服务，将创建新的判题服务"
    JUDGE_EXISTS=false
fi

# 创建数据目录
mkdir -p data/mongo data/redis logs

# 启动 MongoDB
echo ""
echo "[2/4] 配置 MongoDB..."
if docker ps | grep -q zk-mongo; then
    echo "✓ MongoDB 已运行"
elif docker ps -a | grep -q zk-mongo; then
    docker start zk-mongo > /dev/null
    sleep 2
    echo "✓ MongoDB 已启动"
else
    echo "  启动 MongoDB 容器..."
    docker run -d --name zk-mongo \
        -p 27017:27017 \
        -v "$(pwd)/data/mongo:/data/db" \
        mongo:7.0 > /dev/null
    sleep 3
    echo "✓ MongoDB 启动成功"
fi

# 测试 MongoDB 连接
if docker exec zk-mongo mongosh --quiet --eval "db.version()" > /dev/null 2>&1; then
    MONGO_VERSION=$(docker exec zk-mongo mongosh --quiet --eval "db.version()" 2>/dev/null)
    echo "  版本: $MONGO_VERSION"
fi

# 启动 Redis
echo ""
echo "[3/4] 配置 Redis..."
if docker ps | grep -q zk-redis; then
    echo "✓ Redis 已运行"
elif docker ps -a | grep -q zk-redis; then
    docker start zk-redis > /dev/null
    sleep 2
    echo "✓ Redis 已启动"
else
    echo "  启动 Redis 容器..."
    docker run -d --name zk-redis \
        -p 6379:6379 \
        -v "$(pwd)/data/redis:/data" \
        redis:7.2-alpine > /dev/null
    sleep 2
    echo "✓ Redis 启动成功"
fi

# 测试 Redis 连接
if docker exec zk-redis redis-cli PING 2>/dev/null | grep -q PONG; then
    echo "  状态: 正常"
fi

# 判题服务
echo ""
echo "[4/4] 判题服务配置..."
if [ "$JUDGE_EXISTS" = true ]; then
    echo "✓ 使用现有判题服务 (localhost:5050)"
else
    if docker ps -a | grep -q zk-judge; then
        docker start zk-judge > /dev/null
        sleep 2
        echo "✓ 判题服务已启动"
    else
        echo "  启动判题服务容器..."
        docker run -d --name zk-judge \
            -p 5050:5050 \
            --privileged \
            criyle/go-judge:latest > /dev/null
        sleep 3
        echo "✓ 判题服务启动成功"
    fi
fi

# 创建配置文件
echo ""
echo "[5/5] 创建配置文件..."
if [ ! -f "conf/config.wsl2.yaml" ]; then
    cat > conf/config.wsl2.yaml << 'EOF'
App:
  Host: "0.0.0.0"
  Port: "8080"
  Mode: "debug"
  Env: "development"

Mongo:
  Uri: mongodb://localhost:27017
  DbName: zk_code_arena

Judge:
  SandboxURL: "http://localhost:5050"
  Workers: 2
  QueueSize: 50

Redis:
  Host: "localhost"
  Port: 6379
  Password: ""
  DB: 0

RateLimit:
  Enabled: false

Log:
  Level: "debug"
  Format: "text"
  Output: "stdout"
EOF
    echo "✓ 配置文件创建完成: conf/config.wsl2.yaml"
else
    echo "✓ 配置文件已存在"
fi

echo ""
echo "========================================"
echo "环境配置完成！"
echo "========================================"
echo ""
echo "服务状态:"
echo "┌─────────────┬──────────────────────────────────────┐"
printf "│ %-11s │ %-36s │\n" "服务" "状态"
echo "├─────────────┼──────────────────────────────────────┤"

# MongoDB 状态
if docker ps --filter 'name=zk-mongo' --format '{{.Status}}' | grep -q Up; then
    MONGO_STATUS="✓ 运行中"
else
    MONGO_STATUS="✗ 未运行"
fi
printf "│ %-11s │ %-36s │\n" "MongoDB" "$MONGO_STATUS"

# Redis 状态
if docker ps --filter 'name=zk-redis' --format '{{.Status}}' | grep -q Up; then
    REDIS_STATUS="✓ 运行中"
else
    REDIS_STATUS="✗ 未运行"
fi
printf "│ %-11s │ %-36s │\n" "Redis" "$REDIS_STATUS"

# go-judge 状态
if curl -s http://localhost:5050/version > /dev/null 2>&1; then
    JUDGE_STATUS="✓ 运行中 (localhost:5050)"
else
    JUDGE_STATUS="✗ 未运行"
fi
printf "│ %-11s │ %-36s │\n" "go-judge" "$JUDGE_STATUS"

echo "└─────────────┴──────────────────────────────────────┘"
echo ""

# 网络测试
echo "网络连接测试:"
echo "  MongoDB:  $(docker exec zk-mongo mongosh --quiet --eval "db.version()" 2>/dev/null || echo '连接失败')"
echo "  Redis:    $(docker exec zk-redis redis-cli PING 2>/dev/null || echo '连接失败')"
echo "  go-judge: $(curl -s http://localhost:5050/version 2>&1 | head -n 1 || echo '连接失败')"
echo ""

echo "下一步:"
echo "  1. 在 WSL2 中运行:"
echo "     go run cmd/*.go run"
echo ""
echo "  2. 或使用 Air 热更新:"
echo "     air"
echo ""
echo "  3. 在 Windows 浏览器访问:"
echo "     http://localhost:8080/health"
echo ""
echo "  4. 查看 API 文档:"
echo "     http://localhost:8080/swagger/index.html"
echo ""
echo "配置文件: conf/config.wsl2.yaml"
echo "详细文档: WSL2_SETUP.md"
echo "========================================"
