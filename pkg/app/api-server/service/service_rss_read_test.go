/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 已读/未读判定逻辑单元测试。
*/

package service

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestIsRSSItemRead_ByItemRecord(t *testing.T) {
	id := primitive.NewObjectID()
	rc := &rssReadContext{
		readItemSet: map[primitive.ObjectID]struct{}{id: {}},
	}
	item := models.RSSItem{ID: id, FetchedAt: time.Now()}
	if !isRSSItemRead(item, rc) {
		t.Fatal("expected read via item record")
	}
}

func TestIsRSSItemRead_ByLastReadAt(t *testing.T) {
	rc := &rssReadContext{
		lastReadAt:  time.Unix(100, 0),
		readItemSet: map[primitive.ObjectID]struct{}{},
	}
	old := models.RSSItem{ID: primitive.NewObjectID(), FetchedAt: time.Unix(50, 0)}
	newer := models.RSSItem{ID: primitive.NewObjectID(), FetchedAt: time.Unix(200, 0)}
	if !isRSSItemRead(old, rc) {
		t.Fatal("old item should be read")
	}
	if isRSSItemRead(newer, rc) {
		t.Fatal("newer item should be unread")
	}
}

func TestIsRSSItemRead_NeverMarked(t *testing.T) {
	rc := &rssReadContext{readItemSet: map[primitive.ObjectID]struct{}{}}
	item := models.RSSItem{ID: primitive.NewObjectID(), FetchedAt: time.Now()}
	if isRSSItemRead(item, rc) {
		t.Fatal("expected unread when no cursor")
	}
}
