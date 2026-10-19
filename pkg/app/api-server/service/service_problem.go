/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: service_problem.go
@Description: 题目服务
*/

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/common/sandbox"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ProblemService struct {
	sandboxClient       sandbox.Client
	testCaseService     *TestCaseService
	repo                *repository.ProblemRepository
	contentRelationRepo *repository.ContentRelationRepository
	clazzService        *ClazzService // 添加ClazzService支持任务列表功能
	svc                 *Service
}

// NewProblemService 创建题目服务。
// 这里兼容旧的构造参数顺序，避免同学机器上拿到“service.go 已更新但 service_problem.go 未完全同步”时直接编译失败。
func NewProblemService(
	sandboxClient sandbox.Client,
	testCaseService *TestCaseService,
	repo *repository.ProblemRepository,
	args ...interface{},
) *ProblemService {
	service := &ProblemService{
		sandboxClient:   sandboxClient,
		testCaseService: testCaseService,
		repo:            repo,
	}

	switch len(args) {
	case 2:
		if clazzService, ok := args[0].(*ClazzService); ok {
			service.clazzService = clazzService
		}
		if svc, ok := args[1].(*Service); ok {
			service.svc = svc
		}
	case 3:
		if contentRelationRepo, ok := args[0].(*repository.ContentRelationRepository); ok {
			service.contentRelationRepo = contentRelationRepo
		}
		if clazzService, ok := args[1].(*ClazzService); ok {
			service.clazzService = clazzService
		}
		if svc, ok := args[2].(*Service); ok {
			service.svc = svc
		}
	}

	return service
}

// CreateProblem 创建题目
func (s *ProblemService) CreateProblem(ctx context.Context, problem *models.Problem) error {
	targetCourseIDs := append([]primitive.ObjectID(nil), problem.CourseIDs...)
	// 1. 设置系统字段
	problem.ID = primitive.NewObjectID()
	problem.CreatedAt = time.Now()
	problem.UpdatedAt = time.Now()

	// 2. 生成唯一编号 UniqueID
	uniqueID, err := s.repo.GetNextSequenceValue(ctx, "problem_unique_id")
	if err != nil {
		utils.Logger.Errorf("CreateProblem: 获取唯一编号失败, error=%v", err)
		return fmt.Errorf("生成题目编号失败: %w", err)
	}
	problem.UniqueID = uniqueID

	// 3. 设置默认Status（如果未传）
	if problem.Status == "" {
		// 新增题目默认进入主页题库，因此缺省状态使用已发布；草稿仍可由前端显式传入。
		problem.Status = models.StatusPublished
		utils.Logger.Infof("CreateProblem: 未指定状态，设置默认状态为已发布")
	}

	// 4. 应用Status与IsPublic关联规则
	if problem.Status == models.StatusDraft {
		// 草稿状态强制私有
		if problem.IsPublic {
			utils.Logger.Warnf("CreateProblem: 草稿状态不能公开，强制设为私有")
		}
		problem.IsPublic = false
	}
	// Published/Archived状态，保持用户设置或默认私有
	// （IsPublic由Handler层传入，这里不修改）

	// 5. 设置默认计数器
	problem.ACCount = 0
	problem.SubmitCount = 0

	// 6. 设置默认限制（如果为0）
	if problem.TimeLimit == 0 {
		problem.TimeLimit = 1000
		utils.Logger.Debugf("CreateProblem: 使用默认时间限制 1000ms")
	}
	if problem.MemoryLimit == 0 {
		problem.MemoryLimit = 256
		utils.Logger.Debugf("CreateProblem: 使用默认内存限制 256MB")
	}

	// 7. 确保Tags不为nil
	if problem.Tags == nil {
		problem.Tags = []string{}
	}
	if problem.CourseIDs == nil {
		// 课程挂载为空数组表示“不挂载到课程题库”，与 is_public 的主页可见性相互独立。
		problem.CourseIDs = []primitive.ObjectID{}
	}
	// 编程题课程挂载的真源已经切到关系表，文档字段只保留为兼容壳，避免主文档和关系表出现双写漂移。
	problem.CourseIDs = []primitive.ObjectID{}

	// 8. 标签去重+空字符串过滤
	tagMap := make(map[string]struct{})
	var validTags []string
	for _, tag := range problem.Tags {
		// 去除首尾空格，过滤空标签
		trimmedTag := strings.TrimSpace(tag)
		if trimmedTag != "" {
			tagMap[trimmedTag] = struct{}{}
		}
	}
	// 转换map为切片，完成去重
	for tag := range tagMap {
		validTags = append(validTags, tag)
	}
	problem.Tags = validTags

	// 9. 标签存在性校验
	// 调用全局Service获取所有系统有效标签
	allSystemTags, err := s.svc.GetPublicTagList(ctx)
	if err != nil {
		utils.Logger.Errorf("CreateProblem: 获取系统有效标签失败, error=%v", err)
		return fmt.Errorf("校验标签失败：获取系统标签列表失败")
	}

	// 构建系统标签映射表
	systemTagMap := make(map[string]struct{})
	for _, sysTag := range allSystemTags {
		// 转为小写，忽略大小写校验
		systemTagMap[strings.ToLower(sysTag.Name)] = struct{}{}
	}

	// 遍历校验每个标签是否存在
	for _, tag := range problem.Tags {
		lowerTag := strings.ToLower(tag)
		if _, exists := systemTagMap[lowerTag]; !exists {
			utils.Logger.Warnf("CreateProblem: 无效标签，不存在于系统标签库, tag=%s", tag)
			return fmt.Errorf("标签「%s」不存在，请选择下拉选中的有效标签", tag)
		}
	}

	// 10. 记录详细日志
	utils.Logger.Infof("CreateProblem: title=%s, difficulty=%s, status=%s, isPublic=%v, createdBy=%s",
		problem.Title, problem.Difficulty, problem.Status, problem.IsPublic, problem.CreatedBy.Hex())

	// 11. 持久化到数据库 - 委托给Repository层
	err = s.repo.CreateProblem(ctx, problem)
	if err != nil {
		utils.Logger.Errorf("CreateProblem: 数据库插入失败, error=%v", err)
		return fmt.Errorf("数据库操作失败: %w", err)
	}
	if err := s.syncProgrammingCourseRelations(ctx, problem.ID, problem.CreatedBy, targetCourseIDs); err != nil {
		// 关系挂载失败时补偿删除刚创建的题目，避免留下“题目存在但课程范围丢失”的半成品数据。
		if cleanupErr := s.repo.DeleteProblem(ctx, problem.ID); cleanupErr != nil {
			return fmt.Errorf("同步编程题课程关系失败，且补偿删除失败: %v; 原始错误: %w", cleanupErr, err)
		}
		return err
	}
	problem.CourseIDs = targetCourseIDs

	utils.Logger.Infof("CreateProblem: 题目创建成功, id=%s", problem.ID.Hex())
	return nil
}

