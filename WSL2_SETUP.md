# WSL2 环境配置指南

## 🐧 WSL2 + Windows 混合开发环境

### 架构说明

```
┌─────────────────────────────────────────────────────────┐
│                    Windows 主机                          │
│  ┌─────────────────────────────────────────────────┐   │
│  │  Go 应用运行在 Windows                           │   │
│  │  - 监听: localhost:8080                         │   │
│  │  - 通过 localhost 访问 WSL2 服务                │   │
│  └─────────────────────────────────────────────────┘   │
│                         ↓                                │
│  ┌─────────────────────────────────────────────────┐   │
│  │              WSL2 (Ubuntu/Debian)                │   │
│  │  ┌──────────────────────────────────────────┐   │   │
│  │  │  Docker 服务                              │   │   │
│  │  │  - MongoDB:    localhost:27017           │   │   │
│  │  │  - Redis:      localhost:6379            │   │   │
│  │  │  - go-judge:   localhost:5050 (已存在)   │   │   │
│  │  └──────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## 🔧 配置步骤

### 1. 检查 WSL2 中的判题服务

在 WSL2 中运行：

```bash
# 检查判题服务是否运行
curl http://localhost:5050/version

# 或者检查 Docker 容器
docker ps | grep judge

# 查看判题服务监听的端口
netstat -tlnp | grep 5050
```

### 2. 确认 WSL2 网络配置

WSL2 默认使用 NAT 网络，Windows 可以通过 `localhost` 访问 WSL2 的服务。

**测试从 Windows 访问 WSL2 服务：**

在 Windows PowerShell 中运行：

```powershell
# 测试访问 WSL2 的判题服务
curl http://localhost:5050/version

# 如果不通，尝试使用 WSL2 的 IP
wsl hostname -I
# 然后用返回的 IP 测试，例如：
curl http://172.x.x.x:5050/version
```

### 3. 配置文件设置

#### 方案 A：判题服务在 WSL2，应用在 Windows

**conf/config.yaml** 配置：

```yaml
App:
  Host: "0.0.0.0"
  Port: "8080"
  Mode: "debug"
  Env: "development"

Mongo:
  Uri: mongodb://localhost:27017  # ✅ WSL2 的 MongoDB
  DbName: zk_code_arena

Judge:
  SandboxURL: "http://localhost:5050"  # ✅ WSL2 的判题服务
  Workers: 2
  QueueSize: 50

Redis:
  Host: "localhost"  # ✅ WSL2 的 Redis
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

#### 方案 B：所有服务都在 WSL2 中运行（推荐）

如果你想在 WSL2 中运行整个项目：

```bash
# 在 WSL2 中
cd /path/to/project

# 启动 MongoDB 和 Redis（如果还没有）
docker run -d --name zk-mongo -p 27017:27017 mongo:7.0
docker run -d --name zk-redis -p 6379:6379 redis:7.2-alpine

# 运行 Go 应用
go run cmd/*.go run

# 或使用 Air 热更新
air
```

配置文件保持不变（使用 localhost）。

### 4. WSL2 启动脚本

创建 **scripts/setup_wsl2.sh**：

```bash
#!/bin/bash
# WSL2 环境配置脚本

echo "========================================"
echo "WSL2 环境配置检查"
echo "========================================"
echo ""

# 检查判题服务
echo "[1/4] 检查判题服务..."
if curl -s http://localhost:5050/version > /dev/null 2>&1; then
    echo "✓ 判题服务已运行"
    JUDGE_EXISTS=true
else
    echo "⚠ 判题服务未运行，将启动新的判题服务"
    JUDGE_EXISTS=false
fi

# 启动 MongoDB
echo ""
echo "[2/4] 配置 MongoDB..."
if docker ps | grep -q zk-mongo; then
    echo "✓ MongoDB 已运行"
elif docker ps -a | grep -q zk-mongo; then
    docker start zk-mongo
    echo "✓ MongoDB 已启动"
else
    docker run -d --name zk-mongo -p 27017:27017 \
        -v "$(pwd)/data/mongo:/data/db" \
        mongo:7.0
    echo "✓ MongoDB 启动成功"
fi

# 启动 Redis
echo ""
echo "[3/4] 配置 Redis..."
if docker ps | grep -q zk-redis; then
    echo "✓ Redis 已运行"
elif docker ps -a | grep -q zk-redis; then
    docker start zk-redis
    echo "✓ Redis 已启动"
else
    docker run -d --name zk-redis -p 6379:6379 \
        -v "$(pwd)/data/redis:/data" \
        redis:7.2-alpine
    echo "✓ Redis 启动成功"
fi

# 判题服务
echo ""
echo "[4/4] 判题服务配置..."
if [ "$JUDGE_EXISTS" = true ]; then
    echo "✓ 使用现有判题服务"
else
    if docker ps -a | grep -q zk-judge; then
        docker start zk-judge
        echo "✓ 判题服务已启动"
    else
        docker run -d --name zk-judge -p 5050:5050 \
            --privileged \
            criyle/go-judge:latest
        echo "✓ 判题服务启动成功"
    fi
fi

echo ""
echo "========================================"
echo "环境配置完成！"
echo "========================================"
echo ""
echo "服务状态:"
echo "  MongoDB:   $(docker ps --filter 'name=zk-mongo' --format '{{.Status}}')"
echo "  Redis:     $(docker ps --filter 'name=zk-redis' --format '{{.Status}}')"
echo "  go-judge:  $(curl -s http://localhost:5050/version 2>&1 | head -n 1 || echo '未运行')"
echo ""
echo "下一步:"
echo "  在 WSL2 中运行: go run cmd/*.go run"
echo "  或在 Windows 中运行: go run cmd/*.go run"
echo ""
```

