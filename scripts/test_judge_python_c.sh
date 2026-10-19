#!/bin/bash
# 在服务器上测试判题环境是否支持 Python 和 C
# 用法：在项目目录下执行 bash scripts/test_judge_python_c.sh
# 前提：judge 容器已启动且映射了 5050 端口（docker-compose.dev.yml 中 judge 服务）

set -e
JUDGE_URL="${JUDGE_URL:-http://localhost:5050}"
CONTAINER_NAME="${JUDGE_CONTAINER:-zk-arena-judge-dev}"

echo "=========================================="
echo "1. 检查判题容器内是否有 gcc 和 python3"
echo "=========================================="
if ! docker exec "$CONTAINER_NAME" which gcc >/dev/null 2>&1; then
  echo "[失败] 容器内没有 gcc，C 语言环境不可用。请使用多语言镜像 zhku-oj-judge:multilang"
  exit 1
fi
if ! docker exec "$CONTAINER_NAME" which python3 >/dev/null 2>&1; then
  echo "[失败] 容器内没有 python3，Python 环境不可用。请使用多语言镜像 zhku-oj-judge:multilang"
  exit 1
fi
echo "[通过] gcc: $(docker exec "$CONTAINER_NAME" gcc --version | head -1)"
echo "[通过] python3: $(docker exec "$CONTAINER_NAME" python3 --version)"
echo ""

echo "=========================================="
echo "2. 测试 Python：运行 print(1+1)，期望输出 2"
echo "=========================================="
PY_RESP=$(curl -s -X POST "$JUDGE_URL/run" -H "Content-Type: application/json" -d '{
  "cmd": [{
    "args": ["/usr/bin/python3", "-u", "main.py"],
    "env": ["PATH=/usr/bin:/bin", "PYTHONPATH=/w"],
    "cpuLimit": 2000000000,
    "memoryLimit": 268435456,
    "procLimit": 1,
    "copyIn": { "main.py": { "content": "print(1+1)" } },
    "copyOut": ["stdout", "stderr"],
    "files": [
      { "content": "" },
      { "name": "stdout", "max": 10240 },
      { "name": "stderr", "max": 10240 }
    ]
  }]
}')
PY_STATUS=$(echo "$PY_RESP" | grep -o '"status":"[^"]*"' | head -1)
PY_STDOUT=$(echo "$PY_RESP" | grep -o '"stdout":"[^"]*"' | sed 's/"stdout":"\(.*\)"/\1/' | sed 's/\\n/\n/g')
if echo "$PY_RESP" | grep -q '"status":"Accepted"' && echo "$PY_STDOUT" | grep -q '2'; then
  echo "[通过] Python 运行成功，stdout: $PY_STDOUT"
else
  echo "[失败] Python 测试未通过。响应: $PY_RESP"
  exit 1
fi
echo ""

echo "=========================================="
echo "3. 测试 C：编译并运行 printf hello，期望输出 hello"
echo "=========================================="
# 3.1 编译（C 源码内联，避免依赖 jq）
COMPILE_RESP=$(curl -s -X POST "$JUDGE_URL/run" -H "Content-Type: application/json" -d '{
  "cmd": [{
    "args": ["/usr/bin/gcc", "-std=c11", "-O2", "-Wall", "main.c", "-o", "main", "-lm"],
    "env": ["PATH=/usr/bin:/bin"],
    "cpuLimit": 10000000000,
    "memoryLimit": 268435456,
    "procLimit": 50,
    "copyIn": { "main.c": { "content": "#include <stdio.h>\nint main() { printf(\"hello\"); return 0; }" } },
    "copyOut": ["stdout", "stderr"],
    "copyOutCached": ["main"],
    "files": [
      { "content": "" },
      { "name": "stdout", "max": 10240 },
      { "name": "stderr", "max": 10240 }
    ]
  }]
}')
if ! echo "$COMPILE_RESP" | grep -q '"status":"Accepted"'; then
  echo "[失败] C 编译未通过。响应: $COMPILE_RESP"
  exit 1
fi
# 从响应中取出 main 的 fileId（需要 jq；若无 jq 则用简单 grep）
if command -v jq >/dev/null 2>&1; then
  FILE_ID=$(echo "$COMPILE_RESP" | jq -r '.[0].fileIds.main // empty')
else
  FILE_ID=$(echo "$COMPILE_RESP" | grep -o '"main":"[^"]*"' | cut -d'"' -f4)
fi
if [ -z "$FILE_ID" ]; then
  echo "[失败] 无法从编译响应中获取 main 的 fileId。请安装 jq 或检查响应: $COMPILE_RESP"
  exit 1
fi
# 3.2 运行
RUN_RESP=$(curl -s -X POST "$JUDGE_URL/run" -H "Content-Type: application/json" -d "{
  \"cmd\": [{
    \"args\": [\"./main\"],
    \"env\": [\"PATH=/usr/bin:/bin\"],
    \"cpuLimit\": 1000000000,
    \"memoryLimit\": 134217728,
    \"procLimit\": 1,
    \"copyIn\": { \"main\": { \"fileId\": \"$FILE_ID\" } },
    \"copyOut\": [\"stdout\", \"stderr\"],
    \"files\": [
      { \"content\": \"\" },
      { \"name\": \"stdout\", \"max\": 10240 },
      { \"name\": \"stderr\", \"max\": 10240 }
    ]
  }]
}")
if echo "$RUN_RESP" | grep -q '"status":"Accepted"' && echo "$RUN_RESP" | grep -q 'hello'; then
  echo "[通过] C 编译并运行成功，输出包含 hello"
else
  echo "[失败] C 运行未通过。响应: $RUN_RESP"
  exit 1
fi
echo ""
echo "=========================================="
echo "全部通过：Python 和 C 判题环境可用"
echo "=========================================="