// GetProblemByUniqueID 根据唯一编号获取题目
func (s *ProblemService) GetProblemByUniqueID(ctx context.Context, uniqueID int64) (*models.Problem, error) {
	utils.Logger.Debugf("GetProblemByUniqueID: unique_id=%d", uniqueID)

	problem, err := s.repo.GetByUniqueID(ctx, uniqueID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Warnf("GetProblemByUniqueID: 题目不存在, unique_id=%d", uniqueID)
			return nil, models.ErrProblemNotFound
		}
		utils.Logger.Errorf("GetProblemByUniqueID: 查询数据库失败, unique_id=%d, error=%v", uniqueID, err)
		return nil, fmt.Errorf("获取题目失败: %w", err)
	}
	if err := s.fillProblemCourseIDs(ctx, problem); err != nil {
		return nil, err
	}
	return problem, nil
}

// GetProblemByID 根据ID获取题目
func (s *ProblemService) GetProblemByID(ctx context.Context, id primitive.ObjectID) (*models.Problem, error) {
	// 数据操作：直接委托给Repository层
	problem, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.fillProblemCourseIDs(ctx, problem); err != nil {
		return nil, err
	}
	return problem, nil
}

// GetProblems 获取题目列表
func (s *ProblemService) GetProblems(
	ctx context.Context,
	page, pageSize int,
	difficulty models.ProblemDifficulty,
	tags []string,
	includePrivate bool,
	role models.UserRole,
	userID *primitive.ObjectID,
) ([]*models.ProblemList, int64, error) {
	// 业务逻辑：记录日志信息
	if !includePrivate {
		utils.Logger.Debugf("GetProblems: 公开模式，过滤条件: is_public=true, status=published")
	} else {
		switch role {
		case models.RoleAdmin:
			utils.Logger.Debugf("GetProblems: 管理员模式，查看所有题目")
		case models.RoleTeacher:
			if userID != nil {
				utils.Logger.Debugf("GetProblems: 教师模式，查看自己创建的题目, userID=%s", userID.Hex())
			} else {
				utils.Logger.Debugf("GetProblems: 教师模式但无userID，退回公开模式")
			}
		default:
			utils.Logger.Debugf("GetProblems: 学生/游客模式，只看公开已发布题目")
		}
	}

	// 数据操作：委托给Repository层
	return s.repo.GetProblemsWithUserStatus(ctx, page, pageSize, difficulty, tags, includePrivate, role, userID)
}

