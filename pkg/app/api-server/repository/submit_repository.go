/*
@Author: omenkk7
@Date: 2025/10/25
@Description: Submit数据访问层 - 提交相关的数据操作
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

// SubmitRepository Submit数据访问层
type SubmitRepository struct {
	*BaseRepository
}

// NewSubmitRepository 创建SubmitRepository实例
func NewSubmitRepository() *SubmitRepository {
	return &SubmitRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// CreateSubmit 创建提交记录
func (r *SubmitRepository) CreateSubmit(ctx context.Context, submit *models.Submit) error {
	// 设置系统字段
	submit.ID = primitive.NewObjectID()
	submit.CreatedAt = time.Now()
	submit.UpdatedAt = time.Now()

	// 设置默认状态
	if submit.Status == "" {
		submit.Status = models.StatusPending
	}

	_, err := r.InsertOne(ctx, "submits", submit)
	if err != nil {
		utils.Logger.Errorf("CreateSubmit: 创建提交失败, userID=%s, problemID=%s, error=%v",
			submit.UserID.Hex(), submit.ProblemID.Hex(), err)
		return fmt.Errorf("创建提交失败: %w", err)
	}

	utils.Logger.Infof("CreateSubmit: 提交创建成功, id=%s, userID=%s, problemID=%s",
		submit.ID.Hex(), submit.UserID.Hex(), submit.ProblemID.Hex())
	return nil
}

// GetByID 根据ID查询提交记录
func (r *SubmitRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Submit, error) {
	var submit models.Submit
	err := r.FindOne(ctx, "submits", bson.M{"_id": id}, &submit)
	if err != nil {
		return nil, err
	}
	return &submit, nil
}

// UpdateSubmit 更新提交记录（完整更新）
func (r *SubmitRepository) UpdateSubmit(ctx context.Context, submit *models.Submit) error {
	submit.UpdatedAt = time.Now()

	result, err := r.UpdateOne(ctx, "submits", bson.M{"_id": submit.ID}, submit)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("提交记录不存在")
	}

	utils.Logger.Infof("UpdateSubmit: 提交更新成功, id=%s", submit.ID.Hex())
	return nil
}

// UpdateSubmitStatus 更新提交状态
func (r *SubmitRepository) UpdateSubmitStatus(ctx context.Context, submitID primitive.ObjectID, status models.SubmitStatus) error {
	updateData := struct {
		Status models.SubmitStatus `bson:"status"`
	}{
		Status: status,
	}

	result, err := r.UpdateOne(ctx, "submits", bson.M{"_id": submitID}, updateData)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("提交记录不存在")
	}

	utils.Logger.Infof("UpdateSubmitStatus: 状态更新成功, id=%s, status=%s", submitID.Hex(), status)
	return nil
}

// UpdateSubmitResult 更新提交结果
func (r *SubmitRepository) UpdateSubmitResult(ctx context.Context, submitID primitive.ObjectID, result *models.JudgeResult) error {
	updateData := struct {
		Status models.SubmitStatus `bson:"status"`
		Result *models.JudgeResult `bson:"result"`
	}{
		Status: result.Status,
		Result: result,
	}

	updateResult, err := r.UpdateOne(ctx, "submits", bson.M{"_id": submitID}, updateData)
	if err != nil {
		return err
	}

	if updateResult.MatchedCount == 0 {
		return fmt.Errorf("提交记录不存在")
	}

	utils.Logger.Infof("UpdateSubmitResult: 结果更新成功, id=%s, status=%s", submitID.Hex(), result.Status)
	return nil
}

// GetSubmits 查询提交列表 - 使用新的泛型查询方法
func (r *SubmitRepository) GetSubmits(
	ctx context.Context,
	page, pageSize int,
	userID, problemID primitive.ObjectID,
) ([]*models.SubmitList, int64, error) {
	// 构建查询条件
	filter := bson.M{}
	if !userID.IsZero() {
		filter["user_id"] = userID
	}
	if !problemID.IsZero() {
		filter["problem_id"] = problemID
	}

	// 构建查询选项
	sort := r.BuildSort("-created_at")
	opts := r.BuildFindOptionsWithPagination(int64(page), int64(pageSize), sort, nil)

	// 使用泛型分页查询
	submits, total, err := FindWithPaginationTyped[models.Submit](ctx, r.BaseRepository, "submits", filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("查询提交列表失败: %w", err)
	}

	// 转换为SubmitList
	submitList := make([]*models.SubmitList, 0, len(submits))
	for _, submit := range submits {
		item := &models.SubmitList{
			ID:        submit.ID,
			ProblemID: submit.ProblemID,
			UserID:    submit.UserID,
			Language:  submit.Language,
			Status:    submit.Status,
			CreatedAt: submit.CreatedAt,
		}

		// 添加时间和内存使用信息
		if submit.Result != nil {
			item.TimeUsed = r.getTimeUsed(submit.Result)
			item.MemoryUsed = r.getMemoryUsed(submit.Result)
		}

		submitList = append(submitList, item)
	}

	return submitList, total, nil
}

// GetSubmitsByStatus 查询指定状态的提交记录 - 使用新的泛型查询方法
func (r *SubmitRepository) GetSubmitsByStatus(
	ctx context.Context,
	status models.SubmitStatus,
	page, pageSize int,
) ([]*models.Submit, int64, error) {
	// 构建查询条件
	filter := bson.M{"status": status}

	// 构建查询选项
	sort := r.BuildSort("-created_at")
	opts := r.BuildFindOptionsWithPagination(int64(page), int64(pageSize), sort, nil)

	// 使用泛型分页查询
	submits, total, err := FindWithPaginationTyped[models.Submit](ctx, r.BaseRepository, "submits", filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("查询提交记录失败: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Submit, len(submits))
	for i := range submits {
		result[i] = &submits[i]
	}

	return result, total, nil
}

// BatchUpdateStatus 批量更新状态 - 使用 BaseRepository 的 UpdateMany 方法
func (r *SubmitRepository) BatchUpdateStatus(ctx context.Context, fromStatus, toStatus models.SubmitStatus, errorMsg string) error {
	filter := bson.M{"status": fromStatus}

	updateData := struct {
		Status models.SubmitStatus `bson:"status"`
		Result *models.JudgeResult `bson:"result,omitempty"`
	}{
		Status: toStatus,
	}

	// 如果有错误信息，设置结果
	if errorMsg != "" {
		updateData.Result = &models.JudgeResult{
			Status:       toStatus,
			RuntimeError: errorMsg,
		}
	}

	// 使用 BaseRepository 的 UpdateMany 方法
	result, err := r.UpdateMany(ctx, "submits", filter, updateData)
	if err != nil {
		return fmt.Errorf("批量更新状态失败: %w", err)
	}

	utils.Logger.Infof("BatchUpdateStatus: 批量更新成功, from=%s, to=%s, updated=%d",
		fromStatus, toStatus, result.ModifiedCount)
	return nil
}

// GetUserProblemStatuses 批量获取用户对一组题目的最新状态
func (r *SubmitRepository) GetUserProblemStatuses(ctx context.Context, userID primitive.ObjectID, problemIDs []primitive.ObjectID) (map[primitive.ObjectID]models.UserProblemStatus, error) {
	if len(problemIDs) == 0 {
		return make(map[primitive.ObjectID]models.UserProblemStatus), nil
	}

	pipeline := []bson.M{
		// 1. 匹配用户和题目
		{
			"$match": bson.M{
				"user_id":    userID,
				"problem_id": bson.M{"$in": problemIDs},
			},
		},
		// 2. 按题目分组，获取最新状态
		{
			"$group": bson.M{
				"_id":           "$problem_id",
				"latest_status": bson.M{"$last": "$status"},
				"created_at":    bson.M{"$max": "$created_at"},
			},
		},
	}

	cursor, err := r.Aggregate(ctx, "submits", pipeline)
	if err != nil {
		return nil, fmt.Errorf("查询用户题目状态失败: %w", err)
	}
	defer cursor.Close(ctx)

	statusMap := make(map[primitive.ObjectID]models.UserProblemStatus)

	for cursor.Next(ctx) {
		var result struct {
			ID           primitive.ObjectID `bson:"_id"`
			LatestStatus string             `bson:"latest_status"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		// 转换状态
		var userStatus models.UserProblemStatus
		if result.LatestStatus == string(models.StatusAccepted) {
			userStatus = models.UserStatusAccepted
		} else {
			userStatus = models.UserStatusAttempted
		}

		statusMap[result.ID] = userStatus
	}

	// 对于没有提交记录的题目，状态为未尝试
	for _, problemID := range problemIDs {
		if _, exists := statusMap[problemID]; !exists {
			statusMap[problemID] = models.UserStatusNotAttempted
		}
	}

	return statusMap, nil
}

// getTimeUsed 获取时间使用情况
func (r *SubmitRepository) getTimeUsed(result *models.JudgeResult) int {
	if result == nil {
		return 0
	}

	maxTime := result.TimeUsed
	for _, testResult := range result.TestResults {
		if testResult.TimeUsed > maxTime {
			maxTime = testResult.TimeUsed
		}
	}
	return maxTime
}

// getMemoryUsed 获取内存使用情况
func (r *SubmitRepository) getMemoryUsed(result *models.JudgeResult) int {
	if result == nil {
		return 0
	}

	maxMemory := result.MemoryUsed
	for _, testResult := range result.TestResults {
		if testResult.MemoryUsed > maxMemory {
			maxMemory = testResult.MemoryUsed
		}
	}
	return maxMemory
}

// GetSubmitsList 分页查询提交列表 - 使用新的泛型查询方法
func (r *SubmitRepository) GetSubmitsList(
	ctx context.Context,
	page, pageSize int,
	userID *primitive.ObjectID,
	problemID *primitive.ObjectID,
	status models.SubmitStatus,
) ([]*models.Submit, int64, error) {
	// 构建查询条件
	filter := r.buildSubmitListFilter(userID, problemID, status)

	// 构建查询选项
	sort := r.BuildSort("-created_at")
	opts := r.BuildFindOptionsWithPagination(int64(page), int64(pageSize), sort, nil)

	// 使用泛型分页查询
	submits, total, err := FindWithPaginationTyped[models.Submit](ctx, r.BaseRepository, "submits", filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("查询提交列表失败: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.Submit, len(submits))
	for i := range submits {
		result[i] = &submits[i]
	}

	return result, total, nil
}

// buildSubmitListFilter 构建提交列表查询过滤器
func (r *SubmitRepository) buildSubmitListFilter(userID *primitive.ObjectID, problemID *primitive.ObjectID, status models.SubmitStatus) bson.M {
	filter := bson.M{}

	if userID != nil && !userID.IsZero() {
		filter["user_id"] = *userID
	}

	if problemID != nil && !problemID.IsZero() {
		filter["problem_id"] = *problemID
	}

	if status != "" {
		filter["status"] = status
	}

	return filter
}
