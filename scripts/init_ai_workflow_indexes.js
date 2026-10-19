/*
 * MongoDB 索引初始化脚本
 * 用于 AI 工作流相关集合
 *
 * 执行方式：
 * mongo <database_name> < init_ai_workflow_indexes.js
 * 或在 MongoDB Shell 中：
 * use <database_name>
 * load("init_ai_workflow_indexes.js")
 */

print("========================================");
print("开始创建 AI 工作流索引...");
print("========================================");

// ============================================
// 1. ai_workflow_sessions 集合索引
// ============================================
print("\n[1/2] 创建 ai_workflow_sessions 索引...");

db.ai_workflow_sessions.createIndex(
    { "session_id": 1 },
    {
        unique: true,
        name: "idx_session_id_unique",
        background: true
    }
);
print("✓ 创建唯一索引: session_id");

db.ai_workflow_sessions.createIndex(
    { "user_id": 1, "created_at": -1 },
    {
        name: "idx_user_created",
        background: true
    }
);
print("✓ 创建复合索引: user_id + created_at (用于用户会话列表查询)");

db.ai_workflow_sessions.createIndex(
    { "status": 1 },
    {
        name: "idx_status",
        background: true
    }
);
print("✓ 创建索引: status (用于状态筛选)");

db.ai_workflow_sessions.createIndex(
    { "workflow_type": 1 },
    {
        name: "idx_workflow_type",
        background: true
    }
);
print("✓ 创建索引: workflow_type (用于工作流类型筛选)");

db.ai_workflow_sessions.createIndex(
    { "user_id": 1, "status": 1, "created_at": -1 },
    {
        name: "idx_user_status_created",
        background: true
    }
);
print("✓ 创建复合索引: user_id + status + created_at (用于带状态筛选的列表查询)");

db.ai_workflow_sessions.createIndex(
    { "created_at": 1 },
    {
        name: "idx_created_at",
        expireAfterSeconds: 2592000, // 30 天后自动删除
        background: true
    }
);
print("✓ 创建 TTL 索引: created_at (30天后自动清理)");

// ============================================
// 2. ai_workflow_checkpoints 集合索引
// ============================================
print("\n[2/2] 创建 ai_workflow_checkpoints 索引...");

db.ai_workflow_checkpoints.createIndex(
    { "session_id": 1, "step": 1 },
    {
        name: "idx_session_step",
        background: true
    }
);
print("✓ 创建复合索引: session_id + step (用于检查点查询)");

db.ai_workflow_checkpoints.createIndex(
    { "session_id": 1, "created_at": -1 },
    {
        name: "idx_session_created",
        background: true
    }
);
print("✓ 创建复合索引: session_id + created_at (用于获取最新检查点)");

db.ai_workflow_checkpoints.createIndex(
    { "created_at": 1 },
    {
        name: "idx_checkpoint_created_at",
        expireAfterSeconds: 2592000, // 30 天后自动删除
        background: true
    }
);
print("✓ 创建 TTL 索引: created_at (30天后自动清理)");

// ============================================
// 验证索引创建结果
// ============================================
print("\n========================================");
print("索引创建完成！验证结果：");
print("========================================");

print("\nai_workflow_sessions 索引列表：");
db.ai_workflow_sessions.getIndexes().forEach(function(index) {
    print("  - " + index.name + ": " + JSON.stringify(index.key));
});

print("\nai_workflow_checkpoints 索引列表：");
db.ai_workflow_checkpoints.getIndexes().forEach(function(index) {
    print("  - " + index.name + ": " + JSON.stringify(index.key));
});

print("\n========================================");
print("索引初始化完成！");
print("========================================");
