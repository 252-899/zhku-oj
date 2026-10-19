/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一内容提交记录模型定义
*/

package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SubmissionMode 表示提交是独立作答还是任务内作答。
type SubmissionMode string

const (
	SubmissionModeStandalone SubmissionMode = "standalone" // 独立作答，不绑定任务关系。
	SubmissionModeTask       SubmissionMode = "task"       // 任务内作答，保留任务上下文。
)

// ContentSubmission 表示统一内容提交记录。
type ContentSubmission struct {
	ID                    primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	UserID                primitive.ObjectID  `bson:"user_id" json:"user_id"`
	ContentType           ContentType         `bson:"content_type" json:"content_type"`
	ContentID             primitive.ObjectID  `bson:"content_id" json:"content_id"`
	SubmissionMode        SubmissionMode      `bson:"submission_mode" json:"submission_mode"`
	CourseID              primitive.ObjectID  `bson:"course_id" json:"course_id"`
	TaskID                *primitive.ObjectID `bson:"task_id,omitempty" json:"task_id,omitempty"`
	TaskContentRelationID *primitive.ObjectID `bson:"task_content_relation_id,omitempty" json:"task_content_relation_id,omitempty"`
	Answers               map[string]any      `bson:"answers" json:"answers"`
	JudgeStatus           string              `bson:"judge_status" json:"judge_status"`
	Score                 int                 `bson:"score" json:"score"`
	IsCorrect             bool                `bson:"is_correct" json:"is_correct"`
	SubmittedAt           time.Time           `bson:"submitted_at" json:"submitted_at"`
}

// Validate 校验提交记录的最小关系约束，避免独立作答和任务作答的上下文串用。
func (s *ContentSubmission) Validate() error {
	if s == nil {
		return fmt.Errorf("内容提交不能为空")
	}
	switch s.SubmissionMode {
	case SubmissionModeStandalone:
		// 独立作答不允许携带任务关系，避免后续统计和回溯时误入任务维度。
		if s.TaskID != nil || s.TaskContentRelationID != nil {
			return fmt.Errorf("独立作答不能绑定任务关系")
		}
	case SubmissionModeTask:
		// 任务内作答需要同时保留任务与题目挂载关系，否则无法稳定回溯到任务上下文。
		if s.TaskID == nil || s.TaskContentRelationID == nil {
			return fmt.Errorf("任务作答必须同时绑定任务和任务内容关系")
		}
	default:
		return fmt.Errorf("提交模式非法")
	}
	return nil
}
