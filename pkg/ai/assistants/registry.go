package assistants

import (
	"strings"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	assistants map[string]Assistant
}

func NewRegistry() *Registry {
	return &Registry{assistants: make(map[string]Assistant)}
}

func (r *Registry) Register(assistant Assistant) {
	if r == nil || assistant == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assistants[NormalizeType(assistant.Type())] = assistant
}

func (r *Registry) Get(assistantType string) (Assistant, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	assistant, ok := r.assistants[NormalizeType(assistantType)]
	return assistant, ok
}

func NormalizeType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "websearch", "websearchagent", "web_search":
		return "websearch"
	default:
		return value
	}
}
