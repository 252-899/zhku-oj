package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"zk-code-arena-server/conf"
	aiapi "zk-code-arena-server/pkg/ai/api"
	aiassistants "zk-code-arena-server/pkg/ai/assistants"
	aiwebsearch "zk-code-arena-server/pkg/ai/assistants/websearch"
	"zk-code-arena-server/pkg/ai/modules"
	"zk-code-arena-server/pkg/ai/modules/ojquestiongen"
	aigateway "zk-code-arena-server/pkg/ai/runtime/gateway"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
	"zk-code-arena-server/pkg/app/api-server/service"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
)

var loadTestConfigOnce sync.Once

func TestDetachDraftPersistenceContextIgnoresParentCancel(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	detachedCtx := detachDraftPersistenceContext(parentCtx)
	select {
	case <-detachedCtx.Done():
		t.Fatalf("expected detached draft context to ignore parent cancel, got err=%v", detachedCtx.Err())
	default:
	}
}

func TestAgentChatRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	server := &Server{app: engine, svc: &service.Service{}}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"agent_type": aiapi.WebSearchAgentType,
		"message":    "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/agent/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentChatReturnsBadRequestWhenAgentTypeMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHeader := testAuthHeader(t)

	engine := gin.New()
	server := &Server{
		app: engine,
		svc: &service.Service{},
	}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"message": "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/agent/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAgentChatReturnsServiceUnavailableWhenAgentNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHeader := testAuthHeader(t)

	assistantRegistry := aiassistants.NewRegistry()
	assistantRegistry.Register(aiwebsearch.NewUnconfigured())

	engine := gin.New()
	server := &Server{
		app: engine,
		svc: &service.Service{
			AssistantGateway: aigateway.NewAssistantGateway(assistantRegistry, runtimetask.NewMemoryTaskManager()),
		},
	}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"agent_type": aiapi.WebSearchAgentType,
		"message":    "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/agent/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "agent not configured") {
		t.Fatalf("expected not configured message, body=%s", recorder.Body.String())
	}
}

func TestAgentStreamChatWritesSSEMessageEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHeader := testAuthHeader(t)

	assistantRegistry := aiassistants.NewRegistry()
	assistantRegistry.Register(&stubRuntimeAssistant{
		assistantType: "websearch",
		stream: func(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
			out := make(chan *aiapi.RuntimeStreamEvent, 3)
			go func() {
				defer close(out)
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventText, req, "answering", map[string]any{"delta": "assistant stream"})
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventDone, req, "completed", map[string]any{"status": "completed"})
			}()
			return out, nil
		},
	})

	engine := gin.New()
	server := &Server{
		app: engine,
		svc: &service.Service{
			AssistantGateway: aigateway.NewAssistantGateway(assistantRegistry, runtimetask.NewMemoryTaskManager()),
		},
	}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"agent_type": "web_search",
		"message":    "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/agent/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	if recorder.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatalf("expected X-Accel-Buffering=no")
	}

	bodyText := recorder.Body.String()
	if !strings.Contains(bodyText, "event: message") {
		t.Fatalf("expected SSE output, body=%s", bodyText)
	}
	if !strings.Contains(bodyText, "\"content\":\"assistant stream\"") {
		t.Fatalf("expected bridged assistant content, body=%s", bodyText)
	}
	if !strings.Contains(bodyText, "\"agent_type\":\"websearchagent\"") {
		t.Fatalf("expected legacy agent type in SSE payload, body=%s", bodyText)
	}
	if !strings.Contains(bodyText, "\"type\":\"done\"") {
		t.Fatalf("expected done event, body=%s", bodyText)
	}
}

func TestAgentChatUsesAssistantGatewayForWebSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHeader := testAuthHeader(t)

	assistantRegistry := aiassistants.NewRegistry()
	assistantRegistry.Register(&stubRuntimeAssistant{
		assistantType: "websearch",
		stream: func(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
			out := make(chan *aiapi.RuntimeStreamEvent, 3)
			go func() {
				defer close(out)
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventText, req, "answering", map[string]any{"delta": "assistant ok"})
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventDone, req, "completed", map[string]any{"status": "completed"})
			}()
			return out, nil
		},
	})

	engine := gin.New()
	server := &Server{
		app: engine,
		svc: &service.Service{
			AssistantGateway: aigateway.NewAssistantGateway(assistantRegistry, runtimetask.NewMemoryTaskManager()),
		},
	}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"agent_type": aiapi.WebSearchAgentType,
		"message":    "hello",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/agent/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "\"answer\":\"assistant ok\"") {
		t.Fatalf("expected assistant answer, body=%s", recorder.Body.String())
	}
}

