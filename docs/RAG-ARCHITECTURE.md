# RAG 总体框架设计（统一入口/出口，支持高扩展）

## 1. 文档定位

- 目标：定义本项目 RAG 的统一框架，作为后续 TODO 开发基线。
- 范围：`pkg/ai` 侧 RAG 系统设计、Agent 适配规范、落地阶段计划。
- 非范围：本阶段不绑定具体向量库厂商，不展开离线建库全流程实现细节。

---

## 2. 现状与问题

当前代码已具备“基础 RAG 接入点”：

1. `pkg/ai/agent/spec.go` 存在 `Retriever retriever.Retriever` 字段。
2. `pkg/ai/agent/runner.go` 在执行前会调用 `Retrieve`，并将文档注入消息。

但现状仍有三个核心问题：

1. 没有独立 RAG 系统模块，缺少统一入口和统一出口。
2. Agent 对接的是 Eino `retriever.Retriever`，不是项目内部统一 RAG 协议。
3. 路由、召回、重排等能力无法在同一框架内标准化扩展。

---

## 3. 设计目标

### 3.1 单一原则

1. 单一入口：所有调用统一走 `rag.Service.Query(ctx, req)`。
2. 单一出口：所有结果统一返回 `rag.Response`。
3. Agent 依赖单一接口：Agent 不直接依赖多种召回实现。

### 3.2 可扩展原则

1. 内部模块可插拔：Router / Retriever / Reranker / Assembler。
2. 支持多策略并行：vector / keyword / hybrid。
3. 支持按场景和上下文动态路由。

### 3.3 工程原则

1. 失败可降级：RAG 失败不阻断 Agent 主流程。
2. 可观测：每次查询可追踪路由决策、命中和耗时。
3. 可演进：兼容现有 Eino `retriever` 接口，渐进迁移。

---

## 4. 目标架构

```text
Caller (AgentRunner / Workflow / API)
                |
                v
        +------------------+
        |   rag.Service    |   <- 统一入口
        +------------------+
                |
                v
        +------------------+      +------------------+
        |  ContextBuilder  | ---> |   QueryRouter    |
        +------------------+      +------------------+
                |                         |
                |                 生成 RoutePlan
                v                         v
        +------------------------------------------+
        |          Retriever Pipeline              |
        | vector / keyword / hybrid / custom      |
        +------------------------------------------+
                         |
                         v
               +------------------+
               | Merger + Deduper |
               +------------------+
                         |
                         v
               +------------------+
               |     Reranker     |
               +------------------+
                         |
                         v
               +------------------+
               |    Assembler     |
               +------------------+
                         |
                         v
                 rag.Response <- 统一出口
```

---

## 5. 模块分层与职责

### 5.1 对外 Facade 层（稳定接口）

`rag.Service` 是唯一对外调用入口：

1. 接收 `rag.Request`。
2. 调度内部链路。
3. 返回 `rag.Response`。

### 5.2 编排层（Orchestrator）

负责按固定流程串联：

1. 构建上下文。
2. 路由策略决策。
3. 调用一个或多个召回器。
4. 融合去重、重排、裁剪。
5. 输出统一结构。

### 5.3 能力层（可插拔组件）

1. `QueryRouter`：决定用哪个策略和索引。
2. `Retriever`：实际召回文档。
3. `Reranker`：重排文档顺序。
4. `Assembler`：组装 snippets/citations/trace。

### 5.4 基础设施层

1. 向量存储适配。
2. 关键词检索适配。
3. Embedding/Rerank 模型适配。
4. 日志、指标、追踪。

---

## 6. 目录规划（建议）

```text
pkg/ai/components/rag/
  types.go
  service.go
  context.go
  router.go
  retriever.go
  reranker.go
  merger.go
  assembler.go
  options.go
  noop/
    service.go
  adapter/
    eino_retriever.go
    agent.go
  impl/
    router/
      rule_router.go
    retriever/
      vector_retriever.go
      keyword_retriever.go
      hybrid_retriever.go
    reranker/
      model_reranker.go
```

---

## 7. 统一协议定义

### 7.1 统一入口

```go
package rag

import "context"

type Service interface {
    Query(ctx context.Context, req *Request) (*Response, error)
}
```

