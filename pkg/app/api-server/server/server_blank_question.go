/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程填空题接口，负责课程上下文下的填空题创建入口
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterBlankQuestion 注册课程填空题相关接口。
// 独立注册函数可以让题型路由和处理逻辑保持同文件归档。
func (s *Server) RegisterBlankQuestion(g *gin.RouterGroup) {
	rootGroup := g.Group("").Use(middleware.JWTMiddleware())
	{
		rootGroup.POST("/blank-questions", s.CreateBlankQuestion)
		rootGroup.GET("/blank-questions/:questionId", s.GetBlankQuestion)
		rootGroup.PUT("/blank-questions/:questionId", s.UpdateBlankQuestion)
		rootGroup.DELETE("/blank-questions/:questionId", s.DeleteBlankQuestion)
	}
}

// CreateBlankQuestion godoc
// @Summary      创建填空题
// @Description  创建通用填空题，支持全局题库与课程题库两种发布模式。
// @Description  当 `publish_mode` 为 `course_only` 或 `global_and_course` 时，必须通过 `course_ids` 显式指定挂载课程。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        request body models.CreateBlankQuestionRequest true "填空题创建请求"
// @Success      200 {object} utils.Response{data=models.CreateBlankQuestionResponse} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /blank-questions [post]
func (s *Server) CreateBlankQuestion(c *gin.Context) {
	userID, role, ok := s.parseQuestionWriteIdentity(c)
	if !ok {
		return
	}

	var req models.CreateBlankQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	question, err := s.svc.BlankQuestionService.CreateBlankQuestion(c.Request.Context(), userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"question_id": question.ID,
		"question":    question,
	})
}

// GetBlankQuestion godoc
// @Summary      获取填空题详情
// @Description  获取通用填空题详情，支持全局题和课程挂载题的统一读取。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "填空题ID"
// @Success      200 {object} utils.Response{data=models.BlankQuestion} "查询成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /blank-questions/{questionId} [get]
func (s *Server) GetBlankQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionReadIdentity(c, "questionId")
	if !ok {
		return
	}
	question, err := s.svc.BlankQuestionService.GetBlankQuestion(c.Request.Context(), questionID, userID, role)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// UpdateBlankQuestion godoc
// @Summary      更新填空题
// @Description  更新通用填空题；若变更课程挂载范围，需要在 `course_ids` 中显式给出目标课程集合。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "填空题ID"
// @Param        request body models.UpdateBlankQuestionRequest true "填空题更新请求"
// @Success      200 {object} utils.Response{data=models.BlankQuestion} "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /blank-questions/{questionId} [put]
func (s *Server) UpdateBlankQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	var req models.UpdateBlankQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	question, err := s.svc.BlankQuestionService.UpdateBlankQuestion(c.Request.Context(), questionID, userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// DeleteBlankQuestion godoc
// @Summary      删除填空题
// @Description  删除通用填空题，同时清理课程挂载关系。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "填空题ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /blank-questions/{questionId} [delete]
func (s *Server) DeleteBlankQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	if err := s.svc.BlankQuestionService.DeleteBlankQuestion(c.Request.Context(), questionID, userID, role); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}

// CreateCourseBlankQuestion godoc
// @Summary      在课程下创建填空题
// @Description  课程教师或管理员在课程上下文中创建填空题，并按发布范围建立课程挂载关系。
// @Description  `judge_mode=exact_match` 表示自动判分，`judge_mode=manual` 表示保留人工判分。
// @Description  一道填空题整题只允许一种判分方式，`blanks[].order` 必须唯一。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.CreateBlankQuestionRequest true "填空题创建请求"
// @Success      200 {object} utils.Response{data=models.CreateBlankQuestionResponse} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/blank-questions [post]
// CreateCourseBlankQuestion 在课程上下文中创建填空题。
func (s *Server) CreateCourseBlankQuestion(c *gin.Context) {
	courseID, userID, role, ok := s.parseCourseWriteIdentity(c)
	if !ok {
		return
	}

	var req models.CreateBlankQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	question, err := s.svc.BlankQuestionService.CreateCourseBlankQuestion(
		c.Request.Context(),
		courseID,
		userID,
		role,
		&req,
	)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"question_id": question.ID,
		"question":    question,
	})
}

// GetCourseBlankQuestion godoc
// @Summary      获取课程填空题详情
// @Description  在课程上下文中获取单道填空题详情。
// @Description  已登录用户可以发起请求，最终可见性会继续结合课程成员关系、题目发布状态和创建者身份判断。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "填空题ID"
// @Success      200 {object} utils.Response{data=models.BlankQuestion} "查询成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/blank-questions/{questionId} [get]
func (s *Server) GetCourseBlankQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseReadQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	question, err := s.svc.BlankQuestionService.GetCourseBlankQuestion(c.Request.Context(), courseID, questionID, userID, role)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// UpdateCourseBlankQuestion godoc
// @Summary      更新课程填空题
// @Description  仅题目创建者或管理员可在课程上下文中更新填空题。
// @Description  支持部分字段更新；更新后的 `blanks[].order` 仍然必须唯一。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "填空题ID"
// @Param        request body models.UpdateBlankQuestionRequest true "填空题更新请求"
// @Success      200 {object} utils.Response{data=models.BlankQuestion} "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/blank-questions/{questionId} [put]
func (s *Server) UpdateCourseBlankQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	var req models.UpdateBlankQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	question, err := s.svc.BlankQuestionService.UpdateCourseBlankQuestion(c.Request.Context(), courseID, questionID, userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// DeleteCourseBlankQuestion godoc
// @Summary      删除课程填空题
// @Description  仅题目创建者或管理员可在课程上下文中删除填空题。
// @Description  删除时会同时清理课程内容挂载关系，但当前不会级联删除历史单题提交记录。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "填空题ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/blank-questions/{questionId} [delete]
func (s *Server) DeleteCourseBlankQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	if err := s.svc.BlankQuestionService.DeleteCourseBlankQuestion(c.Request.Context(), courseID, questionID, userID, role); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}