// getProblemsByCondition 核心查询方法（私有，用于内部复用）
func (s *ProblemService) getProblemsByCondition(
	ctx context.Context,
	condition *models.ProblemQueryCondition,
) ([]*models.ProblemList, int64, error) {
	// 统一的核心查询逻辑
	return s.repo.GetProblemsWithUserStatus(
		ctx,
		condition.Page,
		condition.PageSize,
		condition.Difficulty,
		condition.Tags,
		condition.IncludePrivate,
		condition.Role,
		condition.UserID,
	)
}

// GetProblemsForUser 获取用户端题目列表
func (s *ProblemService) GetProblemsForUser(
	ctx context.Context,
	page, pageSize int,
	difficulty models.ProblemDifficulty,
	tags []string,
	userID *primitive.ObjectID,
) ([]*models.ProblemList, int64, error) {
	// 构建用户端查询条件
	condition := &models.ProblemQueryCondition{
		Page:           page,
		PageSize:       pageSize,
		Difficulty:     difficulty,
		Tags:           tags,
		IncludePrivate: false,              // 用户端固定为false，只看公开题目
		Role:           models.RoleStudent, // 固定为学生角色
		UserID:         userID,             // 用于获取用户提交状态
	}

	// 业务逻辑：记录用户端查询日志
	if userID != nil {
		utils.Logger.Debugf("GetProblemsForUser: 用户端查询, userID=%s, difficulty=%s, tags=%v",
			userID.Hex(), difficulty, tags)
	} else {
		utils.Logger.Debugf("GetProblemsForUser: 未登录用户查询, difficulty=%s, tags=%v",
			difficulty, tags)
	}

	problems, total, err := s.getProblemsByCondition(ctx, condition)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillProblemListCourseIDs(ctx, problems); err != nil {
		return nil, 0, err
	}
	return problems, total, nil
}

// GetProblemsForAdmin 获取管理员端题目列表
func (s *ProblemService) GetProblemsForAdmin(
	ctx context.Context,
	page, pageSize int,
	difficulty models.ProblemDifficulty,
	tags []string,
) ([]*models.ProblemList, int64, error) {
	// 构建管理员端查询条件
	condition := &models.ProblemQueryCondition{
		Page:           page,
		PageSize:       pageSize,
		Difficulty:     difficulty,
		Tags:           tags,
		IncludePrivate: true,             // 管理员可以看所有题目，包括私有
		Role:           models.RoleAdmin, // 管理员角色
		UserID:         nil,              // 管理员不需要获取用户提交状态
		// 注意：status和createdBy筛选将在后续版本中支持
	}

	// 业务逻辑：记录管理员查询日志
	utils.Logger.Debugf("GetProblemsForAdmin: 管理员查询, difficulty=%s, tags=%v",
		difficulty, tags)
	utils.Logger.Infof("GetProblemsForAdmin: 管理员可查看所有题目（包括私有和草稿状态）")

	problems, total, err := s.getProblemsByCondition(ctx, condition)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillProblemListCourseIDs(ctx, problems); err != nil {
		return nil, 0, err
	}
	return problems, total, nil
}

// GetProblemsByCourseID 获取指定课程题库下的题目列表。
func (s *ProblemService) GetProblemsByCourseID(
	ctx context.Context,
	courseID primitive.ObjectID,
	page, pageSize int,
	difficulty models.ProblemDifficulty,
	tags []string,
	userID *primitive.ObjectID,
) ([]*models.ProblemList, int64, error) {
	problemIDs, err := s.listProgrammingProblemIDsByCourse(ctx, courseID)
	if err != nil {
		return nil, 0, err
	}
	problems, total, err := s.repo.GetProblemsByCourseID(ctx, problemIDs, page, pageSize, difficulty, tags, userID)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillProblemListCourseIDs(ctx, problems); err != nil {
		return nil, 0, err
	}
	return problems, total, nil
}

