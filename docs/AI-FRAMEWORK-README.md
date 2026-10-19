# AI 框架介绍文档

## 1. 整体架构

`pkg/ai` 是系统唯一的 AI 宿主目录，采用插件化架构，核心分为四层：

```
pkg/ai/
├── api/                  # 核心类型定义、错误、请求/响应结构
├── infra/               # 基础设施层
│   ├── llm/            # LLM 模型工厂（配置 → ChatModel 实例）
│   ├── eino/           # Eino Agent 运行时（Runner + EventMapper + Prompt）
│   └── search/         # 搜索工具（Tavily 客户端 + Tool Provider）
├── runtime/             # 运行时支撑
│   ├── gateway/         # AssistantGateway（入口网关）
│   ├── session/        # 会话存储（MongoDB / Memory）
│   ├── task/           # 任务管理（MemoryTaskManager）
│   ├── context/        # 上下文管理（ContextManager）
│   ├── stream/         # 流式事件构造辅助
│   └── audit/          # 审计收集器（SessionCollector）
├── assistants/         # Assistant 插件注册表
│   ├── registry.go     # Registry：注册/查找 Assistant
│   └── websearch/      # WebSearchAssistant 实现
└── compat/
    └── agentbridge/    # 兼容层（ChatRequest ↔ RuntimeRequest）
```

---

## 2. 核心类型

### 2.1 RuntimeRequest

运行时请求，是整个框架的通用请求格式：

```go
type RuntimeRequest struct {
    SessionID     string         // 会话唯一 ID（自动生成）
    RoundID       string         // 本轮对话唯一 ID（自动生成）
    UserID        string         // 用户 ID（从 JWT 提取）
    TaskKind      TaskKind       // assistant | module
    TaskType      string         // 具体任务类型，如 "websearch"
    AssistantType string         // Assistant 类型别名
    Message       string         // 用户消息
    Input         map[string]any // 扩展参数
    Metadata      map[string]any // 元数据
}
```

### 2.2 RuntimeStreamEvent

流式事件，所有 AI 事件统一格式：

```go
type RuntimeStreamEvent struct {
    Type         EventType  // meta | state | text | thinking | tool_call |
                           // tool_result | reference | done | error
    SessionID    string
    RoundID      string
    TaskType     string
    Stage        string     // running | searching | answering | completed
    Data         any        // 事件负载
    ErrorMessage string
    Timestamp    int64
}
```

---

## 3. 会话存储（Session）

### 3.1 接口定义

```go
type SessionService interface {
    CreateIfNeeded(ctx context.Context, req *RuntimeRequest) error
    SaveQuestion(ctx context.Context, req *RuntimeRequest) error
    SaveResult(ctx context.Context, input *SaveResultInput) error
    LoadRecent(ctx context.Context, sessionID, userID string, limit int) ([]*SessionSummary, error)
}
```

### 3.2 两种实现

| 实现 | 存储位置 | 适用场景 |
|------|----------|----------|
| `MongoSessionService` | MongoDB `ai_sessions` 集合 | 生产环境，持久化 |
| `MemorySessionService` | 内存 Map | 开发/测试环境 |

### 3.3 MongoDB Schema

每条记录对应一个 `SessionID + RoundID` 的一次对话轮次：

| 字段 | 说明 |
|------|------|
| `session_id` | 会话 ID |
| `round_id` | 本轮 ID |
| `user_id` | 用户 ID |
| `agent_type` | Assistant 类型 |
| `question` | 用户提问 |
| `answer` | AI 回答 |
| `thinking` | 推理过程 |
| `reference` | 引用列表 |
| `tools` | 工具调用记录 |
| `status` | running / completed / failed / stopped |

**索引策略：**
- `(session_id, user_id, created_at)` — 按时间倒序查询最近会话
- `(session_id, round_id)` — 按 round_id 精确定位
- `(session_id, status)` — 筛选运行中任务

### 3.4 写入流程

每次请求分为两个写入节点：
1. **`SaveQuestion`** — 请求进入时写入 question + status=running
2. **`SaveResult`** — 请求完成时更新 answer/thinking/reference/tools + status=(completed/failed/stopped)

---

## 4. 上下文管理机制（Context）

### 4.1 ContextManager 接口

```go
type ContextManager interface {
    Build(ctx context.Context, req *RuntimeRequest) (*Snapshot, error)
}
```

### 4.2 构建流程

`ContextManager.Build` 从会话存储加载最近 **10 条**已完成的历史记录（默认），生成 `Snapshot`：

```go
type Snapshot struct {
    Messages []HistoryMessage  // [{role: "user", content: "..."}, {role: "assistant", content: "..."}, ...]
}
```

这些消息会在构建 Eino Agent 输入时拼接到对话最前面，作为上下文注入 LLM。

### 4.3 对话历史限制

在 `newAIRuntimeComponents()` 中创建时传入 limit=10，即最多保留最近 10 轮。

---

## 5. 任务管理机制（Task）

### 5.1 TaskManager 接口

