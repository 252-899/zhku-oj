# AI Agent/Module 业务接口接入指南

本文档说明后续如何把 Agent 或 Module 接入到 HTTP 接口层。当前约定是：正式业务功能使用独立业务接口，对外不暴露 `task_kind`、`module_type`、`metadata.input` 这类运行时网关字段。

## 接入原则

正式业务接口采用一业务一入口：

```text
POST /api/v1/ai/oj-question-gen/stream
POST /api/v1/ai/student/learning-diagnosis/stream
POST /api/v1/ai/teacher/assignment-analysis/stream
```

通用入口保留为调试和兼容入口：

```text
POST /api/v1/ai/agent/chat
POST /api/v1/ai/agent/chat/stream
```

前端接正式业务接口时，不需要知道内部的 Agent/Module 类型和运行时结构。

## 接入流程

1. 在 `pkg/app/api-server/server/Doc` 下新增请求 DTO。

示例：

```go
type OJQuestionGenStreamRequestDoc struct {
    SessionID              string   `json:"session_id,omitempty"`
    RoundID                string   `json:"round_id,omitempty"`
    Message                string   `json:"message,omitempty"`
    KnowledgePoints        []string `json:"knowledge_points" binding:"required"`
    Difficulty             string   `json:"difficulty,omitempty"`
    Count                  int      `json:"count,omitempty"`
    TestcaseCount          int      `json:"testcase_count,omitempty"`
    StandardLanguage       string   `json:"standard_language,omitempty"`
    AdditionalRequirements string   `json:"additional_requirements,omitempty"`
}
```

2. 在 `server` 包新增业务 Handler，绑定业务 DTO。

Handler 只做三件事：

- 校验 HTTP 请求参数
- 把业务 DTO 转成 `aiapi.RuntimeRequest`
- 调用内部 Gateway 并转发 SSE

示例：

```go
runtimeReq := &aiapi.RuntimeRequest{
    SessionID:  strings.TrimSpace(req.SessionID),
    RoundID:    strings.TrimSpace(req.RoundID),
    TaskKind:   aiapi.TaskKindModule,
    TaskType:   ojquestiongen.CanonicalModuleType,
    ModuleType: ojquestiongen.CanonicalModuleType,
    Message:    strings.TrimSpace(req.Message),
    Input: map[string]any{
        "knowledge_points":        req.KnowledgePoints,
        "difficulty":              strings.TrimSpace(req.Difficulty),
        "count":                   req.Count,
        "testcase_count":          req.TestcaseCount,
        "standard_language":       strings.TrimSpace(req.StandardLanguage),
        "additional_requirements": strings.TrimSpace(req.AdditionalRequirements),
    },
}
runtimeReq.Normalize()
s.streamRuntimeModuleViaGateway(c.Request.Context(), c, runtimeReq)
```

3. 在路由注册处挂载业务路径。

示例：

```go
ojQuestionGenGroup := g.Group("/ai/oj-question-gen").Use(middleware.JWTMiddleware())
{
    ojQuestionGenGroup.POST("/stream", s.OJQuestionGenStream)
}
```

4. 给 Handler 添加 Swagger 注释。

正式业务接口的 Swagger 请求体必须引用业务 DTO，不要引用通用 `ChatRequest`。

示例：

```go
// @Param request body doc.OJQuestionGenStreamRequestDoc true "OJ 题目生成请求"
// @Router /ai/oj-question-gen/stream [post]
```

5. 补充测试。

测试应至少覆盖：

- 未登录返回 401
- 参数错误返回 400
- Handler 正确生成 `RuntimeRequest.TaskKind`
- Handler 正确生成 `RuntimeRequest.ModuleType` 或 `AssistantType`
- Handler 正确把业务字段放入 `RuntimeRequest.Input`

## Module 接入模板

Module 适合长流程业务，例如出题、学情分析、作业报告生成。HTTP 层使用业务 DTO，内部调用 `ModuleGateway`。

```text
前端
  -> 业务 Handler
  -> aiapi.RuntimeRequest{TaskKind: module, ModuleType: xxx, Input: ...}
  -> ModuleGateway
  -> Module/Coordinator
  -> SSE StreamEvent
```

业务 Handler 可以复用：

```go
s.streamRuntimeModuleViaGateway(ctx, c, runtimeReq)
```

## Assistant/ReactAgent 接入模板

Assistant 或 ReactAgent 适合对话式任务、工具调用型任务。HTTP 层仍然应该使用业务 DTO，内部调用 `AssistantGateway`。

```text
前端
  -> 业务 Handler
  -> aiapi.RuntimeRequest{TaskKind: assistant, AssistantType: xxx, Input/Metadata: ...}
  -> AssistantGateway
  -> Assistant/ReactAgent
  -> SSE StreamEvent 或聚合 JSON
```

如果是流式业务接口，可以复用：

```go
s.streamRuntimeAssistantViaGateway(ctx, c, runtimeReq)
```

## 当前 OJ 题目生成接口

正式业务接口：

```text
POST /api/v1/ai/oj-question-gen/stream
```

请求体：

```json
{
  "message": "生成 2 道数组相关 OJ 编程题",
  "knowledge_points": ["数组", "循环"],
  "difficulty": "easy",
  "count": 2,
  "testcase_count": 3,
  "standard_language": "python",
  "additional_requirements": "题目场景贴近校园生活"
}
```

说明：

- `knowledge_points`：必填，题目知识点。
- `difficulty`：可选，默认由模块处理，支持 `easy`、`medium`、`hard`。
- `count`：可选，生成题目数量，模块内限制为 1 到 10。
- `testcase_count`：可选，每题测试用例数量，模块内限制为 1 到 10。
- `standard_language`：可选，标准答案程序语言，默认 `python`。
- `additional_requirements`：可选，额外出题要求。

## 不推荐的做法

不要让正式业务前端直接调用：

```json
{
  "task_kind": "module",
  "module_type": "oj_question_gen",
  "metadata": {
    "input": {}
  }
}
```

这类结构属于内部运行时网关协议，适合调试和兼容，不适合作为正式业务接口契约。
