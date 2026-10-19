/*
@Author: xiaoTnb
@Date: 2026/05/12
@Description: RSS 资讯 HTTP 路由：用户侧需登录；管理端需教师或管理员；SSE 支持 query token。
*/

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterRSS 注册 RSS 相关路由。
func (s *Server) RegisterRSS(g *gin.RouterGroup) {
	rss := g.Group("/rss")
	{
		user := rss.Group("").Use(middleware.JWTMiddleware())
		{
			user.GET("/items", s.RSSListItems)
			user.GET("/items/latest", s.RSSListLatestItems)
			user.GET("/items/:id", s.RSSGetItem)
			user.GET("/unread-count", s.RSSUnreadCount)
			user.POST("/read", s.RSSMarkRead)
			user.GET("/feed/recommended", s.RSSRecommendedFeed)
			user.POST("/events", s.RSSRecordEvent)
		}
		rss.GET("/stream", s.RSSStream)

		admin := rss.Group("/admin").Use(middleware.JWTMiddleware(), middleware.RequireTeacherOrAdmin())
		{
			admin.GET("/sources", s.RSSAdminListSources)
			admin.POST("/sources", s.RSSAdminCreateSource)
			admin.PUT("/sources/:id", s.RSSAdminUpdateSource)
			admin.DELETE("/sources/:id", s.RSSAdminDeleteSource)
			admin.POST("/sources/:id/refresh", s.RSSAdminRefreshSource)
			admin.POST("/refresh-all", s.RSSAdminRefreshAll)
		}
	}
}

func bearerOrQueryToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == auth {
		token = ""
	}
	if token == "" {
		token = c.Query("token")
	}
	return strings.TrimSpace(token)
}

// RSSStream 新条通知 SSE（支持 query since、Last-Event-ID 补全，连接时先发 snapshot）。
func (s *Server) RSSStream(c *gin.Context) {
	tok := bearerOrQueryToken(c)
	if tok == "" {
		utils.UnauthorizedResponse(c, "缺少认证令牌")
		return
	}
	if _, err := middleware.ParseToken(tok); err != nil {
		utils.UnauthorizedResponse(c, "认证令牌无效")
		return
	}
	if _, ok := c.Writer.(http.Flusher); !ok {
		utils.InternalServerErrorResponse(c, "服务器不支持流式输出")
		return
	}

	ch, ok := s.svc.RSSService.TrySubscribeRSSNotifications()
	if !ok {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "推送连接数已满，请稍后重试")
		return
	}
	defer s.svc.RSSService.UnsubscribeRSSNotifications(ch)

	s.prepareSSE(c)

	after := parseRSSStreamAfter(c)
	ctx := c.Request.Context()
	if snap, err := s.svc.RSSService.SnapshotRSSItemsSince(ctx, after); err != nil {
		utils.Logger.Warnf("RSS snapshot 失败: %v", err)
	} else {
		_ = s.writeRSSSSE(c, "snapshot", extractRSSEventID(snap), snap)
	}

	fmt.Fprintf(c.Writer, ": connected\n\n")
	flushSSE(c)

	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			fmt.Fprintf(c.Writer, ": ping\n\n")
			flushSSE(c)
		case msg, open := <-ch:
			if !open {
				return
			}
			_ = s.writeRSSSSE(c, sseEventNameFromPayload(msg), extractRSSEventID(msg), msg)
		}
	}
}

// parseRSSStreamAfter 合并 query since 与 Last-Event-ID，取较晚时间；均无则默认回溯 24h。
func parseRSSStreamAfter(c *gin.Context) time.Time {
	var candidates []time.Time
	if raw := strings.TrimSpace(c.Query("since")); raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			candidates = append(candidates, t)
		} else if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			candidates = append(candidates, t)
		}
	}
	if raw := strings.TrimSpace(c.GetHeader("Last-Event-ID")); raw != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			candidates = append(candidates, t)
		} else if t, err := time.Parse(time.RFC3339, raw); err == nil {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return time.Now().Add(-24 * time.Hour)
	}
	max := candidates[0]
	for _, t := range candidates[1:] {
		if t.After(max) {
			max = t
		}
	}
	return max
}

func extractRSSEventID(payload []byte) string {
	var ev struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return ""
	}
	return ev.EventID
}

func sseEventNameFromPayload(payload []byte) string {
	var ev struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil || ev.Type == "" {
		return "message"
	}
	return ev.Type
}

// writeRSSSSE 写入带 id 的 SSE 帧（data 为已序列化的 JSON 字节）。
func (s *Server) writeRSSSSE(c *gin.Context, eventName, eventID string, data []byte) error {
	if eventID != "" {
		if _, err := fmt.Fprintf(c.Writer, "id: %s\n", eventID); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventName, data); err != nil {
		return err
	}
	flushSSE(c)
	return nil
}

