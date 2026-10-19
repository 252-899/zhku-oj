# AI 工作流 API 快速参考

## 🚀 快速开始

### Base URL
```
http://localhost:8080/api/v1/ai/workflow
```

### 认证
```http
Authorization: Bearer <jwt_token>
```

---

## 📡 接口列表

### 1. 启动工作流
```http
POST /start
```

**请求**:
```json
{
  "topic": "二叉树遍历",
  "difficulty": "medium",
  "tags": ["树", "递归"],
  "language": "cpp",
  "testcase_count": 10
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "session_id": "uuid",
    "status": "pending"
  }
}
```

---

### 2. 查询状态
```http
GET /:session_id
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "status": "running",
    "progress": 50,
    "current_step": "solution_gen",
    "result": null
  }
}
```

---

### 3. SSE 流式监听
```http
GET /:session_id/stream
```

**前端代码**:
```javascript
const es = new EventSource(`/api/v1/ai/workflow/${sessionId}/stream`);

es.addEventListener('progress', (e) => {
  const { progress, log } = JSON.parse(e.data);
  console.log(`${progress}%: ${log.message}`);
});

es.addEventListener('completed', (e) => {
  const { result } = JSON.parse(e.data);
  console.log('完成！', result);
  es.close();
});
```

---

### 4. 获取列表
```http
GET /list?page=1&page_size=10&status=completed
```

---

### 5. 重试
```http
POST /:session_id/retry
```

---

### 6. 删除
```http
DELETE /:session_id
```

---

## 📊 工作流步骤

| 步骤 | 进度 | 说明 |
|------|------|------|
| problem_gen | 5-25% | 生成题目描述 |
| testcase_gen | 25-40% | 生成测试用例 |
| solution_gen | 40-60% | 生成标准答案 |
| run_testcases | 60-75% | 运行标程 |
| verify | 75-95% | 验证测试用例 |
| formatter | 95-100% | 格式化结果 |

---

## 🎯 完整结果结构

```typescript
interface WorkflowResult {
  // 题目信息
  title: string;
  description: string;
  input_desc: string;
  output_desc: string;
  sample_input: string;
  sample_output: string;
  hint: string;
  tags: string[];
  difficulty: string;
  time_limit: number;
  memory_limit: number;

  // 标准答案
  std_code: string;
  std_language: string;

  // 测试用例
  test_cases: Array<{
    input: string;
    output: string;
    is_sample: boolean;
    description: string;
  }>;

  // 验证信息
  verify_passed: boolean;
  verify_logs: string;
}
```

---

## ⚠️ 注意事项

1. **权限**: 仅老师和管理员可访问
2. **执行时间**: 30-60 秒
3. **重试限制**: 最多 3 次
4. **数据保留**: 30 天后自动删除

---

## 🔗 相关文档

- [完整 API 文档](./API-AI-WORKFLOW.md)
- [数据库索引脚本](../scripts/init_ai_workflow_indexes.js)
