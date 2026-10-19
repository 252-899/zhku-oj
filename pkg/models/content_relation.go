/*
@Author: Codex
@Date: 2026/05/08
@Description: 课程与任务内容关系模型定义
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ContentType 表示统一关系层支持的题目内容类型。
type ContentType string

const (
	ContentTypeProgramming ContentType = "programming" // 现有编程题。
	ContentTypeChoice      ContentType = "choice"      // 新增选择题。
	ContentTypeBlank       ContentType = "blank"       // 新增填空题。
)

// CourseContentRelation 表示课程题库内容挂载关系。
type CourseContentRelation struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourseID    primitive.ObjectID `bson:"course_id" json:"course_id"`
	ContentType ContentType        `bson:"content_type" json:"content_type"`
	ContentID   primitive.ObjectID `bson:"content_id" json:"content_id"`
	CreatedBy   primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// TaskContentRelation 表示任务内容挂载关系。
type TaskContentRelation struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID      primitive.ObjectID `bson:"task_id" json:"task_id"`
	CourseID    primitive.ObjectID `bson:"course_id" json:"course_id"`
	ContentType ContentType        `bson:"content_type" json:"content_type"`
	ContentID   primitive.ObjectID `bson:"content_id" json:"content_id"`
	Sort        int                `bson:"sort" json:"sort"`
	Score       int                `bson:"score" json:"score"`
	Required    bool               `bson:"required" json:"required"`
	CreatedBy   primitive.ObjectID `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}
