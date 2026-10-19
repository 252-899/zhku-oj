package models

import (
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AISession struct {
	ID              primitive.ObjectID    `bson:"_id,omitempty" json:"id,omitempty"`
	SessionID       string                `bson:"session_id" json:"session_id"`
	RoundID         string                `bson:"round_id,omitempty" json:"round_id,omitempty"`
	UserID          string                `bson:"user_id,omitempty" json:"user_id,omitempty"`
	AgentType       string                `bson:"agent_type,omitempty" json:"agent_type,omitempty"`
	Question        string                `bson:"question,omitempty" json:"question,omitempty"`
	Answer          string                `bson:"answer,omitempty" json:"answer,omitempty"`
	Thinking        string                `bson:"thinking,omitempty" json:"thinking,omitempty"`
	Reference       []aiapi.ReferenceItem `bson:"reference,omitempty" json:"reference,omitempty"`
	Recommend       []aiapi.RecommendItem `bson:"recommend,omitempty" json:"recommend,omitempty"`
	Tools           []aiapi.ToolTrace     `bson:"tools,omitempty" json:"tools,omitempty"`
	FirstResponseMS int64                 `bson:"first_response_ms,omitempty" json:"first_response_ms,omitempty"`
	TotalResponseMS int64                 `bson:"total_response_ms,omitempty" json:"total_response_ms,omitempty"`
	Status          string                `bson:"status,omitempty" json:"status,omitempty"`
	ErrorMessage    string                `bson:"error_message,omitempty" json:"error_message,omitempty"`
	CreatedAt       time.Time             `bson:"created_at" json:"created_at"`
	UpdatedAt       time.Time             `bson:"updated_at" json:"updated_at"`
}
