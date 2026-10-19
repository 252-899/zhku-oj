package session

import (
	"context"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

type SessionSummary struct {
	SessionID string
	RoundID   string
	UserID    string
	TaskType  string
	Question  string
	Answer    string
	CreatedAt time.Time
}

type SaveResultInput struct {
	SessionID       string
	RoundID         string
	UserID          string
	TaskType        string
	Answer          string
	Thinking        string
	Reference       []aiapi.ReferenceItem
	Recommend       []aiapi.RecommendItem
	Tools           []aiapi.ToolTrace
	FirstResponseMS int64
	TotalResponseMS int64
	Status          string
	ErrorMessage    string
}

type SessionService interface {
	CreateIfNeeded(ctx context.Context, req *aiapi.RuntimeRequest) error
	SaveQuestion(ctx context.Context, req *aiapi.RuntimeRequest) error
	SaveResult(ctx context.Context, input *SaveResultInput) error
	LoadRecent(ctx context.Context, sessionID, userID string, limit int) ([]*SessionSummary, error)
}
