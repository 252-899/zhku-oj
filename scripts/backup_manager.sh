#!/bin/bash

# ZK Code Arena 备份管理器
# Author: omenkk7
# Date: 2024/10/26

# 配置文件路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$SCRIPT_DIR/backup.conf"

# 默认配置
DEFAULT_BACKUP_DIR="/var/backups/zk-arena"
DEFAULT_DAYS_TO_KEEP=7
DEFAULT_PROJECT_PATH="$PROJECT_DIR"

# 加载配置
load_config() {
    if [ -f "$CONFIG_FILE" ]; then
        source "$CONFIG_FILE"
    else
        create_default_config
    fi
}

# 创建默认配置文件
create_default_config() {
    cat > "$CONFIG_FILE" << EOF
# ZK Code Arena 备份配置
BACKUP_DIR="$DEFAULT_BACKUP_DIR"
PROJECT_PATH="$DEFAULT_PROJECT_PATH"
DAYS_TO_KEEP=$DEFAULT_DAYS_TO_KEEP
MONGODB_CONTAINER="zk-arena-mongo-dev"
NOTIFICATION_EMAIL=""
ENABLE_COMPRESSION=true
EXCLUDE_PATTERNS="*.log tmp/ *.tmp"
EOF
    echo "📝 已创建默认配置文件: $CONFIG_FILE"
}

# 显示帮助信息
show_help() {
    echo "🛠️  ZK Code Arena 备份管理器"
    echo "================================"
    echo ""
    echo "用法: $0 [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  backup          执行完整备份"
    echo "  backup-db       执行MongoDB备份"
    echo "  backup-all      执行所有类型备份"
    echo "  list            列出所有备份"
    echo "  restore         恢复数据"
    echo "  clean           清理旧备份"
    echo "  status          显示备份状态"
    echo "  config          配置备份选项"
    echo "  schedule        设置定时任务"
    echo "  help            显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 backup                    # 执行完整备份"
    echo "  $0 backup-db                 # 仅备份MongoDB"
    echo "  $0 list                      # 列出所有备份"
    echo "  $0 clean 3                   # 清理3天前的备份"
    echo "  $0 schedule daily 02:00      # 设置每日2点备份"
    echo ""
}

# 执行完整备份
do_full_backup() {
    echo "🚀 执行完整数据备份..."
    bash "$SCRIPT_DIR/backup_data.sh"
}

# 执行MongoDB备份
do_mongodb_backup() {
    echo "🚀 执行MongoDB数据备份..."
    bash "$SCRIPT_DIR/backup_mongodb.sh"
}

# 执行所有备份
do_all_backup() {
    echo "🚀 执行所有类型备份..."
    do_full_backup
    echo ""
    do_mongodb_backup
}

# 列出备份
list_backups() {
    echo "📁 备份文件列表"
    echo "================================"
    echo ""
    echo "📦 完整备份:"
    ls -lh "$BACKUP_DIR"/zk_arena_backup_*.tar.gz 2>/dev/null | tail -10 || echo "  无备份文件"
    echo ""
    echo "🗄️ MongoDB备份:"
    ls -lh "$BACKUP_DIR/mongodb"/mongodb_backup_*.tar.gz 2>/dev/null | tail -10 || echo "  无备份文件"
    echo ""
    
    # 统计信息
    FULL_COUNT=$(ls "$BACKUP_DIR"/zk_arena_backup_*.tar.gz 2>/dev/null | wc -l)
    MONGO_COUNT=$(ls "$BACKUP_DIR/mongodb"/mongodb_backup_*.tar.gz 2>/dev/null | wc -l)
    TOTAL_SIZE=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
    
    echo "📈 备份统计:"
    echo "  完整备份数量: $FULL_COUNT"
    echo "  MongoDB备份数量: $MONGO_COUNT"
    echo "  总占用空间: ${TOTAL_SIZE:-0}"
}

# 清理旧备份
clean_backups() {
    local days=${1:-$DAYS_TO_KEEP}
    echo "🧹 清理超过 $days 天的备份..."
    
    find "$BACKUP_DIR" -name "zk_arena_backup_*.tar.gz" -mtime +$days -print -delete
    find "$BACKUP_DIR/mongodb" -name "mongodb_backup_*.tar.gz" -mtime +$days -print -delete
    
    echo "✅ 清理完成"
}

