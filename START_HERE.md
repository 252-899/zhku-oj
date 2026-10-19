# 🚀 快速启动指南

> 项目路径: `G:\code-oj\ZKCodeArenaServer`

## 📋 你的环境

- ✅ Windows 系统
- ✅ WSL2 已安装
- ✅ Docker 运行在 WSL2
- ✅ 判题服务已在 WSL2 中运行

## 🎯 推荐方案：在 WSL2 中运行整个项目

### 方式一：通过 WSL2 访问 Windows 文件系统（简单）

```bash
# 1. 打开 WSL2 终端（在 Windows 中运行）
wsl

# 2. 进入项目目录（Windows 的 G 盘在 WSL2 中映射为 /mnt/g）
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 运行配置脚本
chmod +x scripts/setup_wsl2.sh
./scripts/setup_wsl2.sh

# 4. 启动应用
go run cmd/*.go run

# 或使用 Air 热更新
air
```

### 方式二：在 Windows 中运行（需要配置网络）

```powershell
# 1. 打开 PowerShell，进入项目目录
cd G:\code-oj\ZKCodeArenaServer

# 2. 测试能否访问 WSL2 服务
curl http://localhost:5050/version

# 3. 如果上面的命令成功，运行配置脚本
.\scripts\setup_local.bat

# 4. 启动应用
go run cmd/*.go run

# 或使用 Air
air
```

## 🔧 详细步骤

### 步骤 0: 安装 Go（首次运行需要）

如果你的 WSL2 中还没有安装 Go，需要先安装：

```bash
# 打开 WSL2
wsl

# 进入项目
cd /mnt/g/code-oj/ZKCodeArenaServer

# 给脚本添加执行权限
chmod +x scripts/install_go.sh

# 运行安装脚本
./scripts/install_go.sh

# 重新加载环境变量
source ~/.bashrc

# 验证安装
go version
```

**详细安装指南**: 查看 `INSTALL_GUIDE.md`

### 步骤 1: 在 WSL2 中配置环境

```bash
# 如果还没有打开 WSL2
wsl

# 进入项目（G 盘映射为 /mnt/g）
cd /mnt/g/code-oj/ZKCodeArenaServer

# 给脚本添加执行权限
chmod +x scripts/setup_wsl2.sh scripts/test_api.sh

# 运行配置脚本
./scripts/setup_wsl2.sh
```

**脚本会自动：**
- ✅ 检测你现有的判题服务（不会创建新的）
- ✅ 启动 MongoDB 容器
- ✅ 启动 Redis 容器
- ✅ 创建配置文件 `conf/config.wsl2.yaml`

### 步骤 2: 启动应用

#### 选项 A：在 WSL2 中启动（推荐）

```bash
# 在 WSL2 中
cd /mnt/g/code-oj/ZKCodeArenaServer

# 直接运行
go run cmd/*.go run

# 或使用 Air 热更新（需要先安装）
go install github.com/air-verse/air@v1.49.0
air
```

#### 选项 B：在 Windows 中启动

```powershell
# 在 PowerShell 中
cd G:\code-oj\ZKCodeArenaServer

# 直接运行
go run cmd/*.go run

# 或使用 Air
air
```

### 步骤 3: 验证服务

```bash
# 在 WSL2 或 Windows PowerShell 中
curl http://localhost:8080/health

# 预期响应
# {"data":{"service":"zk-code-arena-server","status":"ok","timestamp":...}}
```

## 🌐 访问服务

启动成功后，在 Windows 浏览器中访问：

- **健康检查**: http://localhost:8080/health
- **API 文档 (Swagger)**: http://localhost:8080/swagger/index.html
- **API 文档 (Scalar)**: http://localhost:8080/scalar
- **API 基础路径**: http://localhost:8080/api/v1

## 🧪 测试基础功能

### 在 WSL2 中运行测试

```bash
cd /mnt/g/code-oj/ZKCodeArenaServer
./scripts/test_api.sh
```

### 手动测试 API

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

# 3. 获取题目列表
curl http://localhost:8080/api/v1/problem/?page=1&page_size=10
```

## 📁 项目路径映射

| Windows 路径 | WSL2 路径 |
|-------------|-----------|
| `G:\code-oj\ZKCodeArenaServer` | `/mnt/g/code-oj/ZKCodeArenaServer` |
| `G:\code-oj\ZKCodeArenaServer\data` | `/mnt/g/code-oj/ZKCodeArenaServer/data` |
| `G:\code-oj\ZKCodeArenaServer\logs` | `/mnt/g/code-oj/ZKCodeArenaServer/logs` |

## 🔍 检查服务状态

### 在 WSL2 中

```bash
# 查看 Docker 容器
docker ps

