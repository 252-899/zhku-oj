# ZK Code Arena Server 快速启动指南

## 🚀 快速开始

### 方式一：使用 Docker Compose（推荐）

1. **克隆项目**
```bash
git clone <repository-url>
cd ZKCodeArenaServer
```

2. **启动服务**
```bash
# 生产环境
chmod +x scripts/start.sh
./scripts/start.sh

# 开发环境
./scripts/start.sh dev
```

3. **访问服务**
- 主服务: http://localhost
- API 接口: http://localhost/api/v1
- 健康检查: http://localhost/health

### 方式二：本地开发环境

1. **安装依赖**
```bash
# 安装 Go 1.23+
# 安装 Docker

# 安装 Air 热更新工具 (使用兼容 Go 1.23 的版本)
go install github.com/air-verse/air@v1.49.0
```

2. **启动依赖服务**
```bash
# 启动 MongoDB
docker run -d --name zk-arena-mongo -p 27017:27017 mongo:7.0

# 启动 Redis
docker run -d --name zk-arena-redis -p 6379:6379 redis:7.2-alpine

# 启动 go-judge
docker run -d --name zk-arena-judge -p 5050:5050 --privileged criyle/go-judge:latest
```

3. **启动应用**
```bash
# 使用 Air 热更新
chmod +x scripts/dev.sh
./scripts/dev.sh

# 或者直接运行
go run cmd/*.go run
```

## 📋 默认配置

### 数据库连接
- MongoDB: `mongodb://localhost:27017`
- 数据库名: `zk_code_arena`
- Redis: `localhost:6379`

### 默认管理员账户
- 用户名: `admin`
- 邮箱: `admin@zk.edu.cn`
- 密码: 需要先设置（通过注册或数据库直接设置）

## 🔧 配置说明

### 环境变量
复制 `env.example` 为 `.env` 并修改相应配置：
```bash
cp env.example .env
```

### 主要配置项
- `APP_PORT`: 应用端口（默认 8080）
- `MONGO_URI`: MongoDB 连接字符串
- `JWT_SECRET`: JWT 密钥
- `LOG_LEVEL`: 日志级别

## 📚 API 使用示例

### 用户注册
```bash
curl -X POST http://localhost/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123",
    "email": "test@example.com",
    "real_name": "测试用户"
  }'
```

### 用户登录
```bash
curl -X POST http://localhost/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "password123"
  }'
```

### 获取题目列表
```bash
curl -X GET "http://localhost/api/v1/problem/?page=1&page_size=10"
```

### 提交代码
```bash
curl -X POST http://localhost/api/v1/submit/ \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "problem_id": "PROBLEM_ID",
    "code": "#include <stdio.h>\nint main() { printf(\"Hello World\"); return 0; }",
    "language": "c"
  }'
```

## 🛠️ 开发工具

### 热更新开发
```bash
# 使用 Air 实现代码热更新
air

# 或者使用 Docker 开发环境
docker-compose -f docker-compose.dev.yml up
```

### 数据库管理
```bash
# 连接 MongoDB
docker exec -it zk-arena-mongo mongosh

# 查看数据库
use zk_code_arena
show collections
```

### 日志查看
```bash
# 查看应用日志
docker-compose logs -f app

# 查看所有服务日志
docker-compose logs -f
```

## 🔍 故障排除

### 常见问题

1. **端口被占用**
```bash
# 检查端口占用
lsof -i :8080
# 或修改配置文件中的端口
```

2. **数据库连接失败**
```bash
# 检查 MongoDB 是否运行
docker ps | grep mongo
# 检查连接字符串配置
```

3. **权限问题**
```bash
# 给脚本添加执行权限
chmod +x scripts/*.sh
```

### 重置环境
```bash
# 停止所有服务
./scripts/stop.sh

# 清理 Docker 资源
docker-compose down -v
docker system prune -f

# 重新启动
./scripts/start.sh
```

## 📖 更多信息

- 详细文档: [README.md](README.md)
- API 文档: 启动后访问 http://localhost/api/v1
- 问题反馈: 请提交 Issue

## 🎯 下一步

1. 配置生产环境变量
2. 设置 SSL 证书
3. 配置监控和日志收集
4. 部署到云服务器
