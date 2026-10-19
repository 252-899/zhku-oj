/*
@Author: Codex
@Date: 2026/05/08
@Description: 选择题服务骨架，负责最小创建预处理与后续课程挂载编排入口
*/

package service

import (
	"context"
	"fmt"
	"time"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ChoiceQuestionService 选择题服务。
// 当前阶段只补齐 service 层最小骨架，避免在 server 层接入前就固化完整接口语义。
type ChoiceQuestionService struct {
	repo                *repository.ChoiceQuestionRepository
	contentRelationRepo *repository.ContentRelationRepository
	courseService       *CourseService
}

// NewChoiceQuestionService 创建选择题服务实例。
func NewChoiceQuestionService(
	repo *repository.ChoiceQuestionRepository,
	contentRelationRepo *repository.ContentRelationRepository,
	courseService *CourseService,
) *ChoiceQuestionService {
	return &ChoiceQuestionService{
		repo:                repo,
		contentRelationRepo: contentRelationRepo,
		courseService:       courseService,
	}
}

// prepareForCreate 补齐选择题创建前的最小系统字段。
// 这里只处理 service 层稳定的默认值与结构校验，避免提前掺入 HTTP 请求细节。
func (s *ChoiceQuestionService) prepareForCreate(ctx context.Context, q *models.ChoiceQuestion) error {
	_ = ctx
	if q == nil {
		return fmt.Errorf("选择题不能为空")
	}
	if q.Status == "" {
		// 新题型默认进入已发布态，草稿态仍允许调用方显式传入以保留后续编辑流程。
		q.Status = models.QuestionStatusPublished
	}
	if q.Tags == nil {
		q.Tags = []string{}
	}
	now := time.Now()
	if q.ID == primitive.NilObjectID {
		q.ID = primitive.NewObjectID()
	}
	if q.CreatedAt.IsZero() {
		q.CreatedAt = now
	}
	q.UpdatedAt = now
	return q.Validate()
}

// CreateCourseChoiceQuestion 在课程上下文中创建选择题。
// 当前阶段采用“课程入口创建，发布模式决定是否建立课程挂载”的策略，避免过早引入独立全局创建入口。
func (s *ChoiceQuestionService) CreateCourseChoiceQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.CreateChoiceQuestionRequest,
) (*models.ChoiceQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("创建选择题请求不能为空")
	}
	if err := s.ensureCanCreateInCourse(ctx, courseID, userID, role); err != nil {
		return nil, err
	}

	courseIDs, err := mergeCourseScope(courseID, req.PublishMode, req.CourseIDs)
	if err != nil {
		return nil, err
	}
	if err := ensureCanManageTargetCourses(ctx, s.courseService, courseIDs, userID, role); err != nil {
		return nil, err
	}
	if err := models.ValidatePublishModeCourseIDs(req.PublishMode, objectIDsToHex(courseIDs)); err != nil {
		return nil, err
	}
	if err := (&models.ChoiceQuestion{
		ChoiceMode:     req.ChoiceMode,
		Options:        req.Options,
		CorrectAnswers: req.Answers,
	}).Validate(); err != nil {
		return nil, err
	}

	question := &models.ChoiceQuestion{
		Title:          req.Title,
		Stem:           req.Stem,
		Difficulty:     req.Difficulty,
		Tags:           req.Tags,
		PublishMode:    req.PublishMode,
		ChoiceMode:     req.ChoiceMode,
		Options:        req.Options,
		CorrectAnswers: req.Answers,
		Analysis:       req.Analysis,
		Score:          req.Score,
		CreatedBy:      userID,
	}
	if req.Status != nil {
		question.Status = *req.Status
	}
	if err := s.prepareForCreate(ctx, question); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, question); err != nil {
		return nil, err
	}
	if err := s.attachCourseRelations(ctx, courseIDs, question.ID, userID); err != nil {
		// 课程挂载失败时补偿删除刚创建的题目，避免留下发布范围和实际可见性不一致的孤儿题。
		if cleanupErr := s.repo.Delete(ctx, question.ID); cleanupErr != nil {
			return nil, fmt.Errorf("创建选择题成功但课程挂载失败，且补偿删除失败: %v; 原始错误: %w", cleanupErr, err)
		}
		return nil, err
	}
	return question, nil
}

