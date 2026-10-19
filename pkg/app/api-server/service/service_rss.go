/*
@Author: xiaoTnb
@Date: 2026/05/12
@Description: RSS 资讯拉取、入库、手动刷新冷却与站内展示用数据组装（正文仅来自 RSS，不二次爬取）。
*/

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/mmcdole/gofeed"
	"github.com/microcosm-cc/bluemonday"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/ai/infra/llm"
	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

const (
	rssMaxItemsPerPull = 200
	rssHTTPTimeout     = 45 * time.Second
	rssPollInterval    = time.Hour
	rssManualCooldown  = time.Minute
	rssUserAgent       = "ZKCodeArena-RSS/1.0"
	rssMaxFeedBytes    = 4 << 20 // 4MB
)

// RSSService 负责订阅源维护、定时/手动同步、SSE 推送与个性化推荐。
type RSSService struct {
	repo       *repository.RSSRepository
	llmFactory llm.ModelFactory
	httpClient *http.Client
	parser     *gofeed.Parser
	policy     *bluemonday.Policy
	manualMu   sync.Mutex
	manualAt   map[string]time.Time
	hub        *rssNotifyHub
}

// NewRSSService 创建 RSS 业务服务实例；llmFactory 可为 nil（未配置 LLM 时跳过重排）。
func NewRSSService(repo *repository.RSSRepository, llmFactory llm.ModelFactory) *RSSService {
	p := gofeed.NewParser()
	p.UserAgent = rssUserAgent
	return &RSSService{
		repo:       repo,
		llmFactory: llmFactory,
		httpClient: &http.Client{
			Timeout: rssHTTPTimeout,
		},
		parser:   p,
		policy:   bluemonday.UGCPolicy(),
		manualAt: make(map[string]time.Time),
		hub:      newRSSNotifyHub(),
	}
}

// StartScheduler 每小时同步一次已启用订阅源；应在独立 goroutine 中调用并传入 stopCh。
func (s *RSSService) StartScheduler(ctx context.Context, stopCh <-chan struct{}) {
	ticker := time.NewTicker(rssPollInterval)
	defer ticker.Stop()
	s.syncAllSources(ctx)
	for {
		select {
		case <-stopCh:
			utils.Logger.Info("RSS 定时同步已停止")
			return
		case <-ticker.C:
			s.syncAllSources(ctx)
		}
	}
}

func (s *RSSService) syncAllSources(ctx context.Context) {
	sources, err := s.repo.ListEnabledRSSSources(ctx)
	if err != nil {
		utils.Logger.Errorf("RSS 列出订阅源失败: %v", err)
		return
	}
	for i := range sources {
		src := sources[i]
		if err := s.SyncSource(ctx, &src); err != nil {
			utils.Logger.Warnf("RSS 同步源失败 id=%s: %v", src.ID.Hex(), err)
		}
	}
}

func (s *RSSService) tryAcquireManual(key string) bool {
	s.manualMu.Lock()
	defer s.manualMu.Unlock()
	now := time.Now()
	if t, ok := s.manualAt[key]; ok && now.Sub(t) < rssManualCooldown {
		return false
	}
	s.manualAt[key] = now
	return true
}

// ManualRefreshSource 手动刷新单个源（1 分钟内同一源不可重复触发）。
func (s *RSSService) ManualRefreshSource(ctx context.Context, id primitive.ObjectID) error {
	if !s.tryAcquireManual(id.Hex()) {
		return fmt.Errorf("操作过于频繁，请间隔至少 1 分钟后再试")
	}
	src, err := s.repo.GetRSSSourceByID(ctx, id)
	if err != nil {
		return err
	}
	if !src.Enabled {
		return fmt.Errorf("该订阅源已禁用")
	}
	return s.SyncSource(ctx, src)
}

// ManualRefreshAll 手动刷新全部已启用源（1 分钟内不可重复触发全量刷新）。
func (s *RSSService) ManualRefreshAll(ctx context.Context) error {
	if !s.tryAcquireManual("all") {
		return fmt.Errorf("操作过于频繁，请间隔至少 1 分钟后再试")
	}
	sources, err := s.repo.ListEnabledRSSSources(ctx)
	if err != nil {
		return err
	}
	for i := range sources {
		if err := s.SyncSource(ctx, &sources[i]); err != nil {
			utils.Logger.Warnf("RSS 手动全量: 源 %s 失败: %v", sources[i].ID.Hex(), err)
		}
	}
	return nil
}

