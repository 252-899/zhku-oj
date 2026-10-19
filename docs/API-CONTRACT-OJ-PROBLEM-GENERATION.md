# OJ 题目生成工作流 - API 契约文档

> **版本**: v1.0
> **状态**: ✅ 已锁定，前端可按此开发
> **更新时间**: 2026-03-06

---

## 📌 契约说明

本文档定义了 OJ 题目生成工作流的 API 契约，**前后端必须严格遵守**。

### 契约保证

✅ **接口路径不变** - 所有 URL 路径已确定
✅ **请求格式不变** - 请求体结构已锁定
✅ **响应格式不变** - 响应体结构已锁定
✅ **状态码不变** - HTTP 状态码已确定
✅ **字段类型不变** - 所有字段类型已确定

### 后端可调整范围

- ⚠️ 内部实现逻辑（不影响接口）
- ⚠️ 性能优化
- ⚠️ Bug 修复

### 后端不可调整

- ❌ 接口路径
- ❌ 请求/响应字段名
- ❌ 字段类型
- ❌ 必填/可选属性

---

## 🔐 通用规范

### 认证方式

```http
Authorization: Bearer {JWT_TOKEN}
```

### 响应格式

所有接口统一使用以下响应格式：

```typescript
interface Response<T> {
  code: number;      // HTTP 状态码
  message: string;   // 响应消息
  data?: T;          // 响应数据（可选）
}
```

### 错误响应

```typescript
interface ErrorResponse {
  code: number;
  message: string;
}
```

常见错误码：
- `400`: 请求参数错误
- `401`: 未授权（Token 无效或过期）
- `403`: 权限不足
- `404`: 资源不存在
- `500`: 服务器内部错误
- `503`: 服务不可用

---

## 📡 API 接口契约

### 1. 启动工作流

#### 基本信息

```
POST /api/v1/ai/workflow/start
```

#### 请求头

```http
Authorization: Bearer {token}
Content-Type: application/json
```

#### 请求体

```typescript
interface StartWorkflowRequest {
  workflow_type: "oj_problem_generation";  // 固定值
  input: {
    // 必填字段
    topic: string;                          // 题目主题
    difficulty: "easy" | "medium" | "hard"; // 难度

    // 可选字段
    tags?: string[];                        // 标签数组
    extra_req?: string;                     // 额外要求
    language?: "cpp" | "java" | "python" | "go" | "c";  // 标程语言，默认 cpp
    testcase_count?: number;                // 测试用例数量，默认 10，范围 1-50
    time_limit?: number;                    // 时间限制(ms)，默认 1000，范围 100-10000
    memory_limit?: number;                  // 内存限制(MB)，默认 256，范围 32-1024
  };
  config?: {
    timeout?: number;                       // 超时时间(秒)
    retry_on_failure?: boolean;             // 失败时是否重试
    max_retries?: number;                   // 最大重试次数
    save_checkpoints?: boolean;             // 是否保存检查点
  };
  metadata?: Record<string, any>;           // 元数据
}
```

#### 请求示例

```json
{
  "workflow_type": "oj_problem_generation",
  "input": {
    "topic": "二叉树遍历",
    "difficulty": "medium",
    "tags": ["树", "递归", "DFS"],
    "extra_req": "要求时间复杂度 O(n)",
    "language": "cpp",
    "testcase_count": 10,
    "time_limit": 1000,
    "memory_limit": 256
  }
}
```

#### 响应体

```typescript
interface StartWorkflowResponse {
  code: 200;
  message: "success";
  data: {
    session_id: string;  // 会话 ID（UUID 格式）
    status: "pending";   // 初始状态
    message: string;     // 提示信息
  };
}
```

#### 响应示例

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

---

### 2. 实时进度推送 (SSE)

#### 基本信息

```
GET /api/v1/ai/workflow/:session_id/stream
```

#### 请求头

```http
Authorization: Bearer {token}
Accept: text/event-stream
```

