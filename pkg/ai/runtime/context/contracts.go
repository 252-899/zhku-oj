package context

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type Snapshot struct {
	Messages []aiapi.HistoryMessage `json:"messages"`
}

type ContextManager interface {
	Build(ctx context.Context, req *aiapi.RuntimeRequest) (*Snapshot, error)
}
