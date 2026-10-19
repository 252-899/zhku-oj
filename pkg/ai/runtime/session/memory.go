package session

import (
	"context"
	"sync"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/models"
)

type MemorySessionService struct {
	mu    sync.RWMutex
	store map[string][]*models.AISession
}

func NewMemorySessionService() *MemorySessionService {
	return &MemorySessionService{store: make(map[string][]*models.AISession)}
}

func (s *MemorySessionService) CreateIfNeeded(ctx context.Context, req *aiapi.RuntimeRequest) error {
	return nil
}

func (s *MemorySessionService) SaveQuestion(ctx context.Context, req *aiapi.RuntimeRequest) error {
	if s == nil || req == nil {
		return nil
	}
	now := time.Now()
	record := &models.AISession{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		AgentType: req.TaskType,
		Question:  req.Message,
		Status:    "running",
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[req.SessionID] = append(s.store[req.SessionID], record)
	return nil
}

func (s *MemorySessionService) SaveResult(ctx context.Context, input *SaveResultInput) error {
	if s == nil || input == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.store[input.SessionID]
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record == nil || record.RoundID != input.RoundID {
			continue
		}
		record.Answer = input.Answer
		record.Thinking = input.Thinking
		record.Reference = append([]aiapi.ReferenceItem(nil), input.Reference...)
		record.Recommend = append([]aiapi.RecommendItem(nil), input.Recommend...)
		record.Tools = append([]aiapi.ToolTrace(nil), input.Tools...)
		record.FirstResponseMS = input.FirstResponseMS
		record.TotalResponseMS = input.TotalResponseMS
		record.Status = input.Status
		record.ErrorMessage = input.ErrorMessage
		record.UpdatedAt = time.Now()
		return nil
	}
	return nil
}

func (s *MemorySessionService) LoadRecent(ctx context.Context, sessionID, userID string, limit int) ([]*SessionSummary, error) {
	if s == nil {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	records := s.store[sessionID]
	if limit <= 0 || limit > len(records) {
		limit = len(records)
	}
	start := len(records) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*SessionSummary, 0, limit)
	for i := start; i < len(records); i++ {
		record := records[i]
		if record == nil || record.Status == "running" {
			continue
		}
		if userID != "" && record.UserID != userID {
			continue
		}
		result = append(result, &SessionSummary{
			SessionID: record.SessionID,
			RoundID:   record.RoundID,
			UserID:    record.UserID,
			TaskType:  record.AgentType,
			Question:  record.Question,
			Answer:    record.Answer,
			CreatedAt: record.CreatedAt,
		})
	}
	return result, nil
}
