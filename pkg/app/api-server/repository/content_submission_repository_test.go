/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一内容提交仓储的过滤器测试
*/

package repository

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"zk-code-arena-server/pkg/models"
)

func TestBuildStandaloneSubmissionFilterExcludesTaskContext(t *testing.T) {
	userID := primitive.NewObjectID()
	contentID := primitive.NewObjectID()

	filter := buildStandaloneSubmissionFilter(userID, models.ContentTypeBlank, contentID)

	if filter["user_id"] != userID {
		t.Fatalf("expected user_id %v, got %#v", userID, filter["user_id"])
	}
	if filter["content_type"] != models.ContentTypeBlank {
		t.Fatalf("expected content_type %q, got %#v", models.ContentTypeBlank, filter["content_type"])
	}
	if filter["content_id"] != contentID {
		t.Fatalf("expected content_id %v, got %#v", contentID, filter["content_id"])
	}
	if filter["submission_mode"] != models.SubmissionModeStandalone {
		t.Fatalf("expected standalone submission mode, got %#v", filter["submission_mode"])
	}
	taskIDFilter, ok := filter["task_id"].(bson.M)
	if !ok {
		t.Fatalf("expected task_id existence filter, got %#v", filter["task_id"])
	}
	if taskIDFilter["$exists"] != false {
		t.Fatalf("expected task_id $exists=false, got %#v", taskIDFilter)
	}
	taskRelationFilter, ok := filter["task_content_relation_id"].(bson.M)
	if !ok {
		t.Fatalf("expected task_content_relation_id existence filter, got %#v", filter["task_content_relation_id"])
	}
	if taskRelationFilter["$exists"] != false {
		t.Fatalf("expected task_content_relation_id $exists=false, got %#v", taskRelationFilter)
	}
}
