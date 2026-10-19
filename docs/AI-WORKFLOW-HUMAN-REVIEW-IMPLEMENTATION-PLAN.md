# 项目任务分解规划

## 已明确的决策

- **错误类型**：已实现 `HumanReviewRequiredError`（pkg/ai/services/workflow/review_error.go）
- **Graph 节点**：已增加 `human_review` 节点，位于 `verify` 和 `formatter` 之间
- **数据模型**：会话模型已增加 `pending_review` 状态和 `DraftResult` 字段
- **权限控制**：使用现有的 `RequireTeacherOrAdmin()` 中间件
- **SSE 事件**：需要新增 `review_required`、`review_approved`、`review_rejected` 事件类型

## 整体规划概述

### 项目目标

完成 AI 工作流人工审核功能的开发，使工作流能够在 `human_review` 节点暂停，等待人工审核决策（批准/拒绝），并根据决策继续执行或重新执行。

### 技术栈

- **后端框架**：Go + Gin
- **数据库**：MongoDB
- **工作流引擎**：Eino Graph
- **实时通信**：Server-Sent Events (SSE)

### 主要阶段

1. **阶段 1：Service 层改造**（识别人工审核错误并转换状态）
2. **阶段 2：Repository 层改造**（完成态清理草稿字段）
3. **阶段 3：审核服务方法实现**（批准/拒绝逻辑）
4. **阶段 4：审核 API 实现**（路由和处理器）

### 详细任务分解

#### 阶段 1：Service 层改造

**任务 1.1**：在 `executeWorkflowUniversal` 中识别 `HumanReviewRequiredError`

- **目标**：当工作流执行返回 `HumanReviewRequiredError` 时，不将其视为失败，而是转为 `pending_review` 状态
- **输入**：工作流执行错误 `err`
- **输出**：会话状态更新为 `pending_review`，`draft_result` 字段保存草稿结果
- **涉及文件**：
  - `pkg/app/api-server/service/service_ai_workflow.go`（修改 `executeWorkflowUniversal` 方法）
- **实现要点**：
  1. 在 `executeWorkflowUniversal` 方法的错误处理部分（第 124-136 行）增加类型判断
  2. 使用 `errors.As(err, &workflow.HumanReviewRequiredError{})` 识别人工审核错误
  3. 提取 `draft_result` 并保存到会话的 `DraftResult` 字段
  4. 更新会话状态为 `pending_review`
  5. 发布 `review_required` SSE 事件
  6. 不执行原有的 `UpdateSessionError` 逻辑
- **预估工作量**：30 分钟

**任务 1.2**：新增 Repository 方法 `UpdateSessionPendingReview`

- **目标**：提供专门的方法更新会话为待审核状态并保存草稿结果
- **输入**：`sessionID`、`userID`、`draftResult`
- **输出**：会话状态更新为 `pending_review`，`draft_result` 字段有值
- **涉及文件**：
  - `pkg/app/api-server/repository/ai_workflow_repository.go`（新增方法）
- **实现要点**：
  1. 新增 `UpdateSessionPendingReview` 方法
  2. 使用 `$set` 更新 `status`、`draft_result`、`updated_at`
  3. 保持用户隔离（`user_id` 过滤）
- **预估工作量**：15 分钟

---

#### 阶段 2：Repository 层改造

**任务 2.1**：修改 `UpdateSessionResult` 方法清理草稿字段

- **目标**：在工作流完成时，清理 `draft_result` 和 `error_msg` 字段
- **输入**：`sessionID`、`userID`、`result`
- **输出**：会话状态为 `completed`，`result` 有值，`draft_result` 和 `error_msg` 被清空
- **涉及文件**：
  - `pkg/app/api-server/repository/ai_workflow_repository.go`（修改 `UpdateSessionResult` 方法，第 129-157 行）
- **实现要点**：
  1. 在 `$set` 中保持现有字段
  2. 新增 `$unset` 操作，清理 `draft_result` 和 `error_msg`
  3. 使用 MongoDB 的 `bson.M` 组合 `$set` 和 `$unset`
- **预估工作量**：15 分钟

---

#### 阶段 3：审核服务方法实现

**任务 3.1**：实现 `ReviewWorkflow` 服务方法

- **目标**：处理人工审核决策（批准/拒绝）
- **输入**：`ctx`、`sessionID`、`userID`、`decision`（"approve" | "reject"）、`comment`（可选）
- **输出**：
  - **批准**：会话状态变为 `completed`，`draft_result` 转为 `result`，发布 `review_approved` 和 `completed` 事件
  - **拒绝**：清理 `draft_result`，状态回到可执行状态，触发重新执行，发布 `review_rejected` 事件
