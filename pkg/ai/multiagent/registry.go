// registry.go 将角色元数据和 runner 工厂从 Orchestrator 中拆出来，
// 让不同业务域可以注册自己的 Agent 团队。
package multiagent

import (
	"fmt"
	"strings"
	"sync"
)

type RoleRegistry struct {
	mu        sync.RWMutex
	roles     map[string]RoleSpec
	factories map[string]AgentFactory
}

// NewRoleRegistry 创建空角色注册表。教师侧通常会在这里注册
// class_analyst、assignment_reviewer、report_writer 等角色。
func NewRoleRegistry() *RoleRegistry {
	return &RoleRegistry{
		roles:     map[string]RoleSpec{},
		factories: map[string]AgentFactory{},
	}
}

// Register 将标准化后的角色名绑定到 AgentFactory。Lead 只能规划这里已注册的角色。
func (r *RoleRegistry) Register(role RoleSpec, factory AgentFactory) error {
	if r == nil {
		return fmt.Errorf("role registry is nil")
	}
	name := normalizeRole(role.Name)
	if name == "" {
		return fmt.Errorf("role name is empty")
	}
	if factory == nil {
		return fmt.Errorf("agent factory for role %q is nil", name)
	}

	role.Name = name
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roles[name] = role
	r.factories[name] = factory
	return nil
}

// Get 将角色名解析成角色元数据和工厂。若 Lead 请求了当前业务入口未启用的角色，
// 返回值中的 ok 会为 false。
func (r *RoleRegistry) Get(name string) (RoleSpec, AgentFactory, bool) {
	if r == nil {
		return RoleSpec{}, nil, false
	}
	name = normalizeRole(name)
	r.mu.RLock()
	defer r.mu.RUnlock()
	role, ok := r.roles[name]
	if !ok {
		return RoleSpec{}, nil, false
	}
	return role, r.factories[name], true
}

// Roles 返回当前可展示给 Lead 的角色目录。
func (r *RoleRegistry) Roles() []RoleSpec {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	roles := make([]RoleSpec, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	return roles
}

// normalizeRole 统一角色名格式，避免提示词、配置和 Go 代码中的大小写差异。
func normalizeRole(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
