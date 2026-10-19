package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zk-code-arena-server/conf"

	"github.com/cloudwego/eino/schema"
	openai "github.com/cloudwego/eino-ext/components/model/openai"
	einomodel "github.com/cloudwego/eino/components/model"
)

type ChatConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

type ModelFactory interface {
	IsChatConfigured() bool
	ChatConfig() ChatConfig
	NewToolCallingChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error)
	// Stream 提供流式对话能力，返回逐 chunk 的消息流（可含 ToolCalls）
	Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error)
	// Call 同步调用 LLM，返回完整响应
	Call(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
}

type ConfigModelFactory struct{}

func NewConfigModelFactory() *ConfigModelFactory {
	return &ConfigModelFactory{}
}

func (f *ConfigModelFactory) ChatConfig() ChatConfig {
	return ChatConfig{
		BaseURL: strings.TrimSpace(conf.Config.AI.Chat.BaseUrl),
		APIKey:  strings.TrimSpace(conf.Config.AI.Chat.ApiKey),
		Model:   strings.TrimSpace(conf.Config.AI.Chat.Model),
	}
}

func (f *ConfigModelFactory) IsChatConfigured() bool {
	cfg := f.ChatConfig()
	return cfg.BaseURL != "" && cfg.APIKey != "" && cfg.Model != ""
}

func (f *ConfigModelFactory) NewToolCallingChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	cfg := f.ChatConfig()
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, fmt.Errorf("chat model config is incomplete")
	}

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Timeout: 90 * time.Second,
	})
}

func (f *ConfigModelFactory) Stream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	cfg := f.ChatConfig()
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, fmt.Errorf("chat model config is incomplete")
	}

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Timeout: 90 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// 绑定工具，返回支持 tool calling 的 ChatModel
	cmWithTools, err := cm.WithTools(tools)
	if err != nil {
		return nil, err
	}

	return cmWithTools.Stream(ctx, messages)
}

func (f *ConfigModelFactory) Call(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	cfg := f.ChatConfig()
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, fmt.Errorf("chat model config is incomplete")
	}

	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		Timeout: 90 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return cm.Generate(ctx, messages)
}
