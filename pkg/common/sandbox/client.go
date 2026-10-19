/*
@Author: omenkk7
@Date: 2025/10/01
@Description: 沙箱客户端核心实现
*/

package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GoJudgeClient go-judge 沙箱客户端实现
type GoJudgeClient struct {
	baseURL    string
	httpClient *http.Client
	languages  map[string]*LanguageConfig
}

// NewGoJudgeClient 创建 go-judge 沙箱客户端
func NewGoJudgeClient(baseURL string, languages map[string]*LanguageConfig) *GoJudgeClient {
	return &GoJudgeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		languages: languages,
	}
}

// CompileCode 编译代码
func (c *GoJudgeClient) CompileCode(ctx context.Context, req *CompileRequest) (*CompileResponse, error) {
	// 获取语言配置
	langConfig, ok := c.languages[req.Language]
	if !ok {
		return nil, fmt.Errorf("不支持的语言: %s", req.Language)
	}

	// 检查是否需要编译
	if langConfig.Compile == nil {
		// 解释型语言，无需编译，直接返回成功
		return &CompileResponse{
			Success:      true,
			ExecutableID: "", // 解释型语言无需编译产物
		}, nil
	}

	// 构建编译请求
	files := map[string]interface{}{
		langConfig.Compile.SourceFile: map[string]interface{}{
			"content": req.SourceCode,
		},
	}

	goJudgeReq := c.buildGoJudgeRequestWithLimits(langConfig.Compile, files, "", langConfig.Compile.TimeLimit, langConfig.Compile.MemoryLimit)

	// 调用 go-judge API
	goJudgeResp, err := c.callGoJudge(ctx, goJudgeReq)
	if err != nil {
		return nil, fmt.Errorf("调用 go-judge 失败: %w", err)
	}

	// 解析编译结果
	resp := &CompileResponse{
		Time:   goJudgeResp.Time,
		Memory: goJudgeResp.Memory,
	}

	// 判断编译是否成功
	if goJudgeResp.Status == "Accepted" && goJudgeResp.ExitStatus == 0 {
		resp.Success = true
		// 获取编译产物文件ID
		if langConfig.Compile.ExecutableFile != "" {
			if fileID, ok := goJudgeResp.FileIDs[langConfig.Compile.ExecutableFile]; ok {
				resp.ExecutableID = fileID
			} else {
				return nil, fmt.Errorf("未找到编译产物文件: %s", langConfig.Compile.ExecutableFile)
			}
		}
	} else {
		resp.Success = false
		// 获取编译错误信息
		if stderr, ok := goJudgeResp.Files["stderr"]; ok {
			resp.CompileError = stderr
		}
		if stdout, ok := goJudgeResp.Files["stdout"]; ok {
			resp.CompileOutput = stdout
		}
		if goJudgeResp.Error != "" {
			resp.CompileError = goJudgeResp.Error
		}
	}

	return resp, nil
}

// RunCode 运行代码
func (c *GoJudgeClient) RunCode(ctx context.Context, req *RunRequest) (*RunResponse, error) {
	// 获取语言配置
	langConfig, ok := c.languages[req.Language]
	if !ok {
		return nil, fmt.Errorf("不支持的语言: %s", req.Language)
	}

	// 构建运行请求
	files := make(map[string]interface{})

	// 如果是编译型语言，使用编译产物
	if langConfig.Compile != nil && req.ExecutableID != "" {
		files[langConfig.Compile.ExecutableFile] = map[string]interface{}{
			"fileId": req.ExecutableID,
		}
	} else if langConfig.Compile == nil {
		// 解释型语言，直接使用源代码
		files[langConfig.Run.SourceFile] = map[string]interface{}{
			"content": req.ExecutableID, // 对于解释型语言，ExecutableID 存储的是源代码
		}
	}

	// 构建 go-judge 请求（使用自定义的时间和内存限制）
	goJudgeReq := c.buildGoJudgeRequestWithLimits(langConfig.Run, files, req.Input, req.TimeLimit, req.MemoryLimit)

	// 调用 go-judge API
	goJudgeResp, err := c.callGoJudge(ctx, goJudgeReq)
	if err != nil {
		return nil, fmt.Errorf("调用 go-judge 失败: %w", err)
	}

	// 解析运行结果
	resp := &RunResponse{
		Time:             goJudgeResp.Time,
		Memory:           goJudgeResp.Memory,
		ExitCode:         goJudgeResp.ExitStatus,
		RawGoJudgeStatus: goJudgeResp.Status,
		GoJudgeAPIError:  goJudgeResp.Error,
	}

	// 获取标准输出和标准错误
	if stdout, ok := goJudgeResp.Files["stdout"]; ok {
		resp.Output = stdout
	}
	if stderr, ok := goJudgeResp.Files["stderr"]; ok {
		resp.Error = stderr
	}

	// 判断运行状态
	resp.Status = c.parseRunStatus(goJudgeResp)

	return resp, nil
}

