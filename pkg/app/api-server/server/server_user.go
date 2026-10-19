/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: server_user.go
@Description: 用户相关路由处理
*/

package server

import (
	"strconv"
	"strings"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterUser 注册用户相关路由
func (s *Server) RegisterUser(g *gin.RouterGroup) {
	userGroup := g.Group("/user")
	{
		// 公开路由
		userGroup.POST("/register", s.CreateUser) // 用户注册
		userGroup.POST("/login", s.LoginUser)     // 用户登录
	}

	// 需要认证的路由（使用 Group("") 而非 Group("/")，避免 basePath 变成 /user/ 触发尾斜杠重定向）
	securedGroup := userGroup.Group("").Use(middleware.JWTMiddleware())
	{
		securedGroup.GET("/profile", s.GetUserProfile)         // 获取用户资料
		securedGroup.PUT("/profile", s.UpdateUserProfile)      // 更新用户资料
		securedGroup.GET("", s.GetUsers)                       // 获取用户列表（管理员）
		securedGroup.POST("/teachers", s.PageQueryAllTeachers) // 分页查询所有教师信息
		securedGroup.GET("/:id", s.GetUserByID)                // 获取指定用户
		securedGroup.PUT("/:id", s.UpdateUser)                 // 更新用户信息
		securedGroup.DELETE("/:id", s.DeleteUser)              // 删除用户
		securedGroup.GET("/tasks", s.PageQueryUserTasks)       // 分页查询用户任务列表
	}
}

// CreateUser godoc
// @Summary      用户注册
// @Description  注册新用户账号（学生/教师/学习者）
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        request body models.User true "用户信息"
// @Success      200 {object} utils.Response{data=object{message=string,user=models.UserProfile}} "注册成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      500 {object} models.ErrorResponse "注册失败"
// @Router       /user/register [post]
func (s *Server) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 自定义验证：当角色为 student 时，MajorClassID 必须填写
	if user.Role == models.RoleStudent && user.MajorClassID == "" {
		utils.BadRequestResponse(c, "请求参数错误: 学生角色必须填写专业班级ID")
		return
	}

	ctx := c.Request.Context()
	if err := s.svc.UserService.CreateUser(ctx, &user); err != nil {
		utils.InternalServerErrorResponse(c, "注册失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "注册成功",
		"user":    user.ToProfile(),
	})
}

// LoginUser godoc
// @Summary      用户登录
// @Description  使用学号/工号和密码登录系统
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        request body models.LoginRequest true "登录信息"
// @Success      200 {object} utils.Response{data=object{message=string,token=string,user=models.UserProfile}} "登录成功，返回token和用户信息"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "账号或密码错误"
// @Router       /user/login [post]
func (s *Server) LoginUser(c *gin.Context) {
	var loginReq models.LoginRequest

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := s.svc.UserService.ValidateUser(ctx, loginReq.StudentID, loginReq.Password)
	if err != nil {
		utils.UnauthorizedResponse(c, err.Error())
		return
	}

	// 生成 JWT Token
	token, err := middleware.GenerateToken(user.ID.Hex(), user.Username, string(user.Role))
	if err != nil {
		utils.InternalServerErrorResponse(c, "生成令牌失败")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "登录成功",
		"token":   token,
		"user":    user.ToProfile(),
	})
}

// GetUserProfile godoc
// @Summary      获取当前用户资料
// @Description  获取当前登录用户的个人资料信息
// @Tags         用户
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=models.UserProfile} "用户资料"
// @Failure      401 {object} models.ErrorResponse "未认证用户"
// @Failure      404 {object} models.ErrorResponse "用户不存在"
// @Security     BearerAuth
// @Router       /user/profile [get]
func (s *Server) GetUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未认证用户")
		return
	}

	id, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	ctx := c.Request.Context()
	user, err := s.svc.UserService.GetUserByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "用户不存在")
		return
	}

	utils.SuccessResponse(c, user.ToProfile())
}

