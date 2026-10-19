/*
@Author: xiaoTnb
@Date: 2026/05/12
@Description: RSS 订阅源与条目的 MongoDB 持久化及索引维护。
*/

package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	collRSSSources   = "rss_sources"
	collRSSItems     = "rss_items"
	collRSSReadState = "rss_read_states"
	collRSSItemReads = "rss_item_reads"
	collRSSUserEvents = "rss_user_events"
)

// RSSRepository RSS 数据访问层。
type RSSRepository struct {
	*BaseRepository
	idxOnce sync.Once
	idxErr  error
}

// NewRSSRepository 构造 RSSRepository，并在首次使用时创建索引。
func NewRSSRepository() *RSSRepository {
	return &RSSRepository{BaseRepository: NewBaseRepository()}
}

func ensureRSSIndexes(ctx context.Context, db *mongo.Database) error {
	src := db.Collection(collRSSSources)
	if _, err := src.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "feed_url", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_sources index: %w", err)
	}

	items := db.Collection(collRSSItems)
	if _, err := items.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "source_id", Value: 1},
			{Key: "guid", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_items index: %w", err)
	}
	if _, err := items.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "published_at", Value: -1}, {Key: "_id", Value: -1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_items sort index: %w", err)
	}
	if _, err := items.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "fetched_at", Value: -1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_items fetched_at index: %w", err)
	}

	readState := db.Collection(collRSSReadState)
	if _, err := readState.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_read_states index: %w", err)
	}

	itemReads := db.Collection(collRSSItemReads)
	if _, err := itemReads.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "item_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_item_reads index: %w", err)
	}
	if _, err := itemReads.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "read_at", Value: -1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_item_reads user index: %w", err)
	}

	events := db.Collection(collRSSUserEvents)
	if _, err := events.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_user_events index: %w", err)
	}
	if _, err := events.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "item_id", Value: 1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_user_events item index: %w", err)
	}
	if _, err := items.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "open_count", Value: -1}},
	}); err != nil && !strings.Contains(err.Error(), "IndexOptionsConflict") {
		return fmt.Errorf("rss_items open_count index: %w", err)
	}
	return nil
}

func (r *RSSRepository) ensure(ctx context.Context) error {
	r.idxOnce.Do(func() {
		db := r.GetCollection(collRSSSources).Database()
		r.idxErr = ensureRSSIndexes(ctx, db)
	})
	return r.idxErr
}

// InsertRSSSource 插入订阅源。
func (r *RSSRepository) InsertRSSSource(ctx context.Context, doc *models.RSSSource) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	doc.ID = primitive.NewObjectID()
	now := time.Now()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	_, err := r.GetCollection(collRSSSources).InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("该订阅地址已存在")
		}
		return fmt.Errorf("插入订阅源失败: %w", err)
	}
	return nil
}

// UpdateRSSSource 按 ID 更新订阅源。
func (r *RSSRepository) UpdateRSSSource(ctx context.Context, id primitive.ObjectID, patch bson.M) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	patch["updated_at"] = time.Now()
	_, err := r.GetCollection(collRSSSources).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": patch})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("该订阅地址已被其他源使用")
		}
		return fmt.Errorf("更新订阅源失败: %w", err)
	}
	return nil
}

// DeleteRSSSource 删除订阅源（条目保留或一并删除：此处仅删源，条目可按需孤儿化；产品上一并删更干净）。
func (r *RSSRepository) DeleteRSSSource(ctx context.Context, id primitive.ObjectID) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	if _, err := r.GetCollection(collRSSItems).DeleteMany(ctx, bson.M{"source_id": id}); err != nil {
		return fmt.Errorf("删除订阅条目失败: %w", err)
	}
	if _, err := r.GetCollection(collRSSSources).DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		return fmt.Errorf("删除订阅源失败: %w", err)
	}
	return nil
}

// GetRSSSourceByID 根据 ID 查询订阅源。
func (r *RSSRepository) GetRSSSourceByID(ctx context.Context, id primitive.ObjectID) (*models.RSSSource, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	var s models.RSSSource
	err := r.GetCollection(collRSSSources).FindOne(ctx, bson.M{"_id": id}).Decode(&s)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("订阅源不存在")
		}
		return nil, err
	}
	return &s, nil
}

