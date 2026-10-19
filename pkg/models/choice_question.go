/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型选择题模型定义
*/

package models

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChoiceMode 表示选择题是单选还是多选。
type ChoiceMode string

const (
	ChoiceModeSingle   ChoiceMode = "single"   // 单选题只允许一个正确答案。
	ChoiceModeMultiple ChoiceMode = "multiple" // 多选题允许多个正确答案。
)

// ChoiceOption 表示选择题单个选项。
type ChoiceOption struct {
	Key     string `bson:"key" json:"key" example:"A"`
	Content string `bson:"content" json:"content" example:"Gin"`
}

// ChoiceQuestion 表示选择题实体。
type ChoiceQuestion struct {
	ID             primitive.ObjectID  `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Title          string              `bson:"title" json:"title" example:"Go 框架选择题"`
	Stem           string              `bson:"stem" json:"stem" example:"下面哪些是 Go Web 框架？"`
	Difficulty     ProblemDifficulty   `bson:"difficulty" json:"difficulty" example:"easy"`
	Tags           []string            `bson:"tags" json:"tags" example:"Go,Web"`
	Status         QuestionStatus      `bson:"status" json:"status" example:"published"`
	PublishMode    QuestionPublishMode `bson:"publish_mode" json:"publish_mode" example:"course_only"`
	ChoiceMode     ChoiceMode          `bson:"choice_mode" json:"choice_mode" example:"multiple"`
	Options        []ChoiceOption      `bson:"options" json:"options"`
	CorrectAnswers []string            `bson:"correct_answers" json:"correct_answers" example:"A,B"`
	Analysis       string              `bson:"analysis,omitempty" json:"analysis,omitempty" example:"Gin 和 Echo 都是常见 Go Web 框架。"`
	Score          int                 `bson:"score" json:"score" example:"10"`
	CreatedBy      primitive.ObjectID  `bson:"created_by" json:"created_by" swaggertype:"string" example:"507f1f77bcf86cd799439012"`
	CreatedAt      time.Time           `bson:"created_at" json:"created_at" example:"2026-05-08T16:31:55Z"`
	UpdatedAt      time.Time           `bson:"updated_at" json:"updated_at" example:"2026-05-08T16:31:55Z"`
}

// Validate 校验选择题的最小结构，先拦住会破坏判题语义的无效数据。
func (q *ChoiceQuestion) Validate() error {
	if q == nil {
		return fmt.Errorf("选择题不能为空")
	}
	switch q.ChoiceMode {
	case ChoiceModeSingle, ChoiceModeMultiple:
	default:
		return fmt.Errorf("选择题作答模式非法")
	}
	if len(q.Options) < 2 {
		return fmt.Errorf("选择题至少需要两个选项")
	}
	// 先收集合法选项键，避免答案引用不存在的选项，导致后续判题语义不稳定。
	optionKeys := make(map[string]struct{}, len(q.Options))
	for _, option := range q.Options {
		// 选择题选项键会直接参与答案判定，拒绝首尾空格可以避免“展示一致、判题不同”的隐性输入。
		if option.Key == "" {
			return fmt.Errorf("选择题选项标识不能为空")
		}
		if option.Key != strings.TrimSpace(option.Key) {
			return fmt.Errorf("选择题选项标识不能包含首尾空格")
		}
		if _, exists := optionKeys[option.Key]; exists {
			return fmt.Errorf("选择题选项标识不能重复")
		}
		optionKeys[option.Key] = struct{}{}
	}
	if len(q.CorrectAnswers) == 0 {
		return fmt.Errorf("选择题至少需要一个正确答案")
	}
	answerKeys := make(map[string]struct{}, len(q.CorrectAnswers))
	for _, answer := range q.CorrectAnswers {
		// 正确答案需要和选项键做精确匹配，因此首尾空格必须作为非法输入显式拒绝。
		if answer == "" {
			return fmt.Errorf("选择题正确答案不能为空")
		}
		if answer != strings.TrimSpace(answer) {
			return fmt.Errorf("选择题正确答案不能包含首尾空格")
		}
		if _, exists := answerKeys[answer]; exists {
			return fmt.Errorf("选择题正确答案不能重复")
		}
		answerKeys[answer] = struct{}{}
		if _, ok := optionKeys[answer]; !ok {
			return fmt.Errorf("选择题正确答案必须匹配已有选项")
		}
	}
	if q.ChoiceMode == ChoiceModeSingle && len(q.CorrectAnswers) != 1 {
		return fmt.Errorf("单选题只能有一个正确答案")
	}
	return nil
}
