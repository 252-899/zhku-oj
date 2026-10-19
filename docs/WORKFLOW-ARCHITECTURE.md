# 通用工作流系统架构设计

## 🎯 设计目标

构建一个**通用的、可扩展的工作流编排系统**，支持：
- ✅ 多种工作流类型（题目生成、代码审查、批改作业等）
- ✅ 动态参数解析
- ✅ 插件化扩展
- ✅ 统一的执行引擎
- ✅ 实时进度监控

---

## 🏗️ 系统架构

### 三层架构

```
┌──────────────────────────────────────────────────┐
│              API 层 (统一接口)                     │
│  POST /api/v1/workflow/execute                   │
│  GET  /api/v1/workflow/:id/status                │
│  GET  /api/v1/workflow/:id/stream                │
└────────────────┬─────────────────────────────────┘
                 │
┌────────────────▼─────────────────────────────────┐
│           工作流编排引擎层                         │
│  - 工作流注册表 (Registry)                        │
│  - 工作流工厂 (Factory)                           │
│  - 执行调度器 (Scheduler)                         │
│  - 状态管理器 (State Manager)                     │
└────────────────┬─────────────────────────────────┘
                 │
┌────────────────▼─────────────────────────────────┐
│           具体工作流实现层                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │OJ题目生成│  │代码审查   │  │批改作业   │       │
│  │Workflow  │  │Workflow  │  │Workflow  │       │
│  └──────────┘  └──────────┘  └──────────┘       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │数据分析   │  │报告生成   │  │自定义...  │       │
│  │Workflow  │  │Workflow  │  │Workflow  │       │
│  └──────────┘  └──────────┘  └──────────┘       │
└──────────────────────────────────────────────────┘
```

---

## 📡 统一 API 设计

### 1. 执行工作流

**接口**: `POST /api/v1/workflow/execute`

**请求体**:
```json
{
  "workflow_type": "oj_problem_generation",
  "config": {
    "timeout": 300,
    "retry_on_failure": true,
    "max_retries": 3
  },
  "input": {
    // 动态参数，根据 workflow_type 不同而不同
  },
  "metadata": {
    "source": "web_ui",
    "user_agent": "Chrome/120.0"
  }
}
```

**字段说明**:

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| workflow_type | string | ✅ | 工作流类型标识 |
| config | object | ❌ | 工作流配置（超时、重试等） |
| input | object | ✅ | 工作流输入参数（动态） |
| metadata | object | ❌ | 元数据（用于追踪、分析） |

**响应**:
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "workflow_id": "wf_550e8400e29b41d4a716446655440000",
    "workflow_type": "oj_problem_generation",
    "status": "pending",
    "estimated_duration": 60
  }
}
```

---

### 2. 查询状态

**接口**: `GET /api/v1/workflow/:workflow_id/status`

**响应**:
```json
{
  "code": 200,
  "data": {
    "workflow_id": "wf_xxx",
    "workflow_type": "oj_problem_generation",
    "status": "running",
    "progress": 45,
    "current_step": "solution_generation",
    "steps": [
      {
        "name": "problem_generation",
        "status": "completed",
        "duration": 15
      },
      {
        "name": "solution_generation",
        "status": "running",
        "progress": 60
      }
    ],
    "result": null,
    "error": null
  }
}
```

---

### 3. SSE 流式监听

**接口**: `GET /api/v1/workflow/:workflow_id/stream`

**事件格式**:
```
event: step_started
data: {"step":"problem_generation","timestamp":1234567890}

event: step_progress
data: {"step":"problem_generation","progress":50,"message":"正在生成题目描述..."}

event: step_completed
data: {"step":"problem_generation","duration":15,"output":{...}}

event: workflow_completed
data: {"workflow_id":"wf_xxx","result":{...},"duration":60}
```

---

## 🔧 工作流类型定义

### 工作流类型注册表

```go
// pkg/ai/workflow/registry.go

type WorkflowType string

const (
    // 题目相关
    WorkflowTypeOJProblemGen      WorkflowType = "oj_problem_generation"
    WorkflowTypeMultipleChoiceGen WorkflowType = "multiple_choice_generation"
    WorkflowTypeFillBlankGen      WorkflowType = "fill_blank_generation"

    // 代码相关
    WorkflowTypeCodeReview        WorkflowType = "code_review"
    WorkflowTypeCodeRefactor      WorkflowType = "code_refactor"
    WorkflowTypeCodeOptimize      WorkflowType = "code_optimize"

    // 批改相关
    WorkflowTypeAssignmentGrade   WorkflowType = "assignment_grade"
    WorkflowTypeEssayGrade        WorkflowType = "essay_grade"

    // 分析相关
    WorkflowTypeDataAnalysis      WorkflowType = "data_analysis"
    WorkflowTypeReportGeneration  WorkflowType = "report_generation"

    // 通用工具
    WorkflowTypeCustom            WorkflowType = "custom"
)
```

---

## 🎨 工作流定义接口

### 核心接口

```go
// pkg/ai/workflow/interface.go

