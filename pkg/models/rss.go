/*
@Author: xiaoTnb
@Date: 2026/05/12
@Description: RSS 资讯聚合相关数据模型（订阅源、条目及请求 DTO）
*/

package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RSSSource 表示一条可拉取的 RSS/Atom 订阅源配置。
type RSSSource struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name          string             `bson:"name" json:"name"`
	FeedURL       string             `bson:"feed_url" json:"feed_url"`
	Enabled       bool               `bson:"enabled" json:"enabled"`
	ETag          string             `bson:"etag,omitempty" json:"etag,omitempty"`
	LastModified  string             `bson:"last_modified,omitempty" json:"last_modified,omitempty"`
	LastFetchAt   *time.Time         `bson:"last_fetch_at,omitempty" json:"last_fetch_at,omitempty"`
	LastSuccessAt *time.Time         `bson:"last_success_at,omitempty" json:"last_success_at,omitempty"`
	LastError     string             `bson:"last_error,omitempty" json:"last_error,omitempty"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// RSSItem 表示从订阅源解析后入库的一条资讯（正文仅来自 RSS，不做二次爬取）。
type RSSItem struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SourceID     primitive.ObjectID `bson:"source_id" json:"source_id"`
	SourceName   string             `bson:"source_name" json:"source_name"`
	GUID         string             `bson:"guid" json:"guid"`
	Link         string             `bson:"link" json:"link"`
	Title        string             `bson:"title" json:"title"`
	BodyHTML     string             `bson:"body_html" json:"body_html"`
	BodySource   string             `bson:"body_source" json:"body_source"` // content | description | empty
	PublishedAt  time.Time          `bson:"published_at" json:"published_at"`
	FetchedAt    time.Time          `bson:"fetched_at" json:"fetched_at"`
	Tags         []string           `bson:"tags,omitempty" json:"tags,omitempty"`
	TermVector   map[string]float64 `bson:"term_vector,omitempty" json:"-"`
	OpenCount    int                `bson:"open_count,omitempty" json:"open_count,omitempty"`
	ClickCount   int                `bson:"click_count,omitempty" json:"click_count,omitempty"`
}

// RSSItemListRow 列表接口返回的单行（不含大段 HTML，减少流量）。
type RSSItemListRow struct {
	ID          primitive.ObjectID `json:"id"`
	SourceID    primitive.ObjectID `json:"source_id"`
	SourceName  string             `json:"source_name"`
	Title       string             `json:"title"`
	Link        string             `json:"link"`
	Summary     string             `json:"summary"`
	PublishedAt time.Time          `json:"published_at"`
	FetchedAt   time.Time          `json:"fetched_at"`
	IsRead      bool               `json:"is_read"`
}

// RSSReadState 用户资讯已读游标（按用户一条，用于「全部已读」与未读计数）。
type RSSReadState struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string             `bson:"user_id" json:"user_id"`
	LastReadAt  time.Time          `bson:"last_read_at" json:"last_read_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// RSSItemRead 单条资讯已读记录（用于点开单条已读且不影响全局游标之前的条目）。
type RSSItemRead struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	ItemID    primitive.ObjectID `bson:"item_id" json:"item_id"`
	ReadAt    time.Time          `bson:"read_at" json:"read_at"`
}

// RSSItemDetail 资讯详情（含当前用户是否已读）。
type RSSItemDetail struct {
	RSSItem
	IsRead bool `json:"is_read"`
}

// MarkRSSReadRequest 标记已读请求：item_id 与 mark_all 二选一。
type MarkRSSReadRequest struct {
	ItemID  string `json:"item_id"`
	MarkAll bool   `json:"mark_all"`
}

// RSSUserEvent 用户资讯行为（推荐画像与热度统计）。
type RSSUserEvent struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	ItemID    primitive.ObjectID `bson:"item_id" json:"item_id"`
	SourceID  primitive.ObjectID `bson:"source_id,omitempty" json:"source_id,omitempty"`
	Action    string             `bson:"action" json:"action"` // open | click | dwell | not_interested
	DwellMs   int64              `bson:"dwell_ms,omitempty" json:"dwell_ms,omitempty"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// RecordRSSEventRequest 上报资讯行为。
type RecordRSSEventRequest struct {
	ItemID  string `json:"item_id" binding:"required"`
	Action  string `json:"action" binding:"required"`
	DwellMs int64  `json:"dwell_ms"`
}

// RSSFeedRecommendRow 个性化推荐列表行。
type RSSFeedRecommendRow struct {
	RSSItemListRow
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
}

// CreateRSSSourceRequest 创建订阅源请求体。
type CreateRSSSourceRequest struct {
	Name    string `json:"name" binding:"required"`
	FeedURL string `json:"feed_url" binding:"required"`
	Enabled *bool  `json:"enabled"`
}

// UpdateRSSSourceRequest 更新订阅源请求体（字段均为可选）。
type UpdateRSSSourceRequest struct {
	Name    string `json:"name"`
	FeedURL string `json:"feed_url"`
	Enabled *bool  `json:"enabled"`
}

// RSSSSEEvent RSS 资讯 SSE 推送事件体（snapshot / rss_new / backlog）。
type RSSSSEEvent struct {
	Type    string           `json:"type"`
	EventID string           `json:"event_id"`
	Ts      int64            `json:"ts"`
	Cursor  string           `json:"cursor,omitempty"`
	Count   int              `json:"count"`
	Items   []RSSItemListRow `json:"items,omitempty"`
	IDs     []string         `json:"ids,omitempty"`
}
