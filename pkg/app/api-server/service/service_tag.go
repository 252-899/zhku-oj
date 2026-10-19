package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InitTagsFromProblems 初始化标签
func (s *Service) InitTagsFromProblems(ctx context.Context) error {
	problemCol := utils.GetCollection("problems")
	tagCol := utils.GetCollection("tags")

	// 获取当前登录用户ID
	userIDVal := ctx.Value("user_id")
	if userIDVal == nil {
		utils.Logger.Errorf("InitTagsFromProblems：未获取到当前登录用户ID")
		return fmt.Errorf("未获取到当前登录用户ID")
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		utils.Logger.Errorf("InitTagsFromProblems：用户ID格式错误")
		return fmt.Errorf("用户ID格式错误")
	}
	createBy, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		utils.Logger.Errorf("InitTagsFromProblems: 转换用户ID失败, userID=%s, error=%v", userIDStr, err)
		return fmt.Errorf("无效的用户ID：%w", err)
	}

	// 预设标签
	presetTags := []struct {
		Name string
		Desc string
	}{
		{"测试1", "测试标签1的默认描述"},
		{"数组", "数组的遍历、排序、双指针技巧"},
		{"哈希表", "哈希表的增删改查、哈希冲突解决"},
		{"链表", "链表的反转、环检测、增删改查"},
		{"Java", "Java面向对象"},
	}

	// 插入预设标签
	for _, pt := range presetTags {
		count, err := tagCol.CountDocuments(ctx, bson.M{"name": pt.Name})
		if err != nil {
			utils.Logger.Warnf("InitTagsFromProblems : 检查预设标签失败, name=%s, error=%v", pt.Name, err)
			continue
		}
		if count == 0 {
			tag := &models.Tag{
				ID:        primitive.NewObjectID(),
				Name:      pt.Name,
				Desc:      pt.Desc,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: createBy,
			}
			_, err := tagCol.InsertOne(ctx, tag)
			if err != nil {
				utils.Logger.Warnf("InitTagsFromProblems：插入预设标签失败，name=%s, error=%v", pt.Name, err)
			} else {
				utils.Logger.Infof("InitTagsFromProblems：成功插入预设标签：%s", pt.Name)
			}
		}
	}

	// 查询所有题目中的唯一标签
	pipeline := []bson.M{
		{"$unwind": "$tags"},
		{"$group": bson.M{"_id": "$tags"}},
		{"$project": bson.M{"name": "$_id", "_id": 0}},
	}
	cursor, err := problemCol.Aggregate(ctx, pipeline)
	if err != nil {
		utils.Logger.Errorf("InitTagsFromProblems: 聚合查询标签失败, error=%v", err)
		return fmt.Errorf("聚合查询标签失败: %w", err)
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			utils.Logger.Errorf("InitTagsFromProblems: 关闭游标失败, error=%v", closeErr)
		}
	}()

	// 检查游标基础错误
	if err := cursor.Err(); err != nil {
		utils.Logger.Errorf("InitTagsFromProblems: 游标遍历失败, error=%v", err)
		return fmt.Errorf("游标遍历失败: %w", err)
	}

	// 解析标签列表
	var tagNames []string
	for cursor.Next(ctx) {
		var result struct{ Name string }
		if err := cursor.Decode(&result); err != nil {
			utils.Logger.Warnf("InitTagsFromProblems: 解析标签失败, error=%v", err)
			continue
		}
		if result.Name != "" {
			tagNames = append(tagNames, result.Name)
		}
	}
	if len(tagNames) == 0 {
		utils.Logger.Info("InitTagsFromProblems: 未找到任何标签")
		return nil
	}

	// 批量插入标签（忽略已经存在的）
	var tagsToInsert []interface{}
	for _, name := range tagNames {
		// 检查标签是否已存在
		count, err := tagCol.CountDocuments(ctx, bson.M{"name": name})
		if err != nil {
			utils.Logger.Errorf("InitTagsFromProblems: 检查标签是否存在失败, name=%s, error=%v", name, err)
			continue
		}
		if count == 0 {
			tags := &models.Tag{
				ID:        primitive.NewObjectID(),
				Name:      name,
				Desc:      "", // 默认空描述
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				CreatedBy: createBy,
			}
			tagsToInsert = append(tagsToInsert, tags)
		}
	}

	// 执行批量插入
	if len(tagsToInsert) > 0 {
		_, err := tagCol.InsertMany(ctx, tagsToInsert)
		if err != nil {
			// 忽略重复键错误
			if mongo.IsDuplicateKeyError(err) {
				utils.Logger.Warnf("InitTagsFromProblems: 部分标签已存在, error=%v", err)
			} else {
				utils.Logger.Errorf("InitTagsFromProblems: 插入标签失败, error=%v", err)
				return fmt.Errorf("插入标签失败: %w", err)
			}
		}
		utils.Logger.Infof("InitTagsFromProblems: 成功初始化%d个标签", len(tagsToInsert))
	} else {
		utils.Logger.Info("InitTagsFromProblems: 所有标签已存在，无需初始化")
	}

	return nil
}

