/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: service_testcase.go
@Description: 测试用例服务
*/

package service

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"
)

// TestCaseService 测试用例服务
type TestCaseService struct {
	repo *repository.TestCaseRepository
}

// NewTestCaseService 创建测试用例服务实例
func NewTestCaseService(repo *repository.TestCaseRepository) *TestCaseService {
	return &TestCaseService{
		repo: repo,
	}
}

// GetTestCasesByProblemID 根据题目 ID 获取测试用例列表
func (s *TestCaseService) GetTestCasesByProblemID(ctx context.Context, problemID primitive.ObjectID) ([]*models.TestCase, error) {
	return s.repo.GetByProblemID(ctx, problemID)
}

// CreateTestCase 创建测试用例
func (s *TestCaseService) CreateTestCase(ctx context.Context, testCase *models.TestCase) error {
	return s.repo.CreateTestCase(ctx, testCase)
}

// UpdateTestCase 更新测试用例
func (s *TestCaseService) UpdateTestCase(ctx context.Context, testCase *models.TestCase) error {
	return s.repo.UpdateTestCaseComplete(ctx, testCase)
}

// DeleteTestCase 删除测试用例
func (s *TestCaseService) DeleteTestCase(ctx context.Context, testCaseID primitive.ObjectID) error {
	return s.repo.DeleteTestCase(ctx, testCaseID)
}

// DeleteByProblemID 删除指定题目的全部测试用例。
func (s *TestCaseService) DeleteByProblemID(ctx context.Context, problemID primitive.ObjectID) error {
	// 保存AI生成题时若中途失败，需要清理已经写入的测试用例，避免半成品残留。
	return s.repo.DeleteByProblemID(ctx, problemID)
}

// GetTestCaseByID 根据 ID 获取测试用例
func (s *TestCaseService) GetTestCaseByID(ctx context.Context, testCaseID primitive.ObjectID) (*models.TestCase, error) {
	return s.repo.GetByID(ctx, testCaseID)
}

// GetTestCaseCount 获取题目测试用例总数
func (s *TestCaseService) GetTestCaseCount(ctx context.Context, problemID primitive.ObjectID) (int64, error) {
	return s.repo.CountByProblemID(ctx, problemID)
}

// BatchImportTestCases 批量导入测试用例
func (s *TestCaseService) BatchImportTestCases(ctx context.Context, problemID primitive.ObjectID, testCases []*models.TestCase) (*repository.BatchImportResult, error) {
	return s.repo.BatchCreateTestCases(ctx, testCases)
}

// GetSampleTestCases 获取示例测试用例
func (s *TestCaseService) GetSampleTestCases(ctx context.Context, problemID primitive.ObjectID) ([]*models.TestCase, error) {
	return s.repo.GetSampleTestCases(ctx, problemID)
}
