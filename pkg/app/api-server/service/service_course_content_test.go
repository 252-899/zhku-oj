/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合内容服务测试，约束最小聚合 helper 行为
*/

package service

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestMergeCourseContentItemsSortsByCreatedTimeDescending(t *testing.T) {
	now := time.Now()
	older := &models.CourseContentListItem{
		ContentID: primitive.NewObjectID(),
		CreatedAt: now.Add(-2 * time.Hour),
	}
	newer := &models.CourseContentListItem{
		ContentID: primitive.NewObjectID(),
		CreatedAt: now.Add(-1 * time.Hour),
	}
	newest := &models.CourseContentListItem{
		ContentID: primitive.NewObjectID(),
		CreatedAt: now,
	}

	items := mergeCourseContentItems(
		[]*models.CourseContentListItem{older},
		[]*models.CourseContentListItem{newest},
		[]*models.CourseContentListItem{newer},
	)
	if items == nil {
		t.Fatalf("expected empty slice or merged slice, got nil")
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 merged items, got %d", len(items))
	}
	if items[0].ContentID != newest.ContentID || items[1].ContentID != newer.ContentID || items[2].ContentID != older.ContentID {
		t.Fatalf("expected items sorted by created_at descending")
	}
}