// GetTagList 分页查询标签列表
func (s *Service) GetTagList(ctx context.Context, req *models.TagQueryRequest) ([]*models.Tag, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
		utils.Logger.Debugf("GetTagList: 页码无效，设置默认值 1")
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
		utils.Logger.Debugf("GetTagList: 每页条数无效，设置默认值 10")
	}

	tagCol := utils.GetCollection("tags")
	filter := bson.M{}
	if req.Keyword != "" {
		filter["name"] = bson.M{"$regex": req.Keyword, "$options": "i"}
	}

	total, err := tagCol.CountDocuments(ctx, filter)
	if err != nil {
		utils.Logger.Errorf("GetTagList: 统计标签总数失败, error=%v", err)
		return nil, 0, fmt.Errorf("统计标签总数失败: %w", err)
	}
	utils.Logger.Debugf("GetTagList: 标签总条数统计完成, total=%d", total)

	opts := options.Find().
		SetSkip(int64((req.Page - 1) * req.PageSize)).
		SetLimit(int64(req.PageSize)).
		SetSort(bson.M{"created_at": -1})

	cursor, err := tagCol.Find(ctx, filter, opts)
	if err != nil {
		utils.Logger.Errorf("GetTagList: 查询标签列表失败, error=%v", err)
		return nil, 0, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			utils.Logger.Errorf("GetTagList: 关闭游标失败, error=%v", err)
		}
	}()

	var tags []*models.Tag
	for cursor.Next(ctx) {
		var t models.Tag
		if err := cursor.Decode(&t); err != nil {
			utils.Logger.Errorf("GetTagList: 解析标签数据失败, error=%v", err)
			return nil, 0, fmt.Errorf("解析标签失败: %w", err)
		}
		tags = append(tags, &t)
	}

	utils.Logger.Infof("GetTagList: 标签列表查询完成, 返回条数=%d", len(tags))
	return tags, total, cursor.Err()
}

// GetTagByID 获取标签详情
func (s *Service) GetTagByID(ctx context.Context, tagID primitive.ObjectID) (*models.Tag, error) {
	tagCol := utils.GetCollection("tags")

	var tag models.Tag
	err := tagCol.FindOne(ctx, bson.M{"_id": tagID}).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Warnf("GetTagByID: 标签不存在, tagID=%s", tagID.Hex())
			return nil, fmt.Errorf("标签ID「%s」不存在", tagID.Hex())
		}
		utils.Logger.Errorf("GetTagByID: 查询标签失败, tagID=%s, error=%v", tagID.Hex(), err)
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}

	utils.Logger.Infof("GetTagByID: 标签查询成功, tagID=%s, tagName=%s", tagID.Hex(), tag.Name)
	return &tag, nil
}