// GetLanguageConfig 获取语言配置
func (c *GoJudgeClient) GetLanguageConfig(language string) *LanguageConfig {
	return c.languages[language]
}

// Close 关闭客户端
func (c *GoJudgeClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// buildGoJudgeRequest 构建 go-judge 请求（使用配置的限制）
func (c *GoJudgeClient) buildGoJudgeRequest(
	stage *StageConfig,
	files map[string]interface{},
	input string,
) *GoJudgeRequest {
	return c.buildGoJudgeRequestWithLimits(stage, files, input, stage.TimeLimit, stage.MemoryLimit)
}

// buildGoJudgeRequestWithLimits 构建 go-judge 请求（使用自定义限制）
func (c *GoJudgeClient) buildGoJudgeRequestWithLimits(
	stage *StageConfig,
	files map[string]interface{},
	input string,
	timeLimit int64,
	memoryLimit int64,
) *GoJudgeRequest {
	cmd := Command{
		Args:        stage.Args,
		Env:         stage.Env,
		CPULimit:    timeLimit,
		MemoryLimit: memoryLimit,
		ProcLimit:   stage.ProcLimit,
		CopyIn:      files,
		CopyOut:     []string{"stdout", "stderr"},
	}

	// 构建文件描述符
	// stdin: 标准输入（必须有 content 字段，即使为空字符串）
	// stdout: 最大 10KB
	// stderr: 最大 10KB
	stdoutName := "stdout"
	stderrName := "stderr"
	maxSize := 10240
	
	cmd.Files = []FileDescriptor{
		{Content: &input}, // stdin（空字符串也要传 content 字段）
		{Name: &stdoutName, Max: &maxSize}, // stdout
		{Name: &stderrName, Max: &maxSize}, // stderr
	}

	// 如果是编译阶段，需要缓存编译产物
	if stage.ExecutableFile != "" {
		cmd.CopyOutCached = []string{stage.ExecutableFile}
	}

	return &GoJudgeRequest{
		Cmd: []Command{cmd},
	}
}

// callGoJudge 调用 go-judge REST API
func (c *GoJudgeClient) callGoJudge(ctx context.Context, req *GoJudgeRequest) (*GoJudgeResponse, error) {
	// 序列化请求
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/run", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("发送 HTTP 请求失败: %w", err)
	}
	defer httpResp.Body.Close()

	// 检查 HTTP 状态码
	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("go-judge 返回错误状态码: %d, body: %s", httpResp.StatusCode, string(body))
	}

	// 解析响应
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	// go-judge 返回的是数组，取第一个元素
	var respArray []GoJudgeResponse
	if err := json.Unmarshal(respBody, &respArray); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, body: %s", err, string(respBody))
	}

	if len(respArray) == 0 {
		return nil, fmt.Errorf("go-judge 返回空响应")
	}

	return &respArray[0], nil
}

// parseRunStatus 解析运行状态（与 https://docs.goj.ac/api 中 Status 枚举一致）
func (c *GoJudgeClient) parseRunStatus(resp *GoJudgeResponse) RunStatus {
	switch resp.Status {
	case "Accepted":
		return RunStatusAccepted
	case "Time Limit Exceeded":
		return RunStatusTimeLimitExceeded
	case "Memory Limit Exceeded":
		return RunStatusMemoryLimitExceeded
	case "Runtime Error":
		return RunStatusRuntimeError
	case "Nonzero Exit Status", "Signalled":
		// 用户程序异常退出或信号终止，按运行时错误展示，避免一律显示「系统错误」
		return RunStatusRuntimeError
	case "Output Limit Exceeded":
		return RunStatusRuntimeError
	case "File Error", "Internal Error":
		return RunStatusSystemError
	default:
		return RunStatusSystemError
	}
}
