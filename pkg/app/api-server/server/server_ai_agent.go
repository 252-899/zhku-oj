package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	agentbridge "zk-code-arena-server/pkg/ai/compat/agentbridge"
	"zk-code-arena-server/pkg/ai/modules/ojquestiongen"
	doc "zk-code-arena-server/pkg/app/api-server/server/Doc"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	_ = doc.AIAgentChatRequestDoc{}
	_ = doc.AIAgentStreamChatRequestDoc{}
	_ = doc.OJQuestionGenStreamRequestDoc{}
)

func (s *Server) RegisterAIAgent(g *gin.RouterGroup) {
	agentGroup := g.Group("/ai/agent").Use(middleware.JWTMiddleware())
	{
		agentGroup.POST("/chat", s.AgentChat)
		agentGroup.POST("/chat/stream", s.AgentStreamChat)
	}

	ojQuestionGenGroup := g.Group("/ai/oj-question-gen").Use(middleware.JWTMiddleware())
	{
		ojQuestionGenGroup.POST("/stream", s.OJQuestionGenStream)
		ojQuestionGenGroup.POST("/save", s.SaveOJQuestionGenDraft)
	}
}

// AgentChat godoc
// @Summary      AI Agent 同步对话
// @Description  Assistant 同步对话入口。当前同步接口走 AssistantGateway，请使用 agent_type，例如 websearchagent；module 任务请使用 /ai/agent/chat/stream。
// @Description  请求要求：message 必填；agent_type 必填；session_id 和 round_id 可不传，后端会自动生成。
// @Tags         AI Agent
// @Accept       json
// @Produce      json
// @Param        request body doc.AIAgentChatRequestDoc true "Assistant 同步对话请求"
// @Success      200 {object} utils.Response{data=aiapi.ChatResponse} "对话结果"
// @Failure      400 {object} utils.Response "请求参数错误"
// @Failure      401 {object} utils.Response "未认证用户"
// @Failure      409 {object} utils.Response "会话已有运行中的任务"
// @Failure      503 {object} utils.Response "Agent 未注册或未配置"
// @Failure      500 {object} utils.Response "服务器内部错误"
// @Security     BearerAuth
// @Router       /ai/agent/chat [post]
func (s *Server) AgentChat(c *gin.Context) {
	req, ok := s.bindAgentChatRequest(c)
	if !ok {
		return
	}

	resp, err := s.chatViaAssistant(c.Request.Context(), req)
	if err != nil {
		s.writeAgentHTTPError(c, err)
		return
	}

	utils.SuccessResponse(c, resp)
}

// AgentStreamChat godoc
// @Summary      AI Agent 流式对话
// @Description  Assistant/Module 统一流式入口，响应为 Server-Sent Events，每条消息格式为 event: message + data: StreamEvent JSON。
// @Description  Assistant 调用：传 agent_type，例如 websearchagent；Module 调用：传 task_kind=module 和 module_type。
// @Description  oj_question_gen 调用：module_type 固定为 oj_question_gen，业务参数放在 metadata.input 中；当前支持 knowledge_points、difficulty、count、testcase_count、standard_language、additional_requirements。
// @Description  oj_question_gen 请求示例：{"message":"生成 2 道数组相关 OJ 编程题","task_kind":"module","module_type":"oj_question_gen","metadata":{"input":{"knowledge_points":["数组","循环"],"difficulty":"easy","count":2,"testcase_count":3,"standard_language":"python","additional_requirements":"题目场景贴近校园生活"}}}
// @Tags         AI Agent
// @Accept       json
// @Produce      text/event-stream
// @Param        request body doc.AIAgentStreamChatRequestDoc true "Assistant/Module 流式对话请求；oj_question_gen 参数放在 metadata.input"
// @Success      200 {string} string "SSE event stream"
// @Failure      400 {object} utils.Response "请求参数错误"
// @Failure      401 {object} utils.Response "未认证用户"
// @Failure      409 {object} utils.Response "会话已有运行中的任务"
// @Failure      503 {object} utils.Response "Agent 或 Module 未注册/未配置"
// @Failure      500 {object} utils.Response "服务器内部错误"
// @Security     BearerAuth
// @Router       /ai/agent/chat/stream [post]
func (s *Server) AgentStreamChat(c *gin.Context) {
	req, ok := s.bindAgentChatRequest(c)
	if !ok {
		return
	}

	s.prepareSSE(c)

	// 根据 TaskKind 路由到不同的 Gateway
	if req.TaskKind == "module" || req.ModuleType != "" {
		s.streamModuleViaGateway(c.Request.Context(), c, req)
	} else {
		s.streamAssistantViaGateway(c.Request.Context(), c, req)
	}
}

