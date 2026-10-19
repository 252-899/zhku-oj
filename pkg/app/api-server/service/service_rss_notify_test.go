/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS SSE Hub 与载荷序列化单元测试。
*/

package service

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestMarshalRSSSSEEvent_TruncatesPreview(t *testing.T) {
	rows := make([]models.RSSItemListRow, 0, 25)
	base := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		rows = append(rows, models.RSSItemListRow{
			ID:        primitive.NewObjectID(),
			Title:     "t",
			FetchedAt: base.Add(time.Duration(i) * time.Second),
		})
	}
	raw := marshalRSSSSEEvent("rss_new", 1, rows)
	var ev models.RSSSSEEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Count != 25 {
		t.Fatalf("count=%d", ev.Count)
	}
	if len(ev.Items) != rssSSEMaxPushItems {
		t.Fatalf("preview len=%d", len(ev.Items))
	}
	if len(ev.IDs) != 25 {
		t.Fatalf("ids len=%d", len(ev.IDs))
	}
	if ev.Cursor == "" {
		t.Fatal("cursor empty")
	}
}

func TestRSSNotifyHub_FlushCoalescesPending(t *testing.T) {
	h := newRSSNotifyHub()
	ch, ok := h.trySubscribe()
	if !ok {
		t.Fatal("subscribe failed")
	}
	defer h.unsubscribe(ch)

	t1 := time.Now()
	id1 := primitive.NewObjectID()
	id2 := primitive.NewObjectID()
	h.publish([]models.RSSItemListRow{{ID: id1, Title: "a", FetchedAt: t1}})
	h.publish([]models.RSSItemListRow{{ID: id2, Title: "b", FetchedAt: t1.Add(time.Second)}})

	h.mu.Lock()
	h.flushLocked()
	h.mu.Unlock()

	select {
	case msg := <-ch:
		var ev models.RSSSSEEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Type != "rss_new" || ev.Count != 2 {
			t.Fatalf("ev=%+v", ev)
		}
	default:
		t.Fatal("expected one coalesced message")
	}
	select {
	case <-ch:
		t.Fatal("unexpected second message")
	default:
	}
}

func TestMergeRSSListRows_DedupByID(t *testing.T) {
	id := primitive.NewObjectID()
	old := models.RSSItemListRow{ID: id, Title: "old", FetchedAt: time.Unix(1, 0)}
	newer := models.RSSItemListRow{ID: id, Title: "new", FetchedAt: time.Unix(2, 0)}
	out := mergeRSSListRows([]models.RSSItemListRow{old}, []models.RSSItemListRow{newer})
	if len(out) != 1 || out[0].Title != "new" {
		t.Fatalf("got %+v", out)
	}
}

func TestRSSNotifyHub_MaxSubscribers(t *testing.T) {
	h := newRSSNotifyHub()
	h.maxSubs = 1

	c1, ok := h.trySubscribe()
	if !ok {
		t.Fatal()
	}
	_, ok = h.trySubscribe()
	if ok {
		t.Fatal("expected limit")
	}
	h.unsubscribe(c1)
	_, ok = h.trySubscribe()
	if !ok {
		t.Fatal("expected slot after unsubscribe")
	}
}
