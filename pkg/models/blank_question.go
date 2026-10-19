/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合题型填空题模型定义
*/

package models

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BlankJudgeMode 表示填空题整题使用的判分方式。
type BlankJudgeMode string

const (
	BlankJudgeModeExactMatch BlankJudgeMode = "exact_match" // 自动判分时要求答案精确匹配。
	BlankJudgeModeManual     BlankJudgeMode = "manual"      // 保留人工判分场景，不强制标准答案。
)

// BlankSlot 表示填空题单个空位。
type BlankSlot struct {
	Order   int      `bson:"order" json:"order" example:"1"`
	Answers []string `bson:"answers" json:"answers" example:"Gin,gin"`
}

// BlankQuestion 表示填空题实体。
type BlankQuestion struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439021"`
	Title       string              `bson:"title" json:"title" example:"Go 填空题"`
	Stem        string              `bson:"stem" json:"stem" example:"Go 常见 Web 框架有 ____ 和 ____。"`
	Difficulty  ProblemDifficulty   `bson:"difficulty" json:"difficulty" example:"easy"`
	Tags        []string            `bson:"tags" json:"tags" example:"Go,Web"`
	Status      QuestionStatus      `bson:"status" json:"status" example:"published"`
	PublishMode QuestionPublishMode `bson:"publish_mode" json:"publish_mode" example:"course_only"`
	JudgeMode   BlankJudgeMode      `bson:"judge_mode" json:"judge_mode" example:"exact_match"`
	Blanks      []BlankSlot         `bson:"blanks" json:"blanks"`
	Analysis    string              `bson:"analysis,omitempty" json:"analysis,omitempty" example:"Gin 和 Echo 都是常见答案。"`
	Score       int                 `bson:"score" json:"score" example:"10"`
	CreatedBy   primitive.ObjectID  `bson:"created_by" json:"created_by" swaggertype:"string" example:"507f1f77bcf86cd799439022"`
	CreatedAt   time.Time           `bson:"created_at" json:"created_at" example:"2026-05-08T16:31:55Z"`
	UpdatedAt   time.Time           `bson:"updated_at" json:"updated_at" example:"2026-05-08T16:31:55Z"`
}

// Validate 校验填空题的最小结构，确保自动判分模式不会缺失关键答案。
func (q *BlankQuestion) Validate() error {
	if q == nil {
		return fmt.Errorf("填空题不能为空")
	}
	switch q.JudgeMode {
	case BlankJudgeModeExactMatch, BlankJudgeModeManual:
	default:
		return fmt.Errorf("填空题判分模式非法")
	}
	if len(q.Blanks) == 0 {
		return fmt.Errorf("填空题至少需要一个空位")
	}
	// 空位顺序会被提交答案和判分结果复用，重复顺序会让同一空位的映射失去确定性。
	orders := make(map[int]struct{}, len(q.Blanks))
	for _, blank := range q.Blanks {
		if blank.Order <= 0 {
			return fmt.Errorf("空位顺序必须大于 0")
		}
		if _, exists := orders[blank.Order]; exists {
			return fmt.Errorf("填空题空位顺序不能重复")
		}
		orders[blank.Order] = struct{}{}
		if len(blank.Answers) == 0 && q.JudgeMode == BlankJudgeModeExactMatch {
			return fmt.Errorf("自动判分填空题的每个空位都需要标准答案")
		}
		answers := make(map[string]struct{}, len(blank.Answers))
		for _, answer := range blank.Answers {
			// 精确判分依赖答案原文，拒绝首尾空格可以避免存储值和判题值出现隐式偏差。
			if answer == "" {
				return fmt.Errorf("填空题标准答案不能为空字符串")
			}
			if answer != strings.TrimSpace(answer) {
				return fmt.Errorf("填空题标准答案不能包含首尾空格")
			}
			if _, exists := answers[answer]; exists {
				return fmt.Errorf("填空题标准答案不能重复")
			}
			answers[answer] = struct{}{}
		}
	}
	return nil
}
