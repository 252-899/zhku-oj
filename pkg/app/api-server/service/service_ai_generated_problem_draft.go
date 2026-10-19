/*
@Author: Codex
@Date: 2026/4/19
@Name: service_ai_generated_problem_draft.go
@Description: AI 生成题目草稿服务，负责生成结果暂存和教师确认保存入正式题库
*/

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zk-code-arena-server/pkg/ai/modules/ojquestiongen"
	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// aiGeneratedProblemDraftTTL AI 生成题目草稿默认保留时间。
const aiGeneratedProblemDraftTTL = 24 * time.Hour

// AIGeneratedProblemDraftService AI 生成题目草稿服务。
type AIGeneratedProblemDraftService struct {
	repo            *repository.AIGeneratedProblemDraftRepository
	problemService  *ProblemService
	testCaseService *TestCaseService
	courseService   *CourseService
}

// NewAIGeneratedProblemDraftService 创建 AI 生成题目草稿服务。
func NewAIGeneratedProblemDraftService(
	repo *repository.AIGeneratedProblemDraftRepository,
	problemService *ProblemService,
	testCaseService *TestCaseService,
	courseService *CourseService,
) *AIGeneratedProblemDraftService {
	return &AIGeneratedProblemDraftService{
		repo:            repo,
		problemService:  problemService,
		testCaseService: testCaseService,
		courseService:   courseService,
	}
}

// CreateDraftFromGeneratedProblems 将 AI 生成结果保存为待确认草稿。
func (s *AIGeneratedProblemDraftService) CreateDraftFromGeneratedProblems(
	ctx context.Context,
	sessionID string,
	roundID string,
	userID primitive.ObjectID,
	problems []*ojquestiongen.GeneratedProblem,
) (*models.AIGeneratedProblemDraft, error) {
	if len(problems) == 0 {
		return nil, fmt.Errorf("生成题目为空，无法创建草稿")
	}
	now := time.Now()
	draft := &models.AIGeneratedProblemDraft{
		ID:         primitive.NewObjectID(),
		SessionID:  strings.TrimSpace(sessionID),
		RoundID:    strings.TrimSpace(roundID),
		UserID:     userID,
		ModuleType: ojquestiongen.CanonicalModuleType,
		Status:     models.AIGeneratedProblemDraftPending,
		Problems:   make([]models.AIGeneratedDraftProblem, 0, len(problems)),
		CreatedAt:  now,
		UpdatedAt:  now,
		ExpiresAt:  now.Add(aiGeneratedProblemDraftTTL),
	}

	for i, problem := range problems {
		if problem == nil {
			continue
		}
		draftProblem := convertGeneratedProblemToDraft(problem)
		if strings.TrimSpace(draftProblem.GeneratedProblemID) == "" {
			draftProblem.GeneratedProblemID = fmt.Sprintf("gen_%d", i+1)
		}
		draft.Problems = append(draft.Problems, draftProblem)
	}
	if len(draft.Problems) == 0 {
		return nil, fmt.Errorf("生成题目为空，无法创建草稿")
	}

	if err := s.repo.Create(ctx, draft); err != nil {
		return nil, err
	}
	return draft, nil
}

