package websearch

import "strings"

const CanonicalAssistantType = "websearch"

func NormalizeType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "websearch", "websearchagent", "web_search":
		return CanonicalAssistantType
	default:
		return value
	}
}
