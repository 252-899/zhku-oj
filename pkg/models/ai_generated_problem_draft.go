/*
@Author: Codex
@Date: 2026/4/19
@Name: ai_generated_problem_draft.go
@Description: AI 生成题目草稿模型，用于在教师确认前保存临时生成结果
*/

package models

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AIGeneratedProblemDraftStatus AI 生成题目草稿整体状态。
type AIGeneratedProblemDraftStatus string

const (
	// AIGeneratedProblemDraftPending 表示草稿等待教师选择保存。
	AIGeneratedProblemDraftPending AIGeneratedProblemDraftStatus = "pending"
	// AIGeneratedProblemDraftSaved 表示选中题目已处理完成。
	AIGeneratedProblemDraftSaved AIGeneratedProblemDraftStatus = "saved"
	// AIGeneratedProblemDraftPartiallySaved 表示部分题目保存失败。
	AIGeneratedProblemDraftPartiallySaved AIGeneratedProblemDraftStatus = "partially_saved"
	// AIGeneratedProblemDraftExpired 表示草稿已过期。
	AIGeneratedProblemDraftExpired AIGeneratedProblemDraftStatus = "expired"
)

// AIGeneratedDraftProblemStatus AI 生成草稿内单题状态。
type AIGeneratedDraftProblemStatus string

const (
	// AIGeneratedDraftProblemGenerated 表示题目已生成，等待教师处理。
	AIGeneratedDraftProblemGenerated AIGeneratedDraftProblemStatus = "generated"
	// AIGeneratedDraftProblemSaved 表示题目已保存到正式题库。
	AIGeneratedDraftProblemSaved AIGeneratedDraftProblemStatus = "saved"
	// AIGeneratedDraftProblemDiscarded 表示题目未被教师选中，后续由草稿 TTL 清理。
	AIGeneratedDraftProblemDiscarded AIGeneratedDraftProblemStatus = "discarded"
	// AIGeneratedDraftProblemFailed 表示题目保存失败。
	AIGeneratedDraftProblemFailed AIGeneratedDraftProblemStatus = "failed"
)

