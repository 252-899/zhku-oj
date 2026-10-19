package repository

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ClazzRepository 班级数据访问层接口
type ClazzRepository interface {
	// ID转换相关
	ConvertToObjectID(id string) (primitive.ObjectID, error)
	ConvertToObjectIDs(ids []string) ([]primitive.ObjectID, error)

	// 权限验证
	CheckCourseAuthorization(ctx context.Context, courseID primitive.ObjectID, userID primitive.ObjectID) (bool, error)

	// 班级基础操作
	GetClazzByID(ctx context.Context, clazzID primitive.ObjectID) (*models.Clazz, error)
	CreateClazz(ctx context.Context, clazz *models.Clazz) (primitive.ObjectID, error)
	UpdateClazz(ctx context.Context, clazzID primitive.ObjectID, updateFields bson.M) (bool, error)
	DeleteClazz(ctx context.Context, clazzID primitive.ObjectID) (bool, error)
	GetClazzesByCourseID(ctx context.Context, courseID primitive.ObjectID) ([]models.Clazz, error)
	GetMajorClazzes(ctx context.Context) ([]models.Clazz, error) // 新增：查询所有专业班级

	// 班级成员操作
	IsClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error)
	AddClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error)
	RemoveClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error)
	BatchRemoveClazzMembers(ctx context.Context, clazzID primitive.ObjectID, memberIDs []primitive.ObjectID) (int, error)

	// 教师操作
	AddClazzTeacher(ctx context.Context, clazzID, teacherID primitive.ObjectID) (bool, error)
	RemoveClazzTeacher(ctx context.Context, clazzID, teacherID primitive.ObjectID) (bool, error)
	BatchAddClazzTeachers(ctx context.Context, clazzID primitive.ObjectID, teacherIDs []primitive.ObjectID) (int, error)
	GetTeachersByIDs(ctx context.Context, teacherIDs []primitive.ObjectID) ([]models.User, error)

	// 二维码相关
	GenerateAndSaveQRCode(ctx context.Context, clazzID string) (string, error)
	VerifyQRCode(ctx context.Context, clazzID string, inviteCode string) (bool, error)

	// 缓存操作
	SetCache(ctx context.Context, key string, value interface{}, expire time.Duration) error
	GetCache(ctx context.Context, key string) (string, error)
	DeleteCache(ctx context.Context, keys ...string) error

	// 学生班级关联操作
	CreateStudentClassRelation(ctx context.Context, relation *models.StudentClass) error
	UpdateStudentClassStatus(ctx context.Context, studentID, clazzID, courseID primitive.ObjectID, status models.StudentClassStatus) (bool, error)
	GetStudentClasses(ctx context.Context, studentID primitive.ObjectID) ([]models.StudentClass, error)
	// FindClazzesByMemberID 查询 member_ids 中包含该学生的所有班级（与 student_classes 对照，用于补全关联缺失的数据）
	FindClazzesByMemberID(ctx context.Context, studentID primitive.ObjectID) ([]models.Clazz, error)
	// FindStudentClassRelationsByClassIDs 批量查询学生与多个班级在 student_classes 中的关联（含 dropped）
	FindStudentClassRelationsByClassIDs(ctx context.Context, studentID primitive.ObjectID, classIDs []primitive.ObjectID) (map[primitive.ObjectID]models.StudentClass, error)
	GetClassStudents(ctx context.Context, clazzID primitive.ObjectID) ([]primitive.ObjectID, error)
	GetClassStudentsWithPagination(ctx context.Context, clazzID primitive.ObjectID, pageNum, pageSize int64) ([]models.StudentClass, int64, error)
	GetUsersByIDs(ctx context.Context, userIDs []primitive.ObjectID, realNameFilter string) ([]*models.User, error)

	// 任务相关
	GetTaskByID(ctx context.Context, taskID primitive.ObjectID) (*models.Task, error)
	GetTasksByClazzID(ctx context.Context, clazzID primitive.ObjectID) ([]models.Task, error)
	GetTasksByFilterWithPagination(ctx context.Context, filter bson.M, pageNum, pageSize int64) ([]models.Task, int64, error)
	UpdateTask(ctx context.Context, taskID primitive.ObjectID, updateFields bson.M) (bool, error)
	GetUserTaskStatus(ctx context.Context, taskID, userID primitive.ObjectID) (*models.UserTask, error)
	GetUserTaskStatusesByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.UserTask, error)
	UpdateUserTaskStatus(ctx context.Context, taskID, userID primitive.ObjectID, finishedCount int, isCompleted bool) error
	AddTaskRelationIds(ctx context.Context, taskID primitive.ObjectID, relationIDs []primitive.ObjectID, operatorID primitive.ObjectID) (bool, error)
	RemoveTaskRelationIds(ctx context.Context, taskID primitive.ObjectID, relationIDs []primitive.ObjectID, operatorID primitive.ObjectID) (bool, error)
	GetUserTaskStatuses(ctx context.Context, taskID primitive.ObjectID, userIDs []primitive.ObjectID) ([]models.UserTask, error)
	GetUserCompletedQuestions(ctx context.Context, taskID, userID primitive.ObjectID) (map[primitive.ObjectID]bool, error) // 新增
	GetProblemsByIDs(ctx context.Context, problemIDs []primitive.ObjectID) ([]models.Problem, error)                       // 新增

	// 专业班级关联操作
	UpdateMajorClassAssociations(ctx context.Context, clazzID primitive.ObjectID, majorClassIDs []primitive.ObjectID) error
}

