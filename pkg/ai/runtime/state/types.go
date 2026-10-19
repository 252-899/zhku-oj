package state

import (
	"context"
	"maps"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusStopped   TaskStatus = "stopped"
)

type RootTaskRuntime struct {
	SessionID string
	RoundID   string
	UserID    string
	TaskKind  aiapi.TaskKind
	TaskType  string
	Stage     string
	Status    TaskStatus
	Metadata  map[string]any
	StartedAt time.Time
	UpdatedAt time.Time
	Cancel    context.CancelFunc
}

func (r *RootTaskRuntime) Clone() *RootTaskRuntime {
	if r == nil {
		return nil
	}
	clone := *r
	clone.Metadata = maps.Clone(r.Metadata)
	clone.Cancel = nil
	return &clone
}

type ChildTaskRuntime struct {
	ChildID       string
	ParentRoundID string
	TaskType      string
	Stage         string
	Status        TaskStatus
	Index         int
	ErrorMessage  string
	Metadata      map[string]any
	StartedAt     time.Time
	UpdatedAt     time.Time
	Cancel        context.CancelFunc
}

func (c *ChildTaskRuntime) Clone() *ChildTaskRuntime {
	if c == nil {
		return nil
	}
	clone := *c
	clone.Metadata = maps.Clone(c.Metadata)
	clone.Cancel = nil
	return &clone
}
