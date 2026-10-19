# AI 工作流前端集成指南

> **版本**: v1.2.0
> **更新时间**: 2026-03-06
> **目标读者**: 前端开发者

---

## 📊 开发进度总览

### 后端完成度：100% ✅

| 模块 | 状态 | 说明 |
|------|------|------|
| 核心工作流引擎 | ✅ 完成 | 7 节点流程（题目→测试用例→标程→运行→验证→格式化） |
| API 接口层 | ✅ 完成 | 6 个 RESTful 接口 + SSE 流式推送 |
| 通用工作流系统 | ✅ 完成 | 插件化架构，支持 13 种工作流类型 |
| 权限控制 | ✅ 完成 | 仅老师/管理员可访问 |
| 数据持久化 | ✅ 完成 | MongoDB 存储 + 30 天 TTL |
| 失败重试 | ✅ 完成 | 最多 3 次自动重试 |

### 前端待开发功能

- [ ] 工作流启动表单（动态表单生成）
- [ ] 实时进度展示（SSE 连接）
- [ ] 结果预览和编辑
- [ ] 历史记录列表
- [ ] 重试和删除操作

---

## 🚀 快速开始

### 1. Base URL

```
http://localhost:8080/api/v1/ai/workflow
```

### 2. 认证

所有请求需要携带 JWT Token：

```javascript
const headers = {
  'Authorization': `Bearer ${token}`,
  'Content-Type': 'application/json'
};
```

### 3. 权限要求

- ✅ 老师（teacher）
- ✅ 管理员（admin）
- ❌ 学生（student）- 返回 403

---

## 📡 核心接口

### 接口 1: 启动工作流

**端点**: `POST /start`

**新版请求格式（推荐）**:

```typescript
interface StartWorkflowRequest {
  workflow_type: string;  // 工作流类型
  input: Record<string, any>;  // 动态输入参数
  config?: {  // 可选配置
    timeout?: number;
    retry_on_failure?: boolean;
    max_retries?: number;
  };
  metadata?: Record<string, any>;  // 可选元数据
}
```

**示例 - OJ 题目生成**:

```javascript
const response = await fetch('/api/v1/ai/workflow/start', {
  method: 'POST',
  headers,
  body: JSON.stringify({
    workflow_type: 'oj_problem_generation',
    input: {
      topic: '二叉树遍历',
      difficulty: 'medium',
      tags: ['树', '递归', 'DFS'],
      language: 'cpp',
      testcase_count: 10,
      time_limit: 1000,
      memory_limit: 256
    }
  })
});

const data = await response.json();
// data.data.session_id - 用于后续查询
```

**响应**:

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

### 接口 2: SSE 流式监听（推荐）

**端点**: `GET /:session_id/stream`

**前端实现**:

```typescript
function connectWorkflowStream(sessionId: string, token: string) {
  const eventSource = new EventSource(
    `/api/v1/ai/workflow/${sessionId}/stream`,
    {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    }
  );

  // 1. 连接建立
  eventSource.addEventListener('connected', (e) => {
    const data = JSON.parse(e.data);
    console.log('已连接:', data.message);
  });

  // 2. 进度更新（最重要）
  eventSource.addEventListener('progress', (e) => {
    const data = JSON.parse(e.data);
    updateProgress(data.progress);  // 0-100
    updateCurrentStep(data.step);   // 当前步骤名称
    addLog(data.log);               // 日志信息
  });

  // 3. 工作流完成
  eventSource.addEventListener('completed', (e) => {
    const data = JSON.parse(e.data);
    showResult(data.result);  // 显示最终结果
    eventSource.close();
  });

  // 4. 错误处理
  eventSource.addEventListener('error', (e) => {
    const data = JSON.parse(e.data);
    showError(data.error);
    eventSource.close();
  });

  // 5. 心跳（每 30 秒）
  eventSource.addEventListener('heartbeat', (e) => {
    console.log('心跳:', new Date(JSON.parse(e.data).timestamp * 1000));
  });

  return eventSource;
}
```

**事件类型说明**:

