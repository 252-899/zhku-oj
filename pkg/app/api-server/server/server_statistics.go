/*
@Author: omenkk7
@Date: 2025/10/7
@Description: 统计相关路由处理
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterStatistics 注册统计相关路由
func (s *Server) RegisterStatistics(g *gin.RouterGroup) {
	statsGroup := g.Group("/statistics")

	// 需要认证的路由（使用 Group("") 避免 basePath 变成 /statistics/ 引发尾斜杠重定向）
	securedGroup := statsGroup.Group("").Use(middleware.JWTMiddleware())
	{
		securedGroup.GET("/user", s.GetUserStatistics)         // 获取当前用户统计
		securedGroup.GET("/user/:id", s.GetUserStatisticsByID) // 获取指定用户统计（管理员）
		securedGroup.GET("/system", s.GetSystemStatistics)     // 获取系统统计（管理员）
	}
}

// RegisterProblemStats 注册题目统计相关路由（公开访问）
func (s *Server) RegisterProblemStats(g *gin.RouterGroup) {
	problemGroup := g.Group("/problems")
	{
		// 公开路由 - 题目难度统计
		problemGroup.GET("/difficulty-stats", s.GetProblemDifficultyStats)
	}
}

// GetUserStatistics godoc
// @Summary      获取当前用户统计
// @Description  获取当前登录用户的提交统计信息
// @Tags         统计
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=models.UserStatsResponse} "用户统计信息"
// @Failure      401 {object} models.ErrorResponse "需要登录"
// @Failure      400 {object} models.ErrorResponse "无效的用户ID"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /statistics/user [get]
func (s *Server) GetUserStatistics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	ctx := c.Request.Context()
	stats, err := s.svc.StatisticsService.GetUserStatistics(ctx, userObjID)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取用户统计失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, stats)
}

// GetUserStatisticsByID godoc
// @Summary      获取指定用户统计（管理员）
// @Description  管理员查看指定用户的提交统计信息
// @Tags         统计
// @Accept       json
// @Produce      json
// @Param        id path string true "用户ID"
// @Success      200 {object} utils.Response{data=models.UserStatsResponse} "用户统计信息"
// @Failure      400 {object} models.ErrorResponse "无效的用户ID"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /statistics/user/{id} [get]
func (s *Server) GetUserStatisticsByID(c *gin.Context) {
	// 检查权限：只有管理员可以查看其他用户的统计
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	idStr := c.Param("id")
	userObjID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	ctx := c.Request.Context()
	stats, err := s.svc.StatisticsService.GetUserStatistics(ctx, userObjID)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取用户统计失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, stats)
}

// GetSystemStatistics godoc
// @Summary      获取系统统计（管理员）
// @Description  管理员查看系统整体统计信息
// @Tags         统计
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=models.SystemStatsResponse} "系统统计信息"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /statistics/system [get]
func (s *Server) GetSystemStatistics(c *gin.Context) {
	// 检查权限：只有管理员可以查看系统统计
	role, exists := c.Get("role")
	if !exists || role.(string) != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	ctx := c.Request.Context()
	stats, err := s.svc.StatisticsService.GetSystemStatistics(ctx)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取系统统计失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, stats)
}

// GetProblemDifficultyStats godoc
// @Summary      获取题目难度分布统计
// @Description  获取各难度题目数量的轻量级统计
// @Tags         统计
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=models.DifficultyStatsResponse} "难度统计"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Router       /problems/difficulty-stats [get]
func (s *Server) GetProblemDifficultyStats(c *gin.Context) {
	ctx := c.Request.Context()

	stats, err := s.svc.StatisticsService.GetProblemDifficultyStats(ctx)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取题目难度统计失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, stats)
}