// SaveDraftProblems 将教师选中的 AI 生成题保存到正式题库。
func (s *AIGeneratedProblemDraftService) SaveDraftProblems(
	ctx context.Context,
	req *models.AIGeneratedProblemSaveRequest,
	userID primitive.ObjectID,
	role models.UserRole,
) (*models.AIGeneratedProblemSaveResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}
	draftID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.DraftID))
	if err != nil {
		return nil, fmt.Errorf("无效的草稿ID")
	}
	draft, err := s.repo.GetByID(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("AI生成题目草稿不存在: %w", err)
	}
	if role != models.RoleAdmin && draft.UserID != userID {
		return nil, fmt.Errorf("无权保存该AI生成题目草稿")
	}
	if !draft.ExpiresAt.IsZero() && time.Now().After(draft.ExpiresAt) {
		return nil, fmt.Errorf("AI生成题目草稿已过期")
	}

	response := &models.AIGeneratedProblemSaveResponse{
		DraftID: draft.ID.Hex(),
		Results: make([]models.AIGeneratedProblemSaveResult, 0, len(req.Items)),
	}
	for _, item := range req.Items {
		result := s.saveOneDraftProblem(ctx, draft, item, userID, role)
		if result.Success {
			response.SavedCount++
		} else {
			response.FailedCount++
		}
		response.Results = append(response.Results, result)
	}
	response.DiscardedCount = countUnselectedDraftProblems(draft, req.Items)

	if err := s.repo.UpdateAfterSave(ctx, draft, response.Results); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *AIGeneratedProblemDraftService) saveOneDraftProblem(
	ctx context.Context,
	draft *models.AIGeneratedProblemDraft,
	item models.AIGeneratedProblemSaveItem,
	userID primitive.ObjectID,
	role models.UserRole,
) models.AIGeneratedProblemSaveResult {
	// 单题保存要先做草稿归属、课程权限和测试用例完整性校验，避免把 AI 半成品写入正式题库。
	result := models.AIGeneratedProblemSaveResult{GeneratedProblemID: strings.TrimSpace(item.GeneratedProblemID)}
	draftProblem, ok := draft.FindProblem(item.GeneratedProblemID)
	if !ok {
		result.Error = "生成题目不存在"
		return result
	}
	courseIDs, err := parseSaveCourseIDs(item.CourseIDs)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if err := s.ensureCanManageCourses(ctx, courseIDs, userID, role); err != nil {
		result.Error = err.Error()
		return result
	}
	if len(draftProblem.TestCases) == 0 {
		result.Error = "生成题目缺少测试用例"
		return result
	}

	problem := buildProblemFromDraftProblem(draftProblem, item.IsPublic, courseIDs, userID)
	if err := s.problemService.CreateProblem(ctx, problem); err != nil {
		result.Error = err.Error()
		return result
	}

	testCases := buildTestCasesFromDraftProblem(draftProblem, problem.ID)
	for _, testCase := range testCases {
		if err := s.testCaseService.CreateTestCase(ctx, testCase); err != nil {
			// 正式题和测试用例要么一起保存成功，要么尽量回滚，避免题库里出现AI半成品。
			_ = s.testCaseService.DeleteByProblemID(ctx, problem.ID)
			_ = s.problemService.DeleteProblem(ctx, problem.ID)
			result.Error = "题目已创建，但测试用例保存失败: " + err.Error()
			return result
		}
	}

	result.Success = true
	result.ProblemID = problem.ID.Hex()
	result.UniqueID = problem.UniqueID
	return result
}

func convertGeneratedProblemToDraft(problem *ojquestiongen.GeneratedProblem) models.AIGeneratedDraftProblem {
	testCases := make([]models.AIGeneratedDraftTestCase, 0, len(problem.TestCases))
	for _, testCase := range problem.TestCases {
		testCases = append(testCases, models.AIGeneratedDraftTestCase{
			Input:          testCase.Input,
			ExpectedOutput: testCase.ExpectedOutput,
			IsSample:       testCase.IsSample,
		})
	}

	var verification *models.AIGeneratedDraftVerification
	if problem.Verification != nil {
		verification = &models.AIGeneratedDraftVerification{
			Passed:     problem.Verification.Passed,
			Message:    problem.Verification.Message,
			FailedCase: problem.Verification.FailedCase,
		}
	}

	return models.AIGeneratedDraftProblem{
		GeneratedProblemID: strings.TrimSpace(problem.ID),
		Title:              strings.TrimSpace(problem.Title),
		Description:        strings.TrimSpace(problem.Description),
		InputFormat:        strings.TrimSpace(problem.InputFormat),
		OutputFormat:       strings.TrimSpace(problem.OutputFormat),
		Hint:               strings.TrimSpace(problem.Hint),
		Difficulty:         normalizeGeneratedDifficulty(problem.Difficulty),
		StandardCode:       problem.StandardCode,
		StandardLanguage:   strings.TrimSpace(problem.Language),
		TestCases:          testCases,
		Verification:       verification,
		Status:             models.AIGeneratedDraftProblemGenerated,
	}
}

