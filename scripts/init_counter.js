// 连接数据库
const conn = new Mongo();
const db = conn.getDB('zk_code_arena');
print("Connected to MongoDB successfully.");

// 定义 async 函数包裹所有逻辑
async function init() {
    let initialValue = 1000;
    try {
        const maxProblem = await db.problems.findOne(
            {},
            { sort: { unique_id: -1 }, projection: { unique_id: 1 } }
        );

        if (maxProblem && maxProblem.unique_id) {
            initialValue = maxProblem.unique_id;
            print(`Found existing max unique_id: ${initialValue}`);
        } else {
            print(`No existing problems found. Starting from default: ${initialValue}`);
        }

        // 初始化计数器
        const result = await db.counters.updateOne(
            { name: 'problem_unique_id' },
            { $set: { sequence_value: initialValue } },
            { upsert: true }
        );

        if (result.upsertedCount > 0) {
            print("Counter initialized successfully.");
        } else if (result.modifiedCount > 0) {
            print("Counter updated successfully.");
        } else {
            print("Counter already exists.");
        }
    } catch (err) {
        print("Error:", err);
    }
}

// 执行 async 函数
init();

// 关键：等待异步操作完成（给10秒足够时间）
sleep(10000);
print("Script completed.");