package modules

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type Module interface {
	Type() string
	Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
}