// ClazzRepositoryImpl 班级数据访问层实现
type ClazzRepositoryImpl struct{}

// NewClazzRepository 创建班级Repository实例
func NewClazzRepository() ClazzRepository {
	return &ClazzRepositoryImpl{}
}

// ConvertToObjectID 转换单个ID为ObjectID
func (r *ClazzRepositoryImpl) ConvertToObjectID(id string) (primitive.ObjectID, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID, errors.New("无效的ID格式: " + id)
	}
	return objID, nil
}

// ConvertToObjectIDs 批量转换ID为ObjectID
func (r *ClazzRepositoryImpl) ConvertToObjectIDs(ids []string) ([]primitive.ObjectID, error) {
	objIDs := make([]primitive.ObjectID, 0, len(ids))
	for _, id := range ids {
		objID, err := r.ConvertToObjectID(id)
		if err != nil {
			return nil, err
		}
		objIDs = append(objIDs, objID)
	}
	return objIDs, nil
}

// CheckCourseAuthorization 检查用户是否有课程操作权限（创建者/教师）
func (r *ClazzRepositoryImpl) CheckCourseAuthorization(ctx context.Context, courseID primitive.ObjectID, userID primitive.ObjectID) (bool, error) {
	coll := utils.GetCollection("courses")
	filter := bson.M{"_id": courseID}
	projection := bson.M{"created_by": 1, "teacher_ids": 1}

	var course struct {
		CreatorID  primitive.ObjectID   `bson:"created_by"`
		TeacherIds []primitive.ObjectID `bson:"teacher_ids,omitempty"`
	}

	err := coll.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&course)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors.New("课程不存在")
		}
		return false, err
	}

	// 检查是否是创建者或教师
	isCreator := course.CreatorID == userID
	isTeacher := false
	for _, tid := range course.TeacherIds {
		if tid == userID {
			isTeacher = true
			break
		}
	}

	return isCreator || isTeacher, nil
}

// GetClazzByID 根据ID获取班级信息
func (r *ClazzRepositoryImpl) GetClazzByID(ctx context.Context, clazzID primitive.ObjectID) (*models.Clazz, error) {
	coll := utils.GetCollection("clazzes")
	var clazz models.Clazz
	err := coll.FindOne(ctx, bson.M{"_id": clazzID}).Decode(&clazz)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("班级不存在")
		}
		return nil, err
	}
	return &clazz, nil
}

// GetTeachersByIDs 根据教师ID列表获取教师详情
func (r *ClazzRepositoryImpl) GetTeachersByIDs(ctx context.Context, teacherIDs []primitive.ObjectID) ([]models.User, error) {
	if len(teacherIDs) == 0 {
		return []models.User{}, nil
	}

	userColl := utils.GetCollection("users")
	teacherFilter := bson.M{
		"_id": bson.M{"$in": teacherIDs},
	}
	teacherCursor, err := userColl.Find(ctx, teacherFilter)
	if err != nil {
		return nil, errors.New("查询教师信息失败: " + err.Error())
	}
	defer teacherCursor.Close(ctx)

	var teachers []models.User
	if err = teacherCursor.All(ctx, &teachers); err != nil {
		return nil, errors.New("解析教师信息失败: " + err.Error())
	}

	return teachers, nil
}

