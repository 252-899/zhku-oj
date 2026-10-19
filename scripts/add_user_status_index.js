/*
 * 添加用户题目状态查询优化索引脚本
 * 用于在已有数据库环境中添加新索引
 * 
 * 使用方法:
 * 1. Docker环境: docker exec <mongo-container> mongo zk_code_arena /scripts/add_user_status_index.js
 * 2. 本地环境: mongo zk_code_arena add_user_status_index.js
 */

// 连接到数据库
db = db.getSiblingDB('zk_code_arena');

print("开始添加用户题目状态查询优化索引...");

try {
    // 检查索引是否已存在
    const existingIndexes = db.submits.getIndexes();
    const indexExists = existingIndexes.some(index => index.name === "idx_user_problem_status");
    
    if (indexExists) {
        print("索引 idx_user_problem_status 已存在，跳过创建");
    } else {
        // 创建用户题目状态查询优化索引
        db.submits.createIndex(
            { "user_id": 1, "problem_id": 1, "status": 1 }, 
            { 
                name: "idx_user_problem_status",
                background: true  // 后台创建，不阻塞其他操作
            }
        );
        print("✅ 成功创建索引 idx_user_problem_status");
    }
    
    // 验证索引创建
    const indexes = db.submits.getIndexes();
    const newIndex = indexes.find(index => index.name === "idx_user_problem_status");
    
    if (newIndex) {
        print("✅ 索引验证成功:");
        print("   - 名称: " + newIndex.name);
        print("   - 字段: " + JSON.stringify(newIndex.key));
        print("   - 选项: " + JSON.stringify(newIndex));
    } else {
        print("❌ 索引验证失败: 未找到 idx_user_problem_status 索引");
    }
    
    // 显示submits集合的所有索引
    print("\n📋 submits 集合当前所有索引:");
    db.submits.getIndexes().forEach(function(index, i) {
        print("   " + (i + 1) + ". " + index.name + " - " + JSON.stringify(index.key));
    });
    
    print("\n🎉 索引添加任务完成！");
    
} catch (error) {
    print("❌ 创建索引时发生错误: " + error.message);
    print("错误详情: " + JSON.stringify(error));
}
