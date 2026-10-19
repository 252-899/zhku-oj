/*
@Author: sir
@Date: 2025/9/24
@Name: service_course.go
@Description: 课程服务层实现（支持多班级嵌入式架构）
*/

package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"strings"
	"time"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"github.com/gabriel-vasile/mimetype"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CourseService 课程服务
type CourseService struct{}

// NewCourseService 创建课程服务实例
func NewCourseService() *CourseService {
	return &CourseService{}
}

// CreateCourse 创建课程
func (s *CourseService) CreateCourse(ctx context.Context, req *models.CreateCourseRequest, creatorID string) (string, error) {
	// 验证创建者权限
	creatorObjID, err := primitive.ObjectIDFromHex(creatorID)
	if err != nil {
		return "", errors.New("无效的用户ID")
	}

	// 创建课程对象，将创建者默认添加为教师
	now := time.Now()
	course := &models.Course{
		ID:          primitive.NewObjectID(),
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   creatorObjID,
		TeacherIds:  []primitive.ObjectID{creatorObjID}, // 将创建者默认添加为教师
		Status:      models.CourseStatusActive,
		CTime:       now,
		MTime:       now,
	}

	// 插入数据库
	_, err = utils.GetCollection("courses").InsertOne(ctx, course)
	if err != nil {
		log.Printf("创建课程失败: %v", err)
		return "", errors.New("创建课程失败")
	}

	return course.ID.Hex(), nil
}

// GetCourseByID 根据ID获取课程详情
func (s *CourseService) GetCourseByID(ctx context.Context, courseID string, userId string) (*models.CourseResponse, error) {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}

	course, err := s.getCourseByID(ctx, courseObjID)
	if err != nil {
		return nil, errors.New("课程不存在")
	}

	//校验权限
	exist := contains(course.TeacherIds, userId)
	if !exist && course.CreatedBy.Hex() != userId {
		return nil, errors.New("权限不足")
	}

	// 获取创建者信息
	creator, err := s.getUserByID(ctx, course.CreatedBy)
	if err != nil {
		return nil, errors.New("获取创建者信息失败: " + err.Error())
	}

	var creatorProfile *models.UserProfile
	if creator != nil {
		creatorProfile = creator.ToProfile()
	}

	// 获取教师及其班级信息
	var teachers []models.TeacherWithClasses
	for _, teacherID := range course.TeacherIds {
		// 获取教师信息
		teacher, err := s.getUserByID(ctx, teacherID)
		if err != nil {
			continue // 忽略获取失败的教师
		}

		// 获取教师对应的班级
		var classes []models.Clazz
		clazzCursor, err := utils.GetCollection("clazzes").Find(ctx, bson.M{
			"course_id":   courseObjID,
			"teacher_ids": teacherID,
		})
		if err != nil {
			continue // 忽略获取失败的班级
		}
		if err := clazzCursor.All(ctx, &classes); err != nil {
			clazzCursor.Close(ctx)
			continue // 忽略获取失败的班级
		}
		clazzCursor.Close(ctx)

		for i := range classes {
			classes[i].SyncAddNumsFromMemberIDs()
		}

		// 添加教师及其班级信息
		teachers = append(teachers, models.TeacherWithClasses{
			UserProfile: teacher.ToProfile(),
			Classes:     classes,
		})
	}

	return &models.CourseResponse{
		ID:            course.ID,
		Name:          course.Name,
		Avatar:        course.Avatar,
		Description:   course.Description,
		TeacherIds:    course.TeacherIds,
		Status:        course.Status,
		CTime:         course.CTime,
		CreatedByUser: creatorProfile,
		Teachers:      teachers, // 教师及其班级信息
	}, nil
}

// CanManageCourse 判断用户是否具备课程管理权限。
func (s *CourseService) CanManageCourse(ctx context.Context, courseID primitive.ObjectID, userID primitive.ObjectID) (bool, error) {
	// 题目挂载到课程时只允许课程创建者或任课教师操作，避免教师把题目挂到无权限课程。
	return authorized(ctx, utils.GetCollection("courses"), courseID, userID)
}

// CanViewCourse 判断用户是否可以查看课程内资源。
func (s *CourseService) CanViewCourse(ctx context.Context, courseID primitive.ObjectID, userID primitive.ObjectID) (bool, error) {
	// 课程题库对课程创建者、任课教师和已加入课程班级的学生可见。
	return courseMemberAuthorized(ctx, utils.GetCollection("courses"), courseID, userID)
}

