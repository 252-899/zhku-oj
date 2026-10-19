package doc

// AIAgentChatRequestDoc AI Agent 同步对话请求文档。
type AIAgentChatRequestDoc struct {
	SessionID string                 `json:"session_id,omitempty" example:""`
	RoundID   string                 `json:"round_id,omitempty" example:""`
	Message   string                 `json:"message" binding:"required" example:"请帮我搜索 Go 语言泛型的用法"`
	AgentType string                 `json:"agent_type" binding:"required" example:"websearchagent"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" swaggertype:"object"`
}

// AIAgentStreamChatRequestDoc AI Agent 流式对话请求文档。
type AIAgentStreamChatRequestDoc struct {
	SessionID  string                         `json:"session_id,omitempty" example:""`
	RoundID    string                         `json:"round_id,omitempty" example:""`
	Message    string                         `json:"message" binding:"required" example:"生成 2 道数组相关 OJ 编程题"`
	TaskKind   string                         `json:"task_kind,omitempty" example:"module" enums:"assistant,module"`
	AgentType  string                         `json:"agent_type,omitempty" example:"websearchagent"`
	ModuleType string                         `json:"module_type,omitempty" example:"oj_question_gen"`
	Metadata   *AIAgentStreamMetadataInputDoc `json:"metadata,omitempty"`
}

// AIAgentStreamMetadataInputDoc AI Agent 流式对话 metadata 文档。
type AIAgentStreamMetadataInputDoc struct {
	Input *OJQuestionGenInputDoc `json:"input,omitempty"`
}

// OJQuestionGenInputDoc oj_question_gen 模块的 metadata.input 请求文档。
type OJQuestionGenInputDoc struct {
	KnowledgePoints        []string `json:"knowledge_points" binding:"required" example:"数组,循环"`
	Difficulty             string   `json:"difficulty,omitempty" example:"easy" enums:"easy,medium,hard"`
	Count                  int      `json:"count,omitempty" example:"2" minimum:"1" maximum:"10"`
	TestcaseCount          int      `json:"testcase_count,omitempty" example:"3" minimum:"1" maximum:"10"`
	StandardLanguage       string   `json:"standard_language,omitempty" example:"python"`
	AdditionalRequirements string   `json:"additional_requirements,omitempty" example:"题目场景贴近校园生活"`
}

// OJQuestionGenStreamRequestDoc OJ 题目生成流式业务接口请求文档。
type OJQuestionGenStreamRequestDoc struct {
	SessionID              string   `json:"session_id,omitempty" example:""`
	RoundID                string   `json:"round_id,omitempty" example:""`
	Message                string   `json:"message,omitempty" example:"生成 2 道数组相关 OJ 编程题"`
	KnowledgePoints        []string `json:"knowledge_points" binding:"required" example:"数组,循环"`
	Difficulty             string   `json:"difficulty,omitempty" example:"easy" enums:"easy,medium,hard"`
	Count                  int      `json:"count,omitempty" example:"2" minimum:"1" maximum:"10"`
	TestcaseCount          int      `json:"testcase_count,omitempty" example:"3" minimum:"1" maximum:"10"`
	StandardLanguage       string   `json:"standard_language,omitempty" example:"python"`
	AdditionalRequirements string   `json:"additional_requirements,omitempty" example:"题目场景贴近校园生活"`
}
