# WSL2 环境安装指南

> 针对项目路径: `G:\code-oj\ZKCodeArenaServer`

## 🔧 首次安装步骤

### 步骤 1: 安装 Go

你的 WSL2 中还没有安装 Go，需要先安装。

#### 方法 A: 使用自动安装脚本（推荐）

```bash
# 在 WSL2 中
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

#### 方法 B: 手动安装

```bash
# 1. 下载 Go 1.23
cd /tmp
wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz

# 2. 删除旧版本（如果有）
sudo rm -rf /usr/local/go

# 3. 解压安装
sudo tar -C /usr/local -xzf go1.23.4.linux-amd64.tar.gz

# 4. 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc

# 5. 重新加载配置
source ~/.bashrc

# 6. 验证安装
go version
```

#### 方法 C: 使用包管理器（最简单但版本可能较旧）

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# 验证安装
go version
```

**注意**: 包管理器安装的版本可能是 1.21，项目需要 1.23+。推荐使用方法 A 或 B。

### 步骤 2: 配置项目环境

安装 Go 后，配置项目环境：

```bash
# 进入项目目录
cd /mnt/g/code-oj/ZKCodeArenaServer

# 给脚本添加执行权限
chmod +x scripts/setup_wsl2.sh scripts/test_api.sh

# 运行配置脚本
./scripts/setup_wsl2.sh
```

这个脚本会：
- ✅ 检测你现有的判题服务
- ✅ 启动 MongoDB 容器
- ✅ 启动 Redis 容器
- ✅ 创建配置文件

### 步骤 3: 安装项目依赖

```bash
# 下载 Go 模块依赖
go mod download

# 或者直接运行，会自动下载依赖
go run cmd/*.go run
```

### 步骤 4: 安装 Air（可选，用于热更新）

```bash
# 安装 Air
go install github.com/air-verse/air@v1.49.0

# 验证安装
air -v
```

### 步骤 5: 启动应用

```bash
# 方式 1: 直接运行
go run cmd/*.go run

# 方式 2: 使用 Air 热更新（推荐开发时使用）
air
```

### 步骤 6: 验证服务

在另一个终端或浏览器中：

```bash
# 健康检查
curl http://localhost:8080/health

# 或在 Windows 浏览器访问
# http://localhost:8080/health
```

## 🚀 完整的一键安装命令

如果你想一次性执行所有步骤：

```bash
# 在 WSL2 中执行
cd /mnt/g/code-oj/ZKCodeArenaServer && \
chmod +x scripts/*.sh && \
./scripts/install_go.sh && \
source ~/.bashrc && \
./scripts/setup_wsl2.sh && \
go mod download && \
echo "安装完成！现在可以运行: go run cmd/*.go run"
```

## 📋 安装检查清单

完成安装后，检查以下项目：

```bash
# 1. Go 版本（应该是 1.23+）
go version

# 2. Docker 服务
docker ps

# 应该看到:
# - zk-mongo (MongoDB)
# - zk-redis (Redis)
# - 你的判题服务

# 3. 测试各个服务
curl http://localhost:5050/version  # 判题服务
docker exec zk-mongo mongosh --quiet --eval "db.version()"  # MongoDB
docker exec zk-redis redis-cli PING  # Redis

# 4. 项目依赖
go mod verify
```

## 🔍 故障排查

### 问题 1: Go 安装后找不到命令

```bash
# 检查 Go 是否安装
ls -la /usr/local/go/bin/go

# 手动添加到 PATH
export PATH=$PATH:/usr/local/go/bin

# 或重新加载配置
source ~/.bashrc

# 或关闭并重新打开 WSL2 终端
```

### 问题 2: 权限错误

```bash
# 给脚本添加执行权限
chmod +x scripts/*.sh

# 如果需要 sudo 权限
sudo ./scripts/install_go.sh
```

### 问题 3: 下载速度慢

```bash
# 使用国内镜像加速 Go 模块下载
go env -w GOPROXY=https://goproxy.cn,direct

# 或使用阿里云镜像
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct
```

### 问题 4: Docker 未安装

```bash
# 检查 Docker
docker --version

# 如果未安装，安装 Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 将当前用户添加到 docker 组
sudo usermod -aG docker $USER

# 重新登录或运行
newgrp docker
```

## 📊 系统要求

- **WSL2**: Ubuntu 20.04+ 或 Debian 11+
- **Go**: 1.23+
- **Docker**: 20.10+
- **磁盘空间**: 至少 5GB
- **内存**: 至少 4GB

## 🎯 安装后的下一步

1. ✅ 安装 Go
2. ✅ 配置项目环境
3. ✅ 启动应用
4. 📝 开始开发

## 💡 开发建议

### 使用 VS Code + WSL 扩展

1. 在 VS Code 中安装 "Remote - WSL" 扩展
2. 按 `Ctrl + Shift + P`，选择 "WSL: Connect to WSL"
3. 在 WSL 中打开项目：`File -> Open Folder -> /mnt/g/code-oj/ZKCodeArenaServer`
4. 安装 Go 扩展（在 WSL 中）

这样可以在 Windows 的 VS Code 中直接编辑 WSL 中的文件，并使用 WSL 的 Go 环境。

### 配置 Go 代理（加速下载）

```bash
# 使用国内镜像
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn

# 验证配置
go env | grep GOPROXY
```

### 安装常用工具

```bash
# Air - 热更新工具
go install github.com/air-verse/air@v1.49.0

# golangci-lint - 代码检查
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# swag - Swagger 文档生成
go install github.com/swaggo/swag/cmd/swag@latest
```

## 📚 相关文档

- **START_HERE.md** - 快速开始指南
- **QUICK_COMMANDS.md** - 常用命令参考
- **WSL2_SETUP.md** - WSL2 详细配置

## 🆘 需要帮助？

如果遇到问题：

1. 查看错误信息
2. 检查 Docker 容器状态：`docker ps`
3. 查看容器日志：`docker logs zk-mongo`
4. 重置环境：删除容器并重新运行配置脚本

---

**准备好了吗？** 运行 `./scripts/install_go.sh` 开始安装！
