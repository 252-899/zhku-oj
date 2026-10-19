/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: service_user.go
@Description: 用户服务
*/

package service

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务
type UserService struct {
	ClazzService *ClazzService
}

// NewUserService 创建用户服务实例
func NewUserService(clazzService *ClazzService) *UserService {
	return &UserService{ClazzService: clazzService}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(ctx context.Context, user *models.User) error {
	// 检查用户名是否已存在
	existingUser, _ := s.GetUserByUsername(ctx, user.Username)
	if existingUser != nil {
		return errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, _ = s.GetUserByEmail(ctx, user.Email)
	if existingUser != nil {
		return errors.New("邮箱已存在")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	// 设置默认值
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.Role == "" {
		user.Role = models.RoleStudent
	}
	user.IsActive = user.Role != models.RoleTeacher

	// 插入数据库
	collection := utils.GetCollection("users")
	_, err = collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	// 将用户添加到所选的专业班级中
	if user.MajorClassID != "" {
		req := models.JoinClazzRequest{}
		if err := s.ClazzService.JoinClazz(ctx, user.MajorClassID, req, user.ID.Hex()); err != nil {
			// 记录错误，但不影响用户注册
			log.Printf("Failed to add user to class: %v", err)
		} else {
			// 查找与该专业班级关联的所有课程班级
			majorClassObjID, err := primitive.ObjectIDFromHex(user.MajorClassID)
			if err == nil {
				// 查询所有课程班级
				clazzColl := utils.GetCollection("clazzes")
				var courseClasses []models.Clazz
				cursor, err := clazzColl.Find(ctx, bson.M{
					"class_type":      models.ClassTypeCourse,
					"major_class_ids": bson.M{"$in": []primitive.ObjectID{majorClassObjID}},
				})
				if err == nil {
					defer cursor.Close(ctx)
					if err = cursor.All(ctx, &courseClasses); err == nil {
						// 将用户添加到每个关联的课程班级中
						for _, courseClass := range courseClasses {
							// 检查用户是否已经是课程班级成员
							isMember, err := s.ClazzService.Repo.IsClazzMember(ctx, courseClass.ID, user.ID)
							if err == nil && !isMember {
								// 检查班级是否已满
								if courseClass.StudentMemberCount() < courseClass.MaxMembers {
									//添加用户到课程班级
									// 1. 添加用户到课程班级 (更新 clazzes 表)
									_, err := s.ClazzService.Repo.AddClazzMember(ctx, courseClass.ID, user.ID)
									if err != nil {
										// 记录错误，但不影响用户注册
										log.Printf("Failed to add user to course class %s: %v", courseClass.ID.Hex(), err)
									} else {
										// 2. 在 student_classes 表中创建记录
										now := time.Now()
										studentClass := &models.StudentClass{
											ID:        primitive.NewObjectID(),
											StudentID: user.ID,
											ClassID:   courseClass.ID,
											CourseID:  courseClass.CourseId,
											JoinTime:  now,
											Status:    models.StudentClassStatusActive,
											CTime:     now,
											MTime:     now,
										}
										studentClassColl := utils.GetCollection("student_classes")
										_, err := studentClassColl.InsertOne(ctx, studentClass)
										if err != nil {
											// 记录错误，但不影响用户注册
											log.Printf("Failed to create student_class record for course class %s: %v", courseClass.ID.Hex(), err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return nil

}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	collection := utils.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	collection := utils.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) GetUserByStudentID(ctx context.Context, stdId string) (*models.User, error) {
	collection := utils.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"student_id": stdId}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	collection := utils.GetCollection("users")
	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ValidateUser 验证用户登录
func (s *UserService) ValidateUser(ctx context.Context, stdId, password string) (*models.User, error) {
	user, err := s.GetUserByStudentID(ctx, stdId)
	if err != nil {
		return nil, errors.New("账号或密码错误")
	}

	if !user.IsActive {
		if user.Role == models.RoleTeacher {
			return nil, errors.New("老师账号待管理员审核")
		}
		return nil, errors.New("账户已被禁用")
	}

	// 验证密码（使用 bcrypt）
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("账号或密码错误")
	}

	// 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	if err = s.UpdateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	collection := utils.GetCollection("users")
	_, err := collection.ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	return err
}

// GetUsers 获取用户列表
func (s *UserService) GetUsers(ctx context.Context, page, pageSize int, role models.UserRole, isActive *bool) ([]*models.User, int64, error) {
	collection := utils.GetCollection("users")

	// 构建查询条件
	filter := bson.M{}
	if role != "" {
		filter["role"] = role
	}
	if isActive != nil {
		filter["is_active"] = *isActive
	}

	// 获取总数
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询
	opts := options.Find().
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize)).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, 0, err
		}
		users = append(users, &user)
	}

	return users, total, nil
}

// PageQueryAllTeachers 分页查询所有教师信息，支持按 username 模糊查询
func (s *UserService) PageQueryAllTeachers(ctx context.Context, req *models.PageQueryAllTeachersRequest, isAdmin bool) (*models.PageQueryCourseTeachersResponse, error) {
	// 设置默认分页参数
	pageNum := int64(1)
	pageSize := int64(10)
	if req.PageNum != nil {
		pageNum = *req.PageNum
	}
	if req.PageSize != nil {
		pageSize = *req.PageSize
	}
	// 设置合理的限制
	if pageSize > 100 {
		pageSize = 50
	}

	// 构建查询条件：角色为教师
	userFilter := bson.M{"role": models.RoleTeacher}
	if isAdmin {
		if req.IsActive != nil {
			userFilter["is_active"] = *req.IsActive
		}
	} else {
		userFilter["is_active"] = true
	}

	// 如果指定了 username 进行模糊查询（字面量子串，避免正则注入）
	if req.Username != nil {
		if kw := strings.TrimSpace(*req.Username); kw != "" {
			userFilter["username"] = bson.M{"$regex": regexp.QuoteMeta(kw), "$options": "i"}
		}
	}

	// 查询教师总数
	total, err := utils.GetCollection("users").CountDocuments(ctx, userFilter)
	if err != nil {
		return nil, errors.New("查询教师总数失败: " + err.Error())
	}

	// 分页查询教师信息
	findOptions := options.Find().
		SetSkip((pageNum - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err := utils.GetCollection("users").Find(ctx, userFilter, findOptions)
	if err != nil {
		return nil, errors.New("查询教师信息失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var users []*models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, errors.New("解析教师信息失败: " + err.Error())
	}

	// 构造分页响应
	res := &models.PageQueryCourseTeachersResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Teachers: make([]models.UserProfile, len(users)),
	}

	// 转换教师信息
	for i, user := range users {
		res.Teachers[i] = *user.ToProfile()
	}

	return res, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	collection := utils.GetCollection("users")
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetStudentClasses 获取学生加入的所有班级
func (s *UserService) GetStudentClasses(ctx context.Context, studentID string) ([]*models.StudentClassResponse, error) {
	// 这里应该调用课程服务来获取学生班级信息
	// 由于服务之间不应该直接依赖，我们可以通过数据库直接查询
	collection := utils.GetCollection("student_classes")
	cursor, err := collection.Find(ctx, bson.M{"student_id": studentID, "status": "active"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var studentClasses []models.StudentClass
	if err = cursor.All(ctx, &studentClasses); err != nil {
		return nil, err
	}

	// 构建响应数据
	var responses []*models.StudentClassResponse
	for _, sc := range studentClasses {
		response := &models.StudentClassResponse{
			ID:        sc.ID,
			StudentID: sc.StudentID,
			ClassID:   sc.ClassID,
			CourseID:  sc.CourseID,
			JoinTime:  sc.JoinTime,
			Status:    sc.Status,
			CTime:     sc.CTime,
			MTime:     sc.MTime,
		}
		responses = append(responses, response)
	}

	return responses, nil
}