// CreateTag 创建标签
func (s *Service) CreateTag(ctx context.Context, tag *models.Tag) error {
	tag.ID = primitive.NewObjectID()
	tag.CreatedAt = time.Now()
	tag.UpdatedAt = time.Now()

	tagCol := utils.GetCollection("tags")

	// 校验标签名唯一
	var existing models.Tag
	err := tagCol.FindOne(ctx, bson.M{"name": tag.Name}).Decode(&existing)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Errorf("CreateTag: 校验标签名唯一性失败, name=%s, error=%v", tag.Name, err)
			return fmt.Errorf("校验标签名失败: %w", err)
		}
	} else {
		utils.Logger.Warnf("CreateTag: 标签名称已存在, name=%s", tag.Name)
		return fmt.Errorf("标签名称「%s」已存在", tag.Name)
	}

	_, err = tagCol.InsertOne(ctx, tag)
	if err != nil {
		utils.Logger.Errorf("CreateTag: 插入标签数据失败, name=%s, id=%s, error=%v", tag.Name, tag.ID.Hex(), err)
		return fmt.Errorf("插入标签失败: %w", err)
	}
	utils.Logger.Infof("CreateTag: 标签创建成功, name=%s, id=%s", tag.Name, tag.ID.Hex())
	return nil
}

// UpdateTag 更新标签
func (s *Service) UpdateTag(ctx context.Context, tagID primitive.ObjectID, req *models.UpdateTagRequest) error {
	tagCol := utils.GetCollection("tags")
	problemCol := utils.GetCollection("problems")

	// 校验标签存在
	var tag models.Tag
	err := tagCol.FindOne(ctx, bson.M{"_id": tagID}).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Warnf("UpdateTag: 标签不存在, tagID=%s", tagID.Hex())
			return fmt.Errorf("标签ID「%s」不存在", tagID.Hex())
		}
		utils.Logger.Errorf("UpdateTag: 查询标签失败, tagID=%s, error=%v", tagID.Hex(), err)
		return fmt.Errorf("查询标签失败: %w", err)
	}
	oldName := tag.Name
	newName := req.Name

	// 校验新名称唯一
	if req.Name != "" && req.Name != oldName {
		var existing models.Tag
		err = tagCol.FindOne(ctx, bson.M{
			"_id":  bson.M{"$ne": tagID},
			"name": req.Name,
		}).Decode(&existing)
		if err != nil {
			if !errors.Is(err, mongo.ErrNoDocuments) {
				utils.Logger.Errorf("UpdateTag: 校验新标签名唯一性失败, newName=%s, tagID=%s, error=%v", req.Name, tagID.Hex(), err)
				return fmt.Errorf("校验新标签名失败: %w", err)
			}
		} else {
			utils.Logger.Warnf("UpdateTag: 新标签名已存在, newName=%s", req.Name)
			return fmt.Errorf("标签名称「%s」已存在", req.Name)
		}
	}

	update := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}
	hasUpdate := false
	if req.Name != "" && req.Name != oldName {
		update["$set"].(bson.M)["name"] = req.Name
		hasUpdate = true

		// 查询包含旧标签的题目
		var problems []struct {
			ID   primitive.ObjectID `bson:"_id"`
			Tags []string           `bson:"tags"`
		}
		cursor, err := problemCol.Find(ctx, bson.M{"tags": oldName})
		if err != nil {
			utils.Logger.Errorf("UpdateTag: 查询包含旧标签的题目失败, oldName=%s, error=%v", oldName, err)
			return fmt.Errorf("查询关联题目失败: %w", err)
		}
		defer func() {
			if closeErr := cursor.Close(ctx); closeErr != nil {
				utils.Logger.Errorf("UpdateTag: 关闭题目游标失败, error=%v", closeErr)
			}
		}()

		// 解析题目列表
		if err := cursor.All(ctx, &problems); err != nil {
			utils.Logger.Errorf("UpdateTag: 解析题目列表失败, error=%v", err)
			return fmt.Errorf("解析关联题目失败: %w", err)
		}
		utils.Logger.Infof("UpdateTag: 找到包含旧标签[%s]的题目共%d道", oldName, len(problems))

		// 遍历更新每道题的标签
		updatedCount := 0
		for _, p := range problems {
			// 替换数组中的旧标签名
			newTags := make([]string, len(p.Tags))
			for i, tagName := range p.Tags {
				if tagName == oldName {
					newTags[i] = newName
				} else {
					newTags[i] = tagName
				}
			}
			// 更新当前题目
			_, err := problemCol.UpdateByID(ctx, p.ID, bson.M{
				"$set": bson.M{
					"tags":       newTags,
					"updated_at": time.Now(),
				},
			})
			if err != nil {
				utils.Logger.Warnf("UpdateTag: 更新题目[%s]标签失败, error=%v", p.ID.Hex(), err)
				continue // 单个题目更新失败不中断整体流程
			}
			updatedCount++
		}
		utils.Logger.Infof("UpdateTag：同步更新题目标签名：%s → %s", oldName, newName)
	}
	if req.Desc != tag.Desc {
		update["$set"].(bson.M)["desc"] = req.Desc
		hasUpdate = true
	}
	if !hasUpdate {
		utils.Logger.Warnf("UpdateTag: 标签无更新内容，tagID=%s", tagID.Hex())
		return fmt.Errorf("无有效更新内容，标签名称和描述均与原内容一致")
	}

	//更新标签本身
	result, err := tagCol.UpdateByID(ctx, tagID, update)
	if err != nil {
		utils.Logger.Errorf("UpdateTag: 更新标签失败, tagID=%s, error=%v", tagID.Hex(), err)
		return fmt.Errorf("更新标签失败: %w", err)
	}
	if result.ModifiedCount == 0 {
		utils.Logger.Warnf("UpdateTag: 标签无更新内容, tagID=%s", tagID.Hex())
		return fmt.Errorf("标签无更新内容")
	}
	utils.Logger.Infof("UpdateTag: 标签更新成功, tagID=%s", tagID.Hex())
	return nil
}

