/*
@Author: Codex
@Date: 2026/05/08
@Description: 填空题服务骨架，负责最小创建预处理与后续课程挂载编排入口
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

// BlankQuestionService 填空题服务。
// 当前先提供最小依赖注入和预处理逻辑，后续再逐步补全增删改查能力。
type BlankQuestionService struct {
	repo                *repository.BlankQuestionRepository
	contentRelationRepo *repository.ContentRelationRepository
	courseService       *CourseService
}

// NewBlankQuestionService 创建填空题服务实例。
func NewBlankQuestionService(
	repo *repository.BlankQuestionRepository,
	contentRelationRepo *repository.ContentRelationRepository,
	courseService *CourseService,
) *BlankQuestionService {
	return &BlankQuestionService{
		repo:                repo,
		contentRelationRepo: contentRelationRepo,
		courseService:       courseService,
	}
}

// prepareForCreate 补齐填空题创建前的最小系统字段。
// 这里复用与选择题一致的生命周期默认值，确保两类新题型的落库前边界保持一致。
func (s *BlankQuestionService) prepareForCreate(ctx context.Context, q *models.BlankQuestion) error {
	_ = ctx
	if q == nil {
		return fmt.Errorf("填空题不能为空")
	}
	if q.Status == "" {
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

// CreateCourseBlankQuestion 在课程上下文中创建填空题。
func (s *BlankQuestionService) CreateCourseBlankQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.CreateBlankQuestionRequest,
) (*models.BlankQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("创建填空题请求不能为空")
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
	if err := (&models.BlankQuestion{
		JudgeMode: req.JudgeMode,
		Blanks:    req.Blanks,
	}).Validate(); err != nil {
		return nil, err
	}

	question := &models.BlankQuestion{
		Title:       req.Title,
		Stem:        req.Stem,
		Difficulty:  req.Difficulty,
		Tags:        req.Tags,
		PublishMode: req.PublishMode,
		JudgeMode:   req.JudgeMode,
		Blanks:      req.Blanks,
		Analysis:    req.Analysis,
		Score:       req.Score,
		CreatedBy:   userID,
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
		// 课程挂载失败时补偿删除刚创建的题目，避免留下课程范围不完整的孤儿题。
		if cleanupErr := s.repo.Delete(ctx, question.ID); cleanupErr != nil {
			return nil, fmt.Errorf("创建填空题成功但课程挂载失败，且补偿删除失败: %v; 原始错误: %w", cleanupErr, err)
		}
		return nil, err
	}
	return question, nil
}

// CreateBlankQuestion 创建通用填空题。
func (s *BlankQuestionService) CreateBlankQuestion(
	ctx context.Context,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.CreateBlankQuestionRequest,
) (*models.BlankQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("创建填空题请求不能为空")
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
	if err := (&models.BlankQuestion{
		JudgeMode: req.JudgeMode,
		Blanks:    req.Blanks,
	}).Validate(); err != nil {
		return nil, err
	}

	question := &models.BlankQuestion{
		Title:       req.Title,
		Stem:        req.Stem,
		Difficulty:  req.Difficulty,
		Tags:        req.Tags,
		PublishMode: req.PublishMode,
		JudgeMode:   req.JudgeMode,
		Blanks:      req.Blanks,
		Analysis:    req.Analysis,
		Score:       req.Score,
		CreatedBy:   userID,
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
			return nil, fmt.Errorf("创建填空题成功但课程挂载失败，且补偿删除失败: %v; 原始错误: %w", cleanupErr, err)
		}
		return nil, err
	}
	return question, nil
}

// GetCourseBlankQuestion 获取课程上下文中的填空题详情。
func (s *BlankQuestionService) GetCourseBlankQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) (*models.BlankQuestion, error) {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCourseQuestionVisible(ctx, courseID, question, userID, role); err != nil {
		return nil, err
	}
	return question, nil
}

// GetBlankQuestion 获取通用填空题详情。
func (s *BlankQuestionService) GetBlankQuestion(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) (*models.BlankQuestion, error) {
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureQuestionVisible(ctx, question, userID, role); err != nil {
		return nil, err
	}
	return question, nil
}

// UpdateCourseBlankQuestion 更新课程上下文中的填空题。
func (s *BlankQuestionService) UpdateCourseBlankQuestion(
	ctx context.Context,
	courseID primitive.ObjectID,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateBlankQuestionRequest,
) (*models.BlankQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("更新填空题请求不能为空")
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

	targetPublishMode, targetCourseIDs, err := s.resolveBlankUpdateScope(ctx, courseID, question, userID, role, req)
	if err != nil {
		return nil, err
	}
	s.applyBlankUpdates(question, req, targetPublishMode)
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

// UpdateBlankQuestion 更新通用填空题。
func (s *BlankQuestionService) UpdateBlankQuestion(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateBlankQuestionRequest,
) (*models.BlankQuestion, error) {
	if req == nil {
		return nil, fmt.Errorf("更新填空题请求不能为空")
	}
	question, err := s.repo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCanModifyQuestion(question.CreatedBy, userID, role); err != nil {
		return nil, err
	}

	targetPublishMode, targetCourseIDs, err := s.resolveGenericBlankUpdateScope(ctx, question, userID, role, req)
	if err != nil {
		return nil, err
	}
	s.applyBlankUpdates(question, req, targetPublishMode)
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

// DeleteCourseBlankQuestion 删除课程上下文中的填空题。
func (s *BlankQuestionService) DeleteCourseBlankQuestion(
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
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeBlank, question.ID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, question.ID)
}

// DeleteBlankQuestion 删除通用填空题。
func (s *BlankQuestionService) DeleteBlankQuestion(
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
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeBlank, question.ID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, question.ID)
}

// ensureCanCreateInCourse 校验课程上下文下的创建权限。
func (s *BlankQuestionService) ensureCanCreateInCourse(
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
		return fmt.Errorf("无权在该课程下创建填空题")
	}
	return nil
}

// attachCourseRelations 根据发布范围补齐课程挂载关系。
func (s *BlankQuestionService) attachCourseRelations(
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
			ContentType: models.ContentTypeBlank,
			ContentID:   questionID,
			CreatedBy:   userID,
		})
	}
	return s.contentRelationRepo.CreateCourseRelations(ctx, relations)
}

// ensureCourseQuestionVisible 校验课程上下文下的填空题查看权限。
func (s *BlankQuestionService) ensureCourseQuestionVisible(
	ctx context.Context,
	courseID primitive.ObjectID,
	question *models.BlankQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if question == nil {
		return fmt.Errorf("填空题不存在")
	}
	if err := s.ensureQuestionLinkedToCourse(ctx, courseID, question.ID); err != nil {
		return err
	}
	if role == models.RoleAdmin || question.CreatedBy == userID {
		return nil
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("无权查看未发布填空题")
	}
	if s.courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	ok, err := s.courseService.CanViewCourse(ctx, courseID, userID)
	if err != nil {
		return fmt.Errorf("课程不存在或无权访问: %s", courseID.Hex())
	}
	if !ok {
		return fmt.Errorf("无权查看该课程填空题")
	}
	return nil
}

// ensureQuestionVisible 校验通用填空题查看权限。
func (s *BlankQuestionService) ensureQuestionVisible(
	ctx context.Context,
	question *models.BlankQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if question == nil {
		return fmt.Errorf("填空题不存在")
	}
	if role == models.RoleAdmin || question.CreatedBy == userID {
		return nil
	}
	if question.Status != models.QuestionStatusPublished {
		return fmt.Errorf("无权查看未发布填空题")
	}
	if question.PublishMode == models.QuestionPublishModeGlobal || question.PublishMode == models.QuestionPublishModeGlobalAndCourse {
		return nil
	}
	courseIDs, err := s.listLinkedCourseIDs(ctx, question.ID)
	if err != nil {
		return err
	}
	if len(courseIDs) == 0 {
		return fmt.Errorf("填空题未挂载到任何课程")
	}
	return ensureCanViewAnyCourse(ctx, s.courseService, courseIDs, userID)
}

// ensureCanModifyQuestion 校验是否具备题目修改权限。
func (s *BlankQuestionService) ensureCanModifyQuestion(createdBy, userID primitive.ObjectID, role models.UserRole) error {
	if role == models.RoleAdmin {
		return nil
	}
	if createdBy != userID {
		return fmt.Errorf("只有题目创建者才能修改该填空题")
	}
	return nil
}

// ensureQuestionLinkedToCourse 校验题目是否挂载在当前课程下。
func (s *BlankQuestionService) ensureQuestionLinkedToCourse(ctx context.Context, courseID, questionID primitive.ObjectID) error {
	exists, err := s.contentRelationRepo.ExistsCourseRelation(ctx, courseID, models.ContentTypeBlank, questionID)
	if err != nil {
		return fmt.Errorf("校验课程填空题关系失败: %w", err)
	}
	if !exists {
		return fmt.Errorf("填空题未挂载到当前课程")
	}
	return nil
}

// resolveBlankUpdateScope 解析更新后的发布范围和课程挂载范围。
func (s *BlankQuestionService) resolveBlankUpdateScope(
	ctx context.Context,
	courseID primitive.ObjectID,
	question *models.BlankQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateBlankQuestionRequest,
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

// resolveGenericBlankUpdateScope 解析通用更新入口下的发布范围和课程挂载范围。
func (s *BlankQuestionService) resolveGenericBlankUpdateScope(
	ctx context.Context,
	question *models.BlankQuestion,
	userID primitive.ObjectID,
	role models.UserRole,
	req *models.UpdateBlankQuestionRequest,
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

// applyBlankUpdates 将更新请求写回实体。
func (s *BlankQuestionService) applyBlankUpdates(
	question *models.BlankQuestion,
	req *models.UpdateBlankQuestionRequest,
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
	if req.JudgeMode != nil {
		question.JudgeMode = *req.JudgeMode
	}
	if req.Blanks != nil {
		question.Blanks = *req.Blanks
	}
	if req.Analysis != nil {
		question.Analysis = *req.Analysis
	}
	if req.Score != nil {
		question.Score = *req.Score
	}
}

// syncCourseRelations 同步题目课程挂载关系。
func (s *BlankQuestionService) syncCourseRelations(
	ctx context.Context,
	questionID primitive.ObjectID,
	userID primitive.ObjectID,
	publishMode models.QuestionPublishMode,
	courseIDs []primitive.ObjectID,
) error {
	if err := s.contentRelationRepo.DeleteCourseRelationsByContent(ctx, models.ContentTypeBlank, questionID); err != nil {
		return err
	}
	if publishMode == models.QuestionPublishModeGlobal || len(courseIDs) == 0 {
		return nil
	}
	return s.attachCourseRelations(ctx, courseIDs, questionID, userID)
}

// resolveQuestionCourseIDsForUpdate 计算更新后的课程挂载范围。
func (s *BlankQuestionService) resolveQuestionCourseIDsForUpdate(
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

// listLinkedCourseIDs 查询填空题当前的课程挂载范围。
func (s *BlankQuestionService) listLinkedCourseIDs(ctx context.Context, questionID primitive.ObjectID) ([]primitive.ObjectID, error) {
	relations, err := s.contentRelationRepo.ListRelationsByContentIDs(ctx, models.ContentTypeBlank, []primitive.ObjectID{questionID})
	if err != nil {
		return nil, fmt.Errorf("查询填空题课程挂载失败: %w", err)
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
