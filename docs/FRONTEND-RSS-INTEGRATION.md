# RSS 资讯 · 前端对接与 Apifox 测试说明

> 版本：与当前后端实现一致（含 SSE 推送、已读未读、个性化推荐）  
> 基础路径：`{{baseUrl}}` 默认 `http://127.0.0.1:8080/api/v1`  
> 认证：除 SSE 外，用户/管理接口均使用 Header `Authorization: Bearer {{token}}`

---

## 1. 统一响应格式

所有 **JSON** 接口（非 SSE）返回：

```json
{
  "code": 200,
  "message": "success",
  "data": { }
}
```

| HTTP 状态 | code 字段 | 说明 |
|-----------|-----------|------|
| 200 | 200 | 成功 |
| 400 | 400 | 参数错误 |
| 401 | 401 | 未登录 / Token 无效 |
| 403 | 403 | 无权限（管理端需教师/管理员） |
| 404 | 404 | 资源不存在 |
| 503 | 503 | SSE 连接数已满 |

错误时 `data` 通常省略，以 `message` 为准。

---

## 2. 端到端链路（前端视角）

```mermaid
sequenceDiagram
  participant FE as 前端
  participant API as REST API
  participant SSE as SSE /rss/stream
  participant DB as MongoDB

  Note over FE,DB: 初始化
  FE->>API: POST /user/login
  API-->>FE: token
  FE->>API: GET /rss/unread-count
  FE->>SSE: EventSource ?token=&since=

  Note over FE,DB: 管理端（教师）准备数据
  FE->>API: POST /rss/admin/sources
  FE->>API: POST /rss/admin/sources/:id/refresh

  Note over FE,DB: 列表与推荐
  FE->>API: GET /rss/feed/recommended
  FE->>API: GET /rss/items

  Note over FE,DB: 实时推送
  SSE-->>FE: event snapshot
  SSE-->>FE: event rss_new
  FE->>API: GET /rss/items/latest?since=

  Note over FE,DB: 阅读与画像
  FE->>API: GET /rss/items/:id
  FE->>API: POST /rss/read
  FE->>API: POST /rss/events
```

### 2.1 推荐页面分工

| 场景 | 接口 | 说明 |
|------|------|------|
| 个性化首页 | `GET /rss/feed/recommended` | 带 `score`、`reason`（含「AI 精选」） |
| 时间线 | `GET /rss/items` | 按发布时间，非个性化 |
| 红点 | `GET /rss/unread-count` | 进入模块 / SSE 新条后刷新 |
| 实时角标 | `GET /rss/stream` + `rss_new` | 收到后 `unread-count++` 或拉 `latest` |
| 断线补全 | `since` + `Last-Event-ID` | 见 SSE 章节 |

### 2.2 本地游标（建议）

```text
rss_cursor  = 最近一次见到的最大 fetched_at（RFC3339Nano）
             来源：SSE data.cursor / event_id、latest 接口、mark_all 后 unread 的 last_read_at
```

---

## 3. 认证

### 3.1 登录（获取 Token）

`POST /user/login`（无需 Token）

**请求体：**

```json
{
  "student_id": "20210001",
  "password": "your_password"
}
```

**成功 `data`：**

| 字段 | 类型 | 说明 |
|------|------|------|
| token | string | JWT，后续 Bearer |
| user | object | 用户资料 |
| message | string | 提示文案 |

