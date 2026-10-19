/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程混合内容服务骨架，负责最小聚合与课程写权限边界校验
*/

package service

import (
	"context"
	"fmt"
	"sort"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CourseContentService 课程混合内容服务。
// 当前阶段只补齐聚合排序与权限 helper，为后续 server 层列表/挂载能力提供稳定入口。
type CourseContentService struct {
	problemRepo         *repository.ProblemRepository
	choiceRepo          *repository.ChoiceQuestionRepository
	blankRepo           *repository.BlankQuestionRepository
	contentRelationRepo *repository.ContentRelationRepository
	courseService       *CourseService
}

// NewCourseContentService 创建课程混合内容服务实例。
func NewCourseContentService(
	problemRepo *repository.ProblemRepository,
	choiceRepo *repository.ChoiceQuestionRepository,
	blankRepo *repository.BlankQuestionRepository,
	contentRelationRepo *repository.ContentRelationRepository,
	courseService *CourseService,
) *CourseContentService {
	return &CourseContentService{
		problemRepo:         problemRepo,
		choiceRepo:          choiceRepo,
		blankRepo:           blankRepo,
		contentRelationRepo: contentRelationRepo,
		courseService:       courseService,
	}
}

// mergeCourseContentItems 合并不同题型的课程内容并按创建时间倒序返回。
// 课程首页后续会混合展示旧编程题与新题型，这里先统一排序规则，避免 server 层散落重复逻辑。
func mergeCourseContentItems(
	programming []*models.CourseContentListItem,
	choices []*models.CourseContentListItem,
	blanks []*models.CourseContentListItem,
) []*models.CourseContentListItem {
	merged := make([]*models.CourseContentListItem, 0, len(programming)+len(choices)+len(blanks))
	merged = append(merged, programming...)
	merged = append(merged, choices...)
	merged = append(merged, blanks...)
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].CreatedAt.After(merged[j].CreatedAt)
	})
	return merged
}

// ensureCanWriteCourse 校验用户是否可以维护课程内容。
// 这里把管理员旁路与课程教师校验固定在 service 层，避免后续不同入口出现权限边界漂移。
func (s *CourseContentService) ensureCanWriteCourse(ctx context.Context, courseID, userID primitive.ObjectID, role models.UserRole) error {
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
		return fmt.Errorf("无权管理该课程内容")
	}
	return nil
}

// ListCourseContents 查询课程下的混合题型内容列表。
// 当前阶段编程题仍走旧 course_ids 链路，选择题和填空题走新关系表，避免在一个迭代里同时改动旧编程题入库逻辑。
func (s *CourseContentService) ListCourseContents(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
	page int,
	pageSize int,
) (*models.CourseContentListResponse, error) {
	includeUnpublished, err := s.resolveCourseReadScope(ctx, courseID, userID, role)
	if err != nil {
		return nil, err
	}

	programmingItems, err := s.listProgrammingItems(ctx, courseID, includeUnpublished)
	if err != nil {
		return nil, err
	}
	choiceItems, err := s.listChoiceItems(ctx, courseID, includeUnpublished)
	if err != nil {
		return nil, err
	}
	blankItems, err := s.listBlankItems(ctx, courseID, includeUnpublished)
	if err != nil {
		return nil, err
	}

	allItems := mergeCourseContentItems(programmingItems, choiceItems, blankItems)
	page, pageSize = normalizePage(page, pageSize)
	total := int64(len(allItems))
	start, end := sliceBounds(len(allItems), page, pageSize)

	return &models.CourseContentListResponse{
		Items:     allItems[start:end],
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: computeTotalPages(total, pageSize),
	}, nil
}

// resolveCourseReadScope 判断当前用户读取课程内容时是否可看到未发布内容。
func (s *CourseContentService) resolveCourseReadScope(
	ctx context.Context,
	courseID primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) (bool, error) {
	if role == models.RoleAdmin {
		return true, nil
	}
	if s.courseService == nil {
		return false, fmt.Errorf("课程服务未初始化")
	}
	canManage, err := s.courseService.CanManageCourse(ctx, courseID, userID)
	if err != nil {
		return false, fmt.Errorf("课程不存在或无权访问: %s", courseID.Hex())
	}
	if canManage {
		return true, nil
	}
	canView, err := s.courseService.CanViewCourse(ctx, courseID, userID)
	if err != nil {
		return false, fmt.Errorf("课程不存在或无权访问: %s", courseID.Hex())
	}
	if !canView {
		return false, fmt.Errorf("无权查看该课程内容")
	}
	return false, nil
}

// listChoiceItems 查询课程内的选择题条目。
func (s *CourseContentService) listChoiceItems(
	ctx context.Context,
	courseID primitive.ObjectID,
	includeUnpublished bool,
) ([]*models.CourseContentListItem, error) {
	relations, err := s.contentRelationRepo.ListCourseRelations(ctx, courseID, models.ContentTypeChoice)
	if err != nil {
		return nil, err
	}
	ids := relationContentIDs(relations)
	questions, err := s.choiceRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return convertChoiceQuestionsToCourseItems(questions, includeUnpublished), nil
}

// listBlankItems 查询课程内的填空题条目。
func (s *CourseContentService) listBlankItems(
	ctx context.Context,
	courseID primitive.ObjectID,
	includeUnpublished bool,
) ([]*models.CourseContentListItem, error) {
	relations, err := s.contentRelationRepo.ListCourseRelations(ctx, courseID, models.ContentTypeBlank)
	if err != nil {
		return nil, err
	}
	ids := relationContentIDs(relations)
	questions, err := s.blankRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return convertBlankQuestionsToCourseItems(questions, includeUnpublished), nil
}

