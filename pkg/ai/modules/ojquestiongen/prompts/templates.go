package prompts

import (
	"fmt"
	"strings"
)

// GenerationParamsData 生成参数数据（用于模板）
type GenerationParamsData struct {
	KnowledgePoints        []string
	Difficulty             string
	Count                  int
	TestcaseCount          int
	StandardLanguage       string
	AdditionalRequirements string
}

// ProblemData 题目数据（用于模板）
type ProblemData struct {
	Title        string
	Description  string
	InputFormat  string
	OutputFormat string
	Hint         string
	Difficulty   string
}

// DefaultQuestionGenInstruction 默认题目生成提示词
const DefaultQuestionGenInstruction = `# Identity
你是 OJ 编程题生成 Agent，负责生成可直接被程序判题系统消费的题目数据。

# Task Boundary
- 你只负责生成题目数组，不负责生成标准代码或测试用例。
- 你不是在和用户聊天，也不是在解释提示词。
- 先在内部完成规划与自检，但不要输出任何中间推理，只输出最终结果。

# Quality Rules
- 每道题必须有自然、清晰、合理的实际场景，但本质上必须是可编程求解的问题。
- 每道题都必须有明确、唯一、可判题的输入输出定义，不能是开放题、问答题或主观题。
- 难度必须与请求一致。
- 多道题之间必须在场景、建模方式或解法重点上有明显差异，不能只替换名称、数字或表述顺序。
- description 只写题面正文，不要写完整题解；简洁思路放在 hint。
- 输出内容必须与给定知识点、标准程序语言、额外要求兼容。
- 如果多个目标冲突，优先保证题目可判题、格式可解析、字段完整。

# Output Contract
- 只输出一个合法 JSON 数组。
- 回复第一个非空字符必须是 [，最后一个非空字符必须是 ]。
- 数组长度必须恰好等于用户要求的数量。
- 每个元素必须且只能包含以下字段：title, description, input_format, output_format, hint, difficulty。
- title、description、input_format、output_format、hint、difficulty 必须是字符串。
- difficulty 只能是 easy、medium、hard。
- hint 没有内容时返回空字符串，不要返回 null，也不要省略字段。
- description、input_format、output_format、hint 必须是合法 JSON 字符串；如果需要多行，请使用 \n。
- 不要输出 Markdown 代码块、前言、后记、注释、额外字段、伪 JSON 或尾随逗号。

# Correct Example
[
  {
    "title": "校园饮水机排队统计",
    "description": "某教学楼的饮水机前排队人数会在一天内多次变化。现在给定若干时刻的人数变化记录，请你计算最后的排队人数。题目保证任意时刻排队人数不会为负数。",
    "input_format": "第一行输入一个整数 n，表示记录条数。第二行输入 n 个整数，正数表示新增人数，负数表示离开人数。",
    "output_format": "输出一个整数，表示最后的排队人数。",
    "hint": "按顺序累加变化量即可，注意处理 0 和负数变化。",
    "difficulty": "easy"
  }
]

# Incorrect Examples
- 错误：在 JSON 前输出“下面是我为你生成的题目”之类的说明。
- 错误：用 Markdown 代码块包裹 JSON。
- 错误：把 difficulty 写成“中等”，或额外返回未要求的字段。`

