/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: server_problem.go
@Description: 题目相关路由处理
*/

package server

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"zk-code-arena-server/pkg/app/api-server/service"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterProblem 注册题目相关路由
func (s *Server) RegisterProblem(g *gin.RouterGroup) {
	// 每日一题（独立路由，不在problem组下）
	g.GET("/daily-problem", s.GetDailyProblem) // 获取每日推荐题目

	// 用户端题目路由组
	problemGroup := g.Group("/problem")
	{
		// 公开路由
		problemGroup.GET("", s.GetProblems)                     // 获取题目列表（用户端）
		problemGroup.GET("/search", s.SearchProblems)           // 搜索题目
		problemGroup.GET("/unique/:id", s.GetProblemByUniqueID) // 根据 unique_id 获取题目详情

		problemGroup.GET("/:id/detail", s.GetProblemDetail) // 获取题目详情聚合信息
		problemGroup.GET("/:id/next", s.GetNextProblem)

		problemGroup.GET("/:id", s.GetProblem) // 获取题目详情
	}

	// 需要认证的路由（使用 Group("") 避免 basePath 变成 /problem/ 引发尾斜杠重定向）
	securedGroup := problemGroup.Group("").Use(middleware.JWTMiddleware())
	{
		securedGroup.POST("", s.CreateProblem)                   // 创建题目
		securedGroup.POST("/batch", s.BatchCreateProblems)       // 批量创建题目
		securedGroup.PUT("/:id", s.UpdateProblem)                // 更新题目
		securedGroup.DELETE("/:id", s.DeleteProblem)             // 删除题目
		securedGroup.POST("/:id/favorite", s.ToggleFavorite)     // 收藏/取消收藏题目
		securedGroup.GET("/favorite", s.GetUserFavoriteProblems) // 获取用户收藏题目列表

		securedGroup.POST("/:id/run", middleware.CodeRunRateLimitMiddleware(), s.RunCode) // 运行代码测试
	}

	// 管理员专用路由组
	adminGroup := g.Group("/admin/problems").Use(middleware.JWTMiddleware())
	{
		adminGroup.GET("", s.GetProblemsForAdmin) // 获取题目列表（管理员端）
	}
}

// GetProblemByUniqueID godoc
// @Summary      通过题号获取题目详情
// @Description  根据题目唯一编号（如 1001）获取完整题目信息
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path int true "题目编号"
// @Success      200 {object} utils.Response{data=models.Problem} "题目详情"
// @Failure      400 {object} models.ErrorResponse "无效的题目编号"
// @Failure      404 {object} models.ErrorResponse "题目不存在"
// @Failure      500 {object} models.ErrorResponse "服务器内部错误"
// @Router       /problem/unique/{id} [get]
func (s *Server) GetProblemByUniqueID(c *gin.Context) {
	// 1. 从 URL 参数中获取题号，并转换为 int64 类型
	uniqueIDStr := c.Param("id")
	uniqueID, err := strconv.ParseInt(uniqueIDStr, 10, 64)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目编号")
		return
	}

	// 2. 调用 Service 层的新方法 GetProblemByUniqueID
	ctx := c.Request.Context()
	problem, err := s.svc.ProblemService.GetProblemByUniqueID(ctx, uniqueID)
	if err != nil {
		// 如果没找到，返回 404
		if errors.Is(err, models.ErrProblemNotFound) {
			utils.NotFoundResponse(c, "题目不存在")
		} else {
			// 其他错误，返回 500
			utils.InternalServerErrorResponse(c, "查询题目失败: "+err.Error())
		}
		return
	}

	// 3. 检查权限：课程私有题只允许课程成员、创建者或管理员查看。
	if ok, needAuth := s.canViewProblem(c.Request.Context(), c, problem); !ok {
		if needAuth {
			utils.UnauthorizedResponse(c, "需要登录才能查看此题目")
			return
		}
		utils.ForbiddenResponse(c, "无权查看此题目")
		return
	}

	// 4. 返回题目信息
	utils.SuccessResponse(c, problem)
}