// ListRSSSources 列出全部订阅源（管理端，数据量小不分页）。
func (r *RSSRepository) ListRSSSources(ctx context.Context) ([]models.RSSSource, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	cur, err := r.GetCollection(collRSSSources).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.RSSSource
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// ListEnabledRSSSources 列出已启用的订阅源供定时拉取。
func (r *RSSRepository) ListEnabledRSSSources(ctx context.Context) ([]models.RSSSource, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	cur, err := r.GetCollection(collRSSSources).Find(ctx, bson.M{"enabled": true})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.RSSSource
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateRSSSourceFetchMeta 更新拉取元数据（ETag、错误信息、时间戳）。
func (r *RSSRepository) UpdateRSSSourceFetchMeta(ctx context.Context, id primitive.ObjectID, etag, lastModified string, fetchAt time.Time, success bool, errMsg string) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	set := bson.M{
		"etag":          etag,
		"last_modified": lastModified,
		"last_fetch_at": fetchAt,
		"updated_at":    time.Now(),
	}
	if success {
		set["last_success_at"] = fetchAt
		set["last_error"] = ""
	} else {
		set["last_error"] = errMsg
	}
	_, err := r.GetCollection(collRSSSources).UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": set})
	return err
}

// InsertRSSItemsIgnoreDuplicate 批量插入条目，忽略重复 guid，返回成功入库的完整文档。
func (r *RSSRepository) InsertRSSItemsIgnoreDuplicate(ctx context.Context, items []models.RSSItem) ([]models.RSSItem, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	coll := r.GetCollection(collRSSItems)
	inserted := make([]models.RSSItem, 0, len(items))
	for i := range items {
		items[i].ID = primitive.NewObjectID()
		res, err := coll.InsertOne(ctx, &items[i])
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			utils.Logger.Warnf("RSS 插入条目失败: %v", err)
			continue
		}
		if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
			items[i].ID = oid
			inserted = append(inserted, items[i])
		}
	}
	return inserted, nil
}

// ListRSSItems 分页列出条目（按发布时间倒序）。
func (r *RSSRepository) ListRSSItems(ctx context.Context, sourceID *primitive.ObjectID, page, pageSize int64) ([]models.RSSItem, int64, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, 0, err
	}
	filter := bson.M{}
	if sourceID != nil {
		filter["source_id"] = *sourceID
	}
	coll := r.GetCollection(collRSSItems)
	total, err := coll.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	skip := (page - 1) * pageSize
	cur, err := coll.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "published_at", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(skip).
		SetLimit(pageSize))
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var list []models.RSSItem
	if err := cur.All(ctx, &list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetRSSItemByID 根据 ID 获取条目。
func (r *RSSRepository) GetRSSItemByID(ctx context.Context, id primitive.ObjectID) (*models.RSSItem, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	var it models.RSSItem
	err := r.GetCollection(collRSSItems).FindOne(ctx, bson.M{"_id": id}).Decode(&it)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("资讯不存在")
		}
		return nil, err
	}
	return &it, nil
}

