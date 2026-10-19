# Multi-Agent Core

`pkg/ai/multiagent` 是一层通用的 multi-agent 基础编排包，当前目标是先把 Shannon 风格的 `LeadAgent + LeadAction + spawn agent` 最小闭环落到 Go 项目里。

它不是教师侧业务本身，也不是现有 AI 网关的替代实现。它只负责通用协议和调度：Lead 负责规划，Orchestrator 负责执行动作，具体业务 Agent 和工具由教师侧、学生侧或后续其他业务自己注册。

## 当前定位

当前这层解决三个问题：

1. 统一 Lead 输出格式，避免每个业务自己定义一套 “我要创建哪些 agent、怎么分配任务” 的结构。
2. 统一子 Agent 注册和执行方式，后续教师侧、学生侧或其他模块只需要实现自己的 `AgentRunner`。
3. 先支持可落地的同步编排闭环：Lead 生成计划，系统根据依赖运行 Agent，最后汇总执行结果。

当前这层暂时不做：

1. 不直接接入 HTTP handler。
2. 不直接接入数据库、班级、作业、题库等业务表。
3. 不实现真正的长循环 Lead 决策。
4. 不实现 Agent 间 P2P 通信。
5. 不依赖 Temporal、Redis Streams 或独立 Python Agent 服务。

这些能力后续可以在这层协议稳定后逐步加。

## 核心概念

### LeadPlanner

`LeadPlanner` 就是 LeadAgent 的抽象接口。它接收用户请求和当前可用角色，返回一组 `LeadAction`。

在教师侧场景里，LeadPlanner 可以是：

1. 第一阶段：规则或固定模板实现，便于先打通链路。
2. 第二阶段：LLM 结构化输出实现，让模型根据教师问题动态拆任务。
3. 第三阶段：加入工具可用性、权限、上下文记忆之后的完整 LeadAgent。

### LeadAction

`LeadAction` 不是 MainAgent 本身，它是 LeadAgent 发给编排器的动作指令结构。

可以理解为 LeadAgent 说：

1. `interim_reply`：先告诉用户我准备怎么分析。
2. `revise_plan`：创建或更新任务计划。
3. `spawn_agent`：直接创建一个子 Agent 任务。
4. `reply` / `done`：给出最终回答。

当前实现重点支持 `revise_plan` 和 `spawn_agent`，因为这两个动作能形成基础调度闭环。

### RoleRegistry

`RoleRegistry` 是角色注册表，负责维护 “角色名 -> Agent 工厂函数” 的映射。

例如教师侧可以注册：

1. `class_analyst`：班级概况分析。
2. `assignment_reviewer`：作业完成情况分析。
3. `student_diagnostic`：学生薄弱点诊断。
4. `content_recommender`：题目推荐。
5. `report_writer`：讲评建议汇总。

这样 Lead 只需要输出 `role: "assignment_reviewer"`，Orchestrator 就能找到对应 AgentRunner 来执行。

### Orchestrator

`Orchestrator` 是真正的执行器。它不会自己理解业务问题，而是执行 LeadPlanner 返回的动作。

当前执行流程：

1. 收到 `RunRequest`。
2. 如果请求中没有显式传 roles，就从 `RoleRegistry` 读取可用角色列表。
3. 调用 `LeadPlanner.Plan` 获取 `LeadDecision`。
4. 把 `LeadAction` 转换为内部任务状态。
5. 找出依赖已满足的任务。
6. 按并发上限调用对应角色的 `AgentRunner`。
7. 每个任务完成后写入结果。
8. 所有任务完成后返回 `RunResult`。

### AgentRunner

`AgentRunner` 是子 Agent 的统一执行接口。具体实现可以很简单，也可以很复杂。

例如：

1. MVP 阶段：Go 里直接查询 service / repository，拼一份结构化分析。
2. 中期：调用 LLM，并给它挂载只读业务查询工具。
3. 后期：独立 Agent 服务，支持多轮工具调用和更复杂的推理。

## 教师侧示例

教师提问：

```text
分析一下 23 软件 1 班这次数组作业，并给我讲评建议
```

一个可能的 LeadDecision：

```json
{
  "decision_summary": "teacher assignment analysis",
  "actions": [
    {
      "type": "interim_reply",
      "content": "我会先确认班级和作业范围，再分析完成情况、错误集中点，并整理讲评建议。"
    },
    {
      "type": "revise_plan",
      "create": [
        {
          "id": "T1",
          "role": "class_analyst",
          "description": "确认 23 软件 1 班的班级信息、学生数量和本次数组作业范围"
        },
        {
          "id": "T2",
          "role": "assignment_reviewer",
          "description": "分析本次数组作业的提交率、通过率、未完成人员和高频错误",
          "depends_on": ["T1"]
        },
        {
          "id": "T3",
          "role": "report_writer",
          "description": "基于班级概况和作业分析结果生成课堂讲评建议",
          "depends_on": ["T2"]
        }
      ]
    }
  ]
}
```

执行时：

1. `T1` 没有依赖，先执行 `class_analyst`。
2. `T2` 依赖 `T1`，等 `T1` 完成后执行 `assignment_reviewer`。
3. `T3` 依赖 `T2`，最后执行 `report_writer`。
4. Orchestrator 汇总三个 Agent 的结果，返回给教师侧 service。

## 当前代码结构

```text
pkg/ai/multiagent
├── types.go              # 通用协议：LeadAction、RunRequest、AgentRunner 等
├── registry.go           # 角色注册表：把 role 绑定到 AgentFactory
├── orchestrator.go       # 编排器：调用 LeadPlanner，并按依赖执行子 Agent
├── state.go              # 单次运行状态：任务状态、结果、依赖判断
├── orchestrator_test.go  # 最小闭环测试：依赖顺序、spawn_agent、容量限制
└── README.md             # 当前分析和接入说明
```

## 当前限制

当前版本是基础架构，不是完整 Shannon 复刻：

1. Lead 只规划一次，不会在每个 Agent 完成后再次动态决策。
2. Agent 之间不能互相发消息。
3. `Tool` 和 `ToolParams` 字段先保留协议位置，当前没有执行工具调用。
4. `Cancel` 字段先保留协议位置，当前没有取消已创建任务。
5. 结果汇总现在只是收集 AgentResult，还没有质量门控和最终综合写作 Agent 的强约束。

这些限制是有意保留的，目的是让第一版更容易接入和测试。

## 后续接入建议

教师侧后续可以新增一个独立入口，例如：

```text
POST /api/v1/ai/teacher/copilot/stream
```

推荐链路：

```text
TeacherCopilotHandler
-> TeacherCopilotService
-> multiagent.Orchestrator
-> TeacherLeadPlanner
-> class_analyst / assignment_reviewer / student_diagnostic / content_recommender / report_writer
-> 业务查询工具
```

这个教师侧 multi-agent 入口不建议继续走当前通用 AI 网关。原因是它的核心不是 “把请求转发给某个 module”，而是教师业务入口自己负责权限、班级上下文、工具白名单和 multi-agent 编排。

后续如果要做意图识别，可以放在教师侧入口内部：

```text
TeacherCopilotHandler
-> IntentClassifier
-> TeacherLeadPlanner
-> multiagent.Orchestrator
```

这样既保留了一个教师助手入口，也不会把教师侧业务能力耦合进通用网关。
