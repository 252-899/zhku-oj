/*
@Author:sir
@Date: 2025/10/25
@Name: service_clazz.go
@Description: 班级服务层实现
*/

package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"
	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ClazzService 班级服务
type ClazzService struct {
	Repo *repository.ClazzRepositoryImpl
}

// NewClazzService 创建班级服务实例
func NewClazzService(repo *repository.ClazzRepositoryImpl) *ClazzService {
	return &ClazzService{Repo: repo}
}

// 刷新二维码
func (s *ClazzService) RefreshQrcode(userId string, courseId string, clazzId string, ctx context.Context) (interface{}, error) {
	// 使用Repository层的ConvertToObjectID方法转换ID
	userObjId, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return nil, errors.New("用户错误")
	}
	courseObjId, err := s.Repo.ConvertToObjectID(courseId)
	if err != nil {
		return nil, errors.New("课程错误")
	}

	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, courseObjId, userObjId)
	if !isAuthorized || err != nil {
		if !isAuthorized {
			return nil, errors.New("权限不足")
		}
		return nil, err
	}

	// 使用Repository层的GenerateAndSaveQRCode方法生成并保存二维码
	return s.Repo.GenerateAndSaveQRCode(ctx, clazzId)
}

// RefreshQrcodeByClazzId 通过班级ID刷新二维码
func (s *ClazzService) RefreshQrcodeByClazzId(userId string, clazzId string, ctx context.Context) (interface{}, error) {
	// 使用Repository层的ConvertToObjectID方法转换ID
	userObjId, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return nil, errors.New("用户错误")
	}
	clazzObjId, err := s.Repo.ConvertToObjectID(clazzId)
	if err != nil {
		return nil, errors.New("班级ID错误")
	}

	// 使用Repository层的GetClazzByID方法获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjId)
	if err != nil {
		return nil, err
	}

	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjId)
	if !isAuthorized || err != nil {
		if !isAuthorized {
			return nil, errors.New("权限不足")
		}
		return nil, err
	}

	// 使用Repository层的GenerateAndSaveQRCode方法生成并保存二维码
	return s.Repo.GenerateAndSaveQRCode(ctx, clazzId)
}

// CreateMajorClass 创建专业班级
func (s *ClazzService) CreateMajorClass(ctx context.Context, req *models.CreateMajorClassRequest, userId string) (*models.GetClazzResponse, error) {
	// 将用户ID转换为ObjectID
	userObjId, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 检查用户是否为管理员
	userColl := utils.GetCollection("users")
	var user models.User
	err = userColl.FindOne(ctx, bson.M{"_id": userObjId}).Decode(&user)
	if err != nil {
		return nil, errors.New("查询用户信息失败: " + err.Error())
	}

	if user.Role != models.RoleAdmin {
		return nil, errors.New("权限不足，只有管理员可以创建专业班级")
	}

	// 根据学号/工号查找用户（支持学生和老师角色）
	userColl = utils.GetCollection("users")
	var users []models.User
	// 使用StudentID字段（学号/工号）匹配，支持学生和老师角色
	filter := bson.M{
		"student_id": bson.M{"$in": req.StudentIds},
		"role":       bson.M{"$in": []models.UserRole{models.RoleStudent, models.RoleTeacher}},
	}
	cursor, err := userColl.Find(ctx, filter)

	if err != nil {
		return nil, errors.New("查询用户信息失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &users); err != nil {
		return nil, errors.New("解析用户信息失败: " + err.Error())
	}

	// 检查是否所有提供的学号/工号都找到了对应的用户
	if len(users) != len(req.StudentIds) {
		return nil, errors.New("部分学号/工号未找到对应的用户账号")
	}

	// 提取用户的ObjectID
	var memberIDs []primitive.ObjectID
	for _, user := range users {
		memberIDs = append(memberIDs, user.ID)
	}

	// 处理班级最大成员数
	var maxMembers int
	if req.MaxMembers == nil {
		maxMembers = 60
	} else if *req.MaxMembers > 100 {
		maxMembers = 100
	} else {
		maxMembers = *req.MaxMembers
	}

	// 创建专业班级对象
	now := time.Now()
	clazz := &models.Clazz{
		ID:            primitive.NewObjectID(),
		Name:          req.Name,
		ClassType:     models.ClassTypeMajor,
		Description:   req.Description,
		MemberIDs:     memberIDs, // 将用户ID添加到班级成员中
		RequireInvite: false,
		MaxMembers:    maxMembers,
		AddNums:       len(memberIDs), // 设置当前成员数量
		Status:        models.ClassStatusActive,
		CTime:         now,
		MTime:         now,
	}

	// 使用Repository层创建班级
	createdClazzId, err := s.Repo.CreateClazz(ctx, clazz)
	if err != nil {
		return nil, err
	}

	// 更新用户的MajorClassID字段
	for _, user := range users {
		update := bson.M{"$set": bson.M{"major_class_id": clazz.ID.Hex()}}
		_, err := userColl.UpdateOne(ctx, bson.M{"_id": user.ID}, update)
		if err != nil {
			// 记录错误但不中断创建流程
			log.Printf("更新用户 %s 的专业班级ID失败: %v", user.StudentID, err)
		}
	}

	// 返回响应
	return &models.GetClazzResponse{
		ID:            createdClazzId,
		Name:          req.Name,
		Description:   req.Description,
		RequireInvite: false,
		MaxMembers:    maxMembers,
		AddNums:       len(memberIDs),
		Status:        models.ClassStatusActive,
		CTime:         now,
	}, nil
}

// CreateCourseClass 创建课程班级
func (s *ClazzService) CreateCourseClass(ctx context.Context, req *models.CreateCourseClassRequest, userId string) (*models.GetClazzResponse, error) {
	// 将用户ID转换为ObjectID
	userObjId, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 检查用户是否有课程权限（创建者或教师）
	courseObjId, err := s.Repo.ConvertToObjectID(req.CourseId)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, courseObjId, userObjId)
	if err != nil {
		return nil, err
	}

	if !isAuthorized {
		return nil, errors.New("权限不足，只有课程创建者或教师可以创建班级")
	}

	// 将请求中的教师ID列表转换为ObjectID
	teacherIds, err := s.Repo.ConvertToObjectIDs(req.TeacherIds)
	if err != nil {
		return nil, errors.New("无效的教师ID")
	}

	// 处理班级最大成员数
	var maxMembers int
	if req.MaxMembers == nil {
		maxMembers = 60
	} else if *req.MaxMembers > 100 {
		maxMembers = 100
	} else {
		maxMembers = *req.MaxMembers
	}

	// 创建课程班级对象
	now := time.Now()
	clazz := &models.Clazz{
		ID:            primitive.NewObjectID(),
		Name:          req.Name,
		ClassType:     models.ClassTypeCourse,
		Description:   req.Description,
		CourseId:      courseObjId,
		Schedule:      req.Schedule,
		TeacherIds:    teacherIds,
		RequireInvite: req.RequireInvite,
		MaxMembers:    maxMembers,
		AddNums:       0,
		Status:        models.ClassStatusActive,
		CTime:         now,
		MTime:         now,
	}

	// 使用Repository层创建班级
	createdClazzId, err := s.Repo.CreateClazz(ctx, clazz)
	if err != nil {
		return nil, err
	}

	// 如果需要邀请码，则生成并保存二维码到Redis
	// 注意：二维码生成失败不应阻止班级创建成功，只记录日志
	if req.RequireInvite {
		qrcode, err := s.Repo.GenerateAndSaveQRCode(ctx, createdClazzId.Hex())
		if err != nil {
			// 记录错误但不中断响应，班级已成功创建
			log.Printf("生成班级二维码失败 (clazz_id=%s): %v", createdClazzId.Hex(), err)
		} else {
			log.Printf("班级二维码生成成功 (clazz_id=%s, qrcode长度=%d)", createdClazzId.Hex(), len(qrcode))
		}
	}

	// 返回响应
	return &models.GetClazzResponse{
		ID:            createdClazzId,
		Name:          req.Name,
		Description:   req.Description,
		CourseId:      courseObjId,
		Schedule:      req.Schedule,
		RequireInvite: req.RequireInvite,
		MaxMembers:    maxMembers,
		AddNums:       0,
		Status:        models.ClassStatusActive,
		CTime:         now,
	}, nil
}

