// service_teacher_copilot_test.go 验证教师侧 Copilot 业务串联服务的基础链路。
package service

import (
	"context"
	"strings"
	"testing"

	"zk-code-arena-server/pkg/ai/infra/llm"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"

	aiapi "zk-code-arena-server/pkg/ai/api"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestTeacherCopilotServiceStream(t *testing.T) {
	sessionService := runtimesession.NewMemorySessionService()
	taskManager := runtimetask.NewMemoryTaskManager()
	contextManager := runtimecontext.NewManager(sessionService, 10)
	model := fakeTeacherCopilotModel{content: `{
		"intent":"assignment_review",
		"decision_summary":"作业分析",
		"interim_reply":"我会先确认班级和作业范围。",
		"need_more_info":false,
		"tasks":[
			{"id":"T1","role":"class_analyst","description":"解析班级和作业范围"},
			{"id":"T2","role":"assignment_reviewer","description":"分析作业完成情况","depends_on":["T1"]},
			{"id":"T3","role":"report_writer","description":"生成讲评建议","depends_on":["T2"]}
		]
	}`}

	svc := NewTeacherCopilotService(
		&UserService{},
		&ClazzService{},
		&ProblemService{},
		sessionService,
		taskManager,
		contextManager,
		model,
	)

	stream, err := svc.Stream(context.Background(), &aiapi.RuntimeRequest{
		UserID:  "teacher-1",
		Message: "分析一下 23 软件 1 班这次数组作业",
		Input: map[string]any{
			"class_name": "23 软件 1 班",
			"task_name":  "数组作业",
		},
	})
	if err != nil {
		t.Fatalf("Stream returned error: %v", err)
	}

	var text strings.Builder
	var sawDone bool
	for event := range stream {
		if event == nil {
			continue
		}
		if event.Type == aiapi.EventText {
			if data, ok := event.Data.(map[string]any); ok {
				if delta, ok := data["delta"].(string); ok {
					text.WriteString(delta)
				}
			}
		}
		if event.Type == aiapi.EventDone {
			sawDone = true
		}
	}

	if !sawDone {
		t.Fatalf("expected done event")
	}
	if !strings.Contains(text.String(), "class_id") {
		t.Fatalf("expected class analyst result in text, got %q", text.String())
	}
}

type fakeTeacherCopilotModel struct {
	content string
}

func (f fakeTeacherCopilotModel) IsChatConfigured() bool {
	return true
}

func (f fakeTeacherCopilotModel) ChatConfig() llm.ChatConfig {
	return llm.ChatConfig{BaseURL: "http://fake", APIKey: "fake", Model: "fake"}
}

func (f fakeTeacherCopilotModel) NewToolCallingChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	return nil, nil
}

func (f fakeTeacherCopilotModel) Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	return schema.StreamReaderFromArray([]*schema.Message{{Role: schema.Assistant, Content: "class_id=mock-class-id, class_name=23 软件 1 班, task_id=mock-task-id, task_name=数组作业"}}), nil
}

func (f fakeTeacherCopilotModel) Call(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return schema.AssistantMessage(f.content, nil), nil
}
