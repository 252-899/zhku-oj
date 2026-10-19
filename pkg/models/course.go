package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CourseStatus 课程状态
type CourseStatus int8

const (
	CourseStatusDraft  CourseStatus = iota // 草稿
	CourseStatusActive                     // 活跃
	CourseStatusArchived
)

// Course 课程模型
type Course struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Avatar      string               `bson:"avatar,omitempty" json:"avatar,omitempty" example:"https://api.dicebear.com/7.x/shapes/svg?seed=algorithm"`
	Name        string               `bson:"name" json:"name" binding:"required" example:"算法与数据结构"`
	TeacherIds  []primitive.ObjectID `bson:"teacher_ids,omitempty" json:"teacher_ids,omitempty" swaggertype:"array,string" example:"507f1f77bcf86cd799439012,507f1f77bcf86cd799439013"`
	Description string               `bson:"description,omitempty" json:"description,omitempty" example:"本课程主要介绍常用的数据结构和算法，包括线性表、栈、队列、树、图等数据结构"`
	CreatedBy   primitive.ObjectID   `bson:"created_by" json:"created_by" swaggertype:"string" example:"507f1f77bcf86cd799439014"`
	Status      CourseStatus         `bson:"status" json:"status" example:"1"`
	CTime       time.Time            `bson:"ctime" json:"ctime" example:"2024-10-26T10:00:00Z"`
	MTime       time.Time            `bson:"mtime" json:"mtime" example:"2024-10-26T10:0:00Z"`
}

type RelationsUsers struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID     primitive.ObjectID `bson:"task_id" json:"task_id"`
	RelationID primitive.ObjectID `bson:"relation_id,omitempty" json:"relation_id,omitempty"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	CTime      time.Time          `bson:"ctime" json:"ctime"`
}

// 请求和响应结构体

// 注意：所有请求参数定义已迁移到 pkg/models/requests.go
// 这里保留的响应类型将来可能迁移到 pkg/models/responses.go

// CourseResponse 课程响应（包含班级信息）
type CourseResponse struct {
	ID            primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Avatar        string               `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Name          string               `bson:"name" json:"name" binding:"required"`
	TeacherIds    []primitive.ObjectID `bson:"teacher_ids,omitempty" json:"teacher_ids,omitempty"`
	Description   string               `bson:"description,omitempty" json:"description,omitempty"`
	Status        CourseStatus         `bson:"status" json:"status"`
	CTime         time.Time            `bson:"ctime" json:"ctime"`
	CreatedByUser *UserProfile         `json:"created_by_user,omitempty"`
	Teachers      []TeacherWithClasses `json:"teachers,omitempty"` // 教师及其班级信息
}

// TeacherWithClasses 教师及其班级信息
type TeacherWithClasses struct {
	UserProfile *UserProfile `json:"user_profile"`
	Classes     []Clazz      `json:"classes"`
}
