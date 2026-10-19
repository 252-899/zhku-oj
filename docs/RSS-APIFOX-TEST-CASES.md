# RSS 模块 · Apifox 测试用例与示例数据

> 配合 [FRONTEND-RSS-INTEGRATION.md](./FRONTEND-RSS-INTEGRATION.md) 使用。  
> 将 `{{baseUrl}}` 设为 `http://127.0.0.1:8080/api/v1`（按你本地端口修改）。

---

## 环境变量

| 变量名 | 初始值 | 说明 |
|--------|--------|------|
| baseUrl | http://127.0.0.1:8080/api/v1 | API 根路径 |
| studentId | 20210001 | 改成你库里的学号 |
| password | 123456 | 改成真实密码 |
| teacherId | T001 | 教师学号（若有） |
| teacherPassword | 123456 | 教师密码 |
| token | | 学生 Token，登录后写入 |
| adminToken | | 教师/管理员 Token |
| sourceId | | 创建订阅源后写入 |
| itemId | | 列表第一条 id |
| since | 2020-01-01T00:00:00Z | SSE / latest 游标 |

---

## 0. 前置：健康检查

**GET** `http://127.0.0.1:8080/health`

无需 Token。期望 `code` 200。

---

## 1. 学生登录

**POST** `{{baseUrl}}/user/login`

**Headers：** `Content-Type: application/json`

**Body：**

```json
{
  "student_id": "{{studentId}}",
  "password": "{{password}}"
}
```

**期望：** HTTP 200，`data.token` 非空。

**后置脚本：**

```javascript
const json = pm.response.json();
if (json.code === 200 && json.data && json.data.token) {
  pm.environment.set("token", json.data.token);
}
```

**示例成功响应：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "登录成功",
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDc4MjQwMDB9.xxx",
    "user": {
      "id": "674000000000000000000001",
      "username": "张三",
      "role": "student"
    }
  }
}
```

---

## 2. 教师/管理员登录（管理端用）

**POST** `{{baseUrl}}/user/login`

**Body：**

```json
{
  "student_id": "{{teacherId}}",
  "password": "{{teacherPassword}}"
}
```

**后置脚本：**

```javascript
const json = pm.response.json();
if (json.code === 200 && json.data?.token) {
  pm.environment.set("adminToken", json.data.token);
}
```

---

## 3. 列出订阅源

**GET** `{{baseUrl}}/rss/admin/sources`

**Headers：**

```http
Authorization: Bearer {{adminToken}}
```

**期望：** `data.sources` 为数组（可为空）。

**后置脚本（有源时取第一个 id）：**

```javascript
const json = pm.response.json();
const sources = json.data?.sources;
if (sources && sources.length > 0) {
  pm.environment.set("sourceId", sources[0].id);
}
```

---

## 4. 创建订阅源

**POST** `{{baseUrl}}/rss/admin/sources`

**Headers：**

```http
Authorization: Bearer {{adminToken}}
Content-Type: application/json
```

**Body（任选其一 feed，不要重复创建同一 feed_url）：**

```json
{
  "name": "阮一峰的网络日志",
  "feed_url": "https://www.ruanyifeng.com/blog/atom.xml",
  "enabled": true
}
```

**备选 feed_url：**

```json
{
  "name": "Hacker News",
  "feed_url": "https://hnrss.org/frontpage",
  "enabled": true
}
```

```json
{
  "name": "Go 官方博客",
  "feed_url": "https://go.dev/blog/feed.atom",
  "enabled": true
}
```

**期望：** `message` 含「创建成功」。

**失败示例（重复地址）：**

```json
{
  "code": 400,
  "message": "该订阅地址已存在"
}
```

> 创建后 **必须执行步骤 5 刷新**，否则没有资讯条目。

---

## 5. 刷新单个订阅源（拉取 RSS 入库 + 触发 SSE）

**POST** `{{baseUrl}}/rss/admin/sources/{{sourceId}}/refresh`

**Headers：** `Authorization: Bearer {{adminToken}}`

**Body：** 无

**期望：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "刷新已触发并完成"
  }
}
```

