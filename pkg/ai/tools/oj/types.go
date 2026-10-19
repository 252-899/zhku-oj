// types.go 定义 OJ 业务工具稳定的 JSON 入参和安全出参。
package oj

import (
	"time"

	"zk-code-arena-server/pkg/models"
)

// GetUserProfileInput 是查询用户资料工具的入参。
type GetUserProfileInput struct {
	UserID    string `json:"user_id,omitempty" jsonschema:"description=Internal Mongo ObjectID of the user."`
	StudentID string `json:"student_id,omitempty" jsonschema:"description=Student or teacher number. Use this when the user_id is unknown."`
}

// GetClassInfoInput 是查询班级信息工具的入参。
type GetClassInfoInput struct {
	ClassID string `json:"class_id" jsonschema:"description=Internal Mongo ObjectID of the class."`
}

// ListTeacherClassesInput 是查询当前教师可见班级列表的入参。
type ListTeacherClassesInput struct {
	Keyword string `json:"keyword,omitempty" jsonschema:"description=Optional fuzzy keyword for class name, such as 23 软件 1 班."`
}

// ListClassStudentsInput 是查询班级学生工具的入参。
type ListClassStudentsInput struct {
	ClassID string `json:"class_id" jsonschema:"description=Internal Mongo ObjectID of the class."`
}

// ListClassTasksInput 是查询班级任务工具的入参。
type ListClassTasksInput struct {
	ClassID string `json:"class_id" jsonschema:"description=Internal Mongo ObjectID of the class."`
}

// GetTaskDetailInput 是查询任务详情工具的入参。
type GetTaskDetailInput struct {
	TaskID string `json:"task_id" jsonschema:"description=Internal Mongo ObjectID of the task."`
}

// GetTaskCompletionInput 是查询任务完成情况工具的入参。
type GetTaskCompletionInput struct {
	ClassID  string `json:"class_id" jsonschema:"description=Internal Mongo ObjectID of the class."`
	TaskID   string `json:"task_id" jsonschema:"description=Internal Mongo ObjectID of the task."`
	PageNum  int64  `json:"page_num,omitempty" jsonschema:"description=Page number, default is 1."`
	PageSize int64  `json:"page_size,omitempty" jsonschema:"description=Page size, default is 10 and max is 50."`
	RealName string `json:"real_name,omitempty" jsonschema:"description=Optional fuzzy real name filter."`
	UserID   string `json:"user_id,omitempty" jsonschema:"description=Optional exact internal user ObjectID filter."`
}

// SearchProblemsInput 是搜索题库工具的入参。
type SearchProblemsInput struct {
	Keyword    string   `json:"keyword,omitempty" jsonschema:"description=Keyword used to search problem title or description."`
	Difficulty string   `json:"difficulty,omitempty" jsonschema:"description=Problem difficulty: easy, medium, hard, or empty."`
	Tags       []string `json:"tags,omitempty" jsonschema:"description=Problem tags or knowledge points."`
	Page       int      `json:"page,omitempty" jsonschema:"description=Page number, default is 1."`
	PageSize   int      `json:"page_size,omitempty" jsonschema:"description=Page size, default is 10 and max is 50."`
	InFavorite bool     `json:"in_favorite,omitempty" jsonschema:"description=Whether to search only the current actor's favorite problems."`
}

// GetProblemDetailInput 是查询题目详情工具的入参。
type GetProblemDetailInput struct {
	ProblemID string `json:"problem_id" jsonschema:"description=Internal Mongo ObjectID of the problem."`
}

// UserSummary 会刻意省略密码、手机号等敏感字段，再返回给 LLM 工具调用。
type UserSummary struct {
	ID           string          `json:"id"`
	Username     string          `json:"username,omitempty"`
	RealName     string          `json:"real_name,omitempty"`
	StudentID    string          `json:"student_id,omitempty"`
	Role         models.UserRole `json:"role,omitempty"`
	School       string          `json:"school,omitempty"`
	Major        string          `json:"major,omitempty"`
	Grade        string          `json:"grade,omitempty"`
	Class        string          `json:"class,omitempty"`
	IsActive     bool            `json:"is_active"`
	MajorClassID string          `json:"major_class_id,omitempty"`
}

// ClassSummary 是返回给 Agent 的班级摘要信息。
type ClassSummary struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Description   string        `json:"description,omitempty"`
	CourseID      string        `json:"course_id,omitempty"`
	Schedule      string        `json:"schedule,omitempty"`
	Teachers      []UserSummary `json:"teachers,omitempty"`
	MaxMembers    int           `json:"max_members,omitempty"`
	StudentCount  int           `json:"student_count"`
	Status        int8          `json:"status"`
	MajorClassIDs []string      `json:"major_class_ids,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

// TaskCompletionSummary 是返回给 Agent 的任务完成情况摘要。
type TaskCompletionSummary struct {
	Total      int64                           `json:"total"`
	PageNum    int64                           `json:"page_num"`
	PageSize   int64                           `json:"page_size"`
	Completion []models.TaskCompletionResponse `json:"completion"`
	Summary    CompletionStats                 `json:"summary"`
}

// CompletionStats 是当前任务完成情况页的聚合统计。
type CompletionStats struct {
	CompletedCount   int     `json:"completed_count"`
	UncompletedCount int     `json:"uncompleted_count"`
	CompletionRate   float64 `json:"completion_rate"`
}