### 5. 网络故障排查

#### 问题：Windows 无法访问 WSL2 的服务

**解决方案 1：使用 WSL2 的 IP 地址**

```powershell
# 在 Windows PowerShell 中获取 WSL2 IP
wsl hostname -I
# 输出例如: 172.28.176.1

# 修改 config.yaml
Judge:
  SandboxURL: "http://172.28.176.1:5050"

Mongo:
  Uri: mongodb://172.28.176.1:27017

Redis:
  Host: "172.28.176.1"
```

**解决方案 2：配置端口转发（如果 localhost 不工作）**

在 Windows PowerShell（管理员）中运行：

```powershell
# 获取 WSL2 IP
$wslIP = (wsl hostname -I).Trim()

# 转发端口
netsh interface portproxy add v4tov4 listenport=5050 listenaddress=0.0.0.0 connectport=5050 connectaddress=$wslIP
netsh interface portproxy add v4tov4 listenport=27017 listenaddress=0.0.0.0 connectport=27017 connectaddress=$wslIP
netsh interface portproxy add v4tov4 listenport=6379 listenaddress=0.0.0.0 connectport=6379 connectaddress=$wslIP

# 查看转发规则
netsh interface portproxy show all

# 删除转发规则（如果需要）
# netsh interface portproxy delete v4tov4 listenport=5050 listenaddress=0.0.0.0
```

**解决方案 3：在 WSL2 中运行整个项目（最简单）**

```bash
# 在 WSL2 中
cd /mnt/c/path/to/your/project
./scripts/setup_wsl2.sh
go run cmd/*.go run
```

然后在 Windows 浏览器访问 `http://localhost:8080`

### 6. 推荐的开发方式

#### 方式 A：全部在 WSL2（推荐）

```bash
# 在 WSL2 中
cd /mnt/c/Users/YourName/Projects/ZKCodeArenaServer

# 配置环境
chmod +x scripts/setup_wsl2.sh
./scripts/setup_wsl2.sh

# 启动应用
air  # 或 go run cmd/*.go run
```

**优点：**
- 网络配置简单，全部使用 localhost
- 性能更好
- 不需要处理跨系统网络问题

**缺点：**
- 需要在 WSL2 中安装 Go 和其他工具

#### 方式 B：应用在 Windows，服务在 WSL2

```powershell
# 在 Windows PowerShell 中
cd C:\path\to\project

# 确保可以访问 WSL2 服务
curl http://localhost:5050/version

# 启动应用
air  # 或 go run cmd/*.go run
```

**优点：**
- 可以使用 Windows 的 IDE 和工具
- 文件系统性能更好

**缺点：**
- 可能需要配置网络转发
- WSL2 IP 可能会变化

### 7. 配置文件模板

创建 **conf/config.wsl2.yaml**：

```yaml
App:
  Host: "0.0.0.0"
  Port: "8080"
  Mode: "debug"
  Env: "development"

Mongo:
  Uri: mongodb://localhost:27017  # 如果不通，改为 WSL2 IP
  DbName: zk_code_arena

Judge:
  SandboxURL: "http://localhost:5050"  # 使用现有判题服务
  Workers: 2
  QueueSize: 50

Redis:
  Host: "localhost"  # 如果不通，改为 WSL2 IP
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

### 8. 快速测试

```bash
# 在 WSL2 中测试所有服务
echo "测试 MongoDB..."
docker exec zk-mongo mongosh --quiet --eval "db.version()"

echo "测试 Redis..."
docker exec zk-redis redis-cli PING

echo "测试 go-judge..."
curl http://localhost:5050/version

# 在 Windows 中测试（PowerShell）
echo "从 Windows 测试 WSL2 服务..."
curl http://localhost:5050/version
curl http://localhost:27017  # 应该返回连接信息
```

## 🎯 推荐配置

**最简单的方式：全部在 WSL2 中运行**

1. 在 WSL2 中克隆项目（或通过 `/mnt/c/...` 访问）
2. 运行 `./scripts/setup_wsl2.sh`
3. 使用 `air` 或 `go run cmd/*.go run` 启动
4. 在 Windows 浏览器访问 `http://localhost:8080`

这样所有服务都在同一个网络环境中，配置最简单！

## 📝 注意事项

1. **WSL2 IP 会变化**：每次重启 WSL2，IP 可能会改变
2. **使用 localhost 优先**：大多数情况下 Windows 可以通过 localhost 访问 WSL2
3. **防火墙**：确保 Windows 防火墙允许相关端口
4. **性能**：跨文件系统访问（/mnt/c）会比较慢，建议在 WSL2 文件系统中开发

## 🔍 故障排查命令

```bash
# 在 WSL2 中
# 查看监听的端口
netstat -tlnp | grep -E '5050|27017|6379|8080'

# 查看 Docker 容器
docker ps

# 测试服务
curl http://localhost:5050/version
curl http://localhost:27017
redis-cli -h localhost PING

# 在 Windows PowerShell 中
# 查看 WSL2 IP
wsl hostname -I

# 测试连接
Test-NetConnection -ComputerName localhost -Port 5050
Test-NetConnection -ComputerName localhost -Port 27017
Test-NetConnection -ComputerName localhost -Port 6379
```
