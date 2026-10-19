/*
@Author: omenkk7
@Date: 2025/10/09
@Description: 统一请求参数定义
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ==================== 用户模块请求 ====================

// LoginRequest 用户登录请求
type LoginRequest struct {
	StudentID string `json:"student_id" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// UpdateProfileRequest 更新用户个人信息请求
type UpdateProfileRequest struct {
	RealName string `json:"real_name"`
	Email    string `json:"email"`
	Bio      string `json:"bio"`
	School   string `json:"school"`
	Major    string `json:"major"`
	Grade    string `json:"grade"`
	Class    string `json:"class"`
	Phone    string `json:"phone"`
}

// UpdateUserRequest 更新用户信息请求（管理员）
type UpdateUserRequest struct {
	Role     string `json:"role"`
	IsActive *bool  `json:"is_active"`
}

// ==================== 题目模块请求 ====================

// CreateProblemRequest 创建题目请求
type CreateProblemRequest struct {
	// 基本信息（必填）
	Title       string `json:"title" binding:"required,min=1,max=200"`
	Description string `json:"description" binding:"required,min=10"`

	// 题目详情（可选）
	Input        string `json:"input"`
	Output       string `json:"output"`
	SampleInput  string `json:"sample_input"`
	SampleOutput string `json:"sample_output"`
	Hint         string `json:"hint"`
	Source       string `json:"source"`
	Author       string `json:"author"`

	// 难度和标签（必填）
	Difficulty ProblemDifficulty `json:"difficulty" binding:"required,oneof=easy medium hard"`
	Tags       []string          `json:"tags" binding:"max=10,dive,min=1,max=20"`

	// 限制条件（可选，有默认值）
	TimeLimit   *int `json:"time_limit" binding:"omitempty,min=100,max=10000"` // 默认1000ms
	MemoryLimit *int `json:"memory_limit" binding:"omitempty,min=32,max=1024"` // 默认256MB

	// 状态控制（可选）
	Status    *ProblemStatus `json:"status" binding:"omitempty,oneof=draft published archived"`
	IsPublic  *bool          `json:"is_public"`                                  // 可选，默认true；控制是否进入首页/全部题目
	CourseIDs []string       `json:"course_ids" binding:"omitempty,dive,len=24"` // 可选；控制题目挂载到哪些课程题库
}

// EffectiveIsPublic 返回创建题目时的有效首页可见性。
func (r CreateProblemRequest) EffectiveIsPublic() bool {
	// 兼容旧客户端：未显式传 is_public 时，继续按“新增题目默认进入主页题库”处理。
	if r.IsPublic == nil {
		return true
	}
	return *r.IsPublic
}

// UpdateProblemRequest 更新题目请求
type UpdateProblemRequest struct {
	// 基本信息（可选）
	Title       *string `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string `json:"description" binding:"omitempty,min=10"`

	// 题目详情（可选）
	Input        *string `json:"input"`
	Output       *string `json:"output"`
	SampleInput  *string `json:"sample_input"`
	SampleOutput *string `json:"sample_output"`
	Hint         *string `json:"hint"`
	Source       *string `json:"source"`
	Author       *string `json:"author"`

	// 难度和标签（可选）
	Difficulty *ProblemDifficulty `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
	Tags       *[]string          `json:"tags" binding:"omitempty,max=10,dive,min=1,max=20"`

	// 限制条件（可选）
	TimeLimit   *int `json:"time_limit" binding:"omitempty,min=100,max=10000"`
	MemoryLimit *int `json:"memory_limit" binding:"omitempty,min=32,max=1024"`

	// 状态控制（可选）
	Status    *ProblemStatus `json:"status" binding:"omitempty,oneof=draft published archived"`
	IsPublic  *bool          `json:"is_public"`
	CourseIDs *[]string      `json:"course_ids" binding:"omitempty,dive,len=24"` // 指针用于区分未传和显式清空课程挂载
}

// RunCodeRequest 运行代码请求
// 注意：这个已在 service 包中定义，这里只是引用说明

// ==================== 提交模块请求 ====================

// SubmitCodeRequest 提交代码请求
type SubmitCodeRequest struct {
	ProblemID primitive.ObjectID `json:"problem_id" binding:"required"`
	Code      string             `json:"code" binding:"required"`
	Language  string             `json:"language" binding:"required"`
}

// ==================== 测试用例模块请求 ====================

// CreateTestCaseRequest 创建测试用例请求
type CreateTestCaseRequest struct {
	ProblemID   string `json:"problem_id" binding:"required"`
	Input       string `json:"input" binding:"required"`
	Output      string `json:"output" binding:"required"`
	IsSample    bool   `json:"is_sample"`
	TimeLimit   *int   `json:"time_limit,omitempty"`   // 可选的超时配置（ms）
	MemoryLimit *int   `json:"memory_limit,omitempty"` // 可选的内存限制（MB）
	Score       int    `json:"score,omitempty"`        // 用例分数
}

// UpdateTestCaseRequest 更新测试用例请求
type UpdateTestCaseRequest struct {
	Input       string `json:"input"`
	Output      string `json:"output"`
	IsSample    *bool  `json:"is_sample,omitempty"`
	TimeLimit   *int   `json:"time_limit,omitempty"`
	MemoryLimit *int   `json:"memory_limit,omitempty"`
	Score       *int   `json:"score,omitempty"`
}

// BatchCreateTestCasesRequest 批量创建测试用例请求
type BatchCreateTestCasesRequest struct {
	ProblemID string     `json:"problem_id" binding:"required"`
	TestCases []TestCase `json:"test_cases" binding:"required,min=1"`
}

// BatchCreateProblemsRequest 批量创建题目请求
type BatchCreateProblemsRequest struct {
	Problems []CreateProblemRequest `json:"problems" binding:"required,min=1,max=100"` // 限制最多100个题目
}

// BatchCreateProblemResult 批量创建题目结果
type BatchCreateProblemResult struct {
	Index     int    `json:"index"`                // 题目在请求中的索引
	Title     string `json:"title"`                // 题目标题
	Success   bool   `json:"success"`              // 是否创建成功
	ProblemID string `json:"problem_id,omitempty"` // 成功时的题目ID
	Error     string `json:"error,omitempty"`      // 失败时的错误信息
}

// BatchCreateProblemsResponse 批量创建题目响应
type BatchCreateProblemsResponse struct {
	SuccessCount int                        `json:"success_count"` // 成功数量
	FailCount    int                        `json:"fail_count"`    // 失败数量
	TotalCount   int                        `json:"total_count"`   // 总数量
	Results      []BatchCreateProblemResult `json:"results"`       // 详细结果
}

// ==================== 课程模块请求 ====================

// CreateCourseRequest 创建课程请求
type CreateCourseRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
}

// UpdateCourseRequest 更新课程请求
type UpdateCourseRequest struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	Status      *CourseStatus `json:"status,omitempty"`
}

// PageQueryCourseRequest 分页查询课程请求（查询创建的课程）
type PageQueryCourseRequest struct {
	PageNum  *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize *int64  `json:"page_size,omitempty" form:"page_size"`
	Status   *int8   `json:"status,omitempty" form:"status"`
	Name     *string `json:"name,omitempty" form:"name"`
}

// PageQueryCourseStudentsRequest 分页查询课程学生请求
type PageQueryCourseStudentsRequest struct {
	PageNum   *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize  *int64  `json:"page_size,omitempty" form:"page_size"`
	RealName  *string `json:"real_name,omitempty" form:"real_name"`
	StudentId *string `json:"student_id,omitempty" form:"student_id"`
}

// PageQueryCourseTeachersRequest 分页查询课程教师请求
type PageQueryCourseTeachersRequest struct {
	PageNum   *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize  *int64  `json:"page_size,omitempty" form:"page_size"`
	RealName  *string `json:"real_name,omitempty" form:"real_name"`
	TeacherId *string `json:"teacher_id,omitempty" form:"teacher_id"`
}

// PageQueryAllTeachersRequest 分页查询所有教师请求
type PageQueryAllTeachersRequest struct {
	PageNum  *int64  `json:"page_num,omitempty"`
	PageSize *int64  `json:"page_size,omitempty"`
	Username *string `json:"username,omitempty"` // 登录名/展示名，支持模糊匹配（不区分大小写）
	IsActive *bool   `json:"is_active,omitempty"`
}

// PageQueryCourseStudentsResponse 分页查询课程学生响应
type PageQueryCourseStudentsResponse struct {
	Total    int64         `json:"total" example:"100"`
	PageNum  int64         `json:"page_num" example:"1"`
	PageSize int64         `json:"page_size" example:"10"`
	Students []UserProfile `json:"students"`
}

// PageQueryCourseTeachersResponse 分页查询课程教师响应
type PageQueryCourseTeachersResponse struct {
	Total    int64         `json:"total" example:"100"`
	PageNum  int64         `json:"page_num" example:"1"`
	PageSize int64         `json:"page_size" example:"10"`
	Teachers []UserProfile `json:"teachers"`
}

// PageQueryTeacherCoursesRequest 分页查询老师加入的课程请求
type PageQueryTeacherCoursesRequest struct {
	PageNum  *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize *int64  `json:"page_size,omitempty" form:"page_size"`
	Status   *int8   `json:"status,omitempty" form:"status"`
	Name     *string `json:"name,omitempty" form:"name"`
}

// PageQueryCourseResponse 分页查询课程响应
type PageQueryCourseResponse struct {
	Total    int64    `json:"total" example:"100"`
	PageNum  int64    `json:"page_num" example:"1"`
	PageSize int64    `json:"page_size" example:"10"`
	Courses  []Course `json:"courses"`
}

// AddCourseTeachersRequest 课程添加教师请求
type AddCourseTeachersRequest struct {
	TeacherIds []string `json:"teacher_ids" binding:"required"`
}

// RemoveCourseTeachersRequest 课程删除教师请求 (已废弃，改为路径参数)
// type RemoveCourseTeachersRequest struct {
// 	CourseId   string   `json:"course_id" binding:"required"`
// 	TeacherIds []string `json:"teacher_ids" binding:"required"`
// }

// ==================== 班级模块请求 ====================

// CreateMajorClassRequest 创建专业班级请求
type CreateMajorClassRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`
	StudentIds  []string `json:"student_ids" binding:"required"`
	MaxMembers  *int     `json:"max_members,omitempty"`
}

// CreateCourseClassRequest 创建课程班级请求
type CreateCourseClassRequest struct {
	Name          string   `json:"name" binding:"required"`
	CourseId      string   `json:"course_id" binding:"required"`
	Description   string   `json:"description,omitempty"`
	TeacherIds    []string `json:"teacher_ids" binding:"required"`
	Schedule      string   `json:"schedule,omitempty"`
	RequireInvite bool     `json:"require_invite" binding:"required"`
	MaxMembers    *int     `json:"max_members,omitempty"`
	//MajorClassIDs []string `json:"major_class_ids,omitempty"` // 关联的专业班级ID数组
}

// UpdateClazzRequest 更新班级请求
type UpdateClazzRequest struct {
	Name          string `json:"name,omitempty"`
	Description   string `json:"description,omitempty"`
	Schedule      string `json:"schedule,omitempty"`
	RequireInvite *bool  `json:"require_invite,omitempty"`
	MaxMembers    *int   `json:"max_members,omitempty"`
}

// JoinClazzRequest 加入班级请求
type JoinClazzRequest struct {
	InviteCode *string `form:"invite_code,omitempty"`
}

// AddClazzMemberRequest 添加课程班级成员请求
type AddClazzMemberRequest struct {
	MemberID string `json:"member_id" binding:"required"`
}

// RemoveClazzMembersRequest 批量移除课程班级成员请求 (内部使用)
type RemoveClazzMembersRequest struct {
	ClazzID   string   `json:"clazz_id" binding:"required"`
	MemberIDs []string `json:"member_ids" binding:"required"`
}

// MajorClassIdsRequest 专业班级ID请求 (内部使用)
type MajorClassIdsRequest struct {
	MajorClassIDs []string `json:"major_class_ids" binding:"required"`
}

// GetClazzResponse 获取班级响应
type GetClazzResponse struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Name          string               `bson:"name" json:"name" binding:"required"`
	Description   string               `bson:"description,omitempty" json:"description,omitempty"`
	CourseId      primitive.ObjectID   `bson:"course_id" json:"course_id" binding:"required"`
	Schedule      string               `bson:"schedule,omitempty" json:"schedule,omitempty"`
	Teachers      []UserProfile        `bson:"-" json:"teachers,omitempty"` // 教师完整信息列表
	RequireInvite bool                 `bson:"require_invite" json:"require_invite"`
	MaxMembers    int                  `bson:"max_members,omitempty" json:"max_members,omitempty"`
	AddNums       int                  `bson:"add_nums" json:"add_nums"`
	Status        ClassStatus          `bson:"status" json:"status"`
	CTime         time.Time            `bson:"ctime" json:"ctime"`
	MajorClassIDs []primitive.ObjectID `bson:"major_class_ids,omitempty" json:"major_class_ids,omitempty"` // 关联的专业班级ID数组
}

// AddClazzTeachersRequest 班级添加教师请求
type AddClazzTeachersRequest struct {
	TeacherIds []string `json:"teacher_ids" binding:"required"`
}

// BindTeacherToClazzesRequest 教师绑定多个班级请求
type BindTeacherToClazzesRequest struct {
	TeacherId string   `json:"teacher_id" binding:"required"`
	ClazzIds  []string `json:"clazz_ids" binding:"required"`
}

// RemoveClazzTeachersRequest 班级移除教师请求 (已废弃，改为路径参数)
// type RemoveClazzTeachersRequest struct {
// 	ClazzId    string   `json:"clazz_id" binding:"required"`
// 	TeacherIds []string `json:"teacher_ids" binding:"required"`
// }

// ==================== 任务模块请求 ====================

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Title       string               `json:"title" binding:"required"`
	Description string               `json:"description,omitempty"`
	Type        TaskType             `json:"type" binding:"required"`
	StartTime   time.Time            `json:"start_time" binding:"required"`
	EndTime     time.Time            `json:"end_time" binding:"required"`
	RelationIDs []primitive.ObjectID `json:"relation_ids,omitempty"`
}

// AddTaskRequest 添加任务请求
type AddTaskRequest struct {
	CourseId    string     `json:"course_id" binding:"required"`
	Title       string     `json:"title" binding:"required"`
	Description string     `json:"description,omitempty"`
	Type        TaskType   `json:"type" binding:"required"`
	StartTime   time.Time  `json:"start_time" binding:"required"` // 必须设置开始时间
	EndTime     *time.Time `json:"end_time"`                      // 如果不设置则没有结束时间
	RelationIDs []string   `bson:"relation_ids,omitempty" json:"relation_ids,omitempty"`
	ClazzIds    []string   `json:"clazz_ids" binding:"required"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	CourseId    string     `bson:"course_id" json:"course_id,omitempty" binding:"required"`
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Type        *TaskType  `json:"type,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	RelationIDs *[]string  `bson:"relation_ids,omitempty" json:"relation_ids,omitempty"`
}

// FinishTaskRequest 完成任务请求
type FinishTaskRequest struct {
	RelationID string `bson:"relation_id" json:"relation_id" binding:"required"`
}

// AddTaskRelationIdsRequest 添加任务关系ID请求
type AddTaskRelationIdsRequest struct {
	RelationIDs []string `json:"relation_ids" binding:"required"`
}

// RemoveTaskRelationIdsRequest 删除任务关系ID请求
type RemoveTaskRelationIdsRequest struct {
	RelationIDs []string `json:"relation_ids" binding:"required"`
}

// CopyTaskToClassRequest 复制任务到班级请求
type CopyTaskToClassRequest struct {
	SourceClassID string `json:"source_class_id" binding:"required"`
	TargetClassID string `json:"target_class_id" binding:"required"`
}

// CheckTaskCompletionResponse 检查任务完成情况响应
type CheckTaskCompletionResponse struct {
	Completed         bool       `json:"completed"`              // 是否已完成
	CompletedAt       *time.Time `json:"completed_at,omitempty"` // 完成时间（如果已完成）
	TotalQuestions    int        `json:"total_questions"`        // 总题目数
	FinishedQuestions int        `json:"finished_questions"`     // 已完成题目数
}

// PageQueryTaskCompletionRequest 分页查询任务完成情况请求
type PageQueryTaskCompletionRequest struct {
	PageNum  *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize *int64  `json:"page_size,omitempty" form:"page_size"`
	RealName *string `json:"real_name,omitempty" form:"real_name"`
	UserID   *string `json:"user_id,omitempty" form:"user_id"`
}

// PageQueryTaskCompletionResponse 分页查询任务完成情况响应
type PageQueryTaskCompletionResponse struct {
	Total      int64                    `json:"total" example:"100"`
	PageNum    int64                    `json:"page_num" example:"1"`
	PageSize   int64                    `json:"page_size" example:"10"`
	Completion []TaskCompletionResponse `json:"completion"`
}

// TaskCompletionResponse 任务完成情况响应
type TaskCompletionResponse struct {
	UserID            primitive.ObjectID `json:"user_id" swaggertype:"string"`
	RealName          string             `json:"real_name"`
	StudentID         string             `json:"student_id"`
	Completed         bool               `json:"completed"`              // 是否已完成
	CompletedAt       *time.Time         `json:"completed_at,omitempty"` // 完成时间（如果已完成）
	TotalQuestions    int                `json:"total_questions"`        // 总题目数
	FinishedQuestions int                `json:"finished_questions"`     // 已完成题目数
}

// PageQueryUserTasksRequest 分页查询用户任务列表请求
type PageQueryUserTasksRequest struct {
	PageNum  *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize *int64  `json:"page_size,omitempty" form:"page_size"`
	State    *int    `json:"state,omitempty" form:"state"`       // 0: 未完成, 1: 已完成
	Type     *int8   `json:"type,omitempty" form:"type"`         // 任务类型：1-题单，2-视频
	ClazzID  *string `json:"clazz_id,omitempty" form:"clazz_id"` // 班级ID，可选
	Status   *int8   `json:"status,omitempty" form:"status"`     // 任务状态：0-未开始, 1-进行中, 2-已结束
}

// PageQueryUserTasksResponse 分页查询用户任务列表响应
type PageQueryUserTasksResponse struct {
	Total    int64          `json:"total" example:"100"`
	PageNum  int64          `json:"page_num" example:"1"`
	PageSize int64          `json:"page_size" example:"10"`
	Tasks    []TaskResponse `json:"tasks"`
}

// ==================== 题目详情响应模型 ====================

// ProblemDetailResponse 题目详情聚合响应
type ProblemDetailResponse struct {
	Problem     *Problem   `json:"problem"`      // 题目基本信息
	SampleCases []TestCase `json:"sample_cases"` // 示例测试用例列表
}

// ==================== 题目查询条件模型 ====================

// ProblemQueryCondition 题目查询条件 (内部使用)
type ProblemQueryCondition struct {
	Page           int                 `json:"page"`
	PageSize       int                 `json:"page_size"`
	Difficulty     ProblemDifficulty   `json:"difficulty"`
	Tags           []string            `json:"tags"`
	IncludePrivate bool                `json:"include_private"`
	Role           UserRole            `json:"role"`
	UserID         *primitive.ObjectID `json:"user_id"`
	// 注意：Status和CreatedBy筛选将在后续版本中支持
	// Status         ProblemStatus      `json:"status"`      // 管理员端专用（待实现）
	// CreatedBy      *primitive.ObjectID `json:"created_by"`  // 管理员端专用（待实现）
}

// ==================== 提交状态响应模型 ====================

// SubmitStatusResponse 轻量级提交状态响应
type SubmitStatusResponse struct {
	ID        primitive.ObjectID `json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"` // 提交ID
	Status    SubmitStatus       `json:"status" example:"accepted"`                                  // 当前状态
	Progress  *JudgeProgress     `json:"progress"`                                                   // 判题进度（可选）
	Message   string             `json:"message" example:"判题完成"`                                     // 状态描述信息
	UpdatedAt time.Time          `json:"updated_at" example:"2024-10-26T10:00:00Z"`                  // 最后更新时间

	// 完成后的基本结果信息（避免返回完整详细结果）
	TimeUsed   *int `json:"time_used,omitempty" example:"150"`    // 时间使用(ms)
	MemoryUsed *int `json:"memory_used,omitempty" example:"1024"` // 内存使用(KB)
}

// JudgeProgress 判题进度信息
type JudgeProgress struct {
	CurrentTestCase int `json:"current_test_case" example:"3"` // 当前测试用例索引（从1开始）
	TotalTestCases  int `json:"total_test_cases" example:"5"`  // 总测试用例数
	Percentage      int `json:"percentage" example:"60"`       // 完成百分比 (0-100)
}

// ==================== WebSocket消息模型 ====================

// WSMessage WebSocket消息基础结构
type WSMessage struct {
	Type      string      `json:"type"`      // 消息类型
	SubmitID  string      `json:"submit_id"` // 提交ID
	Data      interface{} `json:"data"`      // 消息数据
	Timestamp time.Time   `json:"timestamp"` // 消息时间戳
}

// WSStatusUpdate WebSocket状态更新消息
type WSStatusUpdate struct {
	Status   SubmitStatus   `json:"status"`           // 新状态
	Progress *JudgeProgress `json:"progress"`         // 进度信息（可选）
	Message  string         `json:"message"`          // 状态描述
	Result   *WSJudgeResult `json:"result,omitempty"` // 完成时的结果摘要
}

// WSJudgeResult WebSocket判题结果摘要（不包含详细测试用例）
type WSJudgeResult struct {
	Status       SubmitStatus `json:"status"`
	TimeUsed     int          `json:"time_used"`               // 时间使用(ms)
	MemoryUsed   int          `json:"memory_used"`             // 内存使用(KB)
	PassedCases  int          `json:"passed_cases"`            // 通过的测试用例数
	TotalCases   int          `json:"total_cases"`             // 总测试用例数
	CompileError string       `json:"compile_error,omitempty"` // 编译错误（如果有）
}

// ==================== 学生班级模块请求 ====================

// GetStudentClassesRequest 获取学生班级请求
type GetStudentClassesRequest struct {
	StudentID string `json:"student_id" binding:"required"`
}

// GetClassStudentsRequest 获取班级学生请求
type GetClassStudentsRequest struct {
	ClassID string `json:"class_id" binding:"required"`
}

// ==================== 通用响应模型 ====================

// SuccessResponse 通用成功响应
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message" example:"操作成功"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse 通用错误响应
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Error   string `json:"error" example:"请求参数错误"`
	Message string `json:"message" example:"缺少必需的参数"`
}

// ==================== 列表响应模型 ====================

// ProblemListResponse 题目列表响应
type ProblemListResponse struct {
	Problems   []Problem `json:"problems"`
	Total      int64     `json:"total" example:"100"`
	Page       int       `json:"page" example:"1"`
	PageSize   int       `json:"page_size" example:"10"`
	TotalPages int       `json:"total_pages" example:"10"`
}

// SubmitListResponse 提交列表响应
type SubmitListResponse struct {
	Submits    []Submit `json:"submits"`
	Total      int64    `json:"total" example:"50"`
	Page       int      `json:"page" example:"1"`
	PageSize   int      `json:"page_size" example:"10"`
	TotalPages int      `json:"total_pages" example:"5"`
}

// CourseListResponse 课程列表响应
type CourseListResponse struct {
	Courses []Course `json:"courses"`
	Total   int64    `json:"total" example:"20"`
}

// TestCaseListResponse 测试用例列表响应
type TestCaseListResponse struct {
	TestCases []TestCase `json:"test_cases"`
	Total     int64      `json:"total" example:"15"`
}

// ==================== 创建/更新响应模型 ====================

// CreateResponse 创建成功响应
type CreateResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"创建成功"`
	ID      string `json:"id" example:"507f1f77bcf86cd799439011"`
}

// UpdateResponse 更新成功响应
type UpdateResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"更新成功"`
}

// DeleteResponse 删除成功响应
type DeleteResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"删除成功"`
}

// ==================== 认证响应模型 ====================

// LoginResponse 登录响应
type LoginResponse struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message" example:"登录成功"`
	Token   string      `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	User    UserProfile `json:"user"`
}

// RegisterResponse 注册响应
type RegisterResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"注册成功"`
	UserID  string `json:"user_id" example:"507f1f77bcf86cd799439011"`
}

// ==================== 统计响应模型 ====================

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	UserID         string `json:"user_id" example:"507f1f77bcf86cd799439011"`
	Username       string `json:"username" example:"student1"`
	TotalSubmits   int    `json:"total_submits" example:"45"`
	AcceptedCount  int    `json:"accepted_count" example:"20"`
	AcceptanceRate string `json:"acceptance_rate" example:"44.4%"`
	SolvedProblems int    `json:"solved_problems" example:"18"`
	Ranking        int    `json:"ranking" example:"15"`
}

// SystemStatsResponse 系统统计响应
type SystemStatsResponse struct {
	TotalUsers     int `json:"total_users" example:"150"`
	TotalProblems  int `json:"total_problems" example:"80"`
	TotalSubmits   int `json:"total_submits" example:"2500"`
	TotalCourses   int `json:"total_courses" example:"12"`
	TotalClasses   int `json:"total_classes" example:"25"`
	ActiveUsers24h int `json:"active_users_24h" example:"35"`
	NewUsersToday  int `json:"new_users_today" example:"3"`
	SubmitsToday   int `json:"submits_today" example:"120"`
}

// DifficultyStatsResponse 难度统计响应
type DifficultyStatsResponse struct {
	Easy   DifficultyStatItem `json:"easy"`
	Medium DifficultyStatItem `json:"medium"`
	Hard   DifficultyStatItem `json:"hard"`
}

// DifficultyStatItem 难度统计项
type DifficultyStatItem struct {
	Count      int     `json:"count" example:"25"`
	Percentage float64 `json:"percentage" example:"31.25"`
}

// ==================== 代码运行响应模型 ====================

// RunCodeResponse 代码运行响应
type RunCodeResponse struct {
	Success    bool     `json:"success" example:"true"`
	Status     string   `json:"status" example:"accepted"`
	Output     string   `json:"output" example:"Hello World!"`
	Error      string   `json:"error,omitempty" example:""`
	TimeUsed   int      `json:"time_used" example:"126"`
	MemoryUsed int      `json:"memory_used" example:"1024"`
	TestCases  []string `json:"test_cases,omitempty"`
}

// SearchProblemsResponse 搜索题目响应
type SearchProblemsResponse struct {
	Problems   []Problem `json:"problems"`
	Total      int       `json:"total" example:"15"`
	Query      string    `json:"query" example:"二分查找"`
	SearchTime string    `json:"search_time" example:"0.05s"`
}

// DailyProblemResponse 每日一题响应
type DailyProblemResponse struct {
	Problem    Problem `json:"problem"`
	Date       string  `json:"date" example:"2024-10-26"`
	IsFinished bool    `json:"is_finished" example:"false"`
	Progress   string  `json:"progress" example:"今日已有15位同学完成"`
}