// OJQuestionGenStream godoc
// @Summary      OJ 题目生成流式接口
// @Description  OJ 题目生成正式业务入口。前端直接提交题目生成参数；后端内部转换为 oj_question_gen RuntimeRequest 并通过 ModuleGateway 执行。
// @Description  响应为 Server-Sent Events，每条消息格式为 event: message + data: StreamEvent JSON。
// @Description  请求示例：{"message":"生成 2 道数组相关 OJ 编程题","knowledge_points":["数组","循环"],"difficulty":"easy","count":2,"testcase_count":3,"standard_language":"python","additional_requirements":"题目场景贴近校园生活"}
// @Tags         AI OJ题目生成
// @Accept       json
// @Produce      text/event-stream
// @Param        request body doc.OJQuestionGenStreamRequestDoc true "OJ 题目生成请求"
// @Success      200 {string} string "SSE event stream"
// @Failure      400 {object} utils.Response "请求参数错误"
// @Failure      401 {object} utils.Response "未认证用户"
// @Failure      409 {object} utils.Response "会话已有运行中的任务"
// @Failure      503 {object} utils.Response "模块未注册或未配置"
// @Failure      500 {object} utils.Response "服务器内部错误"
// @Security     BearerAuth
// @Router       /ai/oj-question-gen/stream [post]
func (s *Server) OJQuestionGenStream(c *gin.Context) {
	var req doc.OJQuestionGenStreamRequestDoc
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	if len(req.KnowledgePoints) == 0 {
		utils.BadRequestResponse(c, "knowledge_points 不能为空")
		return
	}

	runtimeReq := &aiapi.RuntimeRequest{
		SessionID:  strings.TrimSpace(req.SessionID),
		RoundID:    strings.TrimSpace(req.RoundID),
		TaskKind:   aiapi.TaskKindModule,
		TaskType:   ojquestiongen.CanonicalModuleType,
		ModuleType: ojquestiongen.CanonicalModuleType,
		Message:    strings.TrimSpace(req.Message),
		Input: map[string]any{
			"knowledge_points":        req.KnowledgePoints,
			"difficulty":              strings.TrimSpace(req.Difficulty),
			"count":                   req.Count,
			"testcase_count":          req.TestcaseCount,
			"standard_language":       strings.TrimSpace(req.StandardLanguage),
			"additional_requirements": strings.TrimSpace(req.AdditionalRequirements),
		},
	}
	if runtimeReq.Message == "" {
		runtimeReq.Message = "生成 OJ 编程题"
	}
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			runtimeReq.UserID = strings.TrimSpace(userIDStr)
		}
	}
	runtimeReq.Normalize()

	s.prepareSSE(c)
	s.streamRuntimeModuleViaGateway(c.Request.Context(), c, runtimeReq)
}

