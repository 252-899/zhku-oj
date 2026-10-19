#!/bin/bash

# 服务器部署新分支脚本
# Author: omenkk7
# Date: 2024/10/26

set -e  # 遇到错误立即退出

BRANCH_NAME="feature/20251026_v1"
PROJECT_DIR="/path/to/your/project"  # 请修改为实际项目路径

echo "🚀 开始部署新分支: $BRANCH_NAME"

# 1. 进入项目目录
cd $PROJECT_DIR
echo "📁 当前目录: $(pwd)"

# 2. 备份当前分支信息
CURRENT_BRANCH=$(git branch --show-current)
echo "📋 当前分支: $CURRENT_BRANCH"

# 3. 获取最新代码
echo "📥 获取远程最新信息..."
git fetch origin

# 4. 切换到新分支
echo "🔄 切换到分支: $BRANCH_NAME"
if git show-ref --verify --quiet refs/heads/$BRANCH_NAME; then
    git checkout $BRANCH_NAME
    git pull origin $BRANCH_NAME
else
    git checkout -b $BRANCH_NAME origin/$BRANCH_NAME
fi

# 5. 显示最新提交信息
echo "📝 最新提交信息:"
git log --oneline -5

# 6. 停止现有服务
echo "⏹️  停止现有服务..."
docker-compose down

# 7. 清理旧镜像（可选）
read -p "是否清理旧镜像以节省空间？(y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🧹 清理旧镜像..."
    docker system prune -f
    docker image prune -f
fi

# 8. 创建数据目录
echo "📁 创建数据目录..."
mkdir -p data/mongo data/redis data/judge logs/nginx
chmod -R 755 data/

# 9. 构建新镜像
echo "🔨 构建新镜像..."
docker-compose build --no-cache

# 10. 启动服务
echo "🚀 启动服务..."
docker-compose up -d

# 11. 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 12. 检查服务状态
echo "✅ 检查服务状态:"
docker-compose ps

# 13. 验证部署
echo "🔍 验证部署..."

# 检查应用是否响应
if curl -f -s http://localhost:8080/health > /dev/null; then
    echo "✅ 应用健康检查通过"
else
    echo "❌ 应用健康检查失败"
    echo "📋 查看应用日志:"
    docker-compose logs --tail=20 app
    exit 1
fi

# 检查API文档是否可访问
if curl -f -s http://localhost:8080/docs > /dev/null; then
    echo "✅ API文档访问正常"
else
    echo "⚠️  API文档访问异常，但应用可能仍在启动中"
fi

# 14. 显示访问信息
echo ""
echo "🎉 部署完成！"
echo ""
echo "📊 服务状态:"
docker-compose ps
echo ""
echo "🌐 访问地址:"
echo "- 应用首页: http://localhost:8080"
echo "- API文档 (Scalar): http://localhost:8080/docs"
echo "- API文档 (Swagger): http://localhost:8080/swagger/index.html"
echo ""
echo "📋 有用的命令:"
echo "- 查看日志: docker-compose logs -f app"
echo "- 重启服务: docker-compose restart app"
echo "- 停止服务: docker-compose down"
echo ""
echo "🔧 如需生成测试数据，请运行:"
echo "bash scripts/setup_test_data.sh"