| 事件类型 | 触发时机 | 数据结构 |
|---------|---------|---------|
| `connected` | 连接建立 | `{session_id, message}` |
| `progress` | 进度更新 | `{step, progress, log}` |
| `completed` | 工作流完成 | `{session_id, result}` |
| `error` | 发生错误 | `{error}` |
| `heartbeat` | 每 30 秒 | `{timestamp}` |

---

### 接口 3: 查询会话状态

**端点**: `GET /:session_id`

**使用场景**:
- 页面刷新后恢复状态
- 不使用 SSE 时的轮询查询

```javascript
const response = await fetch(`/api/v1/ai/workflow/${sessionId}`, {
  headers
});

const data = await response.json();
const session = data.data;

// session.status: pending/running/completed/failed/retrying
// session.progress: 0-100
// session.result: 完成后的结果对象
```

---

### 接口 4: 获取历史列表

**端点**: `GET /list?page=1&page_size=10&status=completed`

**查询参数**:

| 参数 | 类型 | 必填 | 说明 | 默认值 |
|------|------|------|------|--------|
| page | int | ❌ | 页码（从 1 开始） | 1 |
| page_size | int | ❌ | 每页数量 | 20 |
| status | string | ❌ | 状态筛选 | 全部 |

```javascript
const response = await fetch(
  '/api/v1/ai/workflow/list?page=1&page_size=10&status=completed',
  { headers }
);

const data = await response.json();
// data.data.total - 总数
// data.data.sessions - 会话列表
```

---

### 接口 5: 重试工作流

**端点**: `POST /:session_id/retry`

**使用场景**: 工作流失败后重新执行（最多 3 次）

```javascript
const response = await fetch(`/api/v1/ai/workflow/${sessionId}/retry`, {
  method: 'POST',
  headers
});
```

---

### 接口 6: 删除会话

**端点**: `DELETE /:session_id`

```javascript
await fetch(`/api/v1/ai/workflow/${sessionId}`, {
  method: 'DELETE',
  headers
});
```

---

## 🎨 UI 组件建议

### 1. 工作流启动表单

```tsx
interface OJProblemFormData {
  topic: string;
  difficulty: 'easy' | 'medium' | 'hard';
  tags: string[];
  extra_req?: string;
  language: 'cpp' | 'java' | 'python' | 'go' | 'c';
  testcase_count: number;
  time_limit: number;
  memory_limit: number;
}

function WorkflowStartForm() {
  const [formData, setFormData] = useState<OJProblemFormData>({
    topic: '',
    difficulty: 'medium',
    tags: [],
    language: 'cpp',
    testcase_count: 10,
    time_limit: 1000,
    memory_limit: 256
  });

  const handleSubmit = async () => {
    const response = await startWorkflow({
      workflow_type: 'oj_problem_generation',
      input: formData
    });

    // 跳转到进度页面
    navigate(`/workflow/${response.data.session_id}`);
  };

  return (
    <Form>
      <Input label="题目主题" required />
      <Select label="难度" options={['easy', 'medium', 'hard']} />
      <TagInput label="标签" />
      <Select label="编程语言" options={['cpp', 'java', 'python', 'go', 'c']} />
      <NumberInput label="测试用例数量" min={1} max={50} />
      <Button onClick={handleSubmit}>开始生成</Button>
    </Form>
  );
}
```

---

### 2. 进度展示组件

```tsx
function WorkflowProgress({ sessionId }: { sessionId: string }) {
  const [progress, setProgress] = useState(0);
  const [currentStep, setCurrentStep] = useState('');
  const [logs, setLogs] = useState<Log[]>([]);
  const [result, setResult] = useState(null);

  useEffect(() => {
    const es = connectWorkflowStream(sessionId, token);

    es.addEventListener('progress', (e) => {
      const data = JSON.parse(e.data);
      setProgress(data.progress);
      setCurrentStep(data.step);
      setLogs(prev => [...prev, data.log]);
    });

    es.addEventListener('completed', (e) => {
      const data = JSON.parse(e.data);
      setResult(data.result);
    });

    return () => es.close();
  }, [sessionId]);

  return (
    <div>
      <ProgressBar value={progress} max={100} />
      <StepIndicator steps={WORKFLOW_STEPS} current={currentStep} />
      <LogViewer logs={logs} />
      {result && <ResultPreview result={result} />}
    </div>
  );
}
```

