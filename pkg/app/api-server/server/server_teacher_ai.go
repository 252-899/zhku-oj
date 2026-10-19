// server_teacher_ai.go 提供教师侧 AI Copilot 的独立 HTTP 入口。
package server

import (
	"context"
	"net/http"
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"
	teachercopilot "zk-code-arena-server/pkg/ai/assistants/teachercopilot"
	agentbridge "zk-code-arena-server/pkg/ai/compat/agentbridge"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
)

// TeacherCopilotStreamRequest 是教师侧 Copilot 流式请求体。
type TeacherCopilotStreamRequest struct {
	SessionID string         `json:"session_id,omitempty"`
	RoundID   string         `json:"round_id,omitempty"`
	Message   string         `json:"message" binding:"required"`
	Input     map[string]any `json:"input,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// RegisterTeacherAI 注册教师侧 AI 业务入口。
func (s *Server) RegisterTeacherAI(g *gin.RouterGroup) {
	teacherGroup := g.Group("/ai/teacher").Use(middleware.JWTMiddleware())
	{
		teacherGroup.POST("/copilot/stream", s.TeacherCopilotStream)
	}
}

// TeacherCopilotStream godoc
// @Summary      教师侧 AI Copilot 流式对话
// @Description  教师侧 multi-agent 独立业务入口，不走通用 AI Gateway。响应为 Server-Sent Events。
// @Tags         AI 教师助手
// @Accept       json
// @Produce      text/event-stream
// @Param        request body TeacherCopilotStreamRequest true "教师侧 Copilot 请求"
// @Success      200 {string} string "SSE event stream"
// @Failure      400 {object} utils.Response "请求参数错误"
// @Failure      401 {object} utils.Response "未认证用户"
// @Failure      409 {object} utils.Response "会话已有运行中的任务"
// @Failure      503 {object} utils.Response "教师助手未配置"
// @Failure      500 {object} utils.Response "服务器内部错误"
// @Security     BearerAuth
// @Router       /ai/teacher/copilot/stream [post]
func (s *Server) TeacherCopilotStream(c *gin.Context) {
	req, ok := s.bindTeacherCopilotRequest(c)
	if !ok {
		return
	}

	s.prepareSSE(c)
	s.streamTeacherCopilot(c.Request.Context(), c, req)
}

// bindTeacherCopilotRequest 绑定教师侧 Copilot 请求，并转换为统一 RuntimeRequest。
func (s *Server) bindTeacherCopilotRequest(c *gin.Context) (*aiapi.RuntimeRequest, bool) {
	var req TeacherCopilotStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return nil, false
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		utils.BadRequestResponse(c, "message 不能为空")
		return nil, false
	}

	runtimeReq := &aiapi.RuntimeRequest{
		SessionID:     strings.TrimSpace(req.SessionID),
		RoundID:       strings.TrimSpace(req.RoundID),
		TaskKind:      aiapi.TaskKindAssistant,
		TaskType:      teachercopilot.AssistantType,
		AssistantType: teachercopilot.AssistantType,
		Message:       message,
		Input:         req.Input,
		Metadata:      req.Metadata,
	}
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			runtimeReq.UserID = strings.TrimSpace(userIDStr)
		}
	}
	runtimeReq.Normalize()
	return runtimeReq, true
}

// streamTeacherCopilot 调用业务 service 并把 RuntimeStreamEvent 写成前端 SSE。
func (s *Server) streamTeacherCopilot(ctx context.Context, c *gin.Context, req *aiapi.RuntimeRequest) {
	if s == nil || s.svc == nil || s.svc.TeacherCopilot == nil {
		s.writeRuntimeSSEError(c, req, http.StatusServiceUnavailable, "teacher copilot not available")
		return
	}

	stream, err := s.svc.TeacherCopilot.Stream(ctx, req)
	if err != nil {
		s.writeRuntimeSSEError(c, req, statusFromAgentError(err), assistantErrorMessage(err))
		return
	}

	for event := range stream {
		if event == nil {
			continue
		}
		streamEvent := agentbridge.ToStreamEvent(event)
		if streamEvent == nil {
			continue
		}
		streamEvent.SessionID = req.SessionID
		streamEvent.RoundID = req.RoundID
		streamEvent.AgentType = teachercopilot.AssistantType
		if err := s.writeSSEMessage(c, streamEvent); err != nil {
			return
		}
	}
}