#### SSE 事件格式

所有事件统一格式：

```typescript
interface WorkflowEvent {
  session_id: string;
  type: "progress" | "review_required" | "completed" | "error" | "review_approved" | "review_rejected";
  data: Record<string, any>;
  timestamp: string;  // ISO 8601 格式
}
```

#### 事件类型 1: progress（进度更新）

```typescript
interface ProgressEvent {
  session_id: string;
  type: "progress";
  data: {
    step: string;      // 当前步骤名称
    progress: number;  // 进度百分比 (0-100)
    log: {
      step: string;
      status: "running" | "success" | "failed";
      message: string;
      timestamp: string;
    };
  };
  timestamp: string;
}
```

**示例**:
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "progress",
  "data": {
    "step": "problem_gen",
    "progress": 10,
    "log": {
      "step": "problem_gen",
      "status": "running",
      "message": "正在生成题目描述",
      "timestamp": "2026-03-06T10:00:00Z"
    }
  },
  "timestamp": "2026-03-06T10:00:00Z"
}
```

#### 事件类型 2: review_required（需要审核）

```typescript
interface ReviewRequiredEvent {
  session_id: string;
  type: "review_required";
  data: {
    message: string;
    draft_result: OJProblemWorkflowResult;  // 草稿结果
  };
  timestamp: string;
}
```

**示例**:
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "review_required",
  "data": {
    "message": "human review required",
    "draft_result": {
      "title": "二叉树的中序遍历",
      "description": "...",
      "test_cases": [...]
    }
  },
  "timestamp": "2026-03-06T10:05:00Z"
}
```

#### 事件类型 3: completed（完成）

```typescript
interface CompletedEvent {
  session_id: string;
  type: "completed";
  data: {
    session_id: string;
    result: OJProblemWorkflowResult;
  };
  timestamp: string;
}
```

#### 事件类型 4: error（错误）

```typescript
interface ErrorEvent {
  session_id: string;
  type: "error";
  data: {
    error: string;  // 错误信息
  };
  timestamp: string;
}
```

#### 事件类型 5: review_approved（审核批准）

```typescript
interface ReviewApprovedEvent {
  session_id: string;
  type: "review_approved";
  data: {
    reviewer_id: string;
    comment: string;
  };
  timestamp: string;
}
```

#### 事件类型 6: review_rejected（审核拒绝）

```typescript
interface ReviewRejectedEvent {
  session_id: string;
  type: "review_rejected";
  data: {
    reviewer_id: string;
    comment: string;
    retry_count: number;  // 当前重试次数
  };
  timestamp: string;
}
```

---

### 3. 查询会话信息

#### 基本信息

```
GET /api/v1/ai/workflow/:session_id
```

#### 请求头

```http
Authorization: Bearer {token}
```

#### 响应体

```typescript
interface GetSessionResponse {
  code: 200;
  message: "success";
  data: AIWorkflowSession;
}

interface AIWorkflowSession {
  id: string;                    // MongoDB ObjectID
  session_id: string;            // 会话 ID
  user_id: string;               // 用户 ID
  workflow_type: string;         // 工作流类型
  status: "pending" | "running" | "pending_review" | "completed" | "failed";
  current_step: string;          // 当前步骤
  progress: number;              // 进度 (0-100)

  input: Record<string, any>;    // 输入参数
  result?: OJProblemWorkflowResult;      // 最终结果（完成后）
  draft_result?: OJProblemWorkflowResult; // 草稿结果（待审核时）

  logs: WorkflowLog[];           // 执行日志
  error_msg?: string;            // 错误信息
  retry_count: number;           // 重试次数
  review_history?: ReviewRecord[]; // 审核历史

  created_at: string;            // 创建时间
  updated_at: string;            // 更新时间
  completed_at?: string;         // 完成时间
}

interface WorkflowLog {
  step: string;
  status: string;
  message: string;
  timestamp: string;
}

interface ReviewRecord {
  reviewer_id: string;
  decision: "approve" | "reject";
  comment: string;
  reviewed_at: string;
}
```

