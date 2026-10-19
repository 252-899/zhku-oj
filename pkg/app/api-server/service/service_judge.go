/*
@Author: omenkk7
@Date: 2025/10/05
@Description: 判题服务 - 业务逻辑编排
*/

package service

import (
	"context"
	"fmt"

	"zk-code-arena-server/conf"
	"zk-code-arena-server/pkg/common/queue"
	"zk-code-arena-server/pkg/common/sandbox"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	wsManager "zk-code-arena-server/pkg/utils/websocket"
)

// JudgeService 判题服务
type JudgeService struct {
	sandboxClient   sandbox.Client
	submitService   *SubmitService
	testCaseService *TestCaseService
	problemService  *ProblemService
	messageQueue    queue.MessageQueue
}

// JudgeContext 判题上下文
type JudgeContext struct {
	Task         *queue.JudgeTask
	Problem      *models.Problem
	TestCases    []*models.TestCase
	ExecutableID string
	Results      []models.TestResult
}

// NewJudgeService 创建判题服务实例
func NewJudgeService(
	sandboxClient sandbox.Client,
	submitService *SubmitService,
	testCaseService *TestCaseService,
	problemService *ProblemService,
	messageQueue queue.MessageQueue,
) *JudgeService {
	return &JudgeService{
		sandboxClient:   sandboxClient,
		submitService:   submitService,
		testCaseService: testCaseService,
		problemService:  problemService,
		messageQueue:    messageQueue,
	}
}

// Start 启动判题服务（启动消费者）
func (js *JudgeService) Start(ctx context.Context) error {
	logger := utils.GetLogger(ctx)
	logger.Info("启动判题服务...")

	// 1. 启动消息队列
	if err := js.messageQueue.Start(ctx); err != nil {
		return fmt.Errorf("启动消息队列失败: %w", err)
	}

	// 2. 恢复未完成的任务
	if err := js.RecoverTasks(ctx); err != nil {
		logger.Error(fmt.Sprintf("恢复任务失败: %v", err))
		// 不阻止服务启动
	}

	// 3. 启动消费者
	workers := conf.Config.Judge.Workers
	if workers <= 0 {
		workers = 4 // 默认值
	}
	if err := js.messageQueue.Consume(ctx, js.judgeTaskHandler, workers); err != nil {
		return fmt.Errorf("启动消费者失败: %w", err)
	}

	logger.Info("判题服务启动成功")
	return nil
}

// Stop 停止判题服务（优雅关闭）
func (js *JudgeService) Stop(ctx context.Context) error {
	logger := utils.GetLogger(ctx)
	logger.Info("正在停止判题服务...")

	if err := js.messageQueue.Stop(ctx); err != nil {
		return fmt.Errorf("停止消息队列失败: %w", err)
	}

	logger.Info("判题服务已停止")
	return nil
}

// SubmitTask 提交判题任务（生产者接口）
func (js *JudgeService) SubmitTask(ctx context.Context, task *queue.JudgeTask) error {
	return js.messageQueue.Push(ctx, task)
}

// RecoverTasks 恢复未完成的任务
func (js *JudgeService) RecoverTasks(ctx context.Context) error {
	logger := utils.GetLogger(ctx)
	logger.Info("开始恢复未完成的任务...")

	// 1. 将 "running" 状态的提交标记为 "system_error"
	if err := js.submitService.BatchUpdateStatus(ctx, models.StatusRunning, models.StatusSystemError, "服务重启，任务中断"); err != nil {
		logger.Error(fmt.Sprintf("批量更新 running 状态失败: %v", err))
	} else {
		logger.Info("已将 running 状态的提交标记为 system_error")
	}

	// 2. 获取待处理任务并重新入队
	pendingTasks, err := js.messageQueue.GetPendingTasks(ctx)
	if err != nil {
		return fmt.Errorf("获取待处理任务失败: %w", err)
	}

	logger.Info(fmt.Sprintf("找到 %d 个待处理任务，准备重新入队", len(pendingTasks)))

	for _, task := range pendingTasks {
		if err := js.messageQueue.Push(ctx, task); err != nil {
			logger.Error(fmt.Sprintf("重新入队任务失败: SubmitID=%s, Error=%v", task.SubmitID.Hex(), err))
		}
	}

	logger.Info(fmt.Sprintf("任务恢复完成，共恢复 %d 个任务", len(pendingTasks)))
	return nil
}