// 私有方法
// batchAddMajorClassStudents 将专业班级的学生批量添加到课程班级
func (s *ClazzService) batchAddMajorClassStudents(ctx context.Context, majorClassID, courseClassID, courseID primitive.ObjectID) error {
	// 获取专业班级信息
	majorClass, err := s.Repo.GetClazzByID(ctx, majorClassID)
	if err != nil {
		return err
	}

	// 验证是否为专业班级
	if majorClass.ClassType != models.ClassTypeMajor {
		return errors.New("不是专业班级")
	}

	// 获取课程班级信息，检查成员数量限制
	courseClass, err := s.Repo.GetClazzByID(ctx, courseClassID)
	if err != nil {
		return err
	}

	// 批量添加学生
	if len(majorClass.MemberIDs) > 0 {
		// 这里可以使用批量添加的方法，避免重复添加
		for _, studentID := range majorClass.MemberIDs {
			// 检查学生是否已经在班级的 member_ids 中
			isMemberInClazz := false
			for _, mid := range courseClass.MemberIDs {
				if mid == studentID {
					isMemberInClazz = true
					break
				}
			}

			// 如果不是成员，则添加
			if !isMemberInClazz {
				// 检查班级是否已满（仅学生占名额）
				if courseClass.StudentMemberCount() >= courseClass.MaxMembers {
					return errors.New("班级已满，无法添加更多成员")
				}

				// 添加学生到班级
				// 1. 更新 clazzes 表
				clazzColl := utils.GetCollection("clazzes")
				seat := repository.ClazzStudentSeatAvailableExpr()
				filter := bson.M{
					"_id":   courseClassID,
					"$expr": seat["$expr"],
				}
				update := bson.M{
					"$inc":      bson.M{"add_nums": 1},
					"$addToSet": bson.M{"member_ids": studentID},
					"$set":      bson.M{"mtime": time.Now()},
				}
				result, err := clazzColl.UpdateOne(ctx, filter, update)
				if err != nil {
					return err
				}
				if result.MatchedCount == 0 {
					return errors.New("班级已满")
				}

				// 2. 在 student_classes 表中创建记录
				now := time.Now()
				studentClass := &models.StudentClass{
					ID:        primitive.NewObjectID(),
					StudentID: studentID,
					ClassID:   courseClassID,
					CourseID:  courseID,
					JoinTime:  now,
					Status:    models.StudentClassStatusActive,
					CTime:     now,
					MTime:     now,
				}
				studentClassColl := utils.GetCollection("student_classes")
				_, err = studentClassColl.InsertOne(ctx, studentClass)
				if err != nil {
					// 回滚 clazzes 表的更新
					clazzColl.UpdateOne(ctx, bson.M{"_id": courseClassID}, bson.M{
						"$inc":  bson.M{"add_nums": -1},
						"$pull": bson.M{"member_ids": studentID},
					})
					return err
				}

				// 更新内存中的课程班级信息
				courseClass.AddNums++
				courseClass.MemberIDs = append(courseClass.MemberIDs, studentID)
			}
		}
	}

	return nil
}

// GetMajorClazzes 获取所有专业班级
func (s *ClazzService) GetMajorClazzes(ctx context.Context, userID string) ([]models.Clazz, error) {

	// 查询所有专业班级
	clazzes, err := s.Repo.GetMajorClazzes(ctx)
	if err != nil {
		return nil, err
	}

	for i := range clazzes {
		clazzes[i].SyncAddNumsFromMemberIDs()
	}

	return clazzes, nil
}

// ListTeacherClasses 查询当前教师可见的课程班级。
// 可见范围包括该教师任课或创建的课程下班级，以及班级 teacher_ids 直接包含该教师的班级。
func (s *ClazzService) ListTeacherClasses(ctx context.Context, userID string) ([]models.Clazz, error) {
	userObjID, err := primitive.ObjectIDFromHex(strings.TrimSpace(userID))
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	courseCursor, err := utils.GetCollection("courses").Find(ctx, bson.M{
		"$or": []bson.M{
			{"created_by": userObjID},
			{"teacher_ids": userObjID},
		},
	})
	if err != nil {
		return nil, err
	}
	defer courseCursor.Close(ctx)

	courseIDs := make([]primitive.ObjectID, 0)
	for courseCursor.Next(ctx) {
		var course struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := courseCursor.Decode(&course); err != nil {
			return nil, err
		}
		if !course.ID.IsZero() {
			courseIDs = append(courseIDs, course.ID)
		}
	}
	if err := courseCursor.Err(); err != nil {
		return nil, err
	}

	filter := bson.M{"teacher_ids": userObjID}
	if len(courseIDs) > 0 {
		filter = bson.M{
			"$or": []bson.M{
				{"teacher_ids": userObjID},
				{"course_id": bson.M{"$in": courseIDs}},
			},
		}
	}

	classCursor, err := utils.GetCollection("clazzes").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer classCursor.Close(ctx)

	var clazzes []models.Clazz
	if err := classCursor.All(ctx, &clazzes); err != nil {
		return nil, err
	}
	for i := range clazzes {
		clazzes[i].SyncAddNumsFromMemberIDs()
	}
	return clazzes, nil
}

// GetClazzByID 获取课程班级详情
func (s *ClazzService) GetClazzByID(ctx context.Context, clazzID string, userID string) (*models.GetClazzResponse, error) {
	// 使用Repository层的ConvertToObjectID方法转换ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return nil, errors.New("无效的班级ID")
	}

	userObjID, err := s.Repo.ConvertToObjectID(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 尝试从Redis缓存中获取课程班级详情
	cacheKey := "clazz_detail:" + clazzID + ":" + userID
	cachedData, err := utils.RedisClient.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		// 缓存命中，直接返回缓存数据
		var response models.GetClazzResponse
		if err := json.Unmarshal([]byte(cachedData), &response); err == nil {
			return &response, nil
		}
		// 如果反序列化失败，继续执行下面的逻辑
	}

	// 使用Repository层的GetClazzByID方法获取班级基本信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return nil, err
	}

	// 构造返回的响应对象
	response := &models.GetClazzResponse{
		ID:            clazz.ID,
		Name:          clazz.Name,
		Description:   clazz.Description,
		CourseId:      clazz.CourseId,
		Schedule:      clazz.Schedule,
		RequireInvite: clazz.RequireInvite,
		MaxMembers:    clazz.MaxMembers,
		AddNums:       clazz.StudentMemberCount(),
		Status:        clazz.Status,
		CTime:         clazz.CTime,
		MajorClassIDs: clazz.MajorClassIDs, // 添加关联的专业班级ID数组
	}

	courseObjID := clazz.CourseId

	// 使用Repository层的IsClazzMember方法检查用户是否是班级成员
	isMember, err := s.Repo.IsClazzMember(ctx, clazzObjID, userObjID)
	if err != nil {
		return nil, errors.New("检查班级成员失败: " + err.Error())
	}

	// 使用Repository层的CheckCourseAuthorization方法检查用户是否是课程创建者或教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, courseObjID, userObjID)
	if err != nil {
		return nil, err
	}

	if !isAuthorized && !isMember {
		return nil, errors.New("权限不足")
	}

	// 使用Repository层的GetTeachersByIDs方法获取班级教师信息
	teachers, err := s.Repo.GetTeachersByIDs(ctx, clazz.TeacherIds)
	if err != nil {
		return nil, err
	}

	// 转换为用户资料数组
	response.Teachers = make([]models.UserProfile, 0, len(teachers))
	for _, teacher := range teachers {
		response.Teachers = append(response.Teachers, *teacher.ToProfile())
	}

	// 将结果缓存到Redis中，缓存10分钟
	cacheData, err := json.Marshal(response)
	if err == nil {
		utils.RedisClient.Set(ctx, cacheKey, cacheData, 10*time.Minute)
	}

	return response, nil
}

// UpdateClazzInfo 更新班级信息（不包括教师和成员）
func (s *ClazzService) UpdateClazzInfo(ctx context.Context, clazzID string, userId string, req *models.UpdateClazzRequest) error {
	// 使用Repository层的ConvertToObjectID方法转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 使用Repository层的ConvertToObjectID方法转换用户ID
	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 使用Repository层的GetClazzByID方法获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证操作者是否有权限修改班级
	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以修改班级")
	}

	// 构建更新字段
	updateFields := bson.M{}
	// 检查每个字段，如果字段非空，则加入更新内容
	if req.Name != "" {
		updateFields["name"] = req.Name
	}
	if req.Description != "" {
		updateFields["description"] = req.Description
	}
	if req.Schedule != "" {
		updateFields["schedule"] = req.Schedule
	}
	if req.RequireInvite != nil {
		updateFields["require_invite"] = *req.RequireInvite
	}
	if req.MaxMembers != nil {
		// 确保最大成员数不超过100
		maxMembers := *req.MaxMembers
		if maxMembers > 100 {
			maxMembers = 100
		}
		updateFields["max_members"] = maxMembers
	}

	// 如果没有任何字段需要更新，返回早期退出
	if len(updateFields) == 0 {
		return errors.New("无更改内容")
	}

	updateFields["mtime"] = time.Now()

	// 使用Repository层的UpdateClazz方法更新班级信息
	updated, err := s.Repo.UpdateClazz(ctx, clazzObjID, updateFields)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("找不到该班级")
	}

	return nil
}

// DeleteClazz 删除课程班级
func (s *ClazzService) DeleteClazz(ctx context.Context, clazzID string, userID string) error {
	// 使用Repository层的ConvertToObjectID方法转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 使用Repository层的ConvertToObjectID方法转换用户ID
	userObjID, err := s.Repo.ConvertToObjectID(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 使用Repository层的GetClazzByID方法获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证操作者是否有权限删除课程班级
	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以删除课程班级")
	}

	// 检查班级是否还有学生成员
	if clazz.StudentMemberCount() > 0 {
		return errors.New("班级还有成员，无法删除")
	}

	// 使用Repository层的DeleteClazz方法删除课程班级
	deleted, err := s.Repo.DeleteClazz(ctx, clazzObjID)
	if err != nil {
		return errors.New("删除课程班级失败: " + err.Error())
	}
	if !deleted {
		return errors.New("班级不存在")
	}

	return nil
}