// GetProblems godoc
// @Summary      获取题目列表（用户端）
// @Description  用户端获取题目列表，只显示公开已发布的题目，支持按难度、标签筛选
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        difficulty query string false "难度" Enums(easy, medium, hard)
// @Param        tags query []string false "标签列表"
// @Success      200 {object} utils.Response{data=object{problems=[]models.ProblemList,total=int64,page=int,page_size=int,total_page=int64}} "题目列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Router       /problem [get]
func (s *Server) GetProblems(c *gin.Context) {
	// 参数解析和验证
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	difficulty := c.Query("difficulty")
	tags := c.QueryArray("tags")

	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取用户ID（可选，用于获取用户提交状态）
	var userObjectID *primitive.ObjectID
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 有token，尝试解析
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != authHeader {
			// token格式正确，解析token
			if claims, err := middleware.ParseToken(tokenString); err == nil {
				// token有效，提取用户ID
				if objID, err := primitive.ObjectIDFromHex(claims.UserID); err == nil {
					userObjectID = &objID
				}
			}
		}
	}

	// 调用新的用户端Service方法
	ctx := c.Request.Context()
	problems, total, err := s.svc.ProblemService.GetProblemsForUser(
		ctx,
		page,
		pageSize,
		models.ProblemDifficulty(difficulty),
		tags,
		userObjectID,
	)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取题目列表失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"problems":   problems,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetCourseProblems godoc
// @Summary      获取课程题库题目列表
// @Description  获取指定课程下挂载的已发布题目；课程私有题不会出现在首页，但会出现在该接口。
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        difficulty query string false "难度" Enums(easy, medium, hard)
// @Param        tags query []string false "标签列表"
// @Success      200 {object} utils.Response{data=object{problems=[]models.ProblemList,total=int64,page=int,page_size=int,total_page=int64}} "课程题目列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/problems [get]
func (s *Server) GetCourseProblems(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("courseId"))
	if err != nil {
		utils.BadRequestResponse(c, "无效的课程ID")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	userIDValue, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	currentUserID, err := primitive.ObjectIDFromHex(userIDValue.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	roleValue, _ := c.Get("role")
	userRole, _ := roleValue.(string)
	if userRole != string(models.RoleAdmin) {
		ok, err := s.svc.CourseService.CanViewCourse(c.Request.Context(), courseID, currentUserID)
		if err != nil || !ok {
			utils.ForbiddenResponse(c, "无权查看该课程题库")
			return
		}
	}

	problems, total, err := s.svc.ProblemService.GetProblemsByCourseID(
		c.Request.Context(),
		courseID,
		page,
		pageSize,
		models.ProblemDifficulty(c.Query("difficulty")),
		c.QueryArray("tags"),
		&currentUserID,
	)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取课程题目失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"problems":   problems,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetProblemsForAdmin godoc
// @Summary      获取题目列表（管理员端）
// @Description  管理员端获取题目列表，可查看所有题目包括私有和草稿状态，支持按难度、标签筛选
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        difficulty query string false "难度" Enums(easy, medium, hard)
// @Param        tags query []string false "标签列表"
// @Success      200 {object} utils.Response{data=object{problems=[]models.ProblemList,total=int64,page=int,page_size=int,total_page=int64}} "题目列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未授权访问"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Router       /admin/problems [get]
// @Security     BearerAuth
func (s *Server) GetProblemsForAdmin(c *gin.Context) {
	// 检查用户是否为管理员
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足，只有管理员可以访问")
		return
	}

	// 参数解析和验证
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	difficulty := c.Query("difficulty")
	tags := c.QueryArray("tags")

	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 调用管理员端Service方法
	ctx := c.Request.Context()
	problems, total, err := s.svc.ProblemService.GetProblemsForAdmin(
		ctx,
		page,
		pageSize,
		models.ProblemDifficulty(difficulty),
		tags,
	)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取题目列表失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"problems":   problems,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetProblem godoc
// @Summary      获取题目详情
// @Description  根据题目ID获取完整题目信息
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path string true "题目ID"
// @Success      200 {object} utils.Response{data=models.Problem} "题目详情"
// @Failure      400 {object} map[string]interface{} "无效的题目ID"
// @Failure      401 {object} map[string]interface{} "需要登录"
// @Failure      404 {object} map[string]interface{} "题目不存在"
// @Router       /problem/{id} [get]
func (s *Server) GetProblem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	ctx := c.Request.Context()
	problem, err := s.svc.ProblemService.GetProblemByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "题目不存在")
		return
	}

	// 检查权限：非公开题目需要认证
	if !problem.IsPublic {
		_, exists := c.Get("user_id")
		if !exists {
			utils.UnauthorizedResponse(c, "需要登录才能查看此题目")
			return
		}
	}

	utils.SuccessResponse(c, problem)
}