- **涉及文件**：
  - `pkg/app/api-server/service/service_ai_workflow.go`（新增 `ReviewWorkflow` 方法）
- **实现要点**：
  1. 验证会话存在且状态为 `pending_review`
  2. **批准分支**：
     - 将 `session.DraftResult` 赋值给 `session.Result`
     - 调用 `UpdateSessionResult` 更新会话（会自动清理 `draft_result`）
     - 发布 `review_approved` 事件
     - 发布 `completed` 事件
  3. **拒绝分支**：
     - 清理 `session.DraftResult` 和 `session.ErrorMsg`
     - 更新会话状态为 `pending`
     - 发布 `review_rejected` 事件
     - 异步调用 `executeWorkflowUniversal` 重新执行工作流
  4. 记录审核日志（可选，添加到 `Logs` 字段）
- **预估工作量**：45 分钟

---

#### 阶段 4：审核 API 实现

**任务 4.1**：新增审核 API 路由

- **目标**：提供 HTTP 接口供前端调用审核功能
- **输入**：`POST /ai/workflow/:session_id/review`，请求体 `{ decision: "approve" | "reject", comment?: string }`
- **输出**：审核成功响应
- **涉及文件**：
  - `pkg/app/api-server/server/server_ai_workflow.go`（修改 `RegisterAIWorkflow` 方法，新增 `ReviewWorkflow` 处理器）
- **实现要点**：
  1. 在 `RegisterAIWorkflow` 方法中新增路由：
     ```go
     workflowGroup.POST("/:session_id/review", s.ReviewWorkflow)
     ```
  2. 定义请求结构体 `ReviewWorkflowRequest`：
     ```go
     type ReviewWorkflowRequest struct {
         Decision string `json:"decision" binding:"required,oneof=approve reject"`
         Comment  string `json:"comment"`
     }
     ```
  3. 实现 `ReviewWorkflow` 处理器方法：
     - 解析请求参数（`session_id`、`decision`、`comment`）
     - 获取用户 ID
     - 调用 `s.svc.AIWorkflowService.ReviewWorkflow`
     - 返回成功响应
  4. 添加 Swagger 注释（参考现有接口风格）
- **预估工作量**：30 分钟

---

## 需要进一步明确的问题

### 问题 1：拒绝后是否需要保留历史审核记录？

**推荐方案**：

- **方案 A**：在会话的 `Logs` 字段中记录审核操作（批准/拒绝、时间、审核人、评论）
  - **优点**：可追溯审核历史，便于审计
  - **缺点**：需要扩展 `WorkflowLog` 结构或新增审核日志字段

- **方案 B**：不保留审核记录，仅通过 SSE 事件通知前端
  - **优点**：实现简单，数据结构无需变更
  - **缺点**：无法追溯历史审核操作

**等待用户选择**：

```
请选择您偏好的方案，或提供其他建议：
[ ] 方案 A（推荐）
[ ] 方案 B
[ ] 其他方案：_______________
```

---

### 问题 2：拒绝后重新执行是否需要限制次数？

**推荐方案**：

- **方案 A**：复用现有的 `retry_count` 字段，限制总重试次数（包括审核拒绝）
  - **优点**：防止无限循环，保护系统资源
  - **缺点**：需要在拒绝时增加 `retry_count`

- **方案 B**：不限制审核拒绝次数，仅限制自动重试次数
  - **优点**：给予人工审核更大灵活性
  - **缺点**：可能导致资源浪费

**等待用户选择**：

```
请选择您偏好的方案，或提供其他建议：
[ ] 方案 A（推荐）
[ ] 方案 B
[ ] 其他方案：_______________
```

---

### 问题 3：审核权限是否需要更细粒度控制？

**推荐方案**：

- **方案 A**：仅允许会话创建者本人审核
  - **优点**：权限更严格，避免误操作
  - **缺点**：团队协作场景不友好

- **方案 B**：允许所有老师/管理员审核任意会话
  - **优点**：便于团队协作和管理
  - **缺点**：可能存在越权风险

- **方案 C**：允许会话创建者 + 管理员审核
  - **优点**：平衡安全性和灵活性
  - **缺点**：实现稍复杂

**等待用户选择**：

