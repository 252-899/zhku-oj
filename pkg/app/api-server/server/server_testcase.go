/*
@Author: omenkk7
@Date: 2025/10/05
@Description: 测试用例相关路由处理
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterTestCase 注册测试用例相关路由
func (s *Server) RegisterTestCase(g *gin.RouterGroup) {
	testCaseGroup := g.Group("/testcase")

	// 需要认证的路由（使用 Group("") 避免 basePath 变成 /testcase/ 引发尾斜杠重定向）
	securedGroup := testCaseGroup.Group("").Use(middleware.JWTMiddleware())
	{
		securedGroup.GET("/problem/:problem_id", s.GetTestCasesByProblemID) // 获取题目的测试用例列表
		securedGroup.GET("/:id", s.GetTestCase)                             // 获取单个测试用例
		securedGroup.POST("", s.CreateTestCase)                             // 创建测试用例
		securedGroup.POST("/batch", s.BatchCreateTestCases)                 // 批量导入测试用例
		securedGroup.PUT("/:id", s.UpdateTestCase)                          // 更新测试用例
		securedGroup.DELETE("/:id", s.DeleteTestCase)                       // 删除测试用例
	}
}

// GetTestCasesByProblemID godoc
// @Summary      获取题目测试用例列表
// @Description  根据题目ID获取所有测试用例
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        problem_id path string true "题目ID"
// @Success      200 {object} utils.Response{data=object{test_cases=[]models.TestCase,total=int64}} "测试用例列表"
// @Failure      400 {object} models.ErrorResponse "无效的题目ID"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /testcase/problem/{problem_id} [get]
func (s *Server) GetTestCasesByProblemID(c *gin.Context) {
	problemIDStr := c.Param("problem_id")
	problemID, err := primitive.ObjectIDFromHex(problemIDStr)
	if err != nil {
		utils.ErrorResponse(c, 400, "无效的题目 ID")
		return
	}

	testCases, err := s.svc.TestCaseService.GetTestCasesByProblemID(c.Request.Context(), problemID)
	if err != nil {
		s.lg.Errorf("获取测试用例列表失败: %v", err)
		utils.ErrorResponse(c, 500, "获取测试用例列表失败")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"test_cases": testCases,
		"total":      len(testCases),
	})
}

// GetTestCase godoc
// @Summary      获取测试用例详情
// @Description  根据测试用例ID获取详细信息
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        id path string true "测试用例ID"
// @Success      200 {object} utils.Response{data=models.TestCase} "测试用例详情"
// @Failure      400 {object} models.ErrorResponse "无效的测试用例ID"
// @Failure      500 {object} models.ErrorResponse "获取失败"
// @Security     BearerAuth
// @Router       /testcase/{id} [get]
func (s *Server) GetTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.ErrorResponse(c, 400, "无效的测试用例 ID")
		return
	}

	testCase, err := s.svc.TestCaseService.GetTestCaseByID(c.Request.Context(), id)
	if err != nil {
		s.lg.Errorf("获取测试用例失败: %v", err)
		utils.ErrorResponse(c, 500, "获取测试用例失败")
		return
	}

	utils.SuccessResponse(c, testCase)
}

// CreateTestCase godoc
// @Summary      创建测试用例
// @Description  为指定题目创建新的测试用例
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        request body models.CreateTestCaseRequest true "测试用例信息"
// @Success      200 {object} utils.Response{data=models.TestCase} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      500 {object} models.ErrorResponse "创建失败"
// @Security     BearerAuth
// @Router       /testcase [post]
func (s *Server) CreateTestCase(c *gin.Context) {
	var req models.CreateTestCaseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "请求参数错误: "+err.Error())
		return
	}

	problemID, err := primitive.ObjectIDFromHex(req.ProblemID)
	if err != nil {
		utils.ErrorResponse(c, 400, "无效的题目 ID")
		return
	}

	// 验证题目是否存在
	_, err = s.svc.ProblemService.GetProblemByID(c.Request.Context(), problemID)
	if err != nil {
		utils.ErrorResponse(c, 404, "题目不存在")
		return
	}

	testCase := &models.TestCase{
		ProblemID:   problemID,
		Input:       req.Input,
		Output:      req.Output,
		IsSample:    req.IsSample,
		TimeLimit:   req.TimeLimit,
		MemoryLimit: req.MemoryLimit,
		Score:       req.Score,
	}

	if err := s.svc.TestCaseService.CreateTestCase(c.Request.Context(), testCase); err != nil {
		s.lg.Errorf("创建测试用例失败: %v", err)
		utils.ErrorResponse(c, 500, "创建测试用例失败")
		return
	}

	utils.SuccessResponse(c, testCase)
}

// UpdateTestCase godoc
// @Summary      更新测试用例
// @Description  更新指定测试用例的信息
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        id path string true "测试用例ID"
// @Param        request body models.UpdateTestCaseRequest true "更新信息"
// @Success      200 {object} utils.Response{data=models.TestCase} "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      404 {object} models.ErrorResponse "测试用例不存在"
// @Failure      500 {object} models.ErrorResponse "更新失败"
// @Security     BearerAuth
// @Router       /testcase/{id} [put]
func (s *Server) UpdateTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.ErrorResponse(c, 400, "无效的测试用例 ID")
		return
	}

	var req models.UpdateTestCaseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 获取现有测试用例
	testCase, err := s.svc.TestCaseService.GetTestCaseByID(c.Request.Context(), id)
	if err != nil {
		utils.ErrorResponse(c, 404, "测试用例不存在")
		return
	}

	// 更新字段
	if req.Input != "" {
		testCase.Input = req.Input
	}
	if req.Output != "" {
		testCase.Output = req.Output
	}
	if req.IsSample != nil {
		testCase.IsSample = *req.IsSample
	}
	if req.TimeLimit != nil {
		testCase.TimeLimit = req.TimeLimit
	}
	if req.MemoryLimit != nil {
		testCase.MemoryLimit = req.MemoryLimit
	}
	if req.Score != nil {
		testCase.Score = *req.Score
	}

	if err := s.svc.TestCaseService.UpdateTestCase(c.Request.Context(), testCase); err != nil {
		s.lg.Errorf("更新测试用例失败: %v", err)
		utils.ErrorResponse(c, 500, "更新测试用例失败")
		return
	}

	utils.SuccessResponse(c, testCase)
}

// DeleteTestCase godoc
// @Summary      删除测试用例
// @Description  删除指定测试用例
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        id path string true "测试用例ID"
// @Success      200 {object} utils.Response{data=object{message=string}} "删除成功"
// @Failure      400 {object} models.ErrorResponse "无效的测试用例ID"
// @Failure      500 {object} models.ErrorResponse "删除失败"
// @Security     BearerAuth
// @Router       /testcase/{id} [delete]
func (s *Server) DeleteTestCase(c *gin.Context) {
	idStr := c.Param("id")
	id, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		utils.ErrorResponse(c, 400, "无效的测试用例 ID")
		return
	}

	if err := s.svc.TestCaseService.DeleteTestCase(c.Request.Context(), id); err != nil {
		s.lg.Errorf("删除测试用例失败: %v", err)
		utils.ErrorResponse(c, 500, "删除测试用例失败")
		return
	}

	utils.SuccessResponse(c, gin.H{
		"message": "测试用例删除成功",
	})
}

// BatchCreateTestCases godoc
// @Summary      批量创建测试用例
// @Description  批量导入题目测试用例（教师/管理员）
// @Tags         测试用例
// @Accept       json
// @Produce      json
// @Param        request body models.BatchCreateTestCasesRequest true "批量测试用例"
// @Success      200 {object} utils.Response{data=object{success_count=int,fail_count=int,errors=[]string}} "创建成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      401 {object} models.ErrorResponse "需要登录"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Failure      404 {object} models.ErrorResponse "题目不存在"
// @Failure      500 {object} models.ErrorResponse "创建失败"
// @Security     BearerAuth
// @Router       /testcase/batch [post]
func (s *Server) BatchCreateTestCases(c *gin.Context) {
	// 检查权限：只有管理员和老师可以批量导入
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}

	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足")
		return
	}

	var req models.BatchCreateTestCasesRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, "请求参数错误: "+err.Error())
		return
	}

	problemID, err := primitive.ObjectIDFromHex(req.ProblemID)
	if err != nil {
		utils.BadRequestResponse(c, "无效的题目ID")
		return
	}

	// 验证题目是否存在
	_, err = s.svc.ProblemService.GetProblemByID(c.Request.Context(), problemID)
	if err != nil {
		utils.NotFoundResponse(c, "题目不存在")
		return
	}

	// 设置所有测试用例的 ProblemID
	testCases := make([]*models.TestCase, len(req.TestCases))
	for i := range req.TestCases {
		testCases[i] = &req.TestCases[i]
		testCases[i].ProblemID = problemID
	}

	// 批量创建
	result, err := s.svc.TestCaseService.BatchImportTestCases(c.Request.Context(), problemID, testCases)
	if err != nil {
		s.lg.Errorf("批量创建测试用例失败: %v", err)
		utils.InternalServerErrorResponse(c, "批量创建测试用例失败: "+err.Error())
		return
	}

	utils.SuccessResponse(c, result)
}
