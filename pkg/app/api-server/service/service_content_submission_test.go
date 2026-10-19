/*
@Author: Codex
@Date: 2026/05/08
@Description: 内容提交服务测试，约束最小作用域 key helper 行为
*/

package service

import "testing"

func TestStandaloneSubmissionKeyIncludesModeOnly(t *testing.T) {
	key := buildSubmissionScopeKey("standalone", "", "")
	if key != "standalone::" {
		t.Fatalf("expected standalone scope key, got %s", key)
	}
}
