/*
@Author: omenkk7
@Date: 2025/10/25
@Description: Repository基础层 - 通用数据访问功能
*/

package repository

import (
	"context"
	"fmt"
	"time"

	common "zk-code-arena-server/pkg/common/mongo"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// BaseRepository 基础Repository，提供通用的数据访问方法
type BaseRepository struct {
	db *mongo.Database
}

// NewBaseRepository 创建BaseRepository实例
func NewBaseRepository() *BaseRepository {
	return &BaseRepository{
		db: utils.MongoDB,
	}
}

// GetCollection 获取集合
func (r *BaseRepository) GetCollection(name string) *mongo.Collection {
	return r.db.Collection(name)
}

///////////////////////////////////////////////////////////////
//              更新构建器（自动过滤空值）
///////////////////////////////////////////////////////////////

// BuildUpdateSet 构建MongoDB更新文档 (基于BSON序列化)
func (r *BaseRepository) BuildUpdateSet(obj interface{}) (bson.M, error) {
	data, err := bson.Marshal(obj)
	if err != nil {
		utils.Logger.Errorf("BuildUpdateSet: BSON序列化失败, error=%v", err)
		return nil, fmt.Errorf("BSON序列化失败: %w", err)
	}

	var update bson.M
	if err := bson.Unmarshal(data, &update); err != nil {
		utils.Logger.Errorf("BuildUpdateSet: BSON反序列化失败, error=%v", err)
		return nil, fmt.Errorf("BSON反序列化失败: %w", err)
	}

	if len(update) == 0 {
		utils.Logger.Debugf("BuildUpdateSet: 没有字段需要更新")
		return nil, nil
	}

	return bson.M{"$set": update}, nil
}

// BuildUpdateSetWithTime 构建带 updated_at 的更新文档
func (r *BaseRepository) BuildUpdateSetWithTime(obj interface{}) (bson.M, error) {
	updateDoc, err := r.BuildUpdateSet(obj)
	if err != nil {
		return nil, err
	}

	if updateDoc == nil {
		return bson.M{
			"$set": bson.M{"updated_at": time.Now()},
		}, nil
	}

	updateDoc["$set"].(bson.M)["updated_at"] = time.Now()
	return updateDoc, nil
}

///////////////////////////////////////////////////////////////
//                      Insert 增强
///////////////////////////////////////////////////////////////

// InsertOne 通用插入单个文档
func (r *BaseRepository) InsertOne(ctx context.Context, collection string, document interface{}) (*mongo.InsertOneResult, error) {
	coll := r.db.Collection(collection)
	result, err := coll.InsertOne(ctx, document)

	if err != nil {
		utils.Logger.Errorf("InsertOne: 插入失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("插入失败: %w", err)
	}

	utils.Logger.Infof("InsertOne: 插入成功, collection=%s, id=%v", collection, result.InsertedID)
	return result, nil
}

// InsertMany 通用批量插入
func (r *BaseRepository) InsertMany(ctx context.Context, collection string, documents []interface{}) (*mongo.InsertManyResult, error) {
	coll := r.db.Collection(collection)
	result, err := coll.InsertMany(ctx, documents)

	if err != nil {
		utils.Logger.Errorf("InsertMany: 批量插入失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("批量插入失败: %w", err)
	}

	utils.Logger.Infof("InsertMany: 批量插入成功, collection=%s, count=%d", collection, len(result.InsertedIDs))
	return result, nil
}

// InsertOneWithTime 自动写入 created_at / updated_at
func (r *BaseRepository) InsertOneWithTime(ctx context.Context, collection string, document bson.M) (*mongo.InsertOneResult, error) {
	now := time.Now()
	document["created_at"] = now
	document["updated_at"] = now
	return r.InsertOne(ctx, collection, document)
}

///////////////////////////////////////////////////////////////
//                         Update
///////////////////////////////////////////////////////////////

// UpdateOne 通用更新方法（自动过滤空值）
func (r *BaseRepository) UpdateOne(ctx context.Context, collection string, filter bson.M, updateData interface{}) (*mongo.UpdateResult, error) {
	updateDoc, err := r.BuildUpdateSetWithTime(updateData)
	if err != nil {
		return nil, err
	}

	if updateDoc == nil {
		return nil, fmt.Errorf("没有字段需要更新")
	}

	coll := r.db.Collection(collection)
	result, err := coll.UpdateOne(ctx, filter, updateDoc)

	if err != nil {
		utils.Logger.Errorf("UpdateOne: 数据库更新失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("数据库更新失败: %w", err)
	}

	utils.Logger.Infof("UpdateOne: 更新成功, collection=%s, matched=%d, modified=%d",
		collection, result.MatchedCount, result.ModifiedCount)

	return result, nil
}

// UpdateMany 批量更新文档（自动过滤空值）
func (r *BaseRepository) UpdateMany(ctx context.Context, collection string, filter bson.M, updateData interface{}) (*mongo.UpdateResult, error) {
	updateDoc, err := r.BuildUpdateSetWithTime(updateData)
	if err != nil {
		return nil, err
	}

	if updateDoc == nil {
		return nil, fmt.Errorf("没有字段需要更新")
	}

	coll := r.db.Collection(collection)
	result, err := coll.UpdateMany(ctx, filter, updateDoc)

	if err != nil {
		utils.Logger.Errorf("UpdateMany: 批量更新失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("批量更新失败: %w", err)
	}

	utils.Logger.Infof("UpdateMany: 批量更新成功, collection=%s, matched=%d, modified=%d",
		collection, result.MatchedCount, result.ModifiedCount)

	return result, nil
}

///////////////////////////////////////////////////////////////
//                        Soft Delete
///////////////////////////////////////////////////////////////

// SoftDeleteOne 软删除（推荐增强功能）
func (r *BaseRepository) SoftDeleteOne(ctx context.Context, collection string, filter bson.M) (*mongo.UpdateResult, error) {
	update := bson.M{
		"$set": bson.M{
			"is_deleted": true,
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		},
	}

	coll := r.db.Collection(collection)
	result, err := coll.UpdateOne(ctx, filter, update)

	if err != nil {
		utils.Logger.Errorf("SoftDeleteOne: 删除失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("软删除失败: %w", err)
	}

	utils.Logger.Infof("SoftDeleteOne: 软删除成功, collection=%s, deleted=%d",
		collection, result.ModifiedCount)

	return result, nil
}

///////////////////////////////////////////////////////////////
//                       基础查询
///////////////////////////////////////////////////////////////

func (r *BaseRepository) FindOne(ctx context.Context, collection string, filter bson.M, result interface{}) error {
	coll := r.db.Collection(collection)
	err := coll.FindOne(ctx, filter).Decode(result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			utils.Logger.Debugf("FindOne: 文档不存在, collection=%s", collection)
			return fmt.Errorf("文档不存在")
		}
		utils.Logger.Errorf("FindOne: 查询失败, collection=%s, error=%v", collection, err)
		return fmt.Errorf("查询失败: %w", err)
	}

	return nil
}

func (r *BaseRepository) Find(ctx context.Context, collection string, filter bson.M, opts *options.FindOptions) (*mongo.Cursor, error) {
	coll := r.db.Collection(collection)
	cursor, err := coll.Find(ctx, filter, opts)

	if err != nil {
		utils.Logger.Errorf("Find: 查询失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("查询失败: %w", err)
	}

	return cursor, nil
}

func (r *BaseRepository) DeleteOne(ctx context.Context, collection string, filter bson.M) (*mongo.DeleteResult, error) {
	coll := r.db.Collection(collection)
	result, err := coll.DeleteOne(ctx, filter)

	if err != nil {
		utils.Logger.Errorf("DeleteOne: 删除失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("删除失败: %w", err)
	}

	utils.Logger.Infof("DeleteOne: 删除成功, collection=%s, deleted=%d", collection, result.DeletedCount)
	return result, nil
}

// DeleteMany 批量删除文档
func (r *BaseRepository) DeleteMany(ctx context.Context, collection string, filter bson.M) (*mongo.DeleteResult, error) {
	coll := r.db.Collection(collection)
	result, err := coll.DeleteMany(ctx, filter)

	if err != nil {
		utils.Logger.Errorf("DeleteMany: 批量删除失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("批量删除失败: %w", err)
	}

	utils.Logger.Infof("DeleteMany: 批量删除成功, collection=%s, deleted=%d", collection, result.DeletedCount)
	return result, nil
}

///////////////////////////////////////////////////////////////
//                        Count / Aggregate
///////////////////////////////////////////////////////////////

func (r *BaseRepository) CountDocuments(ctx context.Context, collection string, filter bson.M) (int64, error) {
	coll := r.db.Collection(collection)
	count, err := coll.CountDocuments(ctx, filter)

	if err != nil {
		utils.Logger.Errorf("CountDocuments: 计数失败, collection=%s, error=%v", collection, err)
		return 0, fmt.Errorf("计数失败: %w", err)
	}

	return count, nil
}

func (r *BaseRepository) Aggregate(ctx context.Context, collection string, pipeline []bson.M) (*mongo.Cursor, error) {
	coll := r.db.Collection(collection)
	cursor, err := coll.Aggregate(ctx, pipeline)

	if err != nil {
		utils.Logger.Errorf("Aggregate: 聚合查询失败, collection=%s, error=%v", collection, err)
		return nil, fmt.Errorf("聚合查询失败: %w", err)
	}

	return cursor, nil
}

///////////////////////////////////////////////////////////////
//              集成 pkg/common/mongo/query.go
///////////////////////////////////////////////////////////////

// FindWithPaginationTyped 泛型分页查询辅助函数
// 注意：Go 不支持泛型方法，只支持泛型函数，所以这是一个包级别的函数
func FindWithPaginationTyped[T any](
	ctx context.Context,
	repo *BaseRepository,
	collection string,
	filter interface{},
	opts *options.FindOptions,
) ([]T, int64, error) {
	coll := repo.GetCollection(collection)
	return common.FindWithPagination[T](ctx, coll, filter, opts)
}

// BuildSort 构建排序（直接调用 common.BuildSort）
func (r *BaseRepository) BuildSort(sortExpr string) bson.D {
	return common.BuildSort(sortExpr)
}

// BuildKeywordFilter 构建关键字过滤器（直接调用 common.BuildKeywordFilter）
func (r *BaseRepository) BuildKeywordFilter(keyword string, fields []string) bson.M {
	return common.BuildKeywordFilter(keyword, fields)
}

// MergeFilters 合并过滤器（直接调用 common.MergeFilters）
func (r *BaseRepository) MergeFilters(filters ...bson.M) bson.M {
	return common.MergeFilters(filters...)
}

// BuildFindOptionsWithPagination 构建查询选项（直接调用 common.BuildFindOptions）
func (r *BaseRepository) BuildFindOptionsWithPagination(
	page, pageSize int64,
	sort bson.D,
	projection bson.M,
) *options.FindOptions {
	return common.BuildFindOptions(page, pageSize, sort, projection)
}