// CreateChoiceQuestion 创建通用选择题。
// 该入口不再依赖单一课程上下文，发布范围与课程挂载完全由请求体决定。
func (s *ChoiceQuestionService) CreateChoiceQuestion(
	ctx context.Context,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.CreateChoiceQuestionRequest,
) (*models.ChoiceQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("创建选择题请求不能为空")
	}
	if role != models.RoleAdmin && role != models.RoleTeacher {
		return nil, fmt.Errorf("权限不足，只有教师和管理员可以操作")
	}

	courseIDs, err := parseQuestionScopeCourseIDs(req.PublishMode, req.CourseIDs)
	if err != nil {
		return nil, err
	}
	if err := ensureCanManageTargetCourses(ctx, s.courseService, courseIDs, userID, role); err != nil {
		return nil, err
	}
	if err := models.ValidatePublishModeCourseIDs(req.PublishMode, objectIDsToHex(courseIDs)); err != nil {
		return nil, err
	}
	if err := (&models.ChoiceQuestion{
		ChoiceMode:     req.ChoiceMode,
		Options:        req.Options,
		CorrectAnswers: req.Answers,
	}).Validate(); err != nil {
		return nil, err
	}

	question := &models.ChoiceQuestion{
		Title:          req.Title,
		Stem:           req.Stem,
		Difficulty:     req.Difficulty,
		Tags:           req.Tags,
		PublishMode:    req.PublishMode,
		ChoiceMode:     req.ChoiceMode,
		Options:        req.Options,
		CorrectAnswers: req.Answers,
		Analysis:       req.Analysis,
		Score:          req.Score,
		CreatedBy:      userID,
	}
	if req.Status != nil {
		question.Status = *req.Status
	}
	if err := s.prepareForCreate(ctx, question); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, question); err != nil {
		return nil, err
	}
	if err := s.attachCourseRelations(ctx, courseIDs, question.ID, userID); err != nil {
		if cleanupErr := s.repo.Delete(ctx, question.ID); cleanupErr != nil {
			return nil, fmt.Errorf("创建选择题成功但课程挂载失败，且补偿删除失败: %v; 原始错误: %w", cleanupErr, err)
		}
		return nil, err
	}
	return question, nil
}

// GetCourseChoiceQuestion 获取课程上下文中的选择题详情。
func (s *ChoiceQuestionService) GetCourseChoiceQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) (*models.ChoiceQuestion, error) {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCourseQuestionVisible(ctx, courseID, question, userID, role); err != nil {
		return nil, err
	}
	return question, nil
}

// GetChoiceQuestion 获取通用选择题详情。
func (s *ChoiceQuestionService) GetChoiceQuestion(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) (*models.ChoiceQuestion, error) {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureQuestionVisible(ctx, question, userID, role); err != nil {
		return nil, err
	}
	return question, nil
}

// UpdateCourseChoiceQuestion 更新课程上下文中的选择题。
func (s *ChoiceQuestionService) UpdateCourseChoiceQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateChoiceQuestionRequest,
) (*models.ChoiceQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("更新选择题请求不能为空")
	}
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCanModifyQuestion(question.CreatedBy, userID, role); err != nil {
		return nil, err
	}
	if err := s.ensureQuestionLinkedToCourse(ctx, courseID, question.ID); err != nil {
		return nil, err
	}

	targetPublishMode, targetCourseIDs, err := s.resolveChoiceUpdateScope(ctx, courseID, question, userID, role, req)
	if err != nil {
		return nil, err
	}
	s.applyChoiceUpdates(question, req, targetPublishMode)
	question.UpdatedAt = time.Now()
	if err := question.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Replace(ctx, question); err != nil {
		return nil, err
	}
	if err := s.syncCourseRelations(ctx, question.ID, userID, targetPublishMode, targetCourseIDs); err != nil {
		return nil, err
	}
	return question, nil
}

// UpdateChoiceQuestion 更新通用选择题。
func (s *ChoiceQuestionService) UpdateChoiceQuestion(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateChoiceQuestionRequest,
) (*models.ChoiceQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("更新选择题请求不能为空")
	}
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCanModifyQuestion(question.CreatedBy, userID, role); err != nil {
		return nil, err
	}

	targetPublishMode, targetCourseIDs, err := s.resolveGenericChoiceUpdateScope(ctx, question, userID, role, req)
	if err != nil {
		return nil, err
	}
	s.applyChoiceUpdates(question, req, targetPublishMode)
	question.UpdatedAt = time.Now()
	if err := question.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Replace(ctx, question); err != nil {
		return nil, err
	}
	if err := s.syncCourseRelations(ctx, question.ID, userID, targetPublishMode, targetCourseIDs); err != nil {
		return nil, err
	}
	return question, nil
}

// DeleteCourseChoiceQuestion 删除课程上下文中的选择题。
func (s *ChoiceQuestionService) DeleteCourseChoiceQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if err := s.ensureCanModifyQuestion(question.CreatedBy, userID, role); err != nil {
		return err
	}
	if err := s.ensureQuestionLinkedToCourse(ctx, courseID, question.ID); err != nil {
		return err
	}
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeChoice, question.ID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, question.ID)
}

