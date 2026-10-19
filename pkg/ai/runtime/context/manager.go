package context

import (
	"context"

	aiapi "zk-code-arena-server/pkg/ai/api"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
)

type Manager struct {
	sessions runtimesession.SessionService
	limit    int
}

func NewManager(sessions runtimesession.SessionService, limit int) *Manager {
	if limit <= 0 {
		limit = 10
	}
	return &Manager{sessions: sessions, limit: limit}
}

func (m *Manager) Build(ctx context.Context, req *aiapi.RuntimeRequest) (*Snapshot, error) {
	if m == nil || m.sessions == nil || req == nil {
		return &Snapshot{}, nil
	}
	history, err := m.sessions.LoadRecent(ctx, req.SessionID, req.UserID, m.limit)
	if err != nil {
		return nil, err
	}
	snapshot := &Snapshot{Messages: make([]aiapi.HistoryMessage, 0, len(history)*2)}
	for _, item := range history {
		if item == nil {
			continue
		}
		if item.Question != "" {
			snapshot.Messages = append(snapshot.Messages, aiapi.HistoryMessage{Role: "user", Content: item.Question})
		}
		if item.Answer != "" {
			snapshot.Messages = append(snapshot.Messages, aiapi.HistoryMessage{Role: "assistant", Content: item.Answer})
		}
	}
	return snapshot, nil
}