func (s *CourseService) UpdateCourseAvatar(ctx context.Context, file multipart.File, header *multipart.FileHeader, userId string, courseId string) error {
	courseObjId, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return err
	}

	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	coll := utils.GetCollection("courses")
	if auth, err := authorized(ctx, coll, courseObjId, userObjId); err != nil || auth != true {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("查不到课程该课程")
		}
		return errors.New("权限不足")
	}
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// 验证文件类型 - 只允许常见的图片格式
	mtype := mimetype.Detect(fileBytes)
	allowedTypes := []string{"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml"}
	isValidType := false
	for _, allowedType := range allowedTypes {
		if mtype.String() == allowedType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("不支持的图片格式，仅支持 JPG、PNG、GIF、WEBP 和 SVG 格式")
	}

	// 将文件内容转为Base64字符串
	base64Str := base64.StdEncoding.EncodeToString(fileBytes)

	filter := bson.M{"_id": courseObjId}
	update := bson.M{"$set": bson.M{"avatar": base64Str}}
	if err = coll.FindOneAndUpdate(ctx, filter, update).Err(); err != nil {
		return err
	}
	return nil
}

// AddCourseTask 添加课程任务
func (s *CourseService) AddCourseTask(ctx context.Context, req *models.AddTaskRequest, userId string) error {
	courseObjId, err := primitive.ObjectIDFromHex(req.CourseId)
	if err != nil {
		return err
	}
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}

	// 转换班级ID数组为ObjectID
	clazzObjIds := make([]primitive.ObjectID, len(req.ClazzIds))
	for i, id := range req.ClazzIds {
		hex, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		clazzObjIds[i] = hex
	}

	if flag, err := authorized(ctx, utils.GetCollection("courses"), courseObjId, userObjId); !flag || err != nil {
		if err != nil {
			return err
		}
		return errors.New("权限不足")
	}

	// StartTime 是必需的，不使用指针
	startTime := req.StartTime
	status := models.TaskStatusNotStarted

	// 检查当前时间与开始时间和结束时间的关系
	now := time.Now()
	if startTime.Before(now) || startTime.Equal(now) {
		// 开始时间已过，检查是否已结束
		if req.EndTime != nil && req.EndTime.Before(now) {
			status = models.TaskStatusEnded // 已结束
		} else {
			status = models.TaskStatusActive // 进行中
		}
	} else {
		status = models.TaskStatusNotStarted // 未开始
	}

	if req.EndTime != nil && req.EndTime.Before(startTime) {
		return errors.New("无效的时间选择")
	}

	// 转换RelationIDs为ObjectID数组
	ids := make([]primitive.ObjectID, 0)
	if req.RelationIDs != nil {
		ids = make([]primitive.ObjectID, len(req.RelationIDs))
		for i, id := range req.RelationIDs {
			hex, err := primitive.ObjectIDFromHex(id)
			if err != nil {
				return err
			}
			ids[i] = hex
		}
	}

	// 创建一个任务
	task := &models.Task{
		ID:          primitive.NewObjectID(),
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		StartTime:   startTime,
		EndTime:     req.EndTime,
		RelationIDs: ids,
		Status:      status,
		CourseId:    courseObjId,
		CTime:       now,
		CID:         userObjId,
		MTime:       now,
	}

	// 插入到独立的tasks集合中
	coll := utils.GetCollection("tasks")
	_, err = coll.InsertOne(ctx, task)
	if err != nil {
		return err
	}

	// 为每个班级创建任务和班级的关联关系
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	for _, clazzObjId := range clazzObjIds {
		taskClazzRelation := &models.TaskClazzRelation{
			ID:       primitive.NewObjectID(),
			TaskID:   task.ID,
			ClazzID:  clazzObjId,
			CourseID: courseObjId,
			CTime:    now,
			MTime:    now,
		}
		_, err = taskClazzColl.InsertOne(ctx, taskClazzRelation)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *CourseService) PageQueryCourse(ctx context.Context, request *models.PageQueryCourseRequest, userId string) (*models.PageQueryCourseResponse, error) {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	pageNum := int64(1)
	pageSize := int64(10)
	if request.PageNum != nil {
		pageNum = *request.PageNum
	}
	if request.PageSize != nil {
		pageSize = *request.PageSize
	}
	// 设置合理的限制
	if pageSize > 100 {
		pageSize = 100
	}

	filter := bson.D{bson.E{Key: "created_by", Value: userObjId}}
	// 只有当 name 不为 nil 且不为空时才添加 name 查询条件
	if request.Name != nil && *request.Name != "" {
		filter = append(filter, bson.E{"name", bson.D{
			{"$regex", *request.Name},
			{"$options", "i"},
		}})
	}
	// 只有当 status 不为 nil 时才添加 status 查询条件
	if request.Status != nil {
		filter = append(filter, bson.E{"status", *request.Status})
	}
	coll := utils.GetCollection("courses")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	findOptions := options.Find().SetSkip((pageNum - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{
			{"status", -1},
			{"ctime", -1},
		})
	cursor, err := coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []models.Course
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	res := &models.PageQueryCourseResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Courses:  results,
	}
	return res, nil
}

