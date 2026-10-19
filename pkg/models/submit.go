/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: submit.go
@Description: 提交数据模型
*/

package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// SubmitStatus 提交状态
type SubmitStatus string

const (
	StatusPending      SubmitStatus = "pending"       // 等待中
	StatusRunning      SubmitStatus = "running"       // 运行中
	StatusAccepted     SubmitStatus = "accepted"      // 通过
	StatusWrongAnswer  SubmitStatus = "wrong_answer"  // 答案错误
	StatusTimeLimit    SubmitStatus = "time_limit"    // 超时
	StatusMemoryLimit  SubmitStatus = "memory_limit"  // 内存超限
	StatusRuntimeError SubmitStatus = "runtime_error" // 运行时错误
	StatusCompileError SubmitStatus = "compile_error" // 编译错误
	StatusSystemError  SubmitStatus = "system_error"  // 系统错误
)

// Language 编程语言
type Language string

const (
	LangC      Language = "c"
	LangCpp    Language = "cpp"
	LangJava   Language = "java"
	LangPython Language = "python"
	LangGo     Language = "go"
)

// Submit 提交模型
type Submit struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	ProblemID primitive.ObjectID `bson:"problem_id" json:"problem_id" binding:"required" swaggertype:"string" example:"507f1f77bcf86cd799439012"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id" binding:"required" swaggertype:"string" example:"507f1f77bcf86cd799439013"`
	Code      string             `bson:"code" json:"code" binding:"required" example:"#include <iostream>\nusing namespace std;\nint main() {\n    cout << \"Hello World!\" << endl;\n    return 0;\n}"`
	Language  Language           `bson:"language" json:"language" binding:"required" example:"cpp"`
	Status    SubmitStatus       `bson:"status" json:"status" example:"accepted"`
	Result    *JudgeResult       `bson:"result" json:"result"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at" example:"2024-10-26T10:00:00Z"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at" example:"2024-10-26T10:00:00Z"`
}

// JudgeResult 评测结果
type JudgeResult struct {
	Status       SubmitStatus `json:"status" example:"accepted"`
	TimeUsed     int          `json:"time_used" example:"126"`     // 时间使用(ms)
	MemoryUsed   int          `json:"memory_used" example:"1024"`  // 内存使用(KB)
	CompileError string       `json:"compile_error" example:""`    // 编译错误信息
	RuntimeError string       `json:"runtime_error" example:""`    // 运行时错误信息
	TestResults  []TestResult `json:"test_results"`                // 测试用例结果
}

// TestResult 测试用例结果
type TestResult struct {
	TestCaseID primitive.ObjectID `json:"test_case_id"`
	Status     SubmitStatus       `json:"status"`
	TimeUsed   int                `json:"time_used"`   // ms
	MemoryUsed int                `json:"memory_used"` // KB
	
	// 是否为示例用例（用于判断是否返回详细信息）
	IsSample   bool               `json:"is_sample"`
	
	// 仅示例用例返回以下字段（隐藏用例不返回详细输出）
	Output     string             `json:"output,omitempty"`   // 实际输出
	Expected   string             `json:"expected,omitempty"` // 期望输出
	Error      string             `json:"error,omitempty"`    // 错误信息
}

// SubmitList 提交列表项
type SubmitList struct {
	ID         primitive.ObjectID `json:"id"`
	ProblemID  primitive.ObjectID `json:"problem_id"`
	UserID     primitive.ObjectID `json:"user_id"`
	Language   Language           `json:"language"`
	Status     SubmitStatus       `json:"status"`
	TimeUsed   int                `json:"time_used"`
	MemoryUsed int                `json:"memory_used"`
	CreatedAt  time.Time          `json:"created_at"`
}
