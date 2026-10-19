// service_teacher_copilot.go 串联教师侧 Copilot 的会话、历史、工具、编排和 SSE 事件。
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	teachercopilot "zk-code-arena-server/pkg/ai/assistants/teachercopilot"
	teachercopilotagents "zk-code-arena-server/pkg/ai/assistants/teachercopilot/agents"
	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/multiagent"
	runtimeaudit "zk-code-arena-server/pkg/ai/runtime/audit"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimestate "zk-code-arena-server/pkg/ai/runtime/state"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
	ojtools "zk-code-arena-server/pkg/ai/tools/oj"
)

// TeacherCopilotService 是教师侧 AI 助手的业务编排服务。
// 它不走通用 Gateway，而是在这里显式串联会话、历史、工具和 multi-agent 编排。
type TeacherCopilotService struct {
	userService    *UserService
	clazzService   *ClazzService
	problemService *ProblemService
	sessionService runtimesession.SessionService
	taskManager    runtimetask.TaskManager
	contextManager runtimecontext.ContextManager
	modelFactory   llm.ModelFactory
}

// NewTeacherCopilotService 创建教师侧 AI 助手业务服务。
func NewTeacherCopilotService(
	userService *UserService,
	clazzService *ClazzService,
	problemService *ProblemService,
	sessionService runtimesession.SessionService,
	taskManager runtimetask.TaskManager,
	contextManager runtimecontext.ContextManager,
	modelFactory llm.ModelFactory,
) *TeacherCopilotService {
	return &TeacherCopilotService{
		userService:    userService,
		clazzService:   clazzService,
		problemService: problemService,
		sessionService: sessionService,
		taskManager:    taskManager,
		contextManager: contextManager,
		modelFactory:   modelFactory,
	}
}

// Stream 启动一次教师侧 Copilot 流式任务，并返回 RuntimeStreamEvent 通道。
func (s *TeacherCopilotService) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}

	normalizeTeacherCopilotRequest(req)
	startedAt := time.Now()

	if err := s.sessionService.CreateIfNeeded(ctx, req); err != nil {
		return nil, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	root := &runtimestate.RootTaskRuntime{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		TaskKind:  aiapi.TaskKindAssistant,
		TaskType:  req.TaskType,
		Stage:     "initializing",
		StartedAt: startedAt,
		Cancel:    cancel,
	}
	if err := s.taskManager.RegisterRoot(root); err != nil {
		cancel()
		return nil, err
	}

	if err := s.sessionService.SaveQuestion(ctx, req); err != nil {
		s.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	snapshot, err := s.contextManager.Build(ctx, req)
	if err != nil {
		_ = s.sessionService.SaveResult(ctx, &runtimesession.SaveResultInput{
			SessionID:    req.SessionID,
			RoundID:      req.RoundID,
			UserID:       req.UserID,
			TaskType:     req.TaskType,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		s.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	collector := runtimeaudit.NewSessionCollector()
	events := make(chan *aiapi.RuntimeStreamEvent, 16)
	go s.run(runCtx, cancel, req, snapshot, collector, startedAt, events)
	return events, nil
}

// run 在后台执行教师侧 Copilot，并在结束时关闭 SSE 通道、释放任务和保存结果。
func (s *TeacherCopilotService) run(
	ctx context.Context,
	cancel context.CancelFunc,
	req *aiapi.RuntimeRequest,
	snapshot *runtimecontext.Snapshot,
	collector *runtimeaudit.SessionCollector,
	startedAt time.Time,
	events chan<- *aiapi.RuntimeStreamEvent,
) {
	defer close(events)
	defer cancel()
	defer s.taskManager.RemoveRoot(req.SessionID, req.RoundID)

	emit := s.newEmitter(ctx, req, collector, events)
	status := "completed"
	errorMessage := ""

	_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventMeta, req, "running", map[string]any{
		"status": "running",
		"domain": teachercopilot.AssistantType,
	}))

	result, err := s.execute(ctx, req, snapshot, emit)
	if err != nil {
		status = statusFromTeacherCopilotError(err)
		errorMessage = err.Error()
		_ = emit(&aiapi.RuntimeStreamEvent{
			Type:         aiapi.EventError,
			SessionID:    req.SessionID,
			RoundID:      req.RoundID,
			TaskType:     req.TaskType,
			Stage:        status,
			ErrorMessage: err.Error(),
			Timestamp:    time.Now().UnixMilli(),
		})
	} else {
		answer := renderTeacherCopilotAnswer(result)
		if strings.TrimSpace(answer) != "" {
			_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventText, req, "answering", map[string]any{
				"delta": answer,
			}))
		}
	}

	if !collector.Done() {
		_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventDone, req, status, map[string]any{
			"status": status,
		}))
	}

	saveInput := collector.BuildSaveResult(req, startedAt, status, errorMessage)
	if saveInput != nil {
		_ = s.sessionService.SaveResult(context.Background(), saveInput)
	}
}

