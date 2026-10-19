/*
@Author: omenkk7
@Date: 2025/10/26
@Description: WebSocket连接管理器
*/

package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

// ConnectionManager WebSocket连接管理器
type ConnectionManager struct {
	// 按提交ID存储连接
	connections map[string]map[*websocket.Conn]bool // submitID -> connections
	// 按用户ID存储连接（用于权限检查）
	userConnections map[string]map[*websocket.Conn]bool // userID -> connections
	// 读写锁
	mutex sync.RWMutex
	// 广播通道
	broadcast chan BroadcastMessage
	// 关闭通道
	shutdown chan bool
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	SubmitID string
	UserID   string
	Message  models.WSMessage
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections:     make(map[string]map[*websocket.Conn]bool),
		userConnections: make(map[string]map[*websocket.Conn]bool),
		broadcast:       make(chan BroadcastMessage, 256),
		shutdown:        make(chan bool),
	}
}

// Start 启动连接管理器
func (cm *ConnectionManager) Start(ctx context.Context) {
	logger := utils.GetLogger(ctx)
	logger.Info("WebSocket连接管理器启动")

	// 启动广播处理goroutine
	go cm.handleBroadcast(ctx)
	
	// 启动清理goroutine
	go cm.cleanup(ctx)
}

// Stop 停止连接管理器
func (cm *ConnectionManager) Stop() {
	close(cm.shutdown)
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(submitID, userID string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 添加到提交ID映射
	if cm.connections[submitID] == nil {
		cm.connections[submitID] = make(map[*websocket.Conn]bool)
	}
	cm.connections[submitID][conn] = true

	// 添加到用户ID映射
	if cm.userConnections[userID] == nil {
		cm.userConnections[userID] = make(map[*websocket.Conn]bool)
	}
	cm.userConnections[userID][conn] = true

	log.Printf("WebSocket连接已建立: SubmitID=%s, UserID=%s, 总连接数=%d", 
		submitID, userID, cm.getTotalConnections())
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(submitID, userID string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// 从提交ID映射中移除
	if connections, exists := cm.connections[submitID]; exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(cm.connections, submitID)
		}
	}

	// 从用户ID映射中移除
	if connections, exists := cm.userConnections[userID]; exists {
		delete(connections, conn)
		if len(connections) == 0 {
			delete(cm.userConnections, userID)
		}
	}

	conn.Close()
	log.Printf("WebSocket连接已断开: SubmitID=%s, UserID=%s, 剩余连接数=%d", 
		submitID, userID, cm.getTotalConnections())
}

// BroadcastToSubmit 向指定提交的所有连接广播消息
func (cm *ConnectionManager) BroadcastToSubmit(submitID, userID string, message models.WSMessage) {
	select {
	case cm.broadcast <- BroadcastMessage{
		SubmitID: submitID,
		UserID:   userID,
		Message:  message,
	}:
	default:
		log.Printf("广播通道已满，消息被丢弃: SubmitID=%s", submitID)
	}
}

// handleBroadcast 处理广播消息
func (cm *ConnectionManager) handleBroadcast(ctx context.Context) {
	logger := utils.GetLogger(ctx)
	
	for {
		select {
		case message := <-cm.broadcast:
			cm.sendMessageToSubmit(message.SubmitID, message.UserID, message.Message)
		case <-cm.shutdown:
			logger.Info("WebSocket广播处理器关闭")
			return
		}
	}
}

// sendMessageToSubmit 发送消息到指定提交的连接
func (cm *ConnectionManager) sendMessageToSubmit(submitID, userID string, message models.WSMessage) {
	cm.mutex.RLock()
	connections := cm.connections[submitID]
	cm.mutex.RUnlock()

	if len(connections) == 0 {
		return
	}

	// 验证消息类型
	if !cm.isValidMessageType(message.Type) {
		log.Printf("无效的WebSocket消息类型: %s", message.Type)
		return
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化WebSocket消息失败: %v", err)
		return
	}

	// 向所有连接发送消息
	var deadConnections []*websocket.Conn
	for conn := range connections {
		// 设置写入超时
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		
		if err := conn.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			log.Printf("发送WebSocket消息失败: %v", err)
			deadConnections = append(deadConnections, conn)
		}
	}

	// 清理死连接
	if len(deadConnections) > 0 {
		cm.cleanupDeadConnectionsFromList(deadConnections, submitID)
	}
}