// GetProblemDetail godoc
// @Summary      获取题目详情聚合信息
// @Description  获取题目基本信息和示例测试用例的聚合数据，用于前端题目详情页面展示
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "题目ID"
// @Success      200  {object}  utils.Response{data=models.ProblemDetailResponse}  "成功"
// @Failure      400  {object}  utils.Response  "请求参数错误"
// @Failure      401  {object}  utils.Response  "未授权访问"
// @Failure      404  {object}  utils.Response  "题目不存在"
// @Failure      500  {object}  utils.Response  "服务器内部错误"
// @Router       /problem/{id}/detail [get]
func (s *Server) GetProblemDetail(c *gin.Context) {
	// 参数验证：解析题目ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	// 数据操作：获取题目详情聚合信息
	ctx := c.Request.Context()
	problemDetail, err := s.svc.ProblemService.GetProblemDetail(ctx, id)
	if err != nil {
		// 业务逻辑：根据错误类型返回不同响应
		if errors.Is(err, models.ErrProblemNotFound) || errors.Is(err, models.ErrProblemInfoFetchFailed) {
			utils.NotFoundResponse(c, "题目不存在")
			return
		}
		if errors.Is(err, models.ErrSampleTestCaseFetchFailed) {
			utils.NotFoundResponse(c, "题目测试用例不存在")
			return
		}
		utils.InternalServerErrorResponse(c, "获取题目详情失败")
		return
	}

	// 权限检查：课程私有题只允许课程成员、创建者或管理员查看。
	if ok, needAuth := s.canViewProblem(c.Request.Context(), c, problemDetail.Problem); !ok {
		if needAuth {
			utils.UnauthorizedResponse(c, "需要登录才能查看此题目")
			return
		}
		utils.ForbiddenResponse(c, "无权查看此题目")
		return
	}

	// 成功响应
	utils.SuccessResponse(c, problemDetail)
}

// CreateProblem godoc
// @Summary      创建题目（教师/管理员）
// @Description  创建新题目，默认状态为草稿，草稿状态不能公开
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        request body models.CreateProblemRequest true "题目信息"
// @Success      200 {object} utils.Response{data=models.Problem} "创建成功"
// @Failure      400 {object} map[string]interface{} "请求参数错误或业务规则错误"
// @Failure      401 {object} map[string]interface{} "需要登录"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      500 {object} map[string]interface{} "创建失败"
// @Security     BearerAuth
// @Router       /problem [post]
func (s *Server) CreateProblem(c *gin.Context) {
	//检查权限 只有管理员和老师可以创建题目
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足，只有管理员和教师可以创建题目")
		return
	}

	// 2. 绑定和验证请求参数
	var req models.CreateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 3. 业务规则预检：草稿状态不能公开
	if req.Status != nil && *req.Status == models.StatusDraft {
		if req.IsPublic != nil && *req.IsPublic == true {
			utils.BadRequestResponse(c, "草稿状态的题目不能设为公开")
			return
		}
	}
	courseIDs, err := parseCourseObjectIDs(req.CourseIDs)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	// 4. 构建Problem对象
	problem := models.Problem{
		Title:        req.Title,
		Description:  req.Description,
		Input:        req.Input,
		Output:       req.Output,
		SampleInput:  req.SampleInput,
		SampleOutput: req.SampleOutput,
		Hint:         req.Hint,
		Source:       req.Source,
		Author:       req.Author,
		Difficulty:   req.Difficulty,
		Tags:         req.Tags,
		IsPublic:     req.EffectiveIsPublic(),
		CourseIDs:    courseIDs,
	}

	// 5. 设置可选字段（使用指针判断是否传入）
	if req.TimeLimit != nil {
		problem.TimeLimit = *req.TimeLimit
	}
	if req.MemoryLimit != nil {
		problem.MemoryLimit = *req.MemoryLimit
	}
	if req.Status != nil {
		problem.Status = *req.Status
	}

	// 6. 设置创建者
	userID, _ := c.Get("user_id")
	problem.CreatedBy, _ = primitive.ObjectIDFromHex(userID.(string))
	if err := s.ensureCourseProblemManagePermission(c.Request.Context(), userRole, problem.CreatedBy, courseIDs); err != nil {
		utils.ForbiddenResponse(c, err.Error())
		return
	}

	// 7. 调用Service层
	ctx := c.Request.Context()
	if err := s.svc.ProblemService.CreateProblem(ctx, &problem); err != nil {
		utils.InternalServerErrorResponse(c, "创建题目失败: "+err.Error())
		return
	}

	// 8. 返回成功响应
	utils.SuccessResponse(c, problem)
}