```go
type TaskManager interface {
    RegisterRoot(root *RootTaskRuntime) error
    GetRoot(sessionID string) (*RootTaskRuntime, bool)
    GetRound(sessionID, roundID string) (*RootTaskRuntime, bool)
    HasRunning(sessionID string) bool
    UpdateRootState(sessionID, roundID, stage string, status TaskStatus) error
    AddChild(sessionID, roundID string, child *ChildTaskRuntime) error
    UpdateChild(sessionID, roundID string, child *ChildTaskRuntime) error
    ListChildren(sessionID, roundID string) []*ChildTaskRuntime
    Stop(ctx context.Context, sessionID, roundID, userID string) error
    RemoveRoot(sessionID, roundID string)
}
```

### 5.2 内存结构

```
roots: map[sessionID] -> rootEntry
                          ├── root: RootTaskRuntime
                          └── children: map[childID] -> ChildTaskRuntime
```

- **RootTaskRuntime**：根任务，元信息 + `context.CancelFunc`
- **ChildTaskRuntime**：子任务，用于工作流节点编排
- **防重**：同一个 `sessionID` 只允许一个 running 任务（`ErrTaskAlreadyRunning`）

### 5.3 Stop 能力

`Stop` 会：
1. 将根任务状态置为 `stopped`
2. 调用根任务的 `context.CancelFunc` 取消当前执行
3. 遍历所有子任务，同样取消

---

## 6. 流式接口与直接返回接口

### 6.1 核心入口：AssistantGateway

```go
type AssistantGateway interface {
    Stream(ctx context.Context, req *RuntimeRequest) (<-chan *RuntimeStreamEvent, error)
    Stop(ctx context.Context, sessionID, roundID, userID string) error
}
```

两种接口都通过 `Stream` 返回的 channel 实现：
- **SSE 接口**（`/ai/agent/chat/stream`）— 实时推送每个事件到 HTTP 响应
- **直接返回接口**（`/ai/agent/chat`）— 遍历完 channel 后聚合成 `ChatResponse`

### 6.2 SSE 推送（非阻塞）

Server 层通过 `text/event-stream` 格式推送每个事件：

```
event: message
data: {"type":"meta","session_id":"...","round_id":"...","timestamp":...}

event: message
data: {"type":"thinking","session_id":"...","content":"..."}

event: message
data: {"type":"tool_call","data":{"name":"web_search","query":"..."}}

event: message
data: {"type":"tool_result","data":{"name":"web_search","result_count":5}}

event: message
data: {"type":"text","content":"回答内容"}

event: message
data: {"type":"reference","data":[...]}

event: message
data: {"type":"done","data":{"status":"completed"}}
```

### 6.3 直接返回聚合

`chatViaAssistant` 遍历完整流后，将所有事件聚合：

| 事件类型 | 聚合到响应字段 |
|----------|----------------|
| `EventThinking` | `Thinking` |
| `EventText` | `Answer` |
| `EventReference` | `Reference[]` |
| `EventToolCall/Result` | `Tools[]` |
| `EventDone` | `Status` |

---

## 7. 调用 AI 的基础设施

### 7.1 LLM 工厂（infra/llm/factory.go）

`ModelFactory` 接口从 `conf.Config.AI.Chat` 读取配置（BaseURL、APIKey、Model），使用 Eino OpenAI 适配器创建 ChatModel 实例。配置未完成时返回错误，避免空指针。

### 7.2 Eino Agent Runner（infra/eino/runner.go）

使用 `cloudwego/eino` 的 ADK（Agent Development Kit）：

```
ChatModelRunnerFactory
    └── NewRunner(ctx, tools)
            ├── modelFactory.NewToolCallingChatModel()  → ChatModel
            ├── adk.NewChatModelAgent(config)          → Agent
            └── adk.NewRunner(runnerConfig)            → Runner
                    └── RunStream(messages)            → AsyncIterator[*AgentEvent]
```

每次请求独立创建 Runner（因为 ChatModel 和 Agent 是无状态的），通过 `EnableStreaming: true` 开启流式输出。

### 7.3 EventMapper（infra/eino/event_mapper.go）

将 Eino 的 `AgentEvent` 映射为框架的 `RuntimeStreamEvent`：

| Eino AgentEvent | 映射为框架事件 |
|-----------------|----------------|
| Assistant 消息 + ToolCalls | `EventToolCall` |
| Assistant 消息 + ReasoningContent | `EventThinking` |
| Assistant 消息 + Content | `EventText` |
| Tool 消息（web_search 结果）| `EventToolResult` |

### 7.4 搜索工具（infra/search/provider.go）

使用 Tavily API，通过 `toolutils.InferTool` 将 Go 函数签名的 `Search(ctx, query, topK)` 自动转换为 Eino Tool。

---

## 8. WebSearchAgent 完整创建流程

### 8.1 组件装配（service.go → newAIRuntimeComponents）

