# 本地开发环境配置指南

## 📋 前置要求

- ✅ Go 1.23+
- ✅ Docker Desktop (用于运行 MongoDB、Redis、go-judge)
- ✅ Git

## 🚀 快速启动步骤

### 1. 启动依赖服务（Docker）

```bash
# 创建数据目录
mkdir -p data/mongo data/redis data/judge

# 启动 MongoDB（无密码，开发环境）
docker run -d \
  --name zk-mongo \
  -p 27017:27017 \
  -v ${PWD}/data/mongo:/data/db \
  mongo:7.0

# 启动 Redis（无密码，开发环境）
docker run -d \
  --name zk-redis \
  -p 6379:6379 \
  -v ${PWD}/data/redis:/data \
  redis:7.2-alpine

# 启动 go-judge 评测服务
docker run -d \
  --name zk-judge \
  -p 5050:5050 \
  --privileged \
  criyle/go-judge:latest
```

### 2. 修改配置文件

编辑 `conf/config.yaml`，确保以下配置正确：

```yaml
# 本地开发配置
Mongo:
  Uri: mongodb://localhost:27017  # ✅ 本地 MongoDB
  DbName: zk_code_arena

Judge:
  SandboxURL: "http://localhost:5050"  # ✅ 改为 localhost（不是 judge）
  Workers: 4
  QueueSize: 100

Redis:
  Host: "localhost"  # ✅ 改为 localhost（不是 redis）
  Port: 6379
  Password: ""       # ✅ 开发环境无密码
  DB: 0
  PoolSize: 10

RateLimit:
  Enabled: false     # ✅ 开发环境关闭限流
```

### 3. 安装 Air（热更新工具）

```bash
go install github.com/air-verse/air@v1.49.0
```

### 4. 启动应用

```bash
# 方式一：使用 Air 热更新（推荐）
air

# 方式二：直接运行
go run cmd/*.go run
```

### 5. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health

# 预期响应
# {"data":{"service":"zk-code-arena-server","status":"ok","timestamp":1700000000}}
```

## 🔧 配置文件对比

### Docker Compose 环境 vs 本地开发环境

| 配置项 | Docker Compose | 本地开发 |
|--------|----------------|----------|
| MongoDB URI | `mongodb://mongo:27017` | `mongodb://localhost:27017` |
| Redis Host | `redis` | `localhost` |
| Judge URL | `http://judge:5050` | `http://localhost:5050` |
| Redis Password | `redis123` | `""` (空) |

## 📝 需要修改的配置

创建 `conf/config.local.yaml`（本地开发专用）：

```yaml
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
```

然后修改代码加载配置时优先使用本地配置：

```go
// conf/config.go 中添加
func LoadConfig() {
    // 优先加载本地配置
    if _, err := os.Stat("conf/config.local.yaml"); err == nil {
        viper.SetConfigFile("conf/config.local.yaml")
    } else {
        viper.SetConfigFile("conf/config.yaml")
    }
    // ...
}
```

## 🧪 测试基础功能

### 1. 测试数据库连接

```bash
# 连接 MongoDB
docker exec -it zk-mongo mongosh

# 在 mongosh 中执行
use zk_code_arena
db.users.find()
```

### 2. 测试 Redis 连接

```bash
# 连接 Redis
docker exec -it zk-redis redis-cli

# 测试命令
PING
# 应该返回 PONG
```

### 3. 测试 go-judge

```bash
curl -X POST http://localhost:5050/run \
  -H "Content-Type: application/json" \
  -d '{
    "cmd": [{
      "args": ["/usr/bin/python3", "-c", "print(\"Hello\")"],
      "env": ["PATH=/usr/bin:/bin"],
      "files": [{
        "content": ""
      }],
      "cpuLimit": 1000000000,
      "memoryLimit": 104857600,
      "procLimit": 50
    }]
  }'
```

### 4. 测试 API 接口

```bash
# 1. 注册用户
curl -X POST http://localhost:8080/api/v1/user/ \
  -H "Content-Type: application/json" \
  -d '{
    "student_id": "2021001",
    "password": "test123456",
    "real_name": "测试用户",
    "email": "test@zk.edu.cn"
  }'

# 2. 登录
curl -X POST http://localhost:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "student_id": "2021001",
    "password": "test123456"
  }'

# 保存返回的 token

# 3. 获取题目列表
curl -X GET "http://localhost:8080/api/v1/problem/?page=1&page_size=10"

# 4. 创建题目（需要管理员权限）
curl -X POST http://localhost:8080/api/v1/problem/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "title": "Hello World",
    "description": "输出 Hello World",
    "difficulty": "easy",
    "time_limit": 1000,
    "memory_limit": 256
  }'
```

## 🐛 常见问题

### 1. MongoDB 连接失败

```bash
# 检查 MongoDB 是否运行
docker ps | grep zk-mongo

# 查看日志
docker logs zk-mongo

# 重启 MongoDB
docker restart zk-mongo
```

### 2. go-judge 连接失败

```bash
# 检查 go-judge 状态
docker ps | grep zk-judge

# 测试连接
curl http://localhost:5050/version

# 重启 go-judge
docker restart zk-judge
```

### 3. 端口被占用

```bash
# Windows
netstat -ano | findstr :8080
taskkill /PID <PID> /F

# Linux/Mac
lsof -i :8080
kill -9 <PID>
```

### 4. Air 热更新不工作

```bash
# 清理缓存
rm -rf tmp/

# 重新安装 Air
go install github.com/air-verse/air@v1.49.0

# 检查 .air.toml 配置
```

## 🔄 重置环境

```bash
# 停止所有容器
docker stop zk-mongo zk-redis zk-judge

# 删除容器
docker rm zk-mongo zk-redis zk-judge

# 清理数据（谨慎！）
rm -rf data/

# 重新启动
# 执行步骤 1 的命令
```

## 📊 开发工具推荐

- **MongoDB 客户端**: MongoDB Compass
- **Redis 客户端**: RedisInsight
- **API 测试**: Postman / Insomnia
- **日志查看**: 浏览器访问 http://localhost:8080/swagger/index.html

## 🎯 下一步

1. ✅ 启动所有依赖服务
2. ✅ 修改配置文件
3. ✅ 启动应用
4. ✅ 测试基础功能
5. 📝 开始开发新功能

## 💡 提示

- 开发环境建议关闭 Redis 密码和限流功能
- 使用 `air` 可以实现代码热更新，提高开发效率
- 定期备份 `data/` 目录中的数据
- 生产环境务必修改 JWT Secret 和数据库密码
