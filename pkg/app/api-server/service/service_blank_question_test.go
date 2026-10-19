/*
@Author: Codex
@Date: 2026/05/08
@Description: 填空题服务最小骨架测试，约束创建前预处理的默认行为
*/

package service

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestBlankQuestionPrepareDefaultsTagsAndTimestamps(t *testing.T) {
	svc := &BlankQuestionService{}
	q := &models.BlankQuestion{
		Title:       "填空题",
		Stem:        "Go Web 框架是 ____",
		PublishMode: models.QuestionPublishModeGlobal,
		JudgeMode:   models.BlankJudgeModeExactMatch,
		Blanks: []models.BlankSlot{
			{Order: 1, Answers: []string{"gin"}},
		},
		CreatedBy: primitive.NewObjectID(),
	}
	if err := svc.prepareForCreate(context.Background(), q); err != nil {
		t.Fatalf("expected prepare to succeed, got %v", err)
	}
	if q.Status != models.QuestionStatusPublished {
		t.Fatalf("expected empty status to default to published, got %s", q.Status)
	}
	if q.Tags == nil {
		t.Fatalf("expected tags to default to empty slice, got nil")
	}
	if q.ID.IsZero() {
		t.Fatalf("expected prepare to assign question id")
	}
	if q.CreatedAt.IsZero() || q.UpdatedAt.IsZero() {
		t.Fatalf("expected prepare to assign timestamps")
	}
}
