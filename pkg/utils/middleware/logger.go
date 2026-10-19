/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: logger.go
@Description: 日志中间件
*/

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"time"
)

// LoggerHandler 日志处理中间件
func LoggerHandler(logger logrus.FieldLogger, skipPaths []string) gin.HandlerFunc {
	var skip map[string]struct{}
	if length := len(skipPaths); length > 0 {
		skip = make(map[string]struct{}, length)
		for _, path := range skipPaths {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 跳过指定路径
		if _, ok := skip[path]; ok {
			return
		}

		// 计算处理时间
		latency := time.Since(start)

		// 记录日志
		logger.WithFields(logrus.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"query":      raw,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"latency":    latency,
		}).Info("HTTP Request")
	}
}
