# 前端对接文档：选择题 / 填空题接口

本文档面向前端联调，描述当前题目模块的通用接口。所有接口前缀均为 `/api/v1`。

## 1. 基本约定

### 1.1 鉴权

所有接口都需要登录后访问，请在请求头中携带：

```http
Authorization: Bearer <token>
Content-Type: application/json
```

### 1.2 角色权限

- 教师、管理员：可创建、修改、删除题目
- 已登录用户：可查看自己有权限访问的题目

### 1.3 发布范围

`publish_mode` 取值：

- `global`：仅全局题库
- `course_only`：仅课程题库
- `global_and_course`：全局题库 + 课程题库

`publish_mode` 和 `course_ids` 的关系：

- `global`：`course_ids` 必须为空或不传
- `course_only`：`course_ids` 必须非空
- `global_and_course`：`course_ids` 必须非空

### 1.4 通用响应结构

成功响应：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

失败响应：

```json
{
  "code": 400,
  "msg": "请求参数错误",
  "error": "具体错误信息"
}
```

## 2. 选择题接口

### 2.1 创建选择题

- 方法：`POST`
- 路径：`/choice-questions`

请求体：

```json
{
  "title": "Go 框架选择题",
  "stem": "下面哪些是 Go Web 框架？",
  "difficulty": "easy",
  "tags": ["Go", "Web"],
  "status": "published",
  "publish_mode": "global_and_course",
  "course_ids": ["507f1f77bcf86cd799439031"],
  "choice_mode": "multiple",
  "options": [
    { "key": "A", "content": "Gin" },
    { "key": "B", "content": "Echo" },
    { "key": "C", "content": "Django" }
  ],
  "answers": ["A", "B"],
  "analysis": "Gin 和 Echo 都是常见 Go Web 框架。",
  "score": 10
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `title` | `string` | 是 | 题目标题 |
| `stem` | `string` | 是 | 题干 |
| `difficulty` | `string` | 是 | 难度 |
| `tags` | `string[]` | 否 | 标签数组 |
| `status` | `string` | 否 | `draft` / `published` / `archived`，不传时后端会给默认值 |
| `publish_mode` | `string` | 是 | 发布范围 |
| `course_ids` | `string[]` | 否 | 课程挂载范围 |
| `choice_mode` | `string` | 是 | `single` / `multiple` |
| `options` | `array` | 是 | 选项列表 |
| `answers` | `string[]` | 是 | 正确答案 key 列表 |
| `analysis` | `string` | 否 | 解析 |
| `score` | `number` | 否 | 分值 |

成功响应示例：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "question_id": "6822f1a77bcf86cd79943901",
    "question": {
      "id": "6822f1a77bcf86cd79943901",
      "title": "Go 框架选择题",
      "stem": "下面哪些是 Go Web 框架？",
      "difficulty": "easy",
      "tags": ["Go", "Web"],
      "status": "published",
      "publish_mode": "global_and_course",
      "choice_mode": "multiple",
      "options": [
        { "key": "A", "content": "Gin" },
        { "key": "B", "content": "Echo" },
        { "key": "C", "content": "Django" }
      ],
      "correct_answers": ["A", "B"],
      "analysis": "Gin 和 Echo 都是常见 Go Web 框架。",
      "score": 10,
      "created_by": "507f1f77bcf86cd799439099",
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    }
  }
}
```

### 2.2 获取选择题详情

- 方法：`GET`
- 路径：`/choice-questions/{questionId}`

