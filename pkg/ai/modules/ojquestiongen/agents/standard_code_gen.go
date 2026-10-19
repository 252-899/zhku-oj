package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/ai/modules/ojquestiongen/prompts"
	runtimecontext "zk-code-arena-server/pkg/ai/runtime/context"

	"github.com/cloudwego/eino/schema"
)

const standardCodeGenAgentType = "oj_standard_code_gen_agent"

// StandardCodeGenAgent 标准程序生成 Agent
type StandardCodeGenAgent struct {
	chatModel    llm.ModelFactory
	systemPrompt string
}

// NewStandardCodeGenAgent 创建标准程序生成 Agent
func NewStandardCodeGenAgent(
	chatModel llm.ModelFactory,
	systemPrompt string,
) *StandardCodeGenAgent {
	if systemPrompt == "" {
		systemPrompt = prompts.DefaultStandardCodeGenInstruction
	}
	return &StandardCodeGenAgent{
		chatModel:    chatModel,
		systemPrompt: systemPrompt,
	}
}

// ChatModel 返回 LLM 模型工厂
func (a *StandardCodeGenAgent) ChatModel() llm.ModelFactory {
	return a.chatModel
}

// Type 返回 Agent 类型标识
func (a *StandardCodeGenAgent) Type() string {
	return standardCodeGenAgentType
}

// Process 主入口 - 封装 LLM 调用 + 解析
func (a *StandardCodeGenAgent) Process(
	ctx context.Context,
	problem *GeneratedProblem,
	language string,
) (string, error) {
	// 1. 构建消息
	messages := a.buildMessages(problem, language)

	// 2. 调用 LLM
	resp, err := a.chatModel.Call(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 3. 解析结果
	code, err := a.parseCode(resp.Content, language)
	if err != nil {
		return "", fmt.Errorf("解析标准程序失败: %w", err)
	}

	return code, nil
}

// buildMessages 构建 LLM 消息列表
func (a *StandardCodeGenAgent) buildMessages(problem *GeneratedProblem, language string) []*schema.Message {
	problemData := &prompts.ProblemData{
		Title:        problem.Title,
		Description:  problem.Description,
		InputFormat:  problem.InputFormat,
		OutputFormat: problem.OutputFormat,
		Hint:         problem.Hint,
		Difficulty:   problem.Difficulty,
	}

	return []*schema.Message{
		schema.SystemMessage(a.systemPrompt),
		schema.UserMessage(prompts.BuildStandardCodeGenUserPrompt(problemData, language)),
	}
}

// parseCode 解析 LLM 返回的标准程序
func (a *StandardCodeGenAgent) parseCode(content, language string) (string, error) {
	jsonContent := extractCodeJSON(content)
	if jsonContent == "" {
		return strings.TrimSpace(content), nil
	}

	var result struct {
		Code     string `json:"code"`
		Language string `json:"language"`
	}
	if err := json.Unmarshal([]byte(jsonContent), &result); err == nil && result.Code != "" {
		return result.Code, nil
	}

	return strings.TrimSpace(content), nil
}

// extractCodeJSON 从内容中提取 JSON
func extractCodeJSON(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(content, "}")
	if end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(content[start : end+1])
}

// BuildMessages 构建 LLM 消息列表（保留给未来聊天 Agent 使用）
func (a *StandardCodeGenAgent) BuildMessages(snapshot *runtimecontext.Snapshot, question string) []*schema.Message {
	count := 2
	if snapshot != nil {
		count += len(snapshot.Messages)
	}
	messages := make([]*schema.Message, 0, count)

	messages = append(messages, schema.SystemMessage(a.systemPrompt))

	if snapshot != nil {
		for _, item := range snapshot.Messages {
			switch item.Role {
			case "assistant":
				messages = append(messages, schema.AssistantMessage(item.Content, nil))
			default:
				messages = append(messages, schema.UserMessage(item.Content))
			}
		}
	}

	messages = append(messages, schema.UserMessage(question))
	return messages
}
