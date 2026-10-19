// catalog.go 定义 OJ AI 工具目录，供工具构建、白名单和提示词说明复用。
package oj

// ToolCategory 表示工具所属业务类别。
type ToolCategory string

const (
	// ToolCategoryUser 表示用户资料相关工具。
	ToolCategoryUser ToolCategory = "user"
	// ToolCategoryClass 表示班级和任务相关工具。
	ToolCategoryClass ToolCategory = "class"
	// ToolCategoryProblem 表示题库相关工具。
	ToolCategoryProblem ToolCategory = "problem"
)

// ToolDependency 表示工具运行所需的业务依赖。
type ToolDependency string

const (
	// ToolDependencyUserReader 表示工具需要 UserReader。
	ToolDependencyUserReader ToolDependency = "user_reader"
	// ToolDependencyClazzReader 表示工具需要 ClazzReader。
	ToolDependencyClazzReader ToolDependency = "clazz_reader"
	// ToolDependencyProblemReader 表示工具需要 ProblemReader。
	ToolDependencyProblemReader ToolDependency = "problem_reader"
)

// ToolSpec 描述一个 OJ AI 工具的稳定元信息。
type ToolSpec struct {
	Name        string
	Description string
	Category    ToolCategory
	Dependency  ToolDependency
	ReadOnly    bool
}

// Catalog 返回当前已实现的 OJ AI 工具目录。
func Catalog() []ToolSpec {
	return []ToolSpec{
		{
			Name:        ToolGetUserProfile,
			Description: "按 user_id 或 student_id 查询安全用户资料，不返回密码、手机号、邮箱等敏感字段。",
			Category:    ToolCategoryUser,
			Dependency:  ToolDependencyUserReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolGetClassInfo,
			Description: "查询当前操作者可见的班级元数据，包括班级名称、容量、教师和状态。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolListTeacherClasses,
			Description: "查询当前教师可见的班级列表，可按班级名称关键词过滤，用于把自然语言班级名解析为 class_id。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolListClassStudents,
			Description: "查询某个班级的学生安全资料列表。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolListClassTasks,
			Description: "查询某个班级中当前操作者可见的任务列表。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolGetTaskDetail,
			Description: "查询单个任务详情，包括关联题目摘要。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolGetTaskCompletion,
			Description: "分页查询某个班级任务的完成情况，包括已完成、未完成和进度信息。",
			Category:    ToolCategoryClass,
			Dependency:  ToolDependencyClazzReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolSearchProblems,
			Description: "按关键词、难度、标签搜索题库题目，用于题目推荐或作业候选题查找。",
			Category:    ToolCategoryProblem,
			Dependency:  ToolDependencyProblemReader,
			ReadOnly:    true,
		},
		{
			Name:        ToolGetProblemDetail,
			Description: "查询单个题目的详情和样例测试用例。",
			Category:    ToolCategoryProblem,
			Dependency:  ToolDependencyProblemReader,
			ReadOnly:    true,
		},
	}
}

// AvailableCatalog 根据当前 Provider 配置返回可用工具目录。
func (p *Provider) AvailableCatalog() []ToolSpec {
	if p == nil {
		return nil
	}
	items := make([]ToolSpec, 0, len(Catalog()))
	for _, item := range Catalog() {
		if p.hasDependency(item.Dependency) {
			items = append(items, item)
		}
	}
	return items
}

// hasDependency 判断当前 Provider 是否具备某个工具需要的业务依赖。
func (p *Provider) hasDependency(dep ToolDependency) bool {
	if p == nil {
		return false
	}
	switch dep {
	case ToolDependencyUserReader:
		return p.deps.UserReader != nil
	case ToolDependencyClazzReader:
		return p.deps.ClazzReader != nil
	case ToolDependencyProblemReader:
		return p.deps.ProblemReader != nil
	default:
		return false
	}
}
