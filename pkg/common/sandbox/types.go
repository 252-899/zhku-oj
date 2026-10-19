/*
@Author: omenkk7
@Date: 2025/10/01
@Description: 描述
*/

package sandbox

import "context"

// ============================================================================
// 沙箱客户端接口
// ============================================================================

// Client 沙箱客户端接口
type Client interface {
	// CompileCode 编译代码
	CompileCode(ctx context.Context, req *CompileRequest) (*CompileResponse, error)

	// RunCode 运行代码
	RunCode(ctx context.Context, req *RunRequest) (*RunResponse, error)

	// GetLanguageConfig 获取语言配置
	GetLanguageConfig(language string) *LanguageConfig

	// Close 关闭客户端
	Close() error
}

// ============================================================================
// 编译相关数据结构
// ============================================================================

// CompileRequest 编译请求
type CompileRequest struct {
	Language   string // 语言标识：java, cpp, python, go, c
	SourceCode string // 源代码
}

// CompileResponse 编译响应
type CompileResponse struct {
	Success       bool   // 是否成功
	ExecutableID  string // 编译文件ID（用于后续运行）
	CompileOutput string // 编译输出（stdout）
	CompileError  string // 编译错误（stderr）
	Time          int64  // 编译时间（纳秒）
	Memory        int64  // 内存使用（字节）
}

// ============================================================================
// 运行相关数据结构
// ============================================================================

// RunRequest 运行请求
type RunRequest struct {
	Language     string // 语言标识
	ExecutableID string // 编译产物文件ID（编译型语言）或源代码（解释型语言）
	Input        string // 标准输入
	TimeLimit    int64  // 时间限制（纳秒）
	MemoryLimit  int64  // 内存限制（字节）
}

// RunResponse 运行响应
type RunResponse struct {
	Status   RunStatus // 运行状态
	Output   string    // 标准输出
	Error    string    // 标准错误（stderr）
	Time     int64     // 运行时间（纳秒）
	Memory   int64     // 内存使用（字节）
	ExitCode int       // 退出码
	// RawGoJudgeStatus go-judge 返回的原始 status，用于日志排障（如 Nonzero Exit Status / Signalled）
	RawGoJudgeStatus string
	// GoJudgeAPIError go-judge 响应 JSON 顶层的 error 字段（与 stderr 不同）
	GoJudgeAPIError string
}

// RunStatus 运行状态
type RunStatus string

const (
	RunStatusAccepted            RunStatus = "Accepted"              // 正常运行
	RunStatusTimeLimitExceeded   RunStatus = "Time Limit Exceeded"   // 超时
	RunStatusMemoryLimitExceeded RunStatus = "Memory Limit Exceeded" // 内存超限
	RunStatusRuntimeError        RunStatus = "Runtime Error"         // 运行时错误
	RunStatusSystemError         RunStatus = "System Error"          // 系统错误
)

// ============================================================================
// 语言配置数据结构
// ============================================================================

// LanguageConfig 语言配置
type LanguageConfig struct {
	Name    string       `yaml:"name"`              // 语言名称
	Compile *StageConfig `yaml:"compile,omitempty"` // 编译配置（可选，解释型语言无需编译）
	Run     *StageConfig `yaml:"run"`               // 运行配置
}

// StageConfig 阶段配置（编译或运行）
type StageConfig struct {
	Args           []string `yaml:"args"`                      // 执行命令参数
	Env            []string `yaml:"env"`                       // 环境变量
	TimeLimit      int64    `yaml:"time_limit"`                // 时间限制（纳秒）
	MemoryLimit    int64    `yaml:"memory_limit"`              // 内存限制（字节）
	ProcLimit      int      `yaml:"proc_limit"`                // 进程数限制
	SourceFile     string   `yaml:"source_file,omitempty"`     // 源文件名
	ExecutableFile string   `yaml:"executable_file,omitempty"` // 可执行文件名
}

// ============================================================================
// go-judge REST API 数据结构
// ============================================================================

// GoJudgeRequest go-judge 请求结构
type GoJudgeRequest struct {
	Cmd []Command `json:"cmd"`
}

// Command go-judge 命令结构
type Command struct {
	Args          []string               `json:"args"`
	Env           []string               `json:"env"`
	Files         []FileDescriptor       `json:"files"`
	CPULimit      int64                  `json:"cpuLimit"`      // 纳秒
	MemoryLimit   int64                  `json:"memoryLimit"`   // 字节
	ProcLimit     int                    `json:"procLimit"`     // 进程数
	CopyIn        map[string]interface{} `json:"copyIn"`        // 输入文件
	CopyOut       []string               `json:"copyOut"`       // 输出文件
	CopyOutCached []string               `json:"copyOutCached"` // 缓存的输出文件
}

// FileDescriptor 文件描述符
type FileDescriptor struct {
	Content *string `json:"content,omitempty"` // 文件内容（指针类型，nil 时省略）
	Name    *string `json:"name,omitempty"`    // 文件名（指针类型，nil 时省略）
	Max     *int    `json:"max,omitempty"`     // 最大大小（字节，指针类型，nil 时省略）
}

// GoJudgeResponse go-judge 响应结构
type GoJudgeResponse struct {
	Status     string            `json:"status"`     // 状态：Accepted, Time Limit Exceeded, Memory Limit Exceeded, Runtime Error
	ExitStatus int               `json:"exitStatus"` // 退出状态码
	Time       int64             `json:"time"`       // 运行时间（纳秒）
	Memory     int64             `json:"memory"`     // 内存使用（字节）
	RunTime    int64             `json:"runTime"`    // 实际运行时间（纳秒）
	Files      map[string]string `json:"files"`      // 输出文件内容
	FileIDs    map[string]string `json:"fileIds"`    // 缓存文件ID
	FileError  []FileError       `json:"fileError"`  // 文件错误
	Error      string            `json:"error"`      // 错误信息
}

// FileError 文件错误
type FileError struct {
	Name    string `json:"name"`    // 文件名
	Type    string `json:"type"`    // 错误类型
	Message string `json:"message"` // 错误信息
}