func flushSSE(c *gin.Context) {
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

func rssUserIDFromContext(c *gin.Context) (string, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return strings.TrimSpace(s), ok && s != ""
}

// RSSListItems 分页资讯列表（需登录，含 is_read）。
func (s *Server) RSSListItems(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	page, err := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	var sourceID *primitive.ObjectID
	if sid := strings.TrimSpace(c.Query("source_id")); sid != "" {
		id, err := primitive.ObjectIDFromHex(sid)
		if err != nil {
			utils.BadRequestResponse(c, "无效的 source_id")
			return
		}
		sourceID = &id
	}
	rows, total, err := s.svc.RSSService.ListRSSItemsForUser(c.Request.Context(), userID, sourceID, page, pageSize)
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{
		"items":       rows,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// RSSListLatestItems 按入库时间查询新条目（轮询用，需登录，含 is_read）。
func (s *Server) RSSListLatestItems(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	sinceStr := strings.TrimSpace(c.Query("since"))
	if sinceStr == "" {
		utils.BadRequestResponse(c, "缺少 since 参数（RFC3339 时间）")
		return
	}
	after, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		if after, err = time.Parse(time.RFC3339Nano, sinceStr); err != nil {
			utils.BadRequestResponse(c, "since 格式应为 RFC3339")
			return
		}
	}
	limit := int64(50)
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}
	rows, err := s.svc.RSSService.ListRSSItemsNewerThanForUser(c.Request.Context(), userID, after, limit)
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"items": rows})
}

// RSSGetItem 资讯详情（需登录，含 is_read）。
func (s *Server) RSSGetItem(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的资讯 ID")
		return
	}
	item, err := s.svc.RSSService.GetRSSItemDetailForUser(c.Request.Context(), userID, id)
	if err != nil {
		utils.NotFoundResponse(c, err.Error())
		return
	}
	_ = s.svc.RSSService.RecordRSSEvent(c.Request.Context(), userID, &models.RecordRSSEventRequest{
		ItemID: idStr,
		Action: "open",
	})
	utils.SuccessResponse(c, item)
}

// RSSUnreadCount 当前用户未读资讯数量。
func (s *Server) RSSUnreadCount(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	count, state, err := s.svc.RSSService.GetRSSUnreadCount(c.Request.Context(), userID)
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	lastReadAt := ""
	if state != nil && !state.LastReadAt.IsZero() {
		lastReadAt = state.LastReadAt.UTC().Format(time.RFC3339Nano)
	}
	utils.SuccessResponse(c, gin.H{
		"unread_count": count,
		"last_read_at": lastReadAt,
	})
}

// RSSMarkRead 标记资讯已读（单条或全部）。
func (s *Server) RSSMarkRead(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	var req models.MarkRSSReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}
	if err := s.svc.RSSService.MarkRSSRead(c.Request.Context(), userID, &req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	count, _, _ := s.svc.RSSService.GetRSSUnreadCount(c.Request.Context(), userID)
	utils.SuccessResponse(c, gin.H{
		"message":      "已更新阅读状态",
		"unread_count": count,
	})
}

// RSSRecommendedFeed 个性化推荐资讯流。
func (s *Server) RSSRecommendedFeed(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	pageSize, _ := strconv.ParseInt(c.DefaultQuery("page_size", "20"), 10, 64)
	rows, total, err := s.svc.RSSService.GetRSSRecommendedFeed(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{
		"items":       rows,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
	})
}

// RSSRecordEvent 上报资讯行为（open/click/dwell/not_interested）。
func (s *Server) RSSRecordEvent(c *gin.Context) {
	userID, ok := rssUserIDFromContext(c)
	if !ok {
		utils.UnauthorizedResponse(c, "未登录")
		return
	}
	var req models.RecordRSSEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}
	if err := s.svc.RSSService.RecordRSSEvent(c.Request.Context(), userID, &req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "已记录"})
}

// RSSAdminListSources 列出订阅源。
func (s *Server) RSSAdminListSources(c *gin.Context) {
	list, err := s.svc.RSSService.ListRSSSources(c.Request.Context())
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"sources": list})
}

// RSSAdminCreateSource 创建订阅源。
func (s *Server) RSSAdminCreateSource(c *gin.Context) {
	var req models.CreateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}
	if err := s.svc.RSSService.CreateRSSSource(c.Request.Context(), &req); err != nil {
		if strings.Contains(err.Error(), "已存在") {
			utils.BadRequestResponse(c, err.Error())
			return
		}
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "创建成功"})
}

// RSSAdminUpdateSource 更新订阅源。
func (s *Server) RSSAdminUpdateSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的订阅源 ID")
		return
	}
	var req models.UpdateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}
	if err := s.svc.RSSService.UpdateRSSSource(c.Request.Context(), id, &req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "更新成功"})
}

// RSSAdminDeleteSource 删除订阅源。
func (s *Server) RSSAdminDeleteSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的订阅源 ID")
		return
	}
	if err := s.svc.RSSService.DeleteRSSSource(c.Request.Context(), id); err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// RSSAdminRefreshSource 手动刷新单个订阅源。
func (s *Server) RSSAdminRefreshSource(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的订阅源 ID")
		return
	}
	if err := s.svc.RSSService.ManualRefreshSource(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "频繁") {
			utils.BadRequestResponse(c, err.Error())
			return
		}
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "刷新已触发并完成"})
}

// RSSAdminRefreshAll 手动刷新全部已启用订阅源。
func (s *Server) RSSAdminRefreshAll(c *gin.Context) {
	if err := s.svc.RSSService.ManualRefreshAll(c.Request.Context()); err != nil {
		if strings.Contains(err.Error(), "频繁") {
			utils.BadRequestResponse(c, err.Error())
			return
		}
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, gin.H{"message": "已依次刷新全部启用源"})
}