// ListRSSItemsAfterFetchedAt 查询在指定时间之后入库的条目（用于新条轮询）。
func (r *RSSRepository) ListRSSItemsAfterFetchedAt(ctx context.Context, after time.Time, limit int64) ([]models.RSSItem, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	cur, err := r.GetCollection(collRSSItems).Find(ctx, bson.M{"fetched_at": bson.M{"$gt": after}},
		options.Find().SetSort(bson.D{{Key: "fetched_at", Value: 1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.RSSItem
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// GetRSSReadState 获取用户已读游标；不存在时返回零值状态（非错误）。
func (r *RSSRepository) GetRSSReadState(ctx context.Context, userID string) (*models.RSSReadState, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	var st models.RSSReadState
	err := r.GetCollection(collRSSReadState).FindOne(ctx, bson.M{"user_id": userID}).Decode(&st)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &models.RSSReadState{UserID: userID}, nil
		}
		return nil, err
	}
	return &st, nil
}

// UpsertRSSLastReadAt 更新用户全局已读游标（全部已读）。
func (r *RSSRepository) UpsertRSSLastReadAt(ctx context.Context, userID string, at time.Time) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	now := time.Now()
	_, err := r.GetCollection(collRSSReadState).UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{
			"$set": bson.M{
				"user_id":      userID,
				"last_read_at": at,
				"updated_at":   now,
			},
			"$setOnInsert": bson.M{"_id": primitive.NewObjectID()},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// UpsertRSSItemRead 记录单条资讯已读（幂等）。
func (r *RSSRepository) UpsertRSSItemRead(ctx context.Context, userID string, itemID primitive.ObjectID, readAt time.Time) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	_, err := r.GetCollection(collRSSItemReads).UpdateOne(ctx,
		bson.M{"user_id": userID, "item_id": itemID},
		bson.M{
			"$set": bson.M{
				"user_id": userID,
				"item_id": itemID,
				"read_at": readAt,
			},
			"$setOnInsert": bson.M{"_id": primitive.NewObjectID()},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

// ListRSSItemReadIDs 返回用户已单独标记已读的条目 ID 列表。
func (r *RSSRepository) ListRSSItemReadIDs(ctx context.Context, userID string) ([]primitive.ObjectID, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	cur, err := r.GetCollection(collRSSItemReads).Find(ctx, bson.M{"user_id": userID},
		options.Find().SetProjection(bson.M{"item_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []struct {
		ItemID primitive.ObjectID `bson:"item_id"`
	}
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	ids := make([]primitive.ObjectID, 0, len(docs))
	for i := range docs {
		ids = append(ids, docs[i].ItemID)
	}
	return ids, nil
}

// CountRSSUnreadItems 统计未读条数：晚于 last_read_at 且未在单条已读表中的条目。
func (r *RSSRepository) CountRSSUnreadItems(ctx context.Context, userID string, lastReadAt time.Time, readItemIDs []primitive.ObjectID) (int64, error) {
	if err := r.ensure(ctx); err != nil {
		return 0, err
	}
	filter := bson.M{"_id": bson.M{"$nin": readItemIDs}}
	if !lastReadAt.IsZero() {
		filter["fetched_at"] = bson.M{"$gt": lastReadAt}
	}
	return r.GetCollection(collRSSItems).CountDocuments(ctx, filter)
}

// InsertRSSUserEvent 写入用户行为事件。
func (r *RSSRepository) InsertRSSUserEvent(ctx context.Context, ev *models.RSSUserEvent) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	ev.ID = primitive.NewObjectID()
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now()
	}
	_, err := r.GetCollection(collRSSUserEvents).InsertOne(ctx, ev)
	return err
}

// ListRSSUserEvents 按时间倒序列出用户近期行为。
func (r *RSSRepository) ListRSSUserEvents(ctx context.Context, userID string, limit int) ([]models.RSSUserEvent, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	cur, err := r.GetCollection(collRSSUserEvents).Find(ctx, bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.RSSUserEvent
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// IncrementRSSItemStat 增加条目打开/点击计数。
func (r *RSSRepository) IncrementRSSItemStat(ctx context.Context, itemID primitive.ObjectID, openDelta, clickDelta int) error {
	if err := r.ensure(ctx); err != nil {
		return err
	}
	inc := bson.M{}
	if openDelta != 0 {
		inc["open_count"] = openDelta
	}
	if clickDelta != 0 {
		inc["click_count"] = clickDelta
	}
	if len(inc) == 0 {
		return nil
	}
	_, err := r.GetCollection(collRSSItems).UpdateOne(ctx, bson.M{"_id": itemID}, bson.M{"$inc": inc})
	return err
}

// ListRSSItemsForRecommend 拉取推荐候选池（按发布时间倒序）。
func (r *RSSRepository) ListRSSItemsForRecommend(ctx context.Context, limit int64) ([]models.RSSItem, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 1000 {
		limit = 400
	}
	cur, err := r.GetCollection(collRSSItems).Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "published_at", Value: -1}}).SetLimit(limit))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var list []models.RSSItem
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// AggregateUserProblemTags 从用户近期提交的题目聚合标签（画像补充）。
func (r *RSSRepository) AggregateUserProblemTags(ctx context.Context, userID primitive.ObjectID, maxTags int) ([]string, error) {
	if err := r.ensure(ctx); err != nil {
		return nil, err
	}
	submitColl := r.GetCollection("submits")
	cur, err := submitColl.Find(ctx, bson.M{"user_id": userID},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(80).
			SetProjection(bson.M{"problem_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var subs []struct {
		ProblemID primitive.ObjectID `bson:"problem_id"`
	}
	if err := cur.All(ctx, &subs); err != nil {
		return nil, err
	}
	if len(subs) == 0 {
		return nil, nil
	}
	seenPID := make(map[primitive.ObjectID]struct{})
	ids := make([]primitive.ObjectID, 0, len(subs))
	for i := range subs {
		if _, ok := seenPID[subs[i].ProblemID]; ok {
			continue
		}
		seenPID[subs[i].ProblemID] = struct{}{}
		ids = append(ids, subs[i].ProblemID)
	}
	probCur, err := r.GetCollection("problems").Find(ctx, bson.M{"_id": bson.M{"$in": ids}},
		options.Find().SetProjection(bson.M{"tags": 1}))
	if err != nil {
		return nil, err
	}
	defer probCur.Close(ctx)
	var probs []struct {
		Tags []string `bson:"tags"`
	}
	if err := probCur.All(ctx, &probs); err != nil {
		return nil, err
	}
	freq := make(map[string]int)
	for i := range probs {
		for _, t := range probs[i].Tags {
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			freq[t]++
		}
	}
	type kv struct {
		k string
		v int
	}
	arr := make([]kv, 0, len(freq))
	for k, v := range freq {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	if maxTags < 1 {
		maxTags = 20
	}
	if len(arr) > maxTags {
		arr = arr[:maxTags]
	}
	out := make([]string, len(arr))
	for i := range arr {
		out[i] = arr[i].k
	}
	return out, nil
}
