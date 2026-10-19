# AI 工作流开发进度交接（OJ 题目生成）

## 1. 模块边界（已确认）
- `pkg/ai/services/workflow`：**具体工作流实现层**（节点、graph、状态）。
- `pkg/ai/workflow`：**抽象框架层**（定义、注册、执行引擎）。

> 后续开发请继续遵守该分层，避免把业务节点逻辑写入框架层。

---

## 2. 当前总体进度

### 2.1 已串联主流程（具体实现层）
当前 OJ 工作流主链路已串联：
1. `problem_gen`
2. `testcase_gen`
3. `solution_gen`
4. `run_testcases`
5. `verify`
6. `human_review`
7. `formatter`

对应文件：
- `pkg/ai/services/workflow/graph.go`

### 2.2 已完成的关键改动（本轮）
1. **新增人工审核中断错误类型**
   - 文件：`pkg/ai/services/workflow/review_error.go`
   - 类型：`HumanReviewRequiredError`
   - 用途：在 `human_review` 节点主动中断流程并携带草稿结果。

2. **`graph` 增加 `human_review` 节点并接入边**
   - 文件：`pkg/ai/services/workflow/graph.go`
   - 关键点：
     - `verify -> human_review -> formatter`
     - `human_review` 节点会先构建草稿结果，然后返回 `HumanReviewRequiredError`，将流程暂停。
     - 新增 `buildDraftResult(...)` 复用格式化数据构建逻辑。

3. **会话模型增加待审核状态与草稿字段**
   - 文件：`pkg/models/ai_workflow.go`
   - 新增：
     - `WorkflowStatusPendingReview = "pending_review"`
     - `AIWorkflowSession.DraftResult`

---

## 3. 当前仍未完成（发版前建议补齐）

### 3.1 Service 层尚未接入“待审核”分支
文件：`pkg/app/api-server/service/service_ai_workflow.go`

现状：
- `executeWorkflowUniversal` 里，工作流报错仍统一按失败处理（`UpdateSessionError` + SSE `error`）。
- 还未识别 `HumanReviewRequiredError` 并转为 `pending_review` 状态。

需要实现：
- `errors.As(err, *workflow.HumanReviewRequiredError)` 分支。
- 将会话状态改为 `pending_review`，保存 `draft_result`，发布 `review_required` 事件。
- 不应将该场景记为 `failed`。

### 3.2 人工审核 API 尚未加路由与处理
文件：
- `pkg/app/api-server/server/server_ai_workflow.go`
- `pkg/app/api-server/service/service_ai_workflow.go`

需要实现：
- 路由：`POST /ai/workflow/:session_id/review`
- 请求：`{ decision: "approve" | "reject", comment?: string }`
- 服务方法：`ReviewWorkflow(...)`
  - `approve`：将 `draft_result` 落为 `result`，状态改 `completed`。
  - `reject`：清理 `draft_result`，回到可执行状态并触发重跑。

### 3.3 Repository 完成态未清草稿
文件：`pkg/app/api-server/repository/ai_workflow_repository.go`

需要实现：
- `UpdateSessionResult` 中增加 `$unset: { draft_result: "" }`。
- 建议同时清理 `error_msg`。

---

## 4. verify 节点现状（与快速上线策略一致）
文件：`pkg/ai/services/workflow/nodes/verify.go`

现状：
- `Verify` 主流程可走通。
- `runCode` 仍是 mock（TODO，返回模拟结果），并非真实判题回查。

结论：
- 现阶段可按“**verify 保持 mock，优先人工审核**”上线。
- 后续可替换为：
  - 直接走 sandbox 执行校验；或
  - 调用已有判题服务接口（推荐与现有判题链路统一）。

---

## 5. 后续开发建议顺序（低风险）
1. 在 `service_ai_workflow.go` 先加 `pending_review` 分支（不改 API）。
2. 补 `ReviewWorkflow` 服务方法。
3. 补 `server_ai_workflow.go` 审核接口。
4. 补 repository 的 `$unset draft_result`。
5. 跑定向测试/编译后再联调前端。

---

## 6. 验收清单
- 启动工作流后，在 `human_review` 节点应进入 `pending_review`，而不是 `failed`。
- 查询会话可看到 `draft_result`。
- 调用 `review approve` 后：
  - 会话为 `completed`
  - `result` 有值，`draft_result` 被清空。
- 调用 `review reject` 后：
  - 会话重新进入执行流程。
- SSE 事件至少包含：`review_required` / `review_approved` / `review_rejected` / `completed`。

---

## 7. 风险提示
- 当前仓库有大量并行改动（AI 模块重构中），建议后续提交时按“最小功能闭环”拆分提交。
- `server_ai_workflow.go` 文件末尾存在历史文本编码/字符串异常风险，补接口时请先做一次编译校验。
