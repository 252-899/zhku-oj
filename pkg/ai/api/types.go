package api

import (
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	WebSearchAgentType     = "websearchagent"
	OJQuestionGenModuelType = "oj_question_gen"
)

type TaskKind string

const (
	TaskKindAssistant TaskKind = "assistant"
	TaskKindModule    TaskKind = "module"
)

type EventType string

const (
	EventMeta             EventType = "meta"
	EventState            EventType = "state"
	EventError            EventType = "error"
	EventDone             EventType = "done"
	EventThinking         EventType = "thinking"
	EventText             EventType = "text"
	EventReference        EventType = "reference"
	EventToolCall         EventType = "tool_call"
	EventToolResult       EventType = "tool_result"
	EventRecommend        EventType = "recommend"
	EventKnowledgeReady   EventType = "knowledge_ready"
	EventPlanReady        EventType = "plan_ready"
	EventQuestionsReady   EventType = "questions_ready"
	EventPostprocessReady EventType = "postprocess_ready"

	// QuestionGen Module 专用事件
	EventStageStarted          EventType = "stage_started"
	EventStageComplete         EventType = "stage_complete"
	EventQuestionGenerated     EventType = "question_generated"
	EventStandardCodeGenerated EventType = "standard_code_generated"
	EventTestCaseGenerated    EventType = "testcase_generated"
	EventVerificationCompleted EventType = "verification_completed"
	EventWaitingConfirmation  EventType = "waiting_confirmation"
	EventConfirmed            EventType = "confirmed"
)

type ReferenceItem struct {
	Title   string `json:"title,omitempty" bson:"title,omitempty"`
	URL     string `json:"url,omitempty" bson:"url,omitempty"`
	Snippet string `json:"snippet,omitempty" bson:"snippet,omitempty"`
}

type RecommendItem struct {
	Title   string `json:"title,omitempty" bson:"title,omitempty"`
	Content string `json:"content,omitempty" bson:"content,omitempty"`
}

type ToolTrace struct {
	Name   string `json:"name,omitempty" bson:"name,omitempty"`
	Stage  string `json:"stage,omitempty" bson:"stage,omitempty"`
	Status string `json:"status,omitempty" bson:"status,omitempty"`
	Detail string `json:"detail,omitempty" bson:"detail,omitempty"`
	Input  any    `json:"input,omitempty" bson:"input,omitempty"`
	Output any    `json:"output,omitempty" bson:"output,omitempty"`
}

type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	SessionID string         `json:"session_id,omitempty"`
	RoundID   string         `json:"round_id,omitempty"`
	Message   string         `json:"message"`
	TaskKind  string         `json:"task_kind,omitempty"`
	AgentType string         `json:"agent_type,omitempty"`
	ModuleType string        `json:"module_type,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	UserID    string         `json:"-"`
}

func (r *ChatRequest) Normalize() {
	if r == nil {
		return
	}

	r.SessionID = strings.TrimSpace(r.SessionID)
	r.RoundID = strings.TrimSpace(r.RoundID)
	r.Message = strings.TrimSpace(r.Message)
	r.TaskKind = strings.TrimSpace(r.TaskKind)
	r.AgentType = strings.TrimSpace(r.AgentType)
	r.ModuleType = strings.TrimSpace(r.ModuleType)
	r.UserID = strings.TrimSpace(r.UserID)

	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
}

type StreamEvent struct {
	Type         EventType `json:"type"`
	SessionID    string    `json:"session_id,omitempty"`
	RoundID      string    `json:"round_id,omitempty"`
	AgentType    string    `json:"agent_type,omitempty"`
	Content      string    `json:"content,omitempty"`
	Data         any       `json:"data,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

func NewStreamEvent(eventType EventType, req *ChatRequest, content string, data any) *StreamEvent {
	event := &StreamEvent{
		Type:      eventType,
		Content:   content,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	if req != nil {
		event.SessionID = req.SessionID
		event.RoundID = req.RoundID
		event.AgentType = req.AgentType
	}
	return event
}

type ChatResponse struct {
	SessionID string          `json:"session_id"`
	RoundID   string          `json:"round_id,omitempty"`
	AgentType string          `json:"agent_type"`
	Answer    string          `json:"answer,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Reference []ReferenceItem `json:"reference,omitempty"`
	Recommend []RecommendItem `json:"recommend,omitempty"`
	Tools     []ToolTrace     `json:"tools,omitempty"`
	Status    string          `json:"status"`
}

type RuntimeRequest struct {
	SessionID     string         `json:"session_id,omitempty"`
	RoundID       string         `json:"round_id,omitempty"`
	UserID        string         `json:"-"`
	TaskKind      TaskKind       `json:"task_kind,omitempty"`
	TaskType      string         `json:"task_type,omitempty"`
	AssistantType string         `json:"assistant_type,omitempty"`
	ModuleType    string         `json:"module_type,omitempty"`
	Message       string         `json:"message,omitempty"`
	Input         map[string]any `json:"input,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (r *RuntimeRequest) Normalize() {
	if r == nil {
		return
	}

	r.SessionID = strings.TrimSpace(r.SessionID)
	r.RoundID = strings.TrimSpace(r.RoundID)
	r.UserID = strings.TrimSpace(r.UserID)
	r.TaskType = normalizeType(r.TaskType)
	r.AssistantType = normalizeType(r.AssistantType)
	r.ModuleType = normalizeType(r.ModuleType)
	r.Message = strings.TrimSpace(r.Message)
	r.TaskKind = TaskKind(strings.ToLower(strings.TrimSpace(string(r.TaskKind))))

	if r.TaskKind == "" {
		switch {
		case r.AssistantType != "":
			r.TaskKind = TaskKindAssistant
		case r.ModuleType != "":
			r.TaskKind = TaskKindModule
		}
	}

	if r.TaskType == "" {
		switch r.TaskKind {
		case TaskKindAssistant:
			r.TaskType = r.AssistantType
		case TaskKindModule:
			r.TaskType = r.ModuleType
		}
	}

	if r.SessionID == "" {
		r.SessionID = primitive.NewObjectID().Hex()
	}
	if r.RoundID == "" {
		r.RoundID = primitive.NewObjectID().Hex()
	}
	if r.Input == nil {
		r.Input = map[string]any{}
	}
	if r.Metadata == nil {
		r.Metadata = map[string]any{}
	}
}

type RuntimeStreamEvent struct {
	Type         EventType `json:"type"`
	SessionID    string    `json:"session_id,omitempty"`
	RoundID      string    `json:"round_id,omitempty"`
	TaskType     string    `json:"task_type,omitempty"`
	Stage        string    `json:"stage,omitempty"`
	Data         any       `json:"data,omitempty"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    int64     `json:"timestamp"`
}

func NewRuntimeStreamEvent(eventType EventType, req *RuntimeRequest, stage string, data any) *RuntimeStreamEvent {
	event := &RuntimeStreamEvent{
		Type:      eventType,
		Stage:     strings.TrimSpace(stage),
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	if req != nil {
		event.SessionID = req.SessionID
		event.RoundID = req.RoundID
		event.TaskType = req.TaskType
	}
	return event
}

func normalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
