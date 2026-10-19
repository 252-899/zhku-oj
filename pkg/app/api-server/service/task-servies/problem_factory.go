/*
@Author: sir
@Date: 2025/10/6
@Name: problem_factory.go
@Description: 题目任务处理策略实现
*/

package taskstrategy

import (
	"context"
	"errors"
	"time"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProblemTaskHandler 题目任务策略
type ProblemTaskHandler struct{}

// NewProblemTaskHandler 创建题目任务策略实例
func NewProblemTaskHandler() *ProblemTaskHandler {
	return &ProblemTaskHandler{}
}

// HandleTask 处理题目任务完成逻辑
func (s *ProblemTaskHandler) HandleTask(ctx context.Context, task *models.Task, relationID primitive.ObjectID, userID primitive.ObjectID) error {
	collRelasUsers := utils.GetCollection("relations_users")
	document := models.RelationsUsers{
		ID:         primitive.NewObjectID(),
		TaskID:     task.ID,
		RelationID: relationID,
		UserID:     userID,
		CTime:      time.Now(),
	}
	if _, err := collRelasUsers.InsertOne(ctx, document); err != nil {
		return err
	}

	remainingTime := task.EndTime.Sub(time.Now())
	if remainingTime <= 0 {
		return errors.New("任务时间截至")
	}

	redisClient := utils.GetRedisClient()
	newValue := redisClient.Incr(ctx, "taskID:"+task.ID.Hex()+":"+userID.Hex())

	redisClient.Expire(ctx, "taskID:"+task.ID.Hex(), remainingTime)
	result, err := newValue.Result()
	if err != nil {
		return err
	}

	if int(result) == len(task.RelationIDs) {
		filter := bson.M{"_id": task.ID}
		update := bson.M{
			"$addToSet": bson.M{
				"finish_ids": userID,
			},
		}

		result, err := utils.GetCollection("tasks").UpdateOne(ctx, filter, update)
		if err != nil {
			return errors.New("完成任务失败: " + err.Error())
		}

		if result.MatchedCount == 0 {
			return errors.New("找不到该任务")
		}

	}
	return nil
}
