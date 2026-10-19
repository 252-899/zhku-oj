// 包 oj 提供 AI Agent 可调用的 OJ 只读业务工具。
package oj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"zk-code-arena-server/pkg/models"

	einotool "github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	// ToolGetUserProfile 是查询安全用户资料的工具名。
	ToolGetUserProfile = "oj_get_user_profile"
	// ToolGetClassInfo 是查询班级基础信息的工具名。
	ToolGetClassInfo = "oj_get_class_info"
	// ToolListTeacherClasses 是查询当前教师可见班级列表的工具名。
	ToolListTeacherClasses = "oj_list_teacher_classes"
	// ToolListClassStudents 是查询班级学生列表的工具名。
	ToolListClassStudents = "oj_list_class_students"
	// ToolListClassTasks 是查询班级任务列表的工具名。
	ToolListClassTasks = "oj_list_class_tasks"
	// ToolGetTaskDetail 是查询任务详情和关联题目的工具名。
	ToolGetTaskDetail = "oj_get_task_detail"
	// ToolGetTaskCompletion 是查询任务完成情况的工具名。
	ToolGetTaskCompletion = "oj_get_task_completion"
	// ToolSearchProblems 是搜索题库的工具名。
	ToolSearchProblems = "oj_search_problems"
	// ToolGetProblemDetail 是查询题目详情和样例的工具名。
	ToolGetProblemDetail = "oj_get_problem_detail"

	defaultPage              = 1
	defaultPageSize          = 10
	maxToolPageSize          = 50
	defaultProblemSearchSize = 10
)

// UserReader 是 AI 工具查询用户信息所需的最小接口。
// api-server 的 UserService 已经满足这个接口形状。
type UserReader interface {
	GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	GetUserByStudentID(ctx context.Context, stdId string) (*models.User, error)
}

// ClazzReader 是教师侧工具查询班级和任务所需的最小接口。
// 保持窄接口可以避免在 AI 包中直接依赖整个 service 层。
type ClazzReader interface {
	ListTeacherClasses(ctx context.Context, userID string) ([]models.Clazz, error)
	GetClazzByID(ctx context.Context, clazzID string, userID string) (*models.GetClazzResponse, error)
	GetClassStudents(ctx context.Context, classID string) ([]*models.UserProfile, error)
	GetTasksByClazzID(ctx context.Context, clazzID string, userID string) ([]models.Task, error)
	GetTaskByID(ctx context.Context, taskID string, userID string) (*models.TaskResponse, error)
	PageQueryTaskCompletion(ctx context.Context, clazzID string, taskID string, req *models.PageQueryTaskCompletionRequest) (*models.PageQueryTaskCompletionResponse, error)
}

// ProblemReader 是推荐和作业分析 Agent 查询题目所需的最小接口。
type ProblemReader interface {
	SearchProblems(ctx context.Context, keyword string, difficulty models.ProblemDifficulty, tags []string, page, pageSize int, userID *primitive.ObjectID, inFavorite bool) ([]*models.ProblemList, int64, error)
	GetProblemDetail(ctx context.Context, problemID primitive.ObjectID) (*models.ProblemDetailResponse, error)
}

// Dependencies 组合可挂载为工具的业务查询接口。Provider 可以只配置其中一部分，
// BuildTools 只会暴露依赖满足的工具。
type Dependencies struct {
	UserReader    UserReader
	ClazzReader   ClazzReader
	ProblemReader ProblemReader
}

// Provider 为某个已认证用户构建并执行 OJ 只读工具。
// actorUserID 由 handler 或 service 固定注入，不接受模型参数传入，
// 避免模型伪造操作者身份。
type Provider struct {
	deps        Dependencies
	actorUserID string
}

// NewProvider 为当前 AI 请求创建业务工具 Provider。
func NewProvider(deps Dependencies, actorUserID string) *Provider {
	return &Provider{
		deps:        deps,
		actorUserID: strings.TrimSpace(actorUserID),
	}
}

// IsConfigured 判断当前 Provider 是否至少配置了一类业务依赖。
func (p *Provider) IsConfigured() bool {
	return p != nil && (p.deps.UserReader != nil || p.deps.ClazzReader != nil || p.deps.ProblemReader != nil)
}

// BuildTools 创建 Eino 工具定义，只暴露依赖已经配置好的工具。
func (p *Provider) BuildTools(ctx context.Context) ([]einotool.BaseTool, error) {
	if !p.IsConfigured() {
		return nil, errors.New("oj tool provider is not configured")
	}

	items := make([]einotool.BaseTool, 0, 8)

	if p.deps.UserReader != nil {
		spec := mustToolSpec(ToolGetUserProfile)
		tool, err := inferJSONTool(
			spec.Name,
			spec.Description,
			p.GetUserProfile,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, tool)
	}

	if p.deps.ClazzReader != nil {
		classTools, err := p.buildClassTools()
		if err != nil {
			return nil, err
		}
		items = append(items, classTools...)
	}

	if p.deps.ProblemReader != nil {
		problemTools, err := p.buildProblemTools()
		if err != nil {
			return nil, err
		}
		items = append(items, problemTools...)
	}

	return items, nil
}

