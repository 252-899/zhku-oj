# 快速命令参考

> 项目路径: `G:\code-oj\ZKCodeArenaServer`

## 🚀 一键启动

### Windows 中运行

```batch
# 双击运行或在 PowerShell/CMD 中执行
start.bat
```

这个脚本会：
1. 检查 WSL2 是否运行
2. 在 WSL2 中配置环境（MongoDB、Redis）
3. 让你选择在 WSL2 还是 Windows 中启动应用

## 📋 常用命令

### 在 Windows PowerShell 中

```powershell
# 进入项目目录
cd G:\code-oj\ZKCodeArenaServer

# 进入 WSL2
wsl

# 在 WSL2 中运行命令（不进入 WSL2）
wsl bash -c "cd /mnt/g/code-oj/ZKCodeArenaServer && go run cmd/*.go run"

# 查看 WSL2 中的 Docker 容器
wsl docker ps

# 获取 WSL2 的 IP 地址
wsl hostname -I

# 测试 WSL2 服务
curl http://localhost:5050/version
curl http://localhost:8080/health
```

### 在 WSL2 中

```bash
# 进入项目目录（从任何位置）
cd /mnt/g/code-oj/ZKCodeArenaServer

# 配置环境（首次运行或重置环境时）
./scripts/setup_wsl2.sh

# 启动应用
go run cmd/*.go run

# 使用 Air 热更新
air

# 运行测试
./scripts/test_api.sh

# 查看 Docker 容器
docker ps

# 查看容器日志
docker logs zk-mongo
docker logs zk-redis

# 重启容器
docker restart zk-mongo zk-redis

# 停止容器
docker stop zk-mongo zk-redis

# 删除容器（重置）
docker rm -f zk-mongo zk-redis
```

## 🔧 服务管理

### 启动所有服务

```bash
# 在 WSL2 中
cd /mnt/g/code-oj/ZKCodeArenaServer
./scripts/setup_wsl2.sh
go run cmd/*.go run
```

### 停止所有服务

```bash
# 停止应用: Ctrl+C

# 停止 Docker 容器
docker stop zk-mongo zk-redis

# 或在 Windows 中
wsl docker stop zk-mongo zk-redis
```

### 重启服务

```bash
# 重启 Docker 容器
docker restart zk-mongo zk-redis

# 重启应用: Ctrl+C 然后重新运行
go run cmd/*.go run
```

### 查看服务状态

```bash
# 查看 Docker 容器状态
docker ps

# 查看应用日志
# 应用日志会输出到终端

# 测试服务连接
curl http://localhost:8080/health
curl http://localhost:5050/version
docker exec zk-mongo mongosh --quiet --eval "db.version()"
docker exec zk-redis redis-cli PING
```

## 🧪 测试命令

### 健康检查

```bash
curl http://localhost:8080/health
```

### 用户注册

```bash
curl -X POST http://localhost:8080/api/v1/user/ \
  -H "Content-Type: application/json" \
  -d '{
    "student_id": "2021001",
    "password": "test123456",
    "real_name": "测试用户",
    "email": "test@zk.edu.cn"
  }'
```

### 用户登录

```bash
curl -X POST http://localhost:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "student_id": "2021001",
    "password": "test123456"
  }'
```

### 获取题目列表

```bash
curl http://localhost:8080/api/v1/problem/?page=1&page_size=10
```

### 完整测试套件

```bash
# 在 WSL2 中
cd /mnt/g/code-oj/ZKCodeArenaServer
./scripts/test_api.sh
```

## 🗄️ 数据库操作

### MongoDB

```bash
# 连接 MongoDB
docker exec -it zk-mongo mongosh

# 在 mongosh 中
use zk_code_arena
show collections
db.users.find()
db.problems.find()

# 退出
exit
```

### Redis

```bash
# 连接 Redis
docker exec -it zk-redis redis-cli

# 在 redis-cli 中
PING
KEYS *
GET some_key

# 退出
exit
```

## 📁 文件操作

### 查看日志

```bash
# 应用日志（如果配置了文件输出）
cat logs/server.log
tail -f logs/server.log

# Docker 容器日志
docker logs zk-mongo
docker logs zk-redis
docker logs -f zk-mongo  # 实时查看
```

### 清理数据

```bash
# 清理 MongoDB 数据
rm -rf data/mongo/*

# 清理 Redis 数据
rm -rf data/redis/*

# 清理日志
rm -rf logs/*

# 重新初始化
./scripts/setup_wsl2.sh
```

## 🔄 开发工作流

### 日常开发

```bash
# 1. 打开 WSL2
wsl

# 2. 进入项目
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 确保服务运行
docker ps  # 检查 MongoDB、Redis 是否运行

# 4. 启动应用（使用 Air 热更新）
air

# 5. 开始编码
# Air 会自动检测文件变化并重启应用
```

### 首次启动

```bash
# 1. 配置环境
cd /mnt/g/code-oj/ZKCodeArenaServer
./scripts/setup_wsl2.sh

# 2. 安装 Air（可选，用于热更新）
go install github.com/air-verse/air@v1.49.0

# 3. 启动应用
air  # 或 go run cmd/*.go run
```

### 重置环境

```bash
# 停止并删除所有容器
docker stop zk-mongo zk-redis
docker rm zk-mongo zk-redis

# 清理数据
rm -rf data/

# 重新配置
./scripts/setup_wsl2.sh
```

## 🌐 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 应用主页 | http://localhost:8080 | API 服务 |
| 健康检查 | http://localhost:8080/health | 服务状态 |
| Swagger 文档 | http://localhost:8080/swagger/index.html | API 文档 |
| Scalar 文档 | http://localhost:8080/scalar | 现代化 API 文档 |
| MongoDB | mongodb://localhost:27017 | 数据库 |
| Redis | localhost:6379 | 缓存 |
| go-judge | http://localhost:5050 | 判题服务 |

## 🛠️ 故障排查

### 服务无法启动

```bash
# 检查端口占用
netstat -tlnp | grep 8080

# 查看容器状态
docker ps -a

# 查看容器日志
docker logs zk-mongo
docker logs zk-redis

# 重启容器
docker restart zk-mongo zk-redis
```

### 无法连接数据库

```bash
# 测试 MongoDB 连接
docker exec zk-mongo mongosh --quiet --eval "db.version()"

# 测试 Redis 连接
docker exec zk-redis redis-cli PING

# 检查容器是否运行
docker ps | grep -E 'zk-mongo|zk-redis'
```

### WSL2 网络问题

```powershell
# 在 Windows PowerShell 中

# 获取 WSL2 IP
wsl hostname -I

# 测试连接
Test-NetConnection -ComputerName localhost -Port 5050
Test-NetConnection -ComputerName localhost -Port 27017

# 重启 WSL2
wsl --shutdown
wsl
```

## 📚 相关文档

- **START_HERE.md** - 快速开始（推荐先看这个）
- **WSL2_SETUP.md** - WSL2 详细配置
- **LOCAL_SETUP.md** - 本地开发配置
- **README.md** - 项目介绍

## 💡 提示

- 使用 `start.bat` 可以一键启动整个环境
- 在 WSL2 中运行项目最简单，网络配置自动
- 使用 `air` 可以实现代码热更新，提高开发效率
- 定期备份 `data/` 目录中的数据
