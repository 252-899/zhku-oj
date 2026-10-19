/*
@Author: Codex
@Date: 2026/05/08
@Description: 混合题型提交服务骨架，负责最小作用域 key 与自动判题 helper 入口
*/

package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ContentSubmissionService 统一内容提交服务。
// 当前阶段仅提供独立作答/任务作答的作用域标识与最小自动判题 helper，避免提前绑定 HTTP 语义。
type ContentSubmissionService struct {
	repo                *repository.ContentSubmissionRepository
	choiceRepo          *repository.ChoiceQuestionRepository
	blankRepo           *repository.BlankQuestionRepository
	contentRelationRepo *repository.ContentRelationRepository
	courseService       *CourseService
}

// NewContentSubmissionService 创建统一内容提交服务实例。
func NewContentSubmissionService(
	repo *repository.ContentSubmissionRepository,
	choiceRepo *repository.ChoiceQuestionRepository,
	blankRepo *repository.BlankQuestionRepository,
	contentRelationRepo *repository.ContentRelationRepository,
	courseService *CourseService,
) *ContentSubmissionService {
	return &ContentSubmissionService{
		repo:                repo,
		choiceRepo:          choiceRepo,
		blankRepo:           blankRepo,
		contentRelationRepo: contentRelationRepo,
		courseService:       courseService,
	}
}

// buildSubmissionScopeKey 构建提交作用域键。
// 统一键格式可以让后续去重、查询和幂等更新共享同一套上下文表达，而不是散落多种拼接方式。
func buildSubmissionScopeKey(mode, taskID, relationID string) string {
	return fmt.Sprintf("%s:%s:%s", mode, taskID, relationID)
}

// judgeChoiceQuestion 对选择题做最小自动判题。
// 当前只比较答案集合是否完全一致；更复杂的历史记录与部分得分逻辑留给后续任务扩展。
func (s *ContentSubmissionService) judgeChoiceQuestion(question *models.ChoiceQuestion, answers []string) (score int, isCorrect bool, err error) {
	if question == nil {
		return 0, false, fmt.Errorf("选择题不能为空")
	}
	expected := append([]string(nil), question.CorrectAnswers...)
	actual := append([]string(nil), answers...)
	slices.Sort(expected)
	slices.Sort(actual)
	if slices.Equal(expected, actual) {
		return question.Score, true, nil
	}
	return 0, false, nil
}

// judgeBlankQuestion 对填空题做最小自动判题。
// 这里只覆盖 exact_match 的骨架能力；manual 模式显式返回未自动判分，避免提前承诺完整评分语义。
func (s *ContentSubmissionService) judgeBlankQuestion(question *models.BlankQuestion, answers map[int]string) (score int, isCorrect bool, err error) {
	if question == nil {
		return 0, false, fmt.Errorf("填空题不能为空")
	}
	if question.JudgeMode == models.BlankJudgeModeManual {
		return 0, false, nil
	}
	// 精确判分要求提交空位和题目定义一一对应，多余空位也视为非法作答。
	if len(answers) != len(question.Blanks) {
		return 0, false, nil
	}
	for _, blank := range question.Blanks {
		answer, ok := answers[blank.Order]
		if !ok {
			return 0, false, nil
		}
		matched := false
		for _, expected := range blank.Answers {
			if answer == expected {
				matched = true
				break
			}
		}
		if !matched {
			return 0, false, nil
		}
	}
	return question.Score, true, nil
}

// SubmitStandaloneAnswer 提交单题独立作答记录。
// 这里显式不复用任务提交流程，确保同一道题未来挂到多个任务时，独立作答记录不会和任务内记录串在一起。
func (s *ContentSubmissionService) SubmitStandaloneAnswer(
	ctx context.Context,
	userID primitive.ObjectID,
	req *models.SubmitContentAnswerRequest,
) (*models.ContentSubmission, error) {
	if req == nil {
		return nil, fmt.Errorf("内容提交请求不能为空")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	contentID, err := primitive.ObjectIDFromHex(req.ContentID)
	if err != nil {
		return nil, fmt.Errorf("无效的内容ID")
	}
	courseID := primitive.NilObjectID
	if req.CourseID != "" {
		courseID, err = primitive.ObjectIDFromHex(req.CourseID)
		if err != nil {
			return nil, fmt.Errorf("无效的课程ID")
		}
	}

	submission := &models.ContentSubmission{
		ID:             primitive.NewObjectID(),
		UserID:         userID,
		ContentType:    req.ContentType,
		ContentID:      contentID,
		SubmissionMode: models.SubmissionModeStandalone,
		CourseID:       courseID,
		Answers:        buildSubmissionAnswers(req),
		SubmittedAt:    time.Now(),
	}

	switch req.ContentType {
	case models.ContentTypeChoice:
		question, err := s.choiceRepo.GetByID(ctx, contentID)
		if err != nil {
			return nil, err
		}
		if err := s.ensureChoiceQuestionSubmittable(ctx, question, userID, courseID); err != nil {
			return nil, err
		}
		score, isCorrect, err := s.judgeChoiceQuestion(question, req.ChoiceAnswers)
		if err != nil {
			return nil, err
		}
		submission.Score = score
		submission.IsCorrect = isCorrect
		submission.JudgeStatus = "judged"
	case models.ContentTypeBlank:
		question, err := s.blankRepo.GetByID(ctx, contentID)
		if err != nil {
			return nil, err
		}
		if err := s.ensureBlankQuestionSubmittable(ctx, question, userID, courseID); err != nil {
			return nil, err
		}
		score, isCorrect, judgeStatus, err := s.judgeBlankSubmission(question, req.BlankAnswers)
		if err != nil {
			return nil, err
		}
		submission.Score = score
		submission.IsCorrect = isCorrect
		submission.JudgeStatus = judgeStatus
	default:
		return nil, fmt.Errorf("暂不支持的内容类型")
	}

	if err := submission.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, err
	}
	return submission, nil
}

