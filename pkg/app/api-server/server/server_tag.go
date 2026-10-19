/*
@Author: xiamoqi
@Date: 2026/2/3
@Name: server_tag.go
@Description: 标签相关路由处理
*/

package server

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// RegisterTag 标签相关路由
func (s *Server) RegisterTag(g *gin.RouterGroup) {
	// 标签模块路由组：/api/v1/tags
	tagGroup := g.Group("/tags")
	{
		tagGroup.GET("", s.GetTagList)             // 获取标签列表（分页/筛选）
		tagGroup.GET("/:id", s.GetTag)             // 根据ID获取标签详情
		tagGroup.GET("/public", s.GetPublicTagList) // 获取所有标签（给前端下拉表单用）

		securedGroup := tagGroup.Group("").Use(middleware.JWTMiddleware())
		{
			securedGroup.POST("", s.CreateTag)                 // 创建标签
			securedGroup.PUT("/:id", s.UpdateTag)              // 更新标签
			securedGroup.DELETE("/:id", s.DeleteTag)           // 删除标签
			securedGroup.POST("/init", s.InitTagsFromProblems) // 初始化标签
		}
	}
}

// InitTagsFromProblems godoc
// @Summary      初始化标签库
// @Description  从现有题目中提取所有标签，自动创建标签记录（仅管理员）
// @Tags         标签
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]interface{} "初始化成功提示"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      500 {object} map[string]interface{} "初始化失败"
// @Security     BearerAuth
// @Router       /tags/init [post]
func (s *Server) InitTagsFromProblems(c *gin.Context) {
	// 获取用户ID
	userIDStr, exists := c.Get("user_id")
	if !exists || userIDStr.(string) == "" {
		utils.UnauthorizedResponse(c, "未获取到当前登录用户ID")
		return
	}

	// 权限校验
	role, roleExists := c.Get("role")
	if !roleExists {
		utils.UnauthorizedResponse(c, "需要登录才能操作")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足，仅管理员可更新标签")
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", userIDStr.(string))
	err := s.svc.InitTagsFromProblems(ctx)
	if err != nil {
		utils.InternalServerErrorResponse(c, "初始化标签库失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "标签库初始化成功，已从现有题目中提取并创建所有标签",
	})
}

// GetTagList godoc
// @Summary      获取标签列表
// @Description  分页获取标签列表
// @Tags         标签
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(20)
// @Param        keyword query string false "标签名称关键词"
// @Success      200 {object} map[string]interface{} "标签列表及分页信息"
// @Failure      500 {object} map[string]interface{} "获取标签列表失败"
// @Router       /tags [get]
func (s *Server) GetTagList(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	keyword := c.Query("keyword")

	// 分页参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 构建查询请求参数
	queryReq := &models.TagQueryRequest{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
	}

	// 调用service层获取标签列表
	ctx := c.Request.Context()
	tags, total, err := s.svc.GetTagList(ctx, queryReq)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取标签列表失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, gin.H{
		"tags":       tags,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetTag godoc
// @Summary      获取标签详情
// @Description  根据标签ID获取标签完整信息
// @Tags         标签
// @Accept       json
// @Produce      json
// @Param        id path string true "标签ID"
// @Success      200 {object} models.Tag "标签详情"
// @Failure      400 {object} map[string]interface{} "无效的标签ID"
// @Failure      404 {object} map[string]interface{} "标签不存在"
// @Failure      500 {object} utils.Response "查询标签失败"
// @Router       /tags/{id} [get]
func (s *Server) GetTag(c *gin.Context) {
	// 提取并校验标签ID
	idStr := c.Param("id")
	tagID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的标签ID")
		return
	}

	// 调用service层获取标签详情
	ctx := c.Request.Context()
	tag, err := s.svc.GetTagByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) || strings.Contains(err.Error(), "不存在") {
			utils.NotFoundResponse(c, "标签不存在")
			return
		}
		utils.InternalServerErrorResponse(c, "查询标签失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, tag)
}

// CreateTag godoc
// @Summary      创建标签（管理员）
// @Description  管理员创建新标签（仅管理员权限，标签描述可选）
// @Tags         标签
// @Accept       json
// @Produce      json
// @Param        request body models.CreateTagRequest true "标签创建信息"
// @Success      200 {object} models.Tag "创建成功的标签信息"
// @Failure      400 {object} map[string]interface{} "请求参数错误"
// @Failure      401 {object} utils.Response "未授权访问（需要登录）"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      500 {object} map[string]interface{} "创建标签失败"
// @Security     BearerAuth
// @Router       /tags [post]
func (s *Server) CreateTag(c *gin.Context) {
	// 权限校验
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录才能操作")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足，仅管理员可创建标签")
		return
	}

	// 绑定请求参数
	var req models.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 仅校验标签名称
	if req.Name == "" {
		utils.BadRequestResponse(c, "标签名称不能为空")
		return
	}

	// 获取当前登录用户ID
	userIDStr, userExists := c.Get("user_id")
	if !userExists || userIDStr.(string) == "" {
		utils.UnauthorizedResponse(c, "未获取到当前登录用户ID")
		return
	}
	createBy, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		utils.BadRequestResponse(c, "无效的用户ID")
		return
	}

	// 转换为 models.Tag
	tag := &models.Tag{
		Name:      req.Name,
		Desc:      req.Desc,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: createBy,
	}

	// 调用service层创建标签
	ctx := c.Request.Context()
	if err := s.svc.CreateTag(ctx, tag); err != nil {
		utils.InternalServerErrorResponse(c, "创建标签失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, tag)
}

// UpdateTag godoc
// @Summary      更新标签（管理员）
// @Description  管理员更新标签信息（仅管理员权限，标签描述可选）
// @Tags         标签
// @Accept       json
// @Produce      json
// @Param        id path string true "标签ID"
// @Param        request body models.UpdateTagRequest true "标签更新信息"
// @Success      200 {object} models.Tag "更新成功的标签信息"
// @Failure      400 {object} map[string]interface{} "请求参数错误"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      404 {object} map[string]interface{} "标签不存在"
// @Failure      500 {object} map[string]interface{} "更新标签失败"
// @Security     BearerAuth
// @Router       /tags/{id} [put]
func (s *Server) UpdateTag(c *gin.Context) {
	// 权限校验
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录才能操作")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足，仅管理员可更新标签")
		return
	}

	// 提取并校验标签ID
	idStr := c.Param("id")
	tagID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的标签ID")
		return
	}

	// 绑定请求参数
	var req models.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	// 先查询标签是否存在
	ctx := c.Request.Context()
	existingTag, err := s.svc.GetTagByID(ctx, tagID)
	if err != nil {
		utils.NotFoundResponse(c, err.Error())
		return
	}

	// 标记是否有实际更新内容
	hasUpdate := false
	// 校验名称是否有变更
	if req.Name != "" && req.Name != existingTag.Name {
		hasUpdate = true
	}
	// 校验描述是否有变更
	if req.Desc != existingTag.Desc {
		hasUpdate = true
	}
	// 无任何更新内容，直接返回400错误
	if !hasUpdate {
		utils.BadRequestResponse(c, "无有效更新内容，标签名称和描述均与原内容一致")
		return
	}

	// 调用service层更新标签
	err = s.svc.UpdateTag(ctx, tagID, &req)
	if err != nil {
		utils.InternalServerErrorResponse(c, "更新标签失败: "+err.Error())
		return
	}

	// 重新查询标签，获取最新信息返回
	updatedTag, err := s.svc.GetTagByID(ctx, tagID)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取更新后标签失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, updatedTag)
}