// AIGeneratedProblemDraft AI 生成题目草稿。
type AIGeneratedProblemDraft struct {
	ID         primitive.ObjectID            `bson:"_id,omitempty" json:"id"`
	SessionID  string                        `bson:"session_id" json:"session_id"`
	RoundID    string                        `bson:"round_id" json:"round_id"`
	UserID     primitive.ObjectID            `bson:"user_id" json:"user_id"`
	ModuleType string                        `bson:"module_type" json:"module_type"`
	Status     AIGeneratedProblemDraftStatus `bson:"status" json:"status"`
	Problems   []AIGeneratedDraftProblem     `bson:"problems" json:"problems"`
	CreatedAt  time.Time                     `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time                     `bson:"updated_at" json:"updated_at"`
	ExpiresAt  time.Time                     `bson:"expires_at" json:"expires_at"`
	SavedAt    *time.Time                    `bson:"saved_at,omitempty" json:"saved_at,omitempty"`
}

// AIGeneratedDraftProblem AI 生成草稿内的单道题目。
type AIGeneratedDraftProblem struct {
	GeneratedProblemID string                        `bson:"generated_problem_id" json:"generated_problem_id"`
	Title              string                        `bson:"title" json:"title"`
	Description        string                        `bson:"description" json:"description"`
	InputFormat        string                        `bson:"input_format" json:"input_format"`
	OutputFormat       string                        `bson:"output_format" json:"output_format"`
	Hint               string                        `bson:"hint,omitempty" json:"hint,omitempty"`
	Difficulty         ProblemDifficulty             `bson:"difficulty" json:"difficulty"`
	StandardCode       string                        `bson:"standard_code,omitempty" json:"standard_code,omitempty"`
	StandardLanguage   string                        `bson:"standard_language,omitempty" json:"standard_language,omitempty"`
	TestCases          []AIGeneratedDraftTestCase    `bson:"test_cases" json:"test_cases"`
	Verification       *AIGeneratedDraftVerification `bson:"verification,omitempty" json:"verification,omitempty"`
	Status             AIGeneratedDraftProblemStatus `bson:"status" json:"status"`
	SavedProblemID     *primitive.ObjectID           `bson:"saved_problem_id,omitempty" json:"saved_problem_id,omitempty"`
	SaveError          string                        `bson:"save_error,omitempty" json:"save_error,omitempty"`
}

// AIGeneratedDraftTestCase AI 生成草稿中的测试用例。
type AIGeneratedDraftTestCase struct {
	Input          string `bson:"input" json:"input"`
	ExpectedOutput string `bson:"expected_output" json:"expected_output"`
	IsSample       bool   `bson:"is_sample" json:"is_sample"`
}

// AIGeneratedDraftVerification AI 生成题目的验证结果快照。
type AIGeneratedDraftVerification struct {
	Passed     bool   `bson:"passed" json:"passed"`
	Message    string `bson:"message" json:"message"`
	FailedCase int    `bson:"failed_case,omitempty" json:"failed_case,omitempty"`
}

// FindProblem 根据生成题目 ID 查找草稿内单题。
func (d *AIGeneratedProblemDraft) FindProblem(generatedProblemID string) (*AIGeneratedDraftProblem, bool) {
	if d == nil {
		return nil, false
	}
	generatedProblemID = strings.TrimSpace(generatedProblemID)
	for i := range d.Problems {
		if d.Problems[i].GeneratedProblemID == generatedProblemID {
			return &d.Problems[i], true
		}
	}
	return nil, false
}

// AIGeneratedProblemSaveRequest AI 生成题目确认保存请求。
type AIGeneratedProblemSaveRequest struct {
	DraftID string                       `json:"draft_id" binding:"required"`
	Items   []AIGeneratedProblemSaveItem `json:"items" binding:"required,min=1"`
}

// AIGeneratedProblemSaveItem 单道 AI 生成题目的保存配置。
type AIGeneratedProblemSaveItem struct {
	GeneratedProblemID string   `json:"generated_problem_id" binding:"required"`
	IsPublic           bool     `json:"is_public"`
	CourseIDs          []string `json:"course_ids" binding:"omitempty,dive,len=24"`
}

// Validate 校验保存请求的业务边界。
func (r AIGeneratedProblemSaveRequest) Validate() error {
	if strings.TrimSpace(r.DraftID) == "" {
		return fmt.Errorf("draft_id 不能为空")
	}
	if len(r.Items) == 0 {
		return fmt.Errorf("items 不能为空")
	}
	seen := make(map[string]struct{}, len(r.Items))
	for _, item := range r.Items {
		id := strings.TrimSpace(item.GeneratedProblemID)
		if id == "" {
			return fmt.Errorf("generated_problem_id 不能为空")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("generated_problem_id 重复: %s", id)
		}
		seen[id] = struct{}{}
		if !item.IsPublic && len(item.CourseIDs) == 0 {
			return fmt.Errorf("题目 %s 不能同时不公开且不挂载课程", id)
		}
	}
	return nil
}

// AIGeneratedProblemSaveResponse AI 生成题目确认保存响应。
type AIGeneratedProblemSaveResponse struct {
	DraftID        string                         `json:"draft_id"`
	SavedCount     int                            `json:"saved_count"`
	DiscardedCount int                            `json:"discarded_count"`
	FailedCount    int                            `json:"failed_count"`
	Results        []AIGeneratedProblemSaveResult `json:"results"`
}

// AIGeneratedProblemSaveResult 单道题保存结果。
type AIGeneratedProblemSaveResult struct {
	GeneratedProblemID string `json:"generated_problem_id"`
	ProblemID          string `json:"problem_id,omitempty"`
	UniqueID           int64  `json:"unique_id,omitempty"`
	Success            bool   `json:"success"`
	Error              string `json:"error,omitempty"`
}
