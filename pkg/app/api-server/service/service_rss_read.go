/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 资讯已读/未读：全局游标 + 单条已读，供列表红点与未读计数。
*/

package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

// rssReadContext 用户已读状态快照，避免列表页重复查库。
type rssReadContext struct {
	lastReadAt  time.Time
	readItemSet map[primitive.ObjectID]struct{}
}

func (s *RSSService) loadRSSReadContext(ctx context.Context, userID string) (*rssReadContext, error) {
	st, err := s.repo.GetRSSReadState(ctx, userID)
	if err != nil {
		return nil, err
	}
	readIDs, err := s.repo.ListRSSItemReadIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := make(map[primitive.ObjectID]struct{}, len(readIDs))
	for _, id := range readIDs {
		set[id] = struct{}{}
	}
	return &rssReadContext{
		lastReadAt:  st.LastReadAt,
		readItemSet: set,
	}, nil
}

// isRSSItemRead 判断单条是否已读：单条记录优先，否则看是否早于全局游标。
func isRSSItemRead(item models.RSSItem, rc *rssReadContext) bool {
	if rc == nil {
		return false
	}
	if _, ok := rc.readItemSet[item.ID]; ok {
		return true
	}
	if rc.lastReadAt.IsZero() {
		return false
	}
	return !item.FetchedAt.After(rc.lastReadAt)
}

func enrichListRowsWithRead(items []models.RSSItem, rc *rssReadContext) []models.RSSItemListRow {
	rows := itemsToListRows(items)
	for i := range rows {
		rows[i].IsRead = isRSSItemRead(items[i], rc)
	}
	return rows
}

// GetRSSUnreadCount 当前用户未读资讯数量。
func (s *RSSService) GetRSSUnreadCount(ctx context.Context, userID string) (int64, *models.RSSReadState, error) {
	rc, err := s.loadRSSReadContext(ctx, userID)
	if err != nil {
		return 0, nil, err
	}
	readIDs := make([]primitive.ObjectID, 0, len(rc.readItemSet))
	for id := range rc.readItemSet {
		readIDs = append(readIDs, id)
	}
	n, err := s.repo.CountRSSUnreadItems(ctx, userID, rc.lastReadAt, readIDs)
	if err != nil {
		return 0, nil, err
	}
	st, _ := s.repo.GetRSSReadState(ctx, userID)
	return n, st, nil
}

// MarkRSSItemRead 标记单条资讯已读。
func (s *RSSService) MarkRSSItemRead(ctx context.Context, userID string, itemID primitive.ObjectID) error {
	if _, err := s.repo.GetRSSItemByID(ctx, itemID); err != nil {
		return err
	}
	return s.repo.UpsertRSSItemRead(ctx, userID, itemID, time.Now())
}

// MarkRSSAllRead 全部已读：将全局游标设为当前时间。
func (s *RSSService) MarkRSSAllRead(ctx context.Context, userID string) error {
	return s.repo.UpsertRSSLastReadAt(ctx, userID, time.Now())
}

// MarkRSSRead 按请求标记已读（单条或全部）。
func (s *RSSService) MarkRSSRead(ctx context.Context, userID string, req *models.MarkRSSReadRequest) error {
	if req == nil {
		return fmt.Errorf("请求体不能为空")
	}
	if req.MarkAll {
		if req.ItemID != "" {
			return fmt.Errorf("mark_all 与 item_id 不能同时设置")
		}
		return s.MarkRSSAllRead(ctx, userID)
	}
	if req.ItemID == "" {
		return fmt.Errorf("请提供 item_id 或设置 mark_all")
	}
	id, err := primitive.ObjectIDFromHex(req.ItemID)
	if err != nil {
		return fmt.Errorf("无效的资讯 ID")
	}
	return s.MarkRSSItemRead(ctx, userID, id)
}

// ListRSSItemsForUser 分页列出资讯并附带 is_read。
func (s *RSSService) ListRSSItemsForUser(ctx context.Context, userID string, sourceID *primitive.ObjectID, page, pageSize int64) ([]models.RSSItemListRow, int64, error) {
	items, total, err := s.repo.ListRSSItems(ctx, sourceID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	rc, err := s.loadRSSReadContext(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return enrichListRowsWithRead(items, rc), total, nil
}

// ListRSSItemsNewerThanForUser 查询新条目并附带 is_read（轮询补拉）。
func (s *RSSService) ListRSSItemsNewerThanForUser(ctx context.Context, userID string, after time.Time, limit int64) ([]models.RSSItemListRow, error) {
	items, err := s.repo.ListRSSItemsAfterFetchedAt(ctx, after, limit)
	if err != nil {
		return nil, err
	}
	rc, err := s.loadRSSReadContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	return enrichListRowsWithRead(items, rc), nil
}

// GetRSSItemDetailForUser 获取详情及是否已读。
func (s *RSSService) GetRSSItemDetailForUser(ctx context.Context, userID string, id primitive.ObjectID) (*models.RSSItemDetail, error) {
	item, err := s.repo.GetRSSItemByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rc, err := s.loadRSSReadContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &models.RSSItemDetail{
		RSSItem: *item,
		IsRead:  isRSSItemRead(*item, rc),
	}, nil
}
