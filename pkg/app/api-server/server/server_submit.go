/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: server_submit.go
@Description: 提交相关路由处理
*/

package server

import (
	"strconv"
	"zk-code-arena-server/pkg/common/queue"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterSubmit 注册提交相关路由
func (s *Server) RegisterSubmit(g *gin.RouterGroup) {
	submitGroup := g.Group("/submit")
	{
		// 需要认证的路由
		submitGroup.Use(middleware.JWTMiddleware())
		{
			submitGroup.POST("", s.SubmitCode)                // 提交代码
			submitGroup.GET("", s.GetSubmits)                 // 获取提交列表
			submitGroup.GET("/:id", s.GetSubmit)              // 获取提交详情
			submitGroup.GET("/:id/status", s.GetSubmitStatus) // 获取提交状态（轻量级）
		}
	}
}

// SubmitCode godoc
// @Summary      提交代码
// @Description  提交代码进行判题
// @Tags         提交
// @Accept       json
// @Produce      json
// @Param        request body models.SubmitCodeRequest true "提交信息"
// @Success      200 {object} utils.Response{data=models.Submit} "提交记录"
// @Failure      400 {object} map[string]interface{} "请求参数错误"
// @Failure      401 {object} map[string]interface{} "未认证用户"
// @Failure      404 {object} map[string]interface{} "题目不存在"
// @Failure      500 {object} map[string]interface{} "提交失败"
// @Security     BearerAuth
// @Router       /submit [post]
func (s *Server) SubmitCode(c *gin.Context) {

	var submitReq models.SubmitCodeRequest

	if err := c.ShouldBindJSON(&submitReq); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 验证编程语言
	language := models.Language(submitReq.Language)
	switch language {
	case models.LangC, models.LangCpp, models.LangJava, models.LangPython, models.LangGo:
		// 支持的语言
	default:
		utils.BadRequestResponse(c, "不支持的编程语言")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未认证用户")
		return
	}

	submitUserID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	// 检查题目是否存在
	ctx := c.Request.Context()
	problem, err := s.svc.ProblemService.GetProblemByID(ctx, submitReq.ProblemID)
	if err != nil {
		utils.NotFoundResponse(c, "题目不存在")
		return
	}

	// 检查题目是否公开
	if !problem.IsPublic {
		utils.ForbiddenResponse(c, "无法提交私有题目")
		return
	}

	// 创建提交记录
	submit := &models.Submit{
		ProblemID: submitReq.ProblemID,
		UserID:    submitUserID,
		Code:      submitReq.Code,
		Language:  language,
	}

	if err := s.svc.SubmitService.CreateSubmit(ctx, submit); err != nil {
		utils.InternalServerErrorResponse(c, "创建提交失败: "+err.Error())
		return
	}

	// 构建判题任务
	judgeTask := &queue.JudgeTask{
		SubmitID:  submit.ID,
		ProblemID: submit.ProblemID,
		UserID:    submit.UserID,
		Code:      submit.Code,
		Language:  string(submit.Language),
	}

	// 推送判题任务到消息队列（异步评测）
	if err := s.svc.JudgeService.SubmitTask(ctx, judgeTask); err != nil {
		// 任务推送失败，更新状态为系统错误
		submit.Status = models.StatusSystemError
		submit.Result = &models.JudgeResult{
			Status:       models.StatusSystemError,
			RuntimeError: "任务推送失败: " + err.Error(),
		}
		s.svc.SubmitService.UpdateSubmit(ctx, submit)
		utils.InternalServerErrorResponse(c, "判题任务推送失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message":   "提交成功，正在评测中",
		"submit_id": submit.ID,
	})
}

// GetSubmits godoc
// @Summary      获取提交列表
// @Description  分页获取提交记录（非管理员仅可查看自己的提交）
// @Tags         提交
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        problem_id query string false "题目ID筛选"
// @Param        user_id query string false "用户ID筛选（管理员可用）"
// @Success      200 {object} utils.Response{data=object{submits=[]models.Submit,total=int64,page=int,page_size=int,total_page=int64}} "提交列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "需要登录"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /submit [get]
func (s *Server) GetSubmits(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	problemIDStr := c.Query("problem_id")
	userIDStr := c.Query("user_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var problemID, userID *primitive.ObjectID
	var err error

	if problemIDStr != "" {
		id, err := primitive.ObjectIDFromHex(problemIDStr)
		if err != nil {
			utils.BadRequestResponse(c, "无效的题目ID")
			return
		}
		problemID = &id
	}

	if userIDStr != "" {
		id, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			utils.BadRequestResponse(c, "无效的用户ID")
			return
		}
		userID = &id
	}

	// 检查权限：非管理员只能查看自己的提交
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	if role.(string) != string(models.RoleAdmin) {
		// 非管理员只能查看自己的提交
		currentUserID, _ := c.Get("user_id")
		currentUserObjectID, _ := primitive.ObjectIDFromHex(currentUserID.(string))
		userID = &currentUserObjectID
	}

	// 获取状态筛选参数（可选）
	statusStr := c.Query("status")
	status := models.SubmitStatus(statusStr)

	ctx := c.Request.Context()
	submits, total, err := s.svc.SubmitService.GetSubmits(ctx, page, pageSize, userID, problemID, status)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取提交列表失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"submits":    submits,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetSubmit godoc
// @Summary      获取提交详情
// @Description  根据提交ID获取详细信息（管理员或提交者本人）
// @Tags         提交
// @Accept       json
// @Produce      json
// @Param        id path string true "提交ID"
// @Success      200 {object} utils.Response{data=models.Submit} "提交详情"
// @Failure      400 {object} models.ErrorResponse "无效的提交ID"
// @Failure      401 {object} models.ErrorResponse "需要登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      404 {object} models.ErrorResponse "提交不存在"
// @Security     BearerAuth
// @Router       /submit/{id} [get]
func (s *Server) GetSubmit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的提交ID")
		return
	}

	ctx := c.Request.Context()
	submit, err := s.svc.SubmitService.GetSubmitByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "提交不存在")
		return
	}

	// 检查权限：非管理员只能查看自己的提交
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	if role.(string) != string(models.RoleAdmin) {
		currentUserID, _ := c.Get("user_id")
		currentUserObjectID, _ := primitive.ObjectIDFromHex(currentUserID.(string))
		if submit.UserID != currentUserObjectID {
			utils.ForbiddenResponse(c, "只能查看自己的提交")
			return
		}
	}

	utils.SuccessResponse(c, submit)
}