// BatchCreateProblems godoc
// @Summary      批量创建题目（教师/管理员）
// @Description  批量创建多个题目，单次最多50个，部分失败不影响整体创建
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        request body models.BatchCreateProblemsRequest true "批量题目信息"
// @Success      200 {object} utils.Response{data=models.BatchCreateProblemsResponse} "批量创建结果"
// @Failure      400 {object} map[string]interface{} "请求参数错误或业务规则错误"
// @Failure      401 {object} map[string]interface{} "需要登录"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      500 {object} map[string]interface{} "创建失败"
// @Security     BearerAuth
// @Router       /problems/batch [post]
func (s *Server) BatchCreateProblems(c *gin.Context) {
	// 1. 检查权限：只有管理员和教师可以批量创建题目
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足，只有管理员和教师可以批量创建题目")
		return
	}

	// 2. 绑定和验证请求参数
	var req models.BatchCreateProblemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 3. 业务规则预检：限制批量创建数量
	if len(req.Problems) == 0 {
		utils.BadRequestResponse(c, "批量创建题目列表不能为空")
		return
	}
	if len(req.Problems) > 50 {
		utils.BadRequestResponse(c, "单次批量创建题目数量不能超过50个")
		return
	}

	// 4. 获取当前用户ID
	userID, _ := c.Get("user_id")
	currentUserID, _ := primitive.ObjectIDFromHex(userID.(string))

	// 5. 业务规则预检：草稿状态不能公开
	for i := range req.Problems {
		if req.Problems[i].Status != nil && *req.Problems[i].Status == models.StatusDraft {
			if req.Problems[i].IsPublic != nil && *req.Problems[i].IsPublic {
				utils.BadRequestResponse(c, fmt.Sprintf("第%d个题目：草稿状态的题目不能设为公开", i+1))
				return
			}
		}
	}

	// 6. 构建Problem对象列表
	problems := make([]*models.Problem, len(req.Problems))
	for i, problemReq := range req.Problems {
		courseIDs, err := parseCourseObjectIDs(problemReq.CourseIDs)
		if err != nil {
			utils.BadRequestResponse(c, fmt.Sprintf("第%d个题目：%s", i+1, err.Error()))
			return
		}
		if err := s.ensureCourseProblemManagePermission(c.Request.Context(), userRole, currentUserID, courseIDs); err != nil {
			utils.ForbiddenResponse(c, fmt.Sprintf("第%d个题目：%s", i+1, err.Error()))
			return
		}
		problem := models.Problem{
			Title:        problemReq.Title,
			Description:  problemReq.Description,
			Input:        problemReq.Input,
			Output:       problemReq.Output,
			SampleInput:  problemReq.SampleInput,
			SampleOutput: problemReq.SampleOutput,
			Hint:         problemReq.Hint,
			Source:       problemReq.Source,
			Author:       problemReq.Author,
			Difficulty:   problemReq.Difficulty,
			Tags:         problemReq.Tags,
			IsPublic:     problemReq.EffectiveIsPublic(),
			CourseIDs:    courseIDs,
			CreatedBy:    currentUserID, // 设置创建者
		}

		// 设置可选字段（使用指针判断是否传入）
		if problemReq.TimeLimit != nil {
			problem.TimeLimit = *problemReq.TimeLimit
		}
		if problemReq.MemoryLimit != nil {
			problem.MemoryLimit = *problemReq.MemoryLimit
		}
		if problemReq.Status != nil {
			problem.Status = *problemReq.Status
		}
		problems[i] = &problem
	}

	// 7. 调用Service层批量创建
	ctx := c.Request.Context()
	response, err := s.svc.ProblemService.BatchCreateProblems(ctx, problems)
	if err != nil {
		utils.InternalServerErrorResponse(c, "批量创建题目失败: "+err.Error())
		return
	}

	// 6. 返回成功响应
	utils.SuccessResponse(c, response)
}