**1 分钟内重复调用期望 400：**

```json
{
  "code": 400,
  "message": "操作过于频繁，请间隔至少 1 分钟后再试"
}
```

---

## 6. 刷新全部启用源

**POST** `{{baseUrl}}/rss/admin/refresh-all`

**Headers：** `Authorization: Bearer {{adminToken}}`

**期望：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "已依次刷新全部启用源"
  }
}
```

---

## 7. 资讯列表（时间线）

**GET** `{{baseUrl}}/rss/items?page=1&page_size=10`

**Headers：** `Authorization: Bearer {{token}}`

**可选 Query：** `source_id={{sourceId}}`

**期望：** `data.items` 长度 ≥ 1（刷新成功后）。

**后置脚本：**

```javascript
const json = pm.response.json();
const items = json.data?.items;
if (items && items.length > 0) {
  pm.environment.set("itemId", items[0].id);
  if (items[0].fetched_at) {
    pm.environment.set("since", items[0].fetched_at);
  }
}
```

**示例 `data.items[0]`：**

```json
{
  "id": "674a1b2c3d4e5f6789012345",
  "source_id": "674a00000000000000000001",
  "source_name": "阮一峰的网络日志",
  "title": "示例文章标题",
  "link": "https://www.ruanyifeng.com/blog/2026/05/example.html",
  "summary": "这是摘要纯文本……",
  "published_at": "2026-05-20T08:00:00Z",
  "fetched_at": "2026-05-21T10:15:30.123456789Z",
  "is_read": false
}
```

---

## 8. 未读数量

**GET** `{{baseUrl}}/rss/unread-count`

**Headers：** `Authorization: Bearer {{token}}`

**期望示例：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "unread_count": 15,
    "last_read_at": ""
  }
}
```

---

## 9. 个性化推荐

**GET** `{{baseUrl}}/rss/feed/recommended?page=1&page_size=10`

**Headers：** `Authorization: Bearer {{token}}`

**期望：** `items[].score`、`items[].reason` 存在。

**示例条目：**

```json
{
  "id": "674a1b2c3d4e5f6789012345",
  "title": "Go 并发模式实践",
  "summary": "……",
  "is_read": false,
  "score": 0.782,
  "reason": "与您的学习兴趣相关"
}
```

**reason 可能值：** `AI 精选`、`与您的学习兴趣相关`、`常读来源`、`内容相似推荐`、`平台热门`、`已读降权`、`最新资讯`

---

## 10. 资讯详情

**GET** `{{baseUrl}}/rss/items/{{itemId}}`

**Headers：** `Authorization: Bearer {{token}}`

**期望：** 含 `body_html`、`tags`、`is_read`；调用后该条应为已读。

**示例：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "674a1b2c3d4e5f6789012345",
    "title": "示例",
    "body_html": "<p>正文</p>",
    "body_source": "content",
    "tags": ["go", "programming"],
    "open_count": 1,
    "click_count": 0,
    "is_read": true
  }
}
```

---

## 11. 行为上报 · 停留

**POST** `{{baseUrl}}/rss/events`

**Headers：**

```http
Authorization: Bearer {{token}}
Content-Type: application/json
```

**Body：**

```json
{
  "item_id": "{{itemId}}",
  "action": "dwell",
  "dwell_ms": 15000
}
```

**其它 action 用例：**

```json
{ "item_id": "{{itemId}}", "action": "click" }
```

```json
{ "item_id": "{{itemId}}", "action": "not_interested" }
```

**期望：**

```json
{
  "code": 200,
  "message": "success",
  "data": { "message": "已记录" }
}
```

---

## 12. 标记已读 · 单条

**POST** `{{baseUrl}}/rss/read`

**Body：**

```json
{
  "item_id": "{{itemId}}"
}
```

**期望：** `data.unread_count` 减少。

---

## 13. 标记已读 · 全部

**POST** `{{baseUrl}}/rss/read`

**Body：**

```json
{
  "mark_all": true
}
```

**期望：** `unread_count` 为 0 或接近 0。

**错误示例（两个都传）：**

```json
{
  "item_id": "{{itemId}}",
  "mark_all": true
}
```

期望 400：`mark_all 与 item_id 不能同时设置`

---

## 14. 新条轮询

**GET** `{{baseUrl}}/rss/items/latest?since={{since}}&limit=20`

**Headers：** `Authorization: Bearer {{token}}`

**since 示例：** `2026-05-21T00:00:00Z` 或 `2026-05-21T10:00:00.123456789Z`

**期望：**

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "items": []
  }
}
```

