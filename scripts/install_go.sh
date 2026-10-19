#!/bin/bash
# 在 WSL2 中安装 Go 1.23

set -e

echo "========================================"
echo "安装 Go 1.23 到 WSL2"
echo "========================================"
echo ""

# 检查是否已安装 Go
if command -v go &> /dev/null; then
    CURRENT_VERSION=$(go version | awk '{print $3}')
    echo "检测到已安装的 Go: $CURRENT_VERSION"
    echo ""
    read -p "是否要重新安装? (y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "跳过安装"
        exit 0
    fi
fi

# Go 版本
GO_VERSION="1.23.4"
GO_TAR="go${GO_VERSION}.linux-amd64.tar.gz"
GO_URL="https://go.dev/dl/${GO_TAR}"

echo "[1/5] 下载 Go ${GO_VERSION}..."
cd /tmp
if [ -f "$GO_TAR" ]; then
    echo "✓ 安装包已存在，跳过下载"
else
    echo "  从 ${GO_URL} 下载..."
    wget -q --show-progress "$GO_URL"
    echo "✓ 下载完成"
fi

echo ""
echo "[2/5] 删除旧版本..."
sudo rm -rf /usr/local/go
echo "✓ 旧版本已删除"

echo ""
echo "[3/5] 解压安装..."
sudo tar -C /usr/local -xzf "$GO_TAR"
echo "✓ 解压完成"

echo ""
echo "[4/5] 配置环境变量..."

# 检查 .bashrc 中是否已有 Go 配置
if grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo "✓ 环境变量已配置"
else
    echo "" >> ~/.bashrc
    echo "# Go environment" >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export GOPATH=$HOME/go' >> ~/.bashrc
    echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
    echo "✓ 环境变量已添加到 ~/.bashrc"
fi

# 立即生效
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

echo ""
echo "[5/5] 验证安装..."
if /usr/local/go/bin/go version &> /dev/null; then
    GO_INSTALLED_VERSION=$(/usr/local/go/bin/go version)
    echo "✓ Go 安装成功: $GO_INSTALLED_VERSION"
else
    echo "✗ Go 安装失败"
    exit 1
fi

echo ""
echo "========================================"
echo "安装完成！"
echo "========================================"
echo ""
echo "Go 版本: $(/usr/local/go/bin/go version)"
echo "Go 路径: /usr/local/go/bin/go"
echo ""
echo "下一步:"
echo "  1. 重新加载环境变量:"
echo "     source ~/.bashrc"
echo ""
echo "  2. 或关闭并重新打开终端"
echo ""
echo "  3. 验证安装:"
echo "     go version"
echo ""
echo "  4. 继续配置项目:"
echo "     cd /mnt/g/code-oj/ZKCodeArenaServer"
echo "     ./scripts/setup_wsl2.sh"
echo "========================================"

# 清理下载文件
rm -f "/tmp/$GO_TAR"
