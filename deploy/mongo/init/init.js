// MongoDB 初始化脚本
// 创建数据库和集合
db = db.getSiblingDB('zk_code_arena');

// 创建用户集合索引
db.users.createIndex({ "username": 1 }, { unique: true });
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "student_id": 1 }, { unique: true, sparse: true });

// 创建题目集合索引
db.problems.createIndex({ "title": "text", "description": "text" });
db.problems.createIndex({ "difficulty": 1 });
db.problems.createIndex({ "tags": 1 });
db.problems.createIndex({ "status": 1 });
db.problems.createIndex({ "is_public": 1 });
db.problems.createIndex({ "created_by": 1 });

// 创建标签集合索引
db.tags.createIndex({ "name": 1 }, { unique: true });

// 创建提交集合索引
db.submits.createIndex({ "user_id": 1 });
db.submits.createIndex({ "problem_id": 1 });
db.submits.createIndex({ "status": 1 });
db.submits.createIndex({ "created_at": -1 });
db.submits.createIndex({ "user_id": 1, "problem_id": 1 });
// 用户题目状态查询优化索引 (用于快速查找用户对各题目的最新状态)
db.submits.createIndex({ "user_id": 1, "problem_id": 1, "status": 1 }, { name: "idx_user_problem_status" });

// 创建测试用例集合索引
db.test_cases.createIndex({ "problem_id": 1 });
db.test_cases.createIndex({ "is_sample": 1 });
db.test_cases.createIndex({ "created_at": -1 });

// 测试用例集合字段说明
/*
测试用例 (test_cases) 数据结构:
{
  "_id": ObjectId,                // 测试用例ID
  "problem_id": ObjectId,         // 关联题目ID (必填)
  "input": String,                // 输入数据 (必填)
  "output": String,               // 期望输出 (必填)
  "is_sample": Boolean,           // 是否为示例用例 (必填, 默认false)
  "time_limit": Number,           // 可选: 时间限制(ms), 优先级高于题目默认配置
  "memory_limit": Number,         // 可选: 内存限制(MB), 优先级高于题目默认配置
  "score": Number,                // 可选: 用例分数, 用于部分分支持 (默认0)
  "created_at": Date,             // 创建时间
  "updated_at": Date              // 更新时间
}

索引说明:
- problem_id: 用于快速查询某个题目的所有测试用例
- is_sample: 用于区分示例用例和隐藏用例
- created_at: 用于按时间排序
*/

// 插入默认管理员用户
// 默认密码：admin123 (bcrypt加密)
db.users.insertOne({
    "username": "admin",
    "password": "$2a$10$vI8aWBnW3fID.ZQ4/zo1G.q1lRps.9cGLcZEiGDMVr5yUP1KUOYTa",
    "email": "admin@zk.edu.cn",
    "real_name": "系统管理员",
    "student_id": "ADMIN001",
    "role": "admin",
    "is_active": true,
    "created_at": new Date(),
    "updated_at": new Date()
});

// ============================================
// AI 工作流集合索引
// ============================================
print("创建 AI 工作流索引...");

// ai_workflow_sessions 集合索引
db.ai_workflow_sessions.createIndex(
    { "session_id": 1 },
    { unique: true, name: "idx_session_id_unique", background: true }
);
db.ai_workflow_sessions.createIndex(
    { "user_id": 1, "created_at": -1 },
    { name: "idx_user_created", background: true }
);
db.ai_workflow_sessions.createIndex(
    { "status": 1 },
    { name: "idx_status", background: true }
);
db.ai_workflow_sessions.createIndex(
    { "workflow_type": 1 },
    { name: "idx_workflow_type", background: true }
);
db.ai_workflow_sessions.createIndex(
    { "user_id": 1, "status": 1, "created_at": -1 },
    { name: "idx_user_status_created", background: true }
);
db.ai_workflow_sessions.createIndex(
    { "created_at": 1 },
    { name: "idx_created_at", expireAfterSeconds: 2592000, background: true } // 30天自动清理
);

// ai_workflow_checkpoints 集合索引
db.ai_workflow_checkpoints.createIndex(
    { "session_id": 1, "step": 1 },
    { name: "idx_session_step", background: true }
);
db.ai_workflow_checkpoints.createIndex(
    { "session_id": 1, "created_at": -1 },
    { name: "idx_session_created", background: true }
);
db.ai_workflow_checkpoints.createIndex(
    { "created_at": 1 },
    { name: "idx_checkpoint_created_at", expireAfterSeconds: 2592000, background: true } // 30天自动清理
);

print("数据库初始化完成（包含 AI 工作流索引）");
