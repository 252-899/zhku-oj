#!/bin/bash

# ZK Code Arena MongoDB 专用备份脚本
# Author: omenkk7
# Date: 2024/10/26

# 配置
BACKUP_DIR="/var/backups/zk-arena/mongodb"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="zk_code_arena"
CONTAINER_NAME="zk-arena-mongo-dev"
DAYS_TO_KEEP=14  # 保留14天的MongoDB备份

echo "🚀 开始 MongoDB 数据备份..."

# 创建备份目录
mkdir -p "$BACKUP_DIR"

# 检查容器是否运行
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    echo "❌ MongoDB容器 $CONTAINER_NAME 未运行"
    exit 1
fi

# 使用 mongodump 备份
echo "📦 使用 mongodump 备份数据库: $DB_NAME"
docker exec "$CONTAINER_NAME" mongodump \
    --db "$DB_NAME" \
    --out "/tmp/backup_$DATE" \
    --gzip

# 从容器复制备份文件到主机
echo "📁 复制备份文件到主机..."
docker cp "$CONTAINER_NAME:/tmp/backup_$DATE" "$BACKUP_DIR/"

# 创建压缩包
cd "$BACKUP_DIR" || exit 1
tar -czf "mongodb_backup_${DATE}.tar.gz" "backup_$DATE/"
rm -rf "backup_$DATE/"

# 清理容器中的临时文件
docker exec "$CONTAINER_NAME" rm -rf "/tmp/backup_$DATE"

# 检查备份是否成功
if [ -f "$BACKUP_DIR/mongodb_backup_${DATE}.tar.gz" ]; then
    echo "✅ MongoDB备份创建成功: mongodb_backup_${DATE}.tar.gz"
    echo "📊 备份大小: $(du -h "$BACKUP_DIR/mongodb_backup_${DATE}.tar.gz" | cut -f1)"
    
    # 验证备份内容
    echo "🔍 备份内容验证:"
    tar -tzf "$BACKUP_DIR/mongodb_backup_${DATE}.tar.gz" | head -10
else
    echo "❌ MongoDB备份创建失败"
    exit 1
fi

# 清理旧备份
echo "🧹 清理超过 ${DAYS_TO_KEEP} 天的旧备份..."
find "$BACKUP_DIR" -name "mongodb_backup_*.tar.gz" -mtime +$DAYS_TO_KEEP -delete

# 显示备份统计
echo "📈 备份统计:"
echo "最新备份: $(ls -t "$BACKUP_DIR"/mongodb_backup_*.tar.gz 2>/dev/null | head -1)"
echo "备份总数: $(ls "$BACKUP_DIR"/mongodb_backup_*.tar.gz 2>/dev/null | wc -l)"
echo "总占用空间: $(du -sh "$BACKUP_DIR" | cut -f1)"

echo "🎉 MongoDB备份完成！"