func (s *ProblemService) UpdateProblem(ctx context.Context, problemID primitive.ObjectID, req *models.UpdateProblemRequest) error {
	if req == nil {
		return fmt.Errorf("题目更新请求不能为空")
	}
	currentProblem, err := s.GetProblemByID(ctx, problemID)
	if err != nil {
		return err
	}
	targetCourseIDs := append([]primitive.ObjectID(nil), currentProblem.CourseIDs...)
	if req.CourseIDs != nil {
		targetCourseIDs = make([]primitive.ObjectID, 0, len(*req.CourseIDs))
		for _, id := range *req.CourseIDs {
			objID, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return fmt.Errorf("无效的课程ID: %s", id)
			}
			targetCourseIDs = append(targetCourseIDs, objID)
		}
	}
	// 业务逻辑：记录更新操作日志和处理特殊业务规则
	if req.Status != nil && *req.Status == models.StatusDraft {
		utils.Logger.Warnf("UpdateProblem: 草稿状态不能公开，强制设为私有, problemID=%s", problemID.Hex())
	}

	utils.Logger.Infof("UpdateProblem: 开始更新题目, problemID=%s", problemID.Hex())

	// 数据操作：委托给Repository层处理字段映射和更新
	err = s.repo.UpdateProblemFromRequest(ctx, problemID, req)
	if err != nil {
		utils.Logger.Errorf("UpdateProblem: 数据库更新失败, problemID=%s, error=%v", problemID.Hex(), err)
		return fmt.Errorf("数据库操作失败: %w", err)
	}
	if err := s.syncProgrammingCourseRelations(ctx, problemID, currentProblem.CreatedBy, targetCourseIDs); err != nil {
		return err
	}

	utils.Logger.Infof("UpdateProblem: 题目更新成功, problemID=%s", problemID.Hex())
	return nil
}

// DeleteProblem 删除题目
func (s *ProblemService) DeleteProblem(ctx context.Context, id primitive.ObjectID) error {
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeProgramming, id); err != nil {
		return err
	}
	return s.repo.DeleteProblem(ctx, id)
}

// UpdateProblemStats 更新题目统计信息
func (s *ProblemService) UpdateProblemStats(ctx context.Context, problemID primitive.ObjectID, isAC bool) error {
	// 数据操作：直接委托给Repository层
	return s.repo.UpdateProblemStats(ctx, problemID, isAC)
}

// RunCodeRequest 代码运行请求
type RunCodeRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Input    string `json:"input"` // 自定义输入（可选）
}

// RunCodeResponse 代码运行响应
type RunCodeResponse struct {
	Success    bool   `json:"success"`
	Status     string `json:"status"` // accepted, runtime_error, time_limit_exceeded, etc.
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	TimeUsed   int    `json:"time_used"`   // ms
	MemoryUsed int    `json:"memory_used"` // KB

	// 如果使用示例用例
	TestResults []models.TestResult `json:"test_results,omitempty"`
}

// RunCode 运行代码测试（非提交评测）
func (s *ProblemService) RunCode(ctx context.Context, problemID primitive.ObjectID, req *RunCodeRequest) (*RunCodeResponse, error) {
	logger := utils.GetLogger(ctx)

	// 1. 获取题目信息
	problem, err := s.GetProblemByID(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("获取题目信息失败: %w", err)
	}

	// 2. 验证语言是否支持
	langConfig := s.sandboxClient.GetLanguageConfig(req.Language)
	if langConfig == nil {
		return nil, fmt.Errorf("不支持的语言: %s", req.Language)
	}

	// 3. 编译（如果需要）
	var executableID string
	if langConfig.Compile != nil {
		logger.Info(fmt.Sprintf("开始编译: Language=%s", req.Language))

		compileReq := &sandbox.CompileRequest{
			Language:   req.Language,
			SourceCode: req.Code,
		}

		compileResp, err := s.sandboxClient.CompileCode(ctx, compileReq)
		if err != nil {
			return nil, fmt.Errorf("编译失败: %w", err)
		}

		if !compileResp.Success {
			return &RunCodeResponse{
				Success: false,
				Status:  "compile_error",
				Error:   compileResp.CompileError,
			}, nil
		}

		executableID = compileResp.ExecutableID
		logger.Info(fmt.Sprintf("编译成功: ExecutableID=%s", executableID))
	} else {
		// 解释型语言，使用源代码
		executableID = req.Code
	}

	// 4. 如果提供了自定义输入，使用自定义输入运行
	if req.Input != "" {
		return s.runWithCustomInput(ctx, executableID, req, problem)
	}

	// 5. 否则使用示例测试用例运行
	return s.runWithSampleTestCases(ctx, executableID, req, problem, problemID)
}

