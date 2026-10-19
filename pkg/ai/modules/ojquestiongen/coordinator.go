package ojquestiongen

import (
	"context"
	"fmt"
	"sync"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/modules"
	"zk-code-arena-server/pkg/ai/modules/ojquestiongen/agents"
	"zk-code-arena-server/pkg/ai/modules/ojquestiongen/prompts"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
	"zk-code-arena-server/pkg/common/sandbox"
	"zk-code-arena-server/pkg/utils"
)

// Coordinator 协调器
// 负责编排题目生成的四个 Stage
type Coordinator struct {
	moduleRunner     *modules.ModuleRunner
	questionGenAgent *agents.QuestionGenAgent
	standardCodeGen  *agents.StandardCodeGenAgent
	testcaseGen      *agents.TestcaseGenAgent
	verifier         *Verifier
}

// NewCoordinator 创建协调器
func NewCoordinator(
	sessionService runtimesession.SessionService,
	taskManager runtimetask.TaskManager,
	contextManager runtimecontext.ContextManager,
	chatModel llm.ModelFactory,
	sandboxClient sandbox.Client,
) *Coordinator {
	questionGenAgent := agents.NewQuestionGenAgent(chatModel, prompts.DefaultQuestionGenInstruction)
	standardCodeGen := agents.NewStandardCodeGenAgent(chatModel, prompts.DefaultStandardCodeGenInstruction)
	testcaseGen := agents.NewTestcaseGenAgent(chatModel, prompts.DefaultTestcaseGenInstruction)
	verifier := NewVerifier(sandboxClient)

	core := &coordinatorCore{
		questionGenAgent: questionGenAgent,
		standardCodeGen:  standardCodeGen,
		testcaseGen:      testcaseGen,
		verifier:         verifier,
	}

	moduleRunner := modules.NewModuleRunner(
		core,
		sessionService,
		taskManager,
		contextManager,
	)

	return &Coordinator{
		moduleRunner:     moduleRunner,
		questionGenAgent: questionGenAgent,
		standardCodeGen:  standardCodeGen,
		testcaseGen:      testcaseGen,
		verifier:         verifier,
	}
}

// Type 返回模块类型标识
func (c *Coordinator) Type() string {
	return CanonicalModuleType
}

// Stream 入口方法（委托给 ModuleRunner）
func (c *Coordinator) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	return c.moduleRunner.Stream(ctx, req)
}

// coordinatorCore 实现 ModuleRunnerCore
type coordinatorCore struct {
	questionGenAgent *agents.QuestionGenAgent
	standardCodeGen  *agents.StandardCodeGenAgent
	testcaseGen      *agents.TestcaseGenAgent
	verifier         *Verifier
}

// Type 返回模块类型标识
func (c *coordinatorCore) Type() string {
	return CanonicalModuleType
}

