// state.go 保存显式 ReAct 循环中单轮模型输出的临时状态。
package react

import (
	"strings"

	"github.com/cloudwego/eino/schema"
)

type roundState struct {
	text         strings.Builder
	toolCalls    []schema.ToolCall
	seenToolCall bool
}

func newRoundState() *roundState {
	return &roundState{toolCalls: []schema.ToolCall{}}
}

// mergeToolCalls 处理模型把同一个工具调用拆成多个 chunk 输出的情况。
// 同一个 ID 或 index 代表同一次工具调用，参数需要拼接。
func (s *roundState) mergeToolCalls(incoming []schema.ToolCall) {
	for _, toolCall := range incoming {
		if toolCall.Type == "" {
			toolCall.Type = "function"
		}
		index := s.findToolCallIndex(toolCall)
		if index < 0 {
			s.toolCalls = append(s.toolCalls, toolCall)
			continue
		}

		existing := s.toolCalls[index]
		if existing.ID == "" {
			existing.ID = toolCall.ID
		}
		if existing.Type == "" {
			existing.Type = toolCall.Type
		}
		if existing.Function.Name == "" {
			existing.Function.Name = toolCall.Function.Name
		}
		existing.Function.Arguments += toolCall.Function.Arguments
		s.toolCalls[index] = existing
	}
}

func (s *roundState) findToolCallIndex(target schema.ToolCall) int {
	for i, existing := range s.toolCalls {
		if target.ID != "" && existing.ID == target.ID {
			return i
		}
		if target.Index != nil && existing.Index != nil && *target.Index == *existing.Index {
			return i
		}
	}
	return -1
}
