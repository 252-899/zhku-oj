package task

import (
	"context"

	"zk-code-arena-server/pkg/ai/runtime/state"
)

type TaskManager interface {
	RegisterRoot(root *state.RootTaskRuntime) error
	GetRoot(sessionID string) (*state.RootTaskRuntime, bool)
	GetRound(sessionID, roundID string) (*state.RootTaskRuntime, bool)
	HasRunning(sessionID string) bool
	UpdateRootState(sessionID, roundID, stage string, status state.TaskStatus) error
	AddChild(sessionID, roundID string, child *state.ChildTaskRuntime) error
	UpdateChild(sessionID, roundID string, child *state.ChildTaskRuntime) error
	ListChildren(sessionID, roundID string) []*state.ChildTaskRuntime
	Stop(ctx context.Context, sessionID, roundID, userID string) error
	RemoveRoot(sessionID, roundID string)
}