// DeleteTag godoc
// @Summary      删除标签（管理员）
// @Description  管理员删除指定标签（仅管理员权限）
// @Tags         标签
// @Accept       json
// @Produce      json
// @Param        id path string true "标签ID"
// @Success      200 {object} map[string]interface{} "删除成功提示"
// @Failure      400 {object} map[string]interface{} "无效的标签ID"
// @Failure      403 {object} map[string]interface{} "权限不足"
// @Failure      404 {object} map[string]interface{} "标签不存在"
// @Failure      500 {object} map[string]interface{} "删除标签失败"
// @Security     BearerAuth
// @Router       /tags/{id} [delete]
func (s *Server) DeleteTag(c *gin.Context) {
	// 权限校验
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录才能操作")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) {
		utils.ForbiddenResponse(c, "权限不足，仅管理员可删除标签")
		return
	}

	// 提取并校验标签ID
	idStr := c.Param("id")
	tagID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.BadRequestResponse(c, "无效的标签ID")
		return
	}

	// 先查询标签是否存在
	ctx := c.Request.Context()
	_, err = s.svc.GetTagByID(ctx, tagID)
	if err != nil {
		utils.NotFoundResponse(c, err.Error())
		return
	}

	// 调用service层删除标签
	err = s.svc.DeleteTag(ctx, tagID)
	if err != nil {
		utils.InternalServerErrorResponse(c, "删除标签失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, gin.H{"message": "标签删除成功"})
}

// GetPublicTagList godoc
// @Summary      获取所有公开标签
// @Description  公开接口，获取所有标签（无分页，按创建时间倒序排列），用于前端下拉选择等场景
// @Tags         标签
// @Accept       json
// @Produce      json
// @Success      200 {object} utils.Response{data=[]models.Tag} "所有公开标签列表"
// @Failure      500 {object} utils.Response "获取标签列表失败"
// @Router       /tags/public [get]
func (s *Server) GetPublicTagList(c *gin.Context) {
	// 调用 Service 层获取所有公开标签
	ctx := c.Request.Context()
	tags, err := s.svc.GetPublicTagList(ctx)
	if err != nil {
		utils.InternalServerErrorResponse(c, "获取标签列表失败: "+err.Error())
		return
	}

	// 统一响应格式
	utils.SuccessResponse(c, tags)
}
