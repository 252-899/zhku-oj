#!/bin/bash
# API 功能测试脚本

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "========================================"
echo "ZK Code Arena API 功能测试"
echo "========================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name=$1
    local method=$2
    local url=$3
    local data=$4
    local headers=$5
    
    echo -n "测试 $name ... "
    
    if [ -n "$data" ]; then
        response=$(curl -s -X $method "$url" \
            -H "Content-Type: application/json" \
            $headers \
            -d "$data")
    else
        response=$(curl -s -X $method "$url" $headers)
    fi
    
    if echo "$response" | grep -q "success\|data\|token"; then
        echo -e "${GREEN}✓ 通过${NC}"
        return 0
    else
        echo -e "${RED}✗ 失败${NC}"
        echo "响应: $response"
        return 1
    fi
}

# 1. 健康检查
echo "[1/6] 健康检查"
test_endpoint "健康检查" "GET" "$BASE_URL/health"
echo ""

# 2. 用户注册
echo "[2/6] 用户管理"
STUDENT_ID="test$(date +%s)"
test_endpoint "用户注册" "POST" "$API_URL/user/" \
    "{\"student_id\":\"$STUDENT_ID\",\"password\":\"test123456\",\"real_name\":\"测试用户\",\"email\":\"test@zk.edu.cn\"}"

# 3. 用户登录
echo -n "用户登录 ... "
LOGIN_RESPONSE=$(curl -s -X POST "$API_URL/user/login" \
    -H "Content-Type: application/json" \
    -d "{\"student_id\":\"$STUDENT_ID\",\"password\":\"test123456\"}")

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo -e "${GREEN}✓ 通过${NC}"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
    echo "Token: ${TOKEN:0:20}..."
else
    echo -e "${RED}✗ 失败${NC}"
    echo "响应: $LOGIN_RESPONSE"
fi
echo ""

# 4. 题目管理
echo "[3/6] 题目管理"
test_endpoint "获取题目列表" "GET" "$API_URL/problem/?page=1&page_size=10"

if [ -n "$TOKEN" ]; then
    test_endpoint "创建题目" "POST" "$API_URL/problem/" \
        "{\"title\":\"测试题目\",\"description\":\"这是一个测试题目\",\"difficulty\":\"easy\",\"time_limit\":1000,\"memory_limit\":256}" \
        "-H \"Authorization: Bearer $TOKEN\""
fi
echo ""

# 5. 提交管理
echo "[4/6] 提交管理"
if [ -n "$TOKEN" ]; then
    test_endpoint "获取提交列表" "GET" "$API_URL/submit/?page=1&page_size=10" \
        "" \
        "-H \"Authorization: Bearer $TOKEN\""
fi
echo ""

# 6. 数据库连接测试
echo "[5/6] 数据库连接"
echo -n "MongoDB 连接 ... "
if docker exec zk-mongo mongosh --quiet --eval "db.version()" &> /dev/null; then
    echo -e "${GREEN}✓ 通过${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
fi

echo -n "Redis 连接 ... "
if docker exec zk-redis redis-cli PING | grep -q "PONG"; then
    echo -e "${GREEN}✓ 通过${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
fi
echo ""

# 7. go-judge 测试
echo "[6/6] go-judge 评测服务"
echo -n "go-judge 连接 ... "
JUDGE_RESPONSE=$(curl -s -X POST http://localhost:5050/run \
    -H "Content-Type: application/json" \
    -d '{
        "cmd": [{
            "args": ["/usr/bin/python3", "-c", "print(\"Hello\")"],
            "env": ["PATH=/usr/bin:/bin"],
            "files": [{"content": ""}],
            "cpuLimit": 1000000000,
            "memoryLimit": 104857600,
            "procLimit": 50
        }]
    }')

if echo "$JUDGE_RESPONSE" | grep -q "Hello\|exitStatus"; then
    echo -e "${GREEN}✓ 通过${NC}"
else
    echo -e "${RED}✗ 失败${NC}"
    echo "响应: $JUDGE_RESPONSE"
fi
echo ""

echo "========================================"
echo "测试完成！"
echo "========================================"
echo ""
echo "如果所有测试都通过，说明环境配置正确"
echo "如果有测试失败，请检查："
echo "  1. Docker 容器是否正常运行"
echo "  2. 配置文件是否正确"
echo "  3. 应用是否正常启动"
echo ""
