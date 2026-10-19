#!/bin/bash
# ZK Code Arena 本地开发环境配置脚本 (Linux/Mac)

set -e

echo "========================================"
echo "ZK Code Arena 本地开发环境配置"
echo "========================================"
echo ""

# 检查 Docker 是否运行
if ! docker version &> /dev/null; then
    echo "[错误] Docker 未运行，请先启动 Docker"
    exit 1
fi

echo "[1/5] 创建数据目录..."
mkdir -p data/mongo data/redis data/judge logs
echo "✓ 数据目录创建完成"

echo ""
echo "[2/5] 启动 MongoDB..."
if docker ps -a | grep -q zk-mongo; then
    docker start zk-mongo &> /dev/null || true
    echo "✓ MongoDB 已在运行"
else
    docker run -d \
        --name zk-mongo \
        -p 27017:27017 \
        -v "$(pwd)/data/mongo:/data/db" \
        mongo:7.0
    echo "✓ MongoDB 启动成功"
fi

echo ""
echo "[3/5] 启动 Redis..."
if docker ps -a | grep -q zk-redis; then
    docker start zk-redis &> /dev/null || true
    echo "✓ Redis 已在运行"
else
    docker run -d \
        --name zk-redis \
        -p 6379:6379 \
        -v "$(pwd)/data/redis:/data" \
        redis:7.2-alpine
    echo "✓ Redis 启动成功"
fi

echo ""
echo "[4/5] 启动 go-judge 评测服务..."
if docker ps -a | grep -q zk-judge; then
    docker start zk-judge &> /dev/null || true
    echo "✓ go-judge 已在运行"
else
    docker run -d \
        --name zk-judge \
        -p 5050:5050 \
        --privileged \
        criyle/go-judge:latest
    echo "✓ go-judge 启动成功"
fi

echo ""
echo "[5/5] 创建本地配置文件..."
if [ ! -f "conf/config.local.yaml" ]; then
    cat > conf/config.local.yaml << 'EOF'
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
    echo "✓ 本地配置文件创建完成"
else
    echo "✓ 本地配置文件已存在"
fi

echo ""
echo "========================================"
echo "环境配置完成！"
echo "========================================"
echo ""
echo "服务状态:"
docker ps --filter "name=zk-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo "下一步:"
echo "  1. 运行 'air' 启动应用（热更新）"
echo "  2. 或运行 'go run cmd/*.go run' 直接启动"
echo "  3. 访问 http://localhost:8080/health 检查服务"
echo ""
echo "查看详细文档: LOCAL_SETUP.md"
echo "========================================"
