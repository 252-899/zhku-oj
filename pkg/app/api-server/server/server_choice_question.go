/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程选择题接口，负责课程上下文下的选择题创建入口
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterChoiceQuestion 注册课程选择题相关接口。
// 这里把题型路由收敛在独立文件内，统一走通用题目接口。
func (s *Server) RegisterChoiceQuestion(g *gin.RouterGroup) {
	rootGroup := g.Group("").Use(middleware.JWTMiddleware())
	{
		rootGroup.POST("/choice-questions", s.CreateChoiceQuestion)
		rootGroup.GET("/choice-questions/:questionId", s.GetChoiceQuestion)
		rootGroup.PUT("/choice-questions/:questionId", s.UpdateChoiceQuestion)
		rootGroup.DELETE("/choice-questions/:questionId", s.DeleteChoiceQuestion)
	}
}

// CreateChoiceQuestion godoc
// @Summary      创建选择题
// @Description  创建通用选择题，支持全局题库与课程题库两种发布模式。
// @Description  当 `publish_mode` 为 `course_only` 或 `global_and_course` 时，必须通过 `course_ids` 显式指定挂载课程。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        request body models.CreateChoiceQuestionRequest true "选择题创建请求"
// @Success      200 {object} utils.Response{data=models.CreateChoiceQuestionResponse} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /choice-questions [post]
func (s *Server) CreateChoiceQuestion(c *gin.Context) {
	userID, role, ok := s.parseQuestionWriteIdentity(c)
	if !ok {
		return
	}

	var req models.CreateChoiceQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	question, err := s.svc.ChoiceQuestionService.CreateChoiceQuestion(c.Request.Context(), userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"question_id": question.ID,
		"question":    question,
	})
}

// GetChoiceQuestion godoc
// @Summary      获取选择题详情
// @Description  获取通用选择题详情，支持全局题和课程挂载题的统一读取。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "选择题ID"
// @Success      200 {object} utils.Response{data=models.ChoiceQuestion} "查询成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /choice-questions/{questionId} [get]
func (s *Server) GetChoiceQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionReadIdentity(c, "questionId")
	if !ok {
		return
	}
	question, err := s.svc.ChoiceQuestionService.GetChoiceQuestion(c.Request.Context(), questionID, userID, role)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// UpdateChoiceQuestion godoc
