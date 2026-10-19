package websearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/assistants"
	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/infra/search"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// runnerInterface 运行时运行器接口
type runnerInterface interface {
	Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
	StreamLLMWithTools(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error)
}

// Assistant WebSearch 问答助手
// 基于显式 ReAct 循环实现：Thought → Action → Observation → ... → Final Answer
// 实现 assistants.AssistantRunnerCore 接口
type Assistant struct {
	runner       runnerInterface       // 运行时运行器
	toolProvider *search.Provider      // 搜索工具提供者
	instruction  string               // 系统提示词
}

// NewAssistant 创建 WebSearchAssistant
func NewAssistant(
	sessionService runtimesession.SessionService,
	taskManager runtimetask.TaskManager,
	contextManager runtimecontext.ContextManager,
	toolProvider *search.Provider,
	chatModel llm.ModelFactory,
	instruction string,
) *Assistant {
	if instruction == "" {
		instruction = DefaultInstruction
	}

	core := &Assistant{
		toolProvider: toolProvider,
		instruction:  instruction,
	}

	core.runner = assistants.NewRunner(
		core, // 实现 AssistantRunnerCore
		sessionService,
		taskManager,
		contextManager,
		chatModel,
	)

	return core
}

// Type 返回 Assistant 类型标识
func (a *Assistant) Type() string {
	return CanonicalAssistantType
}

// BuildMessages 构建 LLM 消息列表
func (a *Assistant) BuildMessages(snapshot *runtimecontext.Snapshot, question string) []*schema.Message {
	count := 2 // System + CurrentQuestion
	if snapshot != nil {
		count += len(snapshot.Messages)
	}
	messages := make([]*schema.Message, 0, count)

	// System Prompt
	messages = append(messages, schema.SystemMessage(a.instruction))

	// 历史对话
	if snapshot != nil {
		for _, item := range snapshot.Messages {
			switch item.Role {
			case "assistant":
				messages = append(messages, schema.AssistantMessage(item.Content, nil))
			default:
				messages = append(messages, schema.UserMessage(item.Content))
			}
		}
	}

	// 当前问题
	messages = append(messages, schema.UserMessage(question))
	return messages
}

// Execute 实现完整 ReAct 循环
func (a *Assistant) Execute(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	snapshot *runtimecontext.Snapshot,
	emit func(*aiapi.RuntimeStreamEvent) error,
) error {
	// Step 1: 构建工具描述
	tools, err := a.toolProvider.BuildTools(ctx)
	if err != nil {
		return err
	}
	toolInfos := toSchemaToolInfos(tools)

	// Step 2: 构建初始消息
	messages := a.BuildMessages(snapshot, req.Message)

	// Step 3: ReAct 主循环
	for round := 0; round < maxRounds; round++ {
		// 3a: LLM 流式推理
		stream, err := a.runner.StreamLLMWithTools(ctx, messages, toolInfos)
		if err != nil {
			return err
		}

		var (
			textContent strings.Builder
			toolCalls  []schema.ToolCall
		)

		// 3b: 流式消费 LLM 输出
		for {
			msg, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return fmt.Errorf("stream recv error: %w", err)
			}
			if msg == nil {
				continue
			}

			// Content 片段：立即推送 SSE
			if msg.Content != "" {
				textContent.WriteString(msg.Content)
				_ = emit(&aiapi.RuntimeStreamEvent{
					Type:      aiapi.EventText,
					SessionID: req.SessionID,
					RoundID:   req.RoundID,
					TaskType:  CanonicalAssistantType,
					Stage:     "answering",
					Data:      map[string]any{"delta": msg.Content},
					Timestamp: time.Now().UnixMilli(),
				})
			}

			// ReasoningContent 片段：立即推送
			if msg.ReasoningContent != "" {
				_ = emit(&aiapi.RuntimeStreamEvent{
					Type:      aiapi.EventThinking,
					SessionID: req.SessionID,
					RoundID:   req.RoundID,
					TaskType:  CanonicalAssistantType,
					Stage:     "answering",
					Data:      map[string]any{"delta": msg.ReasoningContent},
					Timestamp: time.Now().UnixMilli(),
				})
			}

			// ToolCalls：跨 chunk 组装
			if len(msg.ToolCalls) > 0 {
				toolCalls = msg.ToolCalls
			}
		}

		// 3c: 判断本轮结果
		if len(toolCalls) > 0 {
			// 有工具调用：执行 → 追加消息 → 继续
			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventToolCall,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalAssistantType,
				Stage:     "searching",
				Data:      mapToolCallsData(toolCalls),
				Timestamp: time.Now().UnixMilli(),
			})

			messages = append(messages, schema.AssistantMessage(textContent.String(), toolCalls))

			toolMsgs, err := a.executeTools(ctx, req, toolCalls, emit)
			if err != nil {
				return err
			}
			messages = append(messages, toolMsgs...)
			continue
		}

		// 无工具调用：输出最终回答
		if textContent.Len() > 0 {
			messages = append(messages, schema.AssistantMessage(textContent.String(), nil))
		}
		break
	}

	return nil
}

