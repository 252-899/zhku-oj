/*
@Author: omenkk7
@Date: 2025/10/26
@Description: WebSocket路由处理
*/

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils/middleware"
	wsManager "zk-code-arena-server/pkg/utils/websocket"
)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中，应该检查请求来源
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// RegisterWebSocket 注册WebSocket相关路由
func (s *Server) RegisterWebSocket(g *gin.RouterGroup) {
	wsGroup := g.Group("/ws")
	{
		// 需要认证的WebSocket路由
		wsGroup.GET("/submit/:id", s.HandleSubmitWebSocket) // 提交状态实时推送
	}
}

// HandleSubmitWebSocket godoc
// @Summary      提交状态WebSocket连接
// @Description  建立WebSocket连接以实时接收提交状态更新。支持心跳机制和自动重连。
// @Tags         WebSocket
// @Param        id path string true "提交ID"
// @Param        token query string false "访问令牌（可选，也可通过Authorization header传递）"
// @Param        Authorization header string false "Bearer token"
// @Success      101 {object} models.WSMessage "WebSocket连接建立，后续通过WebSocket发送实时状态更新"
// @Failure      400 {object} models.ErrorResponse "无效的提交ID或请求"
// @Failure      401 {object} models.ErrorResponse "需要登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      404 {object} models.ErrorResponse "提交不存在"
// @Router       /ws/submit/{id} [get]
// @Security     BearerAuth
func (s *Server) HandleSubmitWebSocket(c *gin.Context) {
	// 参数验证
	submitIDStr := c.Param("id")
	submitID, err := primitive.ObjectIDFromHex(submitIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的提交ID"})
		return
	}

	// 从查询参数获取token（WebSocket握手时无法使用标准header认证）
	token := c.Query("token")
	if token == "" {
		// 尝试从Authorization header获取
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "需要提供访问令牌"})
		return
	}

	// 验证token并获取用户信息
	userID, role, err := s.validateTokenAndGetUser(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的访问令牌"})
		return
	}

	// 权限验证：检查用户是否可以访问此提交
	ctx := c.Request.Context()
	submit, err := s.svc.SubmitService.GetSubmitByID(ctx, submitID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "提交不存在"})
		return
	}

	// 权限检查：非管理员只能监听自己的提交
	if role != string(models.RoleAdmin) && submit.UserID != *userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "只能监听自己的提交状态"})
		return
	}

	// 升级到WebSocket连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.lg.Errorf("WebSocket升级失败: %v", err)
		return
	}

	// 添加到连接管理器
	wsManager.GlobalConnectionManager.AddConnection(submitIDStr, userID.Hex(), conn)

	// 发送当前状态
	s.sendCurrentStatus(conn, submit)

	// 处理连接
	s.handleWebSocketConnection(conn, submitIDStr, userID.Hex())
}

// validateTokenAndGetUser 验证token并获取用户信息
func (s *Server) validateTokenAndGetUser(token string) (*primitive.ObjectID, string, error) {
	// 解析JWT token
	claims, err := middleware.ParseToken(token)
	if err != nil {
		return nil, "", fmt.Errorf("无效的token: %w", err)
	}
	
	// 验证token是否过期
	if claims.ExpiresAt < time.Now().Unix() {
		return nil, "", fmt.Errorf("token已过期")
	}
	
	// 解析用户ID
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, "", fmt.Errorf("无效的用户ID: %w", err)
	}
	
	// 验证角色
	if claims.Role != string(models.RoleStudent) && claims.Role != string(models.RoleTeacher) && claims.Role != string(models.RoleAdmin) {
		return nil, "", fmt.Errorf("无效的用户角色")
	}
	
	return &userID, claims.Role, nil
}

// sendCurrentStatus 发送当前状态
func (s *Server) sendCurrentStatus(conn *websocket.Conn, submit *models.Submit) {
	message := models.WSMessage{
		Type:      "status_update",
		SubmitID:  submit.ID.Hex(),
		Timestamp: time.Now(),
		Data: models.WSStatusUpdate{
			Status:  submit.Status,
			Message: s.getStatusMessage(submit.Status),
			Result:  s.convertToWSResult(submit.Result),
		},
	}

	if err := conn.WriteJSON(message); err != nil {
		s.lg.Errorf("发送当前状态失败: %v", err)
	}
}