// SyncSource 拉取并解析单个订阅源并入库新条目。
func (s *RSSService) SyncSource(ctx context.Context, source *models.RSSSource) error {
	fetchAt := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.FeedURL, nil)
	if err != nil {
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, source.ETag, source.LastModified, fetchAt, false, err.Error())
		return err
	}
	req.Header.Set("User-Agent", rssUserAgent)
	if source.ETag != "" {
		req.Header.Set("If-None-Match", source.ETag)
	}
	if source.LastModified != "" {
		req.Header.Set("If-Modified-Since", source.LastModified)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, source.ETag, source.LastModified, fetchAt, false, err.Error())
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, rssMaxFeedBytes))
	if err != nil {
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, source.ETag, source.LastModified, fetchAt, false, err.Error())
		return err
	}
	newETag := resp.Header.Get("ETag")
	newLM := resp.Header.Get("Last-Modified")
	if resp.StatusCode == http.StatusNotModified {
		return s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, firstNonEmpty(newETag, source.ETag), firstNonEmpty(newLM, source.LastModified), fetchAt, true, "")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, source.ETag, source.LastModified, fetchAt, false, msg)
		return fmt.Errorf("%s", msg)
	}
	feed, err := s.parser.ParseString(string(body))
	if err != nil {
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, source.ETag, source.LastModified, fetchAt, false, err.Error())
		return err
	}
	items := s.buildItemsFromFeed(feed, source)
	inserted, err := s.repo.InsertRSSItemsIgnoreDuplicate(ctx, items)
	if err != nil {
		_ = s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, newETag, newLM, fetchAt, false, err.Error())
		return err
	}
	if len(inserted) > 0 {
		s.hub.publish(itemsToListRows(inserted))
	}
	return s.repo.UpdateRSSSourceFetchMeta(ctx, source.ID, newETag, newLM, fetchAt, true, "")
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (s *RSSService) buildItemsFromFeed(feed *gofeed.Feed, source *models.RSSSource) []models.RSSItem {
	n := len(feed.Items)
	if n > rssMaxItemsPerPull {
		n = rssMaxItemsPerPull
	}
	now := time.Now()
	out := make([]models.RSSItem, 0, n)
	for i := 0; i < n; i++ {
		it := feed.Items[i]
		raw, srcKey := pickRSSBody(it)
		body := s.policy.Sanitize(strings.TrimSpace(raw))
		link := firstItemLink(it)
		guid := stableItemGUID(it, link)
		pub := now
		if it.PublishedParsed != nil {
			pub = *it.PublishedParsed
		}
		title := strings.TrimSpace(it.Title)
		tags := ExtractRSSTags(title, body, source.Name)
		out = append(out, models.RSSItem{
			SourceID:    source.ID,
			SourceName:  source.Name,
			GUID:        guid,
			Link:        link,
			Title:       title,
			BodyHTML:    body,
			BodySource:  srcKey,
			PublishedAt: pub,
			FetchedAt:   now,
			Tags:        tags,
			TermVector:  BuildRSSTermVector(title, body, tags),
		})
	}
	return out
}

// pickRSSBody 按约定优先使用全文节点再退回 description（仅 RSS 内字段）。
func pickRSSBody(it *gofeed.Item) (rawHTML, sourceKey string) {
	if strings.TrimSpace(it.Content) != "" {
		return it.Content, "content"
	}
	if strings.TrimSpace(it.Description) != "" {
		return it.Description, "description"
	}
	return "", "empty"
}

func firstItemLink(it *gofeed.Item) string {
	if strings.TrimSpace(it.Link) != "" {
		return strings.TrimSpace(it.Link)
	}
	if len(it.Links) > 0 {
		return strings.TrimSpace(it.Links[0])
	}
	return ""
}

func stableItemGUID(it *gofeed.Item, link string) string {
	g := strings.TrimSpace(it.GUID)
	if g != "" {
		return g
	}
	sum := sha256.Sum256([]byte(link + "\x00" + it.Title + "\x00" + it.Published))
	return "zk:" + hex.EncodeToString(sum[:])
}

