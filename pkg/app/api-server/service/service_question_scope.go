/*
@Author: Codex
@Date: 2026/05/13
@Description: 题目通用发布范围辅助逻辑，负责解析课程挂载集合与跨课程可见性判断
*/

package service

import (
	"context"
	"fmt"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// parseQuestionScopeCourseIDs 解析通用题目接口传入的课程挂载范围。
// 这里不隐式注入课程上下文，避免通用接口在全局题库和课程题库之间产生隐藏副作用。
func parseQuestionScopeCourseIDs(publishMode models.QuestionPublishMode, rawCourseIDs []string) ([]primitive.ObjectID, error) {
	if publishMode == models.QuestionPublishModeGlobal {
		return []primitive.ObjectID{}, nil
	}
	courseIDs := make([]primitive.ObjectID, 0, len(rawCourseIDs))
	seen := make(map[string]struct{}, len(rawCourseIDs))
	for _, rawCourseID := range rawCourseIDs {
		courseID, err := primitive.ObjectIDFromHex(rawCourseID)
		if err != nil {
			return nil, fmt.Errorf("无效的课程ID: %s", rawCourseID)
		}
		if _, exists := seen[courseID.Hex()]; exists {
			continue
		}
		seen[courseID.Hex()] = struct{}{}
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, nil
}

// ensureCanViewAnyCourse 校验用户是否至少能访问一个挂载课程。
// 通用题目详情在缺少显式 courseId 时，只能退化为“只要能在任一课程题库看见它即可读取”。
func ensureCanViewAnyCourse(
	ctx context.Context,
	courseService *CourseService,
	courseIDs []primitive.ObjectID,
	userID primitive.ObjectID,
) error {
	if len(courseIDs) == 0 {
		return fmt.Errorf("题目未挂载到任何课程")
	}
	if courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	for _, courseID := range courseIDs {
		canView, err := courseService.CanViewCourse(ctx, courseID, userID)
		if err != nil {
			continue
		}
		if canView {
			return nil
		}
	}
	return fmt.Errorf("无权查看该课程题目")
}
