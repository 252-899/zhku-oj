/*
@Author: xiaoTnb
@Date: 2026/05/12
@Description: RSS 正文字段优先级等纯函数的单元测试。
*/

package service

import (
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestPickRSSBody_ContentFirst(t *testing.T) {
	it := &gofeed.Item{
		Content:     "<p>full</p>",
		Description: "<p>short</p>",
	}
	raw, src := pickRSSBody(it)
	if src != "content" || raw != "<p>full</p>" {
		t.Fatalf("got %q %q", src, raw)
	}
}

func TestPickRSSBody_FallbackDescription(t *testing.T) {
	it := &gofeed.Item{Description: "<p>only desc</p>"}
	raw, src := pickRSSBody(it)
	if src != "description" || raw != "<p>only desc</p>" {
		t.Fatalf("got %q %q", src, raw)
	}
}

func TestPickRSSBody_Empty(t *testing.T) {
	it := &gofeed.Item{}
	raw, src := pickRSSBody(it)
	if src != "empty" || raw != "" {
		t.Fatalf("got %q %q", src, raw)
	}
}
