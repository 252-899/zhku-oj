/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一课程内容关系仓储的过滤器测试
*/

package repository

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"zk-code-arena-server/pkg/models"
)

func TestBuildCourseContentFilterUsesCourseAndType(t *testing.T) {
	courseID := primitive.NewObjectID()

	filter := buildCourseContentFilter(courseID, models.ContentTypeChoice)

	if len(filter) != 2 {
		t.Fatalf("expected minimal two-field filter, got %#v", filter)
	}
	if filter["course_id"] != courseID {
		t.Fatalf("expected course_id %v, got %#v", courseID, filter["course_id"])
	}
	if filter["content_type"] != models.ContentTypeChoice {
		t.Fatalf("expected content_type %q, got %#v", models.ContentTypeChoice, filter["content_type"])
	}
}
