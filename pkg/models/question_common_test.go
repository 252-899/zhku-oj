/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型共享元数据与统一关系模型测试
*/

package models

import "testing"

func TestPublishModeValidationValues(t *testing.T) {
	valid := []QuestionPublishMode{
		QuestionPublishModeGlobal,
		QuestionPublishModeCourseOnly,
		QuestionPublishModeGlobalAndCourse,
	}
	if len(valid) != 3 {
		t.Fatalf("expected 3 valid publish modes, got %d", len(valid))
	}

	testCases := []struct {
		name        string
		publishMode QuestionPublishMode
		courseIDs   []string
		wantErr     bool
	}{
		{
			name:        "global without course ids",
			publishMode: QuestionPublishModeGlobal,
			courseIDs:   nil,
			wantErr:     false,
		},
		{
			name:        "global rejects course ids",
			publishMode: QuestionPublishModeGlobal,
			courseIDs:   []string{"507f1f77bcf86cd799439011"},
			wantErr:     true,
		},
		{
			name:        "course only requires course ids",
			publishMode: QuestionPublishModeCourseOnly,
			courseIDs:   nil,
			wantErr:     true,
		},
		{
			name:        "global and course requires course ids",
			publishMode: QuestionPublishModeGlobalAndCourse,
			courseIDs:   []string{"507f1f77bcf86cd799439011"},
			wantErr:     false,
		},
	}

	for _, tc := range testCases {
		if err := ValidatePublishModeCourseIDs(tc.publishMode, tc.courseIDs); (err != nil) != tc.wantErr {
			t.Fatalf("%s: expected wantErr=%v, got err=%v", tc.name, tc.wantErr, err)
		}
	}
}

func TestQuestionStatusValidationValues(t *testing.T) {
	valid := []QuestionStatus{
		QuestionStatusDraft,
		QuestionStatusPublished,
		QuestionStatusArchived,
	}
	if len(valid) != 3 {
		t.Fatalf("expected 3 valid question statuses, got %d", len(valid))
	}
}

func TestContentRelationSupportsThreeQuestionTypes(t *testing.T) {
	types := []ContentType{
		ContentTypeProgramming,
		ContentTypeChoice,
		ContentTypeBlank,
	}
	if len(types) != 3 {
		t.Fatalf("expected 3 content types, got %d", len(types))
	}
}

func TestContentSubmissionSupportsStandaloneAndTask(t *testing.T) {
	modes := []SubmissionMode{
		SubmissionModeStandalone,
		SubmissionModeTask,
	}
	if len(modes) != 2 {
		t.Fatalf("expected 2 submission modes, got %d", len(modes))
	}
}
