/*
@Author: omenkk7
@Date: 2025/10/9
@Description: 测试工具函数
*/

package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"zk-code-arena-server/pkg/app/api-server/server"
	"zk-code-arena-server/pkg/app/api-server/service"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
)

// SetupTestServer 创建测试服务器实例
func SetupTestServer(t *testing.T) (*server.Server, func()) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化数据库连接（测试环境）
	if err := utils.InitMongoDB(); err != nil {
		t.Fatalf("Failed to init MongoDB: %v", err)
	}

	// 初始化限流器
	if err := middleware.InitRateLimiter(); err != nil {
		t.Fatalf("Failed to init rate limiter: %v", err)
	}

	// 创建服务和服务器实例
	svc := service.NewService()
	lg := utils.GetLogger(context.Background())
	stopCh := make(chan struct{})
	opts := server.NewCmdOptions("localhost", "8080")
	s := server.NewServer(lg, svc, opts, stopCh)
	s.Init()

	// 返回清理函数
	cleanup := func() {
		middleware.CloseRateLimiter()
		s.Shutdown(context.Background())
	}

	return s, cleanup
}

// MakeRequest 发送 HTTP 请求并返回响应
func MakeRequest(
	t *testing.T,
	handler http.HandlerFunc,
	method, path string,
	body interface{},
	headers map[string]string,
) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 添加自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

// MakeGinRequest 使用 Gin 引擎发送请求
func MakeGinRequest(
	t *testing.T,
	router *gin.Engine,
	method, path string,
	body interface{},
	headers map[string]string,
) *httptest.ResponseRecorder {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// 添加自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

// ParseResponse 解析 JSON 响应
func ParseResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("Failed to unmarshal response: %v, body: %s", err, w.Body.String())
	}
}

// AssertStatusCode 断言 HTTP 状态码
func AssertStatusCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	if w.Code != expected {
		t.Errorf("Expected status code %d, got %d. Body: %s", expected, w.Code, w.Body.String())
	}
}

// AssertResponseCode 断言业务响应码
func AssertResponseCode(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	ParseResponse(t, w, &resp)

	if resp.Code != expected {
		t.Errorf("Expected response code %d, got %d. Message: %s", expected, resp.Code, resp.Message)
	}
}

// GenerateAuthToken 生成测试用的 JWT Token
func GenerateAuthToken(userID, username, role string) (string, error) {
	return middleware.GenerateToken(userID, username, role)
}

// WithAuth 创建带 Authorization 头的请求头 map
func WithAuth(token string) map[string]string {
	return map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", "测试内容"),
	}
}

// CreateTestUser 创建测试用户并返回 token
func CreateTestUser(t *testing.T, app *gin.Engine, studentID, password, role string) string {
	// 注册用户
	registerBody := map[string]interface{}{
		"student_id": studentID,
		"username":   "test_" + studentID,
		"password":   password,
		"email":      studentID + "@test.com",
		"role":       role,
	}

	w := MakeGinRequest(t, app, "POST", "/api/v1/user/register", registerBody, nil)
	AssertStatusCode(t, w, http.StatusOK)

	// 登录获取 token
	loginBody := map[string]interface{}{
		"student_id": studentID,
		"password":   password,
	}

	w = MakeGinRequest(t, app, "POST", "/api/v1/user/login", loginBody, nil)
	AssertStatusCode(t, w, http.StatusOK)

	var loginResp struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	ParseResponse(t, w, &loginResp)

	return loginResp.Data.Token
}
