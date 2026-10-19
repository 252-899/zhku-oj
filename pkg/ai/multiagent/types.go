// 包 multiagent 提供可复用的 LeadAgent 动作协议和编排基础能力，
// 供教师侧、学生侧以及后续 AI 工作流复用。
package multiagent

import "context"

const (
	// ActionInterimReply 表示 Lead 在子 Agent 启动前给用户发送阶段性说明。
	ActionInterimReply = "interim_reply"
	// ActionRevisePlan 表示创建或更新 Orchestrator 要执行的任务图。
	ActionRevisePlan = "revise_plan"
	// ActionSpawnAgent 表示直接创建一个可执行的子 Agent 任务。
	ActionSpawnAgent = "spawn_agent"
	// ActionDone 和 ActionReply 都表示 Lead 的最终回复内容。
	ActionDone  = "done"
	ActionReply = "reply"
)

// RoleSpec 描述 Lead 规划时可以选择的一个角色。
// 角色真正能做什么由业务侧注册的 AgentFactory 决定。
type RoleSpec struct {
	Name         string `json:"name"`
	BaseRole     string `json:"base_role,omitempty"`
	Description  string `json:"description,omitempty"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

// TaskSpec 是 Lead 产出的静态任务描述。DependsOn 会形成小型 DAG，
// 使 Orchestrator 可以并行执行独立任务，并等待前置任务完成。
type TaskSpec struct {
	ID          string   `json:"id"`
	Description string   `json:"description"`
	Role        string   `json:"role,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	ModelTier   string   `json:"model_tier,omitempty"`
}

// LeadAction 是 LeadPlanner 输出给 Orchestrator 的结构化指令。
// 它不是主 Agent 本身，而是告诉编排器要说什么、创建什么任务、由哪个角色执行。
type LeadAction struct {
	Type            string         `json:"type"`
	TaskID          string         `json:"task_id,omitempty"`
	AgentID         string         `json:"agent_id,omitempty"`
	Role            string         `json:"role,omitempty"`
	TaskDescription string         `json:"task_description,omitempty"`
	DependsOn       []string       `json:"depends_on,omitempty"`
	Content         string         `json:"content,omitempty"`
	ModelTier       string         `json:"model_tier,omitempty"`
	Create          []TaskSpec     `json:"create,omitempty"`
	Cancel          []string       `json:"cancel,omitempty"`
	Update          []TaskSpec     `json:"update,omitempty"`
	Tool            string         `json:"tool,omitempty"`
	ToolParams      map[string]any `json:"tool_params,omitempty"`
}

// LeadDecision 是 Lead 的一次规划结果。当前 MVP 先执行单次规划，
// 再执行规划得到的 actions。
type LeadDecision struct {
	Summary string       `json:"decision_summary,omitempty"`
	Actions []LeadAction `json:"actions"`
}

// RunRequest 是领域无关的 multi-agent 运行输入。业务层应把登录用户、
// 班级、作业、权限等上下文放入 Context，而不是在这里添加教师侧专属字段。
type RunRequest struct {
	SessionID string         `json:"session_id,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	Domain    string         `json:"domain,omitempty"`
	Query     string         `json:"query"`
	Context   map[string]any `json:"context,omitempty"`
	Roles     []RoleSpec     `json:"roles,omitempty"`
}

// AgentTask 是传给子 AgentRunner 的可执行任务。只有当 Lead 计划已校验、
// 且依赖满足后，Orchestrator 才会创建该任务。
type AgentTask struct {
	ID            string         `json:"id"`
	AgentID       string         `json:"agent_id,omitempty"`
	Role          string         `json:"role"`
	Description   string         `json:"description"`
	ModelTier     string         `json:"model_tier,omitempty"`
	DependsOn     []string       `json:"depends_on,omitempty"`
	OriginalQuery string         `json:"original_query,omitempty"`
	Context       map[string]any `json:"context,omitempty"`
}

// AgentResult 是单个子 Agent 的标准化输出。若 runner 未填写 TaskID 或 Role，
// Orchestrator 会补齐这些字段，方便下游稳定读取。
type AgentResult struct {
	AgentID string         `json:"agent_id"`
	TaskID  string         `json:"task_id"`
	Role    string         `json:"role"`
	Content string         `json:"content,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
	Success bool           `json:"success"`
	Error   string         `json:"error,omitempty"`
}

// RunResult 是一次编排运行的最终输出。后续可以加入报告生成或质量门控步骤，
// 将 Results 转换成更强的最终回答后再返回 handler。
type RunResult struct {
	Summary      string                 `json:"summary,omitempty"`
	InterimReply string                 `json:"interim_reply,omitempty"`
	FinalReply   string                 `json:"final_reply,omitempty"`
	Tasks        []TaskSpec             `json:"tasks,omitempty"`
	Results      []AgentResult          `json:"results,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LeadPlanner 是 LeadAgent 的抽象接口。第一版可以用规则实现，
// 协议稳定后可替换为 LLM 结构化输出。
type LeadPlanner interface {
	Plan(ctx context.Context, req RunRequest) (*LeadDecision, error)
}

// AgentRunner 执行一个子 Agent 任务。业务包可以用 Go service、
// LLM 工具调用或远程 Agent 服务来实现。
type AgentRunner interface {
	Run(ctx context.Context, task AgentTask) (*AgentResult, error)
}

// AgentFactory 为已注册角色创建 runner。工厂会收到角色元数据，
// 以便应用角色专属提示词、工具或模型配置。
type AgentFactory func(ctx context.Context, role RoleSpec) (AgentRunner, error)