// AddClazzMember 添加课程班级成员
func (s *ClazzService) AddClazzMember(ctx context.Context, clazzID string, memberID string, operatorID string) error {
	// 使用Repository层的ConvertToObjectID方法转换ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	memberObjID, err := s.Repo.ConvertToObjectID(memberID)
	if err != nil {
		return errors.New("无效的成员ID")
	}

	operatorObjID, err := s.Repo.ConvertToObjectID(operatorID)
	if err != nil {
		return errors.New("无效的操作者ID")
	}

	// 使用Repository层的GetClazzByID方法获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证操作者是否有权限添加成员
	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, operatorObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以添加成员")
	}

	// 检查班级是否已满（仅统计学生）
	if clazz.StudentMemberCount() >= clazz.MaxMembers {
		return errors.New("班级已满，无法添加更多成员")
	}

	// 使用Repository层的AddClazzMember方法添加成员（包含乐观锁和成员检查）
	added, err := s.Repo.AddClazzMember(ctx, clazzObjID, memberObjID)
	if err != nil {
		return err
	}
	if !added {
		return errors.New("添加成员失败，可能是班级已满或用户已是成员")
	}

	// 同时创建学生班级关联记录
	studentClass := &models.StudentClass{
		ID:        primitive.NewObjectID(),
		StudentID: memberObjID,
		ClassID:   clazzObjID,
		CourseID:  clazz.CourseId,
		JoinTime:  time.Now(),
		Status:    models.StudentClassStatusActive,
		CTime:     time.Now(),
		MTime:     time.Now(),
	}

	// 使用Repository层的CreateStudentClassRelation方法创建关联
	err = s.Repo.CreateStudentClassRelation(ctx, studentClass)
	if err != nil {
		// 如果创建关联记录失败，应该回滚之前的班级成员添加操作
		// 这里简化处理，实际应该使用事务
		_, _ = utils.GetCollection("clazzes").UpdateOne(ctx,
			bson.M{"_id": clazzObjID},
			bson.M{"$inc": bson.M{"add_nums": -1}})
		return err
	}

	return nil
}

// RemoveClazzMember 移除单个班级成员
func (s *ClazzService) RemoveClazzMember(ctx context.Context, clazzID, memberID, operatorID string) error {
	// 构造批量删除请求
	req := models.RemoveClazzMembersRequest{
		ClazzID:   clazzID,
		MemberIDs: []string{memberID},
	}
	return s.RemoveClazzMembers(ctx, req, operatorID)
}

// RemoveClazzMembers 批量移除课程班级成员
func (s *ClazzService) RemoveClazzMembers(ctx context.Context, req models.RemoveClazzMembersRequest, operatorID string) error {
	// 使用Repository层的ConvertToObjectID方法转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(req.ClazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 使用Repository层的ConvertToObjectID方法转换操作者ID
	operatorObjID, err := s.Repo.ConvertToObjectID(operatorID)
	if err != nil {
		return errors.New("无效的操作者ID")
	}

	// 使用Repository层的ConvertToObjectIDs方法批量转换成员ID（已包含去重）
	memberObjIDs, err := s.Repo.ConvertToObjectIDs(req.MemberIDs)
	if err != nil {
		return err
	}

	// 使用Repository层的GetClazzByID方法获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证操作者是否有权限移除成员
	// 使用Repository层的CheckCourseAuthorization方法验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, operatorObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以移除成员")
	}

	// 使用Repository层的BatchRemoveClazzMembers方法批量移除成员
	removedCount, err := s.Repo.BatchRemoveClazzMembers(ctx, clazzObjID, memberObjIDs)
	if err != nil {
		return err
	}

	if removedCount == 0 {
		return errors.New("没有有效的班级成员需要移除")
	}

	return nil
}

// AddClazzTeacher 为班级添加教师
func (s *ClazzService) AddClazzTeacher(ctx context.Context, clazzId string, teacherId string, userId string) error {
	// 验证用户权限（只有课程创建者或课程教师才能添加班级教师）
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	teacherObjID, err := s.Repo.ConvertToObjectID(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 检查班级是否存在
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 验证操作者是否有权限添加教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或课程教师可以添加班级教师")
	}

	// 添加教师到班级
	success, err := s.Repo.AddClazzTeacher(ctx, clazzObjID, teacherObjID)
	if err != nil {
		return errors.New("添加班级教师失败: " + err.Error())
	}

	if !success {
		return errors.New("该教师已经是班级的教师")
	}

	// 清除与该班级相关的所有缓存
	cachePattern := "clazz_detail:" + clazzId + ":*"
	keys, err := utils.RedisClient.Keys(ctx, cachePattern).Result()
	if err != nil {
		log.Printf("获取缓存键失败: %v", err)
	} else if len(keys) > 0 {
		if err := s.Repo.DeleteCache(ctx, keys...); err != nil {
			log.Printf("清除缓存失败: %v", err)
		}
	}

	return nil
}

// AddClazzTeachers 为班级批量添加教师
func (s *ClazzService) AddClazzTeachers(ctx context.Context, clazzId string, teacherIds []string, userId string) error {
	// 转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 转换用户ID
	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 检查班级是否存在
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 验证操作者是否有权限添加教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或课程教师可以添加班级教师")
	}

	// 转换教师ID数组
	teacherObjIDs, err := s.Repo.ConvertToObjectIDs(teacherIds)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 批量添加教师到班级
	_, err = s.Repo.BatchAddClazzTeachers(ctx, clazzObjID, teacherObjIDs)
	if err != nil {
		return err
	}

	// 清除与该班级相关的所有缓存
	cachePattern := "clazz_detail:" + clazzId + ":*"
	keys, err := utils.RedisClient.Keys(ctx, cachePattern).Result()
	if err != nil {
		log.Printf("获取缓存键失败: %v", err)
	} else if len(keys) > 0 {
		if err := s.Repo.DeleteCache(ctx, keys...); err != nil {
			log.Printf("清除缓存失败: %v", err)
		}
	}

	return nil
}

// RemoveClazzTeacher 为班级移除教师
func (s *ClazzService) RemoveClazzTeacher(ctx context.Context, clazzId string, teacherId string, userId string) error {
	// 验证用户权限（只有课程创建者或课程教师才能移除班级教师）
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	teacherObjID, err := s.Repo.ConvertToObjectID(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 检查班级是否存在
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证操作者是否有权限移除教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以移除班级教师")
	}

	// 移除教师
	success, err := s.Repo.RemoveClazzTeacher(ctx, clazzObjID, teacherObjID)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("该教师不是班级的教师")
	}

	// 清除与该班级相关的所有缓存
	cachePattern := "clazz_detail:" + clazzId + ":*"
	keys, err := utils.RedisClient.Keys(ctx, cachePattern).Result()
	if err != nil {
		log.Printf("获取缓存键失败: %v", err)
	} else if len(keys) > 0 {
		if err := s.Repo.DeleteCache(ctx, keys...); err != nil {
			log.Printf("清除缓存失败: %v", err)
		}
	}

	return nil
}

// GetClazzesByCourseId 通过课程ID查询所有班级
func (s *ClazzService) GetClazzesByCourseId(ctx context.Context, courseId string, userId string) ([]models.Clazz, error) {
	courseObjId, err := s.Repo.ConvertToObjectID(courseId)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}

	userObjId, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 验证用户是否有权限查看该课程的班级
	// 用户必须是课程创建者或课程教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, courseObjId, userObjId)
	if err != nil {
		return nil, err
	}

	if !isAuthorized {
		return nil, errors.New("权限不足")
	}

	// 查询该课程的所有班级
	results, err := s.Repo.GetClazzesByCourseID(ctx, courseObjId)
	if err != nil {
		return nil, err
	}

	for i := range results {
		results[i].SyncAddNumsFromMemberIDs()
	}

	return results, nil
}

