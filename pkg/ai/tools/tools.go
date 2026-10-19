// 包 tools 提供 AI 工具 Provider 共享的辅助函数。
package tools

import (
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ToSchemaToolInfos 将 Eino BaseTool 转为可传给 tool-calling 模型的
// schema.ToolInfo。
func ToSchemaToolInfos(items []einotool.BaseTool) []*schema.ToolInfo {
	result := make([]*schema.ToolInfo, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if defined, ok := item.(interface{ GetToolDefinition() *schema.ToolInfo }); ok {
			result = append(result, defined.GetToolDefinition())
		}
	}
	return result
}
