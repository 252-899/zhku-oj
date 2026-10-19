/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型响应模型定义
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CourseContentListItem 表示课程混合内容列表中的统一条目。
type CourseContentListItem struct {
	ContentType ContentType         `json:"content_type" example:"choice"`
	ContentID   primitive.ObjectID  `json:"content_id" swaggertype:"string" example:"507f1f77bcf86cd799439051"`
	Title       string              `json:"title" example:"Go 框架选择题"`
	Status      QuestionStatus      `json:"status" example:"published"`
	PublishMode QuestionPublishMode `json:"publish_mode" example:"course_only"`
	Difficulty  ProblemDifficulty   `json:"difficulty" example:"easy"`
	CreatedBy   primitive.ObjectID  `json:"created_by" swaggertype:"string" example:"507f1f77bcf86cd799439052"`
	CreatedAt   time.Time           `json:"created_at" example:"2026-05-08T16:31:55Z"`
}

// CourseContentListResponse 表示课程混合内容分页响应。
type CourseContentListResponse struct {
	Items     []*CourseContentListItem `json:"items"`
	Total     int64                    `json:"total" example:"2"`
	Page      int                      `json:"page" example:"1"`
	PageSize  int                      `json:"page_size" example:"20"`
	TotalPage int64                    `json:"total_page" example:"1"`
}

// ContentSubmissionResult 表示单题提交后的最小判题回执。
type ContentSubmissionResult struct {
	SubmissionID primitive.ObjectID `json:"submission_id" swaggertype:"string" example:"507f1f77bcf86cd799439061"`
	JudgeStatus  string             `json:"judge_status" example:"judged"`
	Score        int                `json:"score" example:"10"`
	IsCorrect    bool               `json:"is_correct" example:"true"`
}

// CreateChoiceQuestionResponse 表示创建课程选择题的响应体数据。
type CreateChoiceQuestionResponse struct {
	QuestionID primitive.ObjectID `json:"question_id" swaggertype:"string" example:"507f1f77bcf86cd799439071"`
	Question   *ChoiceQuestion    `json:"question"`
}

// CreateBlankQuestionResponse 表示创建课程填空题的响应体数据。
type CreateBlankQuestionResponse struct {
	QuestionID primitive.ObjectID `json:"question_id" swaggertype:"string" example:"507f1f77bcf86cd799439072"`
	Question   *BlankQuestion     `json:"question"`
}
