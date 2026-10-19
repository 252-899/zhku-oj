package repository

import (
	"testing"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestApplyAIGeneratedDraftSaveMarksSavedFailedAndDiscarded(t *testing.T) {
	savedProblemID := primitive.NewObjectID()
	draft := &models.AIGeneratedProblemDraft{
		Problems: []models.AIGeneratedDraftProblem{
			{GeneratedProblemID: "gen_1", Status: models.AIGeneratedDraftProblemGenerated},
			{GeneratedProblemID: "gen_2", Status: models.AIGeneratedDraftProblemGenerated},
			{GeneratedProblemID: "gen_3", Status: models.AIGeneratedDraftProblemGenerated},
		},
	}
	results := []models.AIGeneratedProblemSaveResult{
		{GeneratedProblemID: "gen_1", ProblemID: savedProblemID.Hex(), Success: true},
		{GeneratedProblemID: "gen_2", Success: false, Error: "保存失败"},
	}

	applyAIGeneratedDraftSaveResult(draft, results)

	if draft.Problems[0].Status != models.AIGeneratedDraftProblemSaved {
		t.Fatalf("expected gen_1 saved, got %s", draft.Problems[0].Status)
	}
	if draft.Problems[0].SavedProblemID == nil || *draft.Problems[0].SavedProblemID != savedProblemID {
		t.Fatalf("expected gen_1 saved problem id to be recorded")
	}
	if draft.Problems[1].Status != models.AIGeneratedDraftProblemFailed || draft.Problems[1].SaveError == "" {
		t.Fatalf("expected gen_2 failed with error, got status=%s error=%q", draft.Problems[1].Status, draft.Problems[1].SaveError)
	}
	if draft.Problems[2].Status != models.AIGeneratedDraftProblemDiscarded {
		t.Fatalf("expected unselected gen_3 discarded, got %s", draft.Problems[2].Status)
	}
	if draft.Status != models.AIGeneratedProblemDraftPartiallySaved {
		t.Fatalf("expected draft partially saved, got %s", draft.Status)
	}
}

func TestApplyAIGeneratedDraftSaveMarksDraftSavedWhenNoFailures(t *testing.T) {
	savedProblemID := primitive.NewObjectID()
	draft := &models.AIGeneratedProblemDraft{
		Problems: []models.AIGeneratedDraftProblem{
			{GeneratedProblemID: "gen_1", Status: models.AIGeneratedDraftProblemGenerated},
			{GeneratedProblemID: "gen_2", Status: models.AIGeneratedDraftProblemGenerated},
		},
	}
	results := []models.AIGeneratedProblemSaveResult{
		{GeneratedProblemID: "gen_1", ProblemID: savedProblemID.Hex(), Success: true},
	}

	applyAIGeneratedDraftSaveResult(draft, results)

	if draft.Problems[0].Status != models.AIGeneratedDraftProblemSaved {
		t.Fatalf("expected gen_1 saved, got %s", draft.Problems[0].Status)
	}
	if draft.Problems[1].Status != models.AIGeneratedDraftProblemDiscarded {
		t.Fatalf("expected gen_2 discarded, got %s", draft.Problems[1].Status)
	}
	if draft.Status != models.AIGeneratedProblemDraftSaved {
		t.Fatalf("expected draft saved, got %s", draft.Status)
	}
}