```go
type Request struct {
    Query     string         `json:"query"`
    UserID    string         `json:"user_id,omitempty"`
    SessionID string         `json:"session_id,omitempty"`
    AgentType string         `json:"agent_type,omitempty"`
    Scene     string         `json:"scene,omitempty"`
    ProblemID int64          `json:"problem_id,omitempty"`
    Metadata  map[string]any `json:"metadata,omitempty"`
    Options   QueryOptions   `json:"options,omitempty"`
}

type QueryOptions struct {
    TopK         int      `json:"top_k,omitempty"`
    MinScore     float64  `json:"min_score,omitempty"`
    RecallModes  []string `json:"recall_modes,omitempty"`
    IndexHint    string   `json:"index_hint,omitempty"`
    SubIndexHint string   `json:"sub_index_hint,omitempty"`
    MaxTokens    int      `json:"max_tokens,omitempty"`
    EnableRerank *bool    `json:"enable_rerank,omitempty"`
}
```

### 7.2 统一出口

```go
type Response struct {
    Query     string       `json:"query"`
    Documents []Document   `json:"documents"`
    Snippets  []Snippet    `json:"snippets"`
    Citations []Citation   `json:"citations"`
    Trace     Trace        `json:"trace"`
}

type Document struct {
    ID       string         `json:"id"`
    Content  string         `json:"content"`
    Score    float64        `json:"score"`
    Source   string         `json:"source"`
    ChunkID  string         `json:"chunk_id,omitempty"`
    Title    string         `json:"title,omitempty"`
    URL      string         `json:"url,omitempty"`
    MetaData map[string]any `json:"meta_data,omitempty"`
}

type Snippet struct {
    Index   int     `json:"index"`
    DocID   string  `json:"doc_id"`
    Content string  `json:"content"`
    Score   float64 `json:"score"`
}

type Citation struct {
    DocID  string  `json:"doc_id"`
    Title  string  `json:"title,omitempty"`
    URL    string  `json:"url,omitempty"`
    Source string  `json:"source,omitempty"`
    Score  float64 `json:"score"`
}

type Trace struct {
    RequestID      string          `json:"request_id"`
    RouterDecision string          `json:"router_decision"`
    RetrieverStats []RetrieverStat `json:"retriever_stats"`
    TotalLatencyMS int64           `json:"total_latency_ms"`
    Error          string          `json:"error,omitempty"`
}

type RetrieverStat struct {
    Name      string `json:"name"`
    HitCount  int    `json:"hit_count"`
    LatencyMS int64  `json:"latency_ms"`
}
```

---

## 8. 内部扩展点接口

```go
type ContextBuilder interface {
    Build(ctx context.Context, req *Request) (*Request, error)
}

type QueryRouter interface {
    Route(ctx context.Context, req *Request) (*RoutePlan, error)
}

type RoutePlan struct {
    Modes      []string
    Index      string
    SubIndexes []string
    TopK       int
    MinScore   float64
}

type Retriever interface {
    Name() string
    Retrieve(ctx context.Context, req *Request, plan *RoutePlan) ([]Document, error)
}

type Reranker interface {
    Rerank(ctx context.Context, req *Request, docs []Document) ([]Document, error)
}

type Merger interface {
    Merge(ctx context.Context, docs [][]Document) ([]Document, error)
}

type Assembler interface {
    Assemble(ctx context.Context, req *Request, docs []Document, trace Trace) (*Response, error)
}
```

---

## 9. Agent 适配规范

### 9.1 强制规则

1. Agent 不直接依赖向量库 SDK。
2. Agent 不直接实现召回路由逻辑。
3. Agent 只调用 RAG 统一接口（或其标准适配器）。

### 9.2 适配模式

#### 模式 A（目标态，推荐）

`AgentRunner` 直接依赖 `rag.Service`，构造 `rag.Request`，获取 `rag.Response` 后注入模型消息。

#### 模式 B（兼容态）

通过 `adapter.EinoRetrieverAdapter` 将 `rag.Service` 适配为 Eino 的 `retriever.Retriever`。

```go
type EinoRetrieverAdapter struct {
    svc rag.Service
}

func (a *EinoRetrieverAdapter) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
    resp, err := a.svc.Query(ctx, &rag.Request{Query: query})
    if err != nil {
        return nil, err
    }

    out := make([]*schema.Document, 0, len(resp.Documents))
    for _, d := range resp.Documents {
        out = append(out, (&schema.Document{
            ID:       d.ID,
            Content:  d.Content,
            MetaData: d.MetaData,
        }).WithScore(d.Score))
    }
    return out, nil
}
```

### 9.3 Agent 调 RAG 必须透传字段

1. `UserID`
2. `SessionID`
3. `AgentType`
4. `Scene`
5. `ProblemID`
6. `Metadata`

说明：该信息是路由和召回过滤的关键输入。

---

## 10. 请求时序（Agent 场景）

```text
Client -> AgentGateway -> AgentRunner
AgentRunner -> RAG.Service.Query(req)
RAG.Service -> Router -> Retriever(s) -> Merger -> Reranker -> Assembler
RAG.Service -> AgentRunner (Response)
AgentRunner -> LLM (with RAG snippets)
LLM -> AgentRunner -> AgentGateway -> Client
```

