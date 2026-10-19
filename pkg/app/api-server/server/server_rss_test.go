/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS SSE 路由辅助函数单元测试。
*/

package server

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestParseRSSStreamAfter_SinceAndLastEventID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/rss/stream?since=2026-05-20T08:00:00Z", nil)
	c.Request.Header.Set("Last-Event-ID", "2026-05-21T10:00:00.123456789Z")

	got := parseRSSStreamAfter(c)
	want, _ := time.Parse(time.RFC3339Nano, "2026-05-21T10:00:00.123456789Z")
	if !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestParseRSSStreamAfter_DefaultLookback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/rss/stream", nil)

	before := time.Now()
	got := parseRSSStreamAfter(c)
	if got.After(before) {
		t.Fatalf("got future %v", got)
	}
	if before.Sub(got) < 23*time.Hour {
		t.Fatalf("lookback too short: %v", before.Sub(got))
	}
}

func TestExtractRSSEventID(t *testing.T) {
	raw := []byte(`{"type":"rss_new","event_id":"2026-05-21T12:00:00Z","count":1}`)
	if id := extractRSSEventID(raw); id != "2026-05-21T12:00:00Z" {
		t.Fatalf("id=%q", id)
	}
}
