/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合内容接口，负责编程题/选择题/填空题的统一列表输出
*/

package server

import (
	"strconv"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetCourseContents godoc
// @Summary      获取课程混合内容列表
// @Description  在课程上下文中混合返回编程题、选择题、填空题。
// @Description  返回条目统一使用 `content_type` 区分题型，便于前端在同一列表里混合渲染。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        page query int false "页码，从 1 开始" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Success      200 {object} utils.Response{data=models.CourseContentListResponse} "查询成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Security     BearerAuth
// @Router       /courses/{courseId}/contents [get]
// GetCourseContents 获取课程混合内容列表。
func (s *Server) GetCourseContents(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("courseId"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的课程ID")
		return
	}

	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	userID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	roleValue, _ := c.Get("role")
	role := models.UserRole(roleValue.(string))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	response, err := s.svc.CourseContentService.ListCourseContents(
		c.Request.Context(),
		courseID,
		userID,
		role,
		page,
		pageSize,
	)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, response)
}
