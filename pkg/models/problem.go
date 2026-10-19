/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: problem.go
@Description: 题目数据模型
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProblemDifficulty 题目难度
type ProblemDifficulty string

const (
	DifficultyEasy   ProblemDifficulty = "easy"   // 简单
	DifficultyMedium ProblemDifficulty = "medium" // 中等
	DifficultyHard   ProblemDifficulty = "hard"   // 困难
)

// ProblemStatus 题目状态
type ProblemStatus string

const (
	StatusDraft     ProblemStatus = "draft"     // 草稿
	StatusPublished ProblemStatus = "published" // 已发布
	StatusArchived  ProblemStatus = "archived"  // 已归档
)

// UserProblemStatus 用户题目状态
type UserProblemStatus string

const (
	UserStatusNotAttempted UserProblemStatus = "not_attempted" // 未尝试
	UserStatusAttempted    UserProblemStatus = "attempted"     // 已尝试
	UserStatusAccepted     UserProblemStatus = "accepted"      // 已通过
)

// Problem 题目模型
// 说明：编程题课程挂载关系已经迁移到统一关系表，CourseIDs 字段只用于兼容旧接口返回结构。
type Problem struct {
	ID               primitive.ObjectID   `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	UniqueID         int64                `bson:"unique_id" json:"unique_id" example:"1001"` // 题目唯一编号
	Title            string               `bson:"title" json:"title" binding:"required" example:"两数之和"`
	Description      string               `bson:"description" json:"description" binding:"required" example:"给定一个整数数组 nums 和一个整数目标值 target，请你在该数组中找出和为目标值 target 的那两个整数，并返回它们的数组下标。"`
	Input            string               `bson:"input" json:"input" example:"第一行包含一个整数 n，表示数组长度。第二行包含 n 个整数，表示数组 nums。第三行包含一个整数 target。"`
	Output           string               `bson:"output" json:"output" example:"输出两个整数，表示和为 target 的两个数的下标（从0开始）。"`
	SampleInput      string               `bson:"sample_input" json:"sample_input" example:"4\n2 7 11 15\n9"`
	SampleOutput     string               `bson:"sample_output" json:"sample_output" example:"0 1"`
	Hint             string               `bson:"hint" json:"hint" example:"可以使用哈希表来优化时间复杂度。"`
	Source           string               `bson:"source" json:"source" example:"LeetCode"`
	Author           string               `bson:"author" json:"author" example:"LeetCode"`
	Difficulty       ProblemDifficulty    `bson:"difficulty" json:"difficulty" example:"easy"`
	TimeLimit        int                  `bson:"time_limit" json:"time_limit" example:"1000"`    // 时间限制(ms)
	MemoryLimit      int                  `bson:"memory_limit" json:"memory_limit" example:"256"` // 内存限制(MB)
	Tags             []string             `bson:"tags" json:"tags" example:"数组,哈希表"`
	Status           ProblemStatus        `bson:"status" json:"status" example:"published"`
	IsPublic         bool                 `bson:"is_public" json:"is_public" example:"true"`
	CourseIDs        []primitive.ObjectID `bson:"course_ids,omitempty" json:"course_ids,omitempty" swaggertype:"array,string" example:"507f1f77bcf86cd799439011"` // 课程题库可见范围，空数组表示不挂载到课程
	StandardCode     string               `bson:"standard_code,omitempty" json:"standard_code,omitempty"`                                                         // AI或教师维护的标准参考程序
	StandardLanguage string               `bson:"standard_language,omitempty" json:"standard_language,omitempty"`                                                 // 标准参考程序语言
	ACCount          int                  `bson:"ac_count" json:"ac_count" example:"125"`                                                                         // AC 次数
	SubmitCount      int                  `bson:"submit_count" json:"submit_count" example:"200"`                                                                 // 提交次数
	CreatedBy        primitive.ObjectID   `bson:"created_by" json:"created_by" swaggertype:"string" example:"507f1f77bcf86cd799439012"`
	CreatedAt        time.Time            `bson:"created_at" json:"created_at" example:"2024-10-26T10:00:00Z"`
	UpdatedAt        time.Time            `bson:"updated_at" json:"updated_at" example:"2024-10-26T10:00:00Z"`
	FavoriteCount    int                  `bson:"favorite_count" json:"favorite_count" example:"42"` // 收藏次数
}

// TestCase 测试用例
type TestCase struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	ProblemID primitive.ObjectID `bson:"problem_id" json:"problem_id" swaggertype:"string" example:"507f1f77bcf86cd799439012"`
	Input     string             `bson:"input" json:"input" example:"4\n2 7 11 15\n9"`
	Output    string             `bson:"output" json:"output" example:"0 1"`
	IsSample  bool               `bson:"is_sample" json:"is_sample" example:"true"`

	// 可选的超时配置（优先级高于题目默认配置）
	TimeLimit   *int `bson:"time_limit,omitempty" json:"time_limit,omitempty" example:"1000"`    // 时间限制(ms)，可选
	MemoryLimit *int `bson:"memory_limit,omitempty" json:"memory_limit,omitempty" example:"256"` // 内存限制(MB)，可选

	Score     int       `bson:"score" json:"score" example:"10"` // 用例分数（可选，用于部分分）
	CreatedAt time.Time `bson:"created_at" json:"created_at" example:"2024-10-26T10:00:00Z"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at" example:"2024-10-26T10:00:00Z"`
}

// ProblemList 题目列表项
type ProblemList struct {
	ID            primitive.ObjectID   `json:"id"`
	UniqueID      int64                `json:"unique_id" example:"1001"`
	Title         string               `json:"title"`
	Difficulty    ProblemDifficulty    `json:"difficulty"`
	Tags          []string             `json:"tags"`
	ACCount       int                  `json:"ac_count"`
	SubmitCount   int                  `json:"submit_count"`
	Status        ProblemStatus        `json:"status"`
	IsPublic      bool                 `json:"is_public"`
	CourseIDs     []primitive.ObjectID `json:"course_ids,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	FavoriteCount int                  `json:"favorite_count"` // 新增：题目收藏数
	// 用户状态（仅登录用户返回，未登录时为 nil）
	UserStatus *UserProblemStatus `json:"user_status,omitempty"`
	IsFavorite *bool              `json:"is_favorite,omitempty"` // 用户是否收藏了该题目
}

// UserFavoriteProblem 用户收藏题目关系
type UserFavoriteProblem struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	ProblemID primitive.ObjectID `bson:"problem_id" json:"problem_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
