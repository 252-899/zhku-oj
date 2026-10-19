/*
@Author: omenkk7
@Date: 2025/10/05
@Description: 消息队列接口定义
*/

package queue

import (
	"context"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageQueue 消息队列接口
// 定义统一的消息队列规范，支持多种实现（Channel、RabbitMQ、Kafka 等）v1.0采用channel实现 保留后续扩展性
type MessageQueue interface {
	// Push 推送任务到队列（生产者接口）
	// ctx: 上下文
	// task: 判题任务
	// 返回: 错误信息
	Push(ctx context.Context, task *JudgeTask) error

	// Consume 启动消费者（消费者接口）
	// ctx: 上下文
	// handler: 任务处理函数
	// workers: 消费者数量
	// 返回: 错误信息
	Consume(ctx context.Context, handler TaskHandler, workers int) error

	// Start 启动队列服务
	// ctx: 上下文
	// 返回: 错误信息
	Start(ctx context.Context) error

	// Stop 停止队列服务（优雅关闭）
	// ctx: 上下文
	// 返回: 错误信息
	Stop(ctx context.Context) error

	// GetPendingTasks 获取待处理任务（用于服务重启恢复）
	// ctx: 上下文
	// 返回: 待处理任务列表、错误信息
	GetPendingTasks(ctx context.Context) ([]*JudgeTask, error)

	// HealthCheck 健康检查
	// ctx: 上下文
	// 返回: 错误信息
	HealthCheck(ctx context.Context) error
}

// TaskHandler 任务处理函数类型
// ctx: 上下文
// task: 判题任务
// 返回: 错误信息
type TaskHandler func(ctx context.Context, task *JudgeTask) error

// JudgeTask 判题任务
type JudgeTask struct {
	SubmitID  primitive.ObjectID `json:"submit_id"`  // 提交 ID
	ProblemID primitive.ObjectID `json:"problem_id"` // 题目 ID
	UserID    primitive.ObjectID `json:"user_id"`    // 用户 ID
	Code      string             `json:"code"`       // 代码
	Language  string             `json:"language"`   // 语言
}