// execute 组装工具、角色和 Orchestrator，并执行一次教师侧 multi-agent 编排。
func (s *TeacherCopilotService) execute(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	snapshot *runtimecontext.Snapshot,
	emit func(*aiapi.RuntimeStreamEvent) error,
) (*multiagent.RunResult, error) {
	_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventState, req, "planning", map[string]any{
		"stage_label": "规划教师侧分析任务",
	}))

	provider := ojtools.NewProvider(ojtools.Dependencies{
		UserReader:    s.userService,
		ClazzReader:   s.clazzService,
		ProblemReader: s.problemService,
	}, req.UserID)
	if !provider.IsConfigured() {
		return nil, aiapi.ErrAssistantNotConfigured
	}

	registry, err := s.buildRoleRegistry(req, provider, emit)
	if err != nil {
		return nil, err
	}
	lead := teachercopilot.NewLLMTeacherSupervisorAgent(s.modelFactory, provider.AvailableCatalog()...)
	orchestrator := multiagent.NewOrchestrator(lead, registry, multiagent.OrchestratorConfig{
		MaxAgents:           4,
		MaxConcurrentAgents: 2,
	})

	result, err := orchestrator.Run(ctx, multiagent.RunRequest{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		Domain:    teachercopilot.AssistantType,
		Query:     req.Message,
		Context:   buildTeacherCopilotContext(req, snapshot),
		Roles:     registry.Roles(),
	})
	if err != nil {
		return nil, err
	}
	_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventState, req, "synthesizing", map[string]any{
		"stage_label": "整理教师侧分析结果",
		"task_count":  len(result.Tasks),
	}))
	return result, nil
}

// buildRoleRegistry 注册教师侧第一批角色。当前 runner 是可替换骨架，
// 后续会逐个替换为真实 ReAct Agent。
func (s *TeacherCopilotService) buildRoleRegistry(req *aiapi.RuntimeRequest, provider *ojtools.Provider, emit func(*aiapi.RuntimeStreamEvent) error) (*multiagent.RoleRegistry, error) {
	registry := multiagent.NewRoleRegistry()
	for _, role := range teachercopilot.DefaultRoles() {
		role := role
		if err := registry.Register(role, func(ctx context.Context, spec multiagent.RoleSpec) (multiagent.AgentRunner, error) {
			if spec.Name == teachercopilot.RoleClassAnalyst {
				return teachercopilotagents.NewClassAnalystRunner(s.modelFactory, provider, req, emit), nil
			}
			return &teacherPlaceholderAgentRunner{
				role:     spec,
				provider: provider,
				emit:     emit,
			}, nil
		}); err != nil {
			return nil, err
		}
	}
	return registry, nil
}

// newEmitter 创建统一事件发送函数，负责补齐会话字段、写入审计收集器并推送到 SSE 通道。
func (s *TeacherCopilotService) newEmitter(
	ctx context.Context,
	req *aiapi.RuntimeRequest,
	collector *runtimeaudit.SessionCollector,
	events chan<- *aiapi.RuntimeStreamEvent,
) func(*aiapi.RuntimeStreamEvent) error {
	return func(event *aiapi.RuntimeStreamEvent) error {
		if event == nil {
			return nil
		}
		if event.Timestamp == 0 {
			event.Timestamp = time.Now().UnixMilli()
		}
		if event.SessionID == "" {
			event.SessionID = req.SessionID
		}
		if event.RoundID == "" {
			event.RoundID = req.RoundID
		}
		if event.TaskType == "" {
			event.TaskType = req.TaskType
		}
		collector.Append(event)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- event:
			return nil
		}
	}
}

