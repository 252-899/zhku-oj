package react

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"

	"github.com/cloudwego/eino/schema"
)

func TestRunnerExecutesToolAndContinuesToFinalAnswer(t *testing.T) {
	model := &fakeModel{rounds: [][]*schema.Message{
		{
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
				ID: "call-1", Type: "function",
				Function: schema.FunctionCall{Name: "weather", Arguments: `{"city":"深圳"}`},
			}}},
		},
		{
			{Role: schema.Assistant, Content: "深圳小雨。"},
		},
	}}
	executor := fakeExecutor{results: map[string]string{"weather": `{"weather":"小雨"}`}}
	req := &aiapi.RuntimeRequest{SessionID: "s1", RoundID: "r1", TaskType: "test"}
	events := []*aiapi.RuntimeStreamEvent{}

	runner := NewRunner(model, executor, nil, Config{MaxRounds: 3})
	messages, err := runner.Run(context.Background(), req, []*schema.Message{schema.UserMessage("深圳天气")}, func(event *aiapi.RuntimeStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("expected user + assistant tool + tool + final assistant messages, got %d", len(messages))
	}
	if messages[2].Role != schema.Tool || messages[2].ToolName != "weather" {
		t.Fatalf("expected tool message, got %#v", messages[2])
	}
	if messages[3].Content != "深圳小雨。" {
		t.Fatalf("unexpected final message: %#v", messages[3])
	}
	if !hasEvent(events, aiapi.EventToolCall) || !hasEvent(events, aiapi.EventToolResult) || !hasEvent(events, aiapi.EventText) {
		t.Fatalf("expected tool_call, tool_result and text events, got %#v", events)
	}
}

func TestRunnerMergesStreamingToolCallArguments(t *testing.T) {
	index := 0
	model := &fakeModel{rounds: [][]*schema.Message{
		{
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
				Index: &index, ID: "call-1", Type: "function",
				Function: schema.FunctionCall{Name: "weather", Arguments: `{"city":`},
			}}},
			{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
				Index: &index, ID: "call-1", Type: "function",
				Function: schema.FunctionCall{Name: "weather", Arguments: `"深圳"}`},
			}}},
		},
		{{Role: schema.Assistant, Content: "done"}},
	}}
	executor := &recordingExecutor{result: `{"ok":true}`}

	_, err := NewRunner(model, executor, nil, Config{MaxRounds: 3}).Run(
		context.Background(),
		&aiapi.RuntimeRequest{SessionID: "s1", RoundID: "r1", TaskType: "test"},
		[]*schema.Message{schema.UserMessage("test")},
		nil,
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if executor.arguments != `{"city":"深圳"}` {
		t.Fatalf("expected merged arguments, got %q", executor.arguments)
	}
}

func TestRunnerForcesFinalAnswerWhenMaxRoundsReached(t *testing.T) {
	model := &fakeModel{rounds: [][]*schema.Message{
		{{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "call-1", Type: "function", Function: schema.FunctionCall{Name: "loop", Arguments: `{}`}}}}},
		{{Role: schema.Assistant, Content: "基于已有信息总结。"}},
	}}

	messages, err := NewRunner(model, fakeExecutor{results: map[string]string{"loop": `{}`}}, nil, Config{MaxRounds: 1}).Run(
		context.Background(),
		&aiapi.RuntimeRequest{SessionID: "s1", RoundID: "r1", TaskType: "test"},
		[]*schema.Message{schema.UserMessage("test")},
		nil,
	)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := messages[len(messages)-1].Content; !strings.Contains(got, "基于已有信息") {
		t.Fatalf("expected forced final answer, got %q", got)
	}
}

type fakeModel struct {
	rounds [][]*schema.Message
	calls  int
}

func (f *fakeModel) Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	if f.calls >= len(f.rounds) {
		return schema.StreamReaderFromArray([]*schema.Message{{Role: schema.Assistant, Content: "fallback"}}), nil
	}
	items := f.rounds[f.calls]
	f.calls++
	return schema.StreamReaderFromArray(items), nil
}

type fakeExecutor struct {
	results map[string]string
}

func (f fakeExecutor) Execute(ctx context.Context, name string, arguments string) (string, error) {
	if result, ok := f.results[name]; ok {
		return result, nil
	}
	data, _ := json.Marshal(map[string]string{"name": name})
	return string(data), nil
}

type recordingExecutor struct {
	arguments string
	result    string
}

func (r *recordingExecutor) Execute(ctx context.Context, name string, arguments string) (string, error) {
	r.arguments = arguments
	return r.result, nil
}

func hasEvent(events []*aiapi.RuntimeStreamEvent, eventType aiapi.EventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
