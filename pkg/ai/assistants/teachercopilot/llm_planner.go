// llm_planner.go 使用 LLM 结构化输出实现教师侧 Supervisor 规划。
package teachercopilot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"zk-code-arena-server/pkg/ai/infra/llm"

	"github.com/cloudwego/eino/schema"
)

// LLMTeacherSupervisorPlanner 通过模型理解教师问题、抽取线索并生成任务计划。
type LLMTeacherSupervisorPlanner struct {
	model llm.ModelFactory
}

// NewLLMTeacherSupervisorPlanner 创建 LLM 版本的教师侧 Supervisor 规划器。
func NewLLMTeacherSupervisorPlanner(model llm.ModelFactory) *LLMTeacherSupervisorPlanner {
	return &LLMTeacherSupervisorPlanner{model: model}
}

// Plan 调用模型并解析结构化 JSON 计划。
func (p *LLMTeacherSupervisorPlanner) Plan(ctx context.Context, input SupervisorPlanInput) (*SupervisorPlan, error) {
	if p == nil || p.model == nil {
		return nil, fmt.Errorf("teacher supervisor llm planner is not configured")
	}
	content, err := json.Marshal(map[string]any{
		"query":   input.Query,
		"context": input.Context,
		"roles":   input.Roles,
		"tools":   input.Tools,
	})
	if err != nil {
		return nil, err
	}

	msg, err := p.model.Call(ctx, []*schema.Message{
		schema.SystemMessage(TeacherSupervisorSystemPrompt),
		schema.UserMessage(string(content)),
	})
	if err != nil {
		return nil, err
	}
	if msg == nil || strings.TrimSpace(msg.Content) == "" {
		return nil, fmt.Errorf("teacher supervisor returned empty plan")
	}

	var plan SupervisorPlan
	if err := json.Unmarshal([]byte(extractJSONObject(msg.Content)), &plan); err != nil {
		return nil, fmt.Errorf("parse teacher supervisor plan failed: %w", err)
	}
	normalizeSupervisorPlan(&plan)
	return &plan, nil
}

// normalizeSupervisorPlan 对模型返回的计划做轻量规范化，降低后续调度的容错压力。
func normalizeSupervisorPlan(plan *SupervisorPlan) {
	if plan == nil {
		return
	}
	if plan.Intent == "" {
		plan.Intent = IntentUnsupported
	}
	for i := range plan.Tasks {
		plan.Tasks[i].ID = strings.TrimSpace(plan.Tasks[i].ID)
		plan.Tasks[i].Role = strings.TrimSpace(plan.Tasks[i].Role)
		plan.Tasks[i].Description = strings.TrimSpace(plan.Tasks[i].Description)
	}
}

// extractJSONObject 从模型输出中提取 JSON 对象，兼容模型偶发包裹 Markdown 的情况。
func extractJSONObject(content string) string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end >= start {
		return content[start : end+1]
	}
	return content
}
