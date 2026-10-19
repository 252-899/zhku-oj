#!/bin/bash

# ZK Code Arena 数据备份脚本
# Author: omenkk7
# Date: 2024/10/26

# 配置
BACKUP_DIR="/var/backups/zk-arena"
PROJECT_DIR="/path/to/zhku_oj_projuct"  # 修改为您的项目路径
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="zk_arena_backup_${DATE}.tar.gz"
DAYS_TO_KEEP=7  # 保留7天的备份

echo "🚀 开始备份 ZK Code Arena 数据..."

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 进入项目目录
cd "$PROJECT_DIR" || exit 1

# 创建备份
echo "📦 创建数据备份: $BACKUP_FILE"
tar -czf "$BACKUP_DIR/$BACKUP_FILE" \
    data/ \
    logs/ \
    conf/ \
    --exclude="logs/*.log" \
    --exclude="data/*/diagnostic.data" \
    --exclude="data/*/journal"

# 检查备份是否成功
if [ $? -eq 0 ]; then
    echo "✅ 备份创建成功: $BACKUP_DIR/$BACKUP_FILE"
    echo "📊 备份大小: $(du -h "$BACKUP_DIR/$BACKUP_FILE" | cut -f1)"
else
    echo "❌ 备份创建失败"
    exit 1
fi

# 清理旧备份
echo "🧹 清理超过 ${DAYS_TO_KEEP} 天的旧备份..."
find "$BACKUP_DIR" -name "zk_arena_backup_*.tar.gz" -mtime +$DAYS_TO_KEEP -delete

# 显示当前所有备份
echo "📁 当前备份列表:"
ls -lh "$BACKUP_DIR"/zk_arena_backup_*.tar.gz 2>/dev/null || echo "没有找到备份文件"

echo "🎉 备份完成！"
