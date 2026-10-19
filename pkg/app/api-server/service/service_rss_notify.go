/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 新条 SSE 通知 Hub：连接上限、500ms 合并推送、慢消费者 backlog 提示。
*/

package service

import (
	"encoding/json"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

const (
	rssSSECoalesceWindow  = 500 * time.Millisecond
	rssSSEMaxSubscribers  = 2000
	rssSSESubChannelCap   = 16
	rssSSEMaxPushItems    = 20
	rssSnapshotLimit      = 50
	rssSnapshotLookback   = 24 * time.Hour
)

// rssNotifyHub 进程内 SSE 广播中心，合并短时间内的多条新条通知。
type rssNotifyHub struct {
	mu         sync.Mutex
	subs       map[chan []byte]*rssSubState
	pending    []models.RSSItemListRow
	pendingIDs map[primitive.ObjectID]struct{}
	flushTimer *time.Timer
	eventSeq   uint64
	maxSubs    int // 0 表示使用 rssSSEMaxSubscribers
}

type rssSubState struct {
	dropped int
}

func newRSSNotifyHub() *rssNotifyHub {
	return &rssNotifyHub{
		subs:       make(map[chan []byte]*rssSubState),
		pendingIDs: make(map[primitive.ObjectID]struct{}),
	}
}

func (h *rssNotifyHub) subscriberLimit() int {
	if h.maxSubs > 0 {
		return h.maxSubs
	}
	return rssSSEMaxSubscribers
}

// trySubscribe 注册订阅；连接数超限时返回 false。
func (h *rssNotifyHub) trySubscribe() (chan []byte, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.subs) >= h.subscriberLimit() {
		return nil, false
	}
	ch := make(chan []byte, rssSSESubChannelCap)
	h.subs[ch] = &rssSubState{}
	return ch, true
}

func (h *rssNotifyHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.subs[ch]; ok {
		delete(h.subs, ch)
		close(ch)
	}
}

// publish 将新条加入待推送批次，并在合并窗口结束后统一广播。
func (h *rssNotifyHub) publish(rows []models.RSSItemListRow) {
	if len(rows) == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range rows {
		r := rows[i]
		if _, dup := h.pendingIDs[r.ID]; dup {
			continue
		}
		h.pendingIDs[r.ID] = struct{}{}
		h.pending = append(h.pending, r)
	}
	if h.flushTimer == nil {
		h.flushTimer = time.AfterFunc(rssSSECoalesceWindow, h.flushAsync)
	} else {
		h.flushTimer.Reset(rssSSECoalesceWindow)
	}
}

func (h *rssNotifyHub) flushAsync() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.flushLocked()
}

func (h *rssNotifyHub) flushLocked() {
	if h.flushTimer != nil {
		h.flushTimer.Stop()
		h.flushTimer = nil
	}
	if len(h.pending) == 0 {
		return
	}
	h.eventSeq++
	batch := h.pending
	h.pending = nil
	h.pendingIDs = make(map[primitive.ObjectID]struct{})

	payload := marshalRSSSSEEvent("rss_new", h.eventSeq, batch)
	cursor := rssItemsCursor(batch)
	backlogPayload := marshalRSSSSEEvent("backlog", h.eventSeq, nil)
	backlogPayload = overrideRSSSSEFields(backlogPayload, "backlog", cursor)

	for ch, st := range h.subs {
		select {
		case ch <- payload:
		default:
			st.dropped++
			select {
			case ch <- backlogPayload:
			default:
			}
			utils.Logger.Warnf("RSS SSE 客户端落后，已丢弃实时包 dropped=%d cursor=%s", st.dropped, cursor)
		}
	}
}

// itemsToListRows 将入库条目转为列表行（供 SSE 与 API 复用）。
func itemsToListRows(items []models.RSSItem) []models.RSSItemListRow {
	rows := make([]models.RSSItemListRow, 0, len(items))
	for i := range items {
		it := &items[i]
		rows = append(rows, models.RSSItemListRow{
			ID:          it.ID,
			SourceID:    it.SourceID,
			SourceName:  it.SourceName,
			Title:       it.Title,
			Link:        it.Link,
			Summary:     RSSPlainSummary(it.BodyHTML, 200),
			PublishedAt: it.PublishedAt,
			FetchedAt:   it.FetchedAt,
		})
	}
	return rows
}

// rssItemsCursor 取一批条目中最大的 fetched_at 作为客户端续传游标。
func rssItemsCursor(rows []models.RSSItemListRow) string {
	var max time.Time
	for i := range rows {
		if rows[i].FetchedAt.After(max) {
			max = rows[i].FetchedAt
		}
	}
	if max.IsZero() {
		return ""
	}
	return max.UTC().Format(time.RFC3339Nano)
}

// marshalRSSSSEEvent 序列化 SSE 业务载荷；条数超过上限时仅附带预览 items。
func marshalRSSSSEEvent(eventType string, seq uint64, rows []models.RSSItemListRow) []byte {
	count := len(rows)
	cursor := rssItemsCursor(rows)
	eventID := cursor
	if eventID == "" {
		eventID = formatRSSEventID(seq)
	}
	preview := rows
	if len(preview) > rssSSEMaxPushItems {
		preview = preview[len(preview)-rssSSEMaxPushItems:]
	}
	ids := make([]string, 0, count)
	for i := range rows {
		ids = append(ids, rows[i].ID.Hex())
	}
	ev := models.RSSSSEEvent{
		Type:    eventType,
		EventID: eventID,
		Ts:      time.Now().UnixMilli(),
		Cursor:  cursor,
		Count:   count,
		Items:   preview,
		IDs:     ids,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return []byte(`{"type":"error","event_id":"0"}`)
	}
	return b
}

func formatRSSEventID(seq uint64) string {
	return time.Unix(0, int64(seq)).UTC().Format(time.RFC3339Nano)
}

func overrideRSSSSEFields(raw []byte, eventType, cursor string) []byte {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw
	}
	m["type"] = eventType
	if cursor != "" {
		m["cursor"] = cursor
	}
	m["items"] = nil
	m["ids"] = nil
	m["count"] = 0
	b, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return b
}

// mergeRSSListRows 合并两批列表行并按 ID 去重，保留 fetched_at 较新的顺序末尾条。
func mergeRSSListRows(base, add []models.RSSItemListRow) []models.RSSItemListRow {
	seen := make(map[primitive.ObjectID]models.RSSItemListRow, len(base)+len(add))
	for i := range base {
		seen[base[i].ID] = base[i]
	}
	for i := range add {
		id := add[i].ID
		if old, ok := seen[id]; !ok || add[i].FetchedAt.After(old.FetchedAt) {
			seen[id] = add[i]
		}
	}
	out := make([]models.RSSItemListRow, 0, len(seen))
	for _, r := range seen {
		out = append(out, r)
	}
	// 按 fetched_at 升序，便于 cursor 取 max
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].FetchedAt.Before(out[i].FetchedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
