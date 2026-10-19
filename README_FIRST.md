# 🚀 首次启动 - 超简单版

> 项目路径: `G:\code-oj\ZKCodeArenaServer`

## ⚡ 一键安装（推荐）

```bash
# 1. 打开 WSL2
wsl

# 2. 进入项目
cd /mnt/g/code-oj/ZKCodeArenaServer

# 3. 运行一键安装脚本
chmod +x scripts/first_time_setup.sh
./scripts/first_time_setup.sh

# 4. 启动应用
go run cmd/*.go run
```

就这么简单！脚本会自动安装 Go、配置 Docker、下载依赖。

## 📋 如果一键安装失败

### 步骤 1: 安装 Go

```bash
cd /mnt/g/code-oj/ZKCodeArenaServer
chmod +x scripts/install_go.sh
./scripts/install_go.sh
source ~/.bashrc
```

### 步骤 2: 配置环境

```bash
chmod +x scripts/setup_wsl2.sh
./scripts/setup_wsl2.sh
```

### 步骤 3: 启动应用

```bash
go run cmd/*.go run
```

## ✅ 验证安装

在浏览器访问: http://localhost:8080/health

应该看到:
```json
{
  "data": {
    "service": "zk-code-arena-server",
    "status": "ok",
    "timestamp": 1700000000
  }
}
```

## 🎯 下一步

- 查看 API 文档: http://localhost:8080/swagger/index.html
- 运行测试: `./scripts/test_api.sh`
- 查看常用命令: `QUICK_COMMANDS.md`

## 🆘 遇到问题？

查看详细文档:
- **INSTALL_GUIDE.md** - 详细安装指南
- **START_HERE.md** - 完整启动指南
- **WSL2_SETUP.md** - WSL2 配置说明

---

**现在就开始**: `./scripts/first_time_setup.sh`
