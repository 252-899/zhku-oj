/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: errors.go
@Description: 业务错误定义
*/

package models

import "errors"

// 题目相关错误
var (
	ErrProblemNotFound           = errors.New("题目不存在")
	ErrProblemTestCaseNotFound   = errors.New("题目测试用例不存在")
	ErrProblemInfoFetchFailed    = errors.New("获取题目信息失败")
	ErrSampleTestCaseFetchFailed = errors.New("获取示例测试用例失败")
)

// 班级相关错误
var (
	ErrPermissionDenied = errors.New("权限不足")
	ErrQRCodeExpired    = errors.New("二维码已过期")
)