// DeleteTag 删除标签
func (s *Service) DeleteTag(ctx context.Context, tagID primitive.ObjectID) error {
	tagCol := utils.GetCollection("tags")
	problemCol := utils.GetCollection("problems")

	// 先获取标签详情
	var tag models.Tag
	err := tagCol.FindOne(ctx, bson.M{"_id": tagID}).Decode(&tag)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.Logger.Warnf("DeleteTag: 标签不存在, tagID=%s", tagID.Hex())
			return fmt.Errorf("标签ID「%s」不存在", tagID.Hex())
		}
		utils.Logger.Errorf("DeleteTag: 查询标签失败, tagID=%s, error=%v", tagID.Hex(), err)
		return fmt.Errorf("查询标签失败: %w", err)
	}

	// 清理题目中的该标签
	_, err = problemCol.UpdateMany(ctx,
		bson.M{"tags": tag.Name},
		bson.M{"$pull": bson.M{"tags": tag.Name}},
	)
	if err != nil {
		utils.Logger.Errorf("DeleteTag: 清理题目关联标签失败, tagName=%s, error=%v", tag.Name, err)
		return fmt.Errorf("清理题目关联标签失败: %w", err)
	}

	// 删除标签
	result, err := tagCol.DeleteOne(ctx, bson.M{"_id": tagID})
	if err != nil {
		utils.Logger.Errorf("DeleteTag: 删除标签失败, tagID=%s, error=%v", tagID.Hex(), err)
		return fmt.Errorf("删除标签失败: %w", err)
	}
	if result.DeletedCount == 0 {
		utils.Logger.Warnf("DeleteTag: 标签删除失败, tagID=%s", tagID.Hex())
		return fmt.Errorf("标签删除失败")
	}
	utils.Logger.Infof("DeleteTag: 标签删除成功, tagID=%s, tagName=%s", tagID.Hex(), tag.Name)
	return nil
}

// GetPublicTagList 获取所有标签
func (s *Service) GetPublicTagList(ctx context.Context) ([]*models.Tag, error) {
	tagCol := utils.GetCollection("tags")
	cursor, err := tagCol.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		utils.Logger.Errorf("GetPublicTagList: 查询标签列表失败, error=%v", err)
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			utils.Logger.Errorf("GetPublicTagList: 关闭游标失败, error=%v", err)
		}
	}()

	var tags []*models.Tag
	for cursor.Next(ctx) {
		var t models.Tag
		if err := cursor.Decode(&t); err != nil {
			utils.Logger.Errorf("GetPublicTagList: 解析标签失败, error=%v", err)
			return nil, fmt.Errorf("解析标签失败: %w", err)
		}
		tags = append(tags, &t)
	}
	utils.Logger.Infof("GetPublicTagList: 标签查询完成, 返回条数=%d", len(tags))
	return tags, cursor.Err()
}
