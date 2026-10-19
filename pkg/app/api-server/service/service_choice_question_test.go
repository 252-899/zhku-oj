/*
@Author: Codex
@Date: 2026/05/08
@Description: 选择题服务最小骨架测试，约束创建前预处理的默认行为
*/

package service

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestChoiceQuestionDefaultsDraftToPrivateCourseMode(t *testing.T) {
	svc := &ChoiceQuestionService{}
	q := &models.ChoiceQuestion{
		Title:       "单选题",
		Stem:        "2+2=?",
		Status:      models.QuestionStatusDraft,
		PublishMode: models.QuestionPublishModeCourseOnly,
		ChoiceMode:  models.ChoiceModeSingle,
		Options: []models.ChoiceOption{
			{Key: "A", Content: "3"},
			{Key: "B", Content: "4"},
		},
		CorrectAnswers: []string{"B"},
		CreatedBy:      primitive.NewObjectID(),
	}
	if err := svc.prepareForCreate(context.Background(), q); err != nil {
		t.Fatalf("expected prepare to succeed, got %v", err)
	}
	if q.Status != models.QuestionStatusDraft {
		t.Fatalf("expected draft status to remain draft, got %s", q.Status)
	}
	if q.PublishMode != models.QuestionPublishModeCourseOnly {
		t.Fatalf("expected course-only publish mode to remain unchanged, got %s", q.PublishMode)
	}
}
