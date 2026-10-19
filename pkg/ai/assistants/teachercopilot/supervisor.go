// supervisor.go 实现 TeacherSupervisorAgent。它负责理解教师问题并生成计划，
// 具体执行和依赖调度仍交给 multiagent.Orchestrator。
package teachercopilot

import (
	"context"
	"fmt"

	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/multiagent"
	ojtools "zk-code-arena-server/pkg/ai/tools/oj"
)

// TeacherSupervisorAgent 是教师侧主 Agent。它通过 SupervisorPlanner 生成
// LeadDecision，避免把意图识别、槽位抽取和任务拆解写死在 Go 规则里。
type TeacherSupervisorAgent struct {
	planner SupervisorPlanner
	tools   []ojtools.ToolSpec
}

// NewTeacherSupervisorAgent 创建教师侧主 Agent。
func NewTeacherSupervisorAgent(planner SupervisorPlanner, tools ...ojtools.ToolSpec) *TeacherSupervisorAgent {
	return &TeacherSupervisorAgent{planner: planner, tools: tools}
}

// NewLLMTeacherSupervisorAgent 创建使用 LLM 结构化规划的教师侧主 Agent。
func NewLLMTeacherSupervisorAgent(model llm.ModelFactory, tools ...ojtools.ToolSpec) *TeacherSupervisorAgent {
	return NewTeacherSupervisorAgent(NewLLMTeacherSupervisorPlanner(model), tools...)
}

// Plan 实现 multiagent.LeadPlanner。主 Agent 只负责产出结构化决策，
// 不直接执行工具、不直接调度子 Agent。
func (a *TeacherSupervisorAgent) Plan(ctx context.Context, req multiagent.RunRequest) (*multiagent.LeadDecision, error) {
	if a == nil || a.planner == nil {
		return nil, fmt.Errorf("teacher supervisor planner is not configured")
	}
	plan, err := a.planner.Plan(ctx, SupervisorPlanInput{
		Query:   req.Query,
		Context: req.Context,
		Roles:   req.Roles,
		Tools:   a.tools,
	})
	if err != nil {
		return nil, err
	}
	return plan.ToLeadDecision()
}
