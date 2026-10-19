package gateway

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/modules"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
)

type ModuleGateway interface {
	Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
	Stop(ctx context.Context, sessionID, roundID, userID string) error
}

type moduleGateway struct {
	registry    *modules.Registry
	taskManager runtimetask.TaskManager
}

func NewModuleGateway(registry *modules.Registry, taskManager runtimetask.TaskManager) ModuleGateway {
	return &moduleGateway{registry: registry, taskManager: taskManager}
}

func (g *moduleGateway) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if req == nil {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	if g == nil || g.registry == nil {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	req.Normalize()
	req.TaskKind = aiapi.TaskKindModule
	req.ModuleType = modules.NormalizeType(firstNonEmpty(req.ModuleType, req.TaskType))
	if req.ModuleType == "" {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	req.TaskType = req.ModuleType

	module, ok := g.registry.Get(req.ModuleType)
	if !ok {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	return module.Stream(ctx, req)
}

func (g *moduleGateway) Stop(ctx context.Context, sessionID, roundID, userID string) error {
	if g == nil || g.taskManager == nil {
		return aiapi.ErrStopNotImplemented
	}
	return g.taskManager.Stop(ctx, sessionID, roundID, userID)
}