// PageQueryTeacherCourses 分页查询老师加入的课程
func (s *CourseService) PageQueryTeacherCourses(ctx context.Context, request *models.PageQueryTeacherCoursesRequest, userId string) (*models.PageQueryCourseResponse, error) {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	pageNum := int64(1)
	pageSize := int64(10)
	if request.PageNum != nil {
		pageNum = *request.PageNum
	}
	if request.PageSize != nil {
		pageSize = *request.PageSize
	}
	// 设置合理的限制
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询条件：创建的课程 OR 老师ID在teacher_ids数组中
	filter := bson.M{
		"$or": []bson.M{
			{"created_by": userObjId},
			{"teacher_ids": bson.M{"$in": []primitive.ObjectID{userObjId}}},
		},
	}

	// 只有当 name 不为 nil 且不为空时才添加 name 查询条件
	if request.Name != nil && *request.Name != "" {
		filter["name"] = bson.M{"$regex": *request.Name, "$options": "i"}
	}

	// 只有当 status 不为 nil 时才添加 status 查询条件
	if request.Status != nil {
		filter["status"] = *request.Status
	}

	coll := utils.GetCollection("courses")
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	findOptions := options.Find().SetSkip((pageNum - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{
			{"status", -1},
			{"ctime", -1},
		})
	cursor, err := coll.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []models.Course
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	res := &models.PageQueryCourseResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Courses:  results,
	}
	return res, nil
}

// UpdateCourseInfo 更新课程信息
func (s *CourseService) UpdateCourseInfo(ctx context.Context, userId string, courseId string, req *models.UpdateCourseRequest) error {
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return err
	}
	courseObjId, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return err
	}
	update := bson.M{}

	if req.Name != nil {
		update["name"] = *req.Name
	}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Status != nil {
		update["status"] = *req.Status
	}
	update["mtime"] = time.Now()

	filter := bson.M{
		"_id":        courseObjId,
		"created_by": userObjId, // 只允许创建者修改
	}

	res, err := utils.GetCollection("courses").UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return errors.New("课程不存在或权限不够")
	}
	return nil
}

// GetClazzByID 获取课程班级详情
func (s *CourseService) GetClazzByID(ctx context.Context, clazzID string, userID string) (*models.GetClazzResponse, error) {
	// 此方法已迁移到 clazz 服务中
	return nil, errors.New("此方法已迁移到 clazz 服务中")
}

func cheekIsMember(ids []primitive.ObjectID, userID primitive.ObjectID) bool {
	for _, memberID := range ids {
		if memberID == userID {
			return true
		}
	}
	return false
}

// 私有方法
// getCourseByID 根据ID获取课程
func (s *CourseService) getCourseByID(ctx context.Context, courseID primitive.ObjectID) (*models.Course, error) {
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
func (s *CourseService) getUserByID(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
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

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const length = 6

// generateInviteCode 生成邀请码
func (s *CourseService) generateInviteCode() string {

	// 使用加密安全的随机数生成器
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 回退到时间戳方案
		return fmt.Sprintf("%06d", time.Now().Unix()%1000000)
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}

	return string(bytes)
}

// cacheInviteCode 缓存邀请码
func (s *CourseService) cacheInviteCode(ctx context.Context, classID, inviteCode string, p string) {
	client := utils.GetRedisClient()
	if client == nil {
		return
	}

	key := fmt.Sprintf("invite_code:%s", classID)
	pipe := client.Pipeline()
	pipe.HSet(ctx, key, "code", inviteCode)
	pipe.HSet(ctx, key, "qrcode", p)
	pipe.Expire(ctx, key, 24*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("缓存邀请码失败: %v", err)
	}
}

// validateInviteCode 验证邀请码
func (s *CourseService) validateInviteCode(ctx context.Context, classID, inviteCode string) (bool, error) {
	client := utils.GetRedisClient()
	if client == nil {
		log.Printf("获取redis客户端失败")
		return false, errors.New("获取redis客户端失败")
	}

	key := fmt.Sprintf("invite_code:%s", classID)
	cachedCode, err := client.HGet(ctx, key, "code").Result()
	if err != nil {
		log.Printf("获取缓存邀请码失败: %v", err)
		return false, errors.New("获取缓存邀请码失败")
	}

	return cachedCode == inviteCode, nil
}

func contains(slice []primitive.ObjectID, v string) bool {
	for _, item := range slice {
		if item.Hex() == v {
			return true
		}
	}
	return false
}

func authorized(ctx context.Context, coll *mongo.Collection, courseId primitive.ObjectID, userId primitive.ObjectID) (bool, error) {
	filter := bson.M{"_id": courseId}
	var course models.Course
	projection := bson.M{
		"created_by":  1,
		"teacher_ids": 1,
	}
	if err := coll.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&course); err != nil {
		return false, errors.New("课程不存在")
	}
	isTeacher := contains(course.TeacherIds, userId.Hex())
	isCreator := course.CreatedBy == userId
	return isTeacher || isCreator, nil
}

