/*
@Author: Codex
@Date: 2026/05/08
@Description: 选择题仓储骨架，负责承接后续选择题数据访问实现
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
	choiceQuestionCollection = "choice_questions"
)

// ChoiceQuestionRepository 选择题仓储。
// 这里只先落容器可注册的骨架，后续业务能力由服务层任务按需补齐。
type ChoiceQuestionRepository struct {
	*BaseRepository
}

// NewChoiceQuestionRepository 创建选择题仓储实例。
func NewChoiceQuestionRepository() *ChoiceQuestionRepository {
	return &ChoiceQuestionRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// Create 创建选择题。
func (r *ChoiceQuestionRepository) Create(ctx context.Context, question *models.ChoiceQuestion) error {
	if question == nil {
		return fmt.Errorf("选择题不能为空")
	}
	_, err := r.InsertOne(ctx, choiceQuestionCollection, question)
	return err
}

// GetByID 按 ID 查询选择题。
func (r *ChoiceQuestionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.ChoiceQuestion, error) {
	var question models.ChoiceQuestion
	if err := r.FindOne(ctx, choiceQuestionCollection, bson.M{"_id": id}, &question); err != nil {
		return nil, err
	}
	return &question, nil
}

// ListByIDs 批量查询选择题。
func (r *ChoiceQuestionRepository) ListByIDs(ctx context.Context, ids []primitive.ObjectID) ([]*models.ChoiceQuestion, error) {
	if len(ids) == 0 {
		return []*models.ChoiceQuestion{}, nil
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.Find(ctx, choiceQuestionCollection, bson.M{"_id": bson.M{"$in": ids}}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]*models.ChoiceQuestion, 0, len(ids))
	for cursor.Next(ctx) {
		var item models.ChoiceQuestion
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("解码选择题失败: %w", err)
		}
		items = append(items, &item)
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("遍历选择题失败: %w", err)
	}
	return items, nil
}

// Delete 删除选择题。
func (r *ChoiceQuestionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.DeleteOne(ctx, choiceQuestionCollection, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("选择题不存在")
	}
	return nil
}

// Replace 全量更新选择题。
func (r *ChoiceQuestionRepository) Replace(ctx context.Context, question *models.ChoiceQuestion) error {
	if question == nil {
		return fmt.Errorf("选择题不能为空")
	}
	result, err := r.GetCollection(choiceQuestionCollection).ReplaceOne(ctx, bson.M{"_id": question.ID}, question)
	if err != nil {
		return fmt.Errorf("更新选择题失败: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("选择题不存在")
	}
	return nil
}