// UpdateUserProfile godoc
// @Summary      更新当前用户资料
// @Description  更新当前登录用户的个人资料（姓名、邮箱、学校等）
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        request body object{real_name=string,email=string,bio=string,school=string,major=string,grade=string,class=string,phone=string} true "更新信息"
// @Success      200 {object} utils.Response{data=models.UserProfile} "更新后的用户资料"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "未认证用户"
// @Failure      404 {object} models.ErrorResponse "用户不存在"
// @Failure      500 {object} models.ErrorResponse "更新失败"
// @Security     BearerAuth
// @Router       /user/profile [put]
func (s *Server) UpdateUserProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未认证用户")
		return
	}

	id, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	var updateReq models.UpdateProfileRequest

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := s.svc.UserService.GetUserByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "用户不存在")
		return
	}

	// 更新用户信息
	user.RealName = updateReq.RealName
	user.Email = updateReq.Email
	user.Bio = updateReq.Bio
	user.School = updateReq.School
	user.Major = updateReq.Major
	user.Grade = updateReq.Grade
	user.Class = updateReq.Class
	user.Phone = updateReq.Phone

	if err := s.svc.UserService.UpdateUser(ctx, user); err != nil {
		utils.InternalServerErrorResponse(c, "更新失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, user.ToProfile())
}

// GetUsers godoc
// @Summary      获取用户列表（管理员）
// @Description  分页获取用户列表，可按角色筛选（仅管理员）
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(10)
// @Param        role query string false "用户角色筛选" Enums(admin, teacher, student)
// @Param        is_active query bool false "是否启用账号筛选：true/false，不传则不过滤"
// @Success      200 {object} utils.Response{data=object{users=[]models.UserProfile,total=int64,page=int,page_size=int,total_page=int64}} "用户列表"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /user [get]
func (s *Server) GetUsers(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	roleStr := c.Query("role")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var isActive *bool
	switch strings.ToLower(strings.TrimSpace(c.Query("is_active"))) {
	case "true", "1", "yes":
		v := true
		isActive = &v
	case "false", "0", "no":
		v := false
		isActive = &v
	}

	ctx := c.Request.Context()
	users, total, err := s.svc.UserService.GetUsers(ctx, page, pageSize, models.UserRole(roleStr), isActive)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取用户列表失败: "+err.Error())
		return
	}

	// 转换为用户资料
	profiles := make([]*models.UserProfile, len(users))
	for i, user := range users {
		profiles[i] = user.ToProfile()
	}

	utils.SuccessResponse(c, gin.H{
		"users":      profiles,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// PageQueryAllTeachers godoc
// @Summary      分页查询所有教师信息
// @Description  分页查询所有教师信息，支持按 username 模糊查询（不区分大小写）。管理员可按 is_active 筛选；非管理员仅返回启用中的教师。
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        request body models.PageQueryAllTeachersRequest true "分页查询参数"
// @Success      200 {object} utils.Response{data=models.PageQueryCourseTeachersResponse} "教师列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      500 {object} models.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /user/teachers [post]
func (s *Server) PageQueryAllTeachers(c *gin.Context) {
	var req models.PageQueryAllTeachersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	roleVal, ok := c.Get("role")
	isAdmin := ok && roleVal.(string) == string(models.RoleAdmin)

	ctx := c.Request.Context()
	teachers, err := s.svc.UserService.PageQueryAllTeachers(ctx, &req, isAdmin)
	if err != nil {
		utils.InternalServerErrorResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, teachers)
}

// GetUserByID godoc
// @Summary      获取指定用户信息
// @Description  根据用户ID获取用户信息（管理员或本人）
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        id path string true "用户ID"
// @Success      200 {object} utils.Response{data=models.UserProfile} "用户信息"
// @Failure      400 {object} models.ErrorResponse "无效的用户ID"
// @Failure      401 {object} models.ErrorResponse "未认证用户"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      404 {object} models.ErrorResponse "用户不存在"
// @Security     BearerAuth
// @Router       /user/{id} [get]
func (s *Server) GetUserByID(c *gin.Context) {
	// 权限校验：仅管理员或用户本人可查看
	currentUserID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未认证用户")
		return
	}

	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	// 检查是否为管理员或查询自己的信息
	role, roleExists := c.Get("role")
	isAdmin := roleExists && role.(string) == string(models.RoleAdmin)
	isSelf := currentUserID.(string) == idStr

	if !isAdmin && !isSelf {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	ctx := c.Request.Context()
	user, err := s.svc.UserService.GetUserByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "用户不存在")
		return
	}

	utils.SuccessResponse(c, user.ToProfile())
}

// UpdateUser godoc
// @Summary      更新用户信息（管理员）
// @Description  管理员更新用户角色和激活状态
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        id path string true "用户ID"
// @Param        request body models.UpdateUserRequest true "更新信息"
// @Success      200 {object} utils.Response{data=models.UserProfile} "更新后的用户信息"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      404 {object} models.ErrorResponse "用户不存在"
// @Failure      500 {object} models.ErrorResponse "更新失败"
// @Security     BearerAuth
// @Router       /user/{id} [put]
func (s *Server) UpdateUser(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	var updateReq models.UpdateUserRequest

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	user, err := s.svc.UserService.GetUserByID(ctx, id)
	if err != nil {
		utils.NotFoundResponse(c, "用户不存在")
		return
	}

	// 更新用户信息
	if updateReq.Role != "" {
		user.Role = models.UserRole(updateReq.Role)
	}
	if updateReq.IsActive != nil {
		user.IsActive = *updateReq.IsActive
	}

	if err := s.svc.UserService.UpdateUser(ctx, user); err != nil {
		utils.InternalServerErrorResponse(c, "更新失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, user.ToProfile())
}

// DeleteUser godoc
// @Summary      删除用户（管理员）
// @Description  管理员删除指定用户
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        id path string true "用户ID"
// @Success      200 {object} utils.Response{data=object{message=string}} "删除成功"
// @Failure      400 {object} models.ErrorResponse "无效的用户ID"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "删除失败"
// @Security     BearerAuth
// @Router       /user/{id} [delete]
func (s *Server) DeleteUser(c *gin.Context) {
	// 检查管理员权限
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	ctx := c.Request.Context()
	if err := s.svc.UserService.DeleteUser(ctx, id); err != nil {
		utils.InternalServerErrorResponse(c, "删除失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{"message": "删除成功"})
}

// PageQueryUserTasks godoc
// @Summary      分页查询用户任务列表
// @Description  分页查询当前用户的所有班级任务，支持按任务类型和状态筛选
// @Tags         用户
// @Accept       json
// @Produce      json
// @Param        page query int false "页码，默认1"
// @Param        pageSize query int false "每页大小，默认10"
// @Param        type query int false "任务类型：1-题单，2-视频"
// @Param        state query int false "任务状态：0-未完成, 1-已完成"
// @Param        status query int false "任务状态：0-未开始, 1-进行中, 2-已结束"
// @Success      200 {object} models.PageQueryUserTasksResponse "任务列表"
// @Failure      400 {object} models.ErrorResponse "参数错误"
// @Failure      401 {object} models.ErrorResponse "未授权"
// @Failure      500 {object} models.ErrorResponse "查询失败"
// @Security     BearerAuth
// @Router       /user/tasks [get]
func (s *Server) PageQueryUserTasks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "未授权")
		return
	}

	var req models.PageQueryUserTasksFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestResponse(c, "参数错误: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	resp, err := s.svc.ClazzService.PageQueryUserTasks(ctx, userID.(string), &req)
	if err != nil {
		utils.InternalServerErrorResponse(c, "查询任务失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, resp)
}