// judgeTaskHandler 判题任务处理函数（传递给消息队列的 handler）
func (js *JudgeService) judgeTaskHandler(ctx context.Context, task *queue.JudgeTask) error {
	logger := utils.GetLogger(ctx)
	logger.Info(fmt.Sprintf("开始处理判题任务: SubmitID=%s", task.SubmitID.Hex()))

	if err := js.judgeTask(ctx, task); err != nil {
		logger.Error(fmt.Sprintf("判题失败: SubmitID=%s, Error=%v", task.SubmitID.Hex(), err))
		return err
	}

	logger.Info(fmt.Sprintf("判题完成: SubmitID=%s", task.SubmitID.Hex()))
	return nil
}

// judgeTask 判题主流程
func (js *JudgeService) judgeTask(ctx context.Context, task *queue.JudgeTask) error {
	// 1. 更新状态为 running
	if err := js.submitService.UpdateSubmitStatus(ctx, task.SubmitID, models.StatusRunning); err != nil {
		return fmt.Errorf("更新状态为 running 失败: %w", err)
	}

	// WebSocket推送：开始判题
	wsManager.BroadcastSubmitStatusUpdate(task.SubmitID, task.UserID, models.StatusRunning, nil, nil)

	// 2. 获取题目信息
	problem, err := js.problemService.GetProblemByID(ctx, task.ProblemID)
	if err != nil {
		js.submitService.UpdateSubmitStatus(ctx, task.SubmitID, models.StatusSystemError)
		return fmt.Errorf("获取题目信息失败: %w", err)
	}

	// 3. 获取测试用例
	testCases, err := js.testCaseService.GetTestCasesByProblemID(ctx, task.ProblemID)
	if err != nil {
		js.submitService.UpdateSubmitStatus(ctx, task.SubmitID, models.StatusSystemError)
		return fmt.Errorf("获取测试用例失败: %w", err)
	}

	if len(testCases) == 0 {
		js.submitService.UpdateSubmitStatus(ctx, task.SubmitID, models.StatusSystemError)
		return fmt.Errorf("题目没有测试用例")
	}

	// 4. 初始化判题上下文
	judgeCtx := &JudgeContext{
		Task:      task,
		Problem:   problem,
		TestCases: testCases,
		Results:   make([]models.TestResult, 0, len(testCases)),
	}

	// 5. 编译阶段
	if err := js.compileStage(ctx, judgeCtx); err != nil {
		// 编译错误
		result := &models.JudgeResult{
			Status:       models.StatusCompileError,
			TimeUsed:     0,
			MemoryUsed:   0,
			TestResults:  []models.TestResult{},
			CompileError: err.Error(),
		}

		// WebSocket推送：编译错误
		wsManager.BroadcastSubmitStatusUpdate(task.SubmitID, task.UserID, models.StatusCompileError, nil, result)

		return js.submitService.UpdateSubmitResult(ctx, task.SubmitID, result)
	}

	// 6. 运行测试用例
	if err := js.runTestCases(ctx, judgeCtx); err != nil {
		js.submitService.UpdateSubmitStatus(ctx, task.SubmitID, models.StatusSystemError)
		return fmt.Errorf("运行测试用例失败: %w", err)
	}

	// 7. 确定最终状态
	finalStatus := js.determineFinalStatus(judgeCtx.Results)
	maxTime := js.calculateMaxTime(judgeCtx.Results)
	maxMemory := js.calculateMaxMemory(judgeCtx.Results)

	// 8. 更新判题结果
	result := &models.JudgeResult{
		Status:      finalStatus,
		TimeUsed:    maxTime,
		MemoryUsed:  maxMemory,
		TestResults: judgeCtx.Results,
	}

	err = js.submitService.UpdateSubmitResult(ctx, task.SubmitID, result)
	if err != nil {
		return err
	}

	// 9. 更新题目统计信息（提交次数和通过次数）
	isAC := finalStatus == models.StatusAccepted
	if err := js.problemService.UpdateProblemStats(ctx, task.ProblemID, isAC); err != nil {
		// 记录错误但不影响判题流程
		logger := utils.GetLogger(ctx)
		logger.Error(fmt.Sprintf("更新题目统计失败: ProblemID=%s, Error=%v", task.ProblemID.Hex(), err))
	}

	// WebSocket推送：判题完成
	wsManager.BroadcastSubmitStatusUpdate(task.SubmitID, task.UserID, finalStatus, nil, result)

	return nil
}

