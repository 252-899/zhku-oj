# ZK Code Arena Server

一个基于 Golang + Gin + MongoDB + go-judge 构建的现代化在线刷题平台，为仲恺农业工程学院学生提供专业的编程练习环境。

## ✨ 功能特性

### 用户管理模块
- 用户注册/登录
- 个人信息修改
- 用户角色管理（系统管理员、老师、学生、竞赛发起者等）

### 📚 题库管理
- **题目列表**: 支持分页、搜索、筛选的题目展示
- **难度分类**: 简单、中等、困难三个难度等级
- **分类浏览**: 按算法类型（数组、链表、树、图等）分类
- **状态跟踪**: 已解决、未解决、已尝试状态管理
- **统计展示**: 题目数量统计和通过率展示

### 🏫 班级管理
- 班级创建和管理
- 学生加入/退出班级
- 班级课程管理
- 加入申请审核

### 📖 课程管理
- 课程创建和编辑
- 课程成员管理
- 课程作业分配

### 📝 作业系统
- 作业创建和发布
- 学生作业提交
- 自动评测和手动批改
- 成绩统计和导出

### 🏆 竞赛系统
- 竞赛创建和管理
- 参赛者管理
- 实时排名
- 成绩导出

### ⚡ 代码评测
- 支持多种编程语言（C/C++、Java、Python等）
- 基于 go-judge 的安全评测
- 实时评测结果反馈

## 🚀 快速开始

### 环境要求
- Go 1.23+
- MongoDB 4.4+
- Docker & Docker Compose

### 使用 Docker 部署

1. 克隆项目
```bash
git clone <repository-url>
cd ZKCodeArenaServer
```

2. 启动服务
```bash
docker-compose up -d
```

3. 访问服务
- API 服务: http://localhost:8080
- MongoDB: localhost:27017

### 开发模式（热更新）

1. 安装 Air
```bash
# 使用兼容 Go 1.23 的版本
go install github.com/air-verse/air@v1.49.0
```

2. 启动开发服务器
```bash
air
```

## 📁 项目结构

```
ZKCodeArenaServer/
├── cmd/                    # 应用入口
│   ├── main.go            # 主程序入口
│   └── runServer.go       # 服务器启动
├── conf/                  # 配置文件
│   ├── config.yaml        # 应用配置
│   └── config.go          # 配置管理
├── pkg/                   # 核心包
│   ├── app/              # 应用层
│   │   └── api-server/   # API 服务器
│   │       ├── dto/      # 数据传输对象
│   │       ├── server/   # 路由处理器
│   │       └── service/  # 业务逻辑
│   ├── dao/              # 数据访问层
│   ├── models/           # 数据模型
│   └── utils/            # 工具包
├── Dockerfile            # Docker 镜像构建
├── docker-compose.yml    # Docker Compose 配置
├── .air.toml            # Air 热更新配置
└── go.mod               # Go 模块依赖
```

## 🔧 配置说明

### 环境变量
- `MONGO_URI`: MongoDB 连接字符串
- `JWT_SECRET`: JWT 密钥
- `APP_PORT`: 应用端口（默认 8080）

### 配置文件
主要配置在 `conf/config.yaml` 中：
- 应用配置（端口、主机等）
- MongoDB 配置
- 评测系统配置
- 日志配置

## 📝 API 文档

### 用户管理
- `POST /api/v1/user/` - 用户注册
- `POST /api/v1/user/login` - 用户登录
- `GET /api/v1/user/` - 获取用户列表
- `GET /api/v1/user/:id` - 获取用户详情

### 题目管理
- `GET /api/v1/problem/` - 获取题目列表
- `GET /api/v1/problem/:id` - 获取题目详情
- `POST /api/v1/problem/` - 创建题目
- `PUT /api/v1/problem/:id` - 更新题目

### 提交管理
- `POST /api/v1/submit/` - 提交代码
- `GET /api/v1/submit/:id` - 获取提交结果

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 👥 作者

- **urmsone** - *初始工作* - [urmsone@163.com](mailto:urmsone@163.com)

## 🙏 致谢

- 感谢仲恺农业工程学院提供的支持
- 感谢所有贡献者的努力