// courseMemberAuthorized 检查用户是否是课程成员（创建者、教师或学生）
func courseMemberAuthorized(ctx context.Context, coll *mongo.Collection, courseId primitive.ObjectID, userId primitive.ObjectID) (bool, error) {
	// 先检查是否是课程创建者或教师
	isAuthorized, err := authorized(ctx, coll, courseId, userId)
	if err != nil || isAuthorized {
		return isAuthorized, err
	}

	// 如果不是创建者或教师，检查是否是学生
	// 查询该课程下的所有班级
	clazzColl := utils.GetCollection("clazzes")
	cursor, err := clazzColl.Find(ctx, bson.M{"course_id": courseId})
	if err != nil {
		return false, errors.New("查询班级失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var clazzes []models.Clazz
	if err = cursor.All(ctx, &clazzes); err != nil {
		return false, errors.New("解析班级数据失败: " + err.Error())
	}

	// 收集所有班级ID
	clazzIDs := make([]primitive.ObjectID, len(clazzes))
	for i, clazz := range clazzes {
		clazzIDs[i] = clazz.ID
	}

	// 查询学生是否在这些班级中
	studentClassColl := utils.GetCollection("student_classes")
	count, err := studentClassColl.CountDocuments(ctx, bson.M{
		"student_id": userId,
		"class_id":   bson.M{"$in": clazzIDs},
		"status":     "active",
	})
	if err != nil {
		return false, errors.New("查询学生班级关联失败: " + err.Error())
	}

	// 如果学生在至少一个班级中，则认为是课程成员
	return count > 0, nil
}

// UpdateClazzTeachers 更新班级的教师信息
func (s *CourseService) UpdateClazzTeachers(ctx context.Context, courseID primitive.ObjectID, teacherIds []primitive.ObjectID) error {
	// 更新所有属于该课程的班级的教师信息
	filter := bson.M{"course_id": courseID}
	update := bson.M{"$set": bson.M{"teacher_ids": teacherIds, "mtime": time.Now()}}

	_, err := utils.GetCollection("clazzes").UpdateMany(ctx, filter, update)
	if err != nil {
		log.Printf("更新班级教师信息失败: %v", err)
		return errors.New("更新班级教师信息失败")
	}

	return nil
}

// AddCourseTeacher 为课程添加教师
func (s *CourseService) AddCourseTeacher(ctx context.Context, courseId string, teacherId string, userId string) error {
	// 验证用户权限（只有课程创建者才能添加教师）
	courseObjID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return errors.New("无效的课程ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	teacherObjID, err := primitive.ObjectIDFromHex(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 检查课程是否存在并验证权限
	coll := utils.GetCollection("courses")
	var course models.Course
	if err = coll.FindOne(ctx, bson.M{"_id": courseObjID}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	// 只有课程创建者可以添加教师
	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以添加教师")
	}

	// 添加教师到课程
	filter := bson.M{"_id": courseObjID}
	update := bson.M{"$addToSet": bson.M{"teacher_ids": teacherObjID}, "$set": bson.M{"mtime": time.Now()}}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("添加教师失败: " + err.Error())
	}

	return nil
}

// AddCourseTeachers 为课程添加多个教师
func (s *CourseService) AddCourseTeachers(ctx context.Context, courseId string, teacherIds []string, userId string) error {
	// 验证用户权限（只有课程创建者才能添加教师）
	courseObjID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return errors.New("无效的课程ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 转换教师ID
	teacherObjIDs := make([]primitive.ObjectID, len(teacherIds))
	for i, id := range teacherIds {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New("无效的教师ID: " + id)
		}
		teacherObjIDs[i] = objID
	}

	// 检查课程是否存在并验证权限
	coll := utils.GetCollection("courses")
	var course models.Course
	if err = coll.FindOne(ctx, bson.M{"_id": courseObjID}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	// 只有课程创建者可以添加教师
	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以添加教师")
	}

	// 添加教师到课程
	filter := bson.M{"_id": courseObjID}
	update := bson.M{"$addToSet": bson.M{"teacher_ids": bson.M{"$each": teacherObjIDs}}, "$set": bson.M{"mtime": time.Now()}}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("添加教师失败: " + err.Error())
	}

	return nil
}

// RemoveCourseTeachers 从课程中删除多个教师
func (s *CourseService) RemoveCourseTeachers(ctx context.Context, courseId string, teacherIds []string, userId string) error {
	// 验证用户权限（只有课程创建者才能删除教师）
	courseObjID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return errors.New("无效的课程ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 转换教师ID
	teacherObjIDs := make([]primitive.ObjectID, len(teacherIds))
	for i, id := range teacherIds {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New("无效的教师ID: " + id)
		}
		teacherObjIDs[i] = objID
	}

	// 检查课程是否存在并验证权限
	coll := utils.GetCollection("courses")
	var course models.Course
	if err = coll.FindOne(ctx, bson.M{"_id": courseObjID}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	// 只有课程创建者可以删除教师
	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以删除教师")
	}

	// 检查要删除的教师中是否包含课程创建者，防止创建者删除自己
	for _, teacherObjID := range teacherObjIDs {
		if teacherObjID == course.CreatedBy {
			return errors.New("不能删除课程创建者作为课程教师")
		}
	}

	// 从课程中删除教师
	filter := bson.M{"_id": courseObjID}
	update := bson.M{"$pullAll": bson.M{"teacher_ids": teacherObjIDs}, "$set": bson.M{"mtime": time.Now()}}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("删除教师失败: " + err.Error())
	}

	// 同时从该课程的所有班级中删除这些教师
	clazzFilter := bson.M{"course_id": courseObjID}
	clazzUpdate := bson.M{"$pullAll": bson.M{"teacher_ids": teacherObjIDs}, "$set": bson.M{"mtime": time.Now()}}
	_, err = utils.GetCollection("clazzes").UpdateMany(ctx, clazzFilter, clazzUpdate)
	if err != nil {
		// 这里我们记录日志但不返回错误，因为主要目标是删除课程中的教师
		log.Printf("从班级中删除教师时出错: %v", err)
	}

	return nil
}

