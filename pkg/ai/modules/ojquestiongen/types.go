package ojquestiongen

import (
	"fmt"
	"strconv"
	"strings"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

const (
	// CanonicalModuleType 模块类型标识
	CanonicalModuleType = "oj_question_gen"
)

// GenerationParams 生成参数（来自 HTTP 请求）
type GenerationParams struct {
	KnowledgePoints        []string // 知识点列表
	Difficulty             string   // easy / medium / hard
	Count                  int      // 生成数量
	TestcaseCount          int      // 每题测试用例数量
	StandardLanguage       string   // 标准程序语言
	AdditionalRequirements string   // 额外需求
}

// ParseGenerationParams 从 RuntimeRequest 解析生成参数
func ParseGenerationParams(req *aiapi.RuntimeRequest) *GenerationParams {
	if req == nil {
		return &GenerationParams{}
	}

	params := &GenerationParams{
		Difficulty:             "medium",
		Count:                  1,
		TestcaseCount:          3,
		StandardLanguage:       "python",
		AdditionalRequirements: "",
	}

	// 从 Input 字段解析
	input := req.Input
	if input == nil {
		return params
	}

	if v, ok := input["knowledge_points"].([]any); ok {
		params.KnowledgePoints = toStringSlice(v)
	} else if v, ok := input["knowledge_points"].([]string); ok {
		params.KnowledgePoints = v
	}

	if v, ok := input["knowledge_points"].(string); ok && v != "" {
		// 可能是逗号分隔的字符串
		params.KnowledgePoints = strings.Split(v, ",")
		for i := range params.KnowledgePoints {
			params.KnowledgePoints[i] = strings.TrimSpace(params.KnowledgePoints[i])
		}
	}

	if v, ok := input["difficulty"].(string); ok {
		params.Difficulty = strings.TrimSpace(v)
	}

	if count, ok := parseIntegerInput(input["count"]); ok {
		params.Count = count
	}

	if testcaseCount, ok := parseIntegerInput(input["testcase_count"]); ok {
		params.TestcaseCount = testcaseCount
	}

	if v, ok := input["standard_language"].(string); ok {
		params.StandardLanguage = strings.TrimSpace(v)
	}

	if v, ok := input["additional_requirements"].(string); ok {
		params.AdditionalRequirements = strings.TrimSpace(v)
	}

	// 校验
	if params.Count <= 0 {
		params.Count = 1
	}
	if params.Count > 10 {
		params.Count = 10
	}
	if params.TestcaseCount <= 0 {
		params.TestcaseCount = 3
	}
	if params.TestcaseCount > 10 {
		params.TestcaseCount = 10
	}

	return params
}

// GeneratedProblem 生成中的题目
type GeneratedProblem struct {
	ID           string     `json:"id"`                      // 临时 UUID
	Title        string     `json:"title"`                   // 题目标题
	Description  string     `json:"description"`             // 题目描述（Markdown）
	InputFormat  string     `json:"input_format"`            // 输入描述
	OutputFormat string     `json:"output_format"`           // 输出描述
	Hint         string     `json:"hint,omitempty"`          // 提示（可选）
	Difficulty   string     `json:"difficulty"`              // easy / medium / hard
	StandardCode string     `json:"standard_code,omitempty"` // 标准程序代码
	Language     string     `json:"language,omitempty"`      // 标准程序语言
	TestCases    []TestCase `json:"test_cases,omitempty"`    // 测试用例

	Verification *VerifyResult `json:"verification,omitempty"` // 验证结果

	Status string `json:"status"` // generating/codegen/testcase/verifying/pending/confirmed/rejected
}

// TestCase 测试用例
type TestCase struct {
	Input          string `json:"input"`           // 输入
	ExpectedOutput string `json:"expected_output"` // 期望输出
	IsSample       bool   `json:"is_sample"`       // 是否是示例用例
}

// VerifyResult 验证结果
type VerifyResult struct {
	Passed     bool   `json:"passed"`
	Message    string `json:"message"`
	FailedCase int    `json:"failed_case,omitempty"` // 失败在第几组（1-based）
}

// ConfirmRequest 确认入库请求
type ConfirmRequest struct {
	SessionID  string   `json:"session_id"`
	ProblemIDs []string `json:"problem_ids"` // 确认入库的题目临时 ID
}

// ConfirmResponse 确认入库响应
type ConfirmResponse struct {
	ConfirmedCount int               `json:"confirmed_count"`
	FailedIDs      []string          `json:"failed_ids"`
	Errors         map[string]string `json:"errors"`
}

// toStringSlice 将 []any 转换为 []string
func toStringSlice(v []any) []string {
	result := make([]string, 0, len(v))
	for _, item := range v {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

// parseIntegerInput 兼容业务接口直传 int 与网关场景下 float64 的数字输入。
// 这里统一收口，避免上层因为 map[string]any 的动态类型差异而回退到默认值。
func parseIntegerInput(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case uint:
		return int(v), true
	case uint8:
		return int(v), true
	case uint16:
		return int(v), true
	case uint32:
		return int(v), true
	case uint64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, false
		}
		return parsed, true
	case fmt.Stringer:
		return parseIntegerInput(v.String())
	default:
		return 0, false
	}
}