### 3.2 Header 约定

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Content-Type: application/json
```

SSE 无法自定义 Header 时使用 Query：

```http
GET /api/v1/rss/stream?token={{token}}&since=2026-05-21T00:00:00Z
```

---

## 4. 用户端接口

### 4.1 未读数量

`GET /rss/unread-count`

**响应 `data`：**

| 字段 | 类型 | 说明 |
|------|------|------|
| unread_count | int64 | 未读条数 |
| last_read_at | string | 全局已读游标，RFC3339Nano；从未全部已读时可能为空字符串 |

---

### 4.2 资讯列表（时间线）

`GET /rss/items?page=1&page_size=20&source_id={可选}`

| Query | 类型 | 默认 | 说明 |
|-------|------|------|------|
| page | int | 1 | 页码 |
| page_size | int | 20 | 每页条数，最大 100 |
| source_id | string | - | 订阅源 ObjectId，过滤单源 |

**响应 `data`：**

| 字段 | 类型 | 说明 |
|------|------|------|
| items | RSSItemListRow[] | 列表 |
| total | int64 | 总数 |
| page | int64 | 当前页 |
| page_size | int64 | 每页大小 |
| total_pages | int64 | 总页数 |

**RSSItemListRow：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 资讯 ObjectId |
| source_id | string | 订阅源 ID |
| source_name | string | 源名称 |
| title | string | 标题 |
| link | string | 原文链接 |
| summary | string | 纯文本摘要（约 200 字） |
| published_at | string (ISO8601) | RSS 发布时间 |
| fetched_at | string (ISO8601) | 入库时间，**作游标** |
| is_read | bool | 当前用户是否已读 |

---

### 4.3 个性化推荐

`GET /rss/feed/recommended?page=1&page_size=20`

**响应 `data`：** 与列表相同分页字段，`items` 为 **RSSFeedRecommendRow**：

继承 RSSItemListRow，额外：

| 字段 | 类型 | 说明 |
|------|------|------|
| score | float | 推荐分，越高越靠前 |
| reason | string | 展示理由，见下表 |

**reason 枚举（展示用）：**

| reason | 含义 |
|--------|------|
| AI 精选 | LLM 重排后的头部结果（需配置 AI Chat） |
| 与您的学习兴趣相关 | 标签匹配高 |
| 常读来源 | 常打开该订阅源 |
| 内容相似推荐 | 向量相似度高 |
| 平台热门 | 全站打开/点击多 |
| 已读降权 | 已读但仍可能出现在后排 |
| 最新资讯 | 默认文案 |

---

### 4.4 新条轮询（SSE 兜底）

`GET /rss/items/latest?since={RFC3339}&limit=50`

| Query | 必填 | 说明 |
|-------|------|------|
| since | 是 | `fetched_at` 严格大于该时间的条目 |
| limit | 否 | 默认 50，最大 100 |

**响应 `data`：**

```json
{ "items": [ /* RSSItemListRow[] */ ] }
```

---

### 4.5 资讯详情

`GET /rss/items/:id`

**响应 `data`（RSSItemDetail）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 资讯 ID |
| source_id | string | 源 ID |
| source_name | string | 源名 |
| title | string | 标题 |
| link | string | 外链 |
| body_html | string | 消毒后 HTML 正文 |
| body_source | string | `content` \| `description` \| `empty` |
| published_at | string | 发布时间 |
| fetched_at | string | 入库时间 |
| tags | string[] | 规则标签，如 `go`、`algorithm` |
| open_count | int | 全站打开次数 |
| click_count | int | 全站点击次数 |
| is_read | bool | 是否已读 |

**副作用：** 成功返回后服务端自动记 `open` 行为并标该条已读（与 `POST /events` open 等效）。

---

### 4.6 标记已读

`POST /rss/read`

**请求体（二选一）：**

```json
{ "item_id": "674a1b2c3d4e5f6789012345" }
```

```json
{ "mark_all": true }
```

| 字段 | 说明 |
|------|------|
| item_id | 单条已读 |
| mark_all | 全部已读（更新全局游标为当前时间） |

**响应 `data`：**

```json
{
  "message": "已更新阅读状态",
  "unread_count": 0
}
```

---

### 4.7 行为上报（推荐画像）

`POST /rss/events`

**请求体：**

```json
{
  "item_id": "674a1b2c3d4e5f6789012345",
  "action": "dwell",
  "dwell_ms": 12000
}
```

| action | 说明 | dwell_ms |
|--------|------|----------|
| open | 打开 | 可选 |
| click | 点击外链 | 可选 |
| dwell | 停留 | 建议 ≥3000 才有效 |
| not_interested | 不感兴趣，推荐拉黑 | - |

**响应 `data`：** `{ "message": "已记录" }`

> `open` 会同时增加条目 `open_count` 并标已读。

---

## 5. SSE 推送 `/rss/stream`

### 5.1 连接

```http
GET /api/v1/rss/stream?token={{token}}&since={{rss_cursor}}
Accept: text/event-stream
```

可选 Header（浏览器重连）：

```http
Last-Event-ID: 2026-05-21T12:00:00.123456789Z
```

**since 与 Last-Event-ID：** 取两者中**较晚**的时间；都缺省时快照回溯 **24 小时**。

### 5.2 事件类型

| event 名 | 时机 | data JSON `type` |
|----------|------|------------------|
| snapshot | 连接后立即 | `snapshot` |
| rss_new | 后台同步到新条 | `rss_new` |
| backlog | 客户端消费太慢 | `backlog` |
| （注释行） | `: ping` 每 30s | - |

每条业务帧格式：

```text
id: 2026-05-21T12:00:00.123456789Z
event: rss_new
data: {"type":"rss_new","event_id":"...","ts":1716280000000,"cursor":"...","count":2,"items":[...],"ids":["..."]}

