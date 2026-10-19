package utils

import "testing"

func TestCompareJudgeOutput(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"1\n2", "1\n2", true},
		{"1\r\n2", "1\n2", true},
		{"\ufeff1\n2", "1\n2", true},
		{"  1  \n  2  ", "1\n2", true},
		{"1", "2", false},
		{"1\n2", "1", false},
	}
	for _, tt := range tests {
		if got := CompareJudgeOutput(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareJudgeOutput(%q,%q)=%v want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestEffectiveJudgeTimeLimitMS(t *testing.T) {
	if EffectiveJudgeTimeLimitMS(0) != DefaultJudgeTimeLimitMS {
		t.Fatal()
	}
	if EffectiveJudgeTimeLimitMS(500) != 500 {
		t.Fatal()
	}
}