// UpdateProblem godoc
// @Summary      更新题目（教师/管理员）
// @Description  更新题目信息（管理员或题目创建者），草稿状态不能设为公开
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path string true "题目ID"
// @Param        request body models.UpdateProblemRequest true "更新的题目信息"
// @Success      200 {object} utils.Response{data=models.Problem} "更新成功"
// @Failure      400 {object} map[string]interface{} "请求参数错误或业务规则错误"
// @Failure      401 {object} map[string]interface{} "需要登录"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      404 {object} map[string]interface{} "题目不存在"
// @Failure      500 {object} map[string]interface{} "更新失败"
// @Security     BearerAuth
// @Router       /problem/{id} [put]
func (s *Server) UpdateProblem(c *gin.Context) {
	// 1. 检查权限：只有管理员和老师可以更新题目
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足，只有管理员和教师可以更新题目")
		return
	}

	// 2. 解析题目ID
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	// 3. 绑定和验证请求参数
	var req models.UpdateProblemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 4. 业务规则预检：草稿状态不能公开
	if req.Status != nil && *req.Status == models.StatusDraft {
		if req.IsPublic != nil && *req.IsPublic == true {
			utils.BadRequestResponse(c, "草稿状态的题目不能设为公开")
			return
		}
	}
	if req.CourseIDs != nil {
		courseIDs, err := parseCourseObjectIDs(*req.CourseIDs)
		if err != nil {
			utils.BadRequestResponse(c, err.Error())
			return
		}
		userID, _ := c.Get("user_id")
		currentUserID, _ := primitive.ObjectIDFromHex(userID.(string))
		if err := s.ensureCourseProblemManagePermission(c.Request.Context(), userRole, currentUserID, courseIDs); err != nil {
			utils.ForbiddenResponse(c, err.Error())
			return
		}
	}

	// 5. 获取现有题目（用于权限检查）
	ctx := c.Request.Context()
	problem, err := s.svc.ProblemService.GetProblemByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "题目不存在")
		return
	}

	// 6. 检查权限：只有创建者或管理员可以更新
	userID, _ := c.Get("user_id")
	currentUserID, _ := primitive.ObjectIDFromHex(userID.(string))
	if userRole != string(models.RoleAdmin) && problem.CreatedBy != currentUserID {
		utils.ForbiddenResponse(c, "只能更新自己创建的题目")
		return
	}

	// 7. 调用 Service 层更新（直接传递请求对象）
	if err := s.svc.ProblemService.UpdateProblem(ctx, id, &req); err != nil {
		utils.InternalServerErrorResponse(c, "更新题目失败: "+err.Error())
		return
	}

	// 8. 重新获取更新后的题目（用于返回）
	updatedProblem, err := s.svc.ProblemService.GetProblemByID(ctx, id)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取更新后的题目失败: "+err.Error())
		return
	}

	// 9. 返回成功响应
	utils.SuccessResponse(c, updatedProblem)
}

