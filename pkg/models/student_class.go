/*
@Author:
@Date: 2025/10/25
@Name: student_class.go
@Description: 学生班级关联模型
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type StudentClassStatus string

const (
	StudentClassStatusActive  StudentClassStatus = "active"
	StudentClassStatusDropped StudentClassStatus = "dropped"
)

// StudentClass 学生班级关联模型
type StudentClass struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID primitive.ObjectID `bson:"student_id" json:"student_id" binding:"required"`
	ClassID   primitive.ObjectID `bson:"class_id" json:"class_id" binding:"required"`
	CourseID  primitive.ObjectID `bson:"course_id" json:"course_id" binding:"required"`
	// 加入时间
	JoinTime time.Time `bson:"join_time" json:"join_time"`
	// 状态：active(活跃), dropped(退课)
	Status StudentClassStatus `bson:"status" json:"status"`
	// 创建时间
	CTime time.Time `bson:"ctime" json:"ctime"`
	// 更新时间
	MTime time.Time `bson:"mtime" json:"mtime"`
}

// StudentClassResponse 学生班级响应模型
type StudentClassResponse struct {
	ID        primitive.ObjectID `json:"id"`
	StudentID primitive.ObjectID `json:"student_id"`
	ClassID   primitive.ObjectID `json:"class_id"`
	CourseID  primitive.ObjectID `json:"course_id"`
	// 学生信息
	Student *UserProfile `json:"student,omitempty"`
	// 班级信息
	Class *Clazz `json:"class,omitempty"`
	// 课程信息
	Course   *Course            `json:"course,omitempty"`
	JoinTime time.Time          `json:"join_time"`
	Status   StudentClassStatus `json:"status"`
	CTime    time.Time          `json:"ctime"`
	MTime    time.Time          `json:"mtime"`
}