// runWithCustomInput 使用自定义输入运行
func (s *ProblemService) runWithCustomInput(
	ctx context.Context,
	executableID string,
	req *RunCodeRequest,
	problem *models.Problem,
) (*RunCodeResponse, error) {
	runReq := &sandbox.RunRequest{
		Language:     req.Language,
		ExecutableID: executableID,
		Input:        req.Input,
		TimeLimit:    int64(problem.TimeLimit) * 1_000_000,   // ms -> ns
		MemoryLimit:  int64(problem.MemoryLimit) * 1_048_576, // MB -> bytes
	}

	runResp, err := s.sandboxClient.RunCode(ctx, runReq)
	if err != nil {
		return nil, fmt.Errorf("运行失败: %w", err)
	}

	resp := &RunCodeResponse{
		Success:    runResp.Status == sandbox.RunStatusAccepted,
		Status:     string(runResp.Status),
		Output:     runResp.Output,
		Error:      runResp.Error,
		TimeUsed:   int(runResp.Time / 1_000_000), // ns -> ms
		MemoryUsed: int(runResp.Memory / 1024),    // bytes -> KB
	}

	return resp, nil
}

// runWithSampleTestCases 使用示例测试用例运行
func (s *ProblemService) runWithSampleTestCases(
	ctx context.Context,
	executableID string,
	req *RunCodeRequest,
	problem *models.Problem,
	problemID primitive.ObjectID,
) (*RunCodeResponse, error) {
	// 获取示例测试用例
	testCases, err := s.testCaseService.GetSampleTestCases(ctx, problemID)
	if err != nil {
		return nil, fmt.Errorf("获取示例测试用例失败: %w", err)
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("该题目没有示例测试用例")
	}

	// 运行所有示例测试用例
	var testResults []models.TestResult
	allPassed := true

	for _, testCase := range testCases {
		runReq := &sandbox.RunRequest{
			Language:     req.Language,
			ExecutableID: executableID,
			Input:        testCase.Input,
			TimeLimit:    int64(problem.TimeLimit) * 1_000_000,   // ms -> ns
			MemoryLimit:  int64(problem.MemoryLimit) * 1_048_576, // MB -> bytes
		}

		runResp, err := s.sandboxClient.RunCode(ctx, runReq)
		if err != nil {
			return nil, fmt.Errorf("运行测试用例失败: %w", err)
		}

		// 构建测试结果
		result := models.TestResult{
			TestCaseID: testCase.ID,
			TimeUsed:   int(runResp.Time / 1_000_000), // ns -> ms
			MemoryUsed: int(runResp.Memory / 1024),    // bytes -> KB
			IsSample:   true,
			Output:     runResp.Output,
			Expected:   testCase.Output,
			Error:      runResp.Error,
		}

		// 判断运行状态
		switch runResp.Status {
		case sandbox.RunStatusAccepted:
			if compareOutput(runResp.Output, testCase.Output) {
				result.Status = models.StatusAccepted
			} else {
				result.Status = models.StatusWrongAnswer
				allPassed = false
			}
		case sandbox.RunStatusTimeLimitExceeded:
			result.Status = models.StatusTimeLimit
			allPassed = false
		case sandbox.RunStatusMemoryLimitExceeded:
			result.Status = models.StatusMemoryLimit
			allPassed = false
		case sandbox.RunStatusRuntimeError:
			result.Status = models.StatusRuntimeError
			allPassed = false
		default:
			result.Status = models.StatusSystemError
			allPassed = false
		}

		testResults = append(testResults, result)
	}

	// 构建响应
	status := "accepted"
	if !allPassed {
		// 使用第一个失败的测试用例的状态
		for _, result := range testResults {
			if result.Status != models.StatusAccepted {
				status = string(result.Status)
				break
			}
		}
	}

	resp := &RunCodeResponse{
		Success:     allPassed,
		Status:      status,
		TestResults: testResults,
	}

	// 计算总时间和内存
	for _, result := range testResults {
		if result.TimeUsed > resp.TimeUsed {
			resp.TimeUsed = result.TimeUsed
		}
		if result.MemoryUsed > resp.MemoryUsed {
			resp.MemoryUsed = result.MemoryUsed
		}
	}

	return resp, nil
}

