// roles.go 定义教师侧 Copilot 第一阶段可用的 Agent 角色目录。
package teachercopilot

import "zk-code-arena-server/pkg/ai/multiagent"

// DefaultRoles 返回教师侧主 Agent 可规划的第一批角色。
func DefaultRoles() []multiagent.RoleSpec {
	return []multiagent.RoleSpec{
		{
			Name:        RoleClassAnalyst,
			BaseRole:    "analyst",
			Description: "解析班级、作业任务范围，查询班级概况和任务列表。",
		},
		{
			Name:        RoleAssignmentReviewer,
			BaseRole:    "analyst",
			Description: "分析作业完成率、未完成学生、题目通过情况和异常点。",
		},
		{
			Name:        RoleReportWriter,
			BaseRole:    "writer",
			Description: "根据分析结果生成课堂讲评建议、课后补练建议和提醒文案。",
		},
	}
}