// isValidMessageType 验证消息类型
func (cm *ConnectionManager) isValidMessageType(messageType string) bool {
	validTypes := map[string]bool{
		"status_update":     true,
		"progress_update":   true,
		"judge_complete":    true,
		"error":             true,
		"ping":              true,
		"pong":              true,
	}
	return validTypes[messageType]
}

// cleanupDeadConnectionsFromList 从指定列表中清理死连接
func (cm *ConnectionManager) cleanupDeadConnectionsFromList(deadConnections []*websocket.Conn, submitID string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	for _, conn := range deadConnections {
		// 从提交连接中移除
		if connections, exists := cm.connections[submitID]; exists {
			delete(connections, conn)
			if len(connections) == 0 {
				delete(cm.connections, submitID)
			}
		}
		
		// 从用户连接中移除
		for uid, userConns := range cm.userConnections {
			delete(userConns, conn)
			if len(userConns) == 0 {
				delete(cm.userConnections, uid)
			}
		}
		
		conn.Close()
	}
}

// cleanup 定期清理无效连接
func (cm *ConnectionManager) cleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanupDeadConnections()
		case <-cm.shutdown:
			return
		}
	}
}

// cleanupDeadConnections 清理无效连接
func (cm *ConnectionManager) cleanupDeadConnections() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	var deadConnections []*websocket.Conn
	
	// 检查所有连接
	for submitID, connections := range cm.connections {
		for conn := range connections {
			// 设置写入超时，避免阻塞
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			
			// 发送ping消息测试连接
			if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				deadConnections = append(deadConnections, conn)
				delete(connections, conn)
			}
		}
		
		// 如果提交的所有连接都无效，删除整个映射
		if len(connections) == 0 {
			delete(cm.connections, submitID)
		}
	}

	// 清理用户连接映射中的无效连接
	for userID, connections := range cm.userConnections {
		for conn := range connections {
			// 检查连接是否在死连接列表中
			for _, deadConn := range deadConnections {
				if conn == deadConn {
					delete(connections, conn)
					break
				}
			}
		}
		
		// 如果用户的所有连接都无效，删除整个映射
		if len(connections) == 0 {
			delete(cm.userConnections, userID)
		}
	}

	// 关闭无效连接
	for _, conn := range deadConnections {
		conn.Close()
	}

	if len(deadConnections) > 0 {
		log.Printf("清理了 %d 个无效WebSocket连接", len(deadConnections))
	}
}

// getTotalConnections 获取总连接数（内部方法，调用时需要持有锁）
func (cm *ConnectionManager) getTotalConnections() int {
	total := 0
	for _, connections := range cm.connections {
		total += len(connections)
	}
	return total
}

// GetStats 获取连接统计信息
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return map[string]interface{}{
		"total_connections": cm.getTotalConnections(),
		"submit_connections": len(cm.connections),
		"user_connections":   len(cm.userConnections),
	}
}

// HasConnectionForSubmit 检查是否有连接监听指定提交
func (cm *ConnectionManager) HasConnectionForSubmit(submitID string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	connections, exists := cm.connections[submitID]
	return exists && len(connections) > 0
}

// 创建全局连接管理器实例
var GlobalConnectionManager = NewConnectionManager()

// BroadcastSubmitStatusUpdate 广播提交状态更新（全局方法）
func BroadcastSubmitStatusUpdate(submitID primitive.ObjectID, userID primitive.ObjectID, status models.SubmitStatus, progress *models.JudgeProgress, result *models.JudgeResult) {
	// 构建WebSocket消息
	message := models.WSMessage{
		Type:      "status_update",
		SubmitID:  submitID.Hex(),
		Timestamp: time.Now(),
		Data: models.WSStatusUpdate{
			Status:   status,
			Progress: progress,
			Message:  getStatusMessage(status),
			Result:   convertToWSResult(result),
		},
	}

	GlobalConnectionManager.BroadcastToSubmit(submitID.Hex(), userID.Hex(), message)
}

// getStatusMessage 获取状态消息
func getStatusMessage(status models.SubmitStatus) string {
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

// convertToWSResult 转换判题结果为WebSocket结果格式
func convertToWSResult(result *models.JudgeResult) *models.WSJudgeResult {
	if result == nil {
		return nil
	}

	// 计算通过的测试用例数
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