```

### 5.3 RSSSSEEvent（data 解析）

| 字段 | 类型 | 说明 |
|------|------|------|
| type | string | `snapshot` \| `rss_new` \| `backlog` |
| event_id | string | 单调游标，建议写入 Last-Event-ID |
| ts | int64 | 毫秒时间戳 |
| cursor | string | 本批最大 `fetched_at`，RFC3339Nano |
| count | int | 本批新条总数 |
| items | RSSItemListRow[] | 预览，最多 20 条 |
| ids | string[] | 全部新条 ID（可能多于 items） |

**backlog：** `count=0`，无 items，前端应 `GET /items/latest?since=cursor` 全量补拉。

### 5.4 前端 EventSource 示例

```javascript
const since = localStorage.getItem('rss_cursor') || '';
const url = `/api/v1/rss/stream?token=${encodeURIComponent(token)}&since=${encodeURIComponent(since)}`;
const es = new EventSource(url);

es.addEventListener('snapshot', (e) => {
  const data = JSON.parse(e.data);
  mergeItems(data.items);
  if (data.cursor) localStorage.setItem('rss_cursor', data.cursor);
});

es.addEventListener('rss_new', (e) => {
  const data = JSON.parse(e.data);
  prependItems(data.items);
  bumpUnreadBadge(data.count);
  if (data.cursor) localStorage.setItem('rss_cursor', data.cursor);
});

es.addEventListener('backlog', (e) => {
  const { cursor } = JSON.parse(e.data);
  fetchLatestSince(cursor);
});
```

---

## 6. 管理端接口（教师 / 管理员）

均需：`Authorization` + 角色 `teacher` 或 `admin`。

### 6.1 列出订阅源

`GET /rss/admin/sources`

**`data.sources[]`（RSSSource）：**

| 字段 | 说明 |
|------|------|
| id | 源 ID |
| name | 显示名 |
| feed_url | RSS 地址 |
| enabled | 是否启用定时拉取 |
| last_fetch_at | 最近拉取时间 |
| last_success_at | 最近成功时间 |
| last_error | 最近错误信息 |

### 6.2 创建订阅源

`POST /rss/admin/sources`

```json
{
  "name": "阮一峰的网络日志",
  "feed_url": "https://www.ruanyifeng.com/blog/atom.xml",
  "enabled": true
}
```

### 6.3 更新订阅源

`PUT /rss/admin/sources/:id`

```json
{
  "name": "新名称",
  "enabled": false
}
```

### 6.4 删除订阅源

`DELETE /rss/admin/sources/:id`  
同时删除该源下所有资讯条目。

### 6.5 手动刷新

`POST /rss/admin/sources/:id/refresh`  
`POST /rss/admin/refresh-all`

同一源 / 全量 **1 分钟内不可重复**；成功后会向在线用户 SSE 推送 `rss_new`（若有新条）。

---

## 7. Apifox 测试指南

### 7.1 环境变量

在 Apifox 环境中配置：

| 变量 | 示例值 |
|------|--------|
| baseUrl | `http://127.0.0.1:8080/api/v1` |
| studentId | 你的学号 |
| password | 你的密码 |
| token | （登录后自动填入） |
| sourceId | （创建源后填入） |
| itemId | （列表返回后填入） |

