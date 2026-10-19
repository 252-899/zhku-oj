/*
@Author: omenkk7
@Date: 2025/10/7
@Description: 基于内存的限流器实现（使用 golang.org/x/time/rate）
*/

package ratelimit

import (
	"context"
	"sync"
	"time"
	
	"golang.org/x/time/rate"
)

// MemoryLimiter 基于内存的限流器（使用令牌桶算法）
type MemoryLimiter struct {
	limiters sync.Map // map[string]*rate.Limiter
	mu       sync.Mutex
	
	// 清理过期限流器的配置
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

// limiterEntry 限流器条目（用于存储创建时间）
type limiterEntry struct {
	limiter   *rate.Limiter
	lastUsed  time.Time
	limit     int
	window    time.Duration
}

// NewMemoryLimiter 创建内存限流器
func NewMemoryLimiter() *MemoryLimiter {
	ml := &MemoryLimiter{
		cleanupInterval: 10 * time.Minute, // 每10分钟清理一次
		stopCleanup:     make(chan struct{}),
	}
	
	// 启动清理协程
	go ml.cleanup()
	
	return ml
}

// Allow 检查是否允许通过
func (ml *MemoryLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// 获取或创建限流器
	limiterEntry := ml.getLimiter(key, limit, window)
	
	// 使用令牌桶算法检查
	// 注意: rate.Limiter 使用的是 QPS 而不是时间窗口
	// 我们需要将 limit/window 转换为 QPS
	return limiterEntry.limiter.Allow(), nil
}

// Reset 重置指定key的限流状态
func (ml *MemoryLimiter) Reset(ctx context.Context, key string) error {
	ml.limiters.Delete(key)
	return nil
}

// Close 关闭限流器
func (ml *MemoryLimiter) Close() error {
	close(ml.stopCleanup)
	return nil
}

// getLimiter 获取或创建限流器
func (ml *MemoryLimiter) getLimiter(key string, limit int, window time.Duration) *limiterEntry {
	// 尝试从缓存中获取
	if v, ok := ml.limiters.Load(key); ok {
		entry := v.(*limiterEntry)
		entry.lastUsed = time.Now()
		
		// 如果限流配置发生变化，需要重新创建
		if entry.limit != limit || entry.window != window {
			ml.limiters.Delete(key)
		} else {
			return entry
		}
	}
	
	// 创建新的限流器
	// 计算每秒允许的请求数（QPS）
	qps := float64(limit) / window.Seconds()
	
	// 使用令牌桶算法
	// r: 令牌生成速率（每秒生成 qps 个令牌）
	// b: 桶容量（最多存储 limit 个令牌）
	limiter := rate.NewLimiter(rate.Limit(qps), limit)
	
	entry := &limiterEntry{
		limiter:  limiter,
		lastUsed: time.Now(),
		limit:    limit,
		window:   window,
	}
	
	ml.limiters.Store(key, entry)
	return entry
}

// cleanup 定期清理过期的限流器（释放内存）
func (ml *MemoryLimiter) cleanup() {
	ticker := time.NewTicker(ml.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			ml.limiters.Range(func(key, value interface{}) bool {
				entry := value.(*limiterEntry)
				// 如果超过 2 倍清理间隔未使用，则删除
				if now.Sub(entry.lastUsed) > 2*ml.cleanupInterval {
					ml.limiters.Delete(key)
				}
				return true
			})
		case <-ml.stopCleanup:
			return
		}
	}
}

