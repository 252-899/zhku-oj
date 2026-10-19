// types.go 定义教师侧 Copilot 规划阶段共享的常量和意图类型。
package teachercopilot

const (
	// AssistantType 是教师侧 Copilot 的标准业务类型。
	AssistantType = "teacher_copilot"

	// ContextClassID 是入口层已知的班级 ID。它是可选线索，不是主 Agent 的唯一依赖。
	ContextClassID = "class_id"
	// ContextClassName 是入口层或前端可传入的班级名称线索。
	ContextClassName = "class_name"
	// ContextTaskID 是入口层已知的任务 ID。它是可选线索，不是主 Agent 的唯一依赖。
	ContextTaskID = "task_id"
	// ContextTaskName 是入口层或前端可传入的任务名称线索。
	ContextTaskName = "task_name"

	// RoleClassAnalyst 是负责解析班级、任务范围和班级概况的子 Agent 角色。
	RoleClassAnalyst = "class_analyst"
	// RoleAssignmentReviewer 是负责作业事实分析的子 Agent 角色。
	RoleAssignmentReviewer = "assignment_reviewer"
	// RoleReportWriter 是负责讲评建议生成的子 Agent 角色。
	RoleReportWriter = "report_writer"
)

// Intent 表示教师问题的高层业务意图。
type Intent string

const (
	// IntentAssignmentReview 表示“分析班级作业并生成讲评建议”。
	IntentAssignmentReview Intent = "assignment_review"
	// IntentClassOverview 表示“查看或总结班级整体情况”。
	IntentClassOverview Intent = "class_overview"
	// IntentStudentDiagnosis 表示“诊断学生学习情况”。
	IntentStudentDiagnosis Intent = "student_diagnosis"
	// IntentProblemRecommendation 表示“推荐题目或作业内容”。
	IntentProblemRecommendation Intent = "problem_recommendation"
	// IntentUnsupported 表示当前版本暂不支持的问题类型。
	IntentUnsupported Intent = "unsupported"
)