#### 响应示例

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "65f1234567890abcdef12345",
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "65f1234567890abcdef12346",
    "workflow_type": "oj_problem_generation",
    "status": "pending_review",
    "current_step": "human_review",
    "progress": 90,
    "input": {
      "topic": "二叉树遍历",
      "difficulty": "medium"
    },
    "draft_result": {
      "title": "二叉树的中序遍历",
      "description": "..."
    },
    "logs": [],
    "retry_count": 0,
    "review_history": [],
    "created_at": "2026-03-06T10:00:00Z",
    "updated_at": "2026-03-06T10:05:00Z"
  }
}
```

---

### 4. 人工审核

#### 基本信息

```
POST /api/v1/ai/workflow/:session_id/review
```

#### 请求头

```http
Authorization: Bearer {token}
Content-Type: application/json
```

#### 请求体

```typescript
interface ReviewWorkflowRequest {
  decision: "approve" | "reject";  // 审核决策（必填）
  comment?: string;                // 审核意见（可选）
}
```

#### 请求示例

```json
{
  "decision": "approve",
  "comment": "题目质量很好，批准通过"
}
```

#### 响应体

```typescript
interface ReviewWorkflowResponse {
  code: 200;
  message: "success";
  data: {
    session_id: string;
    decision: "approve" | "reject";
    message: string;
  };
}
```

#### 响应示例（批准）

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "decision": "approve",
    "message": "审核批准，工作流已完成"
  }
}
```

#### 响应示例（拒绝）

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "decision": "reject",
    "message": "审核拒绝，工作流将重新执行"
  }
}
```

#### 错误响应

```json
{
  "code": 403,
  "message": "无权审核此会话，仅创建者本人可审核"
}
```

---

### 5. 获取会话列表

#### 基本信息

```
GET /api/v1/ai/workflow/list
```

#### 请求头

```http
Authorization: Bearer {token}
```

#### 查询参数

```typescript
interface ListSessionsQuery {
  page?: number;       // 页码，默认 1
  page_size?: number;  // 每页数量，默认 10
  status?: string;     // 状态筛选（可选）
}
```

#### 响应体

```typescript
interface ListSessionsResponse {
  code: 200;
  message: "success";
  data: {
    sessions: AIWorkflowSession[];
    total: number;
    page: number;
    page_size: number;
  };
}
```

---

### 6. 重试工作流

#### 基本信息

```
POST /api/v1/ai/workflow/:session_id/retry
```

#### 请求头

```http
Authorization: Bearer {token}
```

#### 响应体

```typescript
interface RetryWorkflowResponse {
  code: 200;
  message: "success";
  data: {
    session_id: string;
    message: string;
  };
}
```

---

### 7. 删除会话

#### 基本信息

```
DELETE /api/v1/ai/workflow/:session_id
```

#### 请求头

```http
Authorization: Bearer {token}
```

#### 响应体

```typescript
interface DeleteSessionResponse {
  code: 200;
  message: "success";
  data: {
    session_id: string;
    message: "会话已删除";
  };
}
```

---

## 📦 数据模型

### OJProblemWorkflowResult（题目结果）

```typescript
interface OJProblemWorkflowResult {
  // 题目基本信息
  title: string;              // 题目标题
  description: string;        // 题目描述
  input_desc: string;         // 输入说明
  output_desc: string;        // 输出说明
  sample_input: string;       // 样例输入
  sample_output: string;      // 样例输出
  hint: string;               // 提示
  tags: string[];             // 标签
  difficulty: string;         // 难度
  time_limit: number;         // 时间限制(ms)
  memory_limit: number;       // 内存限制(MB)

  // 标准答案
  std_code: string;           // 标程代码
  std_language: string;       // 标程语言

  // 测试用例
  test_cases: TestCase[];     // 测试用例列表

