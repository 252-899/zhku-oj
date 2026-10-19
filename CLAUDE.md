# ZK Code Arena - AI 工作流系统

## 项目概览

**项目名称**: ZK Code Arena Server (仲恺在线刷题平台)
**技术栈**: Golang 1.24.0 + Gin + MongoDB + go-judge + Eino AI
**当前版本**: v1.2.0
**主要功能**: 在线编程题库、代码评测、AI 题目生成、班级管理、竞赛系统

## 关键规则

### 1. 代码组织

- 遵循 Go 标准项目布局
- 每个文件 200-500 行为宜，最大不超过 800 行
- 按功能模块组织，不按类型组织
- 高内聚、低耦合
- 使用 `pkg/` 存放可复用包，`cmd/` 存放应用入口

### 2. Go 代码风格

- 遵循 Go 官方代码规范和惯用语
- 使用 `gofmt` 和 `golint` 格式化代码
- 错误处理：永远检查 error，不要忽略
- 使用 context.Context 传递请求上下文
- 避免全局变量，使用依赖注入
- 接口定义在使用方，不在实现方

### 3. 测试

- 使用 Go 标准测试框架
- 表驱动测试（Table-Driven Tests）
- 单元测试覆盖率 > 70%
- 集成测试覆盖关键业务流程
- 测试文件命名: `*_test.go`

### 4. 安全

- 不要硬编码敏感信息（API Key、密码等）
- 使用环境变量或配置文件管理敏感数据
- 所有用户输入必须验证
- MongoDB 查询使用参数化，防止注入
- JWT Token 验证和权限控制

## 项目结构

```
zhku-oj/
├── cmd/                    # 应用入口
│   ├── main.go
│   └── runServer.go
├── conf/                   # 配置文件
│   └── config.yaml
├── pkg/
│   ├── ai/                # AI 模块
│   │   ├── agent/         # Agent 系统
│   │   ├── services/      # AI 服务
│   │   │   └── workflow/  # 工作流引擎（旧实现）
│   │   ├── workflow/      # 通用工作流系统 ⭐ 新架构
│   │   │   ├── types.go           # 核心类型定义
│   │   │   ├── registry.go        # 工作流注册中心
│   │   │   ├── engine.go          # 执行引擎
│   │   │   └── definitions/       # 工作流实现（插件）
│   │   └── components/    # AI 组件
│   ├── app/api-server/    # API 服务器
│   │   ├── dto/           # 数据传输对象
│   │   ├── server/        # 路由处理器
│   │   ├── service/       # 业务逻辑
│   │   └── repository/    # 数据访问层
│   ├── common/            # 公共组件
│   │   ├── queue/         # 消息队列
│   │   └── sandbox/       # 代码沙箱客户端
│   ├── models/            # 数据模型
│   └── utils/             # 工具包
├── deploy/                # 部署配置
├── docs/                  # 文档
│   ├── API-AI-WORKFLOW.md              # AI 工作流 API 文档
│   ├── FRONTEND-INTEGRATION-GUIDE.md   # 前端集成指南
│   └── WORKFLOW-DEVELOPMENT-PROGRESS.md # 开发进度
└── scripts/               # 脚本
```

## 核心模块

### 1. AI 工作流系统（最新开发）

**架构**: 插件化三层架构
- **API Layer**: RESTful 接口 + SSE 流式推送
- **Engine Layer**: 工作流执行引擎
- **Definition Layer**: 工作流实现（插件）

**关键文件**:
- `pkg/ai/workflow/types.go` - 核心接口定义
- `pkg/ai/workflow/engine.go` - 执行引擎
- `pkg/ai/workflow/registry.go` - 注册中心
- `pkg/ai/workflow/definitions/oj_problem_gen.go` - OJ 题目生成工作流

**工作流类型**:
- `oj_problem_generation` - OJ 编程题生成 ✅
- `code_review` - 代码审查 📋
- `assignment_grade` - 作业批改 📋
- 其他 10 种类型待实现

### 2. 代码评测系统

- 使用 go-judge 沙箱
- 支持 C/C++/Java/Python/Go
- 消息队列异步处理
- 实时判题结果推送

### 3. 用户权限系统

- JWT 认证
- 角色：学生/老师/管理员
- 中间件：`middleware.JWTMiddleware()`
- AI 功能限制：仅老师/管理员

## 关键模式

### API 响应格式

```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

// 成功响应
utils.SuccessResponse(c, data)

// 错误响应
utils.BadRequestResponse(c, "错误信息")
utils.InternalServerErrorResponse(c, "错误信息")
```

### 错误处理

```go
result, err := someOperation()
if err != nil {
    utils.Logger.Errorf("Operation failed: %v", err)
    utils.InternalServerErrorResponse(c, "操作失败")
    return
}
```

### 数据库操作

```go
// MongoDB 查询
filter := bson.M{"user_id": userID}
err := collection.FindOne(ctx, filter).Decode(&result)
if err != nil {
    if err == mongo.ErrNoDocuments {
        return nil, fmt.Errorf("not found")
    }
    return nil, err
}
```