// compileStage 编译阶段
func (js *JudgeService) compileStage(ctx context.Context, judgeCtx *JudgeContext) error {
	logger := utils.GetLogger(ctx)

	// 检查语言是否需要编译
	langConfig := js.sandboxClient.GetLanguageConfig(judgeCtx.Task.Language)
	if langConfig == nil {
		return fmt.Errorf("不支持的语言: %s", judgeCtx.Task.Language)
	}

	if langConfig.Compile == nil {
		// 解释型语言，无需编译
		logger.Info(fmt.Sprintf("语言 %s 无需编译", judgeCtx.Task.Language))
		return nil
	}

	logger.Info(fmt.Sprintf("开始编译: Language=%s", judgeCtx.Task.Language))

	compileReq := &sandbox.CompileRequest{
		Language:   judgeCtx.Task.Language,
		SourceCode: judgeCtx.Task.Code,
	}

	compileResp, err := js.sandboxClient.CompileCode(ctx, compileReq)
	if err != nil {
		return fmt.Errorf("沙箱编译失败: %w", err)
	}

	if !compileResp.Success {
		return fmt.Errorf("编译失败: %s", compileResp.CompileError)
	}

	judgeCtx.ExecutableID = compileResp.ExecutableID
	logger.Info(fmt.Sprintf("编译成功: ExecutableID=%s", compileResp.ExecutableID))
	return nil
}

// runTestCases 运行测试用例
func (js *JudgeService) runTestCases(ctx context.Context, judgeCtx *JudgeContext) error {
	logger := utils.GetLogger(ctx)
	logger.Info(fmt.Sprintf("开始运行测试用例，共 %d 个", len(judgeCtx.TestCases)))

	for i, testCase := range judgeCtx.TestCases {
		logger.Info(fmt.Sprintf("运行测试用例 %d/%d: TestCaseID=%s", i+1, len(judgeCtx.TestCases), testCase.ID.Hex()))

		result, err := js.runSingleTestCase(ctx, judgeCtx, testCase)
		if err != nil {
			return fmt.Errorf("运行测试用例失败: %w", err)
		}

		judgeCtx.Results = append(judgeCtx.Results, *result)
	}

	return nil
}