// WorkflowDefinition 工作流定义接口
type WorkflowDefinition interface {
    // 获取工作流类型
    GetType() WorkflowType

    // 获取工作流名称
    GetName() string

    // 获取工作流描述
    GetDescription() string

    // 验证输入参数
    ValidateInput(input map[string]interface{}) error

    // 获取输入参数 Schema（用于前端表单生成）
    GetInputSchema() *InputSchema

    // 构建工作流 Graph
    BuildGraph(ctx context.Context, input map[string]interface{}) (*Graph, error)

    // 格式化输出结果
    FormatOutput(output interface{}) (interface{}, error)
}

// InputSchema 输入参数 Schema
type InputSchema struct {
    Type       string                 `json:"type"`       // "object"
    Properties map[string]PropertyDef `json:"properties"` // 字段定义
    Required   []string               `json:"required"`   // 必填字段
}

// PropertyDef 字段定义
type PropertyDef struct {
    Type        string      `json:"type"`                  // string/number/array/object
    Description string      `json:"description"`           // 字段说明
    Default     interface{} `json:"default,omitempty"`     // 默认值
    Enum        []string    `json:"enum,omitempty"`        // 枚举值
    Min         *float64    `json:"minimum,omitempty"`     // 最小值
    Max         *float64    `json:"maximum,omitempty"`     // 最大值
    Pattern     string      `json:"pattern,omitempty"`     // 正则表达式
}
```

---

## 📦 工作流示例

### 示例 1：OJ 题目生成工作流

```go
// pkg/ai/workflow/definitions/oj_problem_gen.go

type OJProblemGenWorkflow struct {
    chatModel     model.ChatModel
    sandboxClient *sandbox.GoJudgeClient
}

func (w *OJProblemGenWorkflow) GetType() WorkflowType {
    return WorkflowTypeOJProblemGen
}

func (w *OJProblemGenWorkflow) GetName() string {
    return "OJ 编程题生成"
}

func (w *OJProblemGenWorkflow) GetInputSchema() *InputSchema {
    return &InputSchema{
        Type: "object",
        Properties: map[string]PropertyDef{
            "topic": {
                Type:        "string",
                Description: "题目主题",
            },
            "difficulty": {
                Type:        "string",
                Description: "难度等级",
                Enum:        []string{"easy", "medium", "hard"},
                Default:     "medium",
            },
            "language": {
                Type:        "string",
                Description: "标程语言",
                Enum:        []string{"cpp", "java", "python", "go", "c"},
                Default:     "cpp",
            },
            "testcase_count": {
                Type:        "number",
                Description: "测试用例数量",
                Default:     10,
                Min:         &[]float64{1}[0],
                Max:         &[]float64{50}[0],
            },
        },
        Required: []string{"topic", "difficulty"},
    }
}

func (w *OJProblemGenWorkflow) BuildGraph(ctx context.Context, input map[string]interface{}) (*Graph, error) {
    // 构建 7 步骤的 Eino Graph
    // ... (现有实现)
}
```

---

### 示例 2：代码审查工作流

```go
// pkg/ai/workflow/definitions/code_review.go

type CodeReviewWorkflow struct {
    chatModel model.ChatModel
}

func (w *CodeReviewWorkflow) GetType() WorkflowType {
    return WorkflowTypeCodeReview
}

func (w *CodeReviewWorkflow) GetInputSchema() *InputSchema {
    return &InputSchema{
        Type: "object",
        Properties: map[string]PropertyDef{
            "code": {
                Type:        "string",
                Description: "待审查的代码",
            },
            "language": {
                Type:        "string",
                Description: "编程语言",
                Enum:        []string{"go", "java", "python", "javascript", "typescript"},
            },
            "review_aspects": {
                Type:        "array",
                Description: "审查方面",
                Default:     []string{"security", "performance", "style", "bugs"},
            },
        },
        Required: []string{"code", "language"},
    }
}

func (w *CodeReviewWorkflow) BuildGraph(ctx context.Context, input map[string]interface{}) (*Graph, error) {
    // 构建代码审查工作流
    // 步骤：语法检查 → 安全扫描 → 性能分析 → 风格检查 → 生成报告
}
```

---

### 示例 3：作业批改工作流

```go
// pkg/ai/workflow/definitions/assignment_grade.go

type AssignmentGradeWorkflow struct {
    chatModel     model.ChatModel
    sandboxClient *sandbox.GoJudgeClient
}

