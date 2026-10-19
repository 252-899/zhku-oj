/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一内容提交仓储骨架，负责独立作答与任务作答提交记录的数据访问入口
*/

package repository

import (
	"context"
	"fmt"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	contentSubmissionCollection = "content_submissions"
)

// ContentSubmissionRepository 统一内容提交仓储。
// 独立作答与任务作答共用集合，仓储层通过最小 helper 隔离上下文过滤差异。
type ContentSubmissionRepository struct {
	*BaseRepository
}

// NewContentSubmissionRepository 创建统一内容提交仓储实例。
func NewContentSubmissionRepository() *ContentSubmissionRepository {
	return &ContentSubmissionRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// buildStandaloneSubmissionFilter 构建独立作答过滤条件。
// 独立作答必须同时排除任务 ID 和任务内容关系 ID，避免任务上下文脏数据混入题库自测记录。
func buildStandaloneSubmissionFilter(userID primitive.ObjectID, contentType models.ContentType, contentID primitive.ObjectID) bson.M {
	return bson.M{
		"user_id":                  userID,
		"content_type":             contentType,
		"content_id":               contentID,
		"submission_mode":          models.SubmissionModeStandalone,
		"task_id":                  bson.M{"$exists": false},
		"task_content_relation_id": bson.M{"$exists": false},
	}
}

// Create 创建统一内容提交记录。
func (r *ContentSubmissionRepository) Create(ctx context.Context, submission *models.ContentSubmission) error {
	if submission == nil {
		return fmt.Errorf("提交记录不能为空")
	}
	_, err := r.InsertOne(ctx, contentSubmissionCollection, submission)
	return err
}