// handleWebSocketConnection 处理WebSocket连接
func (s *Server) handleWebSocketConnection(conn *websocket.Conn, submitID, userID string) {
	defer func() {
		// 确保连接被正确清理
		wsManager.GlobalConnectionManager.RemoveConnection(submitID, userID, conn)
		s.lg.Infof("WebSocket连接已关闭: SubmitID=%s, UserID=%s", submitID, userID)
	}()

	// 设置连接参数
	conn.SetReadLimit(512) // 限制读取消息大小
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	
	// 设置Pong处理器
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳机制
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	// 启动心跳goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.lg.Errorf("WebSocket心跳goroutine异常: %v", r)
			}
		}()
		
		for {
			select {
			case <-pingTicker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					s.lg.Errorf("发送ping消息失败: %v", err)
					return
				}
			}
		}
	}()

	// 读取客户端消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.lg.Errorf("WebSocket连接异常关闭: %v", err)
			} else {
				s.lg.Debugf("WebSocket连接正常关闭: %v", err)
			}
			break
		}

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			s.handleTextMessage(conn, submitID, userID, message)
		case websocket.BinaryMessage:
			s.lg.Warnf("收到二进制消息，忽略: SubmitID=%s", submitID)
		case websocket.PingMessage:
			// 响应ping消息
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			conn.WriteMessage(websocket.PongMessage, message)
		case websocket.PongMessage:
			// 更新读取超时
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		case websocket.CloseMessage:
			s.lg.Infof("收到关闭消息: SubmitID=%s", submitID)
			return
		}
	}
}

// handleTextMessage 处理文本消息
func (s *Server) handleTextMessage(conn *websocket.Conn, submitID, userID string, message []byte) {
	// 解析消息
	var wsMessage models.WSMessage
	if err := json.Unmarshal(message, &wsMessage); err != nil {
		s.lg.Errorf("解析WebSocket消息失败: %v", err)
		return
	}

	// 处理不同类型的消息
	switch wsMessage.Type {
	case "ping":
		// 响应ping
		response := models.WSMessage{
			Type:      "pong",
			SubmitID:  submitID,
			Timestamp: time.Now(),
		}
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		conn.WriteJSON(response)
		
	case "subscribe":
		// 客户端订阅特定提交的状态更新
		s.lg.Infof("客户端订阅状态更新: SubmitID=%s, UserID=%s", submitID, userID)
		
	case "unsubscribe":
		// 客户端取消订阅
		s.lg.Infof("客户端取消订阅: SubmitID=%s, UserID=%s", submitID, userID)
		
	default:
		s.lg.Warnf("未知的WebSocket消息类型: %s", wsMessage.Type)
	}
}

// getStatusMessage 获取状态消息
func (s *Server) getStatusMessage(status models.SubmitStatus) string {
	switch status {
	case models.StatusPending:
		return "等待判题中..."
	case models.StatusRunning:
		return "正在判题..."
	case models.StatusAccepted:
		return "通过"
	case models.StatusWrongAnswer:
		return "答案错误"
	case models.StatusTimeLimit:
		return "时间超限"
	case models.StatusMemoryLimit:
		return "内存超限"
	case models.StatusRuntimeError:
		return "运行时错误"
	case models.StatusCompileError:
		return "编译错误"
	case models.StatusSystemError:
		return "系统错误"
	default:
		return "未知状态"
	}
}

// convertToWSResult 转换判题结果
func (s *Server) convertToWSResult(result *models.JudgeResult) *models.WSJudgeResult {
	if result == nil {
		return nil
	}

	passedCases := 0
	for _, testResult := range result.TestResults {
		if testResult.Status == models.StatusAccepted {
			passedCases++
		}
	}

	return &models.WSJudgeResult{
		Status:       result.Status,
		TimeUsed:     result.TimeUsed,
		MemoryUsed:   result.MemoryUsed,
		PassedCases:  passedCases,
		TotalCases:   len(result.TestResults),
		CompileError: result.CompileError,
	}
}