// ensureChoiceQuestionSubmittable 校验选择题当前是否允许独立提交。
func (s *ContentSubmissionService) ensureChoiceQuestionSubmittable(
	ctx context.Context,
	question *models.ChoiceQuestion,
	userID primitive.ObjectID,
	courseID primitive.ObjectID,
) error {
	if question == nil {
		return fmt.Errorf("选择题不存在")
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("当前选择题未开放提交")
	}
	return s.ensureQuestionPublishScope(ctx, question.PublishMode, userID, courseID, models.ContentTypeChoice, question.ID)
}

// ensureBlankQuestionSubmittable 校验填空题当前是否允许独立提交。
func (s *ContentSubmissionService) ensureBlankQuestionSubmittable(
	ctx context.Context,
	question *models.BlankQuestion,
	userID primitive.ObjectID,
	courseID primitive.ObjectID,
) error {
	if question == nil {
		return fmt.Errorf("填空题不存在")
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("当前填空题未开放提交")
	}
	return s.ensureQuestionPublishScope(ctx, question.PublishMode, userID, courseID, models.ContentTypeBlank, question.ID)
}

// ensureQuestionPublishScope 校验题目发布范围与提交上下文是否匹配。
func (s *ContentSubmissionService) ensureQuestionPublishScope(
	ctx context.Context,
	publishMode models.QuestionPublishMode,
	userID primitive.ObjectID,
	courseID primitive.ObjectID,
	contentType models.ContentType,
	contentID primitive.ObjectID,
) error {
	switch publishMode {
	case models.QuestionPublishModeGlobal, models.QuestionPublishModeGlobalAndCourse:
		if courseID == primitive.NilObjectID {
			return nil
		}
		if err := s.ensureCanViewCourse(ctx, courseID, userID); err != nil {
			return err
		}
		return s.ensureCourseRelationExists(ctx, courseID, contentType, contentID)
	case models.QuestionPublishModeCourseOnly:
		if courseID == primitive.NilObjectID {
			return fmt.Errorf("课程题提交必须携带课程ID")
		}
		if err := s.ensureCanViewCourse(ctx, courseID, userID); err != nil {
			return err
		}
		return s.ensureCourseRelationExists(ctx, courseID, contentType, contentID)
	default:
		return fmt.Errorf("题目发布范围非法")
	}
}

// ensureCanViewCourse 校验用户是否具备课程查看权限。
func (s *ContentSubmissionService) ensureCanViewCourse(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
) error {
	if s.courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	ok, err := s.courseService.CanViewCourse(ctx, courseID, userID)
	if err != nil {
		return fmt.Errorf("课程不存在或无权访问: %s", courseID.Hex())
	}
	if !ok {
		return fmt.Errorf("无权在该课程上下文中提交题目")
	}
	return nil
}

// ensureCourseRelationExists 校验题目确实挂载在指定课程下，避免跨课程伪造提交上下文。
func (s *ContentSubmissionService) ensureCourseRelationExists(
	ctx context.Context,
	courseID primitive.ObjectID,
	contentType models.ContentType,
	contentID primitive.ObjectID,
) error {
	if s.contentRelationRepo == nil {
		return fmt.Errorf("内容关系仓储未初始化")
	}
	exists, err := s.contentRelationRepo.ExistsCourseRelation(ctx, courseID, contentType, contentID)
	if err != nil {
		return fmt.Errorf("校验课程题目关系失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("题目未挂载到当前课程")
	}
	return nil
}

// judgeBlankSubmission 将填空题请求答案转换为判题输入，并返回判题状态。
func (s *ContentSubmissionService) judgeBlankSubmission(
	question *models.BlankQuestion,
	answers []models.BlankAnswerItem,
) (score int, isCorrect bool, judgeStatus string, err error) {
	answerMap := make(map[int]string, len(answers))
	for _, answer := range answers {
		answerMap[answer.Order] = answer.Answer
	}
	score, isCorrect, err = s.judgeBlankQuestion(question, answerMap)
	if err != nil {
		return 0, false, "", err
	}
	if question != nil && question.JudgeMode == models.BlankJudgeModeManual {
		return 0, false, "pending_manual", nil
	}
	return score, isCorrect, "judged", nil
}

// buildSubmissionAnswers 统一构造提交入库的答案结构。
func buildSubmissionAnswers(req *models.SubmitContentAnswerRequest) map[string]any {
	answers := make(map[string]any)
	if req == nil {
		return answers
	}
	switch req.ContentType {
	case models.ContentTypeChoice:
		answers["choice_answers"] = req.ChoiceAnswers
	case models.ContentTypeBlank:
		blankAnswers := make([]map[string]any, 0, len(req.BlankAnswers))
		for _, answer := range req.BlankAnswers {
			blankAnswers = append(blankAnswers, map[string]any{
				"order":  answer.Order,
				"answer": answer.Answer,
			})
		}
		answers["blank_answers"] = blankAnswers
	}
	return answers
}
