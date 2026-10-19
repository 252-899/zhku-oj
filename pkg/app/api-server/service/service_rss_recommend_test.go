/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 标签与推荐打分单元测试。
*/

package service

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/models"
)

func TestExtractRSSTags_Algorithm(t *testing.T) {
	tags := ExtractRSSTags("动态规划入门", "", "OI News")
	found := false
	for _, tag := range tags {
		if tag == "algorithm" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tags=%v", tags)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := map[string]float64{"go": 0.8, "tag:programming": 0.6}
	b := map[string]float64{"go": 0.7, "tag:programming": 0.7}
	if cosineSimilarity(a, b) <= 0.5 {
		t.Fatal("expected positive similarity")
	}
}

func TestScoreRSSItem_UnreadHigherThanRead(t *testing.T) {
	svc := &RSSService{}
	profile := &rssUserProfile{TagWeights: map[string]float64{"go": 1}}
	readCtx := &rssReadContext{lastReadAt: time.Unix(0, 0)}
	item := models.RSSItem{
		ID:          primitive.NewObjectID(),
		Title:       "Go 并发",
		PublishedAt: time.Now(),
		Tags:        []string{"go", "programming"},
		TermVector:  BuildRSSTermVector("Go 并发", "", []string{"go"}),
		FetchedAt:   time.Now(),
	}
	unreadScore, _ := svc.scoreRSSItem(&item, profile, readCtx, time.Now())
	readCtx.lastReadAt = time.Now().Add(time.Hour)
	readScore, _ := svc.scoreRSSItem(&item, profile, readCtx, time.Now())
	if unreadScore <= readScore {
		t.Fatalf("unread=%v read=%v", unreadScore, readScore)
	}
}

func TestParseLLMRerankIDs(t *testing.T) {
	ids := parseLLMRerankIDs("排序结果如下：\n[\"abc\",\"def\"]")
	if len(ids) != 2 || ids[0] != "abc" {
		t.Fatalf("got %v", ids)
	}
}
