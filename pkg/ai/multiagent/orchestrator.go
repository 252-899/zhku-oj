// orchestrator.go 包含将 LeadAction 计划转换为子 Agent 执行的运行时，
// 并负责容量限制和依赖规则。
package multiagent

import (
	"context"
	"fmt"
	"sync"
)

// OrchestratorConfig 描述编排器的容量和并发限制。
type OrchestratorConfig struct {
	MaxAgents           int
	MaxConcurrentAgents int
}

// Orchestrator 协调一个 LeadPlanner 和一个 RoleRegistry。它保持领域无关，
// 不直接理解班级、作业或题库等业务概念。
type Orchestrator struct {
	lead     LeadPlanner
	registry *RoleRegistry
	config   OrchestratorConfig
}

// NewOrchestrator 应用保守默认值，让调用方可以先启动业务入口，
// 不必一开始就调整并发参数。
func NewOrchestrator(lead LeadPlanner, registry *RoleRegistry, config OrchestratorConfig) *Orchestrator {
	if config.MaxAgents <= 0 {
		config.MaxAgents = 4
	}
	if config.MaxConcurrentAgents <= 0 || config.MaxConcurrentAgents > config.MaxAgents {
		config.MaxConcurrentAgents = config.MaxAgents
	}
	return &Orchestrator{
		lead:     lead,
		registry: registry,
		config:   config,
	}
}

// Run 执行一次单轮 Lead 规划。后续可以在外层包装迭代循环，
// 让 Lead 在每个子 Agent 完成后修订计划。
func (o *Orchestrator) Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	if o == nil || o.lead == nil || o.registry == nil {
		return nil, fmt.Errorf("multiagent orchestrator is not configured")
	}
	// 如果调用方没有传入受限角色列表，则把当前业务入口注册的全部角色暴露给 Lead。
	if len(req.Roles) == 0 {
		req.Roles = o.registry.Roles()
	}

	decision, err := o.lead.Plan(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("lead plan failed: %w", err)
	}
	if decision == nil {
		return nil, fmt.Errorf("lead returned nil decision")
	}

	// 在启动任何子 Agent 前，先把 Lead 的 action 列表转换为本地任务图。
	// 这样可以尽早发现格式错误的计划。
	runState := newRunState(decision.Summary)
	if err := runState.applyActions(decision.Actions); err != nil {
		return nil, err
	}
	if err := o.runTasks(ctx, req, runState); err != nil {
		return nil, err
	}
	return runState.result(), nil
}

// runTasks 会反复查找依赖已完成的待执行任务。每轮执行可能解锁 DAG 的下一层，
// 例如 T1 完成后才解锁 T2。
func (o *Orchestrator) runTasks(ctx context.Context, req RunRequest, state *runState) error {
	for {
		ready := state.readyTasks()
		if len(ready) == 0 {
			break
		}
		if state.spawned+len(ready) > o.config.MaxAgents {
			return fmt.Errorf("lead requested %d agents, max is %d", state.spawned+len(ready), o.config.MaxAgents)
		}
		// ready 任务彼此没有依赖，可以按并发上限分批执行。
		for start := 0; start < len(ready); start += o.config.MaxConcurrentAgents {
			end := start + o.config.MaxConcurrentAgents
			if end > len(ready) {
				end = len(ready)
			}
			if err := o.runBatch(ctx, req, state, ready[start:end]); err != nil {
				return err
			}
		}
	}
	// 如果没有任务可执行但仍有任务 pending，说明 Lead 生成了无法满足的依赖图。
	if blocked := state.blockedTasks(); len(blocked) > 0 {
		return fmt.Errorf("tasks have unmet dependencies: %v", blocked)
	}
	return nil
}

// runBatch 启动一批受并发限制的子 Agent，并等待这一批全部完成后，
// 再让上层继续评估下一层依赖。
func (o *Orchestrator) runBatch(ctx context.Context, req RunRequest, state *runState, tasks []TaskSpec) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(tasks))
	resultCh := make(chan AgentResult, len(tasks))

	for _, task := range tasks {
		task := task
		role, factory, ok := o.registry.Get(task.Role)
		if !ok {
			return fmt.Errorf("agent role %q is not registered", task.Role)
		}
		runner, err := factory(ctx, role)
		if err != nil {
			return fmt.Errorf("create agent for role %q failed: %w", task.Role, err)
		}
		state.markRunning(task.ID)
		// 当前实现只生成单次运行内稳定的 agent id。
		// 未来接入工作流引擎后可以替换为持久化执行 id。
		state.spawned++
		agentID := fmt.Sprintf("%s-%d", task.Role, state.spawned)

		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := runner.Run(ctx, AgentTask{
				ID:            task.ID,
				AgentID:       agentID,
				Role:          task.Role,
				Description:   task.Description,
				ModelTier:     task.ModelTier,
				DependsOn:     task.DependsOn,
				OriginalQuery: req.Query,
				Context:       cloneMap(req.Context),
			})
			if err != nil {
				errCh <- fmt.Errorf("agent task %s failed: %w", task.ID, err)
				return
			}
			if result == nil {
				errCh <- fmt.Errorf("agent task %s returned nil result", task.ID)
				return
			}
			// 标准化必需身份字段，让 AgentRunner 实现可以专注业务输出。
			if result.TaskID == "" {
				result.TaskID = task.ID
			}
			if result.Role == "" {
				result.Role = task.Role
			}
			resultCh <- *result
		}()
	}

	wg.Wait()
	close(errCh)
	close(resultCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	for result := range resultCh {
		state.markCompleted(result)
	}
	return nil
}

// cloneMap 避免子 Agent 意外修改与兄弟 Agent 共享的 RunRequest 上下文。
func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
