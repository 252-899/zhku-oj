/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: response.go
@Description: 统一响应格式
*/

package utils

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// SuccessResponse 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "success",
		Data:    data,
	})
}

// ErrorResponse 错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequestResponse 400 错误响应
func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

// UnauthorizedResponse 401 错误响应
func UnauthorizedResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

// ForbiddenResponse 403 错误响应
func ForbiddenResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusForbidden, message)
}

// NotFoundResponse 404 错误响应
func NotFoundResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

// InternalServerErrorResponse 500 错误响应
func InternalServerErrorResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}

// ServiceUnavailableResponse 503 错误响应
func ServiceUnavailableResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusServiceUnavailable, message)
}
