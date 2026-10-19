package modules

import (
	"strings"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	modules map[string]Module
}

func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]Module)}
}

func (r *Registry) Register(module Module) {
	if r == nil || module == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.modules[NormalizeType(module.Type())] = module
}

func (r *Registry) Get(moduleType string) (Module, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	module, ok := r.modules[NormalizeType(moduleType)]
	return module, ok
}

func NormalizeType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "questiongen", "question_generation", "question_generation_module":
		return "questiongen"
	case "interactivelearning", "interactive_learning":
		return "interactivelearning"
	default:
		return value
	}
}