// GetStudentClasses 获取学生的所有班级
func (s *ClazzService) GetStudentClasses(ctx context.Context, studentID string) ([]*models.StudentClassResponse, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return nil, errors.New("无效的学生ID")
	}

	// 1) student_classes 中 status=active 的关联（历史接口行为）
	studentClasses, err := s.Repo.GetStudentClasses(ctx, studentObjID)
	if err != nil {
		return nil, errors.New("查询学生班级失败: " + err.Error())
	}

	seenClass := make(map[primitive.ObjectID]struct{}, len(studentClasses))
	for _, sc := range studentClasses {
		seenClass[sc.ClassID] = struct{}{}
	}

	// 2) 与 GET /clazzes/{id}/students 一致：以 clazzes.member_ids 为在班依据。只要仍在 member_ids 中，就应出现在本接口（含：仅有 member_ids、或 student_classes 为 dropped 但未从 member_ids 移除等历史脏数据）
	clazzesByMember, err := s.Repo.FindClazzesByMemberID(ctx, studentObjID)
	if err != nil {
		return nil, errors.New("查询学生所在班级失败: " + err.Error())
	}

	for _, c := range clazzesByMember {
		if _, ok := seenClass[c.ID]; ok {
			continue
		}
		studentClasses = append(studentClasses, models.StudentClass{
			ID:        primitive.NilObjectID,
			StudentID: studentObjID,
			ClassID:   c.ID,
			CourseID:  c.CourseId,
			JoinTime:  c.CTime,
			Status:    models.StudentClassStatusActive,
			CTime:     c.CTime,
			MTime:     c.MTime,
		})
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

		// 获取学生信息
		user, err := s.getUserByID(ctx, sc.StudentID)
		if err == nil {
			response.Student = user.ToProfile()
		}

		// 获取班级信息
		clazz, err := s.Repo.GetClazzByID(ctx, sc.ClassID)
		if err == nil {
			response.Class = clazz
		}

		// 获取课程信息
		course, err := s.getCourseByID(ctx, sc.CourseID)
		if err == nil {
			response.Course = course
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetClassStudents 获取班级的所有学生
func (s *ClazzService) GetClassStudents(ctx context.Context, classID string) ([]*models.UserProfile, error) {
	classObjID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return nil, errors.New("无效的班级ID")
	}

	// 获取班级信息，包括 member_ids 字段
	clazz, err := s.Repo.GetClazzByID(ctx, classObjID)
	if err != nil {
		return nil, errors.New("查询班级信息失败: " + err.Error())
	}

	// 从 clazzes 表的 member_ids 字段中获取学生ID列表
	studentIDs := clazz.MemberIDs
	if len(studentIDs) == 0 {
		return []*models.UserProfile{}, nil
	}

	// 调用repository层根据ID列表获取学生信息
	students, err := s.Repo.GetUsersByIDs(ctx, studentIDs, "")
	if err != nil {
		return nil, errors.New("获取学生信息失败: " + err.Error())
	}

	// 转换为UserProfile格式
	var studentProfiles []*models.UserProfile
	for _, student := range students {
		studentProfiles = append(studentProfiles, student.ToProfile())
	}

	return studentProfiles, nil
}

// GetTasksByClazzID 根据班级ID获取任务列表
func (s *ClazzService) GetTasksByClazzID(ctx context.Context, clazzID string, userID string) ([]models.Task, error) {
	clazzObjID, err := primitive.ObjectIDFromHex(clazzID)
	if err != nil {
		return nil, errors.New("无效的班级ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 先验证用户是否有权限查看该班级的任务
	// 获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return nil, err
	}

	// 验证用户权限：必须是班级成员、课程创建者或课程教师
	// 检查用户是否是班级成员
	isMember, err := s.Repo.IsClazzMember(ctx, clazzObjID, userObjID)
	if err != nil {
		return nil, errors.New("检查班级成员失败: " + err.Error())
	}

	// 检查用户是否是课程创建者或教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil {
		return nil, err
	}

	if !isMember && !isAuthorized {
		return nil, errors.New("权限不足")
	}

	// 查询任务和班级的关联关系
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	cursor, err := taskClazzColl.Find(ctx, bson.M{"clazz_id": clazzObjID})
	if err != nil {
		return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var taskClazzRelations []models.TaskClazzRelation
	if err = cursor.All(ctx, &taskClazzRelations); err != nil {
		return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
	}

	// 提取任务ID列表
	taskIDs := make([]primitive.ObjectID, 0, len(taskClazzRelations))
	for _, relation := range taskClazzRelations {
		taskIDs = append(taskIDs, relation.TaskID)
	}

	// 如果没有任务，直接返回空列表
	if len(taskIDs) == 0 {
		return []models.Task{}, nil
	}

	// 查询任务详情
	taskColl := utils.GetCollection("tasks")
	cursor, err = taskColl.Find(ctx, bson.M{"_id": bson.M{"$in": taskIDs}})
	if err != nil {
		return nil, errors.New("查询任务详情失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, errors.New("解析任务详情失败: " + err.Error())
	}

	return tasks, nil
}

// GetTaskByID 根据任务ID获取任务详情
func (s *ClazzService) GetTaskByID(ctx context.Context, taskID string, userID string) (*models.TaskResponse, error) {
	taskObjID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return nil, errors.New("无效的任务ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 获取任务信息
	task, err := s.Repo.GetTaskByID(ctx, taskObjID)
	if err != nil {
		return nil, err
	}

	// 验证用户权限：必须是课程创建者或教师，或者是至少一个关联班级的成员
	// 检查用户是否是课程创建者或教师
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, task.CourseId, userObjID)
	if err != nil {
		return nil, err
	}

	if !isAuthorized {
		// 检查用户是否是至少一个关联班级的成员
		taskClazzColl := utils.GetCollection("task_clazz_relations")
		cursor, err := taskClazzColl.Find(ctx, bson.M{"task_id": taskObjID})
		if err != nil {
			return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
		}
		defer cursor.Close(ctx)

		var taskClazzRelations []models.TaskClazzRelation
		if err = cursor.All(ctx, &taskClazzRelations); err != nil {
			return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
		}

		// 检查用户是否是任何一个关联班级的成员
		isMember := false
		for _, relation := range taskClazzRelations {
			member, err := s.Repo.IsClazzMember(ctx, relation.ClazzID, userObjID)
			if err != nil {
				continue
			}
			if member {
				isMember = true
				break
			}
		}

		if !isMember {
			return nil, errors.New("权限不足")
		}
	}

	// 查询用户的任务完成状态
	state := 0
	userTask, err := s.Repo.GetUserTaskStatus(ctx, taskObjID, userObjID)
	if err != nil {
		return nil, errors.New("查询用户任务状态失败: " + err.Error())
	}
	if userTask != nil && userTask.State == 1 {
		state = 1
	}

	// 查询用户在每道题上的完成情况
	completedQuestions := make(map[primitive.ObjectID]bool)
	userRelationColl := utils.GetCollection("user_relation_question")
	cursor, err := userRelationColl.Find(ctx, bson.M{
		"task_id": taskObjID,
		"user_id": userObjID,
		"state":   1, // 已完成的题目
	})
	if err != nil {
		return nil, errors.New("查询用户题目完成情况失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var userRelations []models.UserRelationQuestion
	if err = cursor.All(ctx, &userRelations); err != nil {
		return nil, errors.New("解析用户题目完成情况失败: " + err.Error())
	}

	for _, ur := range userRelations {
		completedQuestions[ur.RelationID] = true
	}

	// 查询题目详情
	questions := make([]models.QuestionDetail, 0, len(task.RelationIDs))
	if len(task.RelationIDs) > 0 {
		problemColl := utils.GetCollection("problems")
		problemCursor, err := problemColl.Find(ctx, bson.M{
			"_id": bson.M{"$in": task.RelationIDs},
		})
		if err != nil {
			return nil, errors.New("查询题目详情失败: " + err.Error())
		}
		defer problemCursor.Close(ctx)

		var problems []models.Problem
		if err = problemCursor.All(ctx, &problems); err != nil {
			return nil, errors.New("解析题目详情失败: " + err.Error())
		}

		// 创建题目ID到题目的映射
		problemMap := make(map[primitive.ObjectID]models.Problem)
		for _, p := range problems {
			problemMap[p.ID] = p
		}

		// 按照RelationIDs的顺序构建题目详情列表
		for _, relationID := range task.RelationIDs {
			if problem, exists := problemMap[relationID]; exists {
				question := models.QuestionDetail{
					ID:         problem.ID,
					UniqueID:   problem.UniqueID,
					Title:      problem.Title,
					Difficulty: string(problem.Difficulty),
					Completed:  completedQuestions[relationID],
				}
				questions = append(questions, question)
			}
		}
	}

	// 查询任务关联的班级ID列表
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	cursor, err = taskClazzColl.Find(ctx, bson.M{"task_id": taskObjID})
	if err != nil {
		return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var taskClazzRelations []models.TaskClazzRelation
	if err = cursor.All(ctx, &taskClazzRelations); err != nil {
		return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
	}

	// 提取班级ID列表
	clazzIDs := make([]primitive.ObjectID, 0, len(taskClazzRelations))
	for _, relation := range taskClazzRelations {
		clazzIDs = append(clazzIDs, relation.ClazzID)
	}

	// 构建 TaskResponse
	taskResponse := &models.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Type:        task.Type,
		StartTime:   task.StartTime,
		EndTime:     task.EndTime,
		Status:      task.Status,
		CourseId:    task.CourseId,
		CTime:       task.CTime,
		CID:         task.CID,
		MTime:       task.MTime,
		State:       state,
		RelationIDs: task.RelationIDs,
		Questions:   questions,
		ClazzIds:    clazzIDs,
	}

	return taskResponse, nil
}

// 私有方法
// getCourseByID 根据ID获取课程
func (s *ClazzService) getCourseByID(ctx context.Context, courseID primitive.ObjectID) (*models.Course, error) {
	var course models.Course
	err := utils.GetCollection("courses").FindOne(ctx, bson.M{"_id": courseID}).Decode(&course)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("课程不存在")
		}
		return nil, err
	}
	return &course, nil
}

// getUserByID 根据ID获取用户
func (s *ClazzService) getUserByID(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := utils.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// JoinClazz 通过二维码扫描加入班级
func (s *ClazzService) JoinClazz(ctx context.Context, clazzId string, req models.JoinClazzRequest, userID string) error {
	// 验证用户ID和班级ID格式
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	log.Printf("解析班级Obj: %s", clazzId)
	clazzObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return errors.New("班级不存在")
	}

	// 检查是否可以加入
	if !clazz.CanJoin() {
		return errors.New("班级已满或已结束")
	}

	// 检查用户是否已经是班级成员
	isMember, err := s.Repo.IsClazzMember(ctx, clazzObjID, userObjID)
	if err != nil {
		return errors.New("检查班级成员失败: " + err.Error())
	}

	if isMember {
		return errors.New("您已经是该班级成员")
	}

	// 如果班级需要邀请，则验证二维码
	if clazz.RequireInvite {
		if req.InviteCode == nil {
			return errors.New("二维码无效: 请求中未提供邀请码")
		}

		log.Printf("验证二维码, 班级ID: %s, 邀请码: %s", clazzId, *req.InviteCode)
		valid, err := s.Repo.VerifyQRCode(ctx, clazzId, *req.InviteCode)
		if err != nil {
			return err
		}

		if !valid {
			return errors.New("二维码无效: 邀请码不匹配")
		}
	}

	// 执行加入班级的逻辑
	success, err := s.Repo.AddClazzMember(ctx, clazzObjID, userObjID)
	if err != nil {
		return errors.New("加入班级失败: " + err.Error())
	}

	if !success {
		return errors.New("班级不存在或已满")
	}

	// 与 AddClazzMember（教师添加成员）一致：必须写入 student_classes，否则仅 member_ids 有数据会导致「我的班级」等接口查不到
	now := time.Now()
	studentClass := &models.StudentClass{
		ID:        primitive.NewObjectID(),
		StudentID: userObjID,
		ClassID:   clazzObjID,
		CourseID:  clazz.CourseId,
		JoinTime:  now,
		Status:    models.StudentClassStatusActive,
		CTime:     now,
		MTime:     now,
	}
	if err = s.Repo.CreateStudentClassRelation(ctx, studentClass); err != nil {
		_, _ = utils.GetCollection("clazzes").UpdateOne(ctx,
			bson.M{"_id": clazzObjID},
			bson.M{
				"$inc":  bson.M{"add_nums": -1},
				"$pull": bson.M{"member_ids": userObjID},
				"$set":  bson.M{"mtime": time.Now()},
			})
		return errors.New("加入班级失败: " + err.Error())
	}

	return nil
}

// FinishTask 完成任务
func (s *ClazzService) FinishTask(ctx context.Context, clazzId string, taskId string, req models.FinishTaskRequest, userID string) error {
	// 解析ID
	taskObjId, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return errors.New("无效的任务ID")
	}

	relationObjId, err := primitive.ObjectIDFromHex(req.RelationID)
	if err != nil {
		return errors.New("无效的关系ID")
	}

	userObjId, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 检查用户是否是至少一个关联班级的成员
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	cursor, err := taskClazzColl.Find(ctx, bson.M{"task_id": taskObjId})
	if err != nil {
		return errors.New("查询任务和班级关联关系失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var taskClazzRelations []models.TaskClazzRelation
	if err = cursor.All(ctx, &taskClazzRelations); err != nil {
		return errors.New("解析任务和班级关联关系失败: " + err.Error())
	}

	// 检查用户是否是任何一个关联班级的成员
	isMember := false
	for _, relation := range taskClazzRelations {
		member, err := s.Repo.IsClazzMember(ctx, relation.ClazzID, userObjId)
		if err != nil {
			continue
		}
		if member {
			isMember = true
			break
		}
	}

	if !isMember {
		return errors.New("您不是该任务关联班级的成员，无法完成任务")
	}

	// 获取任务信息 - 使用repository方法
	task, err := s.Repo.GetTaskByID(ctx, taskObjId)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.StartTime.After(time.Now()) {
		return errors.New("任务还未开始")
	}

	if task.EndTime != nil && task.EndTime.Before(time.Now()) {
		return errors.New("任务已经结束")
	}

	// 检查 user_relation_question 表中是否已经有完成记录
	userRelationColl := utils.GetCollection("user_relation_question")
	count, err := userRelationColl.CountDocuments(ctx, bson.M{
		"task_id":     taskObjId,
		"user_id":     userObjId,
		"relation_id": relationObjId,
		"state":       1, // 1 表示已完成
	})
	if err != nil {
		return errors.New("检查题目完成状态失败: " + err.Error())
	}

	// 如果已经有完成记录，直接返回
	if count > 0 {
		return nil
	}

	// 在 user_relation_question 表中添加完成记录
	now := time.Now()
	userRelation := &models.UserRelationQuestion{
		ID:         primitive.NewObjectID(),
		RelationID: relationObjId,
		TaskID:     taskObjId,
		UserID:     userObjId,
		State:      1, // 已完成
		CTime:      now,
		MTime:      now,
	}

	_, err = userRelationColl.InsertOne(ctx, userRelation)
	if err != nil {
		return errors.New("记录题目完成状态失败: " + err.Error())
	}

	// 检查是否所有题目都已完成，如果是，则更新任务完成状态
	finishedCount, err := userRelationColl.CountDocuments(ctx, bson.M{
		"task_id": taskObjId,
		"user_id": userObjId,
		"state":   1, // 1 表示已完成
	})

	if err != nil {
		// 记录错误但不中断响应
		log.Printf("查询用户完成题目数量失败: %v", err)
	}

	// 如果已完成题目数等于总题目数，表示任务已完成
	totalQuestions := len(task.RelationIDs)
	isTaskCompleted := int(finishedCount) == totalQuestions && totalQuestions > 0

	// 更新 user_task 表中的状态 - 使用repository方法
	err = s.Repo.UpdateUserTaskStatus(ctx, taskObjId, userObjId, int(finishedCount), isTaskCompleted)
	if err != nil {
		// 记录错误但不中断响应
		log.Printf("更新用户任务状态失败: %v", err)
	}

	// 如果任务已完成，更新任务的完成用户列表
	if isTaskCompleted {
		filter := bson.M{"_id": taskObjId}
		update := bson.M{
			"$addToSet": bson.M{
				"finish_ids": userObjId,
			},
			"$set": bson.M{
				"mtime": time.Now(),
			},
		}

		collTasks := utils.GetCollection("tasks")
		_, err = collTasks.UpdateOne(ctx, filter, update)
		if err != nil {
			// 记录错误但不中断响应
			log.Printf("更新任务完成状态失败: %v", err)
		}
	}

	return nil
}

// UpdateTask 更新任务
func (s *ClazzService) UpdateTask(ctx context.Context, userId string, clazzId string, taskId string, req models.UpdateTaskRequest) error {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}
	courseObjId, err := primitive.ObjectIDFromHex(req.CourseId)
	if err != nil {
		return err
	}
	// 不再验证班级ID，因为任务现在是独立于班级的
	taskObjId, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return err
	}
	if flag, err := authorized(ctx, utils.GetCollection("courses"), courseObjId, userObjId); err != nil || !flag {
		if err != nil {
			return err
		}
		return errors.New("权限不足")
	}

	// 构建更新字段
	updateFields := bson.M{}
	// 检查每个字段，如果字段非空，则加入更新内容
	if req.Title != nil {
		updateFields["title"] = *req.Title
	}
	if req.Description != nil {
		updateFields["description"] = *req.Description
	}
	if req.Type != nil {
		updateFields["type"] = *req.Type
	}
	if req.StartTime != nil {
		updateFields["start_time"] = *req.StartTime
	}
	if req.EndTime != nil {
		updateFields["end_time"] = *req.EndTime
	}
	if req.RelationIDs != nil {
		// 转换RelationIDs为ObjectID数组
		ids := make([]primitive.ObjectID, len(*req.RelationIDs))
		for i, id := range *req.RelationIDs {
			hex, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return err
			}
			ids[i] = hex
		}
		updateFields["relation_ids"] = ids
	}
	updateFields["mtime"] = time.Now()

	// 更新任务
	coll := utils.GetCollection("tasks")
	filter := bson.M{"_id": taskObjId}
	_, err = coll.UpdateOne(ctx, filter, bson.M{"$set": updateFields})
	if err != nil {
		return err
	}

	return nil
}

// AddTaskRelationIds 为任务添加关系ID
func (s *ClazzService) AddTaskRelationIds(ctx context.Context, userId string, taskId string, req models.AddTaskRelationIdsRequest) error {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	taskObjId, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return err
	}

	// 获取任务信息以验证权限
	task, err := s.Repo.GetTaskByID(ctx, taskObjId)
	if err != nil {
		return err
	}

	// 验证权限（只有课程创建者或班级教师可以修改任务）
	if flag, err := s.Repo.CheckCourseAuthorization(ctx, task.CourseId, userObjId); err != nil || !flag {
		if err != nil {
			return err
		}
		return errors.New("权限不足")
	}

	// 转换RelationIDs
	relationObjIds, err := s.Repo.ConvertToObjectIDs(req.RelationIDs)
	if err != nil {
		return err
	}

	// 使用repository层方法添加关系ID
	updated, err := s.Repo.AddTaskRelationIds(ctx, taskObjId, relationObjIds, userObjId)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("找不到该任务")
	}

	return nil
}

