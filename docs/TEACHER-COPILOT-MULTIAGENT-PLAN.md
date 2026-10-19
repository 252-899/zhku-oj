# 教师侧 Multi-Agent 基础架构方案

本文档说明教师侧通用对话 Agent 如何基于 `pkg/ai/multiagent` 落地。目标是吸收 Shannon 的 Lead + Action + Spawn 思路，但先采用 Go 内部轻量编排，不引入 Temporal、P2P 消息队列或独立 Python Agent 服务。

## 设计结论

教师侧 Copilot 不走现有 Gateway。它应该作为正式业务入口直接进入教师侧编排服务：

```text
POST /api/v1/ai/teacher/copilot/stream
-> TeacherCopilotHandler
-> TeacherCopilotService
-> multiagent.Orchestrator
-> TeacherSupervisorAgent
-> SubAgents
-> Tools
```

现有 Gateway 可以继续作为兼容和调试入口保留。等正式业务入口稳定后，再考虑降级或移除 Gateway。

## 通用核心

`pkg/ai/multiagent` 提供以下通用能力：

- `LeadAction`：Lead 输出给编排器的结构化操作指令。
- `LeadPlanner`：Lead Agent 生成计划的接口。
- `RoleRegistry`：注册角色和 AgentFactory。
- `Orchestrator`：执行 Lead 计划、检查依赖、容量控制、并发运行子 Agent。
- `AgentRunner`：子 Agent 执行任务的统一接口。

第一版支持这些 action：

```text
interim_reply
revise_plan
spawn_agent
reply
done
```

后续再扩展：

```text
assign_task
send_message
broadcast
quality_gate
tool_call
```

## 教师侧角色

建议第一版定义这些角色：

```text
teacher_lead
class_analyst
assignment_reviewer
student_diagnostic
content_recommender
report_writer
```

职责划分：

- `teacher_lead`：理解老师问题、抽取 slots、生成 `LeadAction`、决定需要哪些子 Agent。
- `class_analyst`：班级人数、活跃度、完成率、AC 率、薄弱知识点。
- `assignment_reviewer`：作业完成情况、未完成名单、低通过题、错误分布。
- `student_diagnostic`：学生个人提交轨迹、薄弱点、练习建议。
- `content_recommender`：题目推荐、难度梯度、知识点覆盖。
- `report_writer`：课堂讲评、提醒文案、教学总结。

## 示例流程

老师提问：

```text
分析一下 23 软件 1 班这次数组作业，并给我讲评建议
```

`TeacherSupervisorAgent` 可以生成：

```json
{
  "decision_summary": "作业分析 + 讲评建议",
  "actions": [
    {
      "type": "interim_reply",
      "content": "我会先确认班级和作业范围，再分析完成情况并整理讲评建议。"
    },
    {
      "type": "revise_plan",
      "create": [
        {
          "id": "T1",
          "role": "class_analyst",
          "description": "确认 23 软件 1 班和这次数组作业的任务范围"
        },
        {
          "id": "T2",
          "role": "assignment_reviewer",
          "description": "分析这次数组作业的完成率、通过率、未完成学生和低通过率题目",
          "depends_on": ["T1"]
        },
        {
          "id": "T3",
          "role": "report_writer",
          "description": "基于作业分析结果生成课堂讲评建议",
          "depends_on": ["T2"]
        }
      ]
    }
  ]
}
```

`Orchestrator` 会按依赖顺序运行：

```text
T1 class_analyst
-> T2 assignment_reviewer
-> T3 report_writer
```

如果多个任务没有依赖，`Orchestrator` 可以并行执行，但受 `MaxConcurrentAgents` 控制。

当前代码中，教师侧主 Agent 已经实现为：

```text
pkg/ai/assistants/teachercopilot
```

关键约定：

- `TeacherSupervisorAgent` 是教师侧 main agent。
- `SupervisorPlanner` 负责把教师问题转换成结构化 `SupervisorPlan`。
- `LLMTeacherSupervisorPlanner` 是当前的 LLM 结构化规划实现。
- `SupervisorPlan.Tasks` 直接复用 `pkg/ai/multiagent.TaskSpec`，不在业务包重复定义 Task。
- `class_id`、`task_id` 只是可选线索，不是硬编码必填项。
- 如果只有“23 软件 1 班”“这次数组作业”这类自然语言线索，主 Agent 应规划 `class_analyst` 先解析范围。

## 工具层原则

子 Agent 不直接访问 repository，应通过工具层调用业务 service。当前工具目录约定如下：

```text
pkg/ai/tools              # 通用工具辅助函数
pkg/ai/tools/oj           # OJ 平台业务只读工具
```

`pkg/ai/tools/oj` 只依赖窄接口和 `pkg/models`，不直接依赖 `pkg/app/api-server/service`。这样后续 `TeacherCopilotService` 可以把现有 `UserService`、`ClazzService`、`ProblemService` 注入工具 Provider，同时避免 service 和 ai 包之间形成循环引用。

工具元信息统一维护在：

```text
pkg/ai/tools/oj/catalog.go
```

`Catalog()` 负责描述工具名称、说明、类别、依赖和只读属性。`BuildTools(ctx)` 从这个目录读取工具说明并转换成 Eino `BaseTool`；`AvailableCatalog()` 根据 Provider 当前注入的依赖过滤可用工具，后续可以直接传给 SupervisorPlanner 或子 Agent prompt。

当前已落第一批只读工具：

```text
oj_get_user_profile       # 按 user_id 或 student_id 查询安全用户资料
oj_get_class_info         # 查询班级信息
oj_list_class_students    # 查询班级学生列表
oj_list_class_tasks       # 查询班级任务列表
oj_get_task_detail        # 查询任务详情和关联题目
oj_get_task_completion    # 查询任务完成情况
oj_search_problems        # 搜索题库题目
oj_get_problem_detail     # 查询题目详情和样例
```

所有教师侧工具都不允许模型自己传 `teacher_id` 或 `operator_user_id`。入口层必须用当前登录用户 ID 创建工具 Provider，例如 `ojtools.NewProvider(deps, actorUserID)`，工具内部再用这个固定 actor 调用已有 service 的权限校验逻辑。

后续再补这些工具：

```text
get_teacher_classes
get_student_submissions
get_problem_stats
recommend_assignment_problems
generate_oj_questions
```

## 落地步骤

1. 实现 `class_analyst`，负责解析班级、作业任务范围和班级概况。
2. 实现 `assignment_reviewer`，负责作业完成情况、未完成名单和低通过率题目分析。
3. 实现 `report_writer`，负责生成课堂讲评建议和课后补练建议。
4. 新增 `TeacherCopilotHandler` 和 `/api/v1/ai/teacher/copilot/stream`。
5. 新增 `TeacherCopilotService`，组装 `ojtools.Provider`、工具目录、`RoleRegistry` 和 `multiagent.Orchestrator`。
6. 使用 SSE 输出 `interim_reply`、任务进度、工具调用过程和最终回复。
7. 再扩展 `student_diagnostic`、`content_recommender` 等角色。

## 暂不做

第一版不引入：

- Temporal child workflow
- Redis Streams
- P2P agent messaging
- 远程 Python Agent 服务
- 无限循环式 Lead 调度

这些可以在教师侧任务变长、需要恢复、需要跨服务 worker 时再引入。