// SaveOJQuestionGenDraft godoc
// @Summary      保存 AI 生成的 OJ 题目
// @Description  教师从 AI 生成草稿中逐题选择是否公开和挂载课程，后端将选中题目写入正式题库。
// @Tags         AI OJ题目生成
// @Accept       json
// @Produce      json
// @Param        request body models.AIGeneratedProblemSaveRequest true "AI生成题目保存请求"
// @Success      200 {object} utils.Response{data=models.AIGeneratedProblemSaveResponse} "保存结果"
// @Failure      400 {object} utils.Response "请求参数错误"
// @Failure      401 {object} utils.Response "未认证用户"
// @Failure      403 {object} utils.Response "权限不足"
// @Failure      500 {object} utils.Response "服务器内部错误"
// @Security     BearerAuth
// @Router       /ai/oj-question-gen/save [post]
func (s *Server) SaveOJQuestionGenDraft(c *gin.Context) {
	var req models.AIGeneratedProblemSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	roleValue, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	roleStr, ok := roleValue.(string)
	if !ok {
		utils.UnauthorizedResponse(c, "用户角色信息异常")
		return
	}
	role := models.UserRole(roleStr)
	if role != models.RoleAdmin && role != models.RoleTeacher {
		utils.ForbiddenResponse(c, "只有教师或管理员可以保存AI生成题目")
		return
	}

	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	userIDStr, ok := userIDValue.(string)
	if !ok {
		utils.UnauthorizedResponse(c, "用户ID信息异常")
		return
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}
	if s == nil || s.svc == nil || s.svc.AIGeneratedProblemDraftService == nil {
		utils.InternalServerErrorResponse(c, "AI生成题目保存服务未初始化")
		return
	}

	response, err := s.svc.AIGeneratedProblemDraftService.SaveDraftProblems(c.Request.Context(), &req, userID, role)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, response)
}

func (s *Server) streamAssistantViaGateway(ctx context.Context, c *gin.Context, req *aiapi.ChatRequest) {
	stream, err := s.assistantStream(ctx, req)
	if err != nil {
		s.writeSSEError(c, req, statusFromAgentError(err), assistantErrorMessage(err))
		return
	}
	for event := range stream {
		if event == nil {
			continue
		}
		if err := s.writeSSEMessage(c, event); err != nil {
			return
		}
	}
}

func (s *Server) streamRuntimeAssistantViaGateway(ctx context.Context, c *gin.Context, runtimeReq *aiapi.RuntimeRequest) {
	if runtimeReq == nil {
		s.writeRuntimeSSEError(c, &aiapi.RuntimeRequest{}, http.StatusBadRequest, "invalid runtime request")
		return
	}
	if s == nil || s.svc == nil || s.svc.AssistantGateway == nil {
		s.writeRuntimeSSEError(c, runtimeReq, http.StatusServiceUnavailable, "assistant gateway not available")
		return
	}

	runtimeStream, err := s.svc.AssistantGateway.Stream(ctx, runtimeReq)
	if err != nil {
		s.writeRuntimeSSEError(c, runtimeReq, statusFromAgentError(err), assistantErrorMessage(err))
		return
	}

	for event := range runtimeStream {
		if event == nil {
			continue
		}
		s.attachOJQuestionGenDraft(ctx, runtimeReq, event)
		streamEvent := agentbridge.ToStreamEvent(event)
		if streamEvent == nil {
			continue
		}
		streamEvent.SessionID = runtimeReq.SessionID
		streamEvent.RoundID = runtimeReq.RoundID
		if err := s.writeSSEMessage(c, streamEvent); err != nil {
			return
		}
	}
}

func (s *Server) streamModuleViaGateway(ctx context.Context, c *gin.Context, req *aiapi.ChatRequest) {
	if s == nil || s.svc == nil || s.svc.ModuleGateway == nil {
		s.writeSSEError(c, req, http.StatusServiceUnavailable, "module gateway not available")
		return
	}

	runtimeReq := agentbridge.FromChatRequest(req)
	runtimeReq.Normalize()
	req.SessionID = runtimeReq.SessionID
	req.RoundID = runtimeReq.RoundID

	s.streamRuntimeModuleViaGateway(ctx, c, runtimeReq)
}