func buildProblemFromDraftProblem(
	draftProblem *models.AIGeneratedDraftProblem,
	isPublic bool,
	courseIDs []primitive.ObjectID,
	userID primitive.ObjectID,
) *models.Problem {
	problem := &models.Problem{
		Title:            draftProblem.Title,
		Description:      draftProblem.Description,
		Input:            draftProblem.InputFormat,
		Output:           draftProblem.OutputFormat,
		Hint:             draftProblem.Hint,
		Difficulty:       draftProblem.Difficulty,
		Status:           models.StatusPublished,
		IsPublic:         isPublic,
		CourseIDs:        courseIDs,
		Source:           "AI生成",
		CreatedBy:        userID,
		StandardCode:     draftProblem.StandardCode,
		StandardLanguage: draftProblem.StandardLanguage,
	}
	if sample := firstSampleDraftTestCase(draftProblem); sample != nil {
		problem.SampleInput = sample.Input
		problem.SampleOutput = sample.ExpectedOutput
	}
	return problem
}

func buildTestCasesFromDraftProblem(draftProblem *models.AIGeneratedDraftProblem, problemID primitive.ObjectID) []*models.TestCase {
	testCases := make([]*models.TestCase, 0, len(draftProblem.TestCases))
	for _, testCase := range draftProblem.TestCases {
		testCases = append(testCases, &models.TestCase{
			ProblemID: problemID,
			Input:     testCase.Input,
			Output:    testCase.ExpectedOutput,
			IsSample:  testCase.IsSample,
		})
	}
	return testCases
}

func firstSampleDraftTestCase(draftProblem *models.AIGeneratedDraftProblem) *models.AIGeneratedDraftTestCase {
	if draftProblem == nil {
		return nil
	}
	for i := range draftProblem.TestCases {
		if draftProblem.TestCases[i].IsSample {
			return &draftProblem.TestCases[i]
		}
	}
	if len(draftProblem.TestCases) > 0 {
		return &draftProblem.TestCases[0]
	}
	return nil
}

func normalizeGeneratedDifficulty(value string) models.ProblemDifficulty {
	switch models.ProblemDifficulty(strings.ToLower(strings.TrimSpace(value))) {
	case models.DifficultyEasy:
		return models.DifficultyEasy
	case models.DifficultyHard:
		return models.DifficultyHard
	default:
		return models.DifficultyMedium
	}
}

func parseSaveCourseIDs(rawIDs []string) ([]primitive.ObjectID, error) {
	courseIDs := make([]primitive.ObjectID, 0, len(rawIDs))
	seen := make(map[primitive.ObjectID]struct{}, len(rawIDs))
	for _, rawID := range rawIDs {
		rawID = strings.TrimSpace(rawID)
		if rawID == "" {
			continue
		}
		courseID, err := primitive.ObjectIDFromHex(rawID)
		if err != nil {
			return nil, fmt.Errorf("无效的课程ID: %s", rawID)
		}
		if _, ok := seen[courseID]; ok {
			continue
		}
		seen[courseID] = struct{}{}
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, nil
}

func (s *AIGeneratedProblemDraftService) ensureCanManageCourses(ctx context.Context, courseIDs []primitive.ObjectID, userID primitive.ObjectID, role models.UserRole) error {
	if role == models.RoleAdmin {
		return nil
	}
	for _, courseID := range courseIDs {
		ok, err := s.courseService.CanManageCourse(ctx, courseID, userID)
		if err != nil {
			return fmt.Errorf("课程不存在或无权操作: %s", courseID.Hex())
		}
		if !ok {
			return fmt.Errorf("无权将题目挂载到课程: %s", courseID.Hex())
		}
	}
	return nil
}

func countUnselectedDraftProblems(draft *models.AIGeneratedProblemDraft, items []models.AIGeneratedProblemSaveItem) int {
	selected := make(map[string]struct{}, len(items))
	for _, item := range items {
		selected[strings.TrimSpace(item.GeneratedProblemID)] = struct{}{}
	}
	count := 0
	for _, problem := range draft.Problems {
		if _, ok := selected[problem.GeneratedProblemID]; !ok {
			count++
		}
	}
	return count
}
