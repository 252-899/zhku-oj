/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型填空题模型测试
*/

package models

import "testing"

func TestBlankQuestionRejectsInvalidJudgeMode(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: "",
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang"}},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected invalid judge mode validation failure")
	}
}

func TestBlankQuestionValidateSingleJudgeMode(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang"}},
			{Order: 2, Answers: []string{"gin"}},
		},
	}
	if err := q.Validate(); err != nil {
		t.Fatalf("expected multi-blank question to be valid, got %v", err)
	}
}

func TestBlankQuestionRejectsEmptyAnswers(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: nil},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected empty answer validation failure")
	}
}

func TestBlankQuestionRejectsDuplicateBlankOrder(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang"}},
			{Order: 1, Answers: []string{"gin"}},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected duplicate blank order validation failure")
	}
}

func TestBlankQuestionRejectsDuplicateAnswers(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang", "golang"}},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected duplicate blank answer validation failure")
	}
}

func TestBlankQuestionRejectsBlankAnswerContent(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{" "}},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected blank answer content validation failure")
	}
}

func TestBlankQuestionRejectsAnswerWithSurroundingSpaces(t *testing.T) {
	q := &BlankQuestion{
		JudgeMode: BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{" golang "}},
		},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected surrounding space blank answer validation failure")
	}
}
