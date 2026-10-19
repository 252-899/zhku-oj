package models

import "testing"

func TestAIGeneratedProblemDraftFindsProblemByGeneratedID(t *testing.T) {
	draft := &AIGeneratedProblemDraft{
		Problems: []AIGeneratedDraftProblem{
			{GeneratedProblemID: "gen_1", Title: "数组求和"},
			{GeneratedProblemID: "gen_2", Title: "数组最大值"},
		},
	}

	problem, ok := draft.FindProblem("gen_1")
	if !ok {
		t.Fatalf("expected generated problem to be found")
	}
	if problem.Title != "数组求和" {
		t.Fatalf("expected title 数组求和, got %s", problem.Title)
	}
}

func TestAIGeneratedProblemSaveRequestRejectsInvisibleProblem(t *testing.T) {
	req := AIGeneratedProblemSaveRequest{
		DraftID: "507f1f77bcf86cd799439011",
		Items: []AIGeneratedProblemSaveItem{
			{GeneratedProblemID: "gen_1", IsPublic: false},
		},
	}

	if err := req.Validate(); err == nil {
		t.Fatalf("expected invisible problem to be rejected")
	}
}

func TestAIGeneratedProblemSaveRequestAllowsCoursePrivateProblem(t *testing.T) {
	req := AIGeneratedProblemSaveRequest{
		DraftID: "507f1f77bcf86cd799439011",
		Items: []AIGeneratedProblemSaveItem{
			{GeneratedProblemID: "gen_1", IsPublic: false, CourseIDs: []string{"507f1f77bcf86cd799439012"}},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("expected course private problem to be valid, got %v", err)
	}
}