// DeleteProblem godoc
// @Summary      删除题目（管理员/创建者）
// @Description  删除指定题目；管理员可删除全部题目，教师可删除自己创建的题目
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path string true "题目ID"
// @Success      200 {object} utils.Response{data=object{message=string}} "删除成功"
// @Failure      400 {object} models.ErrorResponse "无效的题目ID"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "删除失败"
// @Security     BearerAuth
// @Router       /problem/{id} [delete]
func (s *Server) DeleteProblem(c *gin.Context) {
	// 检查权限：管理员可删除全部题目，教师只能删除自己创建的题目。
	role, exists := c.Get("role")
	if !exists {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	ctx := c.Request.Context()
	if userRole != string(models.RoleAdmin) {
		problem, err := s.svc.ProblemService.GetProblemByID(ctx, id)
		if err != nil {
			utils.NotFoundResponse(c, "题目不存在")
			return
		}
		userID, _ := c.Get("user_id")
		currentUserID, _ := primitive.ObjectIDFromHex(userID.(string))
		if problem.CreatedBy != currentUserID {
			utils.ForbiddenResponse(c, "只能删除自己创建的题目")
			return
		}
	}
	if err := s.svc.ProblemService.DeleteProblem(ctx, id); err != nil {
		utils.InternalServerErrorResponse(c, "删除题目失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// parseCourseObjectIDs 将请求中的课程ID字符串转换为Mongo ObjectID并去重。
func parseCourseObjectIDs(rawIDs []string) ([]primitive.ObjectID, error) {
	if len(rawIDs) == 0 {
		return []primitive.ObjectID{}, nil
	}
	seen := make(map[primitive.ObjectID]struct{}, len(rawIDs))
	courseIDs := make([]primitive.ObjectID, 0, len(rawIDs))
	for _, rawID := range rawIDs {
		rawID = strings.TrimSpace(rawID)
		if rawID == "" {
			continue
		}
		courseID, err := primitive.ObjectIDFromHex(rawID)
		if err != nil {
			return nil, fmt.Errorf("无效的课程ID: %s", rawID)
		}
		if _, ok := seen[courseID]; ok {
			continue
		}
		seen[courseID] = struct{}{}
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, nil
}

// currentProblemViewer 从中间件上下文或可选Bearer Token中解析当前访问者。
func currentProblemViewer(c *gin.Context) (*primitive.ObjectID, string) {
	if userID, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userID.(string); ok {
			if objID, err := primitive.ObjectIDFromHex(userIDStr); err == nil {
				if role, ok := c.Get("role"); ok {
					if roleStr, ok := role.(string); ok {
						return &objID, roleStr
					}
				}
				return &objID, ""
			}
		}
	}

	authHeader := c.GetHeader("Authorization")
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" || tokenString == authHeader {
		return nil, ""
	}
	claims, err := middleware.ParseToken(tokenString)
	if err != nil {
		return nil, ""
	}
	objID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, ""
	}
	return &objID, claims.Role
}

// canViewProblem 判断当前请求是否可查看题目详情。
func (s *Server) canViewProblem(ctx context.Context, c *gin.Context, problem *models.Problem) (bool, bool) {
	if problem == nil {
		return false, false
	}
	if problem.IsPublic {
		return true, false
	}
	userID, role := currentProblemViewer(c)
	if userID == nil {
		return false, true
	}
	if role == string(models.RoleAdmin) || problem.CreatedBy == *userID {
		return true, false
	}
	for _, courseID := range problem.CourseIDs {
		ok, err := s.svc.CourseService.CanViewCourse(ctx, courseID, *userID)
		if err == nil && ok {
			return true, false
		}
	}
	return false, false
}

// ensureCourseProblemManagePermission 校验题目挂载课程时的管理边界。
func (s *Server) ensureCourseProblemManagePermission(ctx context.Context, role string, userID primitive.ObjectID, courseIDs []primitive.ObjectID) error {
	if len(courseIDs) == 0 || role == string(models.RoleAdmin) {
		return nil
	}
	for _, courseID := range courseIDs {
		ok, err := s.svc.CourseService.CanManageCourse(ctx, courseID, userID)
		if err != nil {
			return fmt.Errorf("课程不存在或无权操作: %s", courseID.Hex())
		}
		if !ok {
			return fmt.Errorf("无权将题目挂载到课程: %s", courseID.Hex())
		}
	}
	return nil
}

// RunCode godoc
// @Summary      运行代码测试
// @Description  在线运行代码进行测试（非提交），有频率限制
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path string true "题目ID"
// @Param        request body service.RunCodeRequest true "代码和语言"
// @Success      200 {object} utils.Response{data=models.RunCodeResponse} "运行结果"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      404 {object} models.ErrorResponse "题目不存在"
// @Failure      500 {object} models.ErrorResponse "运行失败"
// @Security     BearerAuth
// @Router       /problem/{id}/run [post]
func (s *Server) RunCode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	var req service.RunCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	resp, err := s.svc.ProblemService.RunCode(ctx, id, &req)
	if err != nil {
		utils.InternalServerErrorResponse(c, "运行代码失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, resp)
}

// SearchProblems godoc
// @Summary      搜索题目
// @Description  根据关键词、难度、标签搜索题目
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        keyword query string false "关键词"
// @Param        difficulty query string false "难度" Enums(easy, medium, hard)
// @Param        tags query []string false "标签列表"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        in_favorite query bool false "是否仅在收藏中搜索" default(false)
// @Success      200 {object} utils.Response{data=object{problems=[]models.ProblemList,total=int64,page=int,page_size=int,total_page=int64}} "搜索结果"
// @Failure      500 {object} models.ErrorResponse "搜索失败"
// @Router       /problem/search [get]
func (s *Server) SearchProblems(c *gin.Context) {
	keyword := c.Query("keyword")
	difficulty := c.Query("difficulty")
	tags := c.QueryArray("tags")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	inFavorite, _ := strconv.ParseBool(c.DefaultQuery("in_favorite", "false"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取用户ID（用于用户状态查询）
	var userObjectID *primitive.ObjectID
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// 有token，尝试解析
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString != authHeader {
			// token格式正确，解析token
			if claims, err := middleware.ParseToken(tokenString); err == nil {
				// token有效，提取用户ID
				if objID, err := primitive.ObjectIDFromHex(claims.UserID); err == nil {
					userObjectID = &objID
				}
			}
		}
	}

	ctx := c.Request.Context()
	problems, total, err := s.svc.ProblemService.SearchProblems(
		ctx,
		keyword,
		models.ProblemDifficulty(difficulty),
		tags,
		page,
		pageSize,
		userObjectID,
		inFavorite,
	)
	if err != nil {
		utils.InternalServerErrorResponse(c, "搜索题目失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"problems":   problems,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetDailyProblem godoc
// @Summary      获取每日推荐题目
// @Description  获取当日推荐的题目，全局统一推荐
// @Tags         题目
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=object{daily_problem=models.Problem,message=string}} "每日推荐题目"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Router       /daily-problem [get]
func (s *Server) GetDailyProblem(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 检查是否有登录用户
	var userObjectID *primitive.ObjectID
	if userIDInterface, exists := c.Get("user_id"); exists {
		if userIDStr, ok := userIDInterface.(string); ok {
			if id, err := primitive.ObjectIDFromHex(userIDStr); err == nil {
				userObjectID = &id
			}
		}
	}

	// 2. 获取每日推荐题目（已经包含用户状态）
	dailyProblem, err := s.svc.DailyProblemService.GetDailyProblem(ctx, userObjectID)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取每日推荐题目失败: "+err.Error())
		return
	}

	// 3. 返回结果
	utils.SuccessResponse(c, gin.H{
		"daily_problem": dailyProblem,
		"message":       "每日推荐题目获取成功",
	})
}

// ToggleFavoriteRequest 收藏/取消收藏请求
type ToggleFavoriteRequest struct {
	IsFavorite *bool `json:"is_favorite" binding:"required"` // 1: 收藏, 0: 取消收藏
}

// ToggleFavorite godoc
// @Summary      收藏或取消收藏题目
// @Description  使用同一个接口收藏或取消收藏题目
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        id path string true "题目ID"
// @Param        request body ToggleFavoriteRequest true "收藏状态"
// @Success      200 {object} utils.Response "操作成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未授权"
// @Failure      500 {object} models.ErrorResponse "服务器内部错误"
// @Security     BearerAuth
// @Router       /problem/{id}/favorite [post]
func (s *Server) ToggleFavorite(c *gin.Context) {
	// 1. 获取题目ID
	problemIDStr := c.Param("id")
	problemID, err := primitive.ObjectIDFromHex(problemIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	// 2. 解析请求体
	var req ToggleFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "无效的请求参数")
		return
	}

	// 3. 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未授权访问")
		return
	}

	// 将userID从字符串转换为primitive.ObjectID
	userIDObj, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.InternalServerErrorResponse(c, "无效的用户ID格式")
		return
	}

	// 4. 调用服务层切换收藏状态
	ctx := c.Request.Context()
	err = s.svc.ProblemService.ToggleFavorite(ctx, userIDObj, problemID, *req.IsFavorite)
	if err != nil {
		utils.InternalServerErrorResponse(c, "操作失败: "+err.Error())
		return
	}

	// 5. 返回成功响应
	action := "收藏"
	if !*req.IsFavorite {
		action = "取消收藏"
	}
	utils.SuccessResponse(c, gin.H{
		"message": fmt.Sprintf("%s成功", action),
	})
}

// GetUserFavoriteProblems godoc
// @Summary      获取用户收藏题目列表
// @Description  获取当前用户收藏的所有题目，支持按难度筛选
// @Tags         题目
// @Accept       json
// @Produce      json
// @Param        page query int false "页码，默认1"
// @Param        page_size query int false "每页数量，默认10"
// @Param        difficulty query string false "难度筛选，可选值：easy, medium, hard"
// @Success      200 {object} utils.Response{data=object{problems=[]models.ProblemList,total=int64,page=int,page_size=int,total_page=int}} "收藏题目列表"
// @Failure      401 {object} models.ErrorResponse "未授权"
// @Failure      500 {object} models.ErrorResponse "服务器内部错误"
// @Security     BearerAuth
// @Router       /problem/favorite [get]
func (s *Server) GetUserFavoriteProblems(c *gin.Context) {
	// 1. 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 获取难度筛选参数
	difficultyStr := c.Query("difficulty")
	var difficulty models.ProblemDifficulty
	if difficultyStr != "" {
		difficulty = models.ProblemDifficulty(difficultyStr)
		// 验证难度值是否合法
		if difficulty != models.DifficultyEasy && difficulty != models.DifficultyMedium && difficulty != models.DifficultyHard {
			utils.BadRequestResponse(c, "无效的难度值，可选值：easy, medium, hard")
			return
		}
	}

	// 2. 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未授权访问")
		return
	}

	// 将userID从字符串转换为primitive.ObjectID
	userIDObj, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.InternalServerErrorResponse(c, "无效的用户ID格式")
		return
	}

	// 3. 调用服务层获取收藏题目
	ctx := c.Request.Context()
	problems, total, err := s.svc.ProblemService.GetUserFavoriteProblems(ctx, userIDObj, difficulty, page, pageSize)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取收藏题目失败: "+err.Error())
		return
	}

	// 4. 构建响应
	utils.SuccessResponse(c, gin.H{
		"problems":   problems,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetNextProblem godoc
// @Summary 获取下一题
// @Description 获取当前题目的下一题，如果提供了任务列表ID和题号，则返回任务列表中的下一题
// @Tags 题目
// @Accept json
// @Produce json
// @Param id path int true "当前题目ID"
// @Param task_id query string false "任务列表ID"
// @Param current_idx query int false "当前任务列表中的题号（从0开始）"
// @Success 200 {object} utils.Response{data=models.Problem} "下一题详情"
// @Failure 400 {object} utils.Response "参数错误"
// @Failure 404 {object} utils.Response "未找到下一题"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /problem/{id}/next [get]
func (s *Server) GetNextProblem(c *gin.Context) {
	// 1. 参数获取与校验
	currentUniqueID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	// 获取查询参数
	taskID := c.Query("task_id")
	currentIdxStr := c.DefaultQuery("current_idx", "-1")
	currentIdx, err := strconv.Atoi(currentIdxStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的当前题号")
		return
	}

	// 2. 调用Service层获取下一题
	ctx := context.Background()
	problem, err := s.svc.ProblemService.GetNextProblem(ctx, currentUniqueID, taskID, currentIdx)
	if err != nil {
		if errors.Is(err, models.ErrProblemNotFound) {
			utils.NotFoundResponse(c, "没有找到下一题")
			return
		}
		utils.BadRequestResponse(c, err.Error())
		return
	}

	// 3. 返回响应
	utils.SuccessResponse(c, problem)
}