// buildClassTools 构建班级、任务相关的 Eino 工具定义。
func (p *Provider) buildClassTools() ([]einotool.BaseTool, error) {
	items := make([]einotool.BaseTool, 0, 6)

	listTeacherClassesSpec := mustToolSpec(ToolListTeacherClasses)
	listTeacherClasses, err := inferJSONTool(
		listTeacherClassesSpec.Name,
		listTeacherClassesSpec.Description,
		p.ListTeacherClasses,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, listTeacherClasses)

	getClassInfoSpec := mustToolSpec(ToolGetClassInfo)
	getClassInfo, err := inferJSONTool(
		getClassInfoSpec.Name,
		getClassInfoSpec.Description,
		p.GetClassInfo,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, getClassInfo)

	listClassStudentsSpec := mustToolSpec(ToolListClassStudents)
	listClassStudents, err := inferJSONTool(
		listClassStudentsSpec.Name,
		listClassStudentsSpec.Description,
		p.ListClassStudents,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, listClassStudents)

	listClassTasksSpec := mustToolSpec(ToolListClassTasks)
	listClassTasks, err := inferJSONTool(
		listClassTasksSpec.Name,
		listClassTasksSpec.Description,
		p.ListClassTasks,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, listClassTasks)

	getTaskDetailSpec := mustToolSpec(ToolGetTaskDetail)
	getTaskDetail, err := inferJSONTool(
		getTaskDetailSpec.Name,
		getTaskDetailSpec.Description,
		p.GetTaskDetail,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, getTaskDetail)

	getTaskCompletionSpec := mustToolSpec(ToolGetTaskCompletion)
	getTaskCompletion, err := inferJSONTool(
		getTaskCompletionSpec.Name,
		getTaskCompletionSpec.Description,
		p.GetTaskCompletion,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, getTaskCompletion)

	return items, nil
}

// buildProblemTools 构建题库相关的 Eino 工具定义。
func (p *Provider) buildProblemTools() ([]einotool.BaseTool, error) {
	items := make([]einotool.BaseTool, 0, 2)

	searchProblemsSpec := mustToolSpec(ToolSearchProblems)
	searchProblems, err := inferJSONTool(
		searchProblemsSpec.Name,
		searchProblemsSpec.Description,
		p.SearchProblems,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, searchProblems)

	getProblemDetailSpec := mustToolSpec(ToolGetProblemDetail)
	getProblemDetail, err := inferJSONTool(
		getProblemDetailSpec.Name,
		getProblemDetailSpec.Description,
		p.GetProblemDetail,
	)
	if err != nil {
		return nil, err
	}
	items = append(items, getProblemDetail)

	return items, nil
}

// inferJSONTool 统一通过 Eino InferTool 生成 JSON 参数工具。
func inferJSONTool[I any](
	name string,
	description string,
	handler func(context.Context, I) (string, error),
) (einotool.BaseTool, error) {
	return toolutils.InferTool(name, description, handler)
}

// mustToolSpec 根据工具名读取工具目录，目录缺失时返回保守兜底定义。
func mustToolSpec(name string) ToolSpec {
	for _, item := range Catalog() {
		if item.Name == name {
			return item
		}
	}
	return ToolSpec{Name: name, Description: name, ReadOnly: true}
}

// Execute 根据工具名和原始 JSON 参数执行工具。它服务于自定义 ReAct 循环，
// 避免直接使用 Eino ReactAgent 黑盒。
func (p *Provider) Execute(ctx context.Context, name string, arguments string) (string, error) {
	switch name {
	case ToolGetUserProfile:
		var input GetUserProfileInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.GetUserProfile(ctx, input)
	case ToolGetClassInfo:
		var input GetClassInfoInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.GetClassInfo(ctx, input)
	case ToolListTeacherClasses:
		var input ListTeacherClassesInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.ListTeacherClasses(ctx, input)
	case ToolListClassStudents:
		var input ListClassStudentsInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.ListClassStudents(ctx, input)
	case ToolListClassTasks:
		var input ListClassTasksInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.ListClassTasks(ctx, input)
	case ToolGetTaskDetail:
		var input GetTaskDetailInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.GetTaskDetail(ctx, input)
	case ToolGetTaskCompletion:
		var input GetTaskCompletionInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.GetTaskCompletion(ctx, input)
	case ToolSearchProblems:
		var input SearchProblemsInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.SearchProblems(ctx, input)
	case ToolGetProblemDetail:
		var input GetProblemDetailInput
		if err := json.Unmarshal([]byte(arguments), &input); err != nil {
			return "", fmt.Errorf("invalid %s arguments: %w", name, err)
		}
		return p.GetProblemDetail(ctx, input)
	default:
		return "", fmt.Errorf("unsupported oj tool: %s", name)
	}
}

// requireActor 校验当前请求是否已经绑定登录用户身份。
func (p *Provider) requireActor() (string, error) {
	if p == nil || p.actorUserID == "" {
		return "", errors.New("actor user id is required")
	}
	return p.actorUserID, nil
}

// marshalToolResult 将工具返回值序列化为模型可消费的 JSON 字符串。
func marshalToolResult(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// parseObjectID 解析 Mongo ObjectID，并返回带字段名的错误信息。
func parseObjectID(name string, value string) (primitive.ObjectID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return primitive.NilObjectID, fmt.Errorf("%s is required", name)
	}
	id, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		return primitive.NilObjectID, fmt.Errorf("%s is invalid", name)
	}
	return id, nil
}

// normalizedPage 规范分页参数，避免模型传入过大的 page_size。
func normalizedPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxToolPageSize {
		pageSize = maxToolPageSize
	}
	return page, pageSize
}

// normalizedPage64 规范 int64 版本的分页参数。
func normalizedPage64(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = defaultPage
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxToolPageSize {
		pageSize = maxToolPageSize
	}
	return page, pageSize
}