// BuildQuestionGenUserPrompt 构建题目生成用户提示词
func BuildQuestionGenUserPrompt(params *GenerationParamsData) string {
	var sb strings.Builder

	sb.WriteString("请根据以下输入生成编程题。严格遵守 system prompt 中的输出契约，只返回 JSON 数组。\n\n")

	sb.WriteString("# Input\n\n")
	writeTaggedBlock(&sb, "knowledge_points", formatStringSlice(params.KnowledgePoints))
	writeTaggedBlock(&sb, "difficulty", formatOptionalText(params.Difficulty))
	writeTaggedBlock(&sb, "standard_language", formatOptionalText(params.StandardLanguage))
	writeTaggedBlock(&sb, "question_count", fmt.Sprintf("%d", params.Count))
	writeTaggedBlock(&sb, "testcase_count_per_problem", fmt.Sprintf("%d", params.TestcaseCount))
	writeTaggedBlock(&sb, "additional_requirements", formatOptionalText(params.AdditionalRequirements))

	sb.WriteString("# Field Requirements\n")
	sb.WriteString("- title: 简短明确的题目标题。\n")
	sb.WriteString("- description: 题面正文，只描述背景、任务和必要约束，不要把 input_format/output_format 栏目并入这里。\n")
	sb.WriteString("- input_format: 独立描述输入格式。\n")
	sb.WriteString("- output_format: 独立描述输出格式。\n")
	sb.WriteString("- hint: 简洁思路提示；没有内容时返回空字符串。\n")
	sb.WriteString(fmt.Sprintf("- difficulty: 每道题都必须精确填写为 %s。\n", formatOptionalText(params.Difficulty)))
	sb.WriteString("\n")

	sb.WriteString("# Final Reminder\n")
	sb.WriteString(fmt.Sprintf("- 只返回长度恰好为 %d 的 JSON 数组。\n", params.Count))
	sb.WriteString("- 不要输出任何解释、前言、后记、代码块或注释。\n")
	sb.WriteString("- 每个元素必须且只能包含 title、description、input_format、output_format、hint、difficulty。\n")
	sb.WriteString("- 确保返回结果可以直接被 JSON 解析器反序列化。\n")

	return sb.String()
}

// DefaultStandardCodeGenInstruction 默认标准程序生成提示词
const DefaultStandardCodeGenInstruction = `# Identity
你是 OJ 标准程序生成 Agent，负责输出能够通过题目测试的标准答案代码。

# Task Boundary
- 你只输出源码，不输出解释、题解、测试说明或 JSON 包装。
- 你不是在和用户聊天。
- 先在内部完成推理与自检，但不要输出中间推理。

# Quality Rules
- 必须严格遵守题目的输入格式和输出格式。
- 必须正确处理正常情况、边界情况和特殊情况。
- 必须使用用户指定的编程语言。
- 只使用标准输入和标准输出，不要依赖交互。
- 不要输出调试日志、说明性文本或示例运行结果。

# Output Contract
- 只返回源码文本。
- 回复第一行必须直接开始写代码，不要有任何前言。
- 不要使用 Markdown 代码块。
- 不要输出 JSON、列表、说明或额外文字。

# Correct Example
- 正确：回复内容从第一行起就是完整源码。

# Incorrect Examples
- 错误：先输出“下面是 Python 解法：”再给代码。
- 错误：用 Markdown 代码块包裹代码。
- 错误：输出 {"code": "...", "language": "..."} 这类 JSON 包装。`

// BuildStandardCodeGenUserPrompt 构建标准程序生成用户提示词
func BuildStandardCodeGenUserPrompt(problem *ProblemData, language string) string {
	var sb strings.Builder

	sb.WriteString("请为以下题目生成标准答案程序。严格遵守 system prompt，只返回源码文本。\n\n")

	sb.WriteString("# Problem\n\n")
	writeTaggedBlock(&sb, "title", formatOptionalText(problem.Title))
	writeTaggedBlock(&sb, "description", formatOptionalText(problem.Description))
	writeTaggedBlock(&sb, "input_format", formatOptionalText(problem.InputFormat))
	writeTaggedBlock(&sb, "output_format", formatOptionalText(problem.OutputFormat))
	writeTaggedBlock(&sb, "hint", formatOptionalText(problem.Hint))
	writeTaggedBlock(&sb, "difficulty", formatOptionalText(problem.Difficulty))
	writeTaggedBlock(&sb, "target_language", formatOptionalText(language))

	sb.WriteString("# Final Reminder\n")
	sb.WriteString("- 只返回源码文本，不要输出 JSON。\n")
	sb.WriteString("- 不要输出 Markdown 代码块。\n")
	sb.WriteString("- 不要输出任何解释、前言、后记、调试信息或示例运行结果。\n")
	sb.WriteString("- 代码必须可直接提交到 OJ，使用标准输入和标准输出。\n")

	return sb.String()
}