成功响应示例：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": "6822f1a77bcf86cd79943901",
    "title": "Go 框架选择题",
    "stem": "下面哪些是 Go Web 框架？",
    "difficulty": "easy",
    "tags": ["Go", "Web"],
    "status": "published",
    "publish_mode": "global_and_course",
    "choice_mode": "multiple",
    "options": [
      { "key": "A", "content": "Gin" },
      { "key": "B", "content": "Echo" },
      { "key": "C", "content": "Django" }
    ],
    "correct_answers": ["A", "B"],
    "analysis": "Gin 和 Echo 都是常见 Go Web 框架。",
    "score": 10,
    "created_by": "507f1f77bcf86cd799439099",
    "created_at": "2026-05-13T10:00:00Z",
    "updated_at": "2026-05-13T10:00:00Z"
  }
}
```

### 2.3 更新选择题

- 方法：`PUT`
- 路径：`/choice-questions/{questionId}`

说明：

- 支持部分字段更新
- 如果要修改课程挂载范围，请显式传完整的 `course_ids`
- 如果改成 `publish_mode=global`，建议同时传空数组或不传 `course_ids`

请求体示例：

```json
{
  "title": "Go 框架选择题-更新",
  "publish_mode": "course_only",
  "course_ids": ["507f1f77bcf86cd799439031", "507f1f77bcf86cd799439032"],
  "analysis": "Gin 和 Echo 都是常见 Go Web 框架。",
  "score": 15
}
```

### 2.4 删除选择题

- 方法：`DELETE`
- 路径：`/choice-questions/{questionId}`

成功响应示例：

```json
{
  "code": 200,
  "msg": "success",
  "data": null
}
```

## 3. 填空题接口

### 3.1 创建填空题

- 方法：`POST`
- 路径：`/blank-questions`

请求体：

```json
{
  "title": "Go 填空题",
  "stem": "Go 常见 Web 框架有 ____ 和 ____。",
  "difficulty": "easy",
  "tags": ["Go", "Web"],
  "status": "published",
  "publish_mode": "course_only",
  "course_ids": ["507f1f77bcf86cd799439032"],
  "judge_mode": "exact_match",
  "blanks": [
    { "order": 1, "answers": ["Gin", "gin"] },
    { "order": 2, "answers": ["Echo", "echo"] }
  ],
  "analysis": "Gin 和 Echo 都是常见答案。",
  "score": 10
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `title` | `string` | 是 | 题目标题 |
| `stem` | `string` | 是 | 题干 |
| `difficulty` | `string` | 是 | 难度 |
| `tags` | `string[]` | 否 | 标签数组 |
| `status` | `string` | 否 | `draft` / `published` / `archived` |
| `publish_mode` | `string` | 是 | 发布范围 |
| `course_ids` | `string[]` | 否 | 课程挂载范围 |
| `judge_mode` | `string` | 是 | `exact_match` / `manual` |
| `blanks` | `array` | 是 | 填空槽位定义 |
| `analysis` | `string` | 否 | 解析 |
| `score` | `number` | 否 | 分值 |

成功响应示例：

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "question_id": "6822f1a77bcf86cd79943902",
    "question": {
      "id": "6822f1a77bcf86cd79943902",
      "title": "Go 填空题",
      "stem": "Go 常见 Web 框架有 ____ 和 ____。",
      "difficulty": "easy",
      "tags": ["Go", "Web"],
      "status": "published",
      "publish_mode": "course_only",
      "judge_mode": "exact_match",
      "blanks": [
        { "order": 1, "answers": ["Gin", "gin"] },
        { "order": 2, "answers": ["Echo", "echo"] }
      ],
      "analysis": "Gin 和 Echo 都是常见答案。",
      "score": 10,
      "created_by": "507f1f77bcf86cd799439099",
      "created_at": "2026-05-13T10:00:00Z",
      "updated_at": "2026-05-13T10:00:00Z"
    }
  }
}
```

### 3.2 获取填空题详情

- 方法：`GET`
- 路径：`/blank-questions/{questionId}`

### 3.3 更新填空题

- 方法：`PUT`
- 路径：`/blank-questions/{questionId}`

请求体示例：

```json
{
  "title": "Go 填空题-更新",
  "publish_mode": "global_and_course",
  "course_ids": ["507f1f77bcf86cd799439032"],
  "judge_mode": "manual",
  "analysis": "人工阅卷时可接受更多表达。",
  "score": 12
}
```

### 3.4 删除填空题

- 方法：`DELETE`
- 路径：`/blank-questions/{questionId}`

## 4. 前端建议处理

### 4.1 创建 / 更新前校验

前端建议在提交前先校验：

- `publish_mode=global` 时，不要传课程
- `publish_mode=course_only` 或 `global_and_course` 时，至少选择一个课程
- 选择题：
  - `single` 时 `answers` 长度应为 1
  - `multiple` 时 `answers` 长度应大于等于 1
- 填空题：
  - `blanks[].order` 不能重复
  - 每个 blank 至少有一个可接受答案

### 4.2 编辑页回填

编辑时建议直接使用详情接口返回的数据回填表单，并注意：

- 更新课程挂载时，提交完整 `course_ids`
- 不修改课程挂载时，也建议把当前 `course_ids` 一并带回，避免范围歧义

### 4.3 常见错误码

| code | 含义 | 前端建议 |
| --- | --- | --- |
| `400` | 参数错误 / 业务规则不满足 | toast 展示后端错误文案 |
| `401` | 未登录 | 跳登录或刷新 token |
| `403` | 无权限 | 展示无权限提示页或 toast |

## 5. 推荐联调顺序

1. 先完成创建接口联调
2. 再做详情回显
3. 再做更新接口
4. 最后接删除

## 6. 文档位置

- 面向开发的说明文档：[course-question-apis.md](/Users/Zhuanz/Documents/Code/school/zhku-oj/zhku-oj/docs/api/course-question-apis.md)
- 当前前端对接文档：[frontend-question-api.md](/Users/Zhuanz/Documents/Code/school/zhku-oj/zhku-oj/docs/api/frontend-question-api.md)
