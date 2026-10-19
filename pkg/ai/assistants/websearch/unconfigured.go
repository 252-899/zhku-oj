package websearch

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type UnconfiguredAssistant struct{}

func NewUnconfigured() *UnconfiguredAssistant {
	return &UnconfiguredAssistant{}
}

func (a *UnconfiguredAssistant) Type() string {
	return CanonicalAssistantType
}

func (a *UnconfiguredAssistant) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	return nil, aiapi.ErrAssistantNotConfigured
}