func (w *AssignmentGradeWorkflow) GetType() WorkflowType {
    return WorkflowTypeAssignmentGrade
}

func (w *AssignmentGradeWorkflow) GetInputSchema() *InputSchema {
    return &InputSchema{
        Type: "object",
        Properties: map[string]PropertyDef{
            "problem_id": {
                Type:        "string",
                Description: "题目 ID",
            },
            "submission_code": {
                Type:        "string",
                Description: "学生提交的代码",
            },
            "grading_criteria": {
                Type:        "object",
                Description: "评分标准",
                Default: map[string]interface{}{
                    "correctness": 60,
                    "code_quality": 20,
                    "efficiency": 20,
                },
            },
        },
        Required: []string{"problem_id", "submission_code"},
    }
}

func (w *AssignmentGradeWorkflow) BuildGraph(ctx context.Context, input map[string]interface{}) (*Graph, error) {
    // 构建批改工作流
    // 步骤：运行测试用例 → 代码质量分析 → 性能评估 → 生成评语 → 计算分数
}
```

---

## 🔌 工作流注册机制

```go
// pkg/ai/workflow/registry.go

type WorkflowRegistry struct {
    workflows map[WorkflowType]WorkflowDefinition
    mu        sync.RWMutex
}

var globalRegistry = &WorkflowRegistry{
    workflows: make(map[WorkflowType]WorkflowDefinition),
}

// Register 注册工作流
func Register(workflow WorkflowDefinition) {
    globalRegistry.mu.Lock()
    defer globalRegistry.mu.Unlock()
    globalRegistry.workflows[workflow.GetType()] = workflow
}

// Get 获取工作流定义
func Get(workflowType WorkflowType) (WorkflowDefinition, error) {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    workflow, ok := globalRegistry.workflows[workflowType]
    if !ok {
        return nil, fmt.Errorf("workflow type not found: %s", workflowType)
    }
    return workflow, nil
}

// List 列出所有工作流
func List() []WorkflowDefinition {
    globalRegistry.mu.RLock()
    defer globalRegistry.mu.RUnlock()

    workflows := make([]WorkflowDefinition, 0, len(globalRegistry.workflows))
    for _, wf := range globalRegistry.workflows {
        workflows = append(workflows, wf)
    }
    return workflows
}

// 初始化时注册所有工作流
func init() {
    Register(&OJProblemGenWorkflow{})
    Register(&CodeReviewWorkflow{})
    Register(&AssignmentGradeWorkflow{})
    // ... 注册更多工作流
}
```

---

## 🎯 工作流执行引擎

```go
// pkg/ai/workflow/engine.go

type WorkflowEngine struct {
    registry      *WorkflowRegistry
    stateManager  *StateManager
    eventBus      *EventBus
}

// Execute 执行工作流
func (e *WorkflowEngine) Execute(ctx context.Context, req *ExecuteRequest) (*WorkflowExecution, error) {
    // 1. 获取工作流定义
    workflow, err := e.registry.Get(req.WorkflowType)
    if err != nil {
        return nil, err
    }

    // 2. 验证输入参数
    if err := workflow.ValidateInput(req.Input); err != nil {
        return nil, fmt.Errorf("invalid input: %w", err)
    }

    // 3. 创建执行实例
    execution := &WorkflowExecution{
        ID:           generateWorkflowID(),
        WorkflowType: req.WorkflowType,
        Status:       StatusPending,
        Input:        req.Input,
        Config:       req.Config,
        CreatedAt:    time.Now(),
    }

    // 4. 保存到数据库
    if err := e.stateManager.Save(execution); err != nil {
        return nil, err
    }

    // 5. 异步执行工作流
    go e.runWorkflow(ctx, workflow, execution)

    return execution, nil
}

// runWorkflow 运行工作流
func (e *WorkflowEngine) runWorkflow(ctx context.Context, workflow WorkflowDefinition, execution *WorkflowExecution) {
    // 1. 构建 Graph
    graph, err := workflow.BuildGraph(ctx, execution.Input)
    if err != nil {
        e.handleError(execution, err)
        return
    }

    // 2. 执行 Graph
    result, err := graph.Run(ctx, execution.Input)
    if err != nil {
        e.handleError(execution, err)
        return
    }

    // 3. 格式化输出
    formattedResult, err := workflow.FormatOutput(result)
    if err != nil {
        e.handleError(execution, err)
        return
    }

    // 4. 更新状态
    execution.Status = StatusCompleted
    execution.Result = formattedResult
    execution.CompletedAt = time.Now()
    e.stateManager.Save(execution)

    // 5. 发送完成事件
    e.eventBus.Publish(WorkflowCompletedEvent{
        WorkflowID: execution.ID,
        Result:     formattedResult,
    })
}
```

---

## 📊 数据模型

```go
// pkg/models/workflow.go

