package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Tag 题目标签项
type Tag struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`           // 标签名称
	Desc      string             `bson:"desc,omitempty" json:"desc"` // 标签描述
	CreatedAt time.Time          `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at,omitempty" json:"updated_at"`
	CreatedBy primitive.ObjectID `bson:"created_by,omitempty" json:"created_by"`
}

// CreateTagRequest 标签创建请求
type CreateTagRequest struct {
	Name string `json:"name" binding:"required"` // 标签名称
	Desc string `json:"desc,omitempty"`          // 标签描述
}

// UpdateTagRequest 标签更新请求
type UpdateTagRequest struct {
	Name string `json:"name,omitempty"` // 标签名称
	Desc string `json:"desc,omitempty"` // 标签描述
}

// TagQueryRequest 标签查询请求
type TagQueryRequest struct {
	Page     int    `bson:"page,omitempty" json:"page"`
	PageSize int    `bson:"page_size,omitempty" json:"page_size"`
	Keyword  string `bson:"keyword,omitempty" json:"keyword"`
}