// @Summary      更新选择题
// @Description  更新通用选择题；若变更课程挂载范围，需要在 `course_ids` 中显式给出目标课程集合。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "选择题ID"
// @Param        request body models.UpdateChoiceQuestionRequest true "选择题更新请求"
// @Success      200 {object} utils.Response{data=models.ChoiceQuestion} "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /choice-questions/{questionId} [put]
func (s *Server) UpdateChoiceQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	var req models.UpdateChoiceQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	question, err := s.svc.ChoiceQuestionService.UpdateChoiceQuestion(c.Request.Context(), questionID, userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// DeleteChoiceQuestion godoc
// @Summary      删除选择题
// @Description  删除通用选择题，同时清理课程挂载关系。
// @Tags         题库
// @Accept       json
// @Produce      json
// @Param        questionId path string true "选择题ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /choice-questions/{questionId} [delete]
func (s *Server) DeleteChoiceQuestion(c *gin.Context) {
	questionID, userID, role, ok := s.parseQuestionWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	if err := s.svc.ChoiceQuestionService.DeleteChoiceQuestion(c.Request.Context(), questionID, userID, role); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}

// CreateCourseChoiceQuestion godoc
// @Summary      在课程下创建选择题
// @Description  课程教师或管理员在课程上下文中创建选择题，并按发布范围建立课程挂载关系。
// @Description  `choice_mode=single` 表示单选题，`choice_mode=multiple` 表示多选题。
// @Description  `publish_mode=course_only` 或 `global_and_course` 时，会自动把当前 `courseId` 纳入课程挂载范围。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.CreateChoiceQuestionRequest true "选择题创建请求"
// @Success      200 {object} utils.Response{data=models.CreateChoiceQuestionResponse} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/choice-questions [post]
// CreateCourseChoiceQuestion 在课程上下文中创建选择题。
func (s *Server) CreateCourseChoiceQuestion(c *gin.Context) {
	courseID, userID, role, ok := s.parseCourseWriteIdentity(c)
	if !ok {
		return
	}

	var req models.CreateChoiceQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	question, err := s.svc.ChoiceQuestionService.CreateCourseChoiceQuestion(
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

// GetCourseChoiceQuestion godoc
// @Summary      获取课程选择题详情
// @Description  在课程上下文中获取单道选择题详情。
// @Description  已登录用户可以发起请求，最终可见性会继续结合课程成员关系、题目发布状态和创建者身份判断。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "选择题ID"
// @Success      200 {object} utils.Response{data=models.ChoiceQuestion} "查询成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/choice-questions/{questionId} [get]
func (s *Server) GetCourseChoiceQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseReadQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	question, err := s.svc.ChoiceQuestionService.GetCourseChoiceQuestion(c.Request.Context(), courseID, questionID, userID, role)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// UpdateCourseChoiceQuestion godoc
// @Summary      更新课程选择题
// @Description  仅题目创建者或管理员可在课程上下文中更新选择题。
// @Description  支持部分字段更新；如果把 `publish_mode` 改为 `global`，题目会脱离当前课程挂载。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "选择题ID"
// @Param        request body models.UpdateChoiceQuestionRequest true "选择题更新请求"
// @Success      200 {object} utils.Response{data=models.ChoiceQuestion} "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/choice-questions/{questionId} [put]
func (s *Server) UpdateCourseChoiceQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	var req models.UpdateChoiceQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	question, err := s.svc.ChoiceQuestionService.UpdateCourseChoiceQuestion(c.Request.Context(), courseID, questionID, userID, role, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, question)
}

// DeleteCourseChoiceQuestion godoc
// @Summary      删除课程选择题
// @Description  仅题目创建者或管理员可在课程上下文中删除选择题。
// @Description  删除时会同时清理课程内容挂载关系，但当前不会级联删除历史单题提交记录。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        questionId path string true "选择题ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/choice-questions/{questionId} [delete]
func (s *Server) DeleteCourseChoiceQuestion(c *gin.Context) {
	courseID, questionID, userID, role, ok := s.parseCourseWriteQuestionIdentity(c, "questionId")
	if !ok {
		return
	}
	if err := s.svc.ChoiceQuestionService.DeleteCourseChoiceQuestion(c.Request.Context(), courseID, questionID, userID, role); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}

// parseCourseWriteIdentity 提取课程写接口所需的课程、用户和角色信息。
func (s *Server) parseCourseWriteIdentity(c *gin.Context) (primitive.ObjectID, primitive.ObjectID, models.UserRole, bool) {
	roleValue, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	role := models.UserRole(roleValue.(string))
	if role != models.RoleAdmin && role != models.RoleTeacher {
		utils.ForbiddenResponse(c, "权限不足，只有教师和管理员可以操作")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}

	courseID, err := primitive.ObjectIDFromHex(c.Param("courseId"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的课程ID")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}

	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	userID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	return courseID, userID, role, true
}

// parseCourseWriteQuestionIdentity 提取课程题目写接口所需的课程、题目、用户和角色信息。
// 这里沿用课程写权限边界，确保编辑和删除始终只开放给教师与管理员。
func (s *Server) parseCourseWriteQuestionIdentity(c *gin.Context, questionParam string) (primitive.ObjectID, primitive.ObjectID, primitive.ObjectID, models.UserRole, bool) {
	courseID, userID, role, ok := s.parseCourseWriteIdentity(c)
	if !ok {
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	questionID, err := primitive.ObjectIDFromHex(c.Param(questionParam))
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	return courseID, questionID, userID, role, true
}

// parseCourseReadQuestionIdentity 提取课程题目读接口所需的课程、题目、用户和角色信息。
// 详情读取会在 service 层继续校验课程可见性，因此这里只要求用户已登录，避免把学生查看链路提前挡掉。
func (s *Server) parseCourseReadQuestionIdentity(c *gin.Context, questionParam string) (primitive.ObjectID, primitive.ObjectID, primitive.ObjectID, models.UserRole, bool) {
	roleValue, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	courseID, err := primitive.ObjectIDFromHex(c.Param("courseId"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的课程ID")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	questionID, err := primitive.ObjectIDFromHex(c.Param(questionParam))
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	userID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return primitive.NilObjectID, primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	return courseID, questionID, userID, models.UserRole(roleValue.(string)), true
}

// parseQuestionWriteIdentity 提取通用题目写接口所需的用户和角色信息。
// 通用题库写操作当前只开放给教师和管理员，避免学生绕过课程边界直接改题。
func (s *Server) parseQuestionWriteIdentity(c *gin.Context) (primitive.ObjectID, models.UserRole, bool) {
	roleValue, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, "", false
	}
	role := models.UserRole(roleValue.(string))
	if role != models.RoleAdmin && role != models.RoleTeacher {
		utils.ForbiddenResponse(c, "权限不足，只有教师和管理员可以操作")
		return primitive.NilObjectID, "", false
	}
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, "", false
	}
	userID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return primitive.NilObjectID, "", false
	}
	return userID, role, true
}

// parseQuestionWriteQuestionIdentity 提取通用题目写接口所需的题目、用户和角色信息。
func (s *Server) parseQuestionWriteQuestionIdentity(c *gin.Context, questionParam string) (primitive.ObjectID, primitive.ObjectID, models.UserRole, bool) {
	userID, role, ok := s.parseQuestionWriteIdentity(c)
	if !ok {
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	questionID, err := primitive.ObjectIDFromHex(c.Param(questionParam))
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	return questionID, userID, role, true
}

// parseQuestionReadIdentity 提取通用题目读接口所需的题目、用户和角色信息。
func (s *Server) parseQuestionReadIdentity(c *gin.Context, questionParam string) (primitive.ObjectID, primitive.ObjectID, models.UserRole, bool) {
	roleValue, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	questionID, err := primitive.ObjectIDFromHex(c.Param(questionParam))
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	userID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return primitive.NilObjectID, primitive.NilObjectID, "", false
	}
	return questionID, userID, models.UserRole(roleValue.(string)), true
}