// runSingleTestCase 运行单个测试用例
func (js *JudgeService) runSingleTestCase(
	ctx context.Context,
	judgeCtx *JudgeContext,
	testCase *models.TestCase,
) (*models.TestResult, error) {
	logger := utils.GetLogger(ctx)

	// 记录测试用例信息（用于调试）
	logger.Debugf("运行测试用例: TestCaseID=%s, Input长度=%d, Output长度=%d, Input预览=%s, Output预览=%s",
		testCase.ID.Hex(),
		len(testCase.Input),
		len(testCase.Output),
		truncateString(testCase.Input, 50),
		truncateString(testCase.Output, 50))

	// 验证测试用例数据是否有效
	if testCase.Input == "" && testCase.Output == "" {
		logger.Warnf("测试用例数据可能为空: TestCaseID=%s", testCase.ID.Hex())
	}

	// 检查是否是占位符数据（常见问题：测试用例使用占位符文本）
	if testCase.Input == "测试用例1" || testCase.Output == "测试用例1" ||
		testCase.Input == "测试用例2" || testCase.Output == "测试用例2" ||
		testCase.Input == "test case 1" || testCase.Output == "test case 1" {
		logger.Warnf("⚠️ 检测到测试用例可能使用占位符数据: TestCaseID=%s, Input=%s, Output=%s",
			testCase.ID.Hex(), testCase.Input, testCase.Output)
	}

	// 获取时间和内存限制
	timeLimit := js.getTimeLimit(testCase, judgeCtx.Problem)
	memoryLimit := js.getMemoryLimit(testCase, judgeCtx.Problem)

	// 构建运行请求
	// 对于编译型语言，ExecutableID 是编译产物的文件ID
	// 对于解释型语言，ExecutableID 是源代码本身
	executableID := judgeCtx.ExecutableID
	if executableID == "" {
		// 解释型语言，使用源代码
		executableID = judgeCtx.Task.Code
	}

	runReq := &sandbox.RunRequest{
		Language:     judgeCtx.Task.Language,
		ExecutableID: executableID,
		Input:        testCase.Input,
		TimeLimit:    int64(timeLimit) * 1_000_000,   // ms -> ns
		MemoryLimit:  int64(memoryLimit) * 1_048_576, // MB -> bytes
	}

	// 调用沙箱运行
	runResp, err := js.sandboxClient.RunCode(ctx, runReq)
	if err != nil {
		logger.Errorf("沙箱运行失败: TestCaseID=%s, Error=%v", testCase.ID.Hex(), err)
		return nil, fmt.Errorf("沙箱运行失败: %w", err)
	}

	// 记录沙箱返回的详细信息
	logger.Debugf("沙箱运行结果: TestCaseID=%s, Status=%s, ExitCode=%d, Output长度=%d, Error长度=%d, Output预览=%s",
		testCase.ID.Hex(),
		string(runResp.Status),
		runResp.ExitCode,
		len(runResp.Output),
		len(runResp.Error),
		truncateString(runResp.Output, 50))

	// 构建测试结果
	result := &models.TestResult{
		TestCaseID: testCase.ID,
		TimeUsed:   int(runResp.Time / 1_000_000), // ns -> ms
		MemoryUsed: int(runResp.Memory / 1024),    // bytes -> KB
		IsSample:   testCase.IsSample,
	}

	// 判断运行状态
	switch runResp.Status {
	case sandbox.RunStatusAccepted:
		// 比对输出
		isMatch := utils.CompareJudgeOutput(runResp.Output, testCase.Output)
		if isMatch {
			result.Status = models.StatusAccepted
			logger.Debugf("输出匹配: TestCaseID=%s", testCase.ID.Hex())
		} else {
			result.Status = models.StatusWrongAnswer
			logger.Infof("输出不匹配: TestCaseID=%s, 期望输出=%s, 实际输出=%s",
				testCase.ID.Hex(),
				truncateString(testCase.Output, 100),
				truncateString(runResp.Output, 100))
		}
	case sandbox.RunStatusTimeLimitExceeded:
		result.Status = models.StatusTimeLimit
	case sandbox.RunStatusMemoryLimitExceeded:
		result.Status = models.StatusMemoryLimit
	case sandbox.RunStatusRuntimeError:
		result.Status = models.StatusRuntimeError
		logger.Warnf("沙箱判定运行时错误: TestCaseID=%s, rawStatus=%q, exitCode=%d, stderr=%q, judgeAPIErr=%q",
			testCase.ID.Hex(), runResp.RawGoJudgeStatus, runResp.ExitCode, runResp.Error, runResp.GoJudgeAPIError)
	case sandbox.RunStatusSystemError:
		// 沙箱返回系统错误
		logger := utils.GetLogger(ctx)
		logger.Warnf("测试用例返回系统错误: TestCaseID=%s, ExitCode=%d, Error=%s",
			testCase.ID.Hex(), runResp.ExitCode, runResp.Error)
		result.Status = models.StatusSystemError
	default:
		// 未知状态，记录详细信息
		logger := utils.GetLogger(ctx)
		logger.Errorf("未知的运行状态: Status=%s, TestCaseID=%s, ExitCode=%d, Error=%s",
			string(runResp.Status), testCase.ID.Hex(), runResp.ExitCode, runResp.Error)
		result.Status = models.StatusSystemError
	}

	// 仅示例用例返回详细信息
	if testCase.IsSample {
		result.Output = runResp.Output
		result.Expected = testCase.Output
		result.Error = runResp.Error
	}

	return result, nil
}

