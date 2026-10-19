# Explicit ReAct Runtime

`pkg/ai/runtime/react` 提供项目内显式 ReAct 循环。

它参考了 `temp/dodo-agent_副本/.../SimpleReactAgent.java` 的落地思路，但按当前 Go/eino 项目做了适配：

1. 使用 eino `schema.Message` 和 `schema.ToolInfo`。
2. 不使用 eino ReactAgent 黑盒。
3. 后端自己控制工具执行、错误处理、轮次限制、SSE 事件和审计。
4. 模型只负责输出自然语言或标准 `tool_calls`。
5. 工具执行结果会作为 `schema.Tool` message 写回上下文。

当前公共 Runner 适合 WebSearch、TeacherCopilot、StudentAgent 等后续 assistant 复用。

