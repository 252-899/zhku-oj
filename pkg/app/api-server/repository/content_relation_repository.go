/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程内容关系仓储骨架，先承接课程题库挂载的最小查询入口
*/

package repository

import (
	"context"
	"fmt"
	"time"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	courseContentRelationCollection = "course_content_relations"
	taskContentRelationCollection   = "task_content_relations"
)

// ContentRelationRepository 内容关系仓储。
// 当前阶段只先落课程内容关系查询骨架；任务关系集合常量保留给后续任务挂载阶段复用。
type ContentRelationRepository struct {
	*BaseRepository
}

// NewContentRelationRepository 创建统一内容关系仓储实例。
func NewContentRelationRepository() *ContentRelationRepository {
	return &ContentRelationRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// buildCourseContentFilter 构建课程内容过滤条件。
// 这里保持最小过滤范围，避免在仓储层提前混入发布态等更高层业务规则。
func buildCourseContentFilter(courseID primitive.ObjectID, contentType models.ContentType) bson.M {
	return bson.M{
		"course_id":    courseID,
		"content_type": contentType,
	}
}

// CreateCourseRelations 批量创建课程内容挂载关系。
func (r *ContentRelationRepository) CreateCourseRelations(ctx context.Context, relations []*models.CourseContentRelation) error {
	if len(relations) == 0 {
		return nil
	}
	documents := make([]interface{}, 0, len(relations))
	now := time.Now()
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		if relation.ID.IsZero() {
			relation.ID = primitive.NewObjectID()
		}
		if relation.CreatedAt.IsZero() {
			relation.CreatedAt = now
		}
		relation.UpdatedAt = now
		documents = append(documents, relation)
	}
	if len(documents) == 0 {
		return nil
	}
	_, err := r.InsertMany(ctx, courseContentRelationCollection, documents)
	return err
}

// ListCourseRelations 查询课程下指定内容类型的挂载关系。
func (r *ContentRelationRepository) ListCourseRelations(ctx context.Context, courseID primitive.ObjectID, contentType models.ContentType) ([]*models.CourseContentRelation, error) {
	cursor, err := r.Find(
		ctx,
		courseContentRelationCollection,
		buildCourseContentFilter(courseID, contentType),
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]*models.CourseContentRelation, 0)
	for cursor.Next(ctx) {
		var item models.CourseContentRelation
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("解码课程内容关系失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("遍历课程内容关系失败: %w", err)
	}
	return items, nil
}

// DeleteCourseRelationsByContent 删除指定题目的全部课程挂载关系。
func (r *ContentRelationRepository) DeleteCourseRelationsByContent(
	ctx context.Context,
	contentType models.ContentType,
	contentID primitive.ObjectID,
) error {
	_, err := r.DeleteMany(ctx, courseContentRelationCollection, bson.M{
		"content_type": contentType,
		"content_id":   contentID,
	})
	return err
}

// ExistsCourseRelation 判断题目是否挂载在指定课程下。
func (r *ContentRelationRepository) ExistsCourseRelation(
	ctx context.Context,
	courseID primitive.ObjectID,
	contentType models.ContentType,
	contentID primitive.ObjectID,
) (bool, error) {
	total, err := r.CountDocuments(ctx, courseContentRelationCollection, bson.M{
		"course_id":    courseID,
		"content_type": contentType,
		"content_id":   contentID,
	})
	if err != nil {
		return false, err
	}
	return total > 0, nil
}

// ListRelationsByContentIDs 按内容类型和内容 ID 集合查询全部课程挂载关系。
// 这个查询结果会被编程题迁移后的兼容层复用，用来把关系表内容回填成旧接口仍在暴露的 course_ids 字段。
func (r *ContentRelationRepository) ListRelationsByContentIDs(
	ctx context.Context,
	contentType models.ContentType,
	contentIDs []primitive.ObjectID,
) ([]*models.CourseContentRelation, error) {
	if len(contentIDs) == 0 {
		return []*models.CourseContentRelation{}, nil
	}
	cursor, err := r.Find(
		ctx,
		courseContentRelationCollection,
		bson.M{
			"content_type": contentType,
			"content_id":   bson.M{"$in": contentIDs},
		},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]*models.CourseContentRelation, 0)
	for cursor.Next(ctx) {
		var item models.CourseContentRelation
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("解码课程内容关系失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("遍历课程内容关系失败: %w", err)
	}
	return items, nil
}
