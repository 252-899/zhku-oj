package teachercopilot

import (
	"context"
	"testing"

	"zk-code-arena-server/pkg/ai/multiagent"
)

func TestTeacherSupervisorUsesPlannerDecision(t *testing.T) {
	planner := fakeSupervisorPlanner{plan: &SupervisorPlan{
		Intent:          IntentAssignmentReview,
		DecisionSummary: string(IntentAssignmentReview),
		InterimReply:    "我会先确认班级和作业范围，再分析完成情况。",
		Slots: map[string]string{
			ContextClassName: "23 软件 1 班",
			ContextTaskName:  "数组作业",
		},
		Tasks: []multiagent.TaskSpec{
			{
				ID:          "T1",
				Role:        RoleClassAnalyst,
				Description: "根据教师问题解析班级与作业任务范围",
			},
			{
				ID:          "T2",
				Role:        RoleAssignmentReviewer,
				Description: "分析作业完成率、未完成学生和异常点",
				DependsOn:   []string{"T1"},
			},
			{
				ID:          "T3",
				Role:        RoleReportWriter,
				Description: "生成课堂讲评建议",
				DependsOn:   []string{"T2"},
			},
		},
	}}

	decision, err := NewTeacherSupervisorAgent(planner).Plan(context.Background(), multiagent.RunRequest{
		Query: "分析一下 23 软件 1 班这次数组作业，并给我讲评建议",
		Context: map[string]any{
			ContextClassName: "23 软件 1 班",
			ContextTaskName:  "数组作业",
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if decision.Summary != string(IntentAssignmentReview) {
		t.Fatalf("unexpected summary: %s", decision.Summary)
	}
	if len(decision.Actions) != 2 {
		t.Fatalf("expected interim + revise_plan actions, got %#v", decision.Actions)
	}
	tasks := decision.Actions[1].Create
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %#v", tasks)
	}
	if tasks[0].Role != RoleClassAnalyst {
		t.Fatalf("expected class analyst first, got %#v", tasks[0])
	}
	if tasks[1].DependsOn[0] != "T1" || tasks[2].DependsOn[0] != "T2" {
		t.Fatalf("unexpected dependencies: %#v", tasks)
	}
}

func TestTeacherSupervisorCanAskForMoreInfo(t *testing.T) {
	planner := fakeSupervisorPlanner{plan: &SupervisorPlan{
		Intent:          IntentAssignmentReview,
		DecisionSummary: "missing_learning_target",
		NeedMoreInfo:    true,
		MissingFields:   []string{"class_or_task"},
		FinalReply:      "请告诉我要分析哪个班级或哪次作业。",
	}}

	decision, err := NewTeacherSupervisorAgent(planner).Plan(context.Background(), multiagent.RunRequest{
		Query: "帮我分析一下",
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(decision.Actions) != 1 || decision.Actions[0].Type != multiagent.ActionReply {
		t.Fatalf("expected direct reply action, got %#v", decision.Actions)
	}
}

func TestTeacherSupervisorRequiresPlanner(t *testing.T) {
	_, err := NewTeacherSupervisorAgent(nil).Plan(context.Background(), multiagent.RunRequest{Query: "分析"})
	if err == nil {
		t.Fatalf("expected missing planner error")
	}
}

type fakeSupervisorPlanner struct {
	plan *SupervisorPlan
	err  error
}

func (f fakeSupervisorPlanner) Plan(ctx context.Context, input SupervisorPlanInput) (*SupervisorPlan, error) {
	return f.plan, f.err
}
