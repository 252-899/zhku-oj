// class_analyst.go 实现教师侧班级范围解析 Agent。
package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/multiagent"
	"zk-code-arena-server/pkg/ai/runtime/react"
	aitools "zk-code-arena-server/pkg/ai/tools"
	ojtools "zk-code-arena-server/pkg/ai/tools/oj"

	"github.com/cloudwego/eino/schema"
)

const classAnalystSystemPrompt = `你是 OJ 综合学习平台的 class_analyst。

你的职责：
1. 根据教师问题和上下文解析班级范围、作业任务范围。
2. 当只有班级名称时，优先调用 oj_list_teacher_classes 查询当前教师可见班级。
3. 拿到 class_id 后，调用 oj_list_class_tasks 查询该班级任务列表。
4. 如果教师提到“数组作业”“这次作业”等任务线索，请从任务标题、描述和时间中选择最可能的 task_id。
5. 输出必须包含可供后续 Agent 使用的 class_id、class_name、task_id、task_name；不确定时说明原因。
6. 不要编造 ID，不要绕过工具结果。`

// ClassAnalystRunner 使用显式 ReAct 循环解析班级和作业任务范围。
type ClassAnalystRunner struct {
	model    llm.ModelFactory
	provider *ojtools.Provider
	req      *aiapi.RuntimeRequest
	emit     react.EmitFunc
}

// NewClassAnalystRunner 创建班级范围解析 Agent。
func NewClassAnalystRunner(
	model llm.ModelFactory,
	provider *ojtools.Provider,
	req *aiapi.RuntimeRequest,
	emit react.EmitFunc,
) *ClassAnalystRunner {
	return &ClassAnalystRunner{
		model:    model,
		provider: provider,
		req:      req,
		emit:     emit,
	}
}

// Run 执行 class_analyst 子任务，并返回结构化文本结果。
func (r *ClassAnalystRunner) Run(ctx context.Context, task multiagent.AgentTask) (*multiagent.AgentResult, error) {
	if r == nil || r.model == nil || r.provider == nil {
		return nil, fmt.Errorf("class analyst runner is not configured")
	}
	tools, err := r.provider.BuildTools(ctx)
	if err != nil {
		return nil, err
	}
	runner := react.NewRunner(
		r.model,
		r.provider,
		aitools.ToSchemaToolInfos(tools),
		react.Config{MaxRounds: 6, Stage: "class_analyst"},
	)

	messages, err := runner.Run(ctx, r.req, r.buildMessages(task), r.emit)
	if err != nil {
		return nil, err
	}
	content := lastAssistantContent(messages)
	return &multiagent.AgentResult{
		AgentID: task.AgentID,
		TaskID:  task.ID,
		Role:    task.Role,
		Content: content,
		Data: map[string]any{
			"agent": "class_analyst",
		},
		Success: true,
	}, nil
}

// buildMessages 构造 class_analyst 的系统提示词和任务上下文。
func (r *ClassAnalystRunner) buildMessages(task multiagent.AgentTask) []*schema.Message {
	payload, _ := json.Marshal(map[string]any{
		"original_query": task.OriginalQuery,
		"task":           task.Description,
		"context":        task.Context,
	})
	return []*schema.Message{
		schema.SystemMessage(react.DefaultSystemPrompt + "\n\n" + classAnalystSystemPrompt),
		schema.UserMessage(string(payload)),
	}
}

// lastAssistantContent 读取 ReAct 循环最后一条自然语言回答。
func lastAssistantContent(messages []*schema.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i] != nil && messages[i].Role == schema.Assistant {
			if content := strings.TrimSpace(messages[i].Content); content != "" {
				return content
			}
		}
	}
	return ""
}