**登录接口后置脚本示例：**

```javascript
const res = pm.response.json();
if (res.code === 200 && res.data?.token) {
  pm.environment.set("token", res.data.token);
}
```

### 7.2 推荐测试顺序

| 序号 | 接口 | 角色 |
|------|------|------|
| 1 | POST `/user/login` | 任意用户 |
| 2 | POST `/user/login` | 教师账号（管理端） |
| 3 | GET `/rss/admin/sources` | 教师 |
| 4 | POST `/rss/admin/sources` | 教师 |
| 5 | POST `/rss/admin/sources/{{sourceId}}/refresh` | 教师 |
| 6 | GET `/rss/items` | 学生 |
| 7 | GET `/rss/unread-count` | 学生 |
| 8 | GET `/rss/feed/recommended` | 学生 |
| 9 | GET `/rss/items/{{itemId}}` | 学生 |
| 10 | POST `/rss/events` | 学生 |
| 11 | POST `/rss/read` | 学生 |
| 12 | GET `/rss/items/latest?since=...` | 学生 |
| 13 | GET `/rss/stream` | 学生（SSE） |

### 7.3 公开 RSS 源（Apifox 创建源用）

| name | feed_url |
|------|----------|
| Hacker News | `https://hnrss.org/frontpage` |
| 阮一峰的网络日志 | `https://www.ruanyifeng.com/blog/atom.xml` |
| Go Blog | `https://go.dev/blog/feed.atom` |

创建后务必调用 **refresh**，否则库中无条目。

### 7.4 SSE 在 Apifox 中测试

1. 新建接口：`GET {{baseUrl}}/rss/stream`
2. Query：`token={{token}}`，`since=2020-01-01T00:00:00Z`
3. 若 Apifox 版本支持 **SSE / 流式响应**，可看到 `snapshot`、`rss_new`
4. 否则用浏览器或 curl：

```bash
curl -N "http://127.0.0.1:8080/api/v1/rss/stream?token=YOUR_JWT&since=2020-01-01T00:00:00Z"
```

5. 另开终端触发刷新：`POST .../rss/admin/sources/{{sourceId}}/refresh`，观察 SSE 是否出现 `event: rss_new`

---

## 8. Apifox 测试用例正文（可直接复制）

见同目录：**[RSS-APIFOX-TEST-CASES.md](./RSS-APIFOX-TEST-CASES.md)**（请求体/断言/变量提取完整清单）。

---

## 9. 常见问题

| 现象 | 处理 |
|------|------|
| 列表为空 | 管理端是否创建源并 **refresh** |
| unread 一直很多 | 调 `POST /read` `mark_all` 或逐条打开 |
| 无 AI 精选 | 检查 `conf` AI Chat 配置；无配置时仍有推荐，只是 reason 不同 |
| SSE 503 | 连接过多，稍后重连 |
| latest 400 | `since` 必填且为 RFC3339 |

---

**维护：** xiaoTnb  
**相关代码：** `pkg/app/api-server/server/server_rss.go`、`pkg/app/api-server/service/service_rss*.go`
