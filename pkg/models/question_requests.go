/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型请求参数定义
*/

package models

import "fmt"

// CreateChoiceQuestionRequest 定义创建选择题请求。
type CreateChoiceQuestionRequest struct {
	Title       string              `json:"title" binding:"required,min=1,max=200" example:"Go 框架选择题"`
	Stem        string              `json:"stem" binding:"required,min=1" example:"下面哪些是 Go Web 框架？"`
	Difficulty  ProblemDifficulty   `json:"difficulty" binding:"required,oneof=easy medium hard" example:"easy"`
	Tags        []string            `json:"tags" binding:"max=10,dive,min=1,max=20" example:"Go,Web"`
	Status      *QuestionStatus     `json:"status" binding:"omitempty,oneof=draft published archived" example:"published"`
	PublishMode QuestionPublishMode `json:"publish_mode" binding:"required,oneof=global course_only global_and_course" example:"course_only"`
	CourseIDs   []string            `json:"course_ids" binding:"omitempty,dive,len=24" example:"507f1f77bcf86cd799439031"`
	ChoiceMode  ChoiceMode          `json:"choice_mode" binding:"required,oneof=single multiple" example:"multiple"`
	Options     []ChoiceOption      `json:"options" binding:"required,min=2,dive"`
	Answers     []string            `json:"answers" binding:"required,min=1" example:"A,B"`
	Analysis    string              `json:"analysis" example:"Gin 和 Echo 都是常见 Go Web 框架。"`
	Score       int                 `json:"score" binding:"omitempty,min=0,max=100" example:"10"`
}

// Validate 校验创建选择题请求中的跨字段约束，避免仅靠 tag 校验漏掉发布边界。
func (r *CreateChoiceQuestionRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("创建选择题请求不能为空")
	}
	if err := ValidatePublishModeCourseIDs(r.PublishMode, r.CourseIDs); err != nil {
		return err
	}
	// 请求层直接复用实体语义校验，避免接口层放过会破坏判题一致性的非法题目数据。
	return (&ChoiceQuestion{
		ChoiceMode:     r.ChoiceMode,
		Options:        r.Options,
		CorrectAnswers: r.Answers,
	}).Validate()
}

// CreateBlankQuestionRequest 定义创建填空题请求。
type CreateBlankQuestionRequest struct {
	Title       string              `json:"title" binding:"required,min=1,max=200" example:"Go 填空题"`
	Stem        string              `json:"stem" binding:"required,min=1" example:"Go 常见 Web 框架有 ____ 和 ____。"`
	Difficulty  ProblemDifficulty   `json:"difficulty" binding:"required,oneof=easy medium hard" example:"easy"`
	Tags        []string            `json:"tags" binding:"max=10,dive,min=1,max=20" example:"Go,Web"`
	Status      *QuestionStatus     `json:"status" binding:"omitempty,oneof=draft published archived" example:"published"`
	PublishMode QuestionPublishMode `json:"publish_mode" binding:"required,oneof=global course_only global_and_course" example:"course_only"`
	CourseIDs   []string            `json:"course_ids" binding:"omitempty,dive,len=24" example:"507f1f77bcf86cd799439032"`
	JudgeMode   BlankJudgeMode      `json:"judge_mode" binding:"required,oneof=exact_match manual" example:"exact_match"`
	Blanks      []BlankSlot         `json:"blanks" binding:"required,min=1,dive"`
	Analysis    string              `json:"analysis" example:"Gin 和 Echo 都是常见答案。"`
	Score       int                 `json:"score" binding:"omitempty,min=0,max=100" example:"10"`
}

// Validate 校验创建填空题请求中的跨字段约束，保证发布范围与课程挂载语义一致。
func (r *CreateBlankQuestionRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("创建填空题请求不能为空")
	}
	if err := ValidatePublishModeCourseIDs(r.PublishMode, r.CourseIDs); err != nil {
		return err
	}
	// 填空题请求需要和实体共享同一套语义边界，防止重复空位或非法标准答案从请求层漏进来。
	return (&BlankQuestion{
		JudgeMode: r.JudgeMode,
		Blanks:    r.Blanks,
	}).Validate()
}