func (s *Server) streamRuntimeModuleViaGateway(ctx context.Context, c *gin.Context, runtimeReq *aiapi.RuntimeRequest) {
	if runtimeReq == nil {
		s.writeRuntimeSSEError(c, &aiapi.RuntimeRequest{}, http.StatusBadRequest, "invalid runtime request")
		return
	}
	if s == nil || s.svc == nil || s.svc.ModuleGateway == nil {
		s.writeRuntimeSSEError(c, runtimeReq, http.StatusServiceUnavailable, "module gateway not available")
		return
	}

	runtimeStream, err := s.svc.ModuleGateway.Stream(ctx, runtimeReq)
	if err != nil {
		s.writeRuntimeSSEError(c, runtimeReq, statusFromAgentError(err), assistantErrorMessage(err))
		return
	}

	for event := range runtimeStream {
		if event == nil {
			continue
		}
		s.attachOJQuestionGenDraft(ctx, runtimeReq, event)
		streamEvent := agentbridge.ToStreamEvent(event)
		if streamEvent == nil {
			continue
		}
		streamEvent.SessionID = runtimeReq.SessionID
		streamEvent.RoundID = runtimeReq.RoundID
		if err := s.writeSSEMessage(c, streamEvent); err != nil {
			return
		}
	}
}

func (s *Server) attachOJQuestionGenDraft(ctx context.Context, runtimeReq *aiapi.RuntimeRequest, event *aiapi.RuntimeStreamEvent) {
	if s == nil || s.svc == nil || s.svc.AIGeneratedProblemDraftService == nil || runtimeReq == nil || event == nil {
		return
	}
	if runtimeReq.TaskType != ojquestiongen.CanonicalModuleType || event.Type != aiapi.EventWaitingConfirmation {
		return
	}
	data, ok := event.Data.(map[string]any)
	if !ok {
		return
	}
	if _, exists := data["draft_id"]; exists {
		return
	}
	// 只在题目生成完成、等待教师确认时创建草稿；SSE 中原 questions 不变，额外挂上 draft_id 供前端后续保存。
	problems := extractGeneratedProblemsFromEventData(data)
	if len(problems) == 0 {
		return
	}
	userID, err := primitive.ObjectIDFromHex(strings.TrimSpace(runtimeReq.UserID))
	if err != nil {
		data["draft_error"] = "无效的用户ID，无法创建AI生成题目草稿"
		return
	}

	// 生成接口保持原有 questions 响应不变，只在同一层补充 draft_id 供保存接口定位草稿。
	draft, err := s.svc.AIGeneratedProblemDraftService.CreateDraftFromGeneratedProblems(
		detachDraftPersistenceContext(ctx),
		runtimeReq.SessionID,
		runtimeReq.RoundID,
		userID,
		problems,
	)
	if err != nil {
		data["draft_error"] = err.Error()
		return
	}
	data["draft_id"] = draft.ID.Hex()
	data["expires_at"] = draft.ExpiresAt
}

// detachDraftPersistenceContext 为草稿落库创建独立上下文，避免浏览器提前断开 SSE 时把结果暂存一起取消掉。
func detachDraftPersistenceContext(parent context.Context) context.Context {
	const draftPersistenceTimeout = 5 * time.Second
	if parent == nil {
		detachedCtx, _ := context.WithTimeout(context.Background(), draftPersistenceTimeout)
		return detachedCtx
	}

	// Go 1.21+ 的 WithoutCancel 可以保留链路中的值，同时去掉请求取消信号。
	detachedCtx, _ := context.WithTimeout(context.WithoutCancel(parent), draftPersistenceTimeout)
	return detachedCtx
}

