package agentbridge

import (
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

func TestChatRequestBridge(t *testing.T) {
	legacy := &aiapi.ChatRequest{
		SessionID: "session-1",
		AgentType: "websearchagent",
		Message:   "hello",
		Metadata: map[string]interface{}{
			"k": "v",
		},
	}

	runtimeReq := FromChatRequest(legacy)
	if runtimeReq.TaskKind != aiapi.TaskKindAssistant {
		t.Fatalf("expected assistant task kind, got %s", runtimeReq.TaskKind)
	}
	if runtimeReq.TaskType != "websearch" {
		t.Fatalf("expected task type websearch, got %s", runtimeReq.TaskType)
	}
	if runtimeReq.Metadata["k"] != "v" {
		t.Fatalf("expected metadata to be copied")
	}
}

func TestAssistantRequestBridge(t *testing.T) {
	runtimeReq := &aiapi.RuntimeRequest{
		SessionID:     "session-1",
		AssistantType: "websearch",
		Message:       "hello",
	}

	legacyReq := ToChatRequest(runtimeReq)
	if legacyReq.AgentType != aiapi.WebSearchAgentType {
		t.Fatalf("expected legacy agent type %s, got %s", aiapi.WebSearchAgentType, legacyReq.AgentType)
	}
}

func TestStreamEventBridge(t *testing.T) {
	event := &aiapi.StreamEvent{
		Type:      aiapi.EventText,
		SessionID: "session-1",
		AgentType: aiapi.WebSearchAgentType,
		Content:   "hello",
	}

	runtimeReq := &aiapi.RuntimeRequest{SessionID: "session-1", RoundID: "round-1", AssistantType: "websearch"}
	runtimeEvent := FromStreamEvent(event, runtimeReq)
	if runtimeEvent.TaskType != "websearch" {
		t.Fatalf("expected task type websearch, got %s", runtimeEvent.TaskType)
	}
	if delta, _ := runtimeEvent.Data.(map[string]any)["delta"].(string); delta != "hello" {
		t.Fatalf("expected delta hello, got %#v", runtimeEvent.Data)
	}
}
