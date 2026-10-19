/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: logger.go
@Description: 日志管理模块
*/

package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"zk-code-arena-server/conf"
)

var logger *logrus.Logger

// Logger 导出的日志实例，供其他包使用
var Logger *logrus.Logger

func init() {
	logger = logrus.New()
	Logger = logger // 导出给外部使用
	
	// 使用默认配置初始化日志器
	initLoggerWithDefaults()
}

// initLoggerWithDefaults 使用默认配置初始化日志器
func initLoggerWithDefaults() {
	// 设置默认日志格式
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	
	// 设置默认日志级别
	logger.SetLevel(logrus.InfoLevel)
	
	// 设置默认输出为标准输出
	logger.SetOutput(os.Stdout)
}

// InitLogger 使用配置文件初始化日志器
func InitLogger() {
	if logger == nil {
		logger = logrus.New()
		Logger = logger // 同步导出的Logger
	}
	
	// 检查配置是否已初始化
	if conf.Config == nil {
		initLoggerWithDefaults()
		return
	}
	
	// 设置日志格式
	format := "text" // 默认格式
	if conf.Config.Log.Format != "" {
		format = conf.Config.Log.Format
	}
	
	if format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}
	
	// 设置日志级别
	levelStr := "info" // 默认级别
	if conf.Config.Log.Level != "" {
		levelStr = conf.Config.Log.Level
	}
	
	level, err := logrus.ParseLevel(levelStr)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	
	// 设置输出
	output := "stdout" // 默认输出
	if conf.Config.Log.Output != "" {
		output = conf.Config.Log.Output
	}
	
	if output == "file" {
		// 文件输出
		filePath := "./logs/server.log" // 默认路径
		if conf.Config.Log.File.Path != "" {
			filePath = conf.Config.Log.File.Path
		}
		
		fileWriter := &lumberjack.Logger{
			Filename:   filePath,
			MaxSize:    100, // 默认 100MB
			MaxBackups: 3,   // 默认保留 3 个备份
			MaxAge:     28,  // 默认保留 28 天
			Compress:   true, // 默认压缩
		}
		
		// 使用配置值覆盖默认值
		if conf.Config.Log.File.MaxSize > 0 {
			fileWriter.MaxSize = conf.Config.Log.File.MaxSize
		}
		if conf.Config.Log.File.MaxBackups > 0 {
			fileWriter.MaxBackups = conf.Config.Log.File.MaxBackups
		}
		if conf.Config.Log.File.MaxAge > 0 {
			fileWriter.MaxAge = conf.Config.Log.File.MaxAge
		}
		fileWriter.Compress = conf.Config.Log.File.Compress
		
		logger.SetOutput(io.MultiWriter(os.Stdout, fileWriter))
	} else {
		// 标准输出
		logger.SetOutput(os.Stdout)
	}
}

func GetLogger(ctx context.Context) logrus.FieldLogger {
	if logger == nil {
		return logrus.New()
	}
	return logger
}

func GetLoggerWithFields(fields logrus.Fields) logrus.FieldLogger {
	if logger == nil {
		return logrus.New()
	}
	return logger.WithFields(fields)
}
