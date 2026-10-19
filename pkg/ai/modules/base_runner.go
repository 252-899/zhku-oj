package modules

import (
	"context"
	"errors"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	runtimeaudit "zk-code-arena-server/pkg/ai/runtime/audit"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimestate "zk-code-arena-server/pkg/ai/runtime/state"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"
)

// ModuleRunnerCore 子类核心接口
// 子类实现此接口后传递给 NewModuleRunner
type ModuleRunnerCore interface {
	// Type 返回模块类型标识
	Type() string
	// Execute 执行模块逻辑
	Execute(ctx context.Context, req *aiapi.RuntimeRequest, snapshot *runtimecontext.Snapshot, emit func(*aiapi.RuntimeStreamEvent) error) error
}

// ModuleRunner Module 的 Stream 骨架
// 封装通用的 Stream → run → execute → save 骨架
type ModuleRunner struct {
	core            ModuleRunnerCore
	sessionService  runtimesession.SessionService
	taskManager     runtimetask.TaskManager
	contextManager  runtimecontext.ContextManager
}

// NewModuleRunner 创建 Module 运行器
func NewModuleRunner(
	core ModuleRunnerCore,
	sessionService runtimesession.SessionService,
	taskManager runtimetask.TaskManager,
	contextManager runtimecontext.ContextManager,
) *ModuleRunner {
	return &ModuleRunner{
		core:           core,
		sessionService: sessionService,
		taskManager:    taskManager,
		contextManager: contextManager,
	}
}

// Stream 入口方法
func (m *ModuleRunner) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if m == nil || m.core == nil {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	if req == nil {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	if err := m.validate(); err != nil {
		return nil, err
	}

	req.Normalize()
	req.TaskKind = aiapi.TaskKindModule
	req.TaskType = m.core.Type()

	startedAt := time.Now()

	// ① 会话持久化
	if err := m.sessionService.CreateIfNeeded(ctx, req); err != nil {
		return nil, err
	}

	// 创建可取消上下文
	runCtx, cancel := context.WithCancel(ctx)

	// ② 任务注册
	taskRuntime := &runtimestate.RootTaskRuntime{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		TaskKind:  aiapi.TaskKindModule,
		TaskType:  req.TaskType,
		StartedAt: startedAt,
		Cancel:    cancel,
	}
	if err := m.taskManager.RegisterRoot(taskRuntime); err != nil {
		cancel()
		return nil, err
	}

	// 写入用户问题
	if err := m.sessionService.SaveQuestion(ctx, req); err != nil {
		m.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	// ③ 上下文构建
	snapshot, err := m.contextManager.Build(ctx, req)
	if err != nil {
		_ = m.sessionService.SaveResult(ctx, &runtimesession.SaveResultInput{
			SessionID:    req.SessionID,
			RoundID:      req.RoundID,
			UserID:       req.UserID,
			TaskType:     req.TaskType,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		m.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	collector := runtimeaudit.NewSessionCollector()
	events := make(chan *aiapi.RuntimeStreamEvent, 16)

	// ④ 异步执行
	go m.run(runCtx, cancel, req, snapshot, collector, startedAt, events)
	return events, nil
}

// run 执行骨架
func (m *ModuleRunner) run(
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
	defer m.taskManager.RemoveRoot(req.SessionID, req.RoundID)

	taskType := m.core.Type()

	// emit 闭包
	emit := func(runtimeEvent *aiapi.RuntimeStreamEvent) error {
		if runtimeEvent == nil {
			return nil
		}
		if runtimeEvent.Timestamp == 0 {
			runtimeEvent.Timestamp = time.Now().UnixMilli()
		}
		if runtimeEvent.SessionID == "" {
			runtimeEvent.SessionID = req.SessionID
		}
		if runtimeEvent.RoundID == "" {
			runtimeEvent.RoundID = req.RoundID
		}
		if runtimeEvent.TaskType == "" {
			runtimeEvent.TaskType = taskType
		}

		collector.Append(runtimeEvent)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case events <- runtimeEvent:
			return nil
		}
	}

	// ⑤ 发送初始状态
	_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventMeta, req, "running", map[string]any{
		"status": "running",
	}))

	// ⑥ 执行
	execErr := m.core.Execute(ctx, req, snapshot, emit)
	status := "completed"
	errorMessage := ""
	if execErr != nil {
		if errors.Is(execErr, context.Canceled) {
			status = "stopped"
			errorMessage = execErr.Error()
		} else {
			status = "failed"
			errorMessage = execErr.Error()
			_ = emit(&aiapi.RuntimeStreamEvent{
				Type:         aiapi.EventError,
				SessionID:    req.SessionID,
				RoundID:      req.RoundID,
				TaskType:     taskType,
				Stage:        "failed",
				ErrorMessage: execErr.Error(),
				Timestamp:    time.Now().UnixMilli(),
			})
		}
	}

	// 发送完成事件
	if !collector.Done() {
		_ = emit(aiapi.NewRuntimeStreamEvent(aiapi.EventDone, req, status, map[string]any{
			"status": status,
		}))
	}

	// ⑦ 结果持久化
	saveInput := collector.BuildSaveResult(req, startedAt, status, errorMessage)
	if saveInput != nil {
		_ = m.sessionService.SaveResult(context.Background(), saveInput)
	}
}

// validate 验证依赖完整性
func (m *ModuleRunner) validate() error {
	if m.sessionService == nil || m.taskManager == nil || m.contextManager == nil {
		return aiapi.ErrAssistantNotConfigured
	}
	return nil
}
