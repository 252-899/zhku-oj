// 插入测试数据脚本
// 连接到远程数据库并插入测试题目和测试用例

// 切换到目标数据库
db = db.getSiblingDB('zk_code_arena');

print("开始插入测试数据...");

// 1. 插入测试题目
var testProblem = {
    "_id": ObjectId(),
    "title": "两数之和",
    "description": "给定一个整数数组 nums 和一个整数目标值 target，请你在该数组中找出和为目标值 target 的那两个整数，并返回它们的数组下标。\n\n你可以假设每种输入只会对应一个答案。但是，数组中同一个元素在答案里不能重复出现。\n\n你可以按任意顺序返回答案。",
    "input": "第一行包含一个整数 n，表示数组长度。\n第二行包含 n 个整数，表示数组 nums。\n第三行包含一个整数 target。",
    "output": "输出两个整数，表示和为 target 的两个数的下标（从0开始）。",
    "sample_input": "4\n2 7 11 15\n9",
    "sample_output": "0 1",
    "hint": "可以使用哈希表来优化时间复杂度。",
    "source": "LeetCode",
    "author": "LeetCode",
    "difficulty": "easy",
    "time_limit": 1000,
    "memory_limit": 256,
    "tags": ["数组", "哈希表", "简单"],
    "status": "published",
    "is_public": true,
    "ac_count": 0,
    "submit_count": 0,
    "created_by": ObjectId(),
    "created_at": new Date(),
    "updated_at": new Date()
};

var result = db.problems.insertOne(testProblem);
print("插入题目成功，ID: " + result.insertedId);

// 2. 插入测试用例
var testCases = [
    {
        "_id": ObjectId(),
        "problem_id": testProblem._id,
        "input": "4\n2 7 11 15\n9",
        "output": "0 1",
        "is_sample": true,
        "time_limit": 1000,
        "memory_limit": 256,
        "score": 10,
        "created_at": new Date()
    },
    {
        "_id": ObjectId(),
        "problem_id": testProblem._id,
        "input": "3\n3 2 4\n6",
        "output": "1 2",
        "is_sample": true,
        "time_limit": 1000,
        "memory_limit": 256,
        "score": 10,
        "created_at": new Date()
    },
    {
        "_id": ObjectId(),
        "problem_id": testProblem._id,
        "input": "2\n3 3\n6",
        "output": "0 1",
        "is_sample": false,
        "time_limit": 1000,
        "memory_limit": 256,
        "score": 20,
        "created_at": new Date()
    },
    {
        "_id": ObjectId(),
        "problem_id": testProblem._id,
        "input": "5\n1 2 3 4 5\n8",
        "output": "2 4",
        "is_sample": false,
        "time_limit": 1000,
        "memory_limit": 256,
        "score": 20,
        "created_at": new Date()
    }
];

var testCaseResult = db.test_cases.insertMany(testCases);
print("插入测试用例成功，数量: " + testCaseResult.insertedIds.length);

// 3. 插入更多测试题目
var moreProblems = [
    {
        "_id": ObjectId(),
        "title": "反转链表",
        "description": "给你单链表的头节点 head ，请你反转链表，并返回反转后的链表。",
        "input": "第一行包含一个整数 n，表示链表长度。\n第二行包含 n 个整数，表示链表的值。",
        "output": "输出反转后的链表。",
        "sample_input": "5\n1 2 3 4 5",
        "sample_output": "5 4 3 2 1",
        "hint": "可以使用迭代或递归的方法。",
        "source": "LeetCode",
        "author": "LeetCode",
        "difficulty": "easy",
        "time_limit": 1000,
        "memory_limit": 256,
        "tags": ["链表", "递归", "简单"],
        "status": "published",
        "is_public": true,
        "ac_count": 0,
        "submit_count": 0,
        "created_by": ObjectId(),
        "created_at": new Date(),
        "updated_at": new Date()
    },
    {
        "_id": ObjectId(),
        "title": "最长公共子序列",
        "description": "给定两个字符串 text1 和 text2，返回这两个字符串的最长公共子序列的长度。如果不存在公共子序列，返回 0。",
        "input": "第一行包含字符串 text1。\n第二行包含字符串 text2。",
        "output": "输出最长公共子序列的长度。",
        "sample_input": "abcde\nace",
        "sample_output": "3",
        "hint": "使用动态规划解决。",
        "source": "LeetCode",
        "author": "LeetCode",
        "difficulty": "medium",
        "time_limit": 2000,
        "memory_limit": 512,
        "tags": ["动态规划", "字符串", "中等"],
        "status": "published",
        "is_public": true,
        "ac_count": 0,
        "submit_count": 0,
        "created_by": ObjectId(),
        "created_at": new Date(),
        "updated_at": new Date()
    }
];

var moreProblemsResult = db.problems.insertMany(moreProblems);
print("插入更多题目成功，数量: " + moreProblemsResult.insertedIds.length);

// 4. 验证数据
print("\n=== 数据验证 ===");
print("题目总数: " + db.problems.countDocuments({}));
print("测试用例总数: " + db.test_cases.countDocuments({}));
print("公开题目数: " + db.problems.countDocuments({"is_public": true, "status": "published"}));

print("\n测试数据插入完成！");