# 应该看到：
# - zk-mongo (MongoDB)
# - zk-redis (Redis)
# - 你的判题服务容器

# 测试各个服务
curl http://localhost:5050/version  # 判题服务
docker exec zk-mongo mongosh --quiet --eval "db.version()"  # MongoDB
docker exec zk-redis redis-cli PING  # Redis
```

### 在 Windows 中

```powershell
# 查看 WSL2 中的 Docker 容器
wsl docker ps

# 测试服务连接
curl http://localhost:5050/version
curl http://localhost:27017
```

## ⚠️ 常见问题

### 问题 1: 无法访问 WSL2 服务

**症状**: 在 Windows 中运行应用，无法连接到 MongoDB/Redis/判题服务

**解决方案**:

```powershell
# 1. 获取 WSL2 的 IP 地址
wsl hostname -I
# 输出例如: 172.28.176.1

# 2. 修改 conf/config.yaml
# 将所有 localhost 改为 WSL2 的 IP
```

或者，**在 WSL2 中运行应用**（推荐）。

### 问题 2: 权限错误

```bash
# 在 WSL2 中给脚本添加执行权限
chmod +x scripts/*.sh
```

### 问题 3: Go 命令未找到

```bash
# 在 WSL2 中安装 Go
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# 或下载最新版本
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### 问题 4: 端口被占用

```bash
# 在 WSL2 中查看端口占用
netstat -tlnp | grep 8080

# 或修改配置文件中的端口
# conf/config.yaml -> App.Port: "8081"
```

### 问题 5: Docker 容器无法启动

```bash
# 查看容器日志
docker logs zk-mongo
docker logs zk-redis

# 重启容器
docker restart zk-mongo zk-redis

# 删除并重新创建
docker rm -f zk-mongo zk-redis
./scripts/setup_wsl2.sh
```

## 🔄 重启服务

### 重启 Docker 服务

```bash
# 在 WSL2 中
docker restart zk-mongo zk-redis

# 如果判题服务也是 Docker 容器
docker restart <judge-container-name>
```

### 重启应用

```bash
# 如果使用 go run
# 按 Ctrl+C 停止，然后重新运行
go run cmd/*.go run

# 如果使用 Air
# Air 会自动重启，或按 Ctrl+C 停止后重新运行
air
```

## 📊 开发工具

### VS Code 配置

如果使用 VS Code，可以安装 WSL 扩展：

1. 安装 "Remote - WSL" 扩展
2. 在 VS Code 中打开 WSL 终端：`Ctrl + Shift + P` -> "WSL: Connect to WSL"
3. 在 WSL 中打开项目：`File -> Open Folder -> /mnt/g/code-oj/ZKCodeArenaServer`

### 数据库管理工具

- **MongoDB**: MongoDB Compass (连接 `mongodb://localhost:27017`)
- **Redis**: RedisInsight (连接 `localhost:6379`)

## 📝 配置文件说明

项目会使用以下配置文件（按优先级）：

1. `conf/config.wsl2.yaml` - WSL2 专用配置（脚本自动生成）
2. `conf/config.local.yaml` - 本地开发配置
3. `conf/config.yaml` - 默认配置

## 🎯 下一步

1. ✅ 运行 `./scripts/setup_wsl2.sh` 配置环境
2. ✅ 启动应用 `go run cmd/*.go run`
3. ✅ 访问 http://localhost:8080/health 验证
4. ✅ 查看 API 文档 http://localhost:8080/swagger/index.html
5. 📝 开始开发！

## 📚 更多文档

- **WSL2_SETUP.md** - WSL2 环境详细配置
- **LOCAL_SETUP.md** - 本地开发环境配置
- **README.md** - 项目总体介绍
- **QUICKSTART.md** - 快速开始指南

## 💡 提示

- 建议在 WSL2 中运行整个项目，网络配置最简单
- 使用 `air` 可以实现代码热更新
- 数据保存在 `data/` 目录，可以安全删除重建
- 生产环境记得修改 JWT Secret 和数据库密码

---

**有问题？** 查看 WSL2_SETUP.md 中的故障排查部分
