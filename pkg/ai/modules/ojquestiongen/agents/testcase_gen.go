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

const testcaseGenAgentType = "oj_testcase_gen_agent"

// TestcaseGenAgent 测试用例生成 Agent
type TestcaseGenAgent struct {
	chatModel    llm.ModelFactory
	systemPrompt string
}

// NewTestcaseGenAgent 创建测试用例生成 Agent
func NewTestcaseGenAgent(
	chatModel llm.ModelFactory,
	systemPrompt string,
) *TestcaseGenAgent {
	if systemPrompt == "" {
		systemPrompt = prompts.DefaultTestcaseGenInstruction
	}
	return &TestcaseGenAgent{
		chatModel:    chatModel,
		systemPrompt: systemPrompt,
	}
}

// ChatModel 返回 LLM 模型工厂
func (a *TestcaseGenAgent) ChatModel() llm.ModelFactory {
	return a.chatModel
}

// Type 返回 Agent 类型标识
func (a *TestcaseGenAgent) Type() string {
	return testcaseGenAgentType
}

// Process 主入口 - 封装 LLM 调用 + 解析
func (a *TestcaseGenAgent) Process(
	ctx context.Context,
	problem *GeneratedProblem,
	standardCode string,
	count int,
) ([]TestCase, error) {
	// 1. 构建消息
	messages := a.buildMessages(problem, standardCode, count)

	// 2. 调用 LLM
	resp, err := a.chatModel.Call(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 3. 解析结果
	testCases, err := a.parseTestCases(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("解析测试用例失败: %w", err)
	}

	return testCases, nil
}

// buildMessages 构建 LLM 消息列表
func (a *TestcaseGenAgent) buildMessages(problem *GeneratedProblem, standardCode string, count int) []*schema.Message {
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
		schema.UserMessage(prompts.BuildTestcaseGenUserPrompt(problemData, standardCode, count)),
	}
}

// parseTestCases 解析 LLM 返回的测试用例
func (a *TestcaseGenAgent) parseTestCases(content string) ([]TestCase, error) {
	jsonContent := extractTestCaseJSON(content)
	if jsonContent == "" {
		return nil, fmt.Errorf("无法提取 JSON")
	}

	var testCases []TestCase
	if err := json.Unmarshal([]byte(jsonContent), &testCases); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}

	if len(testCases) == 0 {
		return nil, fmt.Errorf("没有生成测试用例")
	}

	// 确保第一个是示例
	if len(testCases) > 0 && !testCases[0].IsSample {
		testCases[0].IsSample = true
	}

	return testCases, nil
}

// extractTestCaseJSON 从内容中提取 JSON 数组
func extractTestCaseJSON(content string) string {
	start := strings.Index(content, "[")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(content, "]")
	if end == -1 || end < start {
		return ""
	}
	return strings.TrimSpace(content[start : end+1])
}

// BuildMessages 构建 LLM 消息列表（保留给未来聊天 Agent 使用）
func (a *TestcaseGenAgent) BuildMessages(snapshot *runtimecontext.Snapshot, question string) []*schema.Message {
	cap := 2
	if snapshot != nil {
		cap += len(snapshot.Messages)
	}
	messages := make([]*schema.Message, 0, cap)

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
