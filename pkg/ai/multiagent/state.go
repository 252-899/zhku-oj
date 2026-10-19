// state.go 保存一次编排运行中的可变状态，包括任务、任务状态、
// 子 Agent 结果以及最终 RunResult 投影。
package multiagent

import "fmt"

const (
	taskPending   = "pending"
	taskRunning   = "running"
	taskCompleted = "completed"
)

type runState struct {
	summary      string
	interimReply string
	finalReply   string
	tasks        map[string]TaskSpec
	status       map[string]string
	results      map[string]AgentResult
	order        []string
	spawned      int
}

// newRunState 基于 Lead 的规划摘要创建空任务图。
func newRunState(summary string) *runState {
	return &runState{
		summary: summary,
		tasks:   map[string]TaskSpec{},
		status:  map[string]string{},
		results: map[string]AgentResult{},
		order:   []string{},
	}
}

// applyActions 将 LeadAction 指令转换为本地状态。
// 这里是模型规划结果变成可校验、可执行数据的边界。
func (s *runState) applyActions(actions []LeadAction) error {
	for _, action := range actions {
		switch action.Type {
		case ActionInterimReply:
			s.interimReply = action.Content
		case ActionReply, ActionDone:
			s.finalReply = action.Content
		case ActionRevisePlan:
			for _, task := range action.Create {
				if err := s.addTask(task); err != nil {
					return err
				}
			}
			for _, task := range action.Update {
				if err := s.updateTask(task); err != nil {
					return err
				}
			}
		case ActionSpawnAgent:
			// spawn_agent 是创建单个任务的简写，适用于 Lead 不需要额外发送
			// revise_plan 的场景。
			task := TaskSpec{
				ID:          action.TaskID,
				Description: action.TaskDescription,
				Role:        action.Role,
				DependsOn:   action.DependsOn,
				ModelTier:   action.ModelTier,
			}
			if err := s.addOrMergeTask(task); err != nil {
				return err
			}
		case "":
			return fmt.Errorf("lead action type is empty")
		default:
			return fmt.Errorf("unsupported lead action type %q", action.Type)
		}
	}
	return nil
}

// addOrMergeTask 允许 spawn_agent 补全或合并已由 revise_plan 声明过的任务。
func (s *runState) addOrMergeTask(task TaskSpec) error {
	if existing, ok := s.tasks[task.ID]; ok {
		if task.Description != "" {
			existing.Description = task.Description
		}
		if task.Role != "" {
			existing.Role = task.Role
		}
		if len(task.DependsOn) > 0 {
			existing.DependsOn = task.DependsOn
		}
		if task.ModelTier != "" {
			existing.ModelTier = task.ModelTier
		}
		return s.updateTask(existing)
	}
	return s.addTask(task)
}

// addTask 校验并添加执行子 Agent 所需的最小任务字段。
func (s *runState) addTask(task TaskSpec) error {
	if task.ID == "" {
		return fmt.Errorf("task id is empty")
	}
	if task.Description == "" {
		return fmt.Errorf("task %s description is empty", task.ID)
	}
	if task.Role == "" {
		return fmt.Errorf("task %s role is empty", task.ID)
	}
	if _, exists := s.tasks[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}
	s.tasks[task.ID] = task
	s.status[task.ID] = taskPending
	s.order = append(s.order, task.ID)
	return nil
}

// updateTask 在校验任务仍可执行后替换已有任务。
func (s *runState) updateTask(task TaskSpec) error {
	if task.ID == "" {
		return fmt.Errorf("task id is empty")
	}
	if _, exists := s.tasks[task.ID]; !exists {
		return fmt.Errorf("task %s does not exist", task.ID)
	}
	if task.Description == "" {
		return fmt.Errorf("task %s description is empty", task.ID)
	}
	if task.Role == "" {
		return fmt.Errorf("task %s role is empty", task.ID)
	}
	s.tasks[task.ID] = task
	return nil
}

// readyTasks 返回所有依赖已完成的 pending 任务。返回顺序保持稳定，
// 便于测试和日志理解。
func (s *runState) readyTasks() []TaskSpec {
	ready := []TaskSpec{}
	for _, id := range s.order {
		if s.status[id] != taskPending {
			continue
		}
		task := s.tasks[id]
		if s.dependenciesMet(task) {
			ready = append(ready, task)
		}
	}
	return ready
}

// dependenciesMet 检查任务的每个依赖是否都已完成。
func (s *runState) dependenciesMet(task TaskSpec) bool {
	for _, dep := range task.DependsOn {
		if s.status[dep] != taskCompleted {
			return false
		}
	}
	return true
}

// markRunning 记录某个任务已经启动对应 Agent。
func (s *runState) markRunning(taskID string) {
	s.status[taskID] = taskRunning
}

// markCompleted 保存任务结果，并解锁依赖它的后续任务。
func (s *runState) markCompleted(result AgentResult) {
	s.results[result.TaskID] = result
	s.status[result.TaskID] = taskCompleted
}

// blockedTasks 找出因为依赖始终没有完成而无法执行的任务。
func (s *runState) blockedTasks() []string {
	blocked := []string{}
	for _, id := range s.order {
		if s.status[id] == taskPending {
			blocked = append(blocked, id)
		}
	}
	return blocked
}

// result 将内部 map 状态投影成 API 或 service 更容易消费的有序切片。
func (s *runState) result() *RunResult {
	tasks := make([]TaskSpec, 0, len(s.order))
	results := make([]AgentResult, 0, len(s.order))
	for _, id := range s.order {
		tasks = append(tasks, s.tasks[id])
		if result, ok := s.results[id]; ok {
			results = append(results, result)
		}
	}
	return &RunResult{
		Summary:      s.summary,
		InterimReply: s.interimReply,
		FinalReply:   s.finalReply,
		Tasks:        tasks,
		Results:      results,
		Metadata: map[string]interface{}{
			"spawned_agents": s.spawned,
		},
	}
}
