/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: service_submit.go
@Description: 提交服务
*/

package service

import (
	"context"
	"time"
	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SubmitService struct {
	repo *repository.SubmitRepository
}

func NewSubmitService(repo *repository.SubmitRepository) *SubmitService {
	return &SubmitService{
		repo: repo,
	}
}

// CreateSubmit 创建提交
func (s *SubmitService) CreateSubmit(ctx context.Context, submit *models.Submit) error {
	// 业务逻辑：设置默认状态和系统字段
	submit.ID = primitive.NewObjectID()
	submit.Status = models.StatusPending
	submit.CreatedAt = time.Now()
	submit.UpdatedAt = time.Now()

	// 数据操作：委托给Repository层
	return s.repo.CreateSubmit(ctx, submit)
}

// GetSubmitByID 根据ID获取提交
func (s *SubmitService) GetSubmitByID(ctx context.Context, id primitive.ObjectID) (*models.Submit, error) {
	// 数据操作：直接委托给Repository层
	return s.repo.GetByID(ctx, id)
}

// GetSubmits 获取提交列表
func (s *SubmitService) GetSubmits(ctx context.Context, page, pageSize int, userID, problemID *primitive.ObjectID, status models.SubmitStatus) ([]*models.SubmitList, int64, error) {
	// 数据操作：委托给Repository层进行分页查询
	submits, total, err := s.repo.GetSubmitsList(ctx, page, pageSize, userID, problemID, status)
	if err != nil {
		return nil, 0, err
	}

	// 业务逻辑：转换为SubmitList格式并填充计算字段
	var submitList []*models.SubmitList
	for _, submit := range submits {
		submitList = append(submitList, &models.SubmitList{
			ID:         submit.ID,
			ProblemID:  submit.ProblemID,
			UserID:     submit.UserID,
			Language:   submit.Language,
			Status:     submit.Status,
			TimeUsed:   s.getTimeUsed(submit.Result),   // 业务逻辑：提取时间消耗
			MemoryUsed: s.getMemoryUsed(submit.Result), // 业务逻辑：提取内存消耗
			CreatedAt:  submit.CreatedAt,
		})
	}

	return submitList, total, nil
}

// UpdateSubmit 更新提交
func (s *SubmitService) UpdateSubmit(ctx context.Context, submit *models.Submit) error {
	// 业务逻辑：设置更新时间
	submit.UpdatedAt = time.Now()

	// 数据操作：委托给Repository层
	return s.repo.UpdateSubmit(ctx, submit)
}

// getTimeUsed 获取时间使用
func (s *SubmitService) getTimeUsed(result *models.JudgeResult) int {
	if result == nil {
		return 0
	}
	return result.TimeUsed
}

// getMemoryUsed 获取内存使用
func (s *SubmitService) getMemoryUsed(result *models.JudgeResult) int {
	if result == nil {
		return 0
	}
	return result.MemoryUsed
}

// GetSubmitsByStatus 根据状态获取提交列表（不分页，用于队列恢复）
func (s *SubmitService) GetSubmitsByStatus(ctx context.Context, status models.SubmitStatus) ([]*models.Submit, error) {
	// 数据操作：直接委托给Repository层，使用大分页获取所有记录
	submits, _, err := s.repo.GetSubmitsByStatus(ctx, status, 1, 1000)
	return submits, err
}

// GetSubmitsByStatusWithPagination 根据状态获取提交列表（分页）
func (s *SubmitService) GetSubmitsByStatusWithPagination(ctx context.Context, status models.SubmitStatus, page, pageSize int) ([]*models.Submit, int64, error) {
	// 数据操作：直接委托给Repository层
	return s.repo.GetSubmitsByStatus(ctx, status, page, pageSize)
}

// UpdateSubmitStatus 更新提交状态
func (s *SubmitService) UpdateSubmitStatus(ctx context.Context, submitID primitive.ObjectID, status models.SubmitStatus) error {
	// 数据操作：直接委托给Repository层
	return s.repo.UpdateSubmitStatus(ctx, submitID, status)
}

// UpdateSubmitResult 更新提交结果
func (s *SubmitService) UpdateSubmitResult(ctx context.Context, submitID primitive.ObjectID, result *models.JudgeResult) error {
	// 数据操作：直接委托给Repository层
	return s.repo.UpdateSubmitResult(ctx, submitID, result)
}

// BatchUpdateStatus 批量更新状态（用于服务重启恢复）
func (s *SubmitService) BatchUpdateStatus(ctx context.Context, fromStatus, toStatus models.SubmitStatus, errorMsg string) error {
	// 数据操作：直接委托给Repository层
	return s.repo.BatchUpdateStatus(ctx, fromStatus, toStatus, errorMsg)
}

// GetSubmitStatus 获取提交状态（轻量级）
func (s *SubmitService) GetSubmitStatus(ctx context.Context, submitID primitive.ObjectID) (*models.SubmitStatusResponse, error) {
	// 数据操作：获取基本提交信息
	submit, err := s.repo.GetByID(ctx, submitID)
	if err != nil {
		return nil, err
	}

	// 业务逻辑：构建轻量级状态响应
	response := &models.SubmitStatusResponse{
		ID:        submit.ID,
		Status:    submit.Status,
		Message:   s.getStatusMessage(submit.Status),
		UpdatedAt: submit.UpdatedAt,
	}

	// 业务逻辑：根据状态添加进度信息
	if submit.Status == models.StatusRunning && submit.Result != nil {
		response.Progress = s.calculateProgress(submit.Result)
	}

	// 业务逻辑：完成状态时添加基本结果信息
	if s.isCompletedStatus(submit.Status) && submit.Result != nil {
		timeUsed := submit.Result.TimeUsed
		memoryUsed := submit.Result.MemoryUsed
		response.TimeUsed = &timeUsed
		response.MemoryUsed = &memoryUsed
	}

	return response, nil
}

// getStatusMessage 获取状态描述信息
func (s *SubmitService) getStatusMessage(status models.SubmitStatus) string {
	switch status {
	case models.StatusPending:
		return "等待判题中..."
	case models.StatusRunning:
		return "正在判题..."
	case models.StatusAccepted:
		return "通过"
	case models.StatusWrongAnswer:
		return "答案错误"
	case models.StatusTimeLimit:
		return "时间超限"
	case models.StatusMemoryLimit:
		return "内存超限"
	case models.StatusRuntimeError:
		return "运行时错误"
	case models.StatusCompileError:
		return "编译错误"
	case models.StatusSystemError:
		return "系统错误"
	default:
		return "未知状态"
	}
}

// calculateProgress 计算判题进度
func (s *SubmitService) calculateProgress(result *models.JudgeResult) *models.JudgeProgress {
	if result == nil || result.TestResults == nil {
		return nil
	}

	totalCases := len(result.TestResults)
	if totalCases == 0 {
		return nil
	}

	// 计算已完成的测试用例数
	completedCases := 0
	for _, testResult := range result.TestResults {
		if testResult.Status != "" {
			completedCases++
		}
	}

	percentage := (completedCases * 100) / totalCases

	return &models.JudgeProgress{
		CurrentTestCase: completedCases + 1,
		TotalTestCases:  totalCases,
		Percentage:      percentage,
	}
}

// isCompletedStatus 判断是否为完成状态
func (s *SubmitService) isCompletedStatus(status models.SubmitStatus) bool {
	switch status {
	case models.StatusAccepted, models.StatusWrongAnswer, models.StatusTimeLimit,
		models.StatusMemoryLimit, models.StatusRuntimeError, models.StatusCompileError,
		models.StatusSystemError:
		return true
	default:
		return false
	}
}
