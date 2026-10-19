package gateway

import (
	"context"
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/assistants"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
)

type stubAssistant struct{}

func (s *stubAssistant) Type() string { return "websearch" }

func (s *stubAssistant) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	out := make(chan *aiapi.RuntimeStreamEvent, 1)
	out <- aiapi.NewRuntimeStreamEvent(aiapi.EventMeta, req, "running", map[string]any{"status": "running"})
	close(out)
	return out, nil
}

func TestAssistantGatewayNormalizesAlias(t *testing.T) {
	registry := assistants.NewRegistry()
	registry.Register(&stubAssistant{})
	gateway := NewAssistantGateway(registry, runtimetask.NewMemoryTaskManager())

	stream, err := gateway.Stream(context.Background(), &aiapi.RuntimeRequest{TaskType: "websearchagent"})
	if err != nil {
		t.Fatalf("gateway stream error: %v", err)
	}
	first := <-stream
	if first == nil {
		t.Fatalf("expected event")
	}
	if first.TaskType != "websearch" {
		t.Fatalf("expected normalized task type websearch, got %s", first.TaskType)
	}
}

func TestAssistantGatewayReturnsNotRegistered(t *testing.T) {
	gateway := NewAssistantGateway(assistants.NewRegistry(), runtimetask.NewMemoryTaskManager())
	_, err := gateway.Stream(context.Background(), &aiapi.RuntimeRequest{TaskType: "unknown"})
	if err != aiapi.ErrAssistantNotRegistered {
		t.Fatalf("expected assistant not registered, got %v", err)
	}
}
