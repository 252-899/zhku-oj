/*
@Author: omenkk7
@Date: 2025/10/25
@Description: TestCase数据访问层 - 测试用例相关的数据操作
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

// TestCaseRepository TestCase数据访问层
type TestCaseRepository struct {
	*BaseRepository
}

// BatchImportResult 批量导入结果
type BatchImportResult struct {
	SuccessCount int      `json:"success_count"`
	FailCount    int      `json:"fail_count"`
	Errors       []string `json:"errors,omitempty"`
}

// NewTestCaseRepository 创建TestCaseRepository实例
func NewTestCaseRepository() *TestCaseRepository {
	return &TestCaseRepository{
		BaseRepository: NewBaseRepository(),
	}
}

// CreateTestCase 创建测试用例
func (r *TestCaseRepository) CreateTestCase(ctx context.Context, testCase *models.TestCase) error {
	// 设置系统字段
	now := time.Now()
	testCase.ID = primitive.NewObjectID()
	testCase.CreatedAt = now
	testCase.UpdatedAt = now

	// 验证测试用例
	if err := r.validateTestCase(testCase); err != nil {
		utils.Logger.Errorf("CreateTestCase: 验证失败, problemID=%s, error=%v", testCase.ProblemID.Hex(), err)
		return err
	}

	_, err := r.InsertOne(ctx, "test_cases", testCase)
	if err != nil {
		utils.Logger.Errorf("CreateTestCase: 创建测试用例失败, problemID=%s, error=%v", testCase.ProblemID.Hex(), err)
		return fmt.Errorf("创建测试用例失败: %w", err)
	}

	utils.Logger.Infof("CreateTestCase: 测试用例创建成功, id=%s, problemID=%s", testCase.ID.Hex(), testCase.ProblemID.Hex())
	return nil
}

// GetByID 根据ID查询测试用例
func (r *TestCaseRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.TestCase, error) {
	var testCase models.TestCase
	err := r.FindOne(ctx, "test_cases", bson.M{"_id": id}, &testCase)
	if err != nil {
		return nil, err
	}
	return &testCase, nil
}

// GetByProblemID 根据题目ID查询所有测试用例 - 使用泛型分页查询
func (r *TestCaseRepository) GetByProblemID(ctx context.Context, problemID primitive.ObjectID) ([]*models.TestCase, error) {
	// 构建过滤器和排序
	filter := bson.M{"problem_id": problemID}
	sort := bson.D{{"order", 1}, {"created_at", 1}}

	// 构建查询选项（pageSize=0表示不限制，获取所有结果）
	opts := r.BuildFindOptionsWithPagination(1, 0, sort, nil)

	// 使用泛型分页查询
	testCases, _, err := FindWithPaginationTyped[models.TestCase](ctx, r.BaseRepository, "test_cases", filter, opts)
	if err != nil {
		return nil, fmt.Errorf("查询测试用例失败: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.TestCase, len(testCases))
	for i := range testCases {
		result[i] = &testCases[i]
	}

	return result, nil
}

// GetSampleTestCases 获取示例测试用例 - 使用泛型分页查询
func (r *TestCaseRepository) GetSampleTestCases(ctx context.Context, problemID primitive.ObjectID) ([]*models.TestCase, error) {
	// 构建过滤器和排序
	filter := bson.M{
		"problem_id": problemID,
		"is_sample":  true,
	}
	sort := bson.D{{"order", 1}, {"created_at", 1}}

	// 构建查询选项（pageSize=0表示不限制，获取所有结果）
	opts := r.BuildFindOptionsWithPagination(1, 0, sort, nil)

	// 使用泛型分页查询
	testCases, _, err := FindWithPaginationTyped[models.TestCase](ctx, r.BaseRepository, "test_cases", filter, opts)
	if err != nil {
		return nil, fmt.Errorf("查询示例测试用例失败: %w", err)
	}

	// 转换为指针切片
	result := make([]*models.TestCase, len(testCases))
	for i := range testCases {
		result[i] = &testCases[i]
	}

	return result, nil
}

// UpdateTestCase 更新测试用例（使用通用更新函数）
func (r *TestCaseRepository) UpdateTestCase(ctx context.Context, testCaseID primitive.ObjectID, req interface{}) error {
	// 使用BaseRepository的通用更新方法
	result, err := r.UpdateOne(ctx, "test_cases", bson.M{"_id": testCaseID}, req)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		utils.Logger.Warnf("UpdateTestCase: 测试用例不存在, testCaseID=%s", testCaseID.Hex())
		return fmt.Errorf("测试用例不存在")
	}

	utils.Logger.Infof("UpdateTestCase: 测试用例更新成功, testCaseID=%s", testCaseID.Hex())
	return nil
}

// UpdateTestCaseComplete 完整更新测试用例
func (r *TestCaseRepository) UpdateTestCaseComplete(ctx context.Context, testCase *models.TestCase) error {
	// 验证测试用例
	if err := r.validateTestCase(testCase); err != nil {
		return err
	}

	result, err := r.UpdateOne(ctx, "test_cases", bson.M{"_id": testCase.ID}, testCase)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("测试用例不存在")
	}

	utils.Logger.Infof("UpdateTestCaseComplete: 测试用例更新成功, id=%s", testCase.ID.Hex())
	return nil
}

// DeleteTestCase 删除测试用例
func (r *TestCaseRepository) DeleteTestCase(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.DeleteOne(ctx, "test_cases", bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("测试用例不存在")
	}

	utils.Logger.Infof("DeleteTestCase: 测试用例删除成功, id=%s", id.Hex())
	return nil
}

// CountByProblemID 统计题目的测试用例数量
func (r *TestCaseRepository) CountByProblemID(ctx context.Context, problemID primitive.ObjectID) (int64, error) {
	filter := bson.M{"problem_id": problemID}
	return r.CountDocuments(ctx, "test_cases", filter)
}

// BatchCreateTestCases 批量创建测试用例
func (r *TestCaseRepository) BatchCreateTestCases(ctx context.Context, testCases []*models.TestCase) (*BatchImportResult, error) {
	if len(testCases) == 0 {
		return &BatchImportResult{}, nil
	}

	result := &BatchImportResult{
		Errors: make([]string, 0),
	}

	// 准备批量插入的数据
	documents := make([]interface{}, 0, len(testCases))

	for i, testCase := range testCases {
		// 设置系统字段
		now := time.Now()
		testCase.ID = primitive.NewObjectID()
		testCase.CreatedAt = now
		testCase.UpdatedAt = now

		// 验证测试用例
		if err := r.validateTestCase(testCase); err != nil {
			result.FailCount++
			result.Errors = append(result.Errors, fmt.Sprintf("第%d个测试用例验证失败: %v", i+1, err))
			continue
		}

		documents = append(documents, testCase)
	}

	// 执行批量插入
	if len(documents) > 0 {
		insertResult, err := r.InsertMany(ctx, "test_cases", documents)
		if err != nil {
			utils.Logger.Errorf("BatchCreateTestCases: 批量插入失败, error=%v", err)
			return nil, fmt.Errorf("批量插入失败: %w", err)
		}

		result.SuccessCount = len(insertResult.InsertedIDs)
		utils.Logger.Infof("BatchCreateTestCases: 批量创建成功, success=%d, fail=%d",
			result.SuccessCount, result.FailCount)
	}

	return result, nil
}

// DeleteByProblemID 删除题目的所有测试用例 - 使用 BaseRepository 的 DeleteMany 方法
func (r *TestCaseRepository) DeleteByProblemID(ctx context.Context, problemID primitive.ObjectID) error {
	filter := bson.M{"problem_id": problemID}

	// 使用 BaseRepository 的 DeleteMany 方法
	_, err := r.DeleteMany(ctx, "test_cases", filter)
	if err != nil {
		return fmt.Errorf("删除测试用例失败: %w", err)
	}

	return nil
}

// UpdateTestCaseOrder 更新测试用例顺序
func (r *TestCaseRepository) UpdateTestCaseOrder(ctx context.Context, testCaseID primitive.ObjectID, order int) error {
	updateData := struct {
		Order int `bson:"order"`
	}{
		Order: order,
	}

	result, err := r.UpdateOne(ctx, "test_cases", bson.M{"_id": testCaseID}, updateData)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("测试用例不存在")
	}

	return nil
}

// GetTestCaseStats 获取题目测试用例统计
func (r *TestCaseRepository) GetTestCaseStats(ctx context.Context, problemID primitive.ObjectID) (map[string]int64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{"problem_id": problemID},
		},
		{
			"$group": bson.M{
				"_id":   "$is_sample",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.Aggregate(ctx, "test_cases", pipeline)
	if err != nil {
		return nil, fmt.Errorf("查询测试用例统计失败: %w", err)
	}
	defer cursor.Close(ctx)

	stats := map[string]int64{
		"total":  0,
		"sample": 0,
		"hidden": 0,
	}

	for cursor.Next(ctx) {
		var result struct {
			ID    bool  `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			continue
		}

		stats["total"] += result.Count
		if result.ID {
			stats["sample"] = result.Count
		} else {
			stats["hidden"] = result.Count
		}
	}

	return stats, nil
}

// validateTestCase 验证测试用例
func (r *TestCaseRepository) validateTestCase(testCase *models.TestCase) error {
	if testCase.ProblemID.IsZero() {
		return fmt.Errorf("题目ID不能为空")
	}

	if testCase.Input == "" && testCase.Output == "" {
		return fmt.Errorf("输入和输出不能同时为空")
	}

	return nil
}