// RemoveTaskRelationIds 从任务中删除关系ID
func (s *ClazzService) RemoveTaskRelationIds(ctx context.Context, userId string, taskId string, req models.RemoveTaskRelationIdsRequest) error {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	taskObjId, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return err
	}

	// 获取任务信息以验证权限
	task, err := s.Repo.GetTaskByID(ctx, taskObjId)
	if err != nil {
		return err
	}

	// 验证权限（只有课程创建者或班级教师可以修改任务）
	if flag, err := s.Repo.CheckCourseAuthorization(ctx, task.CourseId, userObjId); err != nil || !flag {
		if err != nil {
			return err
		}
		return errors.New("权限不足")
	}

	// 转换RelationIDs
	relationObjIds, err := s.Repo.ConvertToObjectIDs(req.RelationIDs)
	if err != nil {
		return err
	}

	// 使用repository层方法删除关系ID
	updated, err := s.Repo.RemoveTaskRelationIds(ctx, taskObjId, relationObjIds, userObjId)
	if err != nil {
		return err
	}
	if !updated {
		return errors.New("找不到该任务")
	}

	return nil
}

// PageQueryTaskCompletion 分页查询班级任务完成情况
func (s *ClazzService) PageQueryTaskCompletion(ctx context.Context, clazzId string, taskId string, req *models.PageQueryTaskCompletionRequest) (*models.PageQueryTaskCompletionResponse, error) {
	// 解析任务ID
	taskObjID, err := primitive.ObjectIDFromHex(taskId)
	if err != nil {
		return nil, errors.New("无效的任务ID")
	}

	// 解析班级ID
	classObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return nil, errors.New("无效的班级ID")
	}

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
	if pageSize > 50 {
		pageSize = 50
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageNum < 1 {
		pageNum = 1
	}

	// 验证班级是否存在
	_, err = s.Repo.GetClazzByID(ctx, classObjID)
	if err != nil {
		return nil, err
	}

	// 分页查询班级学生关联记录
	var studentClasses []models.StudentClass
	var total int64
	var studentIDs []primitive.ObjectID
	var userIDs []primitive.ObjectID

	// 如果指定了学生ID进行精确查询
	if req.UserID != nil && *req.UserID != "" {
		userObjID, err := primitive.ObjectIDFromHex(*req.UserID)
		if err != nil {
			return nil, errors.New("无效的学生ID")
		}
		userIDs = append(userIDs, userObjID)

		// 查询班级学生总数
		studentClassColl := utils.GetCollection("student_classes")
		total, err = studentClassColl.CountDocuments(ctx, bson.M{
			"class_id": classObjID,
			"status":   "active",
		})
		if err != nil {
			return nil, errors.New("查询班级学生总数失败: " + err.Error())
		}
	} else {
		// 分页查询班级学生关联记录
		studentClasses, total, err = s.Repo.GetClassStudentsWithPagination(ctx, classObjID, pageNum, pageSize)
		if err != nil {
			return nil, errors.New("查询班级学生关联记录失败: " + err.Error())
		}

		// 收集学生ID
		studentIDs = make([]primitive.ObjectID, len(studentClasses))
		for i, sc := range studentClasses {
			studentIDs[i] = sc.StudentID
		}
		userIDs = studentIDs
	}

	// 查询学生信息
	realNameFilter := ""
	if req.RealName != nil {
		realNameFilter = *req.RealName
	}
	students, err := s.Repo.GetUsersByIDs(ctx, userIDs, realNameFilter)
	if err != nil {
		return nil, errors.New("查询学生信息失败: " + err.Error())
	}

	// 构建学生ID到学生信息的映射
	studentMap := make(map[primitive.ObjectID]*models.User)
	for _, student := range students {
		studentMap[student.ID] = student
	}

	// 查询任务信息
	task, err := s.Repo.GetTaskByID(ctx, taskObjID)
	if err != nil {
		return nil, err
	}

	totalQuestions := len(task.RelationIDs)

	// 查询这些学生的任务完成情况
	userTasks, err := s.Repo.GetUserTaskStatuses(ctx, taskObjID, userIDs)
	if err != nil {
		return nil, errors.New("查询任务完成情况失败: " + err.Error())
	}

	// 构建用户ID到任务完成情况的映射
	userTaskMap := make(map[primitive.ObjectID]*models.UserTask)
	for _, userTask := range userTasks {
		userTaskMap[userTask.UserID] = &userTask
	}

	// 构建响应数据
	completion := make([]models.TaskCompletionResponse, len(students))
	for i, student := range students {
		userTask, exists := userTaskMap[student.ID]
		completed := false
		var completedAt *time.Time
		finishedQuestions := 0

		if exists {
			completed = userTask.State == 1
			completedAt = userTask.CompletedAt
			finishedQuestions = userTask.FinishedCount
		}

		completion[i] = models.TaskCompletionResponse{
			UserID:            student.ID,
			RealName:          student.RealName,
			StudentID:         student.StudentID,
			Completed:         completed,
			CompletedAt:       completedAt,
			TotalQuestions:    totalQuestions,
			FinishedQuestions: finishedQuestions,
		}
	}

	res := &models.PageQueryTaskCompletionResponse{
		Total:      total,
		PageNum:    pageNum,
		PageSize:   pageSize,
		Completion: completion,
	}

	return res, nil
}

