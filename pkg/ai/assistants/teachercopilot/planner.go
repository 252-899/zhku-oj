// planner.go 定义教师侧 Supervisor 的结构化规划协议。
package teachercopilot

import (
	"context"
	"fmt"
	"strings"

	"zk-code-arena-server/pkg/ai/multiagent"
	ojtools "zk-code-arena-server/pkg/ai/tools/oj"
)

// SupervisorPlanner 负责把教师问题转换为结构化计划。
// 生产环境优先使用 LLM 实现，测试中可以使用 fake planner。
type SupervisorPlanner interface {
	Plan(ctx context.Context, input SupervisorPlanInput) (*SupervisorPlan, error)
}

// SupervisorPlanInput 是传给主 Agent 规划器的完整上下文。
type SupervisorPlanInput struct {
	Query   string
	Context map[string]any
	Roles   []multiagent.RoleSpec
	Tools   []ojtools.ToolSpec
}

// SupervisorPlan 是 TeacherSupervisorAgent 的结构化规划结果。
type SupervisorPlan struct {
	Intent          Intent                `json:"intent"`
	DecisionSummary string                `json:"decision_summary"`
	InterimReply    string                `json:"interim_reply"`
	FinalReply      string                `json:"final_reply"`
	NeedMoreInfo    bool                  `json:"need_more_info"`
	MissingFields   []string              `json:"missing_fields,omitempty"`
	Slots           map[string]string     `json:"slots,omitempty"`
	Tasks           []multiagent.TaskSpec `json:"tasks,omitempty"`
}

// ToLeadDecision 将 SupervisorPlan 转成通用 multiagent.LeadDecision。
func (p *SupervisorPlan) ToLeadDecision() (*multiagent.LeadDecision, error) {
	if p == nil {
		return nil, fmt.Errorf("supervisor plan is nil")
	}
	summary := strings.TrimSpace(p.DecisionSummary)
	if summary == "" {
		summary = string(p.Intent)
	}
	if summary == "" {
		summary = string(IntentUnsupported)
	}

	if p.NeedMoreInfo || len(p.Tasks) == 0 {
		content := strings.TrimSpace(p.FinalReply)
		if content == "" {
			content = unsupportedIntentReply
		}
		return &multiagent.LeadDecision{
			Summary: summary,
			Actions: []multiagent.LeadAction{
				{Type: multiagent.ActionReply, Content: content},
			},
		}, nil
	}

	interim := strings.TrimSpace(p.InterimReply)
	if interim == "" {
		interim = "我会先拆解问题并查询相关教学数据，再整理分析结论。"
	}
	return &multiagent.LeadDecision{
		Summary: summary,
		Actions: []multiagent.LeadAction{
			{Type: multiagent.ActionInterimReply, Content: interim},
			{Type: multiagent.ActionRevisePlan, Create: p.Tasks},
		},
	}, nil
}
