/*
@Author: Codex
@Date: 2026/05/08
@Description: 填空题仓储骨架，负责承接后续填空题数据访问实现
*/

package repository

import (
	"context"
	"fmt"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	blankQuestionCollection = "blank_questions"
)

// BlankQuestionRepository 填空题仓储。
// 先提供最小依赖注入入口，避免服务注册前就耦合具体查询实现。
type BlankQuestionRepository struct {
	*BaseRepository
}

// NewBlankQuestionRepository 创建填空题仓储实例。
func NewBlankQuestionRepository() *BlankQuestionRepository {
	return &BlankQuestionRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// Create 创建填空题。
func (r *BlankQuestionRepository) Create(ctx context.Context, question *models.BlankQuestion) error {
	if question == nil {
		return fmt.Errorf("填空题不能为空")
	}
	_, err := r.InsertOne(ctx, blankQuestionCollection, question)
	return err
}

// GetByID 按 ID 查询填空题。
func (r *BlankQuestionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.BlankQuestion, error) {
	var question models.BlankQuestion
	if err := r.FindOne(ctx, blankQuestionCollection, bson.M{"_id": id}, &question); err != nil {
		return nil, err
	}
	return &question, nil
}

// ListByIDs 批量查询填空题。
func (r *BlankQuestionRepository) ListByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*models.BlankQuestion, error) {
	if len(ids) == 0 {
		return []*models.BlankQuestion{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.Find(ctx, blankQuestionCollection, bson.M{"_id": bson.M{"$in": ids}}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]*models.BlankQuestion, 0, len(ids))
	for cursor.Next(ctx) {
		var item models.BlankQuestion
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("解码填空题失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("遍历填空题失败: %w", err)
	}
	return items, nil
}

// Delete 删除填空题。
func (r *BlankQuestionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.DeleteOne(ctx, blankQuestionCollection, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("填空题不存在")
	}
	return nil
}

// Replace 全量更新填空题。
func (r *BlankQuestionRepository) Replace(ctx context.Context, question *models.BlankQuestion) error {
	if question == nil {
		return fmt.Errorf("填空题不能为空")
	}
	result, err := r.GetCollection(blankQuestionCollection).ReplaceOne(ctx, bson.M{"_id": question.ID}, question)
	if err != nil {
		return fmt.Errorf("更新填空题失败: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("填空题不存在")
	}
	return nil
}