func extractGeneratedProblemsFromEventData(data map[string]any) []*ojquestiongen.GeneratedProblem {
	if data == nil {
		return nil
	}
	rawQuestions, ok := data["questions"]
	if !ok || rawQuestions == nil {
		return nil
	}
	if problems, ok := rawQuestions.([]*ojquestiongen.GeneratedProblem); ok {
		return problems
	}
	payload, err := json.Marshal(rawQuestions)
	if err != nil {
		return nil
	}
	var problems []*ojquestiongen.GeneratedProblem
	if err := json.Unmarshal(payload, &problems); err != nil {
		return nil
	}
	return problems
}

func (s *Server) bindAgentChatRequest(c *gin.Context) (*aiapi.ChatRequest, bool) {
	var req aiapi.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return nil, false
	}
	req.AgentType = strings.TrimSpace(req.AgentType)
	req.ModuleType = strings.TrimSpace(req.ModuleType)
	req.Message = strings.TrimSpace(req.Message)

	if req.Message == "" {
		utils.BadRequestResponse(c, "message 不能为空")
		return nil, false
	}
	if req.AgentType == "" && req.ModuleType == "" {
		utils.BadRequestResponse(c, "agent_type 或 module_type 不能同时为空")
		return nil, false
	}
	req.Normalize()

	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			req.UserID = strings.TrimSpace(userIDStr)
		}
	}

	return &req, true
}

func (s *Server) writeAgentHTTPError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, aiapi.ErrAssistantNotRegistered):
		utils.ServiceUnavailableResponse(c, "agent not registered")
	case errors.Is(err, aiapi.ErrAssistantNotConfigured):
		utils.ServiceUnavailableResponse(c, "agent not configured")
	case errors.Is(err, aiapi.ErrTaskAlreadyRunning):
		utils.ErrorResponse(c, http.StatusConflict, "session already has a running task")
	case errors.Is(err, aiapi.ErrTaskNotFound):
		utils.NotFoundResponse(c, "task not found")
	default:
		utils.InternalServerErrorResponse(c, err.Error())
	}
}

func (s *Server) prepareSSE(c *gin.Context) {
	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
}

func (s *Server) writeSSEError(c *gin.Context, req *aiapi.ChatRequest, httpStatus int, message string) {
	if req == nil {
		req = &aiapi.ChatRequest{}
	}

	_ = s.writeSSEMessage(c, &aiapi.StreamEvent{
		Type:         aiapi.EventError,
		SessionID:    req.SessionID,
		RoundID:      req.RoundID,
		AgentType:    req.AgentType,
		ErrorMessage: message,
		Timestamp:    time.Now().UnixMilli(),
		Data: map[string]any{
			"http_status": httpStatus,
		},
	})
	_ = s.writeSSEMessage(c, &aiapi.StreamEvent{
		Type:      aiapi.EventDone,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		AgentType: req.AgentType,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]any{
			"status": "failed",
		},
	})
}

func (s *Server) writeRuntimeSSEError(c *gin.Context, req *aiapi.RuntimeRequest, httpStatus int, message string) {
	if req == nil {
		req = &aiapi.RuntimeRequest{}
	}

	_ = s.writeSSEMessage(c, &aiapi.StreamEvent{
		Type:         aiapi.EventError,
		SessionID:    req.SessionID,
		RoundID:      req.RoundID,
		AgentType:    req.TaskType,
		ErrorMessage: message,
		Timestamp:    time.Now().UnixMilli(),
		Data: map[string]any{
			"http_status": httpStatus,
		},
	})
	_ = s.writeSSEMessage(c, &aiapi.StreamEvent{
		Type:      aiapi.EventDone,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		AgentType: req.TaskType,
		Timestamp: time.Now().UnixMilli(),
		Data: map[string]any{
			"status": "failed",
		},
	})
}

