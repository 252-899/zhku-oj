/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 个性化推荐：规则打分 + 用户画像 + 向量相似 + 可选 LLM 重排。
*/

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

const (
	rssRecommendCandidateLimit = 400
	rssRecommendRerankPool     = 40
	rssRecommendLLMTimeout     = 20 * time.Second
)

// 打分权重：融合三阶段信号。
const (
	weightFreshness   = 0.22
	weightSourceAff   = 0.18
	weightTagMatch    = 0.22
	weightPopularity  = 0.12
	weightVectorSim   = 0.18
	weightUnreadBoost = 0.08
)

type rssUserProfile struct {
	TagWeights     map[string]float64
	SourceAffinity map[string]float64
	TermVector     map[string]float64
	BlockedItems   map[primitive.ObjectID]struct{}
}

type rssScoredItem struct {
	item   models.RSSItem
	score  float64
	reason string
}

// RecordRSSEvent 记录用户行为并更新条目热度。
func (s *RSSService) RecordRSSEvent(ctx context.Context, userID string, req *models.RecordRSSEventRequest) error {
	if req == nil || strings.TrimSpace(req.ItemID) == "" {
		return fmt.Errorf("item_id 不能为空")
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "open", "click", "dwell", "not_interested":
	default:
		return fmt.Errorf("不支持的 action: %s", action)
	}
	itemID, err := primitive.ObjectIDFromHex(req.ItemID)
	if err != nil {
		return fmt.Errorf("无效的资讯 ID")
	}
	item, err := s.repo.GetRSSItemByID(ctx, itemID)
	if err != nil {
		return err
	}
	ev := &models.RSSUserEvent{
		UserID:    userID,
		ItemID:    itemID,
		SourceID:  item.SourceID,
		Action:    action,
		DwellMs:   req.DwellMs,
		CreatedAt: time.Now(),
	}
	if err := s.repo.InsertRSSUserEvent(ctx, ev); err != nil {
		return err
	}
	switch action {
	case "open":
		_ = s.repo.IncrementRSSItemStat(ctx, itemID, 1, 0)
	case "click":
		_ = s.repo.IncrementRSSItemStat(ctx, itemID, 0, 1)
	case "dwell":
		if req.DwellMs >= 3000 {
			_ = s.repo.IncrementRSSItemStat(ctx, itemID, 1, 0)
		}
	}
	if action == "open" {
		_ = s.MarkRSSItemRead(ctx, userID, itemID)
	}
	return nil
}

