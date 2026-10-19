/*
@Author: omenkk7
@Date: 2025/10/7
@Description: 限流中间件
*/

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"
	
	"github.com/gin-gonic/gin"
	"zk-code-arena-server/conf"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/ratelimit"
)

// RateLimitRule 限流规则
type RateLimitRule struct {
	Name   string        // 规则名称
	Limit  int           // 时间窗口内最大请求数
	Window time.Duration // 时间窗口
}

var (
	// 全局限流器实例
	globalLimiter ratelimit.Limiter
	
	// 预定义的限流规则
	CodeRunRule = RateLimitRule{
		Name:   "code_run",
		Limit:  3,
		Window: time.Minute,
	}
	
	SubmitRule = RateLimitRule{
		Name:   "submit",
		Limit:  10,
		Window: time.Minute,
	}
)

// InitRateLimiter 初始化限流器（在服务启动时调用）
func InitRateLimiter() error {
	logger := utils.GetLogger(context.Background())
	
	if !conf.Config.RateLimit.Enabled {
		logger.Info("限流功能已禁用")
		return nil
	}
	
	limiterType := ratelimit.LimiterType(conf.Config.RateLimit.Type)
	
	switch limiterType {
	case ratelimit.LimiterTypeMemory:
		globalLimiter = ratelimit.NewMemoryLimiter()
		logger.Info("使用内存限流器")
		
		// 更新预定义规则（从配置读取）
		if conf.Config.RateLimit.CodeRun.Limit > 0 {
			CodeRunRule.Limit = conf.Config.RateLimit.CodeRun.Limit
			CodeRunRule.Window = time.Duration(conf.Config.RateLimit.CodeRun.Window) * time.Second
		}
		
		if conf.Config.RateLimit.Submit.Limit > 0 {
			SubmitRule.Limit = conf.Config.RateLimit.Submit.Limit
			SubmitRule.Window = time.Duration(conf.Config.RateLimit.Submit.Window) * time.Second
		}
		
		return nil
		
	case ratelimit.LimiterTypeRedis:
		// TODO: 未来可以实现 Redis 限流器
		return fmt.Errorf("Redis 限流器暂未实现")
		
	default:
		return fmt.Errorf("不支持的限流器类型: %s", limiterType)
	}
}

// CloseRateLimiter 关闭限流器（在服务关闭时调用）
func CloseRateLimiter() error {
	if globalLimiter != nil {
		return globalLimiter.Close()
	}
	return nil
}

// RateLimitMiddleware 限流中间件工厂函数
func RateLimitMiddleware(rule RateLimitRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果限流功能未启用，直接放行
		if !conf.Config.RateLimit.Enabled || globalLimiter == nil {
			c.Next()
			return
		}
		
		// 获取用户ID作为限流key
		userID, exists := c.Get("user_id")
		if !exists {
			// 如果未登录，使用 IP 地址作为限流key
			userID = c.ClientIP()
		}
		
		// 构造限流key
		key := fmt.Sprintf("rate_limit:%s:%v", rule.Name, userID)
		
		// 检查是否允许通过
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		allowed, err := globalLimiter.Allow(ctx, key, rule.Limit, rule.Window)
		if err != nil {
			// 限流器出错时，记录日志但允许通过（优雅降级）
			logger := utils.GetLogger(c.Request.Context())
			logger.Errorf("限流检查失败: %v", err)
			c.Next()
			return
		}
		
		if !allowed {
			// 超过限流，返回 429 状态码
			c.JSON(http.StatusTooManyRequests, utils.Response{
				Code:    http.StatusTooManyRequests,
				Message: fmt.Sprintf("请求过于频繁，请在 %d 秒后重试", int(rule.Window.Seconds())),
			})
			c.Abort()
			return
		}
		
		// 允许通过
		c.Next()
	}
}

// CodeRunRateLimitMiddleware 代码运行限流中间件
func CodeRunRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(CodeRunRule)
}

// SubmitRateLimitMiddleware 代码提交限流中间件
func SubmitRateLimitMiddleware() gin.HandlerFunc {
	return RateLimitMiddleware(SubmitRule)
}

