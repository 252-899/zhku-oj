/*
@Author: sir
@Date: 2025/10/6
@Name: task_factory.go
@Description: 任务处理策略模式接口定义
*/

package taskstrategy

import (
	"context"
	"errors"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskHandler 任务处理策略接口
type TaskHandler interface {
	HandleTask(ctx context.Context, task *models.Task, relationID primitive.ObjectID, userID primitive.ObjectID) error
}

// TaskHandlerFactory 任务策略工厂
type TaskHandlerFactory struct{}

// NewTaskHandlerFactory 创建任务策略工厂
func NewTaskHandlerFactory() *TaskHandlerFactory {
	return &TaskHandlerFactory{}
}

// GetHandler 根据任务类型创建对应的策略
func (f *TaskHandlerFactory) GetHandler(taskType models.TaskType) (TaskHandler, error) {
	switch taskType {
	case models.TaskTypeProblem:
		return NewProblemTaskHandler(), nil
	case models.TaskTypeVideo:
		return NewVideoTaskHandler(), nil
	default:
		return nil, errors.New("未知任务处理器")
	}
}