```
请选择您偏好的方案，或提供其他建议：
[ ] 方案 A
[ ] 方案 B（推荐）
[ ] 方案 C
[ ] 其他方案：_______________
```

---

## 用户反馈区域

请在此区域补充您对整体规划的意见和建议：

```
用户补充内容：

---
---
---
```

---

## 实施顺序（按风险从低到高）

1. **阶段 2：Repository 层改造**（风险最低，独立功能）
   - 任务 2.1：修改 `UpdateSessionResult` 清理草稿字段
   - 任务 1.2：新增 `UpdateSessionPendingReview` 方法

2. **阶段 1：Service 层改造**（中等风险，核心逻辑）
   - 任务 1.1：识别 `HumanReviewRequiredError` 并转换状态

3. **阶段 3：审核服务方法实现**（中等风险，业务逻辑）
   - 任务 3.1：实现 `ReviewWorkflow` 服务方法

4. **阶段 4：审核 API 实现**（风险最低，接口层）
   - 任务 4.1：新增审核 API 路由

---

## 潜在风险和注意事项

### 风险 1：并发审核冲突

- **描述**：多个用户同时审核同一会话可能导致状态不一致
- **缓解措施**：
  1. 在 `ReviewWorkflow` 方法中增加状态检查（必须为 `pending_review`）
  2. 使用 MongoDB 的原子更新操作（`UpdateOne` 的 `MatchedCount` 检查）
  3. 考虑增加乐观锁（版本号字段）

### 风险 2：SSE 事件丢失

- **描述**：客户端断开连接后可能错过审核相关事件
- **缓解措施**：
  1. 前端定期轮询会话状态作为兜底
  2. 在 SSE 重连后发送当前状态快照
  3. 考虑增加事件持久化（可选）

### 风险 3：重新执行失败处理

- **描述**：拒绝后重新执行可能再次失败或再次进入待审核状态
- **缓解措施**：
  1. 限制重试次数（见问题 2）
  2. 记录每次执行的日志，便于排查问题
  3. 提供手动终止工作流的接口

### 风险 4：草稿结果数据一致性

- **描述**：`draft_result` 和 `result` 的数据结构必须一致
- **缓解措施**：
  1. 使用相同的 `OJProblemWorkflowResult` 类型
  2. 在批准时直接赋值，无需转换
  3. 增加单元测试验证数据结构

### 风险 5：文件编码问题

- **描述**：交接文档提到 `server_ai_workflow.go` 存在历史编码异常风险
- **缓解措施**：
  1. 修改前先编译验证
  2. 使用 UTF-8 编码保存文件
  3. 提交前进行代码审查

---

## 验收清单

- [ ] 工作流在 `human_review` 节点进入 `pending_review` 状态（不是 `failed`）
- [ ] 查询会话可看到 `draft_result` 字段有值
- [ ] SSE 事件包含 `review_required` 事件
- [ ] 调用 `POST /ai/workflow/:session_id/review` 接口（decision=approve）后：
  - [ ] 会话状态变为 `completed`
  - [ ] `result` 字段有值
  - [ ] `draft_result` 字段被清空
  - [ ] `error_msg` 字段被清空
  - [ ] SSE 事件包含 `review_approved` 和 `completed` 事件
- [ ] 调用 `POST /ai/workflow/:session_id/review` 接口（decision=reject）后：
  - [ ] 会话状态变为 `pending` 或 `running`
  - [ ] `draft_result` 字段被清空
  - [ ] SSE 事件包含 `review_rejected` 事件
  - [ ] 工作流重新开始执行
- [ ] 权限验证：非老师/管理员无法调用审核接口
- [ ] 状态验证：非 `pending_review` 状态的会话无法审核
- [ ] 用户隔离：用户只能审核自己的会话（或根据问题 3 的决策调整）
- [ ] 编译通过，无语法错误
- [ ] 单元测试通过（如有）

---

## 预估总工作量

- **开发时间**：2-3 小时
- **测试时间**：1-2 小时
- **联调时间**：1 小时
- **总计**：4-6 小时

---

## 后续优化建议

1. **增加审核历史记录**：独立的审核日志表，记录所有审核操作
2. **支持审核评论**：在审核时添加评论，便于沟通
3. **审核通知**：通过邮件/站内信通知相关人员
4. **批量审核**：支持一次审核多个会话
5. **审核统计**：统计审核通过率、平均审核时间等指标
6. **审核权限细化**：支持按课程、班级等维度控制审核权限