// DeleteChoiceQuestion 删除通用选择题。
func (s *ChoiceQuestionService) DeleteChoiceQuestion(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if err := s.ensureCanModifyQuestion(question.CreatedBy, userID, role); err != nil {
		return err
	}
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeChoice, question.ID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, question.ID)
}

// ensureCanCreateInCourse 校验课程上下文下的创建权限。
func (s *ChoiceQuestionService) ensureCanCreateInCourse(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if role == models.RoleAdmin {
		return nil
	}
	if s.courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	ok, err := s.courseService.CanManageCourse(ctx, courseID, userID)
	if err != nil {
		return fmt.Errorf("课程不存在或无权操作: %s", courseID.Hex())
	}
	if !ok {
		return fmt.Errorf("无权在该课程下创建选择题")
	}
	return nil
}

// attachCourseRelations 根据发布范围补齐课程挂载关系。
func (s *ChoiceQuestionService) attachCourseRelations(
	ctx context.Context,
	courseIDs []primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
) error {
	if len(courseIDs) == 0 {
		return nil
	}
	relations := make([]*models.CourseContentRelation, 0, len(courseIDs))
	for _, courseID := range courseIDs {
		relations = append(relations, &models.CourseContentRelation{
			CourseID:    courseID,
			ContentType: models.ContentTypeChoice,
			ContentID:   questionID,
			CreatedBy:   userID,
		})
	}
	return s.contentRelationRepo.CreateCourseRelations(ctx, relations)
}

// ensureCourseQuestionVisible 校验课程上下文下的选择题查看权限。
func (s *ChoiceQuestionService) ensureCourseQuestionVisible(
	ctx context.Context,
	courseID primitive.ObjectID,
	question *models.ChoiceQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if question == nil {
		return fmt.Errorf("选择题不存在")
	}
	if err := s.ensureQuestionLinkedToCourse(ctx, courseID, question.ID); err != nil {
		return err
	}
	if role == models.RoleAdmin || question.CreatedBy == userID {
		return nil
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("无权查看未发布选择题")
	}
	if s.courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	ok, err := s.courseService.CanViewCourse(ctx, courseID, userID)
	if err != nil {
		return fmt.Errorf("课程不存在或无权访问: %s", courseID.Hex())
	}
	if !ok {
		return fmt.Errorf("无权查看该课程选择题")
	}
	return nil
}

// ensureQuestionVisible 校验通用选择题查看权限。
// 全局题允许已登录用户读取已发布版本；课程题则要求用户至少能访问一个挂载课程。
func (s *ChoiceQuestionService) ensureQuestionVisible(
	ctx context.Context,
	question *models.ChoiceQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if question == nil {
		return fmt.Errorf("选择题不存在")
	}
	if role == models.RoleAdmin || question.CreatedBy == userID {
		return nil
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("无权查看未发布选择题")
	}
	if question.PublishMode == models.QuestionPublishModeGlobal || question.PublishMode == models.QuestionPublishModeGlobalAndCourse {
		return nil
	}
	courseIDs, err := s.listLinkedCourseIDs(ctx, question.ID)
	if err != nil {
		return err
	}
	if len(courseIDs) == 0 {
		return fmt.Errorf("选择题未挂载到任何课程")
	}
	return ensureCanViewAnyCourse(ctx, s.courseService, courseIDs, userID)
}

// ensureCanModifyQuestion 校验是否具备题目修改权限。
func (s *ChoiceQuestionService) ensureCanModifyQuestion(createdBy, userID primitive.ObjectID, role models.UserRole) error {
	if role == models.RoleAdmin {
		return nil
	}
	if createdBy != userID {
		return fmt.Errorf("只有题目创建者才能修改该选择题")
	}
	return nil
}