// executeTools 执行工具调用
func (a *Assistant) executeTools(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	toolCalls []schema.ToolCall,
	emit func(*aiapi.RuntimeStreamEvent) error,
) ([]*schema.Message, error) {
	results := make([]*schema.Message, 0, len(toolCalls))
	references := make(map[string]aiapi.ReferenceItem)

	for _, tc := range toolCalls {
		// 解析参数
		var input search.SearchInput
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Content:    fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error()),
			})
			continue
		}

		// 发射 thinking
		if strings.Contains(tc.Function.Name, "search") {
			queryThink := "🔍 正在搜索信息"
			if input.Query != "" {
				queryThink += ": " + input.Query
			}
			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventThinking,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalAssistantType,
				Stage:     "searching",
				Data:      map[string]any{"delta": queryThink + "\n"},
				Timestamp: time.Now().UnixMilli(),
			})
		}

		// 执行工具
		var toolResult []search.SearchResult
		var toolErr error

		if strings.Contains(tc.Function.Name, "search") {
			toolResult, toolErr = a.toolProvider.Search(ctx, input.Query, input.TopK)
		} else {
			toolErr = fmt.Errorf("unsupported tool: %s", tc.Function.Name)
		}

		// 记录结果
		if toolErr != nil {
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Content:    fmt.Sprintf(`{"error": "%s"}`, toolErr.Error()),
			})
		} else {
			resultJSON, _ := json.Marshal(toolResult)
			results = append(results, &schema.Message{
				Role:       schema.Tool,
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Content:    string(resultJSON),
			})
		}

		// 发射 tool_result
		resultStatus := "success"
		resultCount := len(toolResult)
		if toolErr != nil {
			resultStatus = "failed"
			resultCount = 0
		}
		_ = emit(&aiapi.RuntimeStreamEvent{
			Type:      aiapi.EventToolResult,
			SessionID: req.SessionID,
			RoundID:   req.RoundID,
			TaskType:  CanonicalAssistantType,
			Stage:     "searching",
			Data: map[string]any{
				"name":    tc.Function.Name,
				"status":  resultStatus,
				"results": resultCount,
			},
			Timestamp: time.Now().UnixMilli(),
		})

		// 收集 references
		for _, r := range toolResult {
			key := r.URL
			if key == "" {
				key = r.Title + "|" + r.Snippet
			}
			references[key] = aiapi.ReferenceItem{
				Title:   r.Title,
				URL:     r.URL,
				Snippet: r.Snippet,
			}
		}
	}

	// 统一发射 references
	if len(references) > 0 {
		items := make([]aiapi.ReferenceItem, 0, len(references))
		for _, ref := range references {
			items = append(items, ref)
		}
		_ = emit(&aiapi.RuntimeStreamEvent{
			Type:      aiapi.EventReference,
			SessionID: req.SessionID,
			RoundID:   req.RoundID,
			TaskType:  CanonicalAssistantType,
			Stage:     "answering",
			Data:      items,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	return results, nil
}

// Stream 委托给 Runner
func (a *Assistant) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if a.runner == nil {
		return nil, aiapi.ErrAssistantNotConfigured
	}
	return a.runner.Stream(ctx, req)
}

// ========== 辅助函数 ==========

const maxRounds = 10

// toSchemaToolInfos 将 eino tools 转换为 schema.ToolInfo 列表
func toSchemaToolInfos(tools []tool.BaseTool) []*schema.ToolInfo {
	result := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		if td, ok := t.(interface{ GetToolDefinition() *schema.ToolInfo }); ok {
			result = append(result, td.GetToolDefinition())
		}
	}
	return result
}

// mapToolCallsData 将 toolCalls 转为可展示的 map
func mapToolCallsData(toolCalls []schema.ToolCall) map[string]any {
	if len(toolCalls) == 0 {
		return nil
	}
	data := make([]map[string]any, 0, len(toolCalls))
	for _, tc := range toolCalls {
		var input struct {
			Query string `json:"query"`
			TopK  int    `json:"top_k"`
		}
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		data = append(data, map[string]any{
			"name":   tc.Function.Name,
			"query":  input.Query,
			"top_k":  input.TopK,
		})
	}
	return map[string]any{"tools": data}
}
