# AI 题目生成工作流 API 文档

## 📋 目录
- [接口概览](#接口概览)
- [认证说明](#认证说明)
- [接口详情](#接口详情)
- [数据结构](#数据结构)
- [错误码](#错误码)
- [使用示例](#使用示例)

---

## 接口概览

### Base URL
```
http://localhost:8080/api/v1/ai/workflow
```

### 接口列表

| 方法 | 路径 | 功能 | 权限 |
|------|------|------|------|
| POST | `/start` | 启动工作流 | 老师/管理员 |
| GET | `/:session_id` | 获取会话信息 | 老师/管理员 |
| GET | `/:session_id/stream` | SSE 流式输出 | 老师/管理员 |
| GET | `/list` | 获取会话列表 | 老师/管理员 |
| POST | `/:session_id/retry` | 重试工作流 | 老师/管理员 |
| DELETE | `/:session_id` | 删除会话 | 老师/管理员 |

---

## 认证说明

### 请求头
所有接口都需要在请求头中携带 JWT Token：

```http
Authorization: Bearer <your_jwt_token>
```

### 权限要求
- **角色限制**: 仅 `teacher`（老师）和 `admin`（管理员）可访问
- **学生无权限**: 返回 `403 Forbidden`

---

## 接口详情

### 1. 启动工作流

**接口**: `POST /api/v1/ai/workflow/start`

**功能**: 创建新的题目生成工作流会话

**请求体**:
```json
{
  "topic": "二叉树遍历",
  "difficulty": "medium",
  "tags": ["树", "递归", "DFS"],
  "extra_req": "要求使用递归实现，并提供迭代解法作为对比",
  "language": "cpp",
  "testcase_count": 10,
  "time_limit": 1000,
  "memory_limit": 256
}
```

**请求参数说明**:

| 字段 | 类型 | 必填 | 说明 | 示例 | 默认值 |
|------|------|------|------|------|--------|
| topic | string | ✅ | 题目主题 | "二叉树遍历" | - |
| difficulty | string | ✅ | 难度等级 | "easy"/"medium"/"hard" | - |
| tags | []string | ❌ | 题目标签 | ["树", "递归"] | [] |
| extra_req | string | ❌ | 额外要求 | "要求时间复杂度 O(n)" | "" |
| language | string | ❌ | 标程语言 | "cpp"/"java"/"python"/"go"/"c" | "cpp" |
| testcase_count | int | ❌ | 测试用例数量 | 10 | 10 |
| time_limit | int | ❌ | 时间限制(ms) | 1000 | 1000 |
| memory_limit | int | ❌ | 内存限制(MB) | 256 | 256 |

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "pending",
    "message": "工作流已启动，请通过 session_id 查询进度"
  }
}
```

**错误响应**:
```json
{
  "code": 503,
  "message": "AI 服务不可用，请检查 LLM 配置",
  "data": null
}
```

---

### 2. 获取会话信息

**接口**: `GET /api/v1/ai/workflow/:session_id`

**功能**: 查询工作流会话的详细状态和结果

**路径参数**:
- `session_id`: 会话 ID（UUID）

**响应示例（运行中）**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "507f1f77bcf86cd799439011",
    "workflow_type": "oj_problem",
    "status": "running",
    "current_step": "solution_gen",
    "progress": 50,
    "input": {
      "topic": "二叉树遍历",
      "difficulty": "medium",
      "tags": ["树", "递归"],
      "language": "cpp",
      "testcase_count": 10
    },
    "result": null,
    "logs": [
      {
        "step": "problem_gen",
        "status": "success",
        "message": "题目描述生成完成",
        "timestamp": "2026-03-06T14:00:00Z"
      },
      {
        "step": "solution_gen",
        "status": "running",
        "message": "正在生成标准答案...",
        "timestamp": "2026-03-06T14:02:00Z"
      }
    ],
    "error_msg": "",
    "retry_count": 0,
    "created_at": "2026-03-06T14:00:00Z",
    "updated_at": "2026-03-06T14:02:00Z",
    "completed_at": null
  }
}
```

**响应示例（已完成）**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "progress": 100,
    "result": {
      "title": "二叉树的中序遍历",
      "description": "给定一个二叉树的根节点 root，返回它的中序遍历结果。\n\n中序遍历是按照访问左子树——根节点——右子树的方式遍历这棵树。",
      "input_desc": "输入：root = [1,null,2,3]\n其中数字表示节点值，null 表示空节点",
      "output_desc": "输出：[1,3,2]\n返回中序遍历的节点值数组",
      "sample_input": "[1,null,2,3]",
      "sample_output": "[1,3,2]",
      "hint": "可以使用递归或迭代方法实现。递归方法更简洁，迭代方法需要使用栈。",
      "tags": ["树", "递归", "DFS"],
      "difficulty": "medium",
      "time_limit": 1000,
      "memory_limit": 256,
      "std_code": "class Solution {\npublic:\n    vector<int> inorderTraversal(TreeNode* root) {\n        vector<int> result;\n        inorder(root, result);\n        return result;\n    }\n    \nprivate:\n    void inorder(TreeNode* node, vector<int>& result) {\n        if (!node) return;\n        inorder(node->left, result);\n        result.push_back(node->val);\n        inorder(node->right, result);\n    }\n};",
      "std_language": "cpp",
      "test_cases": [
        {
          "input": "[1,null,2,3]",
          "output": "[1,3,2]",
          "is_sample": true,
          "description": "基本测试用例"
        },
        {
          "input": "[]",
          "output": "[]",
          "is_sample": false,
          "description": "空树"
        },
        {
          "input": "[1]",
          "output": "[1]",
          "is_sample": false,
          "description": "单节点"
        }
      ],
      "verify_passed": true,
      "verify_logs": "所有测试用例验证通过"
    },
    "completed_at": "2026-03-06T14:05:00Z"
  }
}
```

---

### 3. SSE 流式输出

**接口**: `GET /api/v1/ai/workflow/:session_id/stream`

**功能**: 实时推送工作流执行进度（Server-Sent Events）

**响应格式**: `text/event-stream`

**事件类型**:

#### 连接建立
```
event: connected
data: {"session_id":"550e8400-e29b-41d4-a716-446655440000","message":"已连接到工作流事件流"}
```

#### 进度更新
```
event: progress
data: {"step":"problem_gen","progress":20,"log":{"step":"problem_gen","status":"running","message":"正在生成题目描述...","timestamp":"2026-03-06T14:00:00Z"}}
```

#### 工作流完成
```
event: completed
data: {"session_id":"550e8400-e29b-41d4-a716-446655440000","result":{...}}
```

#### 错误事件
```
event: error
data: {"error":"生成标程失败：LLM 返回格式错误"}
```

#### 心跳（每 30 秒）
```
event: heartbeat
data: {"timestamp":1709726400}
```

**前端使用示例**:
```javascript
const eventSource = new EventSource(
  `http://localhost:8080/api/v1/ai/workflow/${sessionId}/stream`,
  {
    headers: {
      'Authorization': 'Bearer ' + token
    }
  }
);

eventSource.addEventListener('progress', (e) => {
  const data = JSON.parse(e.data);
  console.log(`${data.progress}%: ${data.log.message}`);
});

eventSource.addEventListener('completed', (e) => {
  const data = JSON.parse(e.data);
  console.log('完成！', data.result);
  eventSource.close();
});
```

---

### 4. 获取会话列表

**接口**: `GET /api/v1/ai/workflow/list`

**查询参数**:

| 参数 | 类型 | 必填 | 说明 | 默认值 |
|------|------|------|------|--------|
| page | int | ❌ | 页码（从 1 开始） | 1 |
| page_size | int | ❌ | 每页数量 | 20 |
| status | string | ❌ | 状态筛选 | 全部 |

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 50,
    "page": 1,
    "page_size": 10,
    "sessions": [
      {
        "session_id": "550e8400-e29b-41d4-a716-446655440000",
        "workflow_type": "oj_problem",
        "status": "completed",
        "progress": 100,
        "input": {
          "topic": "二叉树遍历",
          "difficulty": "medium"
        },
        "created_at": "2026-03-06T14:00:00Z",
        "completed_at": "2026-03-06T14:05:00Z"
      }
    ]
  }
}
```

---

### 5. 重试工作流

**接口**: `POST /api/v1/ai/workflow/:session_id/retry`

**功能**: 重新执行失败的工作流（最多 3 次）

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "retrying",
    "retry_count": 1
  }
}
```

---

### 6. 删除会话

**接口**: `DELETE /api/v1/ai/workflow/:session_id`

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": null
}
```