```
newAIRuntimeComponents()
│
├── aiassistants.NewRegistry()                         → registry
├── MongoDB → runtimesession.NewMongoSessionService()   → sessionService
├── runtimesession.NewMemorySessionService()            → (fallback)
├── runtimetask.NewMemoryTaskManager()                  → taskManager
├── runtimecontext.NewManager(sessionService, 10)        → contextManager
├── llm.NewConfigModelFactory()                        → modelFactory
├── infraeino.NewStaticPromptBuilder()                 → promptBuilder
├── infraeino.NewWebSearchEventMapper()                → eventMapper
├── infraeino.NewChatModelRunnerFactory(...)           → runnerFactory
├── toolsearch.ConfigFromEnv()                         → tavilyConfig
└── toolsearch.NewProvider(toolsearch.NewHTTPClient(...))
        │
        └── registry.Register(websearch.NewAssistant(
                sessionService,
                taskManager,
                contextManager,
                toolProvider,
                runnerFactory,
                eventMapper,
        ))
```

如果 `modelFactory.IsChatConfigured()` 和 `toolProvider.IsConfigured()` 均为 true，注册真实 WebSearchAssistant；否则注册一个 `Unconfigured` Assistant（返回配置错误）。

### 8.2 AssistantRegistry 注册表

```go
func NormalizeType(value string) string {
    switch strings.ToLower(value) {
    case "websearch", "websearchagent", "web_search":
        return "websearch"
    }
    return value
}
```

三个别名统一映射为 `"websearch"`。

### 8.3 WebSearchAssistant.Stream 完整流程

```
1. 请求验证 & 归一化
   └── RuntimeRequest.Normalize()（生成 SessionID/RoundID）

2. 会话层：CreateIfNeeded + SaveQuestion
   └── 写入 MongoDB，status = "running"

3. 任务层：RegisterRoot
   └── MemoryTaskManager 注册根任务
   └── 同一个 SessionID 只能有一个 running 任务

4. 上下文构建：ContextManager.Build
   └── 从 MongoDB 加载最近 10 条历史记录
   └── 生成 Snapshot{Messages: [...]}

5. 启动异步 goroutine：Assistant.run()
   ├── 发送 EventMeta（status=running）
   ├── 执行 execute()
   └── 执行完成后发送 EventDone + SaveResult

6. execute() 执行步骤：
   a. toolProvider.BuildTools()
      └── 构建 web_search tool（Tavily API）
   b. runnerFactory.NewRunner(ctx, tools)
      └── 创建 Eino ChatModelAgent + Runner
   c. runner.RunStream(buildMessages(snapshot, question))
      └── 构建 Eino Message 列表（历史 + 当前提问）
      └── 启动流式推理
   d. 遍历 AsyncIterator：
      ├── 收集 reference（Tool 返回的 JSON）
      ├── eventMapper.Map() → RuntimeStreamEvent[]
      └── emit() 推送到返回 channel
   e. 发送 EventReference（汇总所有引用）
```

### 8.4 SSE 响应完整事件序列（一次 WebSearch 请求）

```
1. meta        {status: "running"}
2. tool_call   {name: "web_search", query: "..."}
3. tool_result {name: "web_search", result_count: 5}
4. thinking    {delta: "正在分析..."}
5. text        {delta: "根据搜索结果..."}
6. ...(多个 text 片段)...
7. reference   [{title: "...", url: "...", snippet: "..."}, ...]
8. done        {status: "completed"}
```

### 8.5 数据持久化时机

| 时机 | 操作 | 数据 |
|------|------|------|
| 请求进入 | `SaveQuestion` | session_id, round_id, user_id, question, status=running |
| 首Token输出时 | `collector.markFirstResponseLocked()` | 记录 first_response_at |
| 每个事件 | `collector.Append(event)` | 实时聚合 text/thinking/reference/tools |
| 执行完成/出错 | `SaveResult` | answer, thinking, reference, tools, first_response_ms, total_response_ms, status |

---

## 9. 关键设计决策

1. **SessionID + RoundID 双键**：SessionID 区分会话，RoundID 区分同一会话中的多轮对话，支持并行和串行多种交互模式。
2. **内存任务管理器**：任务运行态存在内存，状态变更实时响应；持久化通过会话存储异步写入 MongoDB。
3. **Channel 异步流**：从 LLM 到 HTTP SSE 全程异步，通过带缓冲的 channel 连接各环节，避免阻塞。
4. **Collector 聚合模式**：流式事件边推边聚合，最终一次性写入 MongoDB，减少写次数同时保证数据完整。
5. **无状态 Runner**：每次请求创建新的 Eino Runner 和 ChatModel，无内部状态，可安全并发。

---

## 10. 新增 Assistant 指南

1. 在 `pkg/ai/assistants/` 下创建新目录，如 `myassistant/`
2. 实现 `Assistant` 接口：
   ```go
   type Assistant interface {
       Type() string
       Stream(ctx context.Context, req *RuntimeRequest) (<-chan *RuntimeStreamEvent, error)
   }
   ```
3. 在 `assistants/registry.go` 的 `NormalizeType` 添加别名映射
4. 在 `newAIRuntimeComponents()` 中实例化并调用 `registry.Register()`
5. 复用已有的 `SessionService`、`TaskManager`、`ContextManager` 等基础设施
