/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一内容提交接口，负责选择题和填空题的独立作答提交
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterContentSubmission 注册统一内容提交路由。
func (s *Server) RegisterContentSubmission(g *gin.RouterGroup) {
	jwtGroup := g.Group("").Use(middleware.JWTMiddleware())
	jwtGroup.POST("/content-submissions", s.SubmitContentAnswer)
}

// SubmitContentAnswer godoc
// @Summary      提交选择题或填空题答案
// @Description  当前阶段支持选择题和填空题的独立作答提交，任务统一提交后续补充。
// @Description  `content_type=choice` 时请传 `choice_answers`，`content_type=blank` 时请传 `blank_answers`，两种答案结构不要混传。
// @Description  课程上下文作答建议始终传入 `course_id`，便于后端按课程挂载关系校验题目可见性。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        request body models.SubmitContentAnswerRequest true "题目提交请求"
// @Success      200 {object} utils.Response{data=models.ContentSubmissionResult} "提交成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Security     BearerAuth
// @Router       /content-submissions [post]
// SubmitContentAnswer 提交选择题或填空题的单题答案。
func (s *Server) SubmitContentAnswer(c *gin.Context) {
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

	var req models.SubmitContentAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	submission, err := s.svc.ContentSubmissionService.SubmitStandaloneAnswer(c.Request.Context(), userID, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, &models.ContentSubmissionResult{
		SubmissionID: submission.ID,
		JudgeStatus:  submission.JudgeStatus,
		Score:        submission.Score,
		IsCorrect:    submission.IsCorrect,
	})
}