// Execute 执行业务逻辑（四 Stage 编排）
func (c *coordinatorCore) Execute(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	snapshot *runtimecontext.Snapshot,
	emit func(*aiapi.RuntimeStreamEvent) error,
) error {
	// 1. 解析生成参数
	params := ParseGenerationParams(req)
	if len(params.KnowledgePoints) == 0 {
		return fmt.Errorf("知识点不能为空")
	}

	// 转换为 agents 用的 GenerationParams
	agentParams := &agents.GenerationParams{
		KnowledgePoints:        params.KnowledgePoints,
		Difficulty:             params.Difficulty,
		Count:                  params.Count,
		TestcaseCount:          params.TestcaseCount,
		StandardLanguage:       params.StandardLanguage,
		AdditionalRequirements: params.AdditionalRequirements,
	}

	// 2. Stage 1: 生成题目
	agentProblems, err := c.stage1GenerateQuestions(ctx, req, agentParams, emit)
	if err != nil {
		return err
	}
	if len(agentProblems) == 0 {
		return fmt.Errorf("题目生成失败")
	}

	// 转换为 ojquestiongen.GeneratedProblem
	problems := make([]*GeneratedProblem, len(agentProblems))
	for i, ap := range agentProblems {
		problems[i] = &GeneratedProblem{
			ID:           ap.ID,
			Title:        ap.Title,
			Description:  ap.Description,
			InputFormat:  ap.InputFormat,
			OutputFormat: ap.OutputFormat,
			Hint:         ap.Hint,
			Difficulty:   ap.Difficulty,
		}
	}

	// 3. Stage 2: 生成标准程序
	if err := c.stage2GenerateStandardCodes(ctx, req, problems, params.StandardLanguage, emit); err != nil {
		return err
	}

	// 4. Stage 3: 生成测试用例
	if err := c.stage3GenerateTestCases(ctx, req, problems, params.TestcaseCount, emit); err != nil {
		return err
	}

	// 5. Stage 4: 验证
	if err := c.stage4Verify(ctx, req, problems, emit); err != nil {
		return err
	}

	// 6. 发送等待确认
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventWaitingConfirmation,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "completed",
		Data: map[string]any{
			"questions": problems,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	return nil
}

// stage1GenerateQuestions 生成题目
func (c *coordinatorCore) stage1GenerateQuestions(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	params *agents.GenerationParams,
	emit func(*aiapi.RuntimeStreamEvent) error,
) ([]*agents.GeneratedProblem, error) {
	stageStartedAt := time.Now()
	// 发射 stage 开始
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageStarted,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "generating",
		Data: map[string]any{
			"stage": "generating",
			"total": params.Count,
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// 调用 Agent.Process() 获取结果
	problems, err := c.questionGenAgent.Process(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("生成题目失败: %w", err)
	}

	problems = ensureProblemCount(problems, params.Count)
	utils.Logger.Infof("OJQuestionGen Stage1完成: session_id=%s, requested=%d, actual=%d, duration_ms=%d",
		req.SessionID,
		params.Count,
		len(problems),
		time.Since(stageStartedAt).Milliseconds(),
	)

	// 发射每道题生成完成事件
	for _, problem := range problems {
		_ = emit(&aiapi.RuntimeStreamEvent{
			Type:      aiapi.EventQuestionGenerated,
			SessionID: req.SessionID,
			RoundID:   req.RoundID,
			TaskType:  CanonicalModuleType,
			Stage:     "generating",
			Data:      problem,
			Timestamp: time.Now().UnixMilli(),
		})
	}

	// 发射 stage 完成
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageComplete,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "generating",
		Data: map[string]any{
			"stage":     "generating",
			"completed": len(problems),
		},
		Timestamp: time.Now().UnixMilli(),
	})

	return problems, nil
}

// stage2GenerateStandardCodes 生成标准程序
func (c *coordinatorCore) stage2GenerateStandardCodes(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	problems []*GeneratedProblem,
	language string,
	emit func(*aiapi.RuntimeStreamEvent) error,
) error {
	stageStartedAt := time.Now()
	// 发射 stage 开始
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageStarted,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "codegen",
		Data: map[string]any{
			"stage": "codegen",
			"total": len(problems),
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// 并行生成
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := []error{}

	for _, problem := range problems {
		wg.Add(1)
		go func(p *GeneratedProblem) {
			defer wg.Done()
			itemStartedAt := time.Now()
			utils.Logger.Infof("OJQuestionGen Stage2开始: session_id=%s, problem_id=%s, language=%s",
				req.SessionID,
				p.ID,
				language,
			)

			// 转换为 agents.ProblemData
			agentProblem := &agents.GeneratedProblem{
				ID:           p.ID,
				Title:        p.Title,
				Description:  p.Description,
				InputFormat:  p.InputFormat,
				OutputFormat: p.OutputFormat,
				Hint:         p.Hint,
				Difficulty:   p.Difficulty,
			}

			// 调用 Agent.Process()
			code, err := c.standardCodeGen.Process(ctx, agentProblem, language)
			if err != nil {
				utils.Logger.Errorf("OJQuestionGen Stage2失败: session_id=%s, problem_id=%s, duration_ms=%d, error=%v",
					req.SessionID,
					p.ID,
					time.Since(itemStartedAt).Milliseconds(),
					err,
				)
				mu.Lock()
				errs = append(errs, fmt.Errorf("生成标准程序失败(%s): %w", p.ID, err))
				mu.Unlock()
				return
			}

			p.StandardCode = code
			p.Language = language
			p.Status = "codegen"

			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventStandardCodeGenerated,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalModuleType,
				Stage:     "codegen",
				Data: map[string]any{
					"id":       p.ID,
					"code":     code,
					"language": language,
				},
				Timestamp: time.Now().UnixMilli(),
			})
			utils.Logger.Infof("OJQuestionGen Stage2完成: session_id=%s, problem_id=%s, duration_ms=%d, code_len=%d",
				req.SessionID,
				p.ID,
				time.Since(itemStartedAt).Milliseconds(),
				len(code),
			)
		}(problem)
	}
	wg.Wait()

	if len(errs) > 0 {
		for _, e := range errs {
			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventError,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalModuleType,
				Stage:     "codegen",
				Data:      map[string]any{"message": e.Error()},
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}

	// 发射 stage 完成
	completed := len(problems) - len(errs)
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageComplete,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "codegen",
		Data: map[string]any{
			"stage":     "codegen",
			"completed": completed,
		},
		Timestamp: time.Now().UnixMilli(),
	})
	utils.Logger.Infof("OJQuestionGen Stage2汇总: session_id=%s, total=%d, failed=%d, duration_ms=%d",
		req.SessionID,
		len(problems),
		len(errs),
		time.Since(stageStartedAt).Milliseconds(),
	)

	return nil
}

