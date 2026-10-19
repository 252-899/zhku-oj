package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	einotool "github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
)

type Provider struct {
	client TavilyClient
	config Config
}

func NewProvider(client TavilyClient, cfg Config) *Provider {
	return &Provider{
		client: client,
		config: cfg,
	}
}

func ConfigFromEnv() Config {
	return Config{
		APIKey:  strings.TrimSpace(os.Getenv("TAVILY_API_KEY")),
		BaseURL: strings.TrimSpace(os.Getenv("TAVILY_BASE_URL")),
	}
}

func (p *Provider) IsConfigured() bool {
	return p != nil && strings.TrimSpace(p.config.APIKey) != ""
}

// Search 直接调用搜索工具（用于显式 ReAct 循环中的工具执行）
func (p *Provider) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("search provider is not initialized")
	}
	return p.client.Search(ctx, query, topK)
}

func (p *Provider) BuildTools(ctx context.Context) ([]einotool.BaseTool, error) {
	if p == nil || p.client == nil {
		return nil, fmt.Errorf("web search provider is not initialized")
	}

	searchTool, err := toolutils.InferTool(
		"web_search",
		"Search the public web for up-to-date information and return structured results with title, url and snippet.",
		func(ctx context.Context, input SearchInput) (string, error) {
			results, err := p.client.Search(ctx, input.Query, input.TopK)
			if err != nil {
				return "", err
			}
			data, err := json.Marshal(results)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	)
	if err != nil {
		return nil, err
	}

	return []einotool.BaseTool{searchTool}, nil
}