// determineFinalStatus 确定最终状态
func (js *JudgeService) determineFinalStatus(testResults []models.TestResult) models.SubmitStatus {
	if len(testResults) == 0 {
		return models.StatusSystemError
	}

	hasWrongAnswer := false
	hasTimeLimit := false
	hasMemoryLimit := false
	hasRuntimeError := false
	hasSystemError := false
	allAccepted := true

	for _, result := range testResults {
		switch result.Status {
		case models.StatusAccepted:
			// 继续
		case models.StatusWrongAnswer:
			hasWrongAnswer = true
			allAccepted = false
		case models.StatusTimeLimit:
			hasTimeLimit = true
			allAccepted = false
		case models.StatusMemoryLimit:
			hasMemoryLimit = true
			allAccepted = false
		case models.StatusRuntimeError:
			hasRuntimeError = true
			allAccepted = false
		case models.StatusSystemError:
			hasSystemError = true
			allAccepted = false
		default:
			allAccepted = false
		}
	}

	// 优先级：运行时错误 > 时间超限 > 内存超限 > 答案错误 > 系统错误 > 通过
	if hasRuntimeError {
		return models.StatusRuntimeError
	}
	if hasTimeLimit {
		return models.StatusTimeLimit
	}
	if hasMemoryLimit {
		return models.StatusMemoryLimit
	}
	if hasWrongAnswer {
		return models.StatusWrongAnswer
	}
	if hasSystemError {
		return models.StatusSystemError
	}
	if allAccepted {
		return models.StatusAccepted
	}

	return models.StatusSystemError
}

// truncateString 截断字符串用于日志输出
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getTimeLimit 获取时间限制（ms）；未配置或为 0 时使用默认值，避免 go-judge cpuLimit=0 异常
func (js *JudgeService) getTimeLimit(testCase *models.TestCase, problem *models.Problem) int {
	if testCase.TimeLimit != nil && *testCase.TimeLimit > 0 {
		return *testCase.TimeLimit
	}
	return utils.EffectiveJudgeTimeLimitMS(problem.TimeLimit)
}

// getMemoryLimit 获取内存限制（MB）；未配置或为 0 时使用默认值
func (js *JudgeService) getMemoryLimit(testCase *models.TestCase, problem *models.Problem) int {
	if testCase.MemoryLimit != nil && *testCase.MemoryLimit > 0 {
		return *testCase.MemoryLimit
	}
	return utils.EffectiveJudgeMemoryMB(problem.MemoryLimit)
}

// calculateMaxTime 计算最大时间使用
func (js *JudgeService) calculateMaxTime(testResults []models.TestResult) int {
	maxTime := 0
	for _, result := range testResults {
		if result.TimeUsed > maxTime {
			maxTime = result.TimeUsed
		}
	}
	return maxTime
}

// calculateMaxMemory 计算最大内存使用
func (js *JudgeService) calculateMaxMemory(testResults []models.TestResult) int {
	maxMemory := 0
	for _, result := range testResults {
		if result.MemoryUsed > maxMemory {
			maxMemory = result.MemoryUsed
		}
	}
	return maxMemory
}