  // 验证结果
  verify_passed: boolean;     // 验证是否通过
  verify_logs: string;        // 验证日志
}

interface TestCase {
  input: string;              // 输入
  output: string;             // 输出
  is_sample: boolean;         // 是否为样例
  description: string;        // 用例描述
}
```

---

## 🔄 状态流转图

```
┌─────────┐
│ pending │ 初始状态
└────┬────┘
     ↓
┌─────────┐
│ running │ 执行中
└────┬────┘
     ├─────────────────┐
     ↓                 ↓
┌────────────────┐  ┌────────┐
│ pending_review │  │ failed │ 失败
└────┬───────────┘  └────────┘
     ├─ approve
     ↓
┌───────────┐
│ completed │ 完成
└───────────┘
     ↑
     └─ reject → pending → running (重新执行)
```

---

## ⚠️ 重要约定

### 1. 字段命名规范

- ✅ 使用 **snake_case**（下划线命名）
- ✅ 示例：`session_id`, `workflow_type`, `draft_result`

### 2. 时间格式

- ✅ 统一使用 **ISO 8601** 格式
- ✅ 示例：`2026-03-06T10:00:00Z`

### 3. ID 格式

- `session_id`: UUID 格式（`550e8400-e29b-41d4-a716-446655440000`）
- `user_id`: MongoDB ObjectID（`65f1234567890abcdef12345`）

### 4. 枚举值

#### 难度 (difficulty)
- `"easy"` - 简单
- `"medium"` - 中等
- `"hard"` - 困难

#### 语言 (language)
- `"cpp"` - C++
- `"java"` - Java
- `"python"` - Python
- `"go"` - Go
- `"c"` - C

#### 状态 (status)
- `"pending"` - 等待中
- `"running"` - 执行中
- `"pending_review"` - 待审核
- `"completed"` - 已完成
- `"failed"` - 失败

#### 审核决策 (decision)
- `"approve"` - 批准
- `"reject"` - 拒绝

---

## 🧪 测试用例

### 正常流程测试

```typescript
// 1. 启动工作流
const startResponse = await fetch('/api/v1/ai/workflow/start', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer xxx',
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    workflow_type: 'oj_problem_generation',
    input: {
      topic: '二叉树遍历',
      difficulty: 'medium'
    }
  })
});
// 期望: { code: 200, data: { session_id: "xxx" } }

// 2. 监听进度
const eventSource = new EventSource('/api/v1/ai/workflow/xxx/stream');
eventSource.onmessage = (e) => {
  const event = JSON.parse(e.data);
  // 期望: type 为 "progress" | "review_required" | "completed" | "error"
};

// 3. 查询会话
const sessionResponse = await fetch('/api/v1/ai/workflow/xxx');
// 期望: { code: 200, data: { status: "pending_review", draft_result: {...} } }

// 4. 提交审核
const reviewResponse = await fetch('/api/v1/ai/workflow/xxx/review', {
  method: 'POST',
  body: JSON.stringify({
    decision: 'approve',
    comment: '很好'
  })
});
// 期望: { code: 200, data: { decision: "approve" } }
```

---

## 📝 变更日志

### v1.0 (2026-03-06)

- ✅ 初始版本
- ✅ 定义 7 个核心接口
- ✅ 定义数据模型
- ✅ 定义 SSE 事件格式
- ✅ 添加人工审核接口

---

## 🤝 契约承诺

### 前端承诺

- ✅ 严格按照本文档定义的接口格式发送请求
- ✅ 正确处理所有定义的响应格式
- ✅ 正确处理所有定义的错误码

### 后端承诺

- ✅ 严格按照本文档定义的接口格式返回响应
- ✅ 不修改已定义的字段名和类型
- ✅ 不修改已定义的接口路径
- ✅ 如需新增字段，只能添加可选字段

---

**文档维护**: AI Workflow Team
**联系方式**: 如有疑问请查看 `docs/FRONTEND-OJ-PROBLEM-GENERATION-GUIDE.md`
