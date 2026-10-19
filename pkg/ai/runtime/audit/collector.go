package audit

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
)

type SessionCollector struct {
	mu              sync.Mutex
	text            strings.Builder
	thinking        strings.Builder
	reference       []aiapi.ReferenceItem
	recommend       []aiapi.RecommendItem
	tools           []aiapi.ToolTrace
	firstResponseAt *time.Time
	done            bool
}

func NewSessionCollector() *SessionCollector { return &SessionCollector{} }

func (c *SessionCollector) Append(event *aiapi.RuntimeStreamEvent) {
	if c == nil || event == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch event.Type {
	case aiapi.EventThinking:
		c.markFirstResponseLocked()
		c.thinking.WriteString(readDelta(event))
	case aiapi.EventText:
		c.markFirstResponseLocked()
		c.text.WriteString(readDelta(event))
	case aiapi.EventReference:
		c.markFirstResponseLocked()
		if items, ok := decodeReferenceItems(event.Data); ok {
			c.reference = append(c.reference, items...)
		}
	case aiapi.EventRecommend:
		c.markFirstResponseLocked()
		if items, ok := decodeRecommendItems(event.Data); ok {
			c.recommend = append(c.recommend, items...)
		}
	case aiapi.EventToolCall, aiapi.EventToolResult:
		c.markFirstResponseLocked()
		if item, ok := decodeToolTrace(event.Data, event.Type); ok {
			c.tools = append(c.tools, item)
		}
	case aiapi.EventDone:
		c.done = true
	case aiapi.EventError:
		c.markFirstResponseLocked()
	}
}

func (c *SessionCollector) Done() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done
}

func (c *SessionCollector) BuildSaveResult(req *aiapi.RuntimeRequest, startedAt time.Time, status, errMessage string) *runtimesession.SaveResultInput {
	if c == nil || req == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	result := &runtimesession.SaveResultInput{
		SessionID: req.SessionID, RoundID: req.RoundID, UserID: req.UserID, TaskType: req.TaskType,
		Answer: c.text.String(), Thinking: c.thinking.String(),
		Reference: append([]aiapi.ReferenceItem(nil), c.reference...), Recommend: append([]aiapi.RecommendItem(nil), c.recommend...), Tools: append([]aiapi.ToolTrace(nil), c.tools...),
		TotalResponseMS: time.Since(startedAt).Milliseconds(), Status: status, ErrorMessage: errMessage,
	}
	if c.firstResponseAt != nil {
		result.FirstResponseMS = c.firstResponseAt.Sub(startedAt).Milliseconds()
	}
	return result
}

func (c *SessionCollector) markFirstResponseLocked() {
	if c.firstResponseAt != nil {
		return
	}
	now := time.Now()
	c.firstResponseAt = &now
}

func readDelta(event *aiapi.RuntimeStreamEvent) string {
	if event == nil || event.Data == nil {
		return ""
	}
	switch value := event.Data.(type) {
	case string:
		return value
	case map[string]any:
		for _, key := range []string{"delta", "content", "text"} {
			if text, ok := value[key].(string); ok {
				return text
			}
		}
	}
	return ""
}

func decodeReferenceItems(data any) ([]aiapi.ReferenceItem, bool) {
	switch value := data.(type) {
	case aiapi.ReferenceItem:
		return []aiapi.ReferenceItem{value}, true
	case []aiapi.ReferenceItem:
		return append([]aiapi.ReferenceItem(nil), value...), true
	case map[string]any:
		var item aiapi.ReferenceItem
		if decodeInto(value, &item) == nil {
			return []aiapi.ReferenceItem{item}, true
		}
	case []any:
		items := make([]aiapi.ReferenceItem, 0, len(value))
		for _, raw := range value {
			if nested, ok := decodeReferenceItems(raw); ok {
				items = append(items, nested...)
			}
		}
		if len(items) > 0 {
			return items, true
		}
	}
	return nil, false
}

func decodeRecommendItems(data any) ([]aiapi.RecommendItem, bool) {
	switch value := data.(type) {
	case aiapi.RecommendItem:
		return []aiapi.RecommendItem{value}, true
	case []aiapi.RecommendItem:
		return append([]aiapi.RecommendItem(nil), value...), true
	case map[string]any:
		var item aiapi.RecommendItem
		if decodeInto(value, &item) == nil {
			return []aiapi.RecommendItem{item}, true
		}
	case []any:
		items := make([]aiapi.RecommendItem, 0, len(value))
		for _, raw := range value {
			if nested, ok := decodeRecommendItems(raw); ok {
				items = append(items, nested...)
			}
		}
		if len(items) > 0 {
			return items, true
		}
	}
	return nil, false
}

func decodeToolTrace(data any, eventType aiapi.EventType) (aiapi.ToolTrace, bool) {
	switch value := data.(type) {
	case aiapi.ToolTrace:
		return value, true
	case map[string]any:
		var item aiapi.ToolTrace
		if decodeInto(value, &item) == nil {
			return item, true
		}
		return aiapi.ToolTrace{Stage: string(eventType), Status: readString(value["status"]), Detail: readString(value["error_message"]), Input: value, Output: value, Name: readString(value["name"])}, true
	}
	return aiapi.ToolTrace{}, false
}

func readString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func decodeInto(input any, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}