---

## 数据结构

### WorkflowStatus（状态）

| 状态 | 说明 |
|------|------|
| pending | 等待开始 |
| running | 运行中 |
| completed | 已完成 |
| failed | 失败 |
| retrying | 重试中 |

### WorkflowStep（步骤）

| 步骤 | 说明 | 进度 |
|------|------|------|
| problem_gen | 生成题目描述 | 5-25% |
| testcase_gen | 生成测试用例 | 25-40% |
| solution_gen | 生成标准答案 | 40-60% |
| run_testcases | 运行标程 | 60-75% |
| verify | 验证测试用例 | 75-95% |
| formatter | 格式化结果 | 95-100% |

---

## 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 503 | AI 服务不可用 |
| 500 | 服务器错误 |

---

## 使用示例

### React 完整示例

```jsx
import { useState, useEffect } from 'react';

function AIWorkflowGenerator() {
  const [sessionId, setSessionId] = useState(null);
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState('');
  const [result, setResult] = useState(null);

  // 启动工作流
  const handleStart = async (formData) => {
    const response = await fetch('/api/v1/ai/workflow/start', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(formData)
    });

    const data = await response.json();
    setSessionId(data.data.session_id);
  };

  // 监听进度
  useEffect(() => {
    if (!sessionId) return;

    const eventSource = new EventSource(
      `/api/v1/ai/workflow/${sessionId}/stream`,
      {
        headers: { 'Authorization': `Bearer ${token}` }
      }
    );

    eventSource.addEventListener('progress', (e) => {
      const data = JSON.parse(e.data);
      setProgress(data.progress);
      setStatus(data.log.message);
    });

    eventSource.addEventListener('completed', (e) => {
      const data = JSON.parse(e.data);
      setResult(data.result);
      eventSource.close();
    });

    return () => eventSource.close();
  }, [sessionId]);

  return (
    <div>
      {!sessionId && <WorkflowForm onSubmit={handleStart} />}
      {sessionId && !result && (
        <ProgressBar progress={progress} status={status} />
      )}
      {result && <ResultDisplay result={result} />}
    </div>
  );
}
```

---

**文档版本**: v1.0.0
**最后更新**: 2026-03-06
**维护者**: AI Assistant
