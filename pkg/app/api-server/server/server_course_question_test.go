/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程题目接口权限解析测试，约束读写链路的角色边界
*/

package server

import (
	"net/http/httptest"
	"testing"

	"zk-code-arena-server/pkg/models"

	"github.com/gin-gonic/gin"
)

// TestParseCourseReadQuestionIdentityAllowsStudent 验证学生可进入题目详情读取链路。
// 具体是否能看到题目仍由 service 层按课程成员关系和发布状态继续裁决。
func TestParseCourseReadQuestionIdentityAllowsStudent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "courseId", Value: "507f1f77bcf86cd799439011"},
		{Key: "questionId", Value: "507f1f77bcf86cd799439012"},
	}
	c.Set("user_id", "507f1f77bcf86cd799439013")
	c.Set("role", string(models.RoleStudent))

	s := &Server{}
	_, _, _, role, ok := s.parseCourseReadQuestionIdentity(c, "questionId")
	if !ok {
		t.Fatalf("expected read identity parsing to succeed for student")
	}
	if role != models.RoleStudent {
		t.Fatalf("expected student role, got %s", role)
	}
}

// TestParseCourseWriteQuestionIdentityRejectsStudent 验证学生不能进入课程题目编辑链路。
func TestParseCourseWriteQuestionIdentityRejectsStudent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "courseId", Value: "507f1f77bcf86cd799439011"},
		{Key: "questionId", Value: "507f1f77bcf86cd799439012"},
	}
	c.Set("user_id", "507f1f77bcf86cd799439013")
	c.Set("role", string(models.RoleStudent))

	s := &Server{}
	if _, _, _, _, ok := s.parseCourseWriteQuestionIdentity(c, "questionId"); ok {
		t.Fatalf("expected write identity parsing to reject student role")
	}
	if w.Code != 403 {
		t.Fatalf("expected forbidden response, got %d", w.Code)
	}
}