// AddClazzTeacher 为班级添加教师
func (s *CourseService) AddClazzTeacher(ctx context.Context, clazzId string, teacherId string, userId string) error {
	// 验证用户权限（只有课程创建者或课程教师才能添加班级教师）
	clazzObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	teacherObjID, err := primitive.ObjectIDFromHex(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 检查班级是否存在并验证权限
	coll := utils.GetCollection("clazzes")
	var clazz models.Clazz
	if err = coll.FindOne(ctx, bson.M{"_id": clazzObjID}).Decode(&clazz); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 验证操作者是否有权限添加教师
	courseColl := utils.GetCollection("courses")
	isAuthorized, err := authorized(ctx, courseColl, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或课程教师可以添加班级教师")
	}

	// 检查教师是否已经是班级的教师
	for _, id := range clazz.TeacherIds {
		if id == teacherObjID {
			return errors.New("该教师已经是班级的教师")
		}
	}

	// 添加教师到班级
	filter := bson.M{"_id": clazzObjID}
	update := bson.M{"$addToSet": bson.M{"teacher_ids": teacherObjID}, "$set": bson.M{"mtime": time.Now()}}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("添加班级教师失败: " + err.Error())
	}

	return nil
}

// AddClazzTeachers 为班级添加多个教师
func (s *CourseService) AddClazzTeachers(ctx context.Context, clazzId string, teacherIds []string, userId string) error {
	// 验证用户权限（只有课程创建者才能添加班级教师）
	clazzObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 转换教师ID
	teacherObjIDs := make([]primitive.ObjectID, len(teacherIds))
	for i, id := range teacherIds {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New("无效的教师ID: " + id)
		}
		teacherObjIDs[i] = objID
	}

	// 检查班级是否存在并验证权限
	clazzColl := utils.GetCollection("clazzes")
	var clazz models.Clazz
	if err = clazzColl.FindOne(ctx, bson.M{"_id": clazzObjID}).Decode(&clazz); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 只有课程创建者可以添加班级教师
	courseColl := utils.GetCollection("courses")
	var course models.Course
	if err = courseColl.FindOne(ctx, bson.M{"_id": clazz.CourseId}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以添加班级教师")
	}

	// 检查这些教师是否已经是课程的教师
	invalidTeachers := make([]string, 0)
	for _, teacherObjID := range teacherObjIDs {
		found := false
		for _, courseTeacherID := range course.TeacherIds {
			if courseTeacherID == teacherObjID {
				found = true
				break
			}
		}
		if !found {
			invalidTeachers = append(invalidTeachers, teacherObjID.Hex())
		}
	}

	if len(invalidTeachers) > 0 {
		return errors.New("以下教师尚未加入课程，无法添加到班级: " + strings.Join(invalidTeachers, ", "))
	}

	// 添加教师到班级
	filter := bson.M{"_id": clazzObjID}
	update := bson.M{"$addToSet": bson.M{"teacher_ids": bson.M{"$each": teacherObjIDs}}, "$set": bson.M{"mtime": time.Now()}}
	_, err = clazzColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("添加班级教师失败: " + err.Error())
	}

	return nil
}

// RemoveClazzTeacher 为班级移除教师
func (s *CourseService) RemoveClazzTeacher(ctx context.Context, clazzId string, teacherId string, userId string) error {
	// 验证用户权限（只有课程创建者或课程教师才能移除班级教师）
	clazzObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	teacherObjID, err := primitive.ObjectIDFromHex(teacherId)
	if err != nil {
		return errors.New("无效的教师ID")
	}

	// 检查班级是否存在并验证权限
	coll := utils.GetCollection("clazzes")
	var clazz models.Clazz
	if err = coll.FindOne(ctx, bson.M{"_id": clazzObjID}).Decode(&clazz); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 验证操作者是否有权限移除教师
	courseColl := utils.GetCollection("courses")
	isAuthorized, err := authorized(ctx, courseColl, clazz.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或课程教师可以移除班级教师")
	}

	// 检查教师是否是班级的教师
	found := false
	for _, id := range clazz.TeacherIds {
		if id == teacherObjID {
			found = true
			break
		}
	}

	if !found {
		return errors.New("该教师不是班级的教师")
	}

	// 不能移除最后一个教师
	if len(clazz.TeacherIds) <= 1 {
		return errors.New("不能移除最后一个教师")
	}

	// 从班级移除教师
	filter := bson.M{"_id": clazzObjID}
	update := bson.M{"$pull": bson.M{"teacher_ids": teacherObjID}, "$set": bson.M{"mtime": time.Now()}}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("移除班级教师失败: " + err.Error())
	}

	return nil
}