func TestOJQuestionGenStreamUsesBusinessRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authHeader := testAuthHeader(t)

	requests := make(chan *aiapi.RuntimeRequest, 1)
	moduleRegistry := modules.NewRegistry()
	moduleRegistry.Register(&stubRuntimeModule{
		moduleType: ojquestiongen.CanonicalModuleType,
		stream: func(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
			requests <- req
			out := make(chan *aiapi.RuntimeStreamEvent, 2)
			go func() {
				defer close(out)
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventStageStarted, req, "generating", map[string]any{"stage": "generating"})
				out <- aiapi.NewRuntimeStreamEvent(aiapi.EventDone, req, "completed", map[string]any{"status": "completed"})
			}()
			return out, nil
		},
	})

	engine := gin.New()
	server := &Server{
		app: engine,
		svc: &service.Service{
			ModuleGateway: aigateway.NewModuleGateway(moduleRegistry, runtimetask.NewMemoryTaskManager()),
		},
	}
	server.RegisterAIAgent(engine.Group("/api/v1"))

	body, _ := json.Marshal(map[string]any{
		"message":                 "生成 2 道数组相关 OJ 编程题",
		"knowledge_points":        []string{"数组", "循环"},
		"difficulty":              "easy",
		"count":                   2,
		"testcase_count":          3,
		"standard_language":       "python",
		"additional_requirements": "题目场景贴近校园生活",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/oj-question-gen/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.Contains(contentType, "text/event-stream") {
		t.Fatalf("expected text/event-stream, got %q", contentType)
	}
	if !strings.Contains(recorder.Body.String(), "\"agent_type\":\"oj_question_gen\"") {
		t.Fatalf("expected module type in SSE payload, body=%s", recorder.Body.String())
	}

	var runtimeReq *aiapi.RuntimeRequest
	select {
	case runtimeReq = <-requests:
	default:
		t.Fatalf("expected module gateway request")
	}
	if runtimeReq.TaskKind != aiapi.TaskKindModule {
		t.Fatalf("expected task kind module, got %q", runtimeReq.TaskKind)
	}
	if runtimeReq.ModuleType != ojquestiongen.CanonicalModuleType {
		t.Fatalf("expected module type %q, got %q", ojquestiongen.CanonicalModuleType, runtimeReq.ModuleType)
	}
	if got := runtimeReq.Input["standard_language"]; got != "python" {
		t.Fatalf("expected standard_language python, got %#v", got)
	}
	if got := runtimeReq.Input["count"]; got != 2 {
		t.Fatalf("expected count 2, got %#v", got)
	}
	if _, ok := runtimeReq.Input["languages"]; ok {
		t.Fatalf("did not expect languages in runtime input: %#v", runtimeReq.Input)
	}
}

type stubRuntimeAssistant struct {
	assistantType string
	stream        func(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
}

func (s *stubRuntimeAssistant) Type() string {
	return s.assistantType
}

func (s *stubRuntimeAssistant) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	return s.stream(ctx, req)
}

type stubRuntimeModule struct {
	moduleType string
	stream     func(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error)
}

func (s *stubRuntimeModule) Type() string {
	return s.moduleType
}

func (s *stubRuntimeModule) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	return s.stream(ctx, req)
}

func testAuthHeader(t *testing.T) string {
	t.Helper()
	ensureTestConfig(t)
	token, err := middleware.GenerateToken("user-1", "tester", "student")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	return "Bearer " + token
}

func ensureTestConfig(t *testing.T) {
	t.Helper()

	loadTestConfigOnce.Do(func() {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd() error = %v", err)
		}

		root := filepath.Clean(filepath.Join(cwd, "../../../../"))
		if err := os.Chdir(root); err != nil {
			t.Fatalf("Chdir(%q) error = %v", root, err)
		}
		defer func() {
			if err := os.Chdir(cwd); err != nil {
				t.Fatalf("restore cwd error = %v", err)
			}
		}()

		conf.Init()
	})
}
