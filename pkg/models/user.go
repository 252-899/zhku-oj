/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: user.go
@Description: 用户数据模型
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRole 用户角色
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // 系统管理员
	RoleTeacher  UserRole = "teacher"  // 老师
	RoleStudent  UserRole = "student"  // 学生
	RoleContest  UserRole = "contest"  // 竞赛发起者
	RoleGuest    UserRole = "guest"    // 游客
	RoleExternal UserRole = "external" // 校外人员
	RoleOther    UserRole = "other"    // 外校人员
)

// User 用户模型
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Username     string             `bson:"username" json:"username" binding:"required" example:"student1"`
	Password     string             `bson:"password" json:"password" binding:"required" example:"$2a$10$N.zmdr9k7uOCQb376NoUnuTJ8imdVMaM7ZX/W3xGD.7xUlT8r2.Uy"`
	Email        string             `bson:"email" json:"email" binding:"required,email" example:"student1@zkcodearena.com"`
	RealName     string             `bson:"real_name" json:"real_name" example:"王小明"`
	StudentID    string             `bson:"student_id" json:"student_id" example:"20240001"`
	Role         UserRole           `bson:"role" json:"role" example:"student"`
	Avatar       string             `bson:"avatar" json:"avatar" example:"https://api.dicebear.com/7.x/avataaars/svg?seed=student1"`
	Bio          string             `bson:"bio" json:"bio" example:"我是王小明，热爱编程！"`
	School       string             `bson:"school" json:"school" example:"xx大学"`
	Major        string             `bson:"major" json:"major" example:"计算机科学与技术"`
	Grade        string             `bson:"grade" json:"grade" example:"2024"`
	Class        string             `bson:"class" json:"class" example:"计科1班"`
	Phone        string             `bson:"phone" json:"phone" example:"13800138001"`
	IsActive     bool               `bson:"is_active" json:"is_active" example:"true"`
	LastLoginAt  *time.Time         `bson:"last_login_at" json:"last_login_at" example:"2024-10-26T10:00:00Z"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at" example:"2024-10-26T10:00:00Z"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at" example:"2024-10-26T10:00:00Z"`
	MajorClassID string             `bson:"major_class_id" json:"major_class_id" example:"507f1f77bcf86cd799439012"`
}

// UserProfile 用户资料（不包含敏感信息）
type UserProfile struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id" swaggertype:"string" example:"507f1f77bcf86cd799439011"`
	Username  string             `bson:"username" json:"username" example:"student1"`
	Email     string             `bson:"email" json:"email" example:"student1@example.com"`
	RealName  string             `bson:"real_name" json:"real_name" example:"张同学"`
	StudentID string             `bson:"student_id" json:"student_id" example:"20240001"`
	Role      UserRole           `bson:"role" json:"role" example:"student"`
	Avatar    string             `bson:"avatar" json:"avatar" example:"https://api.dicebear.com/7.x/avataaars/svg?seed=student1"`
	Bio       string             `bson:"bio" json:"bio" example:"热爱编程的计算机专业学生"`
	School    string             `bson:"school" json:"school" example:"xx大学"`
	Major     string             `bson:"major" json:"major" example:"计算机科学与技术"`
	Grade     string             `bson:"grade" json:"grade" example:"2024"`
	Class     string             `bson:"class" json:"class" example:"计科1班"`
	IsActive  bool               `bson:"is_active" json:"is_active" example:"true"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at" example:"2024-10-26T10:00:00Z"`
}

// ToProfile 转换为用户资料
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		RealName:  u.RealName,
		StudentID: u.StudentID,
		Role:      u.Role,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		School:    u.School,
		Major:     u.Major,
		Grade:     u.Grade,
		Class:     u.Class,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt,
	}
}
