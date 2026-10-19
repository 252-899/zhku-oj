/*
Package models - AI 相关类型定义
用于题目生成、测试用例生成等 AI 功能的请求/响应结构体
*/
package models

// ---------------------- 题目生成相关 ----------------------

// ProblemGenerationInput 题目生成请求
type ProblemGenerationInput struct {
	Topic      string   `json:"topic"`      // 题目主题
	Difficulty string   `json:"difficulty"` // 难度: easy, medium, hard
	Tags       []string `json:"tags"`       // 标签
	ExtraReq   string   `json:"extra_req"`  // 额外要求
}

// GeneratedProblem AI 生成的题目结构
type GeneratedProblem struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	InputDesc    string `json:"input_desc"`
	OutputDesc   string `json:"output_desc"`
	SampleInput  string `json:"sample_input"`
	SampleOutput string `json:"sample_output"`
	Hint         string `json:"hint"`
	TimeLimit    int    `json:"time_limit"`   // 毫秒
	MemoryLimit  int    `json:"memory_limit"` // MB
}

// ---------------------- 测试用例生成相关 ----------------------

// TestCaseGenerationInput 测试用例生成请求
type TestCaseGenerationInput struct {
	ProblemID   string   `json:"problem_id"`
	Description string   `json:"description"`  // 题目描述
	InputDesc   string   `json:"input_desc"`   // 输入格式
	Constraints string   `json:"constraints"`  // 数据范围约束
	StdCode     string   `json:"std_code"`     // 标准解答代码
	StdLanguage string   `json:"std_language"` // 标程语言
	Count       int      `json:"count"`        // 生成数量
	Strategies  []string `json:"strategies"`   // 生成策略: random, edge, min_max
	TimeLimit   int64    `json:"time_limit"`   // 运行时限 (ms)
	MemoryLimit int64    `json:"memory_limit"` // 内存限制 (bytes)
}

// GeneratedTestCase AI 生成的测试用例
type GeneratedTestCase struct {
	Input       string `json:"input"`       // 输入数据
	Output      string `json:"output"`      // 输出数据 (由标程运行得出)
	Description string `json:"description"` // 用例描述
	TimeCost    int64  `json:"time_cost"`   // 运行耗时 (ms)
	MemoryCost  int64  `json:"memory_cost"` // 内存消耗
	IsValid     bool   `json:"is_valid"`    // 是否有效
}

// ---------------------- 生成任务状态相关 ----------------------

// GenerationTaskStatus 生成任务状态
type GenerationTaskStatus string

const (
	TaskStatusProcessing    GenerationTaskStatus = "processing"     // 生成中
	TaskStatusWaitingReview GenerationTaskStatus = "waiting_review" // 等待审核
	TaskStatusFailed        GenerationTaskStatus = "failed"         // 生成失败
	TaskStatusSaved         GenerationTaskStatus = "saved"          // 已保存
)

// QuestionGenerationSession 题目生成会话
type QuestionGenerationSession struct {
	SessionID    string               `json:"session_id"`
	Status       GenerationTaskStatus `json:"status"`
	Step         string               `json:"step"` // 当前步骤
	Problem      *GeneratedProblem    `json:"problem,omitempty"`
	TestCases    []*GeneratedTestCase `json:"test_cases,omitempty"`
	StdCode      string               `json:"std_code,omitempty"`
	VerifyResult string               `json:"verify_result,omitempty"`
	Logs         []string             `json:"logs,omitempty"`
	RetryCount   int                  `json:"retry_count"`
}
