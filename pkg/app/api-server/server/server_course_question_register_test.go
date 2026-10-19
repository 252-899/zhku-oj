/*
@Author: Codex
@Date: 2026/05/13
@Description: 课程题型路由注册测试，确保选择题与填空题接口已经独立挂载到 v1 路由树
*/

package server

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterChoiceQuestionRoutes 验证选择题路由已经通过独立注册函数挂载。
// 这里直接检查 gin 路由表，避免测试耦合到鉴权中间件的运行细节。
func TestRegisterChoiceQuestionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	server := &Server{}
	server.RegisterChoiceQuestion(engine.Group("/api/v1"))

	assertRouteRegistered(t, engine, "POST", "/api/v1/choice-questions")
	assertRouteRegistered(t, engine, "GET", "/api/v1/choice-questions/:questionId")
	assertRouteRegistered(t, engine, "PUT", "/api/v1/choice-questions/:questionId")
	assertRouteRegistered(t, engine, "DELETE", "/api/v1/choice-questions/:questionId")
}

// TestRegisterBlankQuestionRoutes 验证填空题路由已经通过独立注册函数挂载。
func TestRegisterBlankQuestionRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()

	server := &Server{}
	server.RegisterBlankQuestion(engine.Group("/api/v1"))

	assertRouteRegistered(t, engine, "POST", "/api/v1/blank-questions")
	assertRouteRegistered(t, engine, "GET", "/api/v1/blank-questions/:questionId")
	assertRouteRegistered(t, engine, "PUT", "/api/v1/blank-questions/:questionId")
	assertRouteRegistered(t, engine, "DELETE", "/api/v1/blank-questions/:questionId")
}

// assertRouteRegistered 校验目标方法和路径已经进入 gin 路由表。
func assertRouteRegistered(t *testing.T, engine *gin.Engine, method string, path string) {
	t.Helper()
	for _, route := range engine.Routes() {
		if route.Method == method && route.Path == path {
			return
		}
	}
	t.Fatalf("expected route %s %s to be registered", method, path)
}