// CreateClazz 创建班级
func (r *ClazzRepositoryImpl) CreateClazz(ctx context.Context, clazz *models.Clazz) (primitive.ObjectID, error) {
	coll := utils.GetCollection("clazzes")
	result, err := coll.InsertOne(ctx, clazz)
	if err != nil {
		return primitive.NilObjectID, errors.New("创建班级失败: " + err.Error())
	}
	return result.InsertedID.(primitive.ObjectID), nil
}

// UpdateClazz 更新班级信息
func (r *ClazzRepositoryImpl) UpdateClazz(ctx context.Context, clazzID primitive.ObjectID, updateFields bson.M) (bool, error) {
	coll := utils.GetCollection("clazzes")
	filter := bson.M{"_id": clazzID}
	update := bson.M{"$set": updateFields}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

// DeleteClazz 删除课程班级
func (r *ClazzRepositoryImpl) DeleteClazz(ctx context.Context, clazzID primitive.ObjectID) (bool, error) {
	coll := utils.GetCollection("clazzes")
	result, err := coll.DeleteOne(ctx, bson.M{"_id": clazzID})
	if err != nil {
		return false, err
	}
	return result.DeletedCount > 0, nil
}

// GetClazzesByCourseID 根据课程ID获取所有班级
func (r *ClazzRepositoryImpl) GetClazzesByCourseID(ctx context.Context, courseID primitive.ObjectID) ([]models.Clazz, error) {
	coll := utils.GetCollection("clazzes")
	cursor, err := coll.Find(ctx, bson.M{"course_id": courseID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var clazzes []models.Clazz
	if err = cursor.All(ctx, &clazzes); err != nil {
		return nil, err
	}
	return clazzes, nil
}

// IsClazzMember 检查用户是否是班级成员
func (r *ClazzRepositoryImpl) IsClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error) {
	coll := utils.GetCollection("student_classes")
	count, err := coll.CountDocuments(ctx, bson.M{
		"student_id": userID,
		"class_id":   clazzID,
		"status":     models.StudentClassStatusActive,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AddClazzMember 添加课程班级成员（乐观锁）
func (r *ClazzRepositoryImpl) AddClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error) {
	// 先检查是否已存在
	isMember, err := r.IsClazzMember(ctx, clazzID, userID)
	if err != nil {
		return false, err
	}
	if isMember {
		return false, errors.New("用户已是班级成员")
	}

	// 乐观锁：以 member_ids 人数为准（学生名额），与 add_nums 脱钩避免历史脏数据
	coll := utils.GetCollection("clazzes")
	seat := ClazzStudentSeatAvailableExpr()
	filter := bson.M{
		"_id":   clazzID,
		"$expr": seat["$expr"],
	}
	update := bson.M{
		"$inc":      bson.M{"add_nums": 1},
		"$addToSet": bson.M{"member_ids": userID},
		"$set":      bson.M{"mtime": time.Now()},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	if result.MatchedCount == 0 {
		return false, errors.New("班级已满")
	}

	// 创建学生班级关联
	return true, nil
}

// RemoveClazzMember 移除课程班级成员
func (r *ClazzRepositoryImpl) RemoveClazzMember(ctx context.Context, clazzID, userID primitive.ObjectID) (bool, error) {
	// 检查是否是成员
	isMember, err := r.IsClazzMember(ctx, clazzID, userID)
	if err != nil {
		return false, err
	}
	if !isMember {
		return false, errors.New("用户不是班级成员")
	}

	// 更新成员数和member_ids
	coll := utils.GetCollection("clazzes")
	update := bson.M{
		"$inc":  bson.M{"add_nums": -1},
		"$pull": bson.M{"member_ids": userID},
		"$set":  bson.M{"mtime": time.Now()},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": clazzID}, update)
	if err != nil {
		return false, err
	}

	// 更新关联表状态
	_, err = r.UpdateStudentClassStatus(ctx, userID, clazzID, primitive.NilObjectID, models.StudentClassStatusDropped)
	return err == nil, err
}

// BatchRemoveClazzMembers 批量移除课程班级成员
func (r *ClazzRepositoryImpl) BatchRemoveClazzMembers(ctx context.Context, clazzID primitive.ObjectID, memberIDs []primitive.ObjectID) (int, error) {
	// 筛选有效成员
	validMembers := make([]primitive.ObjectID, 0)
	for _, mid := range memberIDs {
		isMember, err := r.IsClazzMember(ctx, clazzID, mid)
		if err != nil {
			return 0, err
		}
		if isMember {
			validMembers = append(validMembers, mid)
		}
	}

	if len(validMembers) == 0 {
		return 0, nil
	}

	// 批量更新成员数和member_ids
	coll := utils.GetCollection("clazzes")
	update := bson.M{
		"$inc":  bson.M{"add_nums": -len(validMembers)},
		"$pull": bson.M{"member_ids": bson.M{"$in": validMembers}},
		"$set":  bson.M{"mtime": time.Now()},
	}
	_, err := coll.UpdateOne(ctx, bson.M{"_id": clazzID}, update)
	if err != nil {
		return 0, err
	}

	// 批量更新关联表
	collStudentClass := utils.GetCollection("student_classes")
	_, err = collStudentClass.UpdateMany(ctx,
		bson.M{
			"class_id":   clazzID,
			"student_id": bson.M{"$in": validMembers},
		},
		bson.M{
			"$set": bson.M{
				"status": models.StudentClassStatusDropped,
				"mtime":  time.Now(),
			},
		},
	)

	return len(validMembers), err
}

// AddClazzTeacher 添加班级教师
func (r *ClazzRepositoryImpl) AddClazzTeacher(ctx context.Context, clazzID, teacherID primitive.ObjectID) (bool, error) {
	// 检查是否已存在
	clazz, err := r.GetClazzByID(ctx, clazzID)
	if err != nil {
		return false, err
	}

	for _, tid := range clazz.TeacherIds {
		if tid == teacherID {
			return false, errors.New("教师已存在")
		}
	}

	// 添加教师
	coll := utils.GetCollection("clazzes")
	update := bson.M{
		"$addToSet": bson.M{"teacher_ids": teacherID},
		"$set":      bson.M{"mtime": time.Now()},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": clazzID}, update)
	return err == nil, err
}

// RemoveClazzTeacher 移除班级教师
func (r *ClazzRepositoryImpl) RemoveClazzTeacher(ctx context.Context, clazzID, teacherID primitive.ObjectID) (bool, error) {
	// 检查是否存在且不是最后一个
	clazz, err := r.GetClazzByID(ctx, clazzID)
	if err != nil {
		return false, err
	}

	// 检查是否存在
	exists := false
	for _, tid := range clazz.TeacherIds {
		if tid == teacherID {
			exists = true
			break
		}
	}
	if !exists {
		return false, errors.New("教师不存在")
	}

	// 检查是否是最后一个
	if len(clazz.TeacherIds) <= 1 {
		return false, errors.New("不能移除最后一个教师")
	}

	// 移除教师
	coll := utils.GetCollection("clazzes")
	update := bson.M{
		"$pull": bson.M{"teacher_ids": teacherID},
		"$set":  bson.M{"mtime": time.Now()},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": clazzID}, update)
	if err != nil {
		return false, err
	}

	// 清除缓存
	r.DeleteCache(ctx, fmt.Sprintf("clazz_detail:%s:*", clazzID.Hex()))
	return true, nil
}

// BatchAddClazzTeachers 批量添加班级教师
func (r *ClazzRepositoryImpl) BatchAddClazzTeachers(ctx context.Context, clazzID primitive.ObjectID, teacherIDs []primitive.ObjectID) (int, error) {
	// 获取现有教师
	clazz, err := r.GetClazzByID(ctx, clazzID)
	if err != nil {
		return 0, err
	}

	// 过滤已存在的教师
	existing := make(map[primitive.ObjectID]bool)
	for _, tid := range clazz.TeacherIds {
		existing[tid] = true
	}

	newTeachers := make([]primitive.ObjectID, 0)
	for _, tid := range teacherIDs {
		if !existing[tid] {
			newTeachers = append(newTeachers, tid)
		}
	}

	if len(newTeachers) == 0 {
		return 0, nil
	}

	// 批量添加
	coll := utils.GetCollection("clazzes")
	update := bson.M{
		"$addToSet": bson.M{"teacher_ids": bson.M{"$each": newTeachers}},
		"$set":      bson.M{"mtime": time.Now()},
	}
	_, err = coll.UpdateOne(ctx, bson.M{"_id": clazzID}, update)
	if err != nil {
		return 0, err
	}

	// 清除缓存
	r.DeleteCache(ctx, fmt.Sprintf("clazz_detail:%s:*", clazzID.Hex()))
	return len(newTeachers), nil
}

// GenerateAndSaveQRCode 生成并保存二维码到Redis
func (r *ClazzRepositoryImpl) GenerateAndSaveQRCode(ctx context.Context, clazzID string) (string, error) {
	// 生成随机数+班级ID
	ran := fmt.Sprintf("%d,%s", rand.Int(), clazzID)
	// 生成二维码
	encode, err := qrcode.Encode(ran, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	// Base64编码
	base64QRCode := base64.StdEncoding.EncodeToString(encode)

	// 保存到Redis
	key := "clazz_qrcode" + clazzID
	err = utils.RedisClient.HSet(ctx, key, "qrcode", base64QRCode, "ran", ran).Err()
	if err != nil {
		return "", err
	}
	// 设置过期时间
	utils.RedisClient.Expire(ctx, key, 30*time.Minute)

	return base64QRCode, nil
}

// VerifyQRCode 验证二维码邀请码
func (r *ClazzRepositoryImpl) VerifyQRCode(ctx context.Context, clazzID string, inviteCode string) (bool, error) {
	key := "clazz_qrcode" + clazzID
	storedRan, err := utils.RedisClient.HGet(ctx, key, "ran").Result()
	if err != nil {
		return false, errors.New("二维码已过期或不存在")
	}

	// 解析存储的ran值
	parts := strings.Split(storedRan, ",")
	if len(parts) != 2 {
		return false, errors.New("二维码格式错误")
	}

	// 验证随机数部分
	return parts[0] == inviteCode, nil
}

// SetCache 设置缓存
func (r *ClazzRepositoryImpl) SetCache(ctx context.Context, key string, value interface{}, expire time.Duration) error {
	return utils.RedisClient.Set(ctx, key, value, expire).Err()
}

// GetCache 获取缓存
func (r *ClazzRepositoryImpl) GetCache(ctx context.Context, key string) (string, error) {
	return utils.RedisClient.Get(ctx, key).Result()
}

// DeleteCache 删除缓存
func (r *ClazzRepositoryImpl) DeleteCache(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return utils.RedisClient.Del(ctx, keys...).Err()
}

// CreateStudentClassRelation 创建学生班级关联
func (r *ClazzRepositoryImpl) CreateStudentClassRelation(ctx context.Context, relation *models.StudentClass) error {
	coll := utils.GetCollection("student_classes")
	_, err := coll.InsertOne(ctx, relation)
	if err != nil {
		return errors.New("创建学生班级关联失败: " + err.Error())
	}
	return nil
}

// UpdateStudentClassStatus 更新学生班级关联状态
func (r *ClazzRepositoryImpl) UpdateStudentClassStatus(ctx context.Context, studentID, clazzID, courseID primitive.ObjectID, status models.StudentClassStatus) (bool, error) {
	filter := bson.M{
		"student_id": studentID,
		"class_id":   clazzID,
		"status":     models.StudentClassStatusActive,
	}
	if courseID != primitive.NilObjectID {
		filter["course_id"] = courseID
	}

	coll := utils.GetCollection("student_classes")
	result, err := coll.UpdateOne(ctx, filter, bson.M{
		"$set": bson.M{
			"status": status,
			"mtime":  time.Now(),
		},
	})
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

// GetStudentClasses 获取学生的所有班级
func (r *ClazzRepositoryImpl) GetStudentClasses(ctx context.Context, studentID primitive.ObjectID) ([]models.StudentClass, error) {
	coll := utils.GetCollection("student_classes")
	cursor, err := coll.Find(ctx, bson.M{
		"student_id": studentID,
		"status":     models.StudentClassStatusActive,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relations []models.StudentClass
	if err = cursor.All(ctx, &relations); err != nil {
		return nil, err
	}
	return relations, nil
}

// FindClazzesByMemberID 查询 member_ids 中包含该学生的所有班级
func (r *ClazzRepositoryImpl) FindClazzesByMemberID(ctx context.Context, studentID primitive.ObjectID) ([]models.Clazz, error) {
	coll := utils.GetCollection("clazzes")
	cursor, err := coll.Find(ctx, bson.M{"member_ids": studentID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []models.Clazz
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// FindStudentClassRelationsByClassIDs 批量查询学生在多个班级下的 student_classes 记录
func (r *ClazzRepositoryImpl) FindStudentClassRelationsByClassIDs(ctx context.Context, studentID primitive.ObjectID, classIDs []primitive.ObjectID) (map[primitive.ObjectID]models.StudentClass, error) {
	out := make(map[primitive.ObjectID]models.StudentClass)
	if len(classIDs) == 0 {
		return out, nil
	}
	coll := utils.GetCollection("student_classes")
	cursor, err := coll.Find(ctx, bson.M{
		"student_id": studentID,
		"class_id":   bson.M{"$in": classIDs},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var list []models.StudentClass
	if err = cursor.All(ctx, &list); err != nil {
		return nil, err
	}
	for _, sc := range list {
		out[sc.ClassID] = sc
	}
	return out, nil
}

// GetClassStudents 获取班级的所有学生ID
func (r *ClazzRepositoryImpl) GetClassStudents(ctx context.Context, clazzID primitive.ObjectID) ([]primitive.ObjectID, error) {
	coll := utils.GetCollection("student_classes")
	cursor, err := coll.Find(ctx, bson.M{
		"class_id": clazzID,
		"status":   models.StudentClassStatusActive,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relations []models.StudentClass
	if err = cursor.All(ctx, &relations); err != nil {
		return nil, err
	}

	studentIDs := make([]primitive.ObjectID, 0, len(relations))
	for _, r := range relations {
		studentIDs = append(studentIDs, r.StudentID)
	}
	return studentIDs, nil
}

// GetTaskByID 根据ID获取任务
func (r *ClazzRepositoryImpl) GetTaskByID(ctx context.Context, taskID primitive.ObjectID) (*models.Task, error) {
	coll := utils.GetCollection("tasks")
	var task models.Task
	err := coll.FindOne(ctx, bson.M{"_id": taskID}).Decode(&task)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("任务不存在")
		}
		return nil, err
	}
	return &task, nil
}

// GetTasksByClazzID 根据班级ID获取任务列表
func (r *ClazzRepositoryImpl) GetTasksByClazzID(ctx context.Context, clazzID primitive.ObjectID) ([]models.Task, error) {
	coll := utils.GetCollection("tasks")
	cursor, err := coll.Find(ctx, bson.M{"clazz_id": clazzID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tasks []models.Task
	if err = cursor.All(ctx, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateTask 更新任务信息
func (r *ClazzRepositoryImpl) UpdateTask(ctx context.Context, taskID primitive.ObjectID, updateFields bson.M) (bool, error) {
	coll := utils.GetCollection("tasks")
	result, err := coll.UpdateOne(ctx, bson.M{"_id": taskID}, bson.M{"$set": updateFields})
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

// AddTaskRelationIds 为任务添加关系ID
func (r *ClazzRepositoryImpl) AddTaskRelationIds(ctx context.Context, taskID primitive.ObjectID, relationIDs []primitive.ObjectID, operatorID primitive.ObjectID) (bool, error) {
	coll := utils.GetCollection("tasks")
	filter := bson.M{"_id": taskID}
	update := bson.M{
		"$addToSet": bson.M{
			"relation_ids": bson.M{"$each": relationIDs},
		},
		"$set": bson.M{
			"c_id":  operatorID,
			"mtime": time.Now(),
		},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

// RemoveTaskRelationIds 从任务中删除关系ID
func (r *ClazzRepositoryImpl) RemoveTaskRelationIds(ctx context.Context, taskID primitive.ObjectID, relationIDs []primitive.ObjectID, operatorID primitive.ObjectID) (bool, error) {
	coll := utils.GetCollection("tasks")
	filter := bson.M{"_id": taskID}
	update := bson.M{
		"$pullAll": bson.M{
			"relation_ids": relationIDs,
		},
		"$set": bson.M{
			"c_id":  operatorID,
			"mtime": time.Now(),
		},
	}

	result, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, err
	}
	return result.MatchedCount > 0, nil
}

// GetUserTaskStatus 获取用户任务完成状态
func (r *ClazzRepositoryImpl) GetUserTaskStatus(ctx context.Context, taskID, userID primitive.ObjectID) (*models.UserTask, error) {
	coll := utils.GetCollection("user_task")
	var userTask models.UserTask
	err := coll.FindOne(ctx, bson.M{
		"task_id": taskID,
		"user_id": userID,
	}).Decode(&userTask)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil // 无记录表示未开始
		}
		return nil, err
	}
	return &userTask, nil
}

// UpdateUserTaskStatus 更新用户任务完成状态
func (r *ClazzRepositoryImpl) UpdateUserTaskStatus(ctx context.Context, taskID, userID primitive.ObjectID, finishedCount int, isCompleted bool) error {
	coll := utils.GetCollection("user_task")
	now := time.Now()

	// 检查是否已有记录
	userTask, err := r.GetUserTaskStatus(ctx, taskID, userID)
	if err != nil {
		return err
	}

	if userTask != nil {
		// 更新现有记录
		updateFields := bson.M{
			"finished_count": finishedCount,
			"mtime":          now,
		}
		if isCompleted {
			updateFields["state"] = 1
			updateFields["completed_at"] = now
		}

		_, err = coll.UpdateOne(ctx, bson.M{"_id": userTask.ID}, bson.M{"$set": updateFields})
	} else {
		// 创建新记录
		newTask := &models.UserTask{
			ID:            primitive.NewObjectID(),
			TaskID:        taskID,
			UserID:        userID,
			State:         0,
			FinishedCount: finishedCount,
			CTime:         now,
			MTime:         now,
		}
		if isCompleted {
			newTask.State = 1
			completedAt := now
			newTask.CompletedAt = &completedAt
		}
		_, err = coll.InsertOne(ctx, newTask)
	}

	return err
}

// GetClassStudentsWithPagination 分页查询班级学生关联记录
func (r *ClazzRepositoryImpl) GetClassStudentsWithPagination(ctx context.Context, clazzID primitive.ObjectID, pageNum, pageSize int64) ([]models.StudentClass, int64, error) {
	studentClassColl := utils.GetCollection("student_classes")
	studentClassFilter := bson.M{
		"class_id": clazzID,
		"status":   "active",
	}

	// 查询班级学生总数
	total, err := studentClassColl.CountDocuments(ctx, studentClassFilter)
	if err != nil {
		return nil, 0, err
	}

	// 分页查询班级学生关联记录
	findOptions := options.Find().
		SetSkip((pageNum - 1) * pageSize).
		SetLimit(pageSize).
		SetSort(bson.D{{"join_time", -1}})

	cursor, err := studentClassColl.Find(ctx, studentClassFilter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var studentClasses []models.StudentClass
	if err = cursor.All(ctx, &studentClasses); err != nil {
		return nil, 0, err
	}

	return studentClasses, total, nil
}

// GetUsersByIDs 根据ID列表获取用户信息
func (r *ClazzRepositoryImpl) GetUsersByIDs(ctx context.Context, userIDs []primitive.ObjectID, realNameFilter string) ([]*models.User, error) {
	userFilter := bson.M{"_id": bson.M{"$in": userIDs}}

	// 如果指定了学生姓名进行模糊查询
	if realNameFilter != "" {
		userFilter["real_name"] = bson.M{"$regex": realNameFilter, "$options": "i"}
	}

	// 查询学生信息
	userColl := utils.GetCollection("users")
	userCursor, err := userColl.Find(ctx, userFilter)
	if err != nil {
		return nil, err
	}
	defer userCursor.Close(ctx)

	var students []*models.User
	if err = userCursor.All(ctx, &students); err != nil {
		return nil, err
	}

	return students, nil
}

// GetUserTaskStatuses 批量获取用户任务完成状态
func (r *ClazzRepositoryImpl) GetUserTaskStatuses(ctx context.Context, taskID primitive.ObjectID, userIDs []primitive.ObjectID) ([]models.UserTask, error) {
	userTaskColl := utils.GetCollection("user_task")
	userTaskFilter := bson.M{
		"task_id": taskID,
		"user_id": bson.M{"$in": userIDs},
	}
	userTaskCursor, err := userTaskColl.Find(ctx, userTaskFilter)
	if err != nil {
		return nil, err
	}
	defer userTaskCursor.Close(ctx)

	var userTasks []models.UserTask
	if err = userTaskCursor.All(ctx, &userTasks); err != nil {
		return nil, err
	}

	return userTasks, nil
}

// GetTasksByFilterWithPagination 根据条件分页查询任务列表
func (r *ClazzRepositoryImpl) GetTasksByFilterWithPagination(ctx context.Context, filter bson.M, pageNum, pageSize int64) ([]models.Task, int64, error) {
	collTasks := utils.GetCollection("tasks")

	// 计算总数
	total, err := collTasks.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	skip := (pageNum - 1) * pageSize

	// 查询任务列表
	taskCursor, err := collTasks.Find(ctx, filter, options.Find().SetSkip(skip).SetLimit(pageSize))
	if err != nil {
		return nil, 0, err
	}
	defer taskCursor.Close(ctx)

	var tasks []models.Task
	if err = taskCursor.All(ctx, &tasks); err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetUserTaskStatusesByUserID 根据用户ID获取所有任务状态
func (r *ClazzRepositoryImpl) GetUserTaskStatusesByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.UserTask, error) {
	userTaskColl := utils.GetCollection("user_task")
	userTaskFilter := bson.M{"user_id": userID}

	userTaskCursor, err := userTaskColl.Find(ctx, userTaskFilter)
	if err != nil {
		return nil, err
	}
	defer userTaskCursor.Close(ctx)

	var userTasks []models.UserTask
	if err = userTaskCursor.All(ctx, &userTasks); err != nil {
		return nil, err
	}

	return userTasks, nil
}

// GetUserCompletedQuestions 获取用户已完成的题目
func (r *ClazzRepositoryImpl) GetUserCompletedQuestions(ctx context.Context, taskID, userID primitive.ObjectID) (map[primitive.ObjectID]bool, error) {
	userRelationColl := utils.GetCollection("user_relation_question")
	userRelationCursor, err := userRelationColl.Find(ctx, bson.M{
		"task_id": taskID,
		"user_id": userID,
		"state":   1, // 已完成的题目
	})
	if err != nil {
		return nil, err
	}
	defer userRelationCursor.Close(ctx)

	var userRelations []models.UserRelationQuestion
	if err = userRelationCursor.All(ctx, &userRelations); err != nil {
		return nil, err
	}

	completedQuestions := make(map[primitive.ObjectID]bool)
	for _, ur := range userRelations {
		completedQuestions[ur.RelationID] = true
	}

	return completedQuestions, nil
}

// GetProblemsByIDs 根据ID列表获取题目详情
func (r *ClazzRepositoryImpl) GetProblemsByIDs(ctx context.Context, problemIDs []primitive.ObjectID) ([]models.Problem, error) {
	if len(problemIDs) == 0 {
		return []models.Problem{}, nil
	}

	problemColl := utils.GetCollection("problems")
	problemCursor, err := problemColl.Find(ctx, bson.M{
		"_id": bson.M{"$in": problemIDs},
	})
	if err != nil {
		return nil, err
	}
	defer problemCursor.Close(ctx)

	var problems []models.Problem
	if err = problemCursor.All(ctx, &problems); err != nil {
		return nil, err
	}

	return problems, nil
}

// UpdateMajorClassAssociations 更新课程班级的专业班级关联
func (r *ClazzRepositoryImpl) UpdateMajorClassAssociations(ctx context.Context, clazzID primitive.ObjectID, majorClassIDs []primitive.ObjectID) error {
	coll := utils.GetCollection("clazzes")

	// 更新班级的major_class_ids字段
	_, err := coll.UpdateOne(ctx,
		bson.M{"_id": clazzID},
		bson.M{"$set": bson.M{"major_class_ids": majorClassIDs, "mtime": time.Now()}},
	)

	return err
}

// GetMajorClazzes 获取所有专业班级
func (r *ClazzRepositoryImpl) GetMajorClazzes(ctx context.Context) ([]models.Clazz, error) {
	coll := utils.GetCollection("clazzes")
	cursor, err := coll.Find(ctx, bson.M{"class_type": models.ClassTypeMajor})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var clazzes []models.Clazz
	if err = cursor.All(ctx, &clazzes); err != nil {
		return nil, err
	}
	return clazzes, nil
}