// GetRSSRecommendedFeed 个性化推荐 Feed（分页）。
func (s *RSSService) GetRSSRecommendedFeed(ctx context.Context, userID string, page, pageSize int64) ([]models.RSSFeedRecommendRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	userOID, _ := primitive.ObjectIDFromHex(userID)

	profile, err := s.buildRSSUserProfile(ctx, userID, userOID)
	if err != nil {
		return nil, 0, err
	}
	readCtx, err := s.loadRSSReadContext(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	candidates, err := s.repo.ListRSSItemsForRecommend(ctx, rssRecommendCandidateLimit)
	if err != nil {
		return nil, 0, err
	}
	scored := make([]rssScoredItem, 0, len(candidates))
	now := time.Now()
	for i := range candidates {
		it := candidates[i]
		if _, blocked := profile.BlockedItems[it.ID]; blocked {
			continue
		}
		EnsureItemEnrichment(&it)
		sc, reason := s.scoreRSSItem(&it, profile, readCtx, now)
		if sc <= 0 {
			continue
		}
		scored = append(scored, rssScoredItem{item: it, score: sc, reason: reason})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if len(scored) > rssRecommendRerankPool {
		head := s.llmRerankRSSItems(ctx, userID, profile, scored[:rssRecommendRerankPool])
		scored = append(head, scored[rssRecommendRerankPool:]...)
	} else if len(scored) > 1 {
		scored = s.llmRerankRSSItems(ctx, userID, profile, scored)
	}

	total := int64(len(scored))
	start := (page - 1) * pageSize
	if start >= int64(len(scored)) {
		return []models.RSSFeedRecommendRow{}, total, nil
	}
	end := start + pageSize
	if end > int64(len(scored)) {
		end = int64(len(scored))
	}
	pageItems := scored[start:end]

	rows := make([]models.RSSFeedRecommendRow, 0, len(pageItems))
	for i := range pageItems {
		it := &pageItems[i].item
		listRow := itemsToListRows([]models.RSSItem{*it})[0]
		listRow.IsRead = isRSSItemRead(*it, readCtx)
		rows = append(rows, models.RSSFeedRecommendRow{
			RSSItemListRow: listRow,
			Score:          roundScore(pageItems[i].score),
			Reason:         pageItems[i].reason,
		})
	}
	return rows, total, nil
}

func (s *RSSService) buildRSSUserProfile(ctx context.Context, userID string, userOID primitive.ObjectID) (*rssUserProfile, error) {
	p := &rssUserProfile{
		TagWeights:     make(map[string]float64),
		SourceAffinity: make(map[string]float64),
		TermVector:     make(map[string]float64),
		BlockedItems:   make(map[primitive.ObjectID]struct{}),
	}

	events, err := s.repo.ListRSSUserEvents(ctx, userID, 120)
	if err != nil {
		return nil, err
	}
	for _, ev := range events {
		if ev.Action == "not_interested" {
			p.BlockedItems[ev.ItemID] = struct{}{}
			continue
		}
		w := 0.3
		switch ev.Action {
		case "click":
			w = 0.6
		case "open":
			w = 1.0
		case "dwell":
			if ev.DwellMs >= 5000 {
				w = 1.2
			} else {
				w = 0.8
			}
		}
		p.SourceAffinity[ev.SourceID.Hex()] += w
		item, err := s.repo.GetRSSItemByID(ctx, ev.ItemID)
		if err != nil {
			continue
		}
		EnsureItemEnrichment(item)
		for _, tag := range item.Tags {
			p.TagWeights[tag] += w
		}
		for k, v := range item.TermVector {
			p.TermVector[k] += v * w
		}
	}

	if !userOID.IsZero() {
		problemTags, err := s.repo.AggregateUserProblemTags(ctx, userOID, 30)
		if err != nil {
			utils.Logger.Warnf("RSS 推荐: 拉取题目标签失败 user=%s: %v", userID, err)
		} else {
			for _, t := range problemTags {
				norm := normalizeProblemTagForRSS(t)
				p.TagWeights[norm] += 1.5
			}
		}
	}

	p.TermVector = normalizeTermVector(p.TermVector)
	return p, nil
}

func normalizeProblemTagForRSS(tag string) string {
	t := strings.ToLower(strings.TrimSpace(tag))
	for _, rule := range rssTagKeywords {
		for _, kw := range rule.keywords {
			if strings.Contains(t, strings.ToLower(kw)) {
				return rule.tag
			}
		}
	}
	return "programming"
}

func (s *RSSService) scoreRSSItem(item *models.RSSItem, profile *rssUserProfile, readCtx *rssReadContext, now time.Time) (float64, string) {
	hours := now.Sub(item.PublishedAt).Hours()
	if hours < 0 {
		hours = 0
	}
	fresh := math.Exp(-hours / 72.0)

	srcAff := 0.0
	if profile != nil {
		srcAff = profile.SourceAffinity[item.SourceID.Hex()]
		if srcAff > 1 {
			srcAff = 1
		}
	}

	tagSc := tagMatchScore(profile.TagWeights, item.Tags)
	vecSc := cosineSimilarity(profile.TermVector, item.TermVector)
	if vecSc < 0 {
		vecSc = 0
	}

	pop := math.Log1p(float64(item.OpenCount)*1.0 + float64(item.ClickCount)*0.5)
	if pop > 3 {
		pop = 3
	}
	pop /= 3.0

	unreadBoost := 0.0
	reason := "最新资讯"
	if isRSSItemRead(*item, readCtx) {
		unreadBoost = -0.35
		reason = "已读降权"
	} else {
		unreadBoost = 1.0
	}

	score := weightFreshness*fresh +
		weightSourceAff*srcAff +
		weightTagMatch*tagSc +
		weightPopularity*pop +
		weightVectorSim*vecSc +
		weightUnreadBoost*unreadBoost

	if tagSc > 0.4 && reason != "已读降权" {
		reason = "与您的学习兴趣相关"
	} else if srcAff > 0.5 && reason != "已读降权" {
		reason = "常读来源"
	} else if vecSc > 0.35 && reason != "已读降权" {
		reason = "内容相似推荐"
	} else if pop > 0.6 && reason != "已读降权" {
		reason = "平台热门"
	}
	return score, reason
}

func roundScore(v float64) float64 {
	return math.Round(v*1000) / 1000
}

// llmRerankRSSItems 阶段 3：用 LLM 对 Top 候选重排；失败则保持原序。
func (s *RSSService) llmRerankRSSItems(ctx context.Context, userID string, profile *rssUserProfile, items []rssScoredItem) []rssScoredItem {
	if s.llmFactory == nil || !s.llmFactory.IsChatConfigured() || len(items) < 2 {
		return items
	}
	topTags := topTagKeys(profile.TagWeights, 8)
	var b strings.Builder
	b.WriteString("用户兴趣标签: ")
	b.WriteString(strings.Join(topTags, ", "))
	b.WriteString("\n请根据相关性对下列资讯标题排序，只返回 JSON 数组，元素为 id 字符串，最相关在前。\n")
	for i := range items {
		if i >= 25 {
			break
		}
		b.WriteString(fmt.Sprintf("- id=%s title=%s\n", items[i].item.ID.Hex(), items[i].item.Title))
	}
	rerankCtx, cancel := context.WithTimeout(ctx, rssRecommendLLMTimeout)
	defer cancel()
	msg, err := s.llmFactory.Call(rerankCtx, []*schema.Message{
		{Role: schema.System, Content: "你是资讯推荐排序助手，只输出 JSON 数组，不要解释。"},
		{Role: schema.User, Content: b.String()},
	})
	if err != nil || msg == nil || strings.TrimSpace(msg.Content) == "" {
		utils.Logger.Warnf("RSS LLM 重排跳过 user=%s: %v", userID, err)
		return items
	}
	ordered := parseLLMRerankIDs(msg.Content)
	if len(ordered) == 0 {
		return items
	}
	byID := make(map[string]rssScoredItem, len(items))
	for _, it := range items {
		byID[it.item.ID.Hex()] = it
	}
	out := make([]rssScoredItem, 0, len(items))
	seen := make(map[string]struct{})
	for _, id := range ordered {
		if it, ok := byID[id]; ok {
			it.reason = "AI 精选"
			out = append(out, it)
			seen[id] = struct{}{}
		}
	}
	for _, it := range items {
		id := it.item.ID.Hex()
		if _, ok := seen[id]; !ok {
			out = append(out, it)
		}
	}
	return out
}

func topTagKeys(m map[string]float64, n int) []string {
	type kv struct {
		k string
		v float64
	}
	arr := make([]kv, 0, len(m))
	for k, v := range m {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	if len(arr) > n {
		arr = arr[:n]
	}
	out := make([]string, len(arr))
	for i := range arr {
		out[i] = arr[i].k
	}
	return out
}

func parseLLMRerankIDs(content string) []string {
	content = strings.TrimSpace(content)
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")
	if start < 0 || end <= start {
		return nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(content[start:end+1]), &ids); err != nil {
		return nil
	}
	return ids
}