func compareOutput(actual, expected string) bool {
	return utils.CompareJudgeOutput(actual, expected)
}

// SearchProblems 搜索题目
func (s *ProblemService) SearchProblems(
	ctx context.Context,
	keyword string,
	difficulty models.ProblemDifficulty,
	tags []string,
	page, pageSize int,
	userID *primitive.ObjectID,
	inFavorite bool,
) ([]*models.ProblemList, int64, error) {
	// 业务逻辑：记录搜索相关日志
	if userID != nil {
		utils.Logger.Debugf("SearchProblems: 开始搜索, keyword=%s, in_favorite=%v, userID=%s", keyword, inFavorite, userID.Hex())
	} else {
		utils.Logger.Debugf("SearchProblems: 未登录用户搜索, keyword=%s, in_favorite=%v", keyword, inFavorite)
	}

	// 数据操作：委托给Repository层
	problems, total, err := s.repo.SearchProblems(ctx, keyword, page, pageSize, difficulty, tags, userID, inFavorite)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillProblemListCourseIDs(ctx, problems); err != nil {
		return nil, 0, err
	}
	return problems, total, nil
}

// GetUserProblemStatuses 批量查询用户对多个题目的状态
// 参数:
//   - ctx: 上下文
//   - userID: 用户ID
//   - problemIDs: 题目ID列表
//
// 返回:
//   - map[string]models.UserProblemStatus: key为题目ID字符串，value为用户状态
//   - error: 错误信息
func (s *ProblemService) GetUserProblemStatuses(
	ctx context.Context,
	userID primitive.ObjectID,
	problemIDs []primitive.ObjectID,
) (map[string]models.UserProblemStatus, error) {
	// 业务逻辑：记录查询日志
	utils.Logger.Debugf("GetUserProblemStatuses: 查询用户状态, userID=%s, problemCount=%d",
		userID.Hex(), len(problemIDs))

	// 数据操作：委托给Repository层
	statusMap, err := s.repo.GetUserProblemStatuses(ctx, userID, problemIDs)
	if err != nil {
		utils.Logger.Errorf("GetUserProblemStatuses: Repository查询失败, userID=%s, error=%v",
			userID.Hex(), err)
		return nil, err
	}

	// 业务逻辑：转换map键类型从ObjectID到string
	result := make(map[string]models.UserProblemStatus, len(statusMap))
	for problemID, status := range statusMap {
		result[problemID.Hex()] = status
	}

	utils.Logger.Infof("GetUserProblemStatuses: 查询完成, userID=%s, 返回%d个题目状态",
		userID.Hex(), len(result))

	return result, nil
}

// GetProblemDetail 获取题目详情聚合信息
func (s *ProblemService) GetProblemDetail(ctx context.Context, problemID primitive.ObjectID) (*models.ProblemDetailResponse, error) {
	// 业务逻辑：记录查询日志
	utils.Logger.Infof("GetProblemDetail: 开始获取题目详情聚合信息, problemID=%s", problemID.Hex())

	// 数据操作1：获取题目基本信息
	problem, err := s.GetProblemByID(ctx, problemID)
	if err != nil {
		utils.Logger.Errorf("GetProblemDetail: 获取题目信息失败, problemID=%s, error=%v", problemID.Hex(), err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, models.ErrProblemNotFound
		}
		return nil, fmt.Errorf("%w: %w", models.ErrProblemInfoFetchFailed, err)
	}

	// 数据操作2：获取示例测试用例
	sampleCases, err := s.testCaseService.GetSampleTestCases(ctx, problemID)
	if err != nil {
		utils.Logger.Errorf("GetProblemDetail: 获取示例测试用例失败, problemID=%s, error=%v", problemID.Hex(), err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, models.ErrProblemTestCaseNotFound
		}
		return nil, fmt.Errorf("%w: %w", models.ErrSampleTestCaseFetchFailed, err)
	}

	// 业务逻辑：数据聚合和组装
	response := &models.ProblemDetailResponse{
		Problem:     problem,
		SampleCases: make([]models.TestCase, len(sampleCases)),
	}

	// 转换指针切片为值切片
	for i, testCase := range sampleCases {
		response.SampleCases[i] = *testCase
	}

	utils.Logger.Infof("GetProblemDetail: 题目详情聚合完成, problemID=%s, sampleCasesCount=%d",
		problemID.Hex(), len(sampleCases))

	return response, nil
}

