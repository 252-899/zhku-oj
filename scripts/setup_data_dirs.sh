#!/bin/bash

# 创建数据目录脚本
# 用于初始化项目所需的数据存储目录

echo "🚀 开始创建数据目录..."

# 创建主数据目录
mkdir -p data

# 创建各服务的数据目录
mkdir -p data/mongo
mkdir -p data/redis  
mkdir -p data/judge

# 创建日志目录
mkdir -p logs
mkdir -p logs/nginx

# 设置目录权限
chmod 755 data
chmod 755 data/mongo
chmod 755 data/redis
chmod 755 data/judge
chmod 755 logs
chmod 755 logs/nginx

echo "✅ 数据目录创建完成！"
echo ""
echo "📁 创建的目录结构："
echo "├── data/"
echo "│   ├── mongo/     # MongoDB 数据存储"
echo "│   ├── redis/     # Redis 数据存储"
echo "│   └── judge/     # 判题服务数据"
echo "└── logs/"
echo "    └── nginx/     # Nginx 日志"
echo ""
echo "🔧 使用方法："
echo "1. 运行此脚本: bash scripts/setup_data_dirs.sh"
echo "2. 启动服务: docker-compose up -d"
echo ""
echo "💡 提示：这些目录会被挂载到 Docker 容器中，数据将持久化保存在主机上"
