# 🚀 最简单的启动方式（无需打包构建）

> 项目路径: `G:\code-oj\ZKCodeArenaServer`

## 📋 前提条件

- ✅ WSL2 已安装并运行
- ✅ WSL2 中已有判题服务运行
- ✅ Go 已安装（Windows 或 WSL2 中）

## 🎯 推荐方案：在 WSL2 中直接运行

### 一次性配置（只需运行一次）

```bash
# 1. 打开 WSL2
wsl

# 2. 进入项目目录
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 给脚本添加执行权限
chmod +x scripts/setup_wsl2.sh

# 4. 运行配置脚本（启动 MongoDB 和 Redis）
./scripts/setup_wsl2.sh
```

### 日常启动（每次开发时）

```bash
# 1. 打开 WSL2
wsl

# 2. 进入项目目录
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 直接运行（不打包，不构建）
go run cmd/*.go run
```

就这么简单！应用会直接启动在 http://localhost:8080

### 停止应用

按 `Ctrl + C` 即可停止

## 🔄 完整流程示例

```bash
# 打开 WSL2 终端
wsl

# 进入项目
cd /mnt/g/code-oj/ZKCodeArenaServer

# 检查 Docker 服务是否运行（可选）
docker ps

# 如果 MongoDB 或 Redis 没运行，启动它们
docker start zk-mongo zk-redis

# 启动应用
go run cmd/*.go run

# 看到类似输出表示成功：
# [GIN-debug] Listening and serving HTTP on 0.0.0.0:8080
```

## 🧪 验证服务

在另一个终端或浏览器中：

```bash
# 健康检查
curl http://localhost:8080/health

# 或在浏览器访问
# http://localhost:8080/health
# http://localhost:8080/swagger/index.html
```

## 📝 配置说明

配置文件会自动使用 `conf/config.yaml`，所有服务都用 `localhost`：

- MongoDB: `localhost:27017`
- Redis: `localhost:6379`
- go-judge: `localhost:5050`

**不需要修改任何配置文件！**

## 🛠️ 常用命令

### 启动 Docker 服务（如果停止了）

```bash
# 启动 MongoDB
docker start zk-mongo

# 启动 Redis
docker start zk-redis

# 或一起启动
docker start zk-mongo zk-redis
```

### 查看服务状态

```bash
# 查看 Docker 容器
docker ps

# 测试各个服务
curl http://localhost:5050/version  # 判题服务
docker exec zk-mongo mongosh --quiet --eval "db.version()"  # MongoDB
docker exec zk-redis redis-cli PING  # Redis
```

### 查看应用日志

应用日志会直接输出到终端，无需额外操作。

## ⚡ 快捷方式（可选）

### 创建启动别名

在 WSL2 的 `~/.bashrc` 中添加：

```bash
# 编辑 .bashrc
nano ~/.bashrc

# 添加以下内容
alias zkstart='cd /mnt/g/code-oj/ZKCodeArenaServer && go run cmd/*.go run'
alias zkcd='cd /mnt/g/code-oj/ZKCodeArenaServer'

# 保存后重新加载
source ~/.bashrc
```

之后只需运行：

```bash
zkstart  # 直接启动应用
zkcd     # 进入项目目录
```

## 🔍 故障排查

### 问题 1: 无法连接数据库

```bash
# 检查容器是否运行
docker ps | grep -E 'zk-mongo|zk-redis'

# 如果没运行，启动它们
docker start zk-mongo zk-redis

# 如果容器不存在，运行配置脚本
./scripts/setup_wsl2.sh
```

### 问题 2: 端口被占用

```bash
# 查看 8080 端口占用
netstat -tlnp | grep 8080

# 如果被占用，可以修改配置文件中的端口
# 或停止占用端口的进程
```

### 问题 3: Go 命令未找到

```bash
# 检查 Go 是否安装
go version

# 如果未安装，在 WSL2 中安装
sudo apt update
sudo apt install golang-go

# 或安装最新版本
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

## 📊 开发工作流

### 每天开始工作

```bash
# 1. 打开 WSL2
wsl

# 2. 进入项目
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 确保 Docker 服务运行
docker ps

# 4. 启动应用
go run cmd/*.go run
```

### 修改代码后

1. 按 `Ctrl + C` 停止应用
2. 重新运行 `go run cmd/*.go run`

**就这么简单！无需构建、打包或热更新。**

## 💡 为什么这样做最简单？

1. ✅ **无需构建**: `go run` 直接运行源码
2. ✅ **无需打包**: 不生成可执行文件
3. ✅ **无需热更新**: 手动重启即可，简单直接
4. ✅ **统一环境**: 所有服务都在 WSL2 中，网络配置简单
5. ✅ **快速启动**: 一条命令搞定

## 🎯 总结

**最简单的启动流程：**

```bash
wsl
cd /mnt/g/code-oj/ZKCodeArenaServer
go run cmd/*.go run
```

**就这三条命令！**

---

**需要帮助？** 查看 START_HERE.md 或 QUICK_COMMANDS.md
