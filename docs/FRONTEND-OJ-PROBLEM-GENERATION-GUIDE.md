# OJ 题目生成工作流 - 前端开发指南

> **文档版本**: v1.0
> **更新时间**: 2026-03-06
> **适用范围**: 前端开发人员

---

## 📋 目录

1. [功能概述](#功能概述)
2. [业务流程](#业务流程)
3. [API 接口](#api-接口)
4. [数据模型](#数据模型)
5. [前端实现建议](#前端实现建议)
6. [完整示例代码](#完整示例代码)

---

## 功能概述

### 核心功能

AI 自动生成 OJ 编程题目，包括：
- ✅ 题目描述（标题、输入输出说明、样例）
- ✅ 测试用例（10+ 个，包含边界情况）
- ✅ 标准答案（支持 C/C++/Java/Python/Go）
- ✅ 自动验证（标程运行验证测试用例）
- ✅ 人工审核（待审核状态，支持批准/拒绝）

### 用户角色

- **权限要求**: 老师或管理员
- **审核权限**: 仅会话创建者本人可审核

---

## 业务流程

### 完整流程图

```
┌─────────────┐
│ 1. 用户输入 │ → 题目主题、难度、标签等
└──────┬──────┘
       ↓
┌─────────────┐
│ 2. 启动工作流│ → POST /ai/workflow/start
└──────┬──────┘
       ↓
┌─────────────┐
│ 3. 实时进度 │ → SSE /ai/workflow/:id/stream
│   (7个步骤) │
└──────┬──────┘
       ↓
┌─────────────┐
│ 4. 人工审核 │ → 状态: pending_review
│  (可选节点) │
└──────┬──────┘
       ├─ 批准 → POST /ai/workflow/:id/review (approve)
       │         ↓
       │    ┌─────────────┐
       │    │ 5. 完成     │
       │    └─────────────┘
       │
       └─ 拒绝 → POST /ai/workflow/:id/review (reject)
                 ↓
            ┌─────────────┐
            │ 6. 重新执行 │ (retry_count++)
            └─────────────┘
```

### 7 个工作流步骤

| 步骤 | 名称 | 进度 | 说明 |
|------|------|------|------|
| 1 | problem_gen | 0-15% | 生成题目描述 |
| 2 | testcase_gen | 15-40% | 生成测试用例输入 |
| 3 | solution_gen | 40-60% | 生成标准答案代码 |
| 4 | run_testcases | 60-75% | 运行标程生成输出 |
| 5 | verify | 75-85% | 验证测试用例正确性 |
| 6 | human_review | 85-95% | 人工审核（可选） |
| 7 | formatter | 95-100% | 格式化最终结果 |

### 状态流转

```
pending → running → pending_review → completed
                 ↓                  ↓
                failed          (reject) → pending → running
```

---

## API 接口

### 基础信息

- **Base URL**: `/api/v1`
- **认证方式**: Bearer Token (JWT)
- **权限要求**: 老师或管理员

### 1. 启动工作流

**接口**: `POST /ai/workflow/start`

**请求头**:
```http
Authorization: Bearer {token}
Content-Type: application/json
```

**请求体**:
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

**参数说明**:

| 字段 | 类型 | 必填 | 说明 | 示例 |
|------|------|------|------|------|
| workflow_type | string | ✅ | 工作流类型 | "oj_problem_generation" |
| input.topic | string | ✅ | 题目主题 | "二叉树遍历" |
| input.difficulty | string | ✅ | 难度 | "easy" / "medium" / "hard" |
| input.tags | array | ❌ | 标签 | ["树", "递归"] |
| input.extra_req | string | ❌ | 额外要求 | "要求时间复杂度 O(n)" |
| input.language | string | ❌ | 标程语言 | "cpp" / "java" / "python" / "go" / "c" |
| input.testcase_count | int | ❌ | 测试用例数 | 10 (默认), 范围 1-50 |
| input.time_limit | int | ❌ | 时间限制(ms) | 1000 (默认), 范围 100-10000 |
| input.memory_limit | int | ❌ | 内存限制(MB) | 256 (默认), 范围 32-1024 |

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

---

### 2. 实时进度推送 (SSE)

**接口**: `GET /ai/workflow/:session_id/stream`

**请求头**:
```http
Authorization: Bearer {token}
Accept: text/event-stream
```

**SSE 事件类型**:

#### 2.1 进度事件 (`progress`)

```
event: message
data: {"session_id":"xxx","type":"progress","data":{"step":"problem_gen","progress":10,"log":{"step":"problem_gen","status":"running","message":"正在生成题目描述","timestamp":"2026-03-06T10:00:00Z"}},"timestamp":"2026-03-06T10:00:00Z"}
```

**data 字段**:
- `step`: 当前步骤名称
- `progress`: 进度百分比 (0-100)
- `log`: 日志对象
  - `status`: "running" / "success" / "failed"
  - `message`: 进度消息

#### 2.2 审核请求事件 (`review_required`)

```
event: message
data: {"session_id":"xxx","type":"review_required","data":{"message":"human review required","draft_result":{...}},"timestamp":"2026-03-06T10:05:00Z"}
```

**说明**: 工作流进入待审核状态，前端应显示审核界面

#### 2.3 完成事件 (`completed`)

```
event: message
data: {"session_id":"xxx","type":"completed","data":{"session_id":"xxx","result":{...}},"timestamp":"2026-03-06T10:06:00Z"}
```

#### 2.4 错误事件 (`error`)

```
event: message
data: {"session_id":"xxx","type":"error","data":{"error":"生成失败: xxx"},"timestamp":"2026-03-06T10:01:00Z"}
```

#### 2.5 审核批准事件 (`review_approved`)

```
event: message
data: {"session_id":"xxx","type":"review_approved","data":{"reviewer_id":"xxx","comment":"题目质量很好"},"timestamp":"2026-03-06T10:07:00Z"}
```

#### 2.6 审核拒绝事件 (`review_rejected`)

```
event: message
data: {"session_id":"xxx","type":"review_rejected","data":{"reviewer_id":"xxx","comment":"测试用例不够全面","retry_count":1},"timestamp":"2026-03-06T10:07:00Z"}
```

---

### 3. 查询会话信息

**接口**: `GET /ai/workflow/:session_id`

**响应示例**:
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
    "input": { ... },
    "draft_result": { ... },
    "result": null,
    "logs": [ ... ],
    "review_history": [
      {
        "reviewer_id": "65f1234567890abcdef12346",
        "decision": "reject",
        "comment": "测试用例不够全面",
        "reviewed_at": "2026-03-06T10:05:00Z"
      }
    ],
    "retry_count": 1,
    "created_at": "2026-03-06T10:00:00Z",
    "updated_at": "2026-03-06T10:05:00Z"
  }
}
```

---

### 4. 人工审核

**接口**: `POST /ai/workflow/:session_id/review`

**请求体**:
```json
{
  "decision": "approve",
  "comment": "题目质量很好，批准通过"
}
```

**参数说明**:

| 字段 | 类型 | 必填 | 说明 | 可选值 |
|------|------|------|------|--------|
| decision | string | ✅ | 审核决策 | "approve" / "reject" |
| comment | string | ❌ | 审核意见 | 任意文本 |

**响应示例 (批准)**:
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

**响应示例 (拒绝)**:
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

**错误响应**:
```json
{
  "code": 403,
  "message": "无权审核此会话，仅创建者本人可审核"
}
```

---

### 5. 获取会话列表

**接口**: `GET /ai/workflow/list`

**查询参数**:
- `page`: 页码 (默认 1)
- `page_size`: 每页数量 (默认 10)
- `status`: 状态筛选 (可选)

**响应示例**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "sessions": [ ... ],
    "total": 25,
    "page": 1,
    "page_size": 10
  }
}
```

---

### 6. 重试工作流

**接口**: `POST /ai/workflow/:session_id/retry`

**说明**: 重试失败的工作流

---

### 7. 删除会话

**接口**: `DELETE /ai/workflow/:session_id`

---

## 数据模型

### 工作流结果 (OJProblemWorkflowResult)

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

## 前端实现建议

### 页面结构

```
┌─────────────────────────────────────┐
│  OJ 题目生成                         │
├─────────────────────────────────────┤
│  1. 输入表单                         │
│     - 题目主题 (必填)                │
│     - 难度选择 (必填)                │
│     - 标签 (可选)                    │
│     - 高级选项 (折叠)                │
│       - 标程语言                     │
│       - 测试用例数量                 │
│       - 时间/内存限制                │
│                                      │
│  2. 进度展示 (生成中)                │
│     ┌──────────────────────┐        │
│     │ ████████░░░░░░░░ 45% │        │
│     └──────────────────────┘        │
│     当前步骤: 生成标准答案           │
│                                      │
│  3. 审核界面 (待审核)                │
│     ┌──────────────────────┐        │
│     │ 题目预览              │        │
│     │ - 标题、描述          │        │
│     │ - 测试用例            │        │
│     │ - 标准答案            │        │
│     └──────────────────────┘        │
│     [批准] [拒绝]                    │
│                                      │
│  4. 结果展示 (完成)                  │
│     [保存到题库] [重新生成]          │
└─────────────────────────────────────┘
```

### 状态管理

```typescript
interface WorkflowState {
  sessionId: string | null;
  status: 'idle' | 'running' | 'pending_review' | 'completed' | 'failed';
  progress: number;
  currentStep: string;
  logs: WorkflowLog[];
  draftResult: OJProblemWorkflowResult | null;
  result: OJProblemWorkflowResult | null;
  error: string | null;
  retryCount: number;
}
```

### 关键交互

#### 1. 启动生成

```typescript
async function startGeneration(input: GenerationInput) {
  const response = await fetch('/api/v1/ai/workflow/start', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      workflow_type: 'oj_problem_generation',
      input
    })
  });

  const { data } = await response.json();
  return data.session_id;
}
```

#### 2. 监听进度

```typescript
function subscribeProgress(sessionId: string, onEvent: (event: WorkflowEvent) => void) {
  const eventSource = new EventSource(
    `/api/v1/ai/workflow/${sessionId}/stream`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );

  eventSource.onmessage = (e) => {
    const event = JSON.parse(e.data);
    onEvent(event);

    // 根据事件类型处理
    switch (event.type) {
      case 'progress':
        updateProgress(event.data.progress, event.data.step);
        break;
      case 'review_required':
        showReviewUI(event.data.draft_result);
        break;
      case 'completed':
        showResult(event.data.result);
        eventSource.close();
        break;
      case 'error':
        showError(event.data.error);
        eventSource.close();
        break;
    }
  };

  return eventSource;
}
```

#### 3. 提交审核

```typescript
async function submitReview(sessionId: string, decision: 'approve' | 'reject', comment?: string) {
  const response = await fetch(`/api/v1/ai/workflow/${sessionId}/review`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ decision, comment })
  });

  return await response.json();
}
```

---

## 完整示例代码

### React + TypeScript 示例

```typescript
import React, { useState, useEffect } from 'react';

interface GenerationInput {
  topic: string;
  difficulty: 'easy' | 'medium' | 'hard';
  tags?: string[];
  extra_req?: string;
  language?: string;
  testcase_count?: number;
  time_limit?: number;
  memory_limit?: number;
}

export function OJProblemGenerator() {
  const [input, setInput] = useState<GenerationInput>({
    topic: '',
    difficulty: 'medium',
    language: 'cpp',
    testcase_count: 10,
    time_limit: 1000,
    memory_limit: 256
  });

  const [sessionId, setSessionId] = useState<string | null>(null);
  const [status, setStatus] = useState<string>('idle');
  const [progress, setProgress] = useState(0);
  const [currentStep, setCurrentStep] = useState('');
  const [draftResult, setDraftResult] = useState<any>(null);
  const [result, setResult] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  // 启动生成
  const handleStart = async () => {
    try {
      const response = await fetch('/api/v1/ai/workflow/start', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          workflow_type: 'oj_problem_generation',
          input
        })
      });

      const { data } = await response.json();
      setSessionId(data.session_id);
      setStatus('running');
    } catch (err) {
      setError(err.message);
    }
  };

  // 监听进度
  useEffect(() => {
    if (!sessionId) return;

    const eventSource = new EventSource(
      `/api/v1/ai/workflow/${sessionId}/stream?token=${localStorage.getItem('token')}`
    );

    eventSource.onmessage = (e) => {
      const event = JSON.parse(e.data);

      switch (event.type) {
        case 'progress':
          setProgress(event.data.progress);
          setCurrentStep(event.data.step);
          break;

        case 'review_required':
          setStatus('pending_review');
          setDraftResult(event.data.draft_result);
          break;

        case 'completed':
          setStatus('completed');
          setResult(event.data.result);
          eventSource.close();
          break;

        case 'error':
          setStatus('failed');
          setError(event.data.error);
          eventSource.close();
          break;
      }
    };

    return () => eventSource.close();
  }, [sessionId]);

  // 提交审核
  const handleReview = async (decision: 'approve' | 'reject', comment?: string) => {
    try {
      await fetch(`/api/v1/ai/workflow/${sessionId}/review`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ decision, comment })
      });

      if (decision === 'approve') {
        setStatus('completed');
        setResult(draftResult);
      } else {
        setStatus('running');
        setProgress(0);
      }
    } catch (err) {
      setError(err.message);
    }
  };

  return (
    <div className="oj-problem-generator">
      {/* 输入表单 */}
      {status === 'idle' && (
        <div className="input-form">
          <h2>生成 OJ 题目</h2>
          <input
            type="text"
            placeholder="题目主题（必填）"
            value={input.topic}
            onChange={(e) => setInput({ ...input, topic: e.target.value })}
          />
          <select
            value={input.difficulty}
            onChange={(e) => setInput({ ...input, difficulty: e.target.value as any })}
          >
            <option value="easy">简单</option>
            <option value="medium">中等</option>
            <option value="hard">困难</option>
          </select>
          <button onClick={handleStart} disabled={!input.topic}>
            开始生成
          </button>
        </div>
      )}

      {/* 进度展示 */}
      {status === 'running' && (
        <div className="progress-view">
          <h2>生成中...</h2>
          <div className="progress-bar">
            <div className="progress-fill" style={{ width: `${progress}%` }} />
          </div>
          <p>{progress}% - {currentStep}</p>
        </div>
      )}

      {/* 审核界面 */}
      {status === 'pending_review' && draftResult && (
        <div className="review-view">
          <h2>待审核</h2>
          <div className="problem-preview">
            <h3>{draftResult.title}</h3>
            <p>{draftResult.description}</p>
            <h4>测试用例: {draftResult.test_cases?.length} 个</h4>
            <h4>标准答案: {draftResult.std_language}</h4>
            <pre>{draftResult.std_code}</pre>
          </div>
          <div className="review-actions">
            <button onClick={() => handleReview('approve')}>
              批准
            </button>
            <button onClick={() => handleReview('reject', '需要改进')}>
              拒绝
            </button>
          </div>
        </div>
      )}

      {/* 结果展示 */}
      {status === 'completed' && result && (
        <div className="result-view">
          <h2>生成完成</h2>
          <div className="problem-result">
            <h3>{result.title}</h3>
            <p>{result.description}</p>
            <button onClick={() => saveToProblemBank(result)}>
              保存到题库
            </button>
          </div>
        </div>
      )}

      {/* 错误提示 */}
      {status === 'failed' && error && (
        <div className="error-view">
          <h2>生成失败</h2>
          <p>{error}</p>
          <button onClick={() => setStatus('idle')}>
            重新开始
          </button>
        </div>
      )}
    </div>
  );
}
```

---

## 注意事项

### 1. 权限控制

- ✅ 所有接口需要 JWT 认证
- ✅ 需要老师或管理员角色
- ✅ 审核接口仅会话创建者可调用

### 2. 错误处理

| 错误码 | 说明 | 处理建议 |
|--------|------|----------|
| 400 | 请求参数错误 | 检查输入参数 |
| 401 | 未授权 | 重新登录 |
| 403 | 权限不足 | 提示用户权限不足 |
| 404 | 会话不存在 | 会话已过期或被删除 |
| 500 | 服务器错误 | 稍后重试 |
| 503 | AI 服务不可用 | 检查 LLM 配置 |

### 3. 重试机制

- 审核拒绝后自动重新执行
- 最多重试 3 次（包含审核拒绝）
- 超过限制后标记为失败

### 4. 会话管理

- 会话保留 30 天（TTL 索引）
- 建议定期清理已完成的会话
- 支持查询历史会话列表

---

## 常见问题

### Q1: SSE 连接断开怎么办？

**A**: 使用 `GET /ai/workflow/:session_id` 查询当前状态，根据状态决定是否重新连接。

### Q2: 如何取消正在执行的工作流？

**A**: 目前不支持取消，可以等待完成后删除会话。

### Q3: 审核拒绝后会立即重新执行吗？

**A**: 是的，拒绝后会异步重新执行，可以通过 SSE 监听新的进度。

### Q4: 生成的题目可以编辑吗？

**A**: 可以，保存到题库后使用题目编辑功能修改。

---

## 更新日志

- **v1.0** (2026-03-06): 初始版本，包含完整的 API 文档和示例代码

---

**文档维护**: AI Workflow Team
**技术支持**: 查看 `docs/API-AI-WORKFLOW.md` 获取更多技术细节
