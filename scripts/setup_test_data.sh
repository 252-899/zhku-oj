#!/bin/bash

# 测试数据初始化脚本
# Author: omenkk7
# Date: 2024/10/26

echo "🚀 开始初始化测试数据..."

# 检查 MongoDB 容器是否运行
if ! docker ps | grep -q "mongo"; then
    echo "❌ MongoDB 容器未运行，请先启动 Docker Compose"
    echo "运行命令: docker-compose up -d"
    exit 1
fi

# 获取 MongoDB 容器名称
# 优先查找开发环境容器，然后查找生产环境容器
MONGO_CONTAINER=$(docker ps --format "{{.Names}}" | grep -E "(dev.*mongo|mongo.*dev)" | head -1)
if [ -z "$MONGO_CONTAINER" ]; then
    MONGO_CONTAINER=$(docker ps --format "{{.Names}}" | grep mongo | head -1)
fi

if [ -z "$MONGO_CONTAINER" ]; then
    echo "❌ 找不到 MongoDB 容器"
    exit 1
fi

echo "📦 找到 MongoDB 容器: $MONGO_CONTAINER"

# 检查数据库是否已有数据
USER_COUNT=$(docker exec $MONGO_CONTAINER mongosh zk_code_arena --quiet --eval "db.users.countDocuments()")

if [ "$USER_COUNT" -gt 1 ]; then
    echo "⚠️  数据库中已存在 $USER_COUNT 个用户"
    read -p "是否要清空现有数据并重新生成？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "🗑️  清空现有数据..."
        docker exec $MONGO_CONTAINER mongo zk_code_arena --eval "
            db.users.drop();
            db.problems.drop();
            db.test_cases.drop();
            db.submits.drop();
            db.courses.drop();
            db.clazzes.drop();
            db.tasks.drop();
            db.relations_users.drop();
            print('数据清空完成');
        "
    else
        echo "❌ 操作已取消"
        exit 0
    fi
fi

# 执行测试数据生成脚本
echo "📝 执行测试数据生成脚本..."
docker exec $MONGO_CONTAINER mongo zk_code_arena /docker-entrypoint-initdb.d/generate_test_data.js

# 验证数据生成结果
echo ""
echo "✅ 验证数据生成结果:"
docker exec $MONGO_CONTAINER mongo zk_code_arena --quiet --eval "
    print('👤 用户数量: ' + db.users.count());
    print('📚 题目数量: ' + db.problems.count());
    print('🧪 测试用例: ' + db.test_cases.count());
    print('💻 提交记录: ' + db.submits.count());
    print('📖 课程数量: ' + db.courses.count());
    print('👥 班级数量: ' + db.clazzes.count());
    print('📋 任务数量: ' + db.tasks.count());
"

echo ""
echo "🎉 测试数据初始化完成！"
echo ""
echo "🔑 默认登录账号:"
echo "管理员: admin / admin123"
echo "教师1: teacher1 / admin123"  
echo "教师2: teacher2 / admin123"
echo "学生: student1-student20 / admin123"
echo ""
echo "🌐 访问地址:"
echo "- API文档 (Scalar): http://localhost:8080/docs"
echo "- API文档 (Swagger): http://localhost:8080/swagger/index.html"
echo "- 应用首页: http://localhost:8080"