// BatchCreateProblems 批量创建题目
func (s *ProblemService) BatchCreateProblems(ctx context.Context, problems []*models.Problem) (*models.BatchCreateProblemsResponse, error) {
	// 业务逻辑：记录批量创建日志
	utils.Logger.Infof("BatchCreateProblems: 开始批量创建题目, count=%d", len(problems))

	// 初始化响应
	response := &models.BatchCreateProblemsResponse{
		SuccessCount: 0,
		FailCount:    0,
		TotalCount:   len(problems),
		Results:      make([]models.BatchCreateProblemResult, 0, len(problems)),
	}

	// 遍历创建每个题目
	for i, problem := range problems {
		// 调用单个创建方法
		err := s.CreateProblem(ctx, problem)

		// 构建结果
		result := models.BatchCreateProblemResult{
			Index:     i,
			Title:     problem.Title,
			Success:   err == nil,
			ProblemID: problem.ID.Hex(),
		}

		if err != nil {
			result.Error = err.Error()
			response.FailCount++
			utils.Logger.Errorf("BatchCreateProblems: 题目创建失败, index=%d, title=%s, error=%v",
				i, problem.Title, err)
		} else {
			response.SuccessCount++
			utils.Logger.Infof("BatchCreateProblems: 题目创建成功, index=%d, title=%s, id=%s",
				i, problem.Title, problem.ID.Hex())
		}

		response.Results = append(response.Results, result)
	}

	utils.Logger.Infof("BatchCreateProblems: 批量创建完成, total=%d, success=%d, fail=%d",
		response.TotalCount, response.SuccessCount, response.FailCount)

	return response, nil
}

// ToggleFavorite 切换题目收藏状态
func (s *ProblemService) ToggleFavorite(ctx context.Context, userID, problemID primitive.ObjectID, isFavorite bool) error {
	if isFavorite {
		return s.repo.AddFavorite(ctx, userID, problemID)
	} else {
		return s.repo.RemoveFavorite(ctx, userID, problemID)
	}
}

// GetUserFavoriteProblems 获取用户收藏的题目列表
func (s *ProblemService) GetUserFavoriteProblems(ctx context.Context, userID primitive.ObjectID, difficulty models.ProblemDifficulty, page, pageSize int) ([]*models.ProblemList, int64, error) {
	problems, total, err := s.repo.GetUserFavoriteProblems(ctx, userID, difficulty, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	if err := s.fillProblemListCourseIDs(ctx, problems); err != nil {
		return nil, 0, err
	}
	return problems, total, nil
}

// GetNextProblem 获取当前题目的下一题
func (s *ProblemService) GetNextProblem(ctx context.Context, currentUniqueID int64, taskID string, currentIdx int) (*models.Problem, error) {
	// 如果提供了任务列表ID，从任务列表中获取下一题
	if taskID != "" {
		utils.Logger.Debugf("GetNextProblem from task: task_id=%s, current_idx=%d", taskID, currentIdx)

		// 转换任务ID为ObjectID
		taskObjID, err := primitive.ObjectIDFromHex(taskID)
		if err != nil {
			return nil, fmt.Errorf("无效的任务ID: %w", err)
		}

		// 获取任务信息
		task, err := s.clazzService.Repo.GetTaskByID(ctx, taskObjID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("任务不存在: %w", err)
			}
			return nil, fmt.Errorf("获取任务信息失败: %w", err)
		}

		// 验证当前题号是否有效
		if currentIdx < 0 || currentIdx >= len(task.RelationIDs) {
			return nil, fmt.Errorf("无效的当前题号")
		}

		// 获取下一题的索引
		nextIdx := currentIdx + 1
		if nextIdx >= len(task.RelationIDs) {
			// 已经是最后一题，返回问题不存在错误
			return nil, models.ErrProblemNotFound
		}

		// 获取下一题的ID
		nextProblemID := task.RelationIDs[nextIdx]

		// 获取下一题的信息
		problem, err := s.GetProblemByID(ctx, nextProblemID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, models.ErrProblemNotFound
			}
			return nil, fmt.Errorf("获取下一题失败: %w", err)
		}

		return problem, nil
	}

	// 否则使用现有逻辑，通过UniqueID获取下一题
	utils.Logger.Debugf("GetNextProblem by uniqueID: current_unique_id=%d", currentUniqueID)

	// 调用Repository层方法获取下一题
	problem, err := s.repo.GetNextProblem(ctx, currentUniqueID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Warnf("GetNextProblem: 没有找到下一题, current_unique_id=%d", currentUniqueID)
			return nil, models.ErrProblemNotFound
		}
		utils.Logger.Errorf("GetNextProblem: 查询数据库失败, current_unique_id=%d, error=%v", currentUniqueID, err)
		return nil, fmt.Errorf("获取下一题失败: %w", err)
	}

	return problem, nil
}

