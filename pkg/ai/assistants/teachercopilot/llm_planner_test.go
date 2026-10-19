package teachercopilot

import (
	"context"
	"strings"
	"testing"

	"zk-code-arena-server/pkg/ai/infra/llm"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestLLMTeacherSupervisorPlannerParsesStructuredPlan(t *testing.T) {
	model := fakePlanModel{content: "```json\n" + `{
		"intent": "assignment_review",
		"decision_summary": "assignment_review",
		"interim_reply": "开始分析",
		"need_more_info": false,
		"slots": {"class_name": "23 软件 1 班", "task_name": "数组作业"},
		"tasks": [
			{"id": " T1 ", "role": " class_analyst ", "description": " 解析范围 "},
			{"id": "T2", "role": "assignment_reviewer", "description": "分析作业", "depends_on": ["T1"]}
		]
	}` + "\n```"}

	plan, err := NewLLMTeacherSupervisorPlanner(model).Plan(context.Background(), SupervisorPlanInput{
		Query: "分析 23 软件 1 班数组作业",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if plan.Intent != IntentAssignmentReview {
		t.Fatalf("unexpected intent: %s", plan.Intent)
	}
	if len(plan.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %#v", plan.Tasks)
	}
	if plan.Tasks[0].ID != "T1" || plan.Tasks[0].Role != RoleClassAnalyst {
		t.Fatalf("expected normalized first task, got %#v", plan.Tasks[0])
	}
}

func TestLLMTeacherSupervisorPlannerReportsInvalidJSON(t *testing.T) {
	_, err := NewLLMTeacherSupervisorPlanner(fakePlanModel{content: "not-json"}).Plan(context.Background(), SupervisorPlanInput{Query: "分析"})
	if err == nil || !strings.Contains(err.Error(), "parse teacher supervisor plan failed") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

type fakePlanModel struct {
	content string
}

func (f fakePlanModel) IsChatConfigured() bool {
	return true
}

func (f fakePlanModel) ChatConfig() llm.ChatConfig {
	return llm.ChatConfig{}
}

func (f fakePlanModel) NewToolCallingChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return nil, nil
}

func (f fakePlanModel) Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{}), nil
}

func (f fakePlanModel) Call(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return schema.AssistantMessage(f.content, nil), nil
}