// stage3GenerateTestCases 生成测试用例
func (c *coordinatorCore) stage3GenerateTestCases(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	problems []*GeneratedProblem,
	testcaseCount int,
	emit func(*aiapi.RuntimeStreamEvent) error,
) error {
	stageStartedAt := time.Now()
	// 发射 stage 开始
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageStarted,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "testcase",
		Data: map[string]any{
			"stage": "testcase",
			"total": len(problems),
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// 并行生成
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := []error{}

	for _, problem := range problems {
		wg.Add(1)
		go func(p *GeneratedProblem) {
			defer wg.Done()
			itemStartedAt := time.Now()
			utils.Logger.Infof("OJQuestionGen Stage3开始: session_id=%s, problem_id=%s, testcase_count=%d",
				req.SessionID,
				p.ID,
				testcaseCount,
			)

			// 转换为 agents.ProblemData
			agentProblem := &agents.GeneratedProblem{
				ID:           p.ID,
				Title:        p.Title,
				Description:  p.Description,
				InputFormat:  p.InputFormat,
				OutputFormat: p.OutputFormat,
				Hint:         p.Hint,
				Difficulty:   p.Difficulty,
			}

			// 调用 Agent.Process()
			agentTestCases, err := c.testcaseGen.Process(ctx, agentProblem, p.StandardCode, testcaseCount)
			if err != nil || len(agentTestCases) == 0 {
				utils.Logger.Errorf("OJQuestionGen Stage3失败: session_id=%s, problem_id=%s, duration_ms=%d, testcase_count=%d, error=%v",
					req.SessionID,
					p.ID,
					time.Since(itemStartedAt).Milliseconds(),
					len(agentTestCases),
					err,
				)
				mu.Lock()
				errs = append(errs, fmt.Errorf("生成测试用例失败(%s): %w", p.ID, err))
				mu.Unlock()
				return
			}

			// 转换回 ojquestiongen.TestCase
			testCases := make([]TestCase, len(agentTestCases))
			for i, tc := range agentTestCases {
				testCases[i] = TestCase{
					Input:          tc.Input,
					ExpectedOutput: tc.ExpectedOutput,
					IsSample:       tc.IsSample,
				}
			}

			p.TestCases = testCases
			p.Status = "testcase"

			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventTestCaseGenerated,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalModuleType,
				Stage:     "testcase",
				Data: map[string]any{
					"id":         p.ID,
					"test_cases": testCases,
				},
				Timestamp: time.Now().UnixMilli(),
			})
			utils.Logger.Infof("OJQuestionGen Stage3完成: session_id=%s, problem_id=%s, duration_ms=%d, testcase_count=%d",
				req.SessionID,
				p.ID,
				time.Since(itemStartedAt).Milliseconds(),
				len(testCases),
			)
		}(problem)
	}
	wg.Wait()

	if len(errs) > 0 {
		for _, e := range errs {
			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventError,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalModuleType,
				Stage:     "testcase",
				Data:      map[string]any{"message": e.Error()},
				Timestamp: time.Now().UnixMilli(),
			})
		}
	}

	// 发射 stage 完成
	completed := len(problems) - len(errs)
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageComplete,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "testcase",
		Data: map[string]any{
			"stage":     "testcase",
			"completed": completed,
		},
		Timestamp: time.Now().UnixMilli(),
	})
	utils.Logger.Infof("OJQuestionGen Stage3汇总: session_id=%s, total=%d, failed=%d, duration_ms=%d",
		req.SessionID,
		len(problems),
		len(errs),
		time.Since(stageStartedAt).Milliseconds(),
	)

	return nil
}