失败策略：

1. RAG 内部错误：返回空文档 + trace.error，Agent 继续回答。
2. RAG 总超时：触发降级（无 RAG）。
3. 召回为空：返回空文档，不视为错误。

---

## 11. 配置模型建议

```yaml
AI:
  RAG:
    Enable: true
    DefaultTopK: 6
    DefaultMinScore: 0.35
    MaxSnippetTokens: 1200
    TimeoutMS: 1200
    Router:
      DefaultMode: hybrid
      SceneRules:
        problem_page: hybrid
    Retrievers:
      Vector:
        Enable: true
        Provider: milvus
      Keyword:
        Enable: true
        Provider: elasticsearch
    Reranker:
      Enable: true
      Provider: bge-reranker
    Fallback:
      OnErrorReturnEmpty: true
```

---

## 12. 可观测性要求

### 12.1 日志

必打字段：

1. `request_id`
2. `session_id`
3. `agent_type`
4. `scene`
5. `router_decision`
6. `retriever_hits`
7. `latency_ms`
8. `error`

### 12.2 指标

1. `rag_request_total`
2. `rag_request_error_total`
3. `rag_request_latency_ms`（P50/P95/P99）
4. `rag_empty_result_total`
5. `rag_retriever_hit_count`

### 12.3 追踪

1. Router 耗时。
2. 每个 Retriever 耗时。
3. Rerank 耗时。
4. 总耗时。

---

## 13. 与现有代码的迁移方案

### 13.1 迁移原则

1. 先新增，不破坏存量流程。
2. 先兼容 `AgentSpec.Retriever`，再演进到 Agent 直接依赖 `rag.Service`。
3. 分阶段交付，每阶段可独立验收。

### 13.2 分阶段实施

#### Phase 1：接口与骨架

1. 建 `pkg/ai/components/rag` 基础接口与类型。
2. 提供 `noop.Service`。
3. 提供 `adapter.EinoRetrieverAdapter`。

DoD：

1. 代码可编译。
2. Agent 可通过 Noop + Adapter 跑通。

#### Phase 2：上下文透传

1. 扩展 `pkg/ai/agent/types.go` 的 `RunRequest`。
2. `agent_gateway/service.go` 透传 `Scene/ProblemID/Metadata`。
3. `runner` 构造完整 `rag.Request`。

DoD：

1. 路由可基于 `scene/problem_id` 生效。

#### Phase 3：多路召回与重排

1. 实现 Router。
2. 实现至少 2 种 Retriever（vector + keyword）。
3. 实现 Merger/Deduper + 可选 Reranker。

DoD：

1. 支持 `RecallModes`。
2. 响应包含来源和分值。

#### Phase 4：观测与质量

1. 指标、日志、追踪接入。
2. 增加最小回归数据集。

DoD：

1. 可查询 P95 延迟和空召回率。

#### Phase 5：业务增强

1. 场景化组装模板（解题/讲解/纠错）。
2. 对外返回 citations。
3. 灰度开关和回滚策略。

---

## 14. TODO 任务分解（可直接建工单）

### 14.1 基础任务

1. `RAG-001` 新建 `rag/types.go`。
2. `RAG-002` 新建 `rag/service.go` 与 `DefaultService`。
3. `RAG-003` 新建 `rag/noop/service.go`。
4. `RAG-004` 新建 `rag/adapter/eino_retriever.go`。

### 14.2 对接任务

1. `RAG-101` 扩展 `agent.RunRequest` 上下文字段。
2. `RAG-102` 网关到 Runner 全链路透传。
3. `RAG-103` `problem_solver` 接入 RAG Adapter。

### 14.3 策略任务

1. `RAG-201` 实现 `rule_router`。
2. `RAG-202` 实现 `vector_retriever`。
3. `RAG-203` 实现 `keyword_retriever`。
4. `RAG-204` 实现 `hybrid merger + deduper`。
5. `RAG-205` 实现 `reranker`（可开关）。

### 14.4 质量任务

1. `RAG-301` 指标埋点。
2. `RAG-302` 结构化日志。
3. `RAG-303` 回归样本集与基线报告。

---

## 15. 风险与约束

1. 风险：多路召回会增加延迟。
2. 对策：并发召回 + 超时控制 + TopK 限制。
3. 风险：召回质量不稳定。
4. 对策：引入重排、评测集和灰度策略。
5. 风险：上下文过长导致 token 膨胀。
6. 对策：snippet 截断和 max token 预算。

---

## 16. 文档版本

- Version: `v1.0`
- Status: `Approved for TODO implementation`
- Updated: `2026-03-06`

