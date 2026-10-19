/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型请求参数测试
*/

package models

import "testing"

func TestCreateChoiceQuestionRequest(t *testing.T) {
	req := &CreateChoiceQuestionRequest{
		Title:       "选择题",
		Stem:        "题干",
		Difficulty:  DifficultyEasy,
		PublishMode: QuestionPublishModeCourseOnly,
		CourseIDs:   []string{"507f1f77bcf86cd799439011"},
		ChoiceMode:  ChoiceModeSingle,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		Answers: []string{"A"},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected choice request to be valid, got %v", err)
	}

	req.CourseIDs = nil
	if err := req.Validate(); err == nil {
		t.Fatalf("expected publish mode and course ids validation failure")
	}
}

func TestCreateChoiceQuestionRequestRejectsDuplicateOptionKey(t *testing.T) {
	req := &CreateChoiceQuestionRequest{
		Title:       "选择题",
		Stem:        "题干",
		Difficulty:  DifficultyEasy,
		PublishMode: QuestionPublishModeGlobal,
		ChoiceMode:  ChoiceModeMultiple,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "A", Content: "2"},
		},
		Answers: []string{"A"},
	}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected duplicate option key validation failure")
	}
}

func TestCreateChoiceQuestionRequestRejectsUnknownAnswer(t *testing.T) {
	req := &CreateChoiceQuestionRequest{
		Title:       "选择题",
		Stem:        "题干",
		Difficulty:  DifficultyEasy,
		PublishMode: QuestionPublishModeGlobal,
		ChoiceMode:  ChoiceModeMultiple,
		Options: []ChoiceOption{
			{Key: "A", Content: "1"},
			{Key: "B", Content: "2"},
		},
		Answers: []string{"C"},
	}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected unknown answer validation failure")
	}
}

func TestCreateBlankQuestionRequest(t *testing.T) {
	req := &CreateBlankQuestionRequest{
		Title:       "填空题",
		Stem:        "题干",
		Difficulty:  DifficultyEasy,
		PublishMode: QuestionPublishModeGlobalAndCourse,
		CourseIDs:   []string{"507f1f77bcf86cd799439011"},
		JudgeMode:   BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang"}},
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected blank request to be valid, got %v", err)
	}

	req.CourseIDs = []string{}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected publish mode and course ids validation failure")
	}
}

func TestCreateBlankQuestionRequestRejectsDuplicateBlankOrder(t *testing.T) {
	req := &CreateBlankQuestionRequest{
		Title:       "填空题",
		Stem:        "题干",
		Difficulty:  DifficultyEasy,
		PublishMode: QuestionPublishModeGlobal,
		JudgeMode:   BlankJudgeModeExactMatch,
		Blanks: []BlankSlot{
			{Order: 1, Answers: []string{"golang"}},
			{Order: 1, Answers: []string{"gin"}},
		},
	}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected duplicate blank order validation failure")
	}
}

func TestSubmitContentAnswerRequest(t *testing.T) {
	req := &SubmitContentAnswerRequest{
		ContentType:   ContentTypeChoice,
		ContentID:     "507f1f77bcf86cd799439011",
		ChoiceAnswers: []string{"A"},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected choice submission request to be valid, got %v", err)
	}

	req.BlankAnswers = []BlankAnswerItem{{Order: 1, Answer: "x"}}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected mixed answer structure validation failure")
	}
}

func TestSubmitBlankAnswerRequestRejectsDuplicateOrder(t *testing.T) {
	req := &SubmitContentAnswerRequest{
		ContentType: ContentTypeBlank,
		ContentID:   "507f1f77bcf86cd799439011",
		BlankAnswers: []BlankAnswerItem{
			{Order: 1, Answer: "gin"},
			{Order: 1, Answer: "echo"},
		},
	}
	if err := req.Validate(); err == nil {
		t.Fatalf("expected duplicate blank answer order validation failure")
	}
}
