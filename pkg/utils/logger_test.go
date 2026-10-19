/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: logger_test.go
@Description: 日志模块测试
*/

package utils

import (
	"context"
	"testing"
)

func TestGetLogger(t *testing.T) {
	// 测试获取日志器
	ctx := context.Background()
	logger := GetLogger(ctx)
	
	if logger == nil {
		t.Error("GetLogger 返回了 nil")
	}
}

func TestGetLoggerWithFields(t *testing.T) {
	// 测试获取带字段的日志器
	fields := map[string]interface{}{
		"test": "value",
	}
	
	logger := GetLoggerWithFields(fields)
	
	if logger == nil {
		t.Error("GetLoggerWithFields 返回了 nil")
	}
}

func TestInitLogger(t *testing.T) {
	// 测试初始化日志器（使用默认配置）
	InitLogger()
	
	// 验证日志器已初始化
	ctx := context.Background()
	logger := GetLogger(ctx)
	
	if logger == nil {
		t.Error("InitLogger 后 GetLogger 返回了 nil")
	}
}
