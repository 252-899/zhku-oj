/*
@Author: omenkk7
@Date: 2026/03/17
@Description: 课程文件（分片上传/断点续传）相关模型
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CourseFile 课程文件元数据（Mongo: course_files）
type CourseFile struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourseID    primitive.ObjectID `bson:"course_id" json:"course_id"`
	UploaderID  primitive.ObjectID `bson:"uploader_id" json:"uploader_id"`
	FileName    string             `bson:"file_name" json:"file_name"`
	ContentType string             `bson:"content_type" json:"content_type"`
	Size        int64              `bson:"size" json:"size"`
	SHA256      string             `bson:"sha256" json:"sha256"`
	StoragePath string             `bson:"storage_path" json:"storage_path"`
	CTime       time.Time          `bson:"ctime" json:"ctime"`
	MTime       time.Time          `bson:"mtime" json:"mtime"`
}

// CourseFileUploadCheckRequest 上传前检查（秒传/断点续传）
type CourseFileUploadCheckRequest struct {
	FileSHA256  string `json:"file_sha256" binding:"required"`
	FileName    string `json:"file_name" binding:"required"`
	TotalChunks int    `json:"total_chunks" binding:"required"`
}

type CourseFileUploadCheckResponse struct {
	Instant  bool  `json:"instant"`  // true 表示秒传（文件已存在）
	Uploaded []int `json:"uploaded"` // 已上传 chunkIndex 列表
}

// CourseFileUploadMergeRequest 合并请求
type CourseFileUploadMergeRequest struct {
	FileSHA256  string `json:"file_sha256" binding:"required"`
	FileName    string `json:"file_name" binding:"required"`
	TotalChunks int    `json:"total_chunks" binding:"required"`
	FileSize    int64  `json:"file_size" binding:"required"`
	ContentType string `json:"content_type"`
}

type CourseFileUploadMergeResponse struct {
	FileID string `json:"file_id"`
}
