/*
@Author: Codex
@Date: 2026/4/19
@Name: ai_generated_problem_draft_repository.go
@Description: AI 生成题目草稿数据访问层，负责草稿持久化和保存状态回写
*/

package repository

import (
	"context"
	"fmt"
	"time"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// aiGeneratedProblemDraftCollection AI 生成题目草稿集合名。
const aiGeneratedProblemDraftCollection = "ai_generated_problem_drafts"

// AIGeneratedProblemDraftRepository AI 生成题目草稿数据访问层。
type AIGeneratedProblemDraftRepository struct {
	*BaseRepository
}

// NewAIGeneratedProblemDraftRepository 创建 AI 生成题目草稿 Repository。
func NewAIGeneratedProblemDraftRepository() *AIGeneratedProblemDraftRepository {
	return &AIGeneratedProblemDraftRepository{BaseRepository: NewBaseRepository()}
}

// Create 创建 AI 生成题目草稿。
func (r *AIGeneratedProblemDraftRepository) Create(ctx context.Context, draft *models.AIGeneratedProblemDraft) error {
	if draft == nil {
		return fmt.Errorf("草稿不能为空")
	}
	now := time.Now()
	if draft.ID.IsZero() {
		draft.ID = primitive.NewObjectID()
	}
	if draft.Status == "" {
		draft.Status = models.AIGeneratedProblemDraftPending
	}
	if draft.CreatedAt.IsZero() {
		draft.CreatedAt = now
	}
	draft.UpdatedAt = now
	if draft.ExpiresAt.IsZero() {
		draft.ExpiresAt = now.Add(24 * time.Hour)
	}
	for i := range draft.Problems {
		if draft.Problems[i].Status == "" {
			draft.Problems[i].Status = models.AIGeneratedDraftProblemGenerated
		}
	}

	if _, err := r.InsertOne(ctx, aiGeneratedProblemDraftCollection, draft); err != nil {
		utils.Logger.Errorf("AIGeneratedProblemDraftRepository.Create: 创建草稿失败, error=%v", err)
		return fmt.Errorf("创建AI生成题目草稿失败: %w", err)
	}
	return nil
}

// GetByID 根据草稿 ID 查询 AI 生成题目草稿。
func (r *AIGeneratedProblemDraftRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.AIGeneratedProblemDraft, error) {
	var draft models.AIGeneratedProblemDraft
	if err := r.FindOne(ctx, aiGeneratedProblemDraftCollection, bson.M{"_id": id}, &draft); err != nil {
		return nil, err
	}
	return &draft, nil
}

// UpdateAfterSave 保存正式题库后回写草稿内每道题的处理状态。
func (r *AIGeneratedProblemDraftRepository) UpdateAfterSave(ctx context.Context, draft *models.AIGeneratedProblemDraft, results []models.AIGeneratedProblemSaveResult) error {
	if draft == nil {
		return fmt.Errorf("草稿不能为空")
	}
	applyAIGeneratedDraftSaveResult(draft, results)
	draft.UpdatedAt = time.Now()
	if draft.SavedAt == nil {
		savedAt := draft.UpdatedAt
		draft.SavedAt = &savedAt
	}

	update := bson.M{
		"$set": bson.M{
			"status":     draft.Status,
			"problems":   draft.Problems,
			"updated_at": draft.UpdatedAt,
			"saved_at":   draft.SavedAt,
		},
	}
	// BaseRepository.UpdateOne 会把普通结构体自动包装为 $set，这里已经是原生 Mongo 更新文档，因此直接调用集合。
	result, err := r.GetCollection(aiGeneratedProblemDraftCollection).UpdateOne(ctx, bson.M{"_id": draft.ID}, update)
	if err != nil {
		return fmt.Errorf("更新AI生成题目草稿失败: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("AI生成题目草稿不存在")
	}
	return nil
}

// applyAIGeneratedDraftSaveResult 根据保存结果更新草稿状态。
func applyAIGeneratedDraftSaveResult(draft *models.AIGeneratedProblemDraft, results []models.AIGeneratedProblemSaveResult) {
	if draft == nil {
		return
	}
	resultMap := make(map[string]models.AIGeneratedProblemSaveResult, len(results))
	failedCount := 0
	for _, result := range results {
		resultMap[result.GeneratedProblemID] = result
		if !result.Success {
			failedCount++
		}
	}

	for i := range draft.Problems {
		problem := &draft.Problems[i]
		result, selected := resultMap[problem.GeneratedProblemID]
		switch {
		case !selected:
			// 未出现在保存请求中的题目视为教师未选择，只在草稿里保留痕迹等待 TTL 清理。
			problem.Status = models.AIGeneratedDraftProblemDiscarded
			problem.SaveError = ""
			problem.SavedProblemID = nil
		case result.Success:
			problem.Status = models.AIGeneratedDraftProblemSaved
			problem.SaveError = ""
			if objectID, err := primitive.ObjectIDFromHex(result.ProblemID); err == nil {
				problem.SavedProblemID = &objectID
			}
		default:
			problem.Status = models.AIGeneratedDraftProblemFailed
			problem.SaveError = result.Error
			problem.SavedProblemID = nil
		}
	}

	if failedCount > 0 {
		draft.Status = models.AIGeneratedProblemDraftPartiallySaved
		return
	}
	draft.Status = models.AIGeneratedProblemDraftSaved
}