// RemoveClazzTeachers 为班级移除多个教师
func (s *CourseService) RemoveClazzTeachers(ctx context.Context, clazzId string, teacherIds []string, userId string) error {
	// 验证用户权限（只有课程创建者才能移除班级教师）
	clazzObjID, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return errors.New("无效的班级ID")
	}

	userObjID, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 转换教师ID
	teacherObjIDs := make([]primitive.ObjectID, len(teacherIds))
	for i, id := range teacherIds {
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return errors.New("无效的教师ID: " + id)
		}
		teacherObjIDs[i] = objID
	}

	// 检查班级是否存在并验证权限
	clazzColl := utils.GetCollection("clazzes")
	var clazz models.Clazz
	if err = clazzColl.FindOne(ctx, bson.M{"_id": clazzObjID}).Decode(&clazz); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("班级不存在")
		}
		return err
	}

	// 只有课程创建者可以移除班级教师
	courseColl := utils.GetCollection("courses")
	var course models.Course
	if err = courseColl.FindOne(ctx, bson.M{"_id": clazz.CourseId}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以移除班级教师")
	}

	// 检查是否尝试移除所有教师
	if len(clazz.TeacherIds) <= len(teacherObjIDs) {
		return errors.New("不能移除所有教师，至少需要保留一个教师")
	}

	// 从班级移除教师
	filter := bson.M{"_id": clazzObjID}
	update := bson.M{"$pullAll": bson.M{"teacher_ids": teacherObjIDs}, "$set": bson.M{"mtime": time.Now()}}
	_, err = clazzColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return errors.New("移除班级教师失败: " + err.Error())
	}

	return nil
}

