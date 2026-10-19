# 通用工作流系统重构完成

## 重构概述

已将 OJ 题目生成工作流从固定实现重构为通用的、可扩展的工作流编排系统。

## 核心架构

### 1. 三层架构

```
API Layer (server_ai_workflow.go)
    ↓
Engine Layer (workflow/engine.go)
    ↓
Workflow Implementations (workflow/definitions/*.go)
```

### 2. 核心组件

#### Registry (workflow/registry.go)
- 工作流注册中心
- 管理所有工作流定义
- 支持动态注册和查询

#### Engine (workflow/engine.go)
- 工作流执行引擎
- 统一的执行接口
- 进度回调和事件发布

#### Definition Interface (workflow/types.go)
```go
type Definition interface {
    GetType() WorkflowType
    GetName() string
    GetDescription() string
    GetInputSchema() *InputSchema
    ValidateInput(input map[string]interface{}) error
    BuildGraph(ctx context.Context, input map[string]interface{}) (compose.Runnable[map[string]interface{}, interface{}], error)
    FormatOutput(output interface{}) (interface{}, error)
}
```

### 3. 工作流类型

已定义 13 种工作流类型：
- `oj_problem_generation` - OJ 编程题生成
- `multiple_choice_generation` - 选择题生成
- `fill_blank_generation` - 填空题生成
- `essay_generation` - 问答题生成
- `code_review` - 代码审查
- `code_refactor` - 代码重构
- `code_optimize` - 代码优化
- `assignment_grade` - 作业批改
- `essay_grade` - 问答题批改
- `data_analysis` - 数据分析
- `report_generation` - 报告生成
- `custom` - 自定义工作流

## API 变更

### 旧版 API（已废弃）

```json
POST /api/v1/ai/workflow/start
{
  "workflow_type": "oj_problem",
  "topic": "二叉树遍历",
  "difficulty": "medium",
  "tags": ["树", "递归"],
  "language": "cpp",
  "testcase_count": 10,
  "time_limit": 1000,
  "memory_limit": 256
}
```

### 新版 API（通用格式）

```json
POST /api/v1/ai/workflow/start
{
  "workflow_type": "oj_problem_generation",
  "input": {
    "topic": "二叉树遍历",
    "difficulty": "medium",
    "tags": ["树", "递归"],
    "language": "cpp",
    "testcase_count": 10,
    "time_limit": 1000,
    "memory_limit": 256
  },
  "config": {
    "timeout": 300,
    "retry_on_failure": true,
    "max_retries": 3
  },
  "metadata": {
    "source": "web_ui",
    "version": "v1.2.0"
  }
}
```

## 新增工作流示例

### 代码审查工作流

```json
{
  "workflow_type": "code_review",
  "input": {
    "code": "...",
    "language": "go",
    "focus_areas": ["security", "performance", "best_practices"]
  }
}
```

### 作业批改工作流

```json
{
  "workflow_type": "assignment_grade",
  "input": {
    "assignment_id": "...",
    "submission_code": "...",
    "rubric": {...}
  }
}
```

## 扩展指南

### 添加新工作流类型

1. 在 `workflow/types.go` 中添加新的 WorkflowType 常量
2. 在 `workflow/definitions/` 下创建新的工作流实现
3. 实现 `Definition` 接口的所有方法
4. 在 `definitions/init.go` 中注册新工作流

示例：

```go
// 1. 定义类型
const WorkflowTypeCodeReview WorkflowType = "code_review"

// 2. 实现工作流
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
                Type: "string",
                Title: "代码内容",
                Description: "需要审查的代码",
            },
            "language": {
                Type: "string",
                Title: "编程语言",
                Enum: []string{"go", "java", "python", "cpp"},
            },
        },
        Required: []string{"code", "language"},
    }
}

// ... 实现其他方法

// 3. 注册
func InitWorkflows(...) error {
    registry.Register(NewCodeReviewWorkflow(chatModel))
    // ...
}
```

## 兼容性

- 旧版 API 格式仍然支持（通过类型转换）
- `workflow_type: "oj_problem"` 自动映射到 `"oj_problem_generation"`
- 现有的 OJ 工作流实现完全复用，无需修改

## 文件清单

### 新增文件
- `pkg/ai/workflow/types.go` - 核心类型定义
- `pkg/ai/workflow/registry.go` - 工作流注册中心
- `pkg/ai/workflow/engine.go` - 执行引擎
- `pkg/ai/workflow/definitions/oj_problem_gen.go` - OJ 工作流实现
- `pkg/ai/workflow/definitions/init.go` - 工作流初始化

### 修改文件
- `pkg/models/ai_workflow.go` - 添加通用请求类型
- `pkg/app/api-server/server/server_ai_workflow.go` - 更新 API handler
- `pkg/app/api-server/service/service_ai_workflow.go` - 更新 Service 层
- `pkg/app/api-server/service/service.go` - 添加工作流注册

## 测试建议

1. 测试 OJ 工作流（确保兼容性）
```bash
curl -X POST http://localhost:8080/api/v1/ai/workflow/start \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_type": "oj_problem_generation",
    "input": {
      "topic": "快速排序",
      "difficulty": "medium",
      "language": "cpp"
    }
  }'
```

2. 测试工作流列表
```bash
curl http://localhost:8080/api/v1/ai/workflow/types \
  -H "Authorization: Bearer <token>"
```

## 下一步计划

1. 实现其他工作流类型（代码审查、作业批改等）
2. 添加工作流模板系统
3. 支持工作流组合和嵌套
4. 添加工作流可视化编辑器
5. 实现工作流版本管理

---

**重构完成时间**: 2026-03-06
**版本**: v1.2.0
