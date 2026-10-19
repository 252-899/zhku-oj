package react

// DefaultSystemPrompt 是共享的 ReAct 工具调用约束，要求模型输出真实
// tool_calls，而不是伪 JSON 或文本形式的工具调用标记。
const DefaultSystemPrompt = `## 工具调用规则
1. 需要调用工具时，必须使用标准 tool_calls 字段。
2. 调用工具时，content 字段不要输出 JSON、函数名、参数、思考过程或工具调用文本。
3. 工具参数必须是有效 JSON，只包含最小必要参数。
4. 如果上下文已经足够回答问题，不要继续调用工具。
5. 没有工具调用的一轮输出会被视为最终自然语言答案。
6. 不要重复调用同名且同参数的工具，除非上一次工具调用失败。`

// DefaultForceFinalPrompt 是达到最大轮次后注入的兜底提示词。
const DefaultForceFinalPrompt = `你已达到最大推理轮次限制。
请基于当前已有上下文直接给出最终答案。
禁止再调用任何工具。
如果信息不完整，请合理说明缺失信息。`
