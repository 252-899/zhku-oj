/*
@Author: omenkk7
@Date: 2025/10/05
@Description: 基于 Go Channel 的消息队列实现
*/

package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChannelQueue 基于 Go Channel 的消息队列实现
type ChannelQueue struct {
	queue     chan *JudgeTask // 任务队列
	queueSize int             // 队列容量
	stopCh    chan struct{}   // 停止信号
	wg        sync.WaitGroup  // 等待组（用于优雅关闭）
	isRunning bool            // 运行状态
	mu        sync.RWMutex    // 读写锁
	submitSvc SubmitService   // 提交服务接口（用于恢复任务）
}

// SubmitService 提交服务接口（用于解耦）
type SubmitService interface {
	GetSubmitsByStatus(ctx context.Context, status models.SubmitStatus) ([]*models.Submit, error)
	UpdateSubmitStatus(ctx context.Context, submitID primitive.ObjectID, status models.SubmitStatus) error
}

// NewChannelQueue 创建基于 Channel 的消息队列
// queueSize: 队列容量
// submitSvc: 提交服务（用于恢复任务）
func NewChannelQueue(queueSize int, submitSvc SubmitService) *ChannelQueue {
	return &ChannelQueue{
		queue:     make(chan *JudgeTask, queueSize),
		queueSize: queueSize,
		stopCh:    make(chan struct{}),
		submitSvc: submitSvc,
	}
}

// Push 推送任务到队列
func (q *ChannelQueue) Push(ctx context.Context, task *JudgeTask) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.isRunning {
		return errors.New("队列未启动")
	}

	select {
	case q.queue <- task:
		utils.GetLogger(ctx).Info(fmt.Sprintf("任务已推送到队列: SubmitID=%s", task.SubmitID.Hex()))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("队列已满，无法推送任务")
	}
}

// Consume 启动消费者
func (q *ChannelQueue) Consume(ctx context.Context, handler TaskHandler, workers int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.isRunning {
		return errors.New("队列未启动")
	}

	utils.GetLogger(ctx).Info(fmt.Sprintf("启动 %d 个消费者", workers))

	for i := 0; i < workers; i++ {
		q.wg.Add(1)
		go q.worker(ctx, i, handler)
	}

	return nil
}

// Start 启动队列服务
func (q *ChannelQueue) Start(ctx context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.isRunning {
		return errors.New("队列已启动")
	}

	q.isRunning = true
	utils.GetLogger(ctx).Info("消息队列已启动")
	return nil
}

// Stop 停止队列服务（优雅关闭）
func (q *ChannelQueue) Stop(ctx context.Context) error {
	q.mu.Lock()
	if !q.isRunning {
		q.mu.Unlock()
		return errors.New("队列未启动")
	}
	q.isRunning = false
	q.mu.Unlock()

	utils.GetLogger(ctx).Info("正在停止消息队列...")

	// 发送停止信号
	close(q.stopCh)

	// 等待所有消费者完成
	q.wg.Wait()

	// 关闭队列
	close(q.queue)

	utils.GetLogger(ctx).Info("消息队列已停止")
	return nil
}

// GetPendingTasks 获取待处理任务（用于服务重启恢复）
func (q *ChannelQueue) GetPendingTasks(ctx context.Context) ([]*JudgeTask, error) {
	if q.submitSvc == nil {
		return nil, errors.New("提交服务未初始化")
	}

	// 从数据库获取所有 pending 状态的提交
	submits, err := q.submitSvc.GetSubmitsByStatus(ctx, models.StatusPending)
	if err != nil {
		return nil, fmt.Errorf("获取待处理任务失败: %w", err)
	}

	// 转换为判题任务
	tasks := make([]*JudgeTask, 0, len(submits))
	for _, submit := range submits {
		tasks = append(tasks, &JudgeTask{
			SubmitID:  submit.ID,
			ProblemID: submit.ProblemID,
			UserID:    submit.UserID,
			Code:      submit.Code,
			Language:  string(submit.Language), // 转换为 string
		})
	}

	utils.GetLogger(ctx).Info(fmt.Sprintf("恢复 %d 个待处理任务", len(tasks)))
	return tasks, nil
}

// HealthCheck 健康检查
func (q *ChannelQueue) HealthCheck(ctx context.Context) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if !q.isRunning {
		return errors.New("队列未运行")
	}

	// 检查队列是否阻塞
	queueLen := len(q.queue)
	if queueLen >= q.queueSize {
		return fmt.Errorf("队列已满: %d/%d", queueLen, q.queueSize)
	}

	return nil
}

// worker 消费者工作协程
func (q *ChannelQueue) worker(ctx context.Context, workerID int, handler TaskHandler) {
	defer q.wg.Done()

	logger := utils.GetLogger(ctx)
	logger.Info(fmt.Sprintf("消费者 #%d 已启动", workerID))

	for {
		select {
		case task, ok := <-q.queue:
			if !ok {
				// 队列已关闭
				logger.Info(fmt.Sprintf("消费者 #%d 已停止（队列关闭）", workerID))
				return
			}

			// 处理任务
			logger.Info(fmt.Sprintf("消费者 #%d 开始处理任务: SubmitID=%s", workerID, task.SubmitID.Hex()))
			if err := handler(ctx, task); err != nil {
				logger.Error(fmt.Sprintf("消费者 #%d 处理任务失败: SubmitID=%s, Error=%v", workerID, task.SubmitID.Hex(), err))
			} else {
				logger.Info(fmt.Sprintf("消费者 #%d 完成任务: SubmitID=%s", workerID, task.SubmitID.Hex()))
			}

		case <-q.stopCh:
			// 收到停止信号
			logger.Info(fmt.Sprintf("消费者 #%d 已停止（收到停止信号）", workerID))
			return

		case <-ctx.Done():
			// 上下文取消
			logger.Info(fmt.Sprintf("消费者 #%d 已停止（上下文取消）", workerID))
			return
		}
	}
}

// RecoverRunningTasks 恢复运行中的任务（标记为系统错误）
func (q *ChannelQueue) RecoverRunningTasks(ctx context.Context) error {
	if q.submitSvc == nil {
		return errors.New("提交服务未初始化")
	}

	// 批量更新所有 running 状态的提交为 system_error
	collection := utils.GetCollection("submits")
	filter := bson.M{"status": models.StatusRunning}
	update := bson.M{
		"$set": bson.M{
			"status": models.StatusSystemError,
			"result": &models.JudgeResult{
				Status:      models.StatusSystemError,
				TimeUsed:    0,
				MemoryUsed:  0,
				TestResults: []models.TestResult{},
			},
		},
	}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("批量更新运行中任务失败: %w", err)
	}

	utils.GetLogger(ctx).Info(fmt.Sprintf("恢复 %d 个运行中任务（标记为系统错误）", result.ModifiedCount))
	return nil
}
