# AI Tools

`pkg/ai/tools` 放 AI Agent 可调用的工具能力。

当前约定：

1. `pkg/ai/tools` 保留通用工具协议和辅助函数。
2. `pkg/ai/tools/oj` 放 OJ 平台业务工具，例如查询用户、班级、任务、题目。
3. 业务工具只依赖窄接口和 `pkg/models`，不直接依赖 `pkg/app/api-server/service`，避免形成 service 和 ai 包之间的循环引用。
4. 第一批工具只做只读查询，不做新增、修改、删除等写操作。

后续 Agent 接入时，由业务入口创建工具 Provider，并把当前登录用户 ID 固定注入 Provider。模型不能自己传 operator user id，避免通过工具参数伪造身份。
