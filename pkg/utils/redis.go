package utils

import (
	"context"
	"fmt"
	"time"
	"zk-code-arena-server/conf"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

// InitRedis 初始化 Redis 连接
func InitRedis() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建Redis客户端
	addr := fmt.Sprintf("%s:%d", conf.Config.Redis.Host, conf.Config.Redis.Port)
	options := &redis.Options{
		Addr: addr,
		DB:   conf.Config.Redis.DB,
	}
	if conf.Config.Redis.PoolSize > 0 {
		options.PoolSize = conf.Config.Redis.PoolSize
	}
	if conf.Config.Redis.Password != "" {
		options.Password = conf.Config.Redis.Password
	}
	RedisClient = redis.NewClient(options)

	// 测试连接
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return err
	}

	return nil
}

// GetRedisClient 获取Redis客户端
func GetRedisClient() *redis.Client {
	return RedisClient
}

// CloseRedis 关闭Redis连接
func CloseRedis() error {
	if RedisClient != nil {
		return RedisClient.Close()
	}
	return nil
}
