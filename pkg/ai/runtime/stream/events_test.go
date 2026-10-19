package stream

import (
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

func TestMetaEventFillsRuntimeEnvelope(t *testing.T) {
	req := &aiapi.RuntimeRequest{
		AssistantType: "websearch",
	}

	event := MetaEvent(req, map[string]any{"title": "websearch"})
	if event.Type != aiapi.EventMeta {
		t.Fatalf("expected meta event, got %s", event.Type)
	}
	if event.SessionID == "" || event.RoundID == "" {
		t.Fatalf("expected identifiers to be generated")
	}
	if event.TaskType != "websearch" {
		t.Fatalf("expected task type websearch, got %s", event.TaskType)
	}
	if event.Stage != "running" {
		t.Fatalf("expected running stage, got %s", event.Stage)
	}
}