// PageQueryUserTasks 分页查询用户所在所有班级的所有任务列表
func (s *ClazzService) PageQueryUserTasks(ctx context.Context, userID string, req *models.PageQueryUserTasksFilterRequest) (*models.PageQueryUserTasksResponse, error) {
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 1. 获取用户所在的所有班级
	studentClasses, err := s.Repo.GetStudentClasses(ctx, userObjID)
	if err != nil {
		return nil, errors.New("查询用户班级失败: " + err.Error())
	}

	if len(studentClasses) == 0 {
		return &models.PageQueryUserTasksResponse{
			Total:    0,
			PageNum:  1,
			PageSize: 10,
			Tasks:    []models.TaskResponse{},
		}, nil
	}

	// 2. 提取所有班级ID
	classIDs := make([]primitive.ObjectID, 0, len(studentClasses))
	for _, sc := range studentClasses {
		classIDs = append(classIDs, sc.ClassID)
	}

	// 3. 查询任务和班级的关联关系，获取用户所在班级的任务ID列表
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	cursor, err := taskClazzColl.Find(ctx, bson.M{"clazz_id": bson.M{"$in": classIDs}})
	if err != nil {
		return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var taskClazzRelations []models.TaskClazzRelation
	if err = cursor.All(ctx, &taskClazzRelations); err != nil {
		return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
	}

	if len(taskClazzRelations) == 0 {
		return &models.PageQueryUserTasksResponse{
			Total:    0,
			PageNum:  1,
			PageSize: 10,
			Tasks:    []models.TaskResponse{},
		}, nil
	}

	// 4. 提取任务ID列表
	taskIDs := make([]primitive.ObjectID, 0, len(taskClazzRelations))
	for _, relation := range taskClazzRelations {
		taskIDs = append(taskIDs, relation.TaskID)
	}

	// 5. 构建任务查询条件
	taskFilter := bson.M{"_id": bson.M{"$in": taskIDs}}

	// 6. 应用可选的筛选条件
	if req.State != nil {
		// 查询用户的任务状态
		userTasks, err := s.Repo.GetUserTaskStatusesByUserID(ctx, userObjID)
		if err != nil {
			return nil, errors.New("查询用户任务状态失败: " + err.Error())
		}

		if *req.State == 1 {
			// 查询已完成的任务
			if len(userTasks) == 0 {
				return &models.PageQueryUserTasksResponse{
					Total:    0,
					PageNum:  1,
					PageSize: 10,
					Tasks:    []models.TaskResponse{},
				}, nil
			}

			// 提取任务ID
			completedTaskIDs := make([]primitive.ObjectID, 0, len(userTasks))
			for _, ut := range userTasks {
				if ut.State == 1 {
					completedTaskIDs = append(completedTaskIDs, ut.TaskID)
				}
			}

			if len(completedTaskIDs) == 0 {
				return &models.PageQueryUserTasksResponse{
					Total:    0,
					PageNum:  1,
					PageSize: 10,
					Tasks:    []models.TaskResponse{},
				}, nil
			}

			// 交集：用户所在班级的任务 && 已完成的任务
			completedTaskIDMap := make(map[primitive.ObjectID]bool)
			for _, id := range completedTaskIDs {
				completedTaskIDMap[id] = true
			}

			filteredTaskIDs := make([]primitive.ObjectID, 0)
			for _, id := range taskIDs {
				if completedTaskIDMap[id] {
					filteredTaskIDs = append(filteredTaskIDs, id)
				}
			}

			if len(filteredTaskIDs) == 0 {
				return &models.PageQueryUserTasksResponse{
					Total:    0,
					PageNum:  1,
					PageSize: 10,
					Tasks:    []models.TaskResponse{},
				}, nil
			}

			taskFilter["_id"] = bson.M{"$in": filteredTaskIDs}
		} else if *req.State == 0 {
			// 查询未完成的任务：包括用户从未做过的任务和已开始但未完成的任务
			// 先查询所有已完成的任务ID
			completedTaskIDs := make([]primitive.ObjectID, 0)
			for _, ut := range userTasks {
				if ut.State == 1 {
					completedTaskIDs = append(completedTaskIDs, ut.TaskID)
				}
			}

			// 如果有已完成的任务，则排除这些任务
			if len(completedTaskIDs) > 0 {
				// 差集：用户所在班级的任务 - 已完成的任务
				completedTaskIDMap := make(map[primitive.ObjectID]bool)
				for _, id := range completedTaskIDs {
					completedTaskIDMap[id] = true
				}

				filteredTaskIDs := make([]primitive.ObjectID, 0)
				for _, id := range taskIDs {
					if !completedTaskIDMap[id] {
						filteredTaskIDs = append(filteredTaskIDs, id)
					}
				}

				if len(filteredTaskIDs) == 0 {
					return &models.PageQueryUserTasksResponse{
						Total:    0,
						PageNum:  1,
						PageSize: 10,
						Tasks:    []models.TaskResponse{},
					}, nil
				}

				taskFilter["_id"] = bson.M{"$in": filteredTaskIDs}
			}
		}
	}

	if req.Type != nil {
		taskFilter["type"] = *req.Type
	}

	if req.CourseID != nil {
		courseObjID, err := primitive.ObjectIDFromHex(*req.CourseID)
		if err != nil {
			return nil, errors.New("invalid course id")
		}

		taskFilter["course_id"] = courseObjID
	}

	if req.ClazzID != nil {
		// 当指定了班级ID时，重新查询该班级的任务
		clazzObjID, err := primitive.ObjectIDFromHex(*req.ClazzID)
		if err != nil {
			return nil, errors.New("无效的班级ID")
		}

		// 检查该班级是否在用户所在的班级列表中
		isInClass := false
		for _, classID := range classIDs {
			if classID == clazzObjID {
				isInClass = true
				break
			}
		}

		if !isInClass {
			return &models.PageQueryUserTasksResponse{
				Total:    0,
				PageNum:  1,
				PageSize: 10,
				Tasks:    []models.TaskResponse{},
			}, nil
		}

		// 查询该班级的任务
		cursor, err := taskClazzColl.Find(ctx, bson.M{"clazz_id": clazzObjID})
		if err != nil {
			return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
		}
		defer cursor.Close(ctx)

		var classTaskRelations []models.TaskClazzRelation
		if err = cursor.All(ctx, &classTaskRelations); err != nil {
			return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
		}

		if len(classTaskRelations) == 0 {
			return &models.PageQueryUserTasksResponse{
				Total:    0,
				PageNum:  1,
				PageSize: 10,
				Tasks:    []models.TaskResponse{},
			}, nil
		}

		// 提取任务ID
		classTaskIDs := make([]primitive.ObjectID, 0, len(classTaskRelations))
		for _, relation := range classTaskRelations {
			classTaskIDs = append(classTaskIDs, relation.TaskID)
		}

		taskFilter["_id"] = bson.M{"$in": classTaskIDs}
	}

	// 添加对Status字段的支持
	if req.Status != nil {
		taskFilter["status"] = *req.Status
	}

	// 7. 执行分页查询
	// 默认分页参数
	pageNum := int64(1)
	pageSize := int64(10)
	if req.PageNum != nil && *req.PageNum > 0 {
		pageNum = *req.PageNum
	}
	if req.PageSize != nil && *req.PageSize > 0 {
		pageSize = *req.PageSize
	}

	// 查询任务列表
	tasks, total, err := s.Repo.GetTasksByFilterWithPagination(ctx, taskFilter, pageNum, pageSize)
	if err != nil {
		return nil, errors.New("查询任务列表失败: " + err.Error())
	}

	// 8. 构建响应
	taskResponses := make([]models.TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		// 查询用户的任务完成状态
		userTask, err := s.Repo.GetUserTaskStatus(ctx, task.ID, userObjID)
		state := 0
		if err == nil && userTask != nil && userTask.State == 1 {
			state = 1
		}

		// 查询用户在每道题上的完成情况
		completedQuestions, err := s.Repo.GetUserCompletedQuestions(ctx, task.ID, userObjID)
		if err != nil {
			return nil, errors.New("查询用户题目完成情况失败: " + err.Error())
		}

		// 查询题目详情
		questions := make([]models.QuestionDetail, 0, len(task.RelationIDs))
		if len(task.RelationIDs) > 0 {
			problems, err := s.Repo.GetProblemsByIDs(ctx, task.RelationIDs)
			if err != nil {
				return nil, errors.New("查询题目详情失败: " + err.Error())
			}

			// 创建题目ID到题目的映射
			problemMap := make(map[primitive.ObjectID]models.Problem)
			for _, p := range problems {
				problemMap[p.ID] = p
			}

			// 按照RelationIDs的顺序构建题目详情列表
			for _, relationID := range task.RelationIDs {
				if problem, exists := problemMap[relationID]; exists {
					question := models.QuestionDetail{
						ID:         problem.ID,
						UniqueID:   problem.UniqueID,
						Title:      problem.Title,
						Difficulty: string(problem.Difficulty),
						Completed:  completedQuestions[relationID],
					}
					questions = append(questions, question)
				}
			}
		}

		// 查询任务关联的班级ID列表
		cursor, err := taskClazzColl.Find(ctx, bson.M{"task_id": task.ID})
		if err != nil {
			return nil, errors.New("查询任务和班级关联关系失败: " + err.Error())
		}
		defer cursor.Close(ctx)

		var taskRelations []models.TaskClazzRelation
		if err = cursor.All(ctx, &taskRelations); err != nil {
			return nil, errors.New("解析任务和班级关联关系失败: " + err.Error())
		}

		// 提取班级ID列表
		clazzIDs := make([]primitive.ObjectID, 0, len(taskRelations))
		for _, relation := range taskRelations {
			clazzIDs = append(clazzIDs, relation.ClazzID)
		}

		// 构建任务响应
		taskResponse := models.TaskResponse{
			ID:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Type:        task.Type,
			StartTime:   task.StartTime,
			EndTime:     task.EndTime,
			Status:      task.Status,
			CourseId:    task.CourseId,
			CTime:       task.CTime,
			CID:         task.CID,
			MTime:       task.MTime,
			State:       state,
			RelationIDs: task.RelationIDs,
			Questions:   questions,
			ClazzIds:    clazzIDs,
		}

		taskResponses = append(taskResponses, taskResponse)
	}

	return &models.PageQueryUserTasksResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Tasks:    taskResponses,
	}, nil
}