### Eino Graph 工作流

```go
// 创建 Graph
g := compose.NewGraph[Input, Output](
    compose.WithGenLocalState(stateGen),
)

// 添加节点
g.AddLambdaNode("node_name", lambda,
    compose.WithStatePreHandler(preHandler),
    compose.WithStatePostHandler(postHandler),
)

// 添加边
g.AddEdge(compose.START, "node_name")
g.AddEdge("node_name", compose.END)

// 编译
runnable, err := g.Compile(ctx)
```

## 环境变量

```bash
# MongoDB
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=zhku_oj

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# LLM (DeepSeek)
DEEPSEEK_API_KEY=your_api_key
DEEPSEEK_BASE_URL=https://api.deepseek.com

# 沙箱
SANDBOX_URL=http://localhost:5050

# 应用
APP_HOST=0.0.0.0
APP_PORT=8080
JWT_SECRET=your_secret_key
```

## 可用命令

### 开发命令

```bash
# 开发模式（热更新）
air

# 构建
go build -o bin/server cmd/main.go

# 测试
go test ./...

# 代码检查
golangci-lint run
```

### Claude Code 命令

- `/go-review` - Go 代码审查
- `/go-test` - Go TDD 工作流
- `/go-build` - 修复 Go 构建错误
- `/plan` - 创建实现计划
- `/code-review` - 代码质量审查

## Git 工作流

### 分支策略

- `main` - 生产分支（保护）
- `feature/20051009_test` - 主开发分支
- `test/*` - 测试分支
- `feature/*` - 功能分支
- `fix/*` - 修复分支

### 提交规范

使用 Conventional Commits：

```
feat: 添加 AI 工作流系统
fix: 修复判题队列阻塞问题
refactor: 重构工作流为插件架构
docs: 更新 API 文档
test: 添加工作流引擎测试
```

### 提交流程

1. 创建功能分支
2. 开发并提交（包含 Co-Authored-By）
3. 推送到远程
4. 创建 Pull Request
5. 代码审查
6. 合并到主分支

## 技术决策

### 1. Eino Graph vs 自定义编排

**选择**: Eino Graph
**原因**:
- 声明式工作流定义
- 内置状态管理
- 支持复杂编排（分支、循环、并行）

### 2. 插件化架构

**选择**: Definition 接口 + Registry + Engine
**原因**:
- 易于扩展新工作流类型
- 代码解耦
- 支持动态注册

### 3. SSE vs WebSocket

**选择**: SSE (Server-Sent Events)
**原因**:
- 单向推送足够
- 实现简单
- 自动重连

### 4. MongoDB vs PostgreSQL

**选择**: MongoDB
**原因**:
- 灵活的 Schema（工作流结果动态）
- 文档存储适合复杂对象
- TTL 索引自动清理

## 开发注意事项

### 1. AI 工作流开发

- 新工作流必须实现 `workflow.Definition` 接口
- 在 `definitions/init.go` 中注册
- 提供完整的 `InputSchema`（用于前端表单生成）
- 实现 `ValidateInput` 验证输入参数

### 2. API 开发

- 所有 AI 接口需要 `RequireTeacherOrAdmin()` 中间件
- 使用 `utils.Response` 统一响应格式
- 记录详细日志（`utils.Logger`）
- 处理所有错误情况

### 3. 数据库操作

- 使用 context.Context 传递超时
- 创建适当的索引
- 使用 TTL 索引自动清理过期数据
- 避免 N+1 查询

### 4. 测试

- 单元测试：测试纯函数和业务逻辑
- 集成测试：测试 API 端到端
- Mock 外部依赖（LLM、沙箱）

## 文档资源

### 开发文档

- [AI 工作流开发进度](docs/WORKFLOW-DEVELOPMENT-PROGRESS.md) - 总体进度
- [前端集成指南](docs/FRONTEND-INTEGRATION-GUIDE.md) - 前端开发参考
- [API 文档](docs/API-AI-WORKFLOW.md) - 完整 API 说明
- [架构设计](docs/WORKFLOW-ARCHITECTURE.md) - 系统架构

### 项目记忆

- [项目记忆](.claude/memory/MEMORY.md) - 跨会话上下文
- [开发历史](.claude/memory/development-history.md) - 详细历史

## 当前开发状态

### 已完成 ✅

- AI 工作流系统（插件化架构）
- OJ 题目生成工作流
- 6 个 RESTful API 接口
- SSE 实时进度推送
- MongoDB 数据持久化
- 权限控制
- 完整文档系统

### 进行中 🚧

- 前端集成开发
- 其他工作流类型实现

### 待开发 📋

- 代码审查工作流
- 作业批改工作流
- 工作流模板系统
- 批量生成功能

---

**最后更新**: 2026-03-06
**维护者**: AI Assistant
**版本**: v1.2.0
