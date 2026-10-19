#!/bin/bash
# 首次安装脚本 - 安装所有依赖并启动项目

set -e

echo "========================================"
echo "ZK Code Arena 首次安装"
echo "========================================"
echo ""
echo "这个脚本将:"
echo "  1. 安装 Go 1.23"
echo "  2. 配置 Docker 服务（MongoDB、Redis）"
echo "  3. 下载项目依赖"
echo "  4. 安装开发工具（Air）"
echo ""
read -p "按 Enter 继续，或 Ctrl+C 取消..."
echo ""

# 检查是否在项目目录
if [ ! -f "go.mod" ]; then
    echo "[错误] 请在项目根目录运行此脚本"
    echo "当前目录: $(pwd)"
    echo "期望目录: /mnt/g/code-oj/ZKCodeArenaServer"
    exit 1
fi

# 步骤 1: 安装 Go
echo "========================================"
echo "[1/5] 安装 Go"
echo "========================================"
echo ""

if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo "✓ Go 已安装: $GO_VERSION"
    
    # 检查版本是否满足要求
    MAJOR=$(echo $GO_VERSION | sed 's/go//' | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | sed 's/go//' | cut -d. -f2)
    
    if [ "$MAJOR" -lt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -lt 23 ]); then
        echo "⚠ Go 版本过低，需要 1.23+，正在升级..."
        chmod +x scripts/install_go.sh
        ./scripts/install_go.sh
        source ~/.bashrc
    fi
else
    echo "正在安装 Go..."
    chmod +x scripts/install_go.sh
    ./scripts/install_go.sh
    source ~/.bashrc
fi

# 验证 Go 安装
if ! command -v go &> /dev/null; then
    export PATH=$PATH:/usr/local/go/bin
fi

echo ""
echo "✓ Go 版本: $(go version)"
echo ""

# 步骤 2: 配置 Docker 服务
echo "========================================"
echo "[2/5] 配置 Docker 服务"
echo "========================================"
echo ""

chmod +x scripts/setup_wsl2.sh
./scripts/setup_wsl2.sh

echo ""

# 步骤 3: 配置 Go 代理（加速下载）
echo "========================================"
echo "[3/5] 配置 Go 代理"
echo "========================================"
echo ""

echo "配置国内镜像加速..."
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
echo "✓ Go 代理配置完成"
echo ""

# 步骤 4: 下载项目依赖
echo "========================================"
echo "[4/5] 下载项目依赖"
echo "========================================"
echo ""

echo "正在下载 Go 模块..."
go mod download
echo "✓ 依赖下载完成"
echo ""

# 步骤 5: 安装开发工具
echo "========================================"
echo "[5/5] 安装开发工具"
echo "========================================"
echo ""

echo "安装 Air（热更新工具）..."
if command -v air &> /dev/null; then
    echo "✓ Air 已安装"
else
    go install github.com/air-verse/air@v1.49.0
    echo "✓ Air 安装完成"
fi

echo ""
echo "========================================"
echo "安装完成！🎉"
echo "========================================"
echo ""
echo "环境信息:"
echo "  Go 版本:    $(go version | awk '{print $3}')"
echo "  项目路径:   $(pwd)"
echo "  Go 代理:    $(go env GOPROXY)"
echo ""
echo "服务状态:"
docker ps --filter "name=zk-" --format "  {{.Names}}: {{.Status}}"
echo ""
echo "下一步:"
echo "  1. 启动应用:"
echo "     go run cmd/*.go run"
echo ""
echo "  2. 或使用 Air 热更新:"
echo "     air"
echo ""
echo "  3. 在浏览器访问:"
echo "     http://localhost:8080/health"
echo ""
echo "  4. 查看 API 文档:"
echo "     http://localhost:8080/swagger/index.html"
echo ""
echo "常用命令:"
echo "  启动应用:   go run cmd/*.go run"
echo "  热更新:     air"
echo "  运行测试:   ./scripts/test_api.sh"
echo "  查看容器:   docker ps"
echo ""
echo "详细文档:"
echo "  快速开始:   START_HERE.md"
echo "  常用命令:   QUICK_COMMANDS.md"
echo "  安装指南:   INSTALL_GUIDE.md"
echo "========================================"
