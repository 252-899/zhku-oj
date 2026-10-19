package assistants

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type Assistant interface {
	Type() string
	Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
}
