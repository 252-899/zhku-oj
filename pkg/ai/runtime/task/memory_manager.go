package task

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/runtime/state"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type rootEntry struct {
	root     *state.RootTaskRuntime
	children map[string]*state.ChildTaskRuntime
}

type MemoryTaskManager struct {
	mu    sync.RWMutex
	roots map[string]*rootEntry
}

func NewMemoryTaskManager() *MemoryTaskManager {
	return &MemoryTaskManager{
		roots: make(map[string]*rootEntry),
	}
}

func (m *MemoryTaskManager) RegisterRoot(root *state.RootTaskRuntime) error {
	if root == nil {
		return aiapi.ErrInvalidRuntimeRequest
	}
	root.SessionID = strings.TrimSpace(root.SessionID)
	root.RoundID = strings.TrimSpace(root.RoundID)
	root.TaskType = strings.TrimSpace(root.TaskType)
	root.UserID = strings.TrimSpace(root.UserID)
	if root.SessionID == "" || root.RoundID == "" || root.TaskType == "" {
		return aiapi.ErrInvalidRuntimeRequest
	}

	now := time.Now()
	if root.StartedAt.IsZero() {
		root.StartedAt = now
	}
	root.UpdatedAt = now
	if root.Status == "" {
		root.Status = state.TaskStatusRunning
	}
	if root.Metadata == nil {
		root.Metadata = map[string]any{}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.roots[root.SessionID]; exists {
		return aiapi.ErrTaskAlreadyRunning
	}
	m.roots[root.SessionID] = &rootEntry{
		root:     root.Clone(),
		children: make(map[string]*state.ChildTaskRuntime),
	}
	m.roots[root.SessionID].root.Cancel = root.Cancel
	return nil
}

func (m *MemoryTaskManager) GetRoot(sessionID string) (*state.RootTaskRuntime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok {
		return nil, false
	}
	return entry.root.Clone(), true
}

func (m *MemoryTaskManager) GetRound(sessionID, roundID string) (*state.RootTaskRuntime, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok || entry.root.RoundID != strings.TrimSpace(roundID) {
		return nil, false
	}
	return entry.root.Clone(), true
}

func (m *MemoryTaskManager) HasRunning(sessionID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.roots[strings.TrimSpace(sessionID)]
	return ok
}

func (m *MemoryTaskManager) UpdateRootState(sessionID, roundID, stage string, status state.TaskStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok || entry.root.RoundID != strings.TrimSpace(roundID) {
		return aiapi.ErrTaskNotFound
	}
	entry.root.Stage = strings.TrimSpace(stage)
	if status != "" {
		entry.root.Status = status
	}
	entry.root.UpdatedAt = time.Now()
	return nil
}

func (m *MemoryTaskManager) AddChild(sessionID, roundID string, child *state.ChildTaskRuntime) error {
	return m.upsertChild(sessionID, roundID, child, false)
}

func (m *MemoryTaskManager) UpdateChild(sessionID, roundID string, child *state.ChildTaskRuntime) error {
	return m.upsertChild(sessionID, roundID, child, true)
}

func (m *MemoryTaskManager) upsertChild(sessionID, roundID string, child *state.ChildTaskRuntime, allowMissing bool) error {
	if child == nil {
		return aiapi.ErrInvalidRuntimeRequest
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok || entry.root.RoundID != strings.TrimSpace(roundID) {
		return aiapi.ErrTaskNotFound
	}

	childID := strings.TrimSpace(child.ChildID)
	if childID == "" {
		childID = primitive.NewObjectID().Hex()
		child.ChildID = childID
	}
	if !allowMissing {
		if _, exists := entry.children[childID]; exists {
			allowMissing = true
		}
	}
	if allowMissing {
		if _, exists := entry.children[childID]; !exists && len(entry.children) > 0 {
			// continue as upsert for workflow convenience.
		}
	}

	now := time.Now()
	if child.StartedAt.IsZero() {
		child.StartedAt = now
	}
	child.UpdatedAt = now
	if child.ParentRoundID == "" {
		child.ParentRoundID = entry.root.RoundID
	}
	if child.Status == "" {
		child.Status = state.TaskStatusRunning
	}
	if child.Metadata == nil {
		child.Metadata = map[string]any{}
	}

	stored := child.Clone()
	stored.Cancel = child.Cancel
	entry.children[childID] = stored
	entry.root.UpdatedAt = now
	return nil
}

func (m *MemoryTaskManager) ListChildren(sessionID, roundID string) []*state.ChildTaskRuntime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok || entry.root.RoundID != strings.TrimSpace(roundID) {
		return nil
	}

	children := make([]*state.ChildTaskRuntime, 0, len(entry.children))
	for _, child := range entry.children {
		children = append(children, child.Clone())
	}
	sort.Slice(children, func(i, j int) bool {
		if children[i].Index != children[j].Index {
			return children[i].Index < children[j].Index
		}
		return children[i].StartedAt.Before(children[j].StartedAt)
	})
	return children
}

func (m *MemoryTaskManager) Stop(ctx context.Context, sessionID, roundID, userID string) error {
	_ = ctx

	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.roots[strings.TrimSpace(sessionID)]
	if !ok {
		return aiapi.ErrTaskNotFound
	}
	if rid := strings.TrimSpace(roundID); rid != "" && entry.root.RoundID != rid {
		return aiapi.ErrTaskNotFound
	}
	if uid := strings.TrimSpace(userID); uid != "" && entry.root.UserID != "" && entry.root.UserID != uid {
		return aiapi.ErrTaskNotFound
	}

	entry.root.Status = state.TaskStatusStopped
	entry.root.Stage = "stopped"
	entry.root.UpdatedAt = time.Now()

	for _, child := range entry.children {
		child.Status = state.TaskStatusStopped
		child.Stage = "stopped"
		child.UpdatedAt = entry.root.UpdatedAt
		if child.Cancel != nil {
			child.Cancel()
		}
	}
	if entry.root.Cancel != nil {
		entry.root.Cancel()
	}
	return nil
}

func (m *MemoryTaskManager) RemoveRoot(sessionID, roundID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sessionID = strings.TrimSpace(sessionID)
	entry, ok := m.roots[sessionID]
	if !ok {
		return
	}
	if rid := strings.TrimSpace(roundID); rid != "" && entry.root.RoundID != rid {
		return
	}
	delete(m.roots, sessionID)
}
