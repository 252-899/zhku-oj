// handlers.go 实现 OJ 只读业务工具的具体处理逻辑。
package oj

import (
	"context"
	"errors"
	"strings"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetUserProfile 根据 user_id 或 student_id 查询安全用户资料。
func (p *Provider) GetUserProfile(ctx context.Context, input GetUserProfileInput) (string, error) {
	if p == nil || p.deps.UserReader == nil {
		return "", errors.New("user reader is not configured")
	}

	var (
		user *models.User
		err  error
	)
	if strings.TrimSpace(input.UserID) != "" {
		userID, parseErr := parseObjectID("user_id", input.UserID)
		if parseErr != nil {
			return "", parseErr
		}
		user, err = p.deps.UserReader.GetUserByID(ctx, userID)
	} else if strings.TrimSpace(input.StudentID) != "" {
		user, err = p.deps.UserReader.GetUserByStudentID(ctx, strings.TrimSpace(input.StudentID))
	} else {
		return "", errors.New("user_id or student_id is required")
	}
	if err != nil {
		return "", err
	}
	return marshalToolResult(toUserSummary(user))
}

// GetClassInfo 返回当前操作者可见的班级元数据。
func (p *Provider) GetClassInfo(ctx context.Context, input GetClassInfoInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	actorUserID, err := p.requireActor()
	if err != nil {
		return "", err
	}
	classID := strings.TrimSpace(input.ClassID)
	if classID == "" {
		return "", errors.New("class_id is required")
	}

	clazz, err := p.deps.ClazzReader.GetClazzByID(ctx, classID, actorUserID)
	if err != nil {
		return "", err
	}
	return marshalToolResult(toClassSummary(clazz))
}

// ListTeacherClasses 返回当前教师可见的班级列表，可用关键词按名称过滤。
func (p *Provider) ListTeacherClasses(ctx context.Context, input ListTeacherClassesInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	actorUserID, err := p.requireActor()
	if err != nil {
		return "", err
	}
	clazzes, err := p.deps.ClazzReader.ListTeacherClasses(ctx, actorUserID)
	if err != nil {
		return "", err
	}
	keyword := strings.ToLower(strings.TrimSpace(input.Keyword))
	items := make([]ClassSummary, 0, len(clazzes))
	for _, clazz := range clazzes {
		if keyword != "" && !strings.Contains(strings.ToLower(clazz.Name), keyword) {
			continue
		}
		items = append(items, classSummaryFromClazz(clazz))
	}
	return marshalToolResult(map[string]any{
		"total":   len(items),
		"classes": items,
	})
}

// ListClassStudents 返回班级内学生的安全资料列表。
func (p *Provider) ListClassStudents(ctx context.Context, input ListClassStudentsInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	classID := strings.TrimSpace(input.ClassID)
	if classID == "" {
		return "", errors.New("class_id is required")
	}

	students, err := p.deps.ClazzReader.GetClassStudents(ctx, classID)
	if err != nil {
		return "", err
	}

	items := make([]UserSummary, 0, len(students))
	for _, student := range students {
		items = append(items, userSummaryFromProfile(student))
	}
	return marshalToolResult(map[string]any{
		"total":    len(items),
		"students": items,
	})
}

// ListClassTasks 返回指定班级中当前操作者可见的任务列表。
func (p *Provider) ListClassTasks(ctx context.Context, input ListClassTasksInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	actorUserID, err := p.requireActor()
	if err != nil {
		return "", err
	}
	classID := strings.TrimSpace(input.ClassID)
	if classID == "" {
		return "", errors.New("class_id is required")
	}

	tasks, err := p.deps.ClazzReader.GetTasksByClazzID(ctx, classID, actorUserID)
	if err != nil {
		return "", err
	}
	return marshalToolResult(map[string]any{
		"total": len(tasks),
		"tasks": tasks,
	})
}

// GetTaskDetail 返回单个任务详情及其关联题目摘要。
func (p *Provider) GetTaskDetail(ctx context.Context, input GetTaskDetailInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	actorUserID, err := p.requireActor()
	if err != nil {
		return "", err
	}
	taskID := strings.TrimSpace(input.TaskID)
	if taskID == "" {
		return "", errors.New("task_id is required")
	}

	task, err := p.deps.ClazzReader.GetTaskByID(ctx, taskID, actorUserID)
	if err != nil {
		return "", err
	}
	return marshalToolResult(task)
}

// GetTaskCompletion 返回任务完成情况分页记录，并附带当前页的聚合摘要。
func (p *Provider) GetTaskCompletion(ctx context.Context, input GetTaskCompletionInput) (string, error) {
	if p == nil || p.deps.ClazzReader == nil {
		return "", errors.New("class reader is not configured")
	}
	classID := strings.TrimSpace(input.ClassID)
	taskID := strings.TrimSpace(input.TaskID)
	if classID == "" {
		return "", errors.New("class_id is required")
	}
	if taskID == "" {
		return "", errors.New("task_id is required")
	}

	pageNum, pageSize := normalizedPage64(input.PageNum, input.PageSize)
	req := &models.PageQueryTaskCompletionRequest{
		PageNum:  &pageNum,
		PageSize: &pageSize,
	}
	if strings.TrimSpace(input.RealName) != "" {
		realName := strings.TrimSpace(input.RealName)
		req.RealName = &realName
	}
	if strings.TrimSpace(input.UserID) != "" {
		userID := strings.TrimSpace(input.UserID)
		req.UserID = &userID
	}

	result, err := p.deps.ClazzReader.PageQueryTaskCompletion(ctx, classID, taskID, req)
	if err != nil {
		return "", err
	}
	return marshalToolResult(toTaskCompletionSummary(result))
}

// SearchProblems 搜索题库。InFavorite 为 true 时，搜索范围限定为当前用户收藏。
func (p *Provider) SearchProblems(ctx context.Context, input SearchProblemsInput) (string, error) {
	if p == nil || p.deps.ProblemReader == nil {
		return "", errors.New("problem reader is not configured")
	}
	page, pageSize := normalizedPage(input.Page, input.PageSize)
	if pageSize == defaultPageSize && input.PageSize == 0 {
		pageSize = defaultProblemSearchSize
	}

	var actorID *primitive.ObjectID
	if input.InFavorite {
		actorUserID, err := p.requireActor()
		if err != nil {
			return "", err
		}
		id, err := parseObjectID("actor_user_id", actorUserID)
		if err != nil {
			return "", err
		}
		actorID = &id
	}

	problems, total, err := p.deps.ProblemReader.SearchProblems(
		ctx,
		strings.TrimSpace(input.Keyword),
		models.ProblemDifficulty(strings.TrimSpace(input.Difficulty)),
		trimStrings(input.Tags),
		page,
		pageSize,
		actorID,
		input.InFavorite,
	)
	if err != nil {
		return "", err
	}
	return marshalToolResult(map[string]any{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"problems":  problems,
	})
}

// GetProblemDetail 返回题目详情及样例测试用例。
func (p *Provider) GetProblemDetail(ctx context.Context, input GetProblemDetailInput) (string, error) {
	if p == nil || p.deps.ProblemReader == nil {
		return "", errors.New("problem reader is not configured")
	}
	problemID, err := parseObjectID("problem_id", input.ProblemID)
	if err != nil {
		return "", err
	}
	detail, err := p.deps.ProblemReader.GetProblemDetail(ctx, problemID)
	if err != nil {
		return "", err
	}
	return marshalToolResult(detail)
}

func toUserSummary(user *models.User) UserSummary {
	if user == nil {
		return UserSummary{}
	}
	return UserSummary{
		ID:           user.ID.Hex(),
		Username:     user.Username,
		RealName:     user.RealName,
		StudentID:    user.StudentID,
		Role:         user.Role,
		School:       user.School,
		Major:        user.Major,
		Grade:        user.Grade,
		Class:        user.Class,
		IsActive:     user.IsActive,
		MajorClassID: user.MajorClassID,
	}
}

func userSummaryFromProfile(profile *models.UserProfile) UserSummary {
	if profile == nil {
		return UserSummary{}
	}
	return UserSummary{
		ID:        profile.ID.Hex(),
		Username:  profile.Username,
		RealName:  profile.RealName,
		StudentID: profile.StudentID,
		Role:      profile.Role,
		School:    profile.School,
		Major:     profile.Major,
		Grade:     profile.Grade,
		Class:     profile.Class,
		IsActive:  profile.IsActive,
	}
}

func toClassSummary(clazz *models.GetClazzResponse) ClassSummary {
	if clazz == nil {
		return ClassSummary{}
	}
	teachers := make([]UserSummary, 0, len(clazz.Teachers))
	for i := range clazz.Teachers {
		teachers = append(teachers, userSummaryFromProfile(&clazz.Teachers[i]))
	}
	majorClassIDs := make([]string, 0, len(clazz.MajorClassIDs))
	for _, id := range clazz.MajorClassIDs {
		majorClassIDs = append(majorClassIDs, id.Hex())
	}
	return ClassSummary{
		ID:            clazz.ID.Hex(),
		Name:          clazz.Name,
		Description:   clazz.Description,
		CourseID:      clazz.CourseId.Hex(),
		Schedule:      clazz.Schedule,
		Teachers:      teachers,
		MaxMembers:    clazz.MaxMembers,
		StudentCount:  clazz.AddNums,
		Status:        int8(clazz.Status),
		MajorClassIDs: majorClassIDs,
		CreatedAt:     clazz.CTime,
	}
}

// classSummaryFromClazz 将数据库班级模型转换为给 Agent 使用的安全摘要。
func classSummaryFromClazz(clazz models.Clazz) ClassSummary {
	majorClassIDs := make([]string, 0, len(clazz.MajorClassIDs))
	for _, id := range clazz.MajorClassIDs {
		majorClassIDs = append(majorClassIDs, id.Hex())
	}
	return ClassSummary{
		ID:            clazz.ID.Hex(),
		Name:          clazz.Name,
		Description:   clazz.Description,
		CourseID:      clazz.CourseId.Hex(),
		Schedule:      clazz.Schedule,
		MaxMembers:    clazz.MaxMembers,
		StudentCount:  clazz.StudentMemberCount(),
		Status:        int8(clazz.Status),
		MajorClassIDs: majorClassIDs,
		CreatedAt:     clazz.CTime,
	}
}

func toTaskCompletionSummary(result *models.PageQueryTaskCompletionResponse) TaskCompletionSummary {
	if result == nil {
		return TaskCompletionSummary{}
	}
	completed := 0
	for _, item := range result.Completion {
		if item.Completed {
			completed++
		}
	}
	totalOnPage := len(result.Completion)
	uncompleted := totalOnPage - completed
	rate := 0.0
	if totalOnPage > 0 {
		rate = float64(completed) / float64(totalOnPage)
	}
	return TaskCompletionSummary{
		Total:      result.Total,
		PageNum:    result.PageNum,
		PageSize:   result.PageSize,
		Completion: result.Completion,
		Summary: CompletionStats{
			CompletedCount:   completed,
			UncompletedCount: uncompleted,
			CompletionRate:   rate,
		},
	}
}

func trimStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