func (s *Server) writeSSEMessage(c *gin.Context, event *aiapi.StreamEvent) error {
	if event == nil {
		return nil
	}
	if event.Timestamp == 0 {
		event.Timestamp = time.Now().UnixMilli()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", payload); err != nil {
		return err
	}

	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func statusFromAgentError(err error) int {
	switch {
	case errors.Is(err, aiapi.ErrAssistantNotRegistered):
		return http.StatusServiceUnavailable
	case errors.Is(err, aiapi.ErrAssistantNotConfigured):
		return http.StatusServiceUnavailable
	case errors.Is(err, aiapi.ErrTaskAlreadyRunning):
		return http.StatusConflict
	case errors.Is(err, aiapi.ErrTaskNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func (s *Server) chatViaAssistant(ctx context.Context, req *aiapi.ChatRequest) (*aiapi.ChatResponse, error) {
	stream, err := s.assistantStream(ctx, req)
	if err != nil {
		return nil, err
	}

	response := &aiapi.ChatResponse{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		AgentType: aiapi.WebSearchAgentType,
		Status:    "completed",
	}

	var streamErr error
	for event := range stream {
		if event == nil {
			continue
		}
		switch event.Type {
		case aiapi.EventThinking:
			response.Thinking += event.Content
		case aiapi.EventText:
			response.Answer += event.Content
		case aiapi.EventReference:
			if items, ok := event.Data.([]aiapi.ReferenceItem); ok {
				response.Reference = append(response.Reference, items...)
			}
		case aiapi.EventToolCall, aiapi.EventToolResult:
			if trace, ok := event.Data.(map[string]any); ok {
				response.Tools = append(response.Tools, aiapi.ToolTrace{
					Name:   stringValue(trace["name"]),
					Stage:  string(event.Type),
					Status: stringValue(trace["status"]),
					Detail: stringValue(trace["error_message"]),
					Input:  trace,
					Output: trace,
				})
			}
		case aiapi.EventDone:
			if payload, ok := event.Data.(map[string]any); ok {
				if status := stringValue(payload["status"]); status != "" {
					response.Status = status
				}
			}
		case aiapi.EventError:
			response.Status = "failed"
			if event.ErrorMessage != "" {
				streamErr = errors.New(event.ErrorMessage)
			}
		}
	}

	return response, streamErr
}

func (s *Server) assistantStream(ctx context.Context, req *aiapi.ChatRequest) (<-chan *aiapi.StreamEvent, error) {
	if s == nil || s.svc == nil || s.svc.AssistantGateway == nil {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	runtimeReq := agentbridge.FromChatRequest(req)
	runtimeReq.Normalize()
	req.SessionID = runtimeReq.SessionID
	req.RoundID = runtimeReq.RoundID
	//获取得到对应的服务
	runtimeStream, err := s.svc.AssistantGateway.Stream(ctx, runtimeReq)
	if err != nil {
		return nil, err
	}

	out := make(chan *aiapi.StreamEvent, 16)
	go func() {
		defer close(out)
		for event := range runtimeStream {
			agentEvent := agentbridge.ToStreamEvent(event)
			if agentEvent == nil {
				continue
			}
			agentEvent.SessionID = req.SessionID
			agentEvent.RoundID = req.RoundID
			agentEvent.AgentType = aiapi.WebSearchAgentType
			select {
			case <-ctx.Done():
				return
			case out <- agentEvent:
			}
		}
	}()
	return out, nil
}

func assistantErrorMessage(err error) string {
	switch {
	case errors.Is(err, aiapi.ErrAssistantNotRegistered):
		return "agent not registered"
	case errors.Is(err, aiapi.ErrAssistantNotConfigured):
		return "agent not configured"
	default:
		if err == nil {
			return ""
		}
		return err.Error()
	}
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func (s *Server) streamAgentStop(ctx context.Context, sessionID, roundID, userID string) error {
	if s == nil || s.svc == nil || s.svc.AssistantGateway == nil {
		return aiapi.ErrStopNotImplemented
	}
	return s.svc.AssistantGateway.Stop(ctx, sessionID, roundID, userID)
}
