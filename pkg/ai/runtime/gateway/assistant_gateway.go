package gateway

import (
	"context"
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/assistants"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
)

type AssistantGateway interface {
	Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
	Stop(ctx context.Context, sessionID, roundID, userID string) error
}

type assistantGateway struct {
	registry    *assistants.Registry
	taskManager runtimetask.TaskManager
}

func NewAssistantGateway(registry *assistants.Registry, taskManager runtimetask.TaskManager) AssistantGateway {
	return &assistantGateway{registry: registry, taskManager: taskManager}
}

func (g *assistantGateway) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if req == nil {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	if g == nil || g.registry == nil {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	req.Normalize()
	req.TaskKind = aiapi.TaskKindAssistant
	req.AssistantType = assistants.NormalizeType(firstNonEmpty(req.AssistantType, req.TaskType))
	if req.AssistantType == "" {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	req.TaskType = req.AssistantType

	assistant, ok := g.registry.Get(req.AssistantType)
	if !ok {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	return assistant.Stream(ctx, req)
}

func (g *assistantGateway) Stop(ctx context.Context, sessionID, roundID, userID string) error {
	if g == nil || g.taskManager == nil {
		return aiapi.ErrStopNotImplemented
	}
	return g.taskManager.Stop(ctx, sessionID, roundID, userID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
