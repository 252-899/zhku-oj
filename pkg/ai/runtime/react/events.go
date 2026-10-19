// events.go 将 ReAct 循环内的活动映射为现有运行时流事件。
package react

import (
	"encoding/json"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"

	"github.com/cloudwego/eino/schema"
)

// NewEvent 填充显式 ReAct 循环所需的通用运行时事件字段。
func NewEvent(eventType aiapi.EventType, req *aiapi.RuntimeRequest, stage string, data any) *aiapi.RuntimeStreamEvent {
	event := &aiapi.RuntimeStreamEvent{
		Type:      eventType,
		Stage:     stage,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	if req != nil {
		event.SessionID = req.SessionID
		event.RoundID = req.RoundID
		event.TaskType = req.TaskType
	}
	return event
}

func mapToolCallsData(toolCalls []schema.ToolCall) map[string]any {
	items := make([]map[string]any, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		item := map[string]any{
			"id":   toolCall.ID,
			"name": toolCall.Function.Name,
		}
		var args map[string]any
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
			item["arguments"] = args
		}
		items = append(items, item)
	}
	return map[string]any{"tools": items}
}
