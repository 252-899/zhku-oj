/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型选择题模型测试
*/

package models

import "testing"

func TestChoiceQuestionValidateRejectsInvalidChoiceMode(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: "",
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"A"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected invalid choice mode validation failure")
	}
}

func TestChoiceQuestionValidateSingleChoice(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeSingle,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"A"},
	}
	if err := q.Validate(); err != nil {
		t.Fatalf("expected single choice question to be valid, got %v", err)
	}
}

func TestChoiceQuestionRejectsMultipleAnswersForSingleChoice(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeSingle,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"A", "B"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected validation error for single choice with multiple answers")
	}
}

func TestChoiceQuestionRejectsUnknownCorrectAnswer(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeMultiple,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"A", "C"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected validation error for unknown correct answer")
	}
}

func TestChoiceQuestionRejectsDuplicateOptionKey(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeMultiple,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "A", Content: "2"},
		},
		CorrectAnswers: []string{"A"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected duplicate option key validation failure")
	}
}

func TestChoiceQuestionRejectsDuplicateCorrectAnswer(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeMultiple,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"A", "A"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected duplicate correct answer validation failure")
	}
}

func TestChoiceQuestionRejectsOptionKeyWithSurroundingSpaces(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeSingle,
		Options: []ChoiceOption{
			{Key: " A ", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{"B"},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected surrounding space option key validation failure")
	}
}

func TestChoiceQuestionRejectsCorrectAnswerWithSurroundingSpaces(t *testing.T) {
	q := &ChoiceQuestion{
		ChoiceMode: ChoiceModeSingle,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		CorrectAnswers: []string{" A "},
	}
	if err := q.Validate(); err == nil {
		t.Fatalf("expected surrounding space answer validation failure")
	}
}