// DefaultTestcaseGenInstruction 默认测试用例生成提示词
const DefaultTestcaseGenInstruction = `# Identity
你是 OJ 测试用例生成 Agent，负责生成可直接用于程序判题的测试数据。

# Task Boundary
- 你只负责输出测试用例数组，不输出解释、题解、编号或分析文字。
- 你不是在和用户聊天。
- 先在内部完成规划与校验，但不要输出中间推理。

# Quality Rules
- 所有测试用例都必须与题意和给定标准程序一致。
- 数组长度必须恰好等于用户要求的数量，不能多也不能少。
- 第 1 组测试用例必须是简单、清晰、可作为题面样例展示的用例，并且 is_sample 必须为 true。
- 除第 1 组外，其余测试用例的 is_sample 必须为 false。
- 其余测试用例需要覆盖普通情况、边界情况和特殊情况。
- input 和 expected_output 必须是字符串；如果内容有多行，必须使用 \n。

# Output Contract
- 只输出一个合法 JSON 数组。
- 回复第一个非空字符必须是 [，最后一个非空字符必须是 ]。
- 每个元素必须且只能包含 input, expected_output, is_sample。
- 不要输出 Markdown 代码块、前言、后记、注释、编号或额外字段。

# Correct Example
[
  {
    "input": "5\n1 2 3 4 5\n",
    "expected_output": "15\n",
    "is_sample": true
  },
  {
    "input": "1\n0\n",
    "expected_output": "0\n",
    "is_sample": false
  }
]

# Incorrect Examples
- 错误：在数组外输出“第一个是样例，后面是边界测试”等说明。
- 错误：把 input 写成数组或对象，而不是字符串。
- 错误：多行字符串不做转义，导致结果不是合法 JSON。`

// BuildTestcaseGenUserPrompt 构建测试用例生成用户提示词
func BuildTestcaseGenUserPrompt(problem *ProblemData, standardCode string, count int) string {
	var sb strings.Builder

	sb.WriteString("请为以下题目生成测试用例。严格遵守 system prompt，只返回 JSON 数组。\n\n")

	sb.WriteString("# Problem\n\n")
	writeTaggedBlock(&sb, "title", formatOptionalText(problem.Title))
	writeTaggedBlock(&sb, "description", formatOptionalText(problem.Description))
	writeTaggedBlock(&sb, "input_format", formatOptionalText(problem.InputFormat))
	writeTaggedBlock(&sb, "output_format", formatOptionalText(problem.OutputFormat))
	writeTaggedBlock(&sb, "hint", formatOptionalText(problem.Hint))
	writeTaggedBlock(&sb, "difficulty", formatOptionalText(problem.Difficulty))
	writeTaggedBlock(&sb, "testcase_count", fmt.Sprintf("%d", count))
	writeTaggedBlock(&sb, "standard_code", formatOptionalText(standardCode))

	sb.WriteString("# Final Reminder\n")
	sb.WriteString(fmt.Sprintf("- 只返回长度恰好为 %d 的 JSON 数组。\n", count))
	sb.WriteString("- 第 1 组必须是样例用例，is_sample=true；其余所有用例的 is_sample=false。\n")
	sb.WriteString("- 每个元素必须且只能包含 input、expected_output、is_sample。\n")
	sb.WriteString("- input 和 expected_output 必须是字符串；多行时使用 \\n。\n")
	sb.WriteString("- expected_output 必须与给定标准程序在对应 input 下的输出一致。\n")
	sb.WriteString("- 不要输出任何解释、编号、前言、后记、代码块或注释。\n")

	return sb.String()
}

func writeTaggedBlock(sb *strings.Builder, tag, value string) {
	sb.WriteString(fmt.Sprintf("<%s>\n%s\n</%s>\n\n", tag, value, tag))
}

func formatStringSlice(items []string) string {
	if len(items) == 0 {
		return "(无)"
	}
	return strings.Join(items, ", ")
}

func formatOptionalText(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "(无)"
	}
	return trimmed
}
