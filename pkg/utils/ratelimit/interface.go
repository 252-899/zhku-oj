/*
@Author: omenkk7
@Date: 2025/10/7
@Description: 限流接口定义
*/

package ratelimit

import (
	"context"
	"time"
)

// Limiter 限流器接口
type Limiter interface {
	// Allow 检查是否允许通过
	// key: 限流键（通常是用户ID或IP）
	// limit: 时间窗口内最大请求数
	// window: 时间窗口大小
	// 返回: 是否允许, 错误信息
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
	
	// Reset 重置指定key的限流状态（用于测试或特殊情况）
	Reset(ctx context.Context, key string) error
	
	// Close 关闭限流器，释放资源
	Close() error
}

// LimiterType 限流器类型
type LimiterType string

const (
	LimiterTypeMemory LimiterType = "memory" // 内存限流器
	LimiterTypeRedis  LimiterType = "redis"  // Redis限流器
)

