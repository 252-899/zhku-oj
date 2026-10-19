// 包 react 实现项目内显式 ReAct 循环。
package react

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"

	"github.com/cloudwego/eino/schema"
)

const (
	defaultMaxRounds = 10
	defaultStage     = "answering"
	toolStage        = "tool_calling"
)

// ModelStreamer 是 ReAct 循环需要的最小模型能力接口。
// llm.ModelFactory 已经满足该接口。
type ModelStreamer interface {
	Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error)
}

// ToolExecutor 执行模型请求的工具调用。OJ 工具 Provider 已通过
// Execute(ctx, name, arguments) 暴露同样形状的能力。
type ToolExecutor interface {
	Execute(ctx context.Context, name string, arguments string) (string, error)
}

// EmitFunc 将运行时事件发送给外层 assistant 或 module runner。
type EmitFunc func(*aiapi.RuntimeStreamEvent) error

// Config 控制显式 ReAct 循环的轮次、兜底提示词和事件阶段。
type Config struct {
	MaxRounds        int
	ForceFinalPrompt string
	Stage            string
}

// Runner 表示一次可复用的显式 ReAct 执行器配置。
type Runner struct {
	model    ModelStreamer
	executor ToolExecutor
	tools    []*schema.ToolInfo
	config   Config
}

// NewRunner 创建可复用的显式 ReAct runner。
func NewRunner(model ModelStreamer, executor ToolExecutor, tools []*schema.ToolInfo, config Config) *Runner {
	if config.MaxRounds <= 0 {
		config.MaxRounds = defaultMaxRounds
	}
	if strings.TrimSpace(config.Stage) == "" {
		config.Stage = defaultStage
	}
	if strings.TrimSpace(config.ForceFinalPrompt) == "" {
		config.ForceFinalPrompt = DefaultForceFinalPrompt
	}
	return &Runner{
		model:    model,
		executor: executor,
		tools:    tools,
		config:   config,
	}
}

// Run 执行一次流式 ReAct 循环。返回的 messages 会包含本次运行追加的
// assistant 消息和 tool 消息。
func (r *Runner) Run(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	messages []*schema.Message,
	emit EmitFunc,
) ([]*schema.Message, error) {
	if r == nil || r.model == nil {
		return nil, errors.New("react runner model is not configured")
	}
	if r.executor == nil {
		return nil, errors.New("react runner tool executor is not configured")
	}
	if emit == nil {
		emit = func(*aiapi.RuntimeStreamEvent) error { return nil }
	}

	working := append([]*schema.Message(nil), messages...)
	for round := 1; ; round++ {
		if round > r.config.MaxRounds {
			return r.forceFinal(ctx, req, working, emit)
		}

		state, err := r.runRound(ctx, req, working, emit)
		if err != nil {
			return working, err
		}

		if len(state.toolCalls) == 0 {
			if state.text.Len() > 0 {
				working = append(working, schema.AssistantMessage(state.text.String(), nil))
			}
			return working, nil
		}

		if err := emit(NewEvent(aiapi.EventToolCall, req, toolStage, mapToolCallsData(state.toolCalls))); err != nil {
			return working, err
		}

		working = append(working, schema.AssistantMessage(state.text.String(), state.toolCalls))
		toolMessages := r.executeToolCalls(ctx, req, state.toolCalls, emit)
		working = append(working, toolMessages...)
	}
}

func (r *Runner) runRound(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	messages []*schema.Message,
	emit EmitFunc,
) (*roundState, error) {
	stream, err := r.model.Stream(ctx, messages, r.tools)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	state := newRoundState()
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("react stream recv failed: %w", err)
		}
		if msg == nil {
			continue
		}

		if msg.ReasoningContent != "" {
			if err := emit(NewEvent(aiapi.EventThinking, req, r.config.Stage, map[string]any{"delta": msg.ReasoningContent})); err != nil {
				return nil, err
			}
		}

		if len(msg.ToolCalls) > 0 {
			state.seenToolCall = true
			state.mergeToolCalls(msg.ToolCalls)
			continue
		}

		if msg.Content != "" {
			state.text.WriteString(msg.Content)
			if !state.seenToolCall {
				if err := emit(NewEvent(aiapi.EventText, req, r.config.Stage, map[string]any{"delta": msg.Content})); err != nil {
					return nil, err
				}
			}
		}
	}

	return state, nil
}

func (r *Runner) executeToolCalls(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	toolCalls []schema.ToolCall,
	emit EmitFunc,
) []*schema.Message {
	results := make([]*schema.Message, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		result, err := r.executor.Execute(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
		status := "success"
		if err != nil {
			status = "failed"
			result = marshalToolError(err)
		}

		_ = emit(NewEvent(aiapi.EventToolResult, req, toolStage, map[string]any{
			"name":   toolCall.Function.Name,
			"status": status,
		}))

		results = append(results, schema.ToolMessage(result, toolCall.ID, schema.WithToolName(toolCall.Function.Name)))
	}
	return results
}

func (r *Runner) forceFinal(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	messages []*schema.Message,
	emit EmitFunc,
) ([]*schema.Message, error) {
	working := append([]*schema.Message(nil), messages...)
	working = append(working, schema.UserMessage(r.config.ForceFinalPrompt))
	state, err := r.runRound(ctx, req, working, emit)
	if err != nil {
		return working, err
	}
	if state.text.Len() > 0 {
		working = append(working, schema.AssistantMessage(state.text.String(), nil))
	}
	return working, nil
}

func marshalToolError(err error) string {
	data, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return `{"error":"tool execution failed"}`
	}
	return string(data)
}
