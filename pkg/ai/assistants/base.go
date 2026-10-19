package assistants

import (
	"context"
	"errors"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/infra/llm"
	runtimeaudit "zk-code-arena-server/pkg/ai/runtime/audit"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"
	runtimesession "zk-code-arena-server/pkg/ai/runtime/session"
	runtimestate "zk-code-arena-server/pkg/ai/runtime/state"
	runtimetask "zk-code-arena-server/pkg/ai/runtime/task"

	"github.com/cloudwego/eino/schema"
)

// AssistantRunnerCore 子类核心接口
// 子类实现此接口后传递给 NewRunner
type AssistantRunnerCore interface {
	// Type 返回助手类型标识
	Type() string
	// BuildMessages 构建 LLM 消息列表
	BuildMessages(snapshot *runtimecontext.Snapshot, question string) []*schema.Message
	// Execute 执行逻辑
	Execute(ctx context.Context, req *aiapi.RuntimeRequest, snapshot *runtimecontext.Snapshot, emit func(*aiapi.RuntimeStreamEvent) error) error
}

// runner SSE 流式运行器
// 封装通用的 Stream → run → execute → save 骨架
type runner struct {
	core            AssistantRunnerCore
	sessionService  runtimesession.SessionService
	taskManager     runtimetask.TaskManager
	contextManager  runtimecontext.ContextManager
	chatModel       llm.ModelFactory
}

// NewRunner 创建运行器
func NewRunner(
	core AssistantRunnerCore,
	sessionService runtimesession.SessionService,
	taskManager runtimetask.TaskManager,
	contextManager runtimecontext.ContextManager,
	chatModel llm.ModelFactory,
) *runner {
	return &runner{
		core:           core,
		sessionService: sessionService,
		taskManager:    taskManager,
		contextManager: contextManager,
		chatModel:      chatModel,
	}
}

// Stream 入口方法
func (a *runner) Stream(ctx context.Context, req *aiapi.RuntimeRequest) (<-chan *aiapi.RuntimeStreamEvent, error) {
	if a == nil || a.core == nil {
		return nil, aiapi.ErrAssistantNotRegistered
	}
	if req == nil {
		return nil, aiapi.ErrInvalidRuntimeRequest
	}
	if err := a.validate(); err != nil {
		return nil, err
	}

	req.Normalize()
	req.TaskKind = aiapi.TaskKindAssistant
	req.TaskType = a.core.Type()

	startedAt := time.Now()

	// ① 会话持久化
	if err := a.sessionService.CreateIfNeeded(ctx, req); err != nil {
		return nil, err
	}

	// 创建可取消上下文
	runCtx, cancel := context.WithCancel(ctx)

	// ② 任务注册
	taskRuntime := &runtimestate.RootTaskRuntime{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		TaskKind:  aiapi.TaskKindAssistant,
		TaskType:  req.TaskType,
		StartedAt: startedAt,
		Cancel:    cancel,
	}
	if err := a.taskManager.RegisterRoot(taskRuntime); err != nil {
		cancel()
		return nil, err
	}

	// 写入用户问题
	if err := a.sessionService.SaveQuestion(ctx, req); err != nil {
		a.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	// ③ 上下文构建
	snapshot, err := a.contextManager.Build(ctx, req)
	if err != nil {
		_ = a.sessionService.SaveResult(ctx, &runtimesession.SaveResultInput{
			SessionID:    req.SessionID,
			RoundID:      req.RoundID,
			UserID:       req.UserID,
			TaskType:     req.TaskType,
			Status:       "failed",
			ErrorMessage: err.Error(),
		})
		a.taskManager.RemoveRoot(req.SessionID, req.RoundID)
		cancel()
		return nil, err
	}

	collector := runtimeaudit.NewSessionCollector()
	events := make(chan *aiapi.RuntimeStreamEvent, 16)

	// ④ 异步执行
	go a.run(runCtx, cancel, req, snapshot, collector, startedAt, events)
	return events, nil
}

// run 执行骨架
func (a *runner) run(
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
	defer a.taskManager.RemoveRoot(req.SessionID, req.RoundID)

	taskType := a.core.Type()

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
	execErr := a.core.Execute(ctx, req, snapshot, emit)
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
		_ = a.sessionService.SaveResult(context.Background(), saveInput)
	}
}

// ========== LLM 调用封装（子类可调用）==========

// StreamLLMWithTools 流式调用 LLM（带工具绑定）
func (a *runner) StreamLLMWithTools(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	return a.chatModel.Stream(ctx, messages, tools)
}

// callLLM 同步调用 LLM
func (a *runner) callLLM(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return a.chatModel.Call(ctx, messages)
}

// streamLLM 流式调用 LLM
func (a *runner) streamLLM(ctx context.Context, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	return a.chatModel.Stream(ctx, messages, nil)
}

// validate 验证依赖完整性
func (a *runner) validate() error {
	if a.sessionService == nil || a.taskManager == nil || a.contextManager == nil || a.chatModel == nil {
		return aiapi.ErrAssistantNotConfigured
	}
	return nil
}