# 显示备份状态
show_status() {
    echo "📊 备份系统状态"
    echo "================================"
    echo ""
    echo "📂 配置信息:"
    echo "  备份目录: $BACKUP_DIR"
    echo "  项目路径: $PROJECT_PATH"
    echo "  保留天数: $DAYS_TO_KEEP 天"
    echo "  MongoDB容器: $MONGODB_CONTAINER"
    echo ""
    
    # 检查容器状态
    if docker ps | grep -q "$MONGODB_CONTAINER"; then
        echo "✅ MongoDB容器运行正常"
    else
        echo "❌ MongoDB容器未运行"
    fi
    
    # 检查备份目录
    if [ -d "$BACKUP_DIR" ]; then
        echo "✅ 备份目录存在"
    else
        echo "⚠️  备份目录不存在，将会自动创建"
    fi
    
    # 最新备份信息
    LATEST_FULL=$(ls -t "$BACKUP_DIR"/zk_arena_backup_*.tar.gz 2>/dev/null | head -1)
    LATEST_MONGO=$(ls -t "$BACKUP_DIR/mongodb"/mongodb_backup_*.tar.gz 2>/dev/null | head -1)
    
    echo ""
    echo "📅 最新备份:"
    if [ -n "$LATEST_FULL" ]; then
        echo "  完整备份: $(basename "$LATEST_FULL") ($(stat -c %y "$LATEST_FULL" | cut -d' ' -f1,2 | cut -d'.' -f1))"
    else
        echo "  完整备份: 无"
    fi
    
    if [ -n "$LATEST_MONGO" ]; then
        echo "  MongoDB备份: $(basename "$LATEST_MONGO") ($(stat -c %y "$LATEST_MONGO" | cut -d' ' -f1,2 | cut -d'.' -f1))"
    else
        echo "  MongoDB备份: 无"
    fi
    
    # 检查定时任务
    echo ""
    echo "⏰ 定时任务:"
    crontab -l 2>/dev/null | grep -E "(backup|zk.*arena)" || echo "  未设置定时备份"
}

# 配置备份选项
configure_backup() {
    echo "⚙️  备份配置"
    echo "================================"
    echo ""
    echo "当前配置:"
    cat "$CONFIG_FILE"
    echo ""
    
    read -p "是否要编辑配置文件? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        ${EDITOR:-nano} "$CONFIG_FILE"
        echo "✅ 配置已更新"
    fi
}

# 设置定时任务
setup_schedule() {
    local frequency=$1
    local time=$2
    
    if [ -z "$frequency" ] || [ -z "$time" ]; then
        echo "📅 定时备份设置"
        echo "================================"
        echo ""
        echo "用法: $0 schedule <频率> <时间>"
        echo ""
        echo "频率选项:"
        echo "  daily    - 每日备份"
        echo "  weekly   - 每周备份"
        echo "  monthly  - 每月备份"
        echo ""
        echo "时间格式: HH:MM (24小时制)"
        echo ""
        echo "示例:"
        echo "  $0 schedule daily 02:00     # 每日凌晨2点"
        echo "  $0 schedule weekly 03:30    # 每周日凌晨3:30"
        echo "  $0 schedule monthly 01:00   # 每月1日凌晨1点"
        return 1
    fi
    
    # 解析时间
    local hour=$(echo "$time" | cut -d':' -f1)
    local minute=$(echo "$time" | cut -d':' -f2)
    
    # 构建cron表达式
    local cron_expr=""
    case $frequency in
        "daily")
            cron_expr="$minute $hour * * * $SCRIPT_DIR/backup_manager.sh backup-all >/dev/null 2>&1"
            ;;
        "weekly")
            cron_expr="$minute $hour * * 0 $SCRIPT_DIR/backup_manager.sh backup-all >/dev/null 2>&1"
            ;;
        "monthly")
            cron_expr="$minute $hour 1 * * $SCRIPT_DIR/backup_manager.sh backup-all >/dev/null 2>&1"
            ;;
        *)
            echo "❌ 不支持的频率: $frequency"
            return 1
            ;;
    esac
    
    # 添加到crontab
    (crontab -l 2>/dev/null | grep -v "backup_manager.sh"; echo "$cron_expr") | crontab -
    
    echo "✅ 定时任务已设置: $frequency $time"
    echo "📝 Cron表达式: $cron_expr"
}

# 主程序
main() {
    load_config
    
    case ${1:-help} in
        "backup")
            do_full_backup
            ;;
        "backup-db")
            do_mongodb_backup
            ;;
        "backup-all")
            do_all_backup
            ;;
        "list")
            list_backups
            ;;
        "restore")
            bash "$SCRIPT_DIR/restore_data.sh" "${@:2}"
            ;;
        "clean")
            clean_backups "$2"
            ;;
        "status")
            show_status
            ;;
        "config")
            configure_backup
            ;;
        "schedule")
            setup_schedule "$2" "$3"
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# 运行主程序
main "$@"