// GetSubmitStatus godoc
// @Summary      获取提交状态（轻量级）
// @Description  获取提交的当前状态和进度信息，用于实时状态监控，比完整详情接口更轻量
// @Tags         提交
// @Accept       json
// @Produce      json
// @Param        id path string true "提交ID"
// @Success      200 {object} utils.Response{data=models.SubmitStatusResponse} "提交状态信息"
// @Failure      400 {object} map[string]interface{} "无效的提交ID"
// @Failure      401 {object} map[string]interface{} "需要登录"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      404 {object} map[string]interface{} "提交不存在"
// @Security     BearerAuth
// @Router       /submit/{id}/status [get]
func (s *Server) GetSubmitStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的提交ID")
		return
	}

	ctx := c.Request.Context()

	// 首先获取提交基本信息以进行权限检查
	submit, err := s.svc.SubmitService.GetSubmitByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "提交不存在")
		return
	}

	// 检查权限：非管理员只能查看自己的提交
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	if role.(string) != string(models.RoleAdmin) {
		currentUserID, _ := c.Get("user_id")
		currentUserObjectID, _ := primitive.ObjectIDFromHex(currentUserID.(string))
		if submit.UserID != currentUserObjectID {
			utils.ForbiddenResponse(c, "只能查看自己的提交状态")
			return
		}
	}

	// 获取轻量级状态信息
	status, err := s.svc.SubmitService.GetSubmitStatus(ctx, id)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取提交状态失败")
		return
	}

	utils.SuccessResponse(c, status)
}
