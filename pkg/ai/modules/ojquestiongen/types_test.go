/*
@Author: Codex
@Date: 2026/4/21
@Name: types_test.go
@Description: OJ 题目生成参数解析测试，覆盖不同输入类型下的 count/testcase_count 解析行为
*/

package ojquestiongen

import (
	"testing"

	aiapi "zk-code-arena-server/pkg/ai/api"
)

// TestParseGenerationParamsSupportsIntegerInputs 验证业务接口直接写入 int 时不会退回默认值。
func TestParseGenerationParamsSupportsIntegerInputs(t *testing.T) {
	req := &aiapi.RuntimeRequest{
		Input: map[string]any{
			"knowledge_points": []string{"数组", "动态规划"},
			"count":            3,
			"testcase_count":   5,
		},
	}

	params := ParseGenerationParams(req)

	if params.Count != 3 {
		t.Fatalf("expected count=3, got %d", params.Count)
	}
	if params.TestcaseCount != 5 {
		t.Fatalf("expected testcase_count=5, got %d", params.TestcaseCount)
	}
}

// TestParseGenerationParamsSupportsFloatInputs 验证兼容通用网关等场景下的 float64 数字输入。
func TestParseGenerationParamsSupportsFloatInputs(t *testing.T) {
	req := &aiapi.RuntimeRequest{
		Input: map[string]any{
			"knowledge_points": []any{"数组"},
			"count":            float64(4),
			"testcase_count":   float64(6),
		},
	}

	params := ParseGenerationParams(req)

	if params.Count != 4 {
		t.Fatalf("expected count=4, got %d", params.Count)
	}
	if params.TestcaseCount != 6 {
		t.Fatalf("expected testcase_count=6, got %d", params.TestcaseCount)
	}
}