// WorkflowExecution 工作流执行实例
type WorkflowExecution struct {
    ID           string                 `bson:"_id" json:"id"`
    WorkflowType WorkflowType           `bson:"workflow_type" json:"workflow_type"`
    UserID       primitive.ObjectID     `bson:"user_id" json:"user_id"`
    Status       ExecutionStatus        `bson:"status" json:"status"`
    Progress     int                    `bson:"progress" json:"progress"`

    // 输入输出（动态类型）
    Input        map[string]interface{} `bson:"input" json:"input"`
    Result       interface{}            `bson:"result,omitempty" json:"result,omitempty"`

    // 执行信息
    Steps        []StepExecution        `bson:"steps" json:"steps"`
    Error        string                 `bson:"error,omitempty" json:"error,omitempty"`

    // 配置
    Config       *WorkflowConfig        `bson:"config" json:"config"`

    // 时间戳
    CreatedAt    time.Time              `bson:"created_at" json:"created_at"`
    StartedAt    *time.Time             `bson:"started_at,omitempty" json:"started_at,omitempty"`
    CompletedAt  *time.Time             `bson:"completed_at,omitempty" json:"completed_at,omitempty"`
}

// StepExecution 步骤执行信息
type StepExecution struct {
    Name      string      `bson:"name" json:"name"`
    Status    StepStatus  `bson:"status" json:"status"`
    Progress  int         `bson:"progress" json:"progress"`
    StartedAt time.Time   `bson:"started_at" json:"started_at"`
    Duration  int         `bson:"duration" json:"duration"` // 秒
    Output    interface{} `bson:"output,omitempty" json:"output,omitempty"`
    Error     string      `bson:"error,omitempty" json:"error,omitempty"`
}
```

---

## 🌟 核心优势

### 1. 高度可扩展
- ✅ 新增工作流只需实现接口并注册
- ✅ 无需修改核心代码
- ✅ 支持插件化开发

### 2. 统一管理
- ✅ 统一的 API 接口
- ✅ 统一的状态管理
- ✅ 统一的监控和日志

### 3. 类型安全
- ✅ 输入参数 Schema 验证
- ✅ 前端可根据 Schema 自动生成表单
- ✅ 类型错误在执行前捕获

### 4. 灵活配置
- ✅ 超时控制
- ✅ 重试策略
- ✅ 并发控制

---

## 🚀 使用示例

### 前端：获取工作流列表

```javascript
// GET /api/v1/workflow/types
const response = await fetch('/api/v1/workflow/types');
const workflows = await response.json();

// 响应：
// [
//   {
//     "type": "oj_problem_generation",
//     "name": "OJ 编程题生成",
//     "description": "自动生成编程题目、标程和测试用例",
//     "input_schema": {...}
//   },
//   {
//     "type": "code_review",
//     "name": "代码审查",
//     "description": "自动审查代码质量、安全性和性能",
//     "input_schema": {...}
//   }
// ]
```

### 前端：根据 Schema 生成表单

```javascript
// 根据 input_schema 自动生成表单
function generateForm(schema) {
  return Object.entries(schema.properties).map(([key, prop]) => {
    if (prop.type === 'string' && prop.enum) {
      return <Select name={key} options={prop.enum} />;
    }
    if (prop.type === 'number') {
      return <NumberInput name={key} min={prop.min} max={prop.max} />;
    }
    return <TextInput name={key} />;
  });
}
```

### 前端：执行工作流

```javascript
const response = await fetch('/api/v1/workflow/execute', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    workflow_type: 'oj_problem_generation',
    input: {
      topic: '二叉树遍历',
      difficulty: 'medium',
      language: 'cpp',
      testcase_count: 10
    }
  })
});

const { workflow_id } = (await response.json()).data;

// 监听进度
const es = new EventSource(`/api/v1/workflow/${workflow_id}/stream`);
es.addEventListener('workflow_completed', (e) => {
  const { result } = JSON.parse(e.data);
  console.log('完成！', result);
});
```

---

## 📝 实施计划

### Phase 1: 核心框架（2-3 小时）
1. 定义工作流接口
2. 实现工作流注册表
3. 实现工作流引擎
4. 统一 API 接口

### Phase 2: 迁移现有工作流（1-2 小时）
1. 将 OJ 题目生成改造为插件
2. 测试兼容性

### Phase 3: 扩展新工作流（按需）
1. 代码审查工作流
2. 作业批改工作流
3. 其他工作流...

---

**文档版本**: v1.0.0
**最后更新**: 2026-03-06