// AddMajorClassAssociations 添加专业班级关联
func (s *ClazzService) AddMajorClassAssociations(ctx context.Context, clazzID string, userId string, req *models.MajorClassIdsRequest) error {
	// 转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 转换用户ID
	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil || !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以修改班级")
	}

	// 转换专业班级ID数组
	majorClassObjectIDs, err := s.Repo.ConvertToObjectIDs(req.MajorClassIDs)
	if err != nil {
		return errors.New("无效的专业班级ID")
	}

	// 创建新的关联ID数组（去重）
	currentMap := make(map[primitive.ObjectID]bool)
	var newAssociations []primitive.ObjectID
	var addedClasses []primitive.ObjectID

	// 先将所有现有的关联ID添加到newAssociations和currentMap
	for _, id := range clazz.MajorClassIDs {
		currentMap[id] = true
		newAssociations = append(newAssociations, id)
	}

	// 添加新的关联ID并记录新增的班级
	for _, id := range majorClassObjectIDs {
		if !currentMap[id] {
			currentMap[id] = true
			newAssociations = append(newAssociations, id)
			addedClasses = append(addedClasses, id)
		}
	}

	// 更新专业班级关联
	if err := s.Repo.UpdateMajorClassAssociations(ctx, clazzObjID, newAssociations); err != nil {
		return err
	}

	// 为新增的班级添加学生
	for _, majorClassID := range addedClasses {
		err = s.batchAddMajorClassStudents(ctx, majorClassID, clazzObjID, clazz.CourseId)
		if err != nil {
			// 移除刚刚添加的关联，保持数据一致性
			if err := s.Repo.UpdateMajorClassAssociations(ctx, clazzObjID, clazz.MajorClassIDs); err != nil {
				log.Printf("回滚专业班级关联失败: %v", err)
			}
			return err
		}
	}

	// 清除与该班级相关的所有缓存
	cachePattern := "clazz_detail:" + clazzID + ":*"
	keys, err := utils.RedisClient.Keys(ctx, cachePattern).Result()
	if err != nil {
		log.Printf("获取缓存键失败: %v", err)
	} else if len(keys) > 0 {
		if err := s.Repo.DeleteCache(ctx, keys...); err != nil {
			log.Printf("清除缓存失败: %v", err)
		}
	}

	return nil
}

