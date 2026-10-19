/*
@Author: omenkk7
@Date: 2026/02/26
@Description: 判题输出规范化与比对（避免 CRLF/BOM/题目 time_limit=0 等导致误判）
*/

package utils

import "strings"

const (
	// DefaultJudgeTimeLimitMS 题目或测试用例未配置时间限制时的默认毫秒数（避免传 0 给 go-judge 导致异常）
	DefaultJudgeTimeLimitMS = 1000
	// DefaultJudgeMemoryMB 题目或测试用例未配置内存限制时的默认 MB
	DefaultJudgeMemoryMB = 256
)

// NormalizeJudgeOutput 规范化后再比对：统一换行、去掉 UTF-8 BOM、去掉首尾空白
func NormalizeJudgeOutput(s string) string {
	s = strings.TrimPrefix(s, "\ufeff")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// CompareJudgeOutput 判题输出比对：逐行比较，忽略行尾空格与换行风格差异
func CompareJudgeOutput(actual, expected string) bool {
	actual = NormalizeJudgeOutput(actual)
	expected = NormalizeJudgeOutput(expected)

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

// EffectiveJudgeTimeLimitMS 返回有效时间限制（毫秒），<=0 时用默认值
func EffectiveJudgeTimeLimitMS(ms int) int {
	if ms > 0 {
		return ms
	}
	return DefaultJudgeTimeLimitMS
}

// EffectiveJudgeMemoryMB 返回有效内存限制（MB），<=0 时用默认值
func EffectiveJudgeMemoryMB(mb int) int {
	if mb > 0 {
		return mb
	}
	return DefaultJudgeMemoryMB
}