// UpdateChoiceQuestionRequest 定义更新选择题请求。
type UpdateChoiceQuestionRequest struct {
	Title       *string              `json:"title" binding:"omitempty,min=1,max=200" example:"Go 框架选择题-更新"`
	Stem        *string              `json:"stem" binding:"omitempty,min=1" example:"下面哪些是 Go Web 框架？"`
	Difficulty  *ProblemDifficulty   `json:"difficulty" binding:"omitempty,oneof=easy medium hard" example:"easy"`
	Tags        *[]string            `json:"tags" binding:"omitempty,max=10,dive,min=1,max=20" example:"Go,Web,HTTP"`
	Status      *QuestionStatus      `json:"status" binding:"omitempty,oneof=draft published archived" example:"published"`
	PublishMode *QuestionPublishMode `json:"publish_mode" binding:"omitempty,oneof=global course_only global_and_course" example:"course_only"`
	CourseIDs   *[]string            `json:"course_ids" binding:"omitempty,dive,len=24" example:"507f1f77bcf86cd799439031"`
	ChoiceMode  *ChoiceMode          `json:"choice_mode" binding:"omitempty,oneof=single multiple" example:"multiple"`
	Options     *[]ChoiceOption      `json:"options" binding:"omitempty,min=2,dive"`
	Answers     *[]string            `json:"answers" binding:"omitempty,min=1" example:"A,B"`
	Analysis    *string              `json:"analysis" example:"Gin 和 Echo 都是常见 Go Web 框架。"`
	Score       *int                 `json:"score" binding:"omitempty,min=0,max=100" example:"15"`
}

// UpdateBlankQuestionRequest 定义更新填空题请求。
type UpdateBlankQuestionRequest struct {
	Title       *string              `json:"title" binding:"omitempty,min=1,max=200" example:"Go 填空题-更新"`
	Stem        *string              `json:"stem" binding:"omitempty,min=1" example:"Go 常见 Web 框架有 ____ 和 ____。"`
	Difficulty  *ProblemDifficulty   `json:"difficulty" binding:"omitempty,oneof=easy medium hard" example:"easy"`
	Tags        *[]string            `json:"tags" binding:"omitempty,max=10,dive,min=1,max=20" example:"Go,Web"`
	Status      *QuestionStatus      `json:"status" binding:"omitempty,oneof=draft published archived" example:"published"`
	PublishMode *QuestionPublishMode `json:"publish_mode" binding:"omitempty,oneof=global course_only global_and_course" example:"course_only"`
	CourseIDs   *[]string            `json:"course_ids" binding:"omitempty,dive,len=24" example:"507f1f77bcf86cd799439032"`
	JudgeMode   *BlankJudgeMode      `json:"judge_mode" binding:"omitempty,oneof=exact_match manual" example:"exact_match"`
	Blanks      *[]BlankSlot         `json:"blanks" binding:"omitempty,min=1,dive"`
	Analysis    *string              `json:"analysis" example:"Gin 和 Fiber 都是常见答案。"`
	Score       *int                 `json:"score" binding:"omitempty,min=0,max=100" example:"12"`
}

// BlankAnswerItem 定义填空题单个空位的作答内容。
type BlankAnswerItem struct {
	Order  int    `json:"order" binding:"required,min=1" example:"1"`
	Answer string `json:"answer" binding:"required" example:"Gin"`
}

// SubmitContentAnswerRequest 定义统一内容单题提交请求。
// 当前阶段先保留独立作答入口，任务内统一提交会在后续任务系统接入时补齐。
type SubmitContentAnswerRequest struct {
	ContentType   ContentType       `json:"content_type" binding:"required,oneof=choice blank" example:"choice"`
	ContentID     string            `json:"content_id" binding:"required,len=24" example:"507f1f77bcf86cd799439041"`
	CourseID      string            `json:"course_id,omitempty" binding:"omitempty,len=24" example:"507f1f77bcf86cd799439042"`
	ChoiceAnswers []string          `json:"choice_answers,omitempty" example:"A,B"`
	BlankAnswers  []BlankAnswerItem `json:"blank_answers,omitempty"`
}

// Validate 校验统一内容单题提交请求，避免不同题型的答案结构串用。
func (r *SubmitContentAnswerRequest) Validate() error {
	if r == nil {
		return fmt.Errorf("内容提交请求不能为空")
	}
	switch r.ContentType {
	case ContentTypeChoice:
		if len(r.ChoiceAnswers) == 0 {
			return fmt.Errorf("选择题提交至少需要一个答案")
		}
		if len(r.BlankAnswers) > 0 {
			return fmt.Errorf("选择题提交不能携带填空答案")
		}
	case ContentTypeBlank:
		if len(r.BlankAnswers) == 0 {
			return fmt.Errorf("填空题提交至少需要一个空位答案")
		}
		if len(r.ChoiceAnswers) > 0 {
			return fmt.Errorf("填空题提交不能携带选择题答案")
		}
		orders := make(map[int]struct{}, len(r.BlankAnswers))
		for _, item := range r.BlankAnswers {
			if item.Order <= 0 {
				return fmt.Errorf("填空题空位顺序必须大于 0")
			}
			if _, exists := orders[item.Order]; exists {
				return fmt.Errorf("填空题空位顺序不能重复")
			}
			orders[item.Order] = struct{}{}
		}
	default:
		return fmt.Errorf("内容类型非法")
	}
	return nil
}
