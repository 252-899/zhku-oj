/*
@Author: Codex
@Date: 2026/4/21
@Name: coordinator_test.go
@Description: OJ 题目生成协调器测试，覆盖题目补齐等关键编排边界
*/

package ojquestiongen

import (
	"testing"

	"zk-code-arena-server/pkg/ai/modules/ojquestiongen/agents"
)

// TestEnsureProblemCountClonesMissingProblems 验证补齐题目时不会复用同一个指针和同一个 ID。
func TestEnsureProblemCountClonesMissingProblems(t *testing.T) {
	source := []*agents.GeneratedProblem{
		{
			ID:           "gen_1",
			Title:        "数组统计",
			Description:  "desc",
			InputFormat:  "in",
			OutputFormat: "out",
			Hint:         "hint",
			Difficulty:   "easy",
		},
	}

	result := ensureProblemCount(source, 2)
	if len(result) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(result))
	}
	if result[0] == result[1] {
		t.Fatalf("expected padded problem to be a cloned struct, but pointers are identical")
	}
	if result[0].ID == result[1].ID {
		t.Fatalf("expected padded problem to have a new id, got duplicated id %q", result[0].ID)
	}
	if result[1].Title != result[0].Title {
		t.Fatalf("expected padded problem to preserve content, got %q", result[1].Title)
	}
}
