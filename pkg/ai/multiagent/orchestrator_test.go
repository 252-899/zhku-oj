// orchestrator_test.go 记录 MVP 编排器的预期行为：依赖感知执行、
// spawn_agent 简写支持以及容量保护。
package multiagent

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type staticLead struct {
	decision *LeadDecision
	err      error
}

// staticLead 让测试可以直接描述 Lead 输出，而不需要引入 LLM。
func (s staticLead) Plan(ctx context.Context, req RunRequest) (*LeadDecision, error) {
	return s.decision, s.err
}

type recordingAgent struct {
	role  string
	calls *[]string
}

// recordingAgent 记录执行顺序，便于验证依赖调度行为。
func (a recordingAgent) Run(ctx context.Context, task AgentTask) (*AgentResult, error) {
	*a.calls = append(*a.calls, task.ID)
	return &AgentResult{
		AgentID: task.AgentID,
		TaskID:  task.ID,
		Role:    a.role,
		Content: fmt.Sprintf("%s done", task.ID),
		Success: true,
	}, nil
}

func TestOrchestratorRunsLeadPlannedTasksWithDependencies(t *testing.T) {
	calls := []string{}
	registry := NewRoleRegistry()
	for _, role := range []string{"class_analyst", "assignment_reviewer", "report_writer"} {
		role := role
		if err := registry.Register(RoleSpec{Name: role}, func(ctx context.Context, spec RoleSpec) (AgentRunner, error) {
			return recordingAgent{role: role, calls: &calls}, nil
		}); err != nil {
			t.Fatalf("register role: %v", err)
		}
	}

	lead := staticLead{decision: &LeadDecision{
		Summary: "teacher assignment analysis",
		Actions: []LeadAction{
			{Type: ActionInterimReply, Content: "分析作业情况"},
			{Type: ActionRevisePlan, Create: []TaskSpec{
				{ID: "T1", Role: "class_analyst", Description: "确认班级和作业"},
				{ID: "T2", Role: "assignment_reviewer", Description: "分析作业完成情况", DependsOn: []string{"T1"}},
				{ID: "T3", Role: "report_writer", Description: "生成讲评建议", DependsOn: []string{"T2"}},
			}},
		},
	}}

	result, err := NewOrchestrator(lead, registry, OrchestratorConfig{MaxAgents: 4}).Run(context.Background(), RunRequest{
		Query: "分析一下 23 软件 1 班这次数组作业，并给我讲评建议",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Summary != "teacher assignment analysis" {
		t.Fatalf("unexpected summary %q", result.Summary)
	}
	if len(result.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result.Results))
	}
	if got := strings.Join(calls, ","); got != "T1,T2,T3" {
		t.Fatalf("expected dependency order T1,T2,T3, got %s", got)
	}
}

func TestOrchestratorSupportsSpawnAgentAction(t *testing.T) {
	calls := []string{}
	registry := NewRoleRegistry()
	if err := registry.Register(RoleSpec{Name: "class_analyst"}, func(ctx context.Context, spec RoleSpec) (AgentRunner, error) {
		return recordingAgent{role: spec.Name, calls: &calls}, nil
	}); err != nil {
		t.Fatalf("register role: %v", err)
	}

	lead := staticLead{decision: &LeadDecision{Actions: []LeadAction{
		{
			Type:            ActionSpawnAgent,
			TaskID:          "T1",
			Role:            "class_analyst",
			TaskDescription: "分析班级概况",
		},
	}}}

	result, err := NewOrchestrator(lead, registry, OrchestratorConfig{MaxAgents: 1}).Run(context.Background(), RunRequest{Query: "班级情况怎么样"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Results) != 1 || result.Results[0].TaskID != "T1" {
		t.Fatalf("unexpected results: %#v", result.Results)
	}
}

func TestOrchestratorRejectsOverCapacityPlan(t *testing.T) {
	calls := []string{}
	registry := NewRoleRegistry()
	if err := registry.Register(RoleSpec{Name: "analyst"}, func(ctx context.Context, spec RoleSpec) (AgentRunner, error) {
		return recordingAgent{role: spec.Name, calls: &calls}, nil
	}); err != nil {
		t.Fatalf("register role: %v", err)
	}

	lead := staticLead{decision: &LeadDecision{Actions: []LeadAction{
		{Type: ActionRevisePlan, Create: []TaskSpec{
			{ID: "T1", Role: "analyst", Description: "task 1"},
			{ID: "T2", Role: "analyst", Description: "task 2"},
		}},
	}}}

	_, err := NewOrchestrator(lead, registry, OrchestratorConfig{MaxAgents: 1}).Run(context.Background(), RunRequest{Query: "分析"})
	if err == nil {
		t.Fatalf("expected capacity error")
	}
	if !strings.Contains(err.Error(), "max is 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}
