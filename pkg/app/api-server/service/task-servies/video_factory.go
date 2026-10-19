/*
@Author: sir
@Date: 2025/10/6
@Name: video_factory.go
@Description: 视频任务处理策略实现
*/

package taskstrategy

import (
	"context"
	"errors"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VideoTaskHandler 视频任务策略
type VideoTaskHandler struct{}

// NewVideoTaskHandler 创建视频任务策略实例
func NewVideoTaskHandler() *VideoTaskHandler {
	return &VideoTaskHandler{}
}

// HandleTask 处理视频任务完成逻辑
func (s *VideoTaskHandler) HandleTask(ctx context.Context, task *models.Task, relationID primitive.ObjectID, userID primitive.ObjectID) error {
	// 视频任务的处理逻辑
	// 可以记录用户完成视频任务的时间等信息

	//TODO
	return errors.New("接口未完成")
}