// ensureQuestionLinkedToCourse 校验题目是否挂载在当前课程下。
func (s *ChoiceQuestionService) ensureQuestionLinkedToCourse(ctx context.Context, courseID, questionID primitive.ObjectID) error {
	exists, err := s.contentRelationRepo.ExistsCourseRelation(ctx, courseID, models.ContentTypeChoice, questionID)
	if err != nil {
		return fmt.Errorf("校验课程选择题关系失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("选择题未挂载到当前课程")
	}
	return nil
}

// resolveChoiceUpdateScope 解析更新后的发布范围和课程挂载范围。
func (s *ChoiceQuestionService) resolveChoiceUpdateScope(
	ctx context.Context,
	courseID primitive.ObjectID,
	question *models.ChoiceQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateChoiceQuestionRequest,
) (models.QuestionPublishMode, []primitive.ObjectID, error) {
	publishMode := question.PublishMode
	if req.PublishMode != nil {
		publishMode = *req.PublishMode
	}
	rawCourseIDs := []string{}
	if req.CourseIDs != nil {
		rawCourseIDs = *req.CourseIDs
	}
	targetCourseIDs, err := mergeCourseScope(courseID, publishMode, rawCourseIDs)
	if err != nil {
		return "", nil, err
	}
	if publishMode == models.QuestionPublishModeGlobal {
		targetCourseIDs = []primitive.ObjectID{}
	}
	if err := ensureCanManageTargetCourses(ctx, s.courseService, targetCourseIDs, userID, role); err != nil {
		return "", nil, err
	}
	if err := models.ValidatePublishModeCourseIDs(publishMode, objectIDsToHex(targetCourseIDs)); err != nil {
		return "", nil, err
	}
	return publishMode, targetCourseIDs, nil
}

// resolveGenericChoiceUpdateScope 解析通用更新入口下的发布范围和课程挂载范围。
func (s *ChoiceQuestionService) resolveGenericChoiceUpdateScope(
	ctx context.Context,
	question *models.ChoiceQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateChoiceQuestionRequest,
) (models.QuestionPublishMode, []primitive.ObjectID, error) {
	publishMode := question.PublishMode
	if req.PublishMode != nil {
		publishMode = *req.PublishMode
	}
	targetCourseIDs, err := s.resolveQuestionCourseIDsForUpdate(ctx, question.ID, publishMode, req.CourseIDs)
	if err != nil {
		return "", nil, err
	}
	if err := ensureCanManageTargetCourses(ctx, s.courseService, targetCourseIDs, userID, role); err != nil {
		return "", nil, err
	}
	if err := models.ValidatePublishModeCourseIDs(publishMode, objectIDsToHex(targetCourseIDs)); err != nil {
		return "", nil, err
	}
	return publishMode, targetCourseIDs, nil
}

// applyChoiceUpdates 将更新请求写回实体。
func (s *ChoiceQuestionService) applyChoiceUpdates(
	question *models.ChoiceQuestion,
	req *models.UpdateChoiceQuestionRequest,
	publishMode models.QuestionPublishMode,
) {
	if req.Title != nil {
		question.Title = *req.Title
	}
	if req.Stem != nil {
		question.Stem = *req.Stem
	}
	if req.Difficulty != nil {
		question.Difficulty = *req.Difficulty
	}
	if req.Tags != nil {
		question.Tags = *req.Tags
	}
	if req.Status != nil {
		question.Status = *req.Status
	}
	question.PublishMode = publishMode
	if req.ChoiceMode != nil {
		question.ChoiceMode = *req.ChoiceMode
	}
	if req.Options != nil {
		question.Options = *req.Options
	}
	if req.Answers != nil {
		question.CorrectAnswers = *req.Answers
	}
	if req.Analysis != nil {
		question.Analysis = *req.Analysis
	}
	if req.Score != nil {
		question.Score = *req.Score
	}
}

// syncCourseRelations 同步题目课程挂载关系。
func (s *ChoiceQuestionService) syncCourseRelations(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	publishMode models.QuestionPublishMode,
	courseIDs []primitive.ObjectID,
) error {
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeChoice, questionID); err != nil {
		return err
	}
	if publishMode == models.QuestionPublishModeGlobal || len(courseIDs) == 0 {
		return nil
	}
	return s.attachCourseRelations(ctx, courseIDs, questionID, userID)
}

// resolveQuestionCourseIDsForUpdate 计算更新后的课程挂载范围。
// 通用更新接口需要在“未传 course_ids 时保留原挂载”与“显式传空数组时清空挂载”之间做出区分。
func (s *ChoiceQuestionService) resolveQuestionCourseIDsForUpdate(
	ctx context.Context,
	questionID primitive.ObjectID,
	publishMode models.QuestionPublishMode,
	rawCourseIDs *[]string,
) ([]primitive.ObjectID, error) {
	if publishMode == models.QuestionPublishModeGlobal {
		return []primitive.ObjectID{}, nil
	}
	if rawCourseIDs != nil {
		return parseQuestionScopeCourseIDs(publishMode, *rawCourseIDs)
	}
	return s.listLinkedCourseIDs(ctx, questionID)
}

// listLinkedCourseIDs 查询选择题当前的课程挂载范围。
func (s *ChoiceQuestionService) listLinkedCourseIDs(ctx context.Context, questionID primitive.ObjectID) ([]primitive.ObjectID, error) {
	relations, err := s.contentRelationRepo.ListRelationsByContentIDs(ctx, models.ContentTypeChoice, []primitive.ObjectID{questionID})
	if err != nil {
		return nil, fmt.Errorf("查询选择题课程挂载失败: %w", err)
	}
	courseIDMap := make(map[string]primitive.ObjectID, len(relations))
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		courseIDMap[relation.CourseID.Hex()] = relation.CourseID
	}
	courseIDs := make([]primitive.ObjectID, 0, len(courseIDMap))
	for _, courseID := range courseIDMap {
		courseIDs = append(courseIDs, courseID)
	}
	return courseIDs, nil
}