// RemoveMajorClassAssociations 移除专业班级关联
func (s *ClazzService) RemoveMajorClassAssociations(ctx context.Context, clazzID string, userId string, req *models.MajorClassIdsRequest) error {
	// 转换班级ID
	clazzObjID, err := s.Repo.ConvertToObjectID(clazzID)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	// 转换用户ID
	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 获取班级信息
	clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
	if err != nil {
		return err
	}

	// 验证权限
	isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
	if err != nil || !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以修改班级")
	}

	// 转换专业班级ID数组
	majorClassObjectIDs, err := s.Repo.ConvertToObjectIDs(req.MajorClassIDs)
	if err != nil {
		return errors.New("无效的专业班级ID")
	}

	// 创建要移除的ID映射
	removeMap := make(map[primitive.ObjectID]bool)
	for _, id := range majorClassObjectIDs {
		removeMap[id] = true
	}

	// 创建新的关联ID数组（排除要移除的ID）
	var newAssociations []primitive.ObjectID
	var removedClasses []primitive.ObjectID

	// 过滤掉要移除的关联ID
	for _, id := range clazz.MajorClassIDs {
		if !removeMap[id] {
			newAssociations = append(newAssociations, id)
		} else {
			removedClasses = append(removedClasses, id)
		}
	}

	// 更新专业班级关联
	if err := s.Repo.UpdateMajorClassAssociations(ctx, clazzObjID, newAssociations); err != nil {
		return err
	}

	// 移除不再关联班级的学生
	if len(removedClasses) > 0 {
		// 提取所有被移除专业班级的学生ID
		removedStudentsMap := make(map[primitive.ObjectID]bool)
		for _, majorClassID := range removedClasses {
			// 获取专业班级信息，包括 member_ids 字段
			majorClass, err := s.Repo.GetClazzByID(ctx, majorClassID)
			if err != nil {
				log.Printf("获取专业班级信息失败: %v", err)
				continue
			}
			// 验证是否为专业班级
			if majorClass.ClassType != models.ClassTypeMajor {
				continue
			}
			// 添加专业班级的学生ID到移除列表
			for _, studentID := range majorClass.MemberIDs {
				removedStudentsMap[studentID] = true
			}
		}

		// 如果还有其他关联班级，过滤掉这些班级的学生
		if len(newAssociations) > 0 {
			for _, majorClassID := range newAssociations {
				// 获取专业班级信息，包括 member_ids 字段
				majorClass, err := s.Repo.GetClazzByID(ctx, majorClassID)
				if err != nil {
					log.Printf("获取专业班级信息失败: %v", err)
					continue
				}
				// 验证是否为专业班级
				if majorClass.ClassType != models.ClassTypeMajor {
					continue
				}
				// 过滤掉剩余班级的学生
				for _, studentID := range majorClass.MemberIDs {
					delete(removedStudentsMap, studentID)
				}
			}
		}

		// 转换为切片
		var studentsToRemove []primitive.ObjectID
		for studentID := range removedStudentsMap {
			studentsToRemove = append(studentsToRemove, studentID)
		}

		// 批量删除学生，直接操作 clazzes 表和 student_classes 表
		if len(studentsToRemove) > 0 {
			// 获取课程班级信息
			courseClass, err := s.Repo.GetClazzByID(ctx, clazzObjID)
			if err != nil {
				log.Printf("获取课程班级信息失败: %v", err)
				return err
			}

			// 计算需要移除的学生数量
			removeCount := 0
			for _, studentID := range studentsToRemove {
				// 检查学生是否在班级的 member_ids 中
				for _, mid := range courseClass.MemberIDs {
					if mid == studentID {
						removeCount++
						break
					}
				}
			}

			if removeCount > 0 {
				// 1. 直接从 clazzes 表中移除学生
				clazzColl := utils.GetCollection("clazzes")
				update := bson.M{
					"$inc":  bson.M{"add_nums": -removeCount},
					"$pull": bson.M{"member_ids": bson.M{"$in": studentsToRemove}},
					"$set":  bson.M{"mtime": time.Now()},
				}
				_, err := clazzColl.UpdateOne(ctx, bson.M{"_id": clazzObjID}, update)
				if err != nil {
					log.Printf("批量删除学生失败: %v", err)
					return err
				}

				// 2. 更新 student_classes 表中的记录状态
				studentClassColl := utils.GetCollection("student_classes")
				_, err = studentClassColl.UpdateMany(ctx,
					bson.M{
						"student_id": bson.M{"$in": studentsToRemove},
						"class_id":   clazzObjID,
						"status":     models.StudentClassStatusActive,
					},
					bson.M{
						"$set": bson.M{
							"status": models.StudentClassStatusDropped,
							"mtime":  time.Now(),
						},
					},
				)
				if err != nil {
					log.Printf("更新 student_classes 记录失败: %v", err)
				}
			}
		}
	}

	// 清除与该班级相关的所有缓存
	cachePattern := "clazz_detail:" + clazzID + ":*"
	keys, err := utils.RedisClient.Keys(ctx, cachePattern).Result()
	if err != nil {
		log.Printf("获取缓存键失败: %v", err)
	} else if len(keys) > 0 {
		if err := s.Repo.DeleteCache(ctx, keys...); err != nil {
			log.Printf("清除缓存失败: %v", err)
		}
	}

	return nil
}

// UpdateTasksStatus 根据当前时间更新所有任务的状态
// 状态说明：0-未开始，1-进行中，2-已结束
func (s *ClazzService) UpdateTasksStatus(ctx context.Context) error {
	// 获取当前时间
	now := time.Now()

	// 1. 更新已到开始时间但状态仍为未开始的任务为进行中
	pendingFilter := bson.M{
		"start_time": bson.M{"$lte": now},         // 任务已到开始时间
		"status":     models.TaskStatusNotStarted, // 状态为未开始
	}
	pendingUpdate := bson.M{
		"status": models.TaskStatusActive,
		"mtime":  now,
	}

	// 2. 更新已过期但状态仍为进行中的任务为已结束
	expiredFilter := bson.M{
		"end_time": bson.M{"$lte": now},     // 任务已过期
		"status":   models.TaskStatusActive, // 状态为进行中
	}
	expiredUpdate := bson.M{
		"status": models.TaskStatusEnded,
		"mtime":  now,
	}

	// 使用 UpdateMany 一次性更新所有符合条件的任务
	coll := utils.GetCollection("tasks")

	// 更新未开始任务为进行中
	pendingResult, err := coll.UpdateMany(ctx, pendingFilter, bson.M{"$set": pendingUpdate})
	if err != nil {
		log.Printf("更新未开始任务状态失败: %v", err)
		return err
	}

	// 更新过期任务为已结束
	expiredResult, err := coll.UpdateMany(ctx, expiredFilter, bson.M{"$set": expiredUpdate})
	if err != nil {
		log.Printf("更新过期任务状态失败: %v", err)
		return err
	}

	log.Printf("成功更新 %d 个未开始任务的状态为进行中，%d 个过期任务的状态为已结束", pendingResult.ModifiedCount, expiredResult.ModifiedCount)
	return nil
}

// BindTeacherToClazzes 将教师绑定到多个班级
func (s *ClazzService) BindTeacherToClazzes(ctx context.Context, teacherId string, clazzIds []string, userId string) error {
	// 验证用户权限（只有课程创建者或课程教师才能添加班级教师）
	userObjID, err := s.Repo.ConvertToObjectID(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 验证并转换教师ID
	teacherObjID, err := s.Repo.ConvertToObjectID(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 验证并转换所有班级ID
	clazzObjIDs, err := s.Repo.ConvertToObjectIDs(clazzIds)
	if err != nil {
		return err
	}

	// 遍历所有班级，为每个班级添加教师
	for _, clazzObjID := range clazzObjIDs {
		// 检查班级是否存在
		clazz, err := s.Repo.GetClazzByID(ctx, clazzObjID)
		if err != nil {
			return err
		}

		// 验证操作者是否有权限添加教师
		isAuthorized, err := s.Repo.CheckCourseAuthorization(ctx, clazz.CourseId, userObjID)
		if err != nil {
			return err
		}

		if !isAuthorized {
			return errors.New("权限不足，只有课程创建者或课程教师可以添加班级教师")
		}

		// 为班级添加教师
		_, err = s.Repo.AddClazzTeacher(ctx, clazzObjID, teacherObjID)
		if err != nil && !strings.Contains(err.Error(), "教师已存在") {
			return errors.New("添加班级教师失败: " + err.Error())
		}
	}

	return nil
}