// listProgrammingProblemIDsByCourse 查询指定课程下挂载的编程题 ID。
func (s *ProblemService) listProgrammingProblemIDsByCourse(ctx context.Context, courseID primitive.ObjectID) ([]primitive.ObjectID, error) {
	if s.contentRelationRepo == nil {
		return nil, fmt.Errorf("课程内容关系仓储未初始化")
	}
	relations, err := s.contentRelationRepo.ListCourseRelations(ctx, courseID, models.ContentTypeProgramming)
	if err != nil {
		return nil, err
	}
	ids := make([]primitive.ObjectID, 0, len(relations))
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		ids = append(ids, relation.ContentID)
	}
	return ids, nil
}

// syncProgrammingCourseRelations 同步编程题与课程的挂载关系。
func (s *ProblemService) syncProgrammingCourseRelations(
	ctx context.Context,
	problemID primitive.ObjectID,
	createdBy primitive.ObjectID,
	courseIDs []primitive.ObjectID,
) error {
	if s.contentRelationRepo == nil {
		return fmt.Errorf("课程内容关系仓储未初始化")
	}
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeProgramming, problemID); err != nil {
		return err
	}
	if len(courseIDs) == 0 {
		return nil
	}
	relations := make([]*models.CourseContentRelation, 0, len(courseIDs))
	for _, courseID := range courseIDs {
		relations = append(relations, &models.CourseContentRelation{
			CourseID:    courseID,
			ContentType: models.ContentTypeProgramming,
			ContentID:   problemID,
			CreatedBy:   createdBy,
		})
	}
	return s.contentRelationRepo.CreateCourseRelations(ctx, relations)
}

// fillProblemCourseIDs 将关系表中的课程挂载范围回填到题目实体，兼容旧接口仍在暴露的 course_ids 字段。
func (s *ProblemService) fillProblemCourseIDs(ctx context.Context, problem *models.Problem) error {
	if problem == nil {
		return nil
	}
	courseIDMap, err := s.loadProgrammingCourseIDs(ctx, []primitive.ObjectID{problem.ID})
	if err != nil {
		return err
	}
	problem.CourseIDs = courseIDMap[problem.ID]
	if problem.CourseIDs == nil {
		problem.CourseIDs = []primitive.ObjectID{}
	}
	return nil
}

// fillProblemListCourseIDs 批量回填题目列表里的课程挂载范围。
func (s *ProblemService) fillProblemListCourseIDs(ctx context.Context, problems []*models.ProblemList) error {
	if len(problems) == 0 {
		return nil
	}
	ids := make([]primitive.ObjectID, 0, len(problems))
	for _, problem := range problems {
		if problem == nil {
			continue
		}
		ids = append(ids, problem.ID)
	}
	courseIDMap, err := s.loadProgrammingCourseIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, problem := range problems {
		if problem == nil {
			continue
		}
		problem.CourseIDs = courseIDMap[problem.ID]
		if problem.CourseIDs == nil {
			problem.CourseIDs = []primitive.ObjectID{}
		}
	}
	return nil
}

// loadProgrammingCourseIDs 批量读取编程题的课程挂载关系。
func (s *ProblemService) loadProgrammingCourseIDs(ctx context.Context, problemIDs []primitive.ObjectID) (map[primitive.ObjectID][]primitive.ObjectID, error) {
	result := make(map[primitive.ObjectID][]primitive.ObjectID)
	if len(problemIDs) == 0 {
		return result, nil
	}
	if s.contentRelationRepo == nil {
		return nil, fmt.Errorf("课程内容关系仓储未初始化")
	}
	relations, err := s.contentRelationRepo.ListRelationsByContentIDs(ctx, models.ContentTypeProgramming, problemIDs)
	if err != nil {
		return nil, err
	}
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		result[relation.ContentID] = append(result[relation.ContentID], relation.CourseID)
	}
	return result, nil
}
