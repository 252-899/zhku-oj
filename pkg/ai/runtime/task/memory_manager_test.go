package task

import (
	"context"
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/runtime/state"
)

func TestMemoryTaskManagerRegisterAndStopByRound(t *testing.T) {
	manager := NewMemoryTaskManager()
	root := &state.RootTaskRuntime{
		SessionID: "session-1",
		RoundID:   "round-1",
		UserID:    "user-1",
		TaskKind:  aiapi.TaskKindAssistant,
		TaskType:  "websearch",
	}

	stopped := false
	root.Cancel = func() {
		stopped = true
	}

	if err := manager.RegisterRoot(root); err != nil {
		t.Fatalf("register root: %v", err)
	}
	if err := manager.Stop(context.Background(), "session-1", "round-mismatch", "user-1"); err != aiapi.ErrTaskNotFound {
		t.Fatalf("expected round mismatch to return task not found, got %v", err)
	}
	if err := manager.Stop(context.Background(), "session-1", "round-1", "user-1"); err != nil {
		t.Fatalf("stop root: %v", err)
	}
	if !stopped {
		t.Fatalf("expected cancel func to be called")
	}

	rootSnapshot, ok := manager.GetRound("session-1", "round-1")
	if !ok {
		t.Fatalf("expected round to exist")
	}
	if rootSnapshot.Status != state.TaskStatusStopped {
		t.Fatalf("expected stopped status, got %s", rootSnapshot.Status)
	}
}

func TestMemoryTaskManagerAddAndListChildren(t *testing.T) {
	manager := NewMemoryTaskManager()
	if err := manager.RegisterRoot(&state.RootTaskRuntime{
		SessionID: "session-2",
		RoundID:   "round-2",
		TaskKind:  aiapi.TaskKindModule,
		TaskType:  "questiongen",
	}); err != nil {
		t.Fatalf("register root: %v", err)
	}

	if err := manager.AddChild("session-2", "round-2", &state.ChildTaskRuntime{
		ChildID:  "child-2",
		Index:    2,
		TaskType: "generate_problem",
	}); err != nil {
		t.Fatalf("add child 2: %v", err)
	}
	if err := manager.AddChild("session-2", "round-2", &state.ChildTaskRuntime{
		ChildID:  "child-1",
		Index:    1,
		TaskType: "generate_problem",
	}); err != nil {
		t.Fatalf("add child 1: %v", err)
	}

	children := manager.ListChildren("session-2", "round-2")
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}
	if children[0].ChildID != "child-1" {
		t.Fatalf("expected children to be sorted by index, got %s", children[0].ChildID)
	}
}
