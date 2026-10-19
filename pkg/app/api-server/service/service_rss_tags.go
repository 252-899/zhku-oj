/*
@Author: xiaoTnb
@Date: 2026/05/21
@Description: RSS 资讯标签抽取与稀疏向量（供规则匹配与向量相似度召回）。
*/

package service

import (
	"math"
	"strings"
	"unicode"

	"zk-code-arena-server/pkg/models"
)

// rssTagKeywords 规则标签词典：命中关键词则打上归一化标签（阶段 1+2）。
var rssTagKeywords = []struct {
	tag      string
	keywords []string
}{
	{tag: "algorithm", keywords: []string{"算法", "algorithm", "leetcode", "竞赛", "dp", "动态规划", "贪心", "dfs", "bfs"}},
	{tag: "data-structure", keywords: []string{"数据结构", "链表", "树", "图", "哈希", "堆", "栈", "队列"}},
	{tag: "programming", keywords: []string{"编程", "代码", "programming", "coding", "开发", "工程师"}},
	{tag: "go", keywords: []string{"golang", " go ", "go语言"}},
	{tag: "python", keywords: []string{"python", "django", "flask"}},
	{tag: "java", keywords: []string{"java", "spring", "jvm"}},
	{tag: "cpp", keywords: []string{"c++", "cpp"}},
	{tag: "ai", keywords: []string{"人工智能", "机器学习", "deep learning", "llm", "大模型", "chatgpt", "ai"}},
	{tag: "security", keywords: []string{"安全", "漏洞", "渗透", "security"}},
	{tag: "database", keywords: []string{"数据库", "mysql", "redis", "mongodb", "sql"}},
	{tag: "web", keywords: []string{"前端", "后端", "web", "http", "react", "vue", "gin"}},
	{tag: "os", keywords: []string{"操作系统", "linux", "内核", "docker", "kubernetes", "k8s"}},
	{tag: "career", keywords: []string{"面试", "求职", "薪资", "职场", "校招"}},
	{tag: "news", keywords: []string{"发布", "更新", "资讯", "新闻", "announcement"}},
}

// ExtractRSSTags 从标题、正文摘要、源名称抽取标签。
func ExtractRSSTags(title, bodyHTML, sourceName string) []string {
	text := strings.ToLower(title + " " + RSSPlainSummary(bodyHTML, 400) + " " + sourceName)
	seen := make(map[string]struct{})
	var tags []string
	for _, rule := range rssTagKeywords {
		for _, kw := range rule.keywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				if _, ok := seen[rule.tag]; !ok {
					seen[rule.tag] = struct{}{}
					tags = append(tags, rule.tag)
				}
				break
			}
		}
	}
	if len(tags) == 0 {
		tags = []string{"general"}
	}
	return tags
}

// tokenizeRSSText 简单分词（英文按词、中文按连续汉字块）。
func tokenizeRSSText(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		tokens = append(tokens, buf.String())
		buf.Reset()
	}
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			buf.WriteRune(r)
			continue
		}
		if unicode.Is(unicode.Han, r) {
			flush()
			tokens = append(tokens, string(r))
			continue
		}
		flush()
	}
	flush()
	return tokens
}

// BuildRSSTermVector 构建稀疏 TF 向量（阶段 3 向量召回，无需外部 embedding API）。
func BuildRSSTermVector(title, bodyHTML string, tags []string) map[string]float64 {
	text := title + " " + RSSPlainSummary(bodyHTML, 300)
	tokens := tokenizeRSSText(text)
	vec := make(map[string]float64)
	for _, t := range tokens {
		runes := []rune(t)
		if len(runes) < 2 && (len(runes) == 0 || !unicode.Is(unicode.Han, runes[0])) {
			continue
		}
		vec[t]++
	}
	for _, tag := range tags {
		vec["tag:"+tag] += 2.0
	}
	return normalizeTermVector(vec)
}

// EnsureItemEnrichment 为缺少标签/向量的旧数据补全（读时修复）。
func EnsureItemEnrichment(item *models.RSSItem) {
	if item == nil {
		return
	}
	if len(item.Tags) == 0 {
		item.Tags = ExtractRSSTags(item.Title, item.BodyHTML, item.SourceName)
	}
	if len(item.TermVector) == 0 {
		item.TermVector = BuildRSSTermVector(item.Title, item.BodyHTML, item.Tags)
	}
}

func normalizeTermVector(vec map[string]float64) map[string]float64 {
	if len(vec) == 0 {
		return vec
	}
	var sum float64
	for _, v := range vec {
		sum += v * v
	}
	if sum == 0 {
		return vec
	}
	norm := math.Sqrt(sum)
	out := make(map[string]float64, len(vec))
	for k, v := range vec {
		out[k] = v / norm
	}
	return out
}

// cosineSimilarity 两稀疏向量余弦相似度。
func cosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dot float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	return dot
}

// tagMatchScore 用户兴趣标签与条目标签的匹配度 [0,1]。
func tagMatchScore(userTags map[string]float64, itemTags []string) float64 {
	if len(userTags) == 0 || len(itemTags) == 0 {
		return 0
	}
	var hit float64
	for _, t := range itemTags {
		if w, ok := userTags[t]; ok {
			hit += w
		}
	}
	if hit == 0 {
		return 0
	}
	if hit > 1 {
		hit = 1
	}
	return hit
}