**工作流步骤常量**:

```typescript
const WORKFLOW_STEPS = [
  { key: 'problem_gen', label: '生成题目描述', progress: [5, 25] },
  { key: 'testcase_gen', label: '生成测试用例', progress: [25, 40] },
  { key: 'solution_gen', label: '生成标准答案', progress: [40, 60] },
  { key: 'run_testcases', label: '运行标程', progress: [60, 75] },
  { key: 'verify', label: '验证测试用例', progress: [75, 95] },
  { key: 'formatter', label: '格式化结果', progress: [95, 100] }
];
```

---

### 3. 结果预览组件

```tsx
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

function ResultPreview({ result }: { result: WorkflowResult }) {
  return (
    <div>
      <Section title="题目信息">
        <h2>{result.title}</h2>
        <Badge>{result.difficulty}</Badge>
        <Tags items={result.tags} />
        <Markdown content={result.description} />
      </Section>

      <Section title="标准答案">
        <CodeEditor
          value={result.std_code}
          language={result.std_language}
          readOnly
        />
      </Section>

      <Section title="测试用例">
        <TestCaseList cases={result.test_cases} />
      </Section>

      <Section title="验证结果">
        <StatusBadge passed={result.verify_passed} />
        <pre>{result.verify_logs}</pre>
      </Section>

      <Actions>
        <Button onClick={handleSaveToProblemBank}>保存到题库</Button>
        <Button onClick={handleEdit}>编辑</Button>
        <Button onClick={handleRegenerate}>重新生成</Button>
      </Actions>
    </div>
  );
}
```

---

## 🔄 完整流程示例

### React + TypeScript 完整实现

```typescript
import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

// 1. 启动工作流
function StartWorkflowPage() {
  const navigate = useNavigate();

  const handleStart = async (formData: any) => {
    const response = await fetch('/api/v1/ai/workflow/start', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${getToken()}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        workflow_type: 'oj_problem_generation',
        input: formData
      })
    });

    const data = await response.json();
    if (data.code === 200) {
      navigate(`/workflow/${data.data.session_id}`);
    }
  };

  return <WorkflowForm onSubmit={handleStart} />;
}

// 2. 监听进度
function WorkflowProgressPage({ sessionId }: { sessionId: string }) {
  const [state, setState] = useState({
    progress: 0,
    currentStep: '',
    logs: [],
    result: null,
    error: null
  });

  useEffect(() => {
    const token = getToken();
    const es = new EventSource(
      `/api/v1/ai/workflow/${sessionId}/stream`,
      { headers: { 'Authorization': `Bearer ${token}` } }
    );

    es.addEventListener('progress', (e) => {
      const data = JSON.parse(e.data);
      setState(prev => ({
        ...prev,
        progress: data.progress,
        currentStep: data.step,
        logs: [...prev.logs, data.log]
      }));
    });

    es.addEventListener('completed', (e) => {
      const data = JSON.parse(e.data);
      setState(prev => ({ ...prev, result: data.result }));
      es.close();
    });

    es.addEventListener('error', (e) => {
      const data = JSON.parse(e.data);
      setState(prev => ({ ...prev, error: data.error }));
      es.close();
    });

    return () => es.close();
  }, [sessionId]);

  if (state.error) {
    return <ErrorView error={state.error} onRetry={() => retryWorkflow(sessionId)} />;
  }

  if (state.result) {
    return <ResultView result={state.result} />;
  }

  return (
    <ProgressView
      progress={state.progress}
      currentStep={state.currentStep}
      logs={state.logs}
    />
  );
}

// 3. 历史记录列表
function WorkflowHistoryPage() {
  const [sessions, setSessions] = useState([]);
  const [page, setPage] = useState(1);

  useEffect(() => {
    fetchSessions(page);
  }, [page]);

  const fetchSessions = async (page: number) => {
    const response = await fetch(
      `/api/v1/ai/workflow/list?page=${page}&page_size=10`,
      { headers: { 'Authorization': `Bearer ${getToken()}` } }
    );
    const data = await response.json();
    setSessions(data.data.sessions);
  };

  return (
    <div>
      <SessionList sessions={sessions} />
      <Pagination page={page} onChange={setPage} />
    </div>
  );
}
```

