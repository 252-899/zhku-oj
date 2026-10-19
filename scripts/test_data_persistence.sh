#!/bin/bash

echo "🧪 数据持久化测试脚本"
echo "========================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}📋 测试步骤：${NC}"
echo "1. 启动服务并插入测试数据"
echo "2. 停止并删除容器"
echo "3. 重新启动服务"
echo "4. 检查数据是否还在"
echo ""

# 检查 Docker 是否运行
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker 未运行，请先启动 Docker${NC}"
    exit 1
fi

echo -e "${GREEN}🚀 步骤1: 启动服务${NC}"
docker-compose up -d mongo

echo "等待 MongoDB 启动..."
sleep 10

echo -e "${GREEN}📝 步骤2: 插入测试数据${NC}"
docker exec zk-arena-mongo mongo zk_code_arena --eval "
db.test_persistence.insertOne({
    message: '这是持久化测试数据',
    timestamp: new Date(),
    test_id: 'persistence_test_001'
});
print('✅ 测试数据已插入');
"

echo -e "${GREEN}🔍 验证数据插入${NC}"
docker exec zk-arena-mongo mongo zk_code_arena --eval "
var count = db.test_persistence.count();
print('📊 测试数据条数: ' + count);
if (count > 0) {
    var doc = db.test_persistence.findOne();
    print('📄 数据内容: ' + JSON.stringify(doc, null, 2));
}
"

echo ""
echo -e "${YELLOW}⚠️  步骤3: 停止并删除容器${NC}"
echo "这将删除容器，但数据应该保留在主机目录中..."
docker-compose down

echo ""
echo -e "${GREEN}🔄 步骤4: 重新启动服务${NC}"
docker-compose up -d mongo

echo "等待 MongoDB 重新启动..."
sleep 10

echo -e "${GREEN}🔍 步骤5: 检查数据是否还在${NC}"
docker exec zk-arena-mongo mongo zk_code_arena --eval "
var count = db.test_persistence.count();
print('📊 重启后测试数据条数: ' + count);
if (count > 0) {
    print('✅ 数据持久化成功！容器删除后数据仍然存在');
    var doc = db.test_persistence.findOne();
    print('📄 数据内容: ' + JSON.stringify(doc, null, 2));
} else {
    print('❌ 数据丢失！请检查挂载配置');
}
"

echo ""
echo -e "${GREEN}🗂️  检查主机目录${NC}"
if [ -d "./data/mongo" ]; then
    echo "✅ 主机数据目录存在: ./data/mongo"
    echo "📁 目录大小: $(du -sh ./data/mongo 2>/dev/null || echo '无法计算')"
    echo "📄 文件列表:"
    ls -la ./data/mongo/ 2>/dev/null || echo "目录为空或无权限访问"
else
    echo "❌ 主机数据目录不存在: ./data/mongo"
fi

echo ""
echo -e "${GREEN}🧹 清理测试数据${NC}"
docker exec zk-arena-mongo mongo zk_code_arena --eval "
db.test_persistence.drop();
print('🗑️  测试数据已清理');
"

echo ""
echo -e "${GREEN}🎉 测试完成！${NC}"
echo ""
echo -e "${YELLOW}📝 总结：${NC}"
echo "- 使用主机目录挂载 (./data/mongo:/data/db)"
echo "- 容器删除后数据仍然保留在主机上"
echo "- 可以直接访问 ./data/mongo 目录查看数据文件"
echo "- 便于备份、迁移和维护"