// stage4Verify 验证标准程序
func (c *coordinatorCore) stage4Verify(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	problems []*GeneratedProblem,
	emit func(*aiapi.RuntimeStreamEvent) error,
) error {
	stageStartedAt := time.Now()
	// 发射 stage 开始
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageStarted,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "verifying",
		Data: map[string]any{
			"stage": "verifying",
			"total": len(problems),
		},
		Timestamp: time.Now().UnixMilli(),
	})

	// 并行验证
	var wg sync.WaitGroup

	for _, problem := range problems {
		wg.Add(1)
		go func(p *GeneratedProblem) {
			defer wg.Done()
			itemStartedAt := time.Now()
			utils.Logger.Infof("OJQuestionGen Stage4开始: session_id=%s, problem_id=%s, testcase_count=%d",
				req.SessionID,
				p.ID,
				len(p.TestCases),
			)

			result, err := c.verifier.Verify(ctx, p.StandardCode, p.Language, p.TestCases)
			p.Verification = result
			p.Status = "pending"

			if err != nil || !result.Passed {
				p.Status = "verification_failed"
				utils.Logger.Warnf("OJQuestionGen Stage4未通过: session_id=%s, problem_id=%s, duration_ms=%d, passed=%v, error=%v, message=%s",
					req.SessionID,
					p.ID,
					time.Since(itemStartedAt).Milliseconds(),
					result.Passed,
					err,
					result.Message,
				)
			} else {
				utils.Logger.Infof("OJQuestionGen Stage4完成: session_id=%s, problem_id=%s, duration_ms=%d",
					req.SessionID,
					p.ID,
					time.Since(itemStartedAt).Milliseconds(),
				)
			}

			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:      aiapi.EventVerificationCompleted,
				SessionID: req.SessionID,
				RoundID:   req.RoundID,
				TaskType:  CanonicalModuleType,
				Stage:     "verifying",
				Data: map[string]any{
					"id":          p.ID,
					"passed":      result.Passed,
					"message":     result.Message,
					"failed_case": result.FailedCase,
				},
				Timestamp: time.Now().UnixMilli(),
			})
		}(problem)
	}
	wg.Wait()

	// 发射 stage 完成
	_ = emit(&aiapi.RuntimeStreamEvent{
		Type:      aiapi.EventStageComplete,
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		TaskType:  CanonicalModuleType,
		Stage:     "verifying",
		Data: map[string]any{
			"stage":     "verifying",
			"completed": len(problems),
		},
		Timestamp: time.Now().UnixMilli(),
	})
	utils.Logger.Infof("OJQuestionGen Stage4汇总: session_id=%s, total=%d, duration_ms=%d",
		req.SessionID,
		len(problems),
		time.Since(stageStartedAt).Milliseconds(),
	)

	return nil
}

// ensureProblemCount 保证题目数量符合预期；当模型少返回题目时，补齐出的题目必须是新对象，避免前端按相同 ID 去重。
func ensureProblemCount(problems []*agents.GeneratedProblem, count int) []*agents.GeneratedProblem {
	if len(problems) == 0 {
		return problems
	}
	if len(problems) > count {
		return problems[:count]
	}
	if len(problems) == count {
		return problems
	}

	baseProblems := append([]*agents.GeneratedProblem(nil), problems...)
	nextID := len(problems) + 1
	for len(problems) < count {
		source := baseProblems[(len(problems)-len(baseProblems))%len(baseProblems)]
		cloned := cloneGeneratedProblem(source, nextID)
		problems = append(problems, cloned)
		nextID++
	}
	return problems
}

// cloneGeneratedProblem 克隆题目骨架并分配新的生成 ID，避免并发阶段操作同一指针。
func cloneGeneratedProblem(problem *agents.GeneratedProblem, ordinal int) *agents.GeneratedProblem {
	if problem == nil {
		return &agents.GeneratedProblem{ID: fmt.Sprintf("gen_%d", ordinal)}
	}
	cloned := *problem
	cloned.ID = fmt.Sprintf("gen_%d", ordinal)
	return &cloned
}
