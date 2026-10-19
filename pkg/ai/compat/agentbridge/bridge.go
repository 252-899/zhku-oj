package agentbridge

import (
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

func FromChatRequest(req *aiapi.ChatRequest) *aiapi.RuntimeRequest {
	if req == nil {
		return nil
	}

	// 根据 TaskKind 或 ModuleType 判断路由目标
	taskKind := aiapi.TaskKindAssistant
	moduleType := ""
	assistantType := ""

	if req.TaskKind == string(aiapi.TaskKindModule) || req.ModuleType != "" {
		taskKind = aiapi.TaskKindModule
		moduleType = normalizeTaskType(req.ModuleType)
	} else {
		assistantType = normalizeTaskType(req.AgentType)
	}

	// 提取 metadata.input 到 Input（供 Module 使用）
	input := extractInputFromMetadata(req.Metadata)

	return &aiapi.RuntimeRequest{
		SessionID:     strings.TrimSpace(req.SessionID),
		RoundID:       strings.TrimSpace(req.RoundID),
		UserID:        strings.TrimSpace(req.UserID),
		TaskKind:      taskKind,
		TaskType:      moduleType,
		ModuleType:    moduleType,
		AssistantType: assistantType,
		Message:       strings.TrimSpace(req.Message),
		Metadata:      cloneMap(req.Metadata),
		Input:         input,
	}
}

// extractInputFromMetadata 从 metadata.input 提取 Input map
func extractInputFromMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}
	if input, ok := metadata["input"].(map[string]any); ok {
		return input
	}
	return nil
}

func ToChatRequest(req *aiapi.RuntimeRequest) *aiapi.ChatRequest {
	if req == nil {
		return nil
	}
	req.Normalize()

	chatReq := &aiapi.ChatRequest{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		Message:   req.Message,
		Metadata:  cloneMap(req.Metadata),
	}

	switch req.TaskKind {
	case aiapi.TaskKindModule:
		chatReq.ModuleType = agentTypeFromTaskType(req.ModuleType)
		chatReq.TaskKind = string(aiapi.TaskKindModule)
	default:
		chatReq.AgentType = agentTypeFromTaskType(req.AssistantType)
	}

	return chatReq
}

func ToStreamEvent(event *aiapi.RuntimeStreamEvent) *aiapi.StreamEvent {
	if event == nil {
		return nil
	}
	return &aiapi.StreamEvent{
		Type:         event.Type,
		SessionID:    event.SessionID,
		RoundID:      event.RoundID,
		AgentType:    agentTypeFromTaskType(event.TaskType),
		Content:      extractContent(event),
		Data:         event.Data,
		ErrorMessage: event.ErrorMessage,
		Timestamp:    event.Timestamp,
	}
}

func FromStreamEvent(event *aiapi.StreamEvent, req *aiapi.RuntimeRequest) *aiapi.RuntimeStreamEvent {
	if event == nil {
		return nil
	}
	if req != nil {
		req.Normalize()
	}

	mapped := aiapi.NewRuntimeStreamEvent(event.Type, req, inferStage(event), normalizeData(event))
	if mapped.SessionID == "" {
		mapped.SessionID = event.SessionID
	}
	if mapped.RoundID == "" {
		mapped.RoundID = event.RoundID
	}
	if mapped.TaskType == "" {
		mapped.TaskType = normalizeTaskType(event.AgentType)
	}
	if event.Timestamp != 0 {
		mapped.Timestamp = event.Timestamp
	}
	if event.Type == aiapi.EventError {
		mapped.ErrorMessage = event.ErrorMessage
	}
	return mapped
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func extractContent(event *aiapi.RuntimeStreamEvent) string {
	if event == nil || event.Data == nil {
		return ""
	}
	switch value := event.Data.(type) {
	case string:
		return value
	case map[string]any:
		for _, key := range []string{"delta", "content", "text"} {
			if content, ok := value[key].(string); ok {
				return content
			}
		}
	}
	return ""
}

func inferStage(event *aiapi.StreamEvent) string {
	if event == nil {
		return ""
	}
	switch event.Type {
	case aiapi.EventMeta:
		return "running"
	case aiapi.EventToolCall, aiapi.EventToolResult:
		return "searching"
	case aiapi.EventText, aiapi.EventThinking, aiapi.EventReference:
		return "answering"
	case aiapi.EventDone:
		if payload, ok := event.Data.(map[string]any); ok {
			if status, ok := payload["status"].(string); ok {
				return status
			}
		}
		return "completed"
	case aiapi.EventError:
		return "failed"
	default:
		return ""
	}
}

func normalizeData(event *aiapi.StreamEvent) any {
	if event == nil {
		return nil
	}
	switch event.Type {
	case aiapi.EventText, aiapi.EventThinking:
		if event.Content != "" {
			return map[string]any{"delta": event.Content}
		}
	}
	if event.Data != nil {
		return event.Data
	}
	if event.Content != "" {
		return event.Content
	}
	return nil
}

func normalizeTaskType(agentType string) string {
	switch strings.ToLower(strings.TrimSpace(agentType)) {
	case "", aiapi.WebSearchAgentType, "websearch", "web_search":
		return "websearch"
	default:
		return strings.ToLower(strings.TrimSpace(agentType))
	}
}

func agentTypeFromTaskType(taskType string) string {
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "", "websearch", "web_search", aiapi.WebSearchAgentType:
		return aiapi.WebSearchAgentType
	case aiapi.OJQuestionGenModuelType, "ojquestiongen":
		return aiapi.OJQuestionGenModuelType
	default:
		return strings.ToLower(strings.TrimSpace(taskType))
	}
}