// listProgrammingItems 查询课程内的编程题条目。
func (s *CourseContentService) listProgrammingItems(
	ctx context.Context,
	courseID primitive.ObjectID,
	includeUnpublished bool,
) ([]*models.CourseContentListItem, error) {
	relations, err := s.contentRelationRepo.ListCourseRelations(ctx, courseID, models.ContentTypeProgramming)
	if err != nil {
		return nil, err
	}
	ids := relationContentIDs(relations)
	return s.problemRepo.ListCourseContentItems(ctx, ids, includeUnpublished)
}

// mergeCourseScope 将当前课程和请求内额外挂载课程合并。
// 课程模式必须至少挂到当前课程；全局模式不建立课程关系，保持“创建入口”和“可见范围”解耦。
func mergeCourseScope(
	currentCourseID primitive.ObjectID,
	publishMode models.QuestionPublishMode,
	extraCourseIDs []string,
) ([]primitive.ObjectID, error) {
	if publishMode == models.QuestionPublishModeGlobal {
		return []primitive.ObjectID{}, nil
	}
	idMap := map[string]primitive.ObjectID{
		currentCourseID.Hex(): currentCourseID,
	}
	for _, courseID := range extraCourseIDs {
		objID, err := primitive.ObjectIDFromHex(courseID)
		if err != nil {
			return nil, fmt.Errorf("无效的课程ID: %s", courseID)
		}
		idMap[objID.Hex()] = objID
	}
	merged := make([]primitive.ObjectID, 0, len(idMap))
	for _, courseID := range idMap {
		merged = append(merged, courseID)
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Hex() < merged[j].Hex()
	})
	return merged, nil
}

// relationContentIDs 提取关系表中的题目 ID。
func relationContentIDs(relations []*models.CourseContentRelation) []primitive.ObjectID {
	ids := make([]primitive.ObjectID, 0, len(relations))
	for _, relation := range relations {
		if relation == nil {
			continue
		}
		ids = append(ids, relation.ContentID)
	}
	return ids
}

// objectIDsToHex 将 ObjectID 列表转换为十六进制字符串列表。
func objectIDsToHex(ids []primitive.ObjectID) []string {
	hexIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		hexIDs = append(hexIDs, id.Hex())
	}
	return hexIDs
}

// ensureCanManageTargetCourses 校验用户是否可以把题目挂到目标课程集合。
func ensureCanManageTargetCourses(
	ctx context.Context,
	courseService *CourseService,
	courseIDs []primitive.ObjectID,
	userID primitive.ObjectID,
	role models.UserRole,
) error {
	if role == models.RoleAdmin {
		return nil
	}
	if courseService == nil {
		return fmt.Errorf("课程服务未初始化")
	}
	for _, courseID := range courseIDs {
		ok, err := courseService.CanManageCourse(ctx, courseID, userID)
		if err != nil {
			return fmt.Errorf("课程不存在或无权操作: %s", courseID.Hex())
		}
		if !ok {
			return fmt.Errorf("无权挂载到课程: %s", courseID.Hex())
		}
	}
	return nil
}

// convertChoiceQuestionsToCourseItems 将选择题实体转换为课程内容列表条目。
func convertChoiceQuestionsToCourseItems(
	questions []*models.ChoiceQuestion,
	includeUnpublished bool,
) []*models.CourseContentListItem {
	items := make([]*models.CourseContentListItem, 0, len(questions))
	for _, question := range questions {
		if question == nil {
			continue
		}
		if !includeUnpublished && question.Status != models.QuestionStatusPublished {
			continue
		}
		items = append(items, &models.CourseContentListItem{
			ContentType: models.ContentTypeChoice,
			ContentID:   question.ID,
			Title:       question.Title,
			Status:      question.Status,
			PublishMode: question.PublishMode,
			Difficulty:  question.Difficulty,
			CreatedBy:   question.CreatedBy,
			CreatedAt:   question.CreatedAt,
		})
	}
	return items
}

// convertBlankQuestionsToCourseItems 将填空题实体转换为课程内容列表条目。
func convertBlankQuestionsToCourseItems(
	questions []*models.BlankQuestion,
	includeUnpublished bool,
) []*models.CourseContentListItem {
	items := make([]*models.CourseContentListItem, 0, len(questions))
	for _, question := range questions {
		if question == nil {
			continue
		}
		if !includeUnpublished && question.Status != models.QuestionStatusPublished {
			continue
		}
		items = append(items, &models.CourseContentListItem{
			ContentType: models.ContentTypeBlank,
			ContentID:   question.ID,
			Title:       question.Title,
			Status:      question.Status,
			PublishMode: question.PublishMode,
			Difficulty:  question.Difficulty,
			CreatedBy:   question.CreatedBy,
			CreatedAt:   question.CreatedAt,
		})
	}
	return items
}

// normalizePage 归一化分页参数。
func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

// sliceBounds 计算分页切片边界。
func sliceBounds(total, page, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

// computeTotalPages 计算总页数。
func computeTotalPages(total int64, pageSize int) int64 {
	if total == 0 {
		return 0
	}
	return (total + int64(pageSize) - 1) / int64(pageSize)
}
