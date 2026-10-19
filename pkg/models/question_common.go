/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型共享元数据定义
*/

package models

import "fmt"

// QuestionStatus 表示新题型题目的生命周期状态。
type QuestionStatus string

const (
	QuestionStatusDraft     QuestionStatus = "draft"     // 草稿态，允许继续编辑。
	QuestionStatusPublished QuestionStatus = "published" // 已发布，可参与课程题库或任务。
	QuestionStatusArchived  QuestionStatus = "archived"  // 已归档，仅保留历史记录。
)

// QuestionPublishMode 表示题目发布范围。
type QuestionPublishMode string

const (
	QuestionPublishModeGlobal          QuestionPublishMode = "global"            // 进入全局题库。
	QuestionPublishModeCourseOnly      QuestionPublishMode = "course_only"       // 仅在课程上下文可见。
	QuestionPublishModeGlobalAndCourse QuestionPublishMode = "global_and_course" // 同时进入全局与课程题库。
)

// ValidatePublishModeCourseIDs 校验发布范围与课程挂载列表的组合约束。
func ValidatePublishModeCourseIDs(publishMode QuestionPublishMode, courseIDs []string) error {
	switch publishMode {
	case QuestionPublishModeGlobal:
		// 全局题库题目不应再携带课程范围，避免调用方误以为会自动建立课程挂载。
		if len(courseIDs) > 0 {
			return fmt.Errorf("全局发布题目不能携带课程范围")
		}
	case QuestionPublishModeCourseOnly, QuestionPublishModeGlobalAndCourse:
		// 涉及课程可见性的模式必须显式给出课程集合，否则题目可见边界不明确。
		if len(courseIDs) == 0 {
			return fmt.Errorf("课程发布题目必须指定课程范围")
		}
	default:
		return fmt.Errorf("题目发布范围非法")
	}
	return nil
}