// validate 校验教师侧 Copilot 运行所需的核心依赖是否已经注入。
func (s *TeacherCopilotService) validate() error {
	if s == nil || s.sessionService == nil || s.taskManager == nil || s.contextManager == nil || s.modelFactory == nil {
		return aiapi.ErrAssistantNotConfigured
	}
	if !s.modelFactory.IsChatConfigured() {
		return aiapi.ErrAssistantNotConfigured
	}
	if s.userService == nil || s.clazzService == nil || s.problemService == nil {
		return aiapi.ErrAssistantNotConfigured
	}
	return nil
}

// teacherPlaceholderAgentRunner 是子 Agent 的第一版可替换骨架。
// 后续会逐步替换为 class_analyst、assignment_reviewer、report_writer 的 ReAct 实现。
type teacherPlaceholderAgentRunner struct {
	role     multiagent.RoleSpec
	provider *ojtools.Provider
	emit     func(*aiapi.RuntimeStreamEvent) error
}

// Run 先记录子 Agent 被调度的事实，保证 TeacherCopilotService 的主链路可运行。
func (r *teacherPlaceholderAgentRunner) Run(ctx context.Context, task multiagent.AgentTask) (*multiagent.AgentResult, error) {
	_ = ctx
	if r == nil {
		return nil, fmt.Errorf("teacher agent runner is nil")
	}
	if r.emit != nil {
		_ = r.emit(&aiapi.RuntimeStreamEvent{
			Type:      aiapi.EventState,
			TaskType:  teachercopilot.AssistantType,
			Stage:     "agent_running",
			Timestamp: time.Now().UnixMilli(),
			Data: map[string]any{
				"task_id":     task.ID,
				"agent_id":    task.AgentID,
				"role":        task.Role,
				"description": task.Description,
			},
		})
	}
	availableTools := 0
	if r.provider != nil {
		availableTools = len(r.provider.AvailableCatalog())
	}
	return &multiagent.AgentResult{
		AgentID: task.AgentID,
		TaskID:  task.ID,
		Role:    task.Role,
		Content: fmt.Sprintf("%s 已接入调度框架，后续会替换为真实 ReAct Agent。任务：%s", task.Role, task.Description),
		Data: map[string]any{
			"available_tools": availableTools,
			"base_role":       r.role.BaseRole,
		},
		Success: true,
	}, nil
}

// normalizeTeacherCopilotRequest 固定教师侧 Copilot 的任务类型并补齐 session_id、round_id。
func normalizeTeacherCopilotRequest(req *aiapi.RuntimeRequest) {
	req.TaskKind = aiapi.TaskKindAssistant
	req.TaskType = teachercopilot.AssistantType
	req.AssistantType = teachercopilot.AssistantType
	req.Normalize()
}

// buildTeacherCopilotContext 合并前端输入、元数据和历史消息，作为 Lead 规划上下文。
func buildTeacherCopilotContext(req *aiapi.RuntimeRequest, snapshot *runtimecontext.Snapshot) map[string]any {
	contextMap := map[string]any{}
	if req != nil {
		for key, value := range req.Input {
			contextMap[key] = value
		}
		for key, value := range req.Metadata {
			contextMap[key] = value
		}
	}
	if snapshot != nil && len(snapshot.Messages) > 0 {
		contextMap["history"] = snapshot.Messages
	}
	return contextMap
}

// renderTeacherCopilotAnswer 将 Orchestrator 结果渲染成第一版可保存、可返回的文本。
func renderTeacherCopilotAnswer(result *multiagent.RunResult) string {
	if result == nil {
		return ""
	}
	if strings.TrimSpace(result.FinalReply) != "" {
		return result.FinalReply
	}
	parts := make([]string, 0, len(result.Results)+1)
	if strings.TrimSpace(result.InterimReply) != "" {
		parts = append(parts, result.InterimReply)
	}
	for _, item := range result.Results {
		if strings.TrimSpace(item.Content) != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", item.Role, item.Content))
		}
	}
	return strings.Join(parts, "\n")
}

// statusFromTeacherCopilotError 将运行错误映射为会话保存状态。
func statusFromTeacherCopilotError(err error) string {
	if errors.Is(err, context.Canceled) {
		return "stopped"
	}
	return "failed"
}