---

## ⚠️ 错误处理

### 常见错误码

| 状态码 | 说明 | 处理建议 |
|--------|------|---------|
| 200 | 成功 | - |
| 400 | 请求参数错误 | 检查表单验证 |
| 401 | 未授权 | 重新登录 |
| 403 | 权限不足 | 提示"仅老师/管理员可用" |
| 404 | 会话不存在 | 返回列表页 |
| 503 | AI 服务不可用 | 提示"服务暂时不可用，请稍后重试" |
| 500 | 服务器错误 | 联系管理员 |

### 错误处理示例

```typescript
async function handleApiCall(apiFunc: () => Promise<Response>) {
  try {
    const response = await apiFunc();
    const data = await response.json();

    if (data.code !== 200) {
      switch (data.code) {
        case 401:
          // 跳转登录
          navigate('/login');
          break;
        case 403:
          message.error('权限不足：仅老师和管理员可使用此功能');
          break;
        case 503:
          message.error('AI 服务暂时不可用，请稍后重试');
          break;
        default:
          message.error(data.message || '操作失败');
      }
      return null;
    }

    return data.data;
  } catch (error) {
    message.error('网络错误，请检查连接');
    return null;
  }
}
```

---

## 📝 TypeScript 类型定义

```typescript
// 工作流类型
type WorkflowType =
  | 'oj_problem_generation'
  | 'multiple_choice_generation'
  | 'fill_blank_generation'
  | 'essay_generation'
  | 'code_review'
  | 'code_refactor'
  | 'code_optimize'
  | 'assignment_grade'
  | 'essay_grade'
  | 'data_analysis'
  | 'report_generation'
  | 'custom';

// 工作流状态
type WorkflowStatus = 'pending' | 'running' | 'completed' | 'failed' | 'retrying';

// 步骤状态
type StepStatus = 'pending' | 'running' | 'completed' | 'failed' | 'skipped';

// 会话信息
interface WorkflowSession {
  session_id: string;
  user_id: string;
  workflow_type: WorkflowType;
  status: WorkflowStatus;
  current_step: string;
  progress: number;  // 0-100
  input: Record<string, any>;
  result?: any;
  logs: WorkflowLog[];
  error_msg?: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

// 日志
interface WorkflowLog {
  step: string;
  status: string;
  message: string;
  timestamp: string;
}

// SSE 事件
interface ProgressEvent {
  step: string;
  progress: number;
  log: WorkflowLog;
}

interface CompletedEvent {
  session_id: string;
  result: any;
}

interface ErrorEvent {
  error: string;
}
```

---

## 🎯 开发检查清单

### 必须实现
- [ ] 工作流启动表单（支持 OJ 题目生成）
- [ ] SSE 实时进度展示
- [ ] 结果预览界面
- [ ] 错误处理和重试

### 推荐实现
- [ ] 历史记录列表
- [ ] 会话详情页（支持页面刷新恢复）
- [ ] 删除会话功能
- [ ] 结果编辑和保存到题库

### 可选实现
- [ ] 工作流模板（预设常用配置）
- [ ] 批量生成
- [ ] 导出功能
- [ ] 数据统计（成功率、平均耗时等）

---

## 📞 技术支持

### 后端接口问题
- 查看完整 API 文档: `docs/API-AI-WORKFLOW.md`
- 查看快速参考: `docs/API-AI-WORKFLOW-QUICK.md`

### 架构设计问题
- 查看架构文档: `docs/WORKFLOW-ARCHITECTURE.md`
- 查看重构总结: `docs/WORKFLOW-REFACTOR-SUMMARY.md`

### 测试环境
- Base URL: `http://localhost:8080`
- 需要先登录获取 JWT Token
- 确保用户角色为 `teacher` 或 `admin`

---

**文档版本**: v1.2.0
**最后更新**: 2026-03-06
**维护者**: AI Assistant