// RSSPlainSummary 从 HTML 生成列表用纯文本摘要（与列表接口配合）。
func RSSPlainSummary(html string, maxRunes int) string {
	plain := bluemonday.StrictPolicy().Sanitize(html)
	plain = strings.Join(strings.Fields(plain), " ")
	if maxRunes <= 0 {
		maxRunes = 160
	}
	if utf8.RuneCountInString(plain) <= maxRunes {
		return plain
	}
	r := []rune(plain)
	return string(r[:maxRunes]) + "…"
}

// CreateRSSSource 创建订阅源。
func (s *RSSService) CreateRSSSource(ctx context.Context, req *models.CreateRSSSourceRequest) error {
	if err := validateFeedURL(req.FeedURL); err != nil {
		return err
	}
	doc := &models.RSSSource{
		Name:    strings.TrimSpace(req.Name),
		FeedURL: strings.TrimSpace(req.FeedURL),
		Enabled: true,
	}
	if req.Enabled != nil {
		doc.Enabled = *req.Enabled
	}
	return s.repo.InsertRSSSource(ctx, doc)
}

// UpdateRSSSource 更新订阅源。
func (s *RSSService) UpdateRSSSource(ctx context.Context, id primitive.ObjectID, req *models.UpdateRSSSourceRequest) error {
	patch := bson.M{}
	if req.Name != "" {
		patch["name"] = strings.TrimSpace(req.Name)
	}
	if req.FeedURL != "" {
		if err := validateFeedURL(req.FeedURL); err != nil {
			return err
		}
		patch["feed_url"] = strings.TrimSpace(req.FeedURL)
	}
	if req.Enabled != nil {
		patch["enabled"] = *req.Enabled
	}
	if len(patch) == 0 {
		return fmt.Errorf("没有需要更新的字段")
	}
	if err := s.repo.UpdateRSSSource(ctx, id, patch); err != nil {
		return err
	}
	return nil
}

func validateFeedURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("无效的订阅地址")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("仅支持 http(s) 订阅地址")
	}
	return nil
}

// DeleteRSSSource 删除订阅源及其条目。
func (s *RSSService) DeleteRSSSource(ctx context.Context, id primitive.ObjectID) error {
	return s.repo.DeleteRSSSource(ctx, id)
}

// ListRSSSources 管理端列出订阅源。
func (s *RSSService) ListRSSSources(ctx context.Context) ([]models.RSSSource, error) {
	return s.repo.ListRSSSources(ctx)
}

// GetRSSSource 获取单个订阅源。
func (s *RSSService) GetRSSSource(ctx context.Context, id primitive.ObjectID) (*models.RSSSource, error) {
	return s.repo.GetRSSSourceByID(ctx, id)
}

// ListRSSItemsNewerThan 按入库时间查询新条目（轮询/SSE 快照，不含已读状态）。
func (s *RSSService) ListRSSItemsNewerThan(ctx context.Context, after time.Time, limit int64) ([]models.RSSItemListRow, error) {
	items, err := s.repo.ListRSSItemsAfterFetchedAt(ctx, after, limit)
	if err != nil {
		return nil, err
	}
	return itemsToListRows(items), nil
}

// SnapshotRSSItemsSince 查询 since 之后的新条并生成 snapshot 用 SSE 载荷（连接建立时补全）。
func (s *RSSService) SnapshotRSSItemsSince(ctx context.Context, after time.Time) ([]byte, error) {
	if after.IsZero() {
		after = time.Now().Add(-rssSnapshotLookback)
	}
	rows, err := s.ListRSSItemsNewerThan(ctx, after, rssSnapshotLimit)
	if err != nil {
		return nil, err
	}
	return marshalRSSSSEEvent("snapshot", 0, rows), nil
}

// TrySubscribeRSSNotifications 尝试订阅 SSE；连接数已满时 ok=false。
func (s *RSSService) TrySubscribeRSSNotifications() (ch chan []byte, ok bool) {
	return s.hub.trySubscribe()
}

// UnsubscribeRSSNotifications 取消订阅并关闭通道。
func (s *RSSService) UnsubscribeRSSNotifications(ch chan []byte) {
	s.hub.unsubscribe(ch)
}