// GetStudentClasses 获取学生的所有班级
func (s *CourseService) GetStudentClasses(ctx context.Context, studentID string) ([]*models.StudentClassResponse, error) {
	studentObjID, err := primitive.ObjectIDFromHex(studentID)
	if err != nil {
		return nil, errors.New("无效的学生ID")
	}

	// 查询学生的所有班级关联记录
	coll := utils.GetCollection("student_classes")
	cursor, err := coll.Find(ctx, bson.M{"student_id": studentObjID, "status": models.StudentClassStatusActive})
	if err != nil {
		return nil, errors.New("查询学生班级失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var studentClasses []models.StudentClass
	if err = cursor.All(ctx, &studentClasses); err != nil {
		return nil, errors.New("解析学生班级数据失败: " + err.Error())
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
		clazzColl := utils.GetCollection("clazzes")
		var clazz models.Clazz
		if err = clazzColl.FindOne(ctx, bson.M{"_id": sc.ClassID}).Decode(&clazz); err == nil {
			response.Class = &clazz
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
func (s *CourseService) GetClassStudents(ctx context.Context, classID string) ([]*models.UserProfile, error) {
	classObjID, err := primitive.ObjectIDFromHex(classID)
	if err != nil {
		return nil, errors.New("无效的班级ID")
	}

	// 查询班级的所有学生关联记录
	coll := utils.GetCollection("student_classes")
	cursor, err := coll.Find(ctx, bson.M{"class_id": classObjID, "status": models.StudentClassStatusActive})
	if err != nil {
		return nil, errors.New("查询班级学生失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var studentClasses []models.StudentClass
	if err = cursor.All(ctx, &studentClasses); err != nil {
		return nil, errors.New("解析班级学生数据失败: " + err.Error())
	}

	// 获取所有学生信息
	var students []*models.UserProfile
	for _, sc := range studentClasses {
		user, err := s.getUserByID(ctx, sc.StudentID)
		if err != nil {
			continue
		}
		students = append(students, user.ToProfile())
	}

	return students, nil
}

// GetCourseStudents 分页查询课程下的学生
func (s *CourseService) GetCourseStudents(ctx context.Context, userId string, courseId string, req *models.PageQueryCourseStudentsRequest) (*models.PageQueryCourseStudentsResponse, error) {
	// 验证课程ID格式
	courseObjID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}

	// 验证userID格式
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, errors.New("无效的userId")
	}

	// 验证权限：课程成员（创建者、教师、学生）都可以查询学生信息
	coll := utils.GetCollection("courses")
	isAuthorized, err := courseMemberAuthorized(ctx, coll, courseObjID, userObjId)
	if err != nil {
		return nil, err
	}

	if !isAuthorized {
		return nil, errors.New("权限不足，只有课程成员可以查询学生信息")
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
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询该课程下的所有班级
	clazzColl := utils.GetCollection("clazzes")
	cursor, err := clazzColl.Find(ctx, bson.M{"course_id": courseObjID})
	if err != nil {
		return nil, errors.New("查询班级失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var clazzes []models.Clazz
	if err = cursor.All(ctx, &clazzes); err != nil {
		return nil, errors.New("解析班级数据失败: " + err.Error())
	}

	// 收集所有班级ID
	clazzIDs := make([]primitive.ObjectID, len(clazzes))
	for i, clazz := range clazzes {
		clazzIDs[i] = clazz.ID
	}

	// 查询所有班级的学生关联记录
	studentClassColl := utils.GetCollection("student_classes")
	filter := bson.M{
		"class_id": bson.M{"$in": clazzIDs},
		"status":   "active",
	}
	cursor, err = studentClassColl.Find(ctx, filter)
	if err != nil {
		return nil, errors.New("查询学生班级关联失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var studentClasses []models.StudentClass
	if err = cursor.All(ctx, &studentClasses); err != nil {
		return nil, errors.New("解析学生班级关联数据失败: " + err.Error())
	}

	// 收集所有唯一的学生ID
	studentIDMap := make(map[primitive.ObjectID]bool)
	for _, sc := range studentClasses {
		studentIDMap[sc.StudentID] = true
	}

	// 转换为切片
	studentIDs := make([]primitive.ObjectID, 0, len(studentIDMap))
	for studentID := range studentIDMap {
		studentIDs = append(studentIDs, studentID)
	}

	// 构建学生查询条件
	userFilter := bson.M{"_id": bson.M{"$in": studentIDs}}

	// 如果指定了学生ID进行精确查询
	if req.StudentId != nil && *req.StudentId != "" {
		studentObjID, err := primitive.ObjectIDFromHex(*req.StudentId)
		if err != nil {
			return nil, errors.New("无效的学生ID")
		}
		userFilter = bson.M{"_id": studentObjID}
	}

	// 如果指定了学生姓名进行模糊查询
	if req.RealName != nil && *req.RealName != "" {
		userFilter["real_name"] = bson.M{"$regex": *req.RealName, "$options": "i"}
	}

	// 查询学生总数
	total, err := utils.GetCollection("users").CountDocuments(ctx, userFilter)
	if err != nil {
		return nil, errors.New("查询学生总数失败: " + err.Error())
	}

	// 分页查询学生信息
	findOptions := options.Find().
		SetSkip((pageNum - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{"created_at", -1}})

	cursor, err = utils.GetCollection("users").Find(ctx, userFilter, findOptions)
	if err != nil {
		return nil, errors.New("查询学生信息失败: " + err.Error())
	}
	defer cursor.Close(ctx)

	var students []*models.UserProfile
	if err = cursor.All(ctx, &students); err != nil {
		return nil, errors.New("解析学生信息失败: " + err.Error())
	}

	// 构造分页响应
	res := &models.PageQueryCourseStudentsResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Students: make([]models.UserProfile, len(students)),
	}

	// 转换学生信息
	for i, student := range students {
		res.Students[i] = *student
	}

	return res, nil
}

// GetCourseTeachers 分页查询课程下的教师
func (s *CourseService) GetCourseTeachers(ctx context.Context, userId string, courseId string, req *models.PageQueryCourseTeachersRequest) (*models.PageQueryCourseTeachersResponse, error) {
	// 验证课程ID格式
	courseObjID, err := primitive.ObjectIDFromHex(courseId)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}

	// 验证userID格式
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, errors.New("无效的userId")
	}

	// 验证权限：课程成员（创建者、教师、学生）都可以查询教师信息
	coll := utils.GetCollection("courses")
	isAuthorized, err := courseMemberAuthorized(ctx, coll, courseObjID, userObjId)
	if err != nil {
		return nil, err
	}

	if !isAuthorized {
		return nil, errors.New("权限不足，只有课程成员可以查询教师信息")
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
	if pageSize > 100 {
		pageSize = 100
	}

	// 查询课程信息以获取教师ID列表
	var course models.Course
	if err = coll.FindOne(ctx, bson.M{"_id": courseObjID}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("课程不存在")
		}
		return nil, err
	}

	// 构建教师查询条件
	teacherIDs := course.TeacherIds
	userFilter := bson.M{"_id": bson.M{"$in": teacherIDs}}

	// 如果指定了教师ID进行精确查询
	if req.TeacherId != nil && *req.TeacherId != "" {
		teacherObjID, err := primitive.ObjectIDFromHex(*req.TeacherId)
		if err != nil {
			return nil, errors.New("无效的教师ID")
		}
		userFilter = bson.M{"_id": teacherObjID}
	}

	// 如果指定了教师姓名进行模糊查询
	if req.RealName != nil && *req.RealName != "" {
		userFilter["real_name"] = bson.M{"$regex": *req.RealName, "$options": "i"}
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

	var teachers []*models.UserProfile
	if err = cursor.All(ctx, &teachers); err != nil {
		return nil, errors.New("解析教师信息失败: " + err.Error())
	}

	// 构造分页响应
	res := &models.PageQueryCourseTeachersResponse{
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
		Teachers: make([]models.UserProfile, len(teachers)),
	}

	// 转换教师信息
	for i, teacher := range teachers {
		res.Teachers[i] = *teacher
	}

	return res, nil
}

// DeleteTask 删除任务
func (s *CourseService) DeleteTask(ctx context.Context, userID string, taskID string) error {
	// 验证用户权限
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	taskObjID, err := primitive.ObjectIDFromHex(taskID)
	if err != nil {
		return errors.New("无效的任务ID")
	}

	// 获取任务信息
	collTasks := utils.GetCollection("tasks")
	var task models.Task
	if err = collTasks.FindOne(ctx, bson.M{"_id": taskObjID}).Decode(&task); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("任务不存在")
		}
		return err
	}

	// 验证权限：只有课程创建者或教师可以删除任务
	collCourse := utils.GetCollection("courses")
	isAuthorized, err := authorized(ctx, collCourse, task.CourseId, userObjID)
	if err != nil {
		return err
	}

	if !isAuthorized {
		return errors.New("权限不足，只有课程创建者或教师可以删除任务")
	}

	// 删除任务和班级的关联关系
	taskClazzColl := utils.GetCollection("task_clazz_relations")
	if _, err = taskClazzColl.DeleteMany(ctx, bson.M{"task_id": taskObjID}); err != nil {
		return errors.New("删除任务和班级关联关系失败: " + err.Error())
	}

	// 删除任务
	if _, err = collTasks.DeleteOne(ctx, bson.M{"_id": taskObjID}); err != nil {
		return errors.New("删除任务失败: " + err.Error())
	}

	return nil
}

// RemoveCourse 删除课程
func (s *CourseService) RemoveCourse(ctx context.Context, userID string, courseID string) error {
	// 验证用户权限
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return errors.New("无效的课程ID")
	}

	// 检查课程是否存在并验证权限
	coll := utils.GetCollection("courses")
	var course models.Course
	if err = coll.FindOne(ctx, bson.M{"_id": courseObjID}).Decode(&course); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("课程不存在")
		}
		return err
	}

	// 只有课程创建者可以删除课程
	if course.CreatedBy != userObjID {
		return errors.New("权限不足，只有课程创建者可以删除课程")
	}

	// 检查课程是否还有班级
	clazzColl := utils.GetCollection("clazzes")
	count, err := clazzColl.CountDocuments(ctx, bson.M{"course_id": courseObjID})
	if err != nil {
		return errors.New("检查班级失败: " + err.Error())
	}

	if count > 0 {
		return errors.New("课程还有班级，无法删除")
	}

	// 删除课程
	if _, err = coll.DeleteOne(ctx, bson.M{"_id": courseObjID}); err != nil {
		return errors.New("删除课程失败: " + err.Error())
	}

	return nil
}

func (s2 *CourseService) GetQrcode(userId string, clazzId string, ctx context.Context) (interface{}, error) {
	// 验证用户ID和班级ID格式
	userObjId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		return nil, errors.New("用户ID格式错误")
	}
	clazzObjId, err := primitive.ObjectIDFromHex(clazzId)
	if err != nil {
		return nil, errors.New("班级ID格式错误")
	}

	// 获取班级信息
	clazzColl := utils.GetCollection("clazzes")
	var clazz models.Clazz
	if err = clazzColl.FindOne(ctx, bson.M{"_id": clazzObjId}).Decode(&clazz); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("班级不存在")
		}
		return nil, err
	}

	// 验证用户权限：必须是班级成员、课程创建者或课程教师
	// 检查用户是否是班级成员（通过查询学生班级关联表）
	count, err := utils.GetCollection("student_classes").CountDocuments(ctx, bson.M{
		"student_id": userObjId,
		"class_id":   clazzObjId,
	})
	if err != nil {
		return nil, errors.New("检查班级成员失败: " + err.Error())
	}
	isMember := count > 0

	// 检查用户是否是课程创建者或课程教师
	collCourse := utils.GetCollection("courses")
	isAuthorized, err := authorized(ctx, collCourse, clazz.CourseId, userObjId)
	if err != nil {
		return nil, err
	}

	// 如果既不是成员也不是授权用户，则无权限
	if !isMember && !isAuthorized {
		return nil, models.ErrPermissionDenied
	}

	// 从Redis获取二维码
	qrcode, err := utils.RedisClient.HGet(ctx, "clazz_qrcode"+clazzId, "qrcode").Result()
	if err != nil {
		// Redis中没有找到二维码，说明二维码已过期
		return nil, models.ErrQRCodeExpired
	}

	return qrcode, nil
}
