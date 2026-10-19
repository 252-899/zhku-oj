#!/bin/bash

# ZK Code Arena 数据恢复脚本
# Author: omenkk7
# Date: 2024/10/26

# 配置
BACKUP_DIR="/var/backups/zk-arena"
PROJECT_DIR="/path/to/zhku_oj_projuct"  # 修改为您的项目路径
CONTAINER_NAME="zk-arena-mongo-dev"

echo "🔄 ZK Code Arena 数据恢复工具"
echo "================================"

# 检查参数
if [ $# -eq 0 ]; then
    echo "📁 可用的备份文件:"
    echo ""
    echo "📦 完整备份 (data + logs + conf):"
    ls -lh "$BACKUP_DIR"/zk_arena_backup_*.tar.gz 2>/dev/null | tail -10
    echo ""
    echo "🗄️ MongoDB专用备份:"
    ls -lh "$BACKUP_DIR/mongodb"/mongodb_backup_*.tar.gz 2>/dev/null | tail -10
    echo ""
    echo "使用方法:"
    echo "  $0 <backup_file>           # 恢复完整备份"
    echo "  $0 mongodb <backup_file>   # 恢复MongoDB备份"
    echo ""
    echo "示例:"
    echo "  $0 zk_arena_backup_20241026_143000.tar.gz"
    echo "  $0 mongodb mongodb_backup_20241026_143000.tar.gz"
    exit 1
fi

# 判断恢复类型
if [ "$1" = "mongodb" ]; then
    RESTORE_TYPE="mongodb"
    BACKUP_FILE="$2"
    BACKUP_PATH="$BACKUP_DIR/mongodb/$BACKUP_FILE"
else
    RESTORE_TYPE="full"
    BACKUP_FILE="$1"
    BACKUP_PATH="$BACKUP_DIR/$BACKUP_FILE"
fi

# 检查备份文件是否存在
if [ ! -f "$BACKUP_PATH" ]; then
    echo "❌ 备份文件不存在: $BACKUP_PATH"
    exit 1
fi

echo "📂 恢复类型: $RESTORE_TYPE"
echo "📁 备份文件: $BACKUP_PATH"
echo "📊 文件大小: $(du -h "$BACKUP_PATH" | cut -f1)"
echo ""

# 确认恢复操作
read -p "⚠️  确定要恢复数据吗？这将覆盖现有数据 (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ 恢复操作已取消"
    exit 0
fi

if [ "$RESTORE_TYPE" = "full" ]; then
    # 完整恢复
    echo "🔄 开始完整数据恢复..."
    
    # 停止容器
    echo "⏹️  停止Docker容器..."
    cd "$PROJECT_DIR" || exit 1
    docker-compose -f docker-compose.dev.yml down
    
    # 备份当前数据
    if [ -d "data" ]; then
        echo "💾 备份当前数据到 data_backup_$(date +%Y%m%d_%H%M%S)..."
        mv data "data_backup_$(date +%Y%m%d_%H%M%S)"
    fi
    
    # 解压备份
    echo "📦 解压备份文件..."
    tar -xzf "$BACKUP_PATH"
    
    # 启动容器
    echo "🚀 启动Docker容器..."
    docker-compose -f docker-compose.dev.yml up -d
    
    echo "✅ 完整数据恢复完成！"
    
else
    # MongoDB恢复
    echo "🔄 开始MongoDB数据恢复..."
    
    # 检查容器是否运行
    if ! docker ps | grep -q "$CONTAINER_NAME"; then
        echo "❌ MongoDB容器 $CONTAINER_NAME 未运行"
        echo "请先启动容器: docker-compose -f docker-compose.dev.yml up -d mongo"
        exit 1
    fi
    
    # 创建临时目录
    TEMP_DIR="/tmp/restore_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$TEMP_DIR"
    
    # 解压备份到临时目录
    echo "📦 解压备份文件..."
    tar -xzf "$BACKUP_PATH" -C "$TEMP_DIR"
    
    # 复制到容器
    echo "📁 复制备份到容器..."
    docker cp "$TEMP_DIR"/* "$CONTAINER_NAME:/tmp/"
    
    # 清空现有数据库
    echo "🗑️  清空现有数据库..."
    docker exec "$CONTAINER_NAME" mongosh zk_code_arena --eval "
        db.dropDatabase();
        print('数据库已清空');
    "
    
    # 恢复数据
    echo "🔄 恢复MongoDB数据..."
    RESTORE_PATH=$(find "$TEMP_DIR" -name "zk_code_arena" -type d | head -1)
    RESTORE_PATH_IN_CONTAINER="/tmp/$(basename "$(dirname "$RESTORE_PATH")")/zk_code_arena"
    
    docker exec "$CONTAINER_NAME" mongorestore \
        --db zk_code_arena \
        --gzip \
        "$RESTORE_PATH_IN_CONTAINER"
    
    # 清理临时文件
    echo "🧹 清理临时文件..."
    rm -rf "$TEMP_DIR"
    docker exec "$CONTAINER_NAME" rm -rf /tmp/backup_*
    
    echo "✅ MongoDB数据恢复完成！"
fi

# 验证恢复结果
echo ""
echo "🔍 验证恢复结果:"
sleep 3
docker exec "$CONTAINER_NAME" mongosh zk_code_arena --eval "
    print('👤 用户数量: ' + db.users.countDocuments());
    print('📚 题目数量: ' + db.problems.countDocuments());
    print('💻 提交数量: ' + db.submits.countDocuments());
"

echo ""
echo "🎉 数据恢复操作完成！"
