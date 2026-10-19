package ojquestiongen

import (
	"context"
	"fmt"
	"strings"

	"zk-code-arena-server/pkg/common/sandbox"
)

// Verifier 判题验证器（轻量级，不走 JudgeService）
// 直接调用 sandbox 验证标准程序的正确性
type Verifier struct {
	sandboxClient sandbox.Client
}

// NewVerifier 创建验证器
func NewVerifier(sandboxClient sandbox.Client) *Verifier {
	return &Verifier{sandboxClient: sandboxClient}
}

// Verify 验证标准程序在测试用例上的正确性
func (v *Verifier) Verify(ctx context.Context, code, lang string, testCases []TestCase) (*VerifyResult, error) {
	if len(testCases) == 0 {
		return &VerifyResult{Passed: false, Message: "没有测试用例"}, nil
	}

	// 1. 获取语言配置
	langConfig := v.sandboxClient.GetLanguageConfig(lang)

	// 2. 编译（如需要）
	var executableID string

	if langConfig != nil && langConfig.Compile != nil {
		compileResp, err := v.sandboxClient.CompileCode(ctx, &sandbox.CompileRequest{
			Language:   lang,
			SourceCode: code,
		})
		if err != nil {
			return &VerifyResult{Passed: false, Message: fmt.Sprintf("编译失败: %v", err)}, nil
		}
		if !compileResp.Success {
			return &VerifyResult{
				Passed:     false,
				Message:    fmt.Sprintf("编译错误: %s", compileResp.CompileError),
				FailedCase: 1,
			}, nil
		}
		executableID = compileResp.ExecutableID
	} else {
		// 解释型语言，使用源代码作为 executableID
		executableID = code
	}

	// 3. 逐测试用例运行
	for i, tc := range testCases {
		caseNum := i + 1

		runResp, err := v.sandboxClient.RunCode(ctx, &sandbox.RunRequest{
			Language:     lang,
			ExecutableID: executableID,
			Input:        tc.Input,
			TimeLimit:    2_000_000_000,    // 2s in nanoseconds
			MemoryLimit:  256 * 1024 * 1024, // 256MB in bytes
		})
		if err != nil {
			return &VerifyResult{
				Passed:     false,
				Message:    fmt.Sprintf("运行失败: %v", err),
				FailedCase: caseNum,
			}, nil
		}

		// 4. 对比输出
		actual := strings.TrimSpace(runResp.Output)
		expected := strings.TrimSpace(tc.ExpectedOutput)

		if !v.compareOutput(actual, expected) {
			return &VerifyResult{
				Passed:     false,
				Message:    fmt.Sprintf("用例%d: 期望=%s, 实际=%s", caseNum, expected, truncate(actual, 100)),
				FailedCase: caseNum,
			}, nil
		}
	}

	return &VerifyResult{Passed: true, Message: "All test cases passed"}, nil
}

// compareOutput 比对输出（逐行比对，忽略行尾空格）
func (v *Verifier) compareOutput(actual, expected string) bool {
	actualLines := strings.Split(actual, "\n")
	expectedLines := strings.Split(expected, "\n")

	if len(actualLines) != len(expectedLines) {
		return false
	}

	for i := range actualLines {
		if strings.TrimRight(actualLines[i], " \t\r") != strings.TrimRight(expectedLines[i], " \t\r") {
			return false
		}
	}

	return true
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