**缺少 since 期望 400：**

```json
{
  "code": 400,
  "message": "缺少 since 参数（RFC3339 时间）"
}
```

---

## 15. SSE 推送流

**GET** `{{baseUrl}}/rss/stream?token={{token}}&since=2020-01-01T00:00:00Z`

**说明：** Apifox 部分版本对流式支持有限，建议 **curl** 联调：

```bash
curl -N -H "Accept: text/event-stream" ^
  "http://127.0.0.1:8080/api/v1/rss/stream?token=粘贴token&since=2020-01-01T00:00:00Z"
```

**典型输出片段：**

```text
id: 2026-05-21T10:20:00.123456789Z
event: snapshot
data: {"type":"snapshot","event_id":"2026-05-21T10:20:00.123456789Z","ts":1716280000000,"cursor":"2026-05-21T10:20:00.123456789Z","count":3,"items":[...],"ids":["..."]}

: connected

: ping

id: 2026-05-21T11:00:00.000000000Z
event: rss_new
data: {"type":"rss_new","event_id":"2026-05-21T11:00:00.000000000Z","ts":1716283600000,"cursor":"2026-05-21T11:00:00.000000000Z","count":1,"items":[...],"ids":["674a..."]}
```

**联调步骤：**

1. 终端 A 运行 curl 保持连接  
2. Apifox 执行 **步骤 5 refresh**  
3. 终端 A 应出现 `event: rss_new`

**backlog 示例：**

```json
{
  "type": "backlog",
  "event_id": "2026-05-21T11:00:00.000000000Z",
  "cursor": "2026-05-21T11:00:00.000000000Z",
  "count": 0
}
```

→ 前端应调用 `GET /items/latest?since=cursor`

---

## 16. 更新订阅源

**PUT** `{{baseUrl}}/rss/admin/sources/{{sourceId}}`

**Body：**

```json
{
  "name": "阮一峰博客（已改名）",
  "enabled": true
}
```

---

## 17. 删除订阅源（慎用）

**DELETE** `{{baseUrl}}/rss/admin/sources/{{sourceId}}`

**期望：**

```json
{
  "code": 200,
  "message": "success",
  "data": { "message": "删除成功" }
}
```

---

## 18. 完整链路检查清单（Apifox 手动跑一遍）

- [ ] 学生登录拿到 `token`
- [ ] 教师登录拿到 `adminToken`
- [ ] 创建订阅源成功
- [ ] refresh 后 `GET /items` 有数据
- [ ] `unread-count` > 0
- [ ] `feed/recommended` 有 `score`、`reason`
- [ ] `GET /items/:id` 后 `is_read=true`，未读减少
- [ ] `POST /events` dwell 成功
- [ ] `POST /read` mark_all 后 unread≈0
- [ ] `latest?since=` 返回合理
- [ ] curl SSE 能收到 snapshot；refresh 后能收到 rss_new

---

## 19. 错误用例（可选）

| 接口 | 操作 | 期望 |
|------|------|------|
| GET /rss/items | 无 Authorization | 401 |
| GET /rss/stream | 无 token | 401 |
| POST /rss/read | `{}` 空 body | 400 |
| POST /rss/events | action=invalid | 400 |
| GET /rss/items/invalid_hex | id 非法 | 400 |
| POST /rss/admin/sources | 学生 token | 403 |

---

## 20. MongoDB 无登录数据时

若本地库无用户，需先通过平台注册/种子脚本创建学生与教师账号，或将 `studentId` / `password` 换成你环境中真实账号。

无 RSS 数据时顺序必须是：**创建源 → refresh → 再测用户接口**。

---

**文档作者：** xiaoTnb
