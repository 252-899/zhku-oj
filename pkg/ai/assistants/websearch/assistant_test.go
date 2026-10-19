package websearch

import (
	"context"
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/infra/search"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// TestAssistantStreamSuccess 测试完整 ReAct 流程：
// 第一轮：LLM 请求 web_search 工具 → 执行搜索 → 第二轮：LLM 输出最终回答
func TestAssistantStreamSuccess(t *testing.T) {
	sessionService := newSpySessionService()
	taskManager := runtimetask.NewMemoryTaskManager()
	contextManager := runtimecontext.NewManager(sessionService, 10)
	toolProvider := search.NewProvider(&fakeTavilyClient{}, search.Config{APIKey: "test-key"})

	// 模拟 LLM 返回两条消息：
	// 第一条：tool_calls（触发搜索）
	// 第二条：最终文本回答
	llmMessages := []*schema.Message{
		{
			Role:       schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "call-1",
					Type: "function",
					Function: schema.FunctionCall{
						Name:      "web_search",
						Arguments: `{"query":"golang release","top_k":2}`,
					},
				},
			},
		},
		{
			Role:    schema.Assistant,
			Content: "Go release notes are available on the official website.",
		},
	}
	chatModel := &fakeChatModel{messages: llmMessages}

	assistant := NewAssistant(
		sessionService,
		taskManager,
		contextManager,
		toolProvider,
		chatModel,
		DefaultInstruction,
	)

	stream, err := assistant.Stream(context.Background(), &aiapi.RuntimeRequest{
		SessionID:     "session-1",
		RoundID:       "round-1",
		UserID:        "user-1",
		AssistantType: "websearch",
		Message:       "golang 最新版本是什么",
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var eventTypes []aiapi.EventType
	var referenceEvents int
	var finalAnswer string
	for event := range stream {
		if event == nil {
			continue
		}
		eventTypes = append(eventTypes, event.Type)
		if event.Type == aiapi.EventReference {
			referenceEvents++
			refs, ok := event.Data.([]aiapi.ReferenceItem)
			if !ok {
				t.Fatalf("reference event should contain []ReferenceItem, got %#v", event.Data)
			}
			if len(refs) != 1 || refs[0].URL != "https://go.dev/doc/devel/release" {
				t.Fatalf("unexpected references: %#v", refs)
			}
		}
		if event.Type == aiapi.EventText {
			if data, ok := event.Data.(map[string]any); ok {
				if delta, ok := data["delta"].(string); ok {
					finalAnswer += delta
				}
			}
		}
	}

	// 验证关键事件存在（顺序可能因 goroutine 调度而波动）
	eventSet := make(map[aiapi.EventType]bool)
	for _, e := range eventTypes {
		eventSet[e] = true
	}
	for _, e := range []aiapi.EventType{aiapi.EventMeta, aiapi.EventToolCall, aiapi.EventToolResult, aiapi.EventReference, aiapi.EventDone} {
		if !eventSet[e] {
			t.Fatalf("missing event type: %v (got: %v)", e, eventTypes)
		}
	}
	if referenceEvents != 1 {
		t.Fatalf("expected one reference event, got %d", referenceEvents)
	}
	if chatModel.lastToolInfos == nil {
		t.Fatal("expected tools to be passed to chatModel")
	}

	// 验证最终保存结果
	if sessionService.lastSaveAnswer == nil {
		t.Fatal("expected SaveAnswer to be called")
	}
	if sessionService.lastSaveAnswer.Status != "completed" {
		t.Fatalf("unexpected status: %q", sessionService.lastSaveAnswer.Status)
	}
	if sessionService.lastSaveAnswer.Answer != "Go release notes are available on the official website." {
		t.Fatalf("unexpected answer: %q", sessionService.lastSaveAnswer.Answer)
	}
	if len(sessionService.lastSaveAnswer.Reference) != 1 {
		t.Fatalf("unexpected saved references: %#v", sessionService.lastSaveAnswer.Reference)
	}
	// ToolCall + ToolResult = 2 条工具审计记录
	if len(sessionService.lastSaveAnswer.Tools) != 2 {
		t.Fatalf("expected two tool audit records, got %#v", sessionService.lastSaveAnswer.Tools)
	}
}

func TestAssistantReturnsNotConfiguredWhenDepsMissing(t *testing.T) {
	assistant := &Assistant{}
	_, err := assistant.Stream(context.Background(), &aiapi.RuntimeRequest{AssistantType: "websearch", Message: "hello"})
	if err != aiapi.ErrAssistantNotConfigured {
		t.Fatalf("expected assistant not configured, got %v", err)
	}
}

// ========== Test Fakes ==========

type fakeTavilyClient struct{}

func (c *fakeTavilyClient) Search(ctx context.Context, query string, topK int) ([]search.SearchResult, error) {
	return []search.SearchResult{{
		Title:   "Go Releases",
		URL:     "https://go.dev/doc/devel/release",
		Snippet: "Official Go release notes",
	}}, nil
}

// fakeChatModel 实现 llm.ModelFactory 接口（Stream 方法被实际调用）
// 模拟 LLM 的 ReAct 行为：
//   - 第一轮调用：返回 tool_calls（触发搜索）
//   - 第二轮调用（搜索结果已追加到 messages）：返回最终文本回答
type fakeChatModel struct {
	messages      []*schema.Message
	lastToolInfos []*schema.ToolInfo
	callCount     int // 追踪 Stream 被调用次数，模拟多轮 ReAct
}

func (f *fakeChatModel) IsChatConfigured() bool { return true }
func (f *fakeChatModel) ChatConfig() llm.ChatConfig { return llm.ChatConfig{} }

func (f *fakeChatModel) NewToolCallingChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	return &nopChatModel{}, nil
}

// Stream 第一轮返回 tool_calls，第二轮返回最终文本回答
func (f *fakeChatModel) Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	f.lastToolInfos = tools
	f.callCount++

	var msg *schema.Message
	if f.callCount == 1 {
		// 第一轮：LLM 决定调用工具（返回 f.messages[0]，即带有 tool_calls 的消息）
		msg = f.messages[0]
	} else {
		// 第二轮：LLM 基于工具结果输出最终文本回答
		msg = &schema.Message{
			Role:    schema.Assistant,
			Content: "Go release notes are available on the official website.",
		}
	}

	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(msg, nil)
	}()
	return sr, nil
}

// Call 同步调用（websearch 不使用，但实现接口）
func (f *fakeChatModel) Call(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "not used by websearch",
	}, nil
}

// nopChatModel 实现 model.ToolCallingChatModel（测试中不会被调用）
type nopChatModel struct{}

func (n *nopChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return nil, nil
}
func (n *nopChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}
func (n *nopChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return n, nil
}

// spySessionService 包装 MemorySessionService，记录最后保存的答案
type spySessionService struct {
	*runtimesession.MemorySessionService
	lastSaveAnswer *runtimesession.SaveResultInput
}

func newSpySessionService() *spySessionService {
	return &spySessionService{MemorySessionService: runtimesession.NewMemorySessionService()}
}

func (s *spySessionService) SaveResult(ctx context.Context, input *runtimesession.SaveResultInput) error {
	if input != nil {
		copied := *input
		copied.Reference = append([]aiapi.ReferenceItem(nil), input.Reference...)
		copied.Recommend = append([]aiapi.RecommendItem(nil), input.Recommend...)
		copied.Tools = append([]aiapi.ToolTrace(nil), input.Tools...)
		s.lastSaveAnswer = &copied
	}
	return s.MemorySessionService.SaveResult(ctx, input)
}
