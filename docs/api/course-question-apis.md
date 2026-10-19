# 题目接口文档

本文档描述新的通用选择题与填空题接口。所有接口前缀均为 `/api/v1`，并且都需要携带 `Bearer Token`。

## 1. 设计约定

- 题目本体统一通过通用接口维护，不再区分“从课程下创建”还是“从全局下创建”。
- `publish_mode=global` 表示题目只进入全局题库，此时 `course_ids` 必须为空。
- `publish_mode=course_only` 表示题目只挂载到课程题库，此时 `course_ids` 必须非空。
- `publish_mode=global_and_course` 表示题目同时进入全局题库和课程题库，此时 `course_ids` 必须非空。
- `course_ids` 表示题目当前挂载到哪些课程，后续任务关联会在单独接口中处理。

## 2. 路由总览

### 2.1 选择题

| 方法 | 路径 | 说明 | 角色边界 |
| --- | --- | --- | --- |
| POST | `/choice-questions` | 创建选择题 | 教师、管理员 |
| GET | `/choice-questions/{questionId}` | 获取选择题详情 | 已登录用户，最终可见性由服务层裁决 |
| PUT | `/choice-questions/{questionId}` | 更新选择题 | 教师、管理员 |
| DELETE | `/choice-questions/{questionId}` | 删除选择题 | 教师、管理员 |

### 2.2 填空题

| 方法 | 路径 | 说明 | 角色边界 |
| --- | --- | --- | --- |
| POST | `/blank-questions` | 创建填空题 | 教师、管理员 |
| GET | `/blank-questions/{questionId}` | 获取填空题详情 | 已登录用户，最终可见性由服务层裁决 |
| PUT | `/blank-questions/{questionId}` | 更新填空题 | 教师、管理员 |
| DELETE | `/blank-questions/{questionId}` | 删除填空题 | 教师、管理员 |

## 3. 选择题接口

### 3.1 创建选择题

- 方法：`POST`
- 路径：`/api/v1/choice-questions`

请求体示例：

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
      "score": 10
    }
  }
}
```

### 3.2 获取选择题详情

- 方法：`GET`
- 路径：`/api/v1/choice-questions/{questionId}`

### 3.3 更新选择题

- 方法：`PUT`
- 路径：`/api/v1/choice-questions/{questionId}`
- 说明：若要调整课程挂载范围，需要显式传入完整 `course_ids`。

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

### 3.4 删除选择题

- 方法：`DELETE`
- 路径：`/api/v1/choice-questions/{questionId}`
- 说明：删除时会同步清理课程挂载关系，但不会级联删除历史单题提交记录。

## 4. 填空题接口

### 4.1 创建填空题

- 方法：`POST`
- 路径：`/api/v1/blank-questions`

请求体示例：

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
      "score": 10
    }
  }
}
```

### 4.2 获取填空题详情

- 方法：`GET`
- 路径：`/api/v1/blank-questions/{questionId}`

### 4.3 更新填空题

- 方法：`PUT`
- 路径：`/api/v1/blank-questions/{questionId}`
- 说明：若要调整课程挂载范围，需要显式传入完整 `course_ids`。

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

### 4.4 删除填空题

- 方法：`DELETE`
- 路径：`/api/v1/blank-questions/{questionId}`
- 说明：删除时会同步清理课程挂载关系，但不会级联删除历史单题提交记录。

## 5. 常见错误响应

统一错误体结构为：

```json
{
  "code": 400,
  "msg": "请求参数错误",
  "error": "具体错误信息"
}
```

常见场景：

- `400`：题目 ID 或课程 ID 非法、请求体校验失败、业务规则不满足。
- `401`：未登录或缺少有效 Token。
- `403`：学生尝试写操作，或用户无权读取当前题目。

## 6. 在线文档入口

- Swagger：`/swagger/index.html`
- Scalar Light：`/docs/light`
- Scalar Dark：`/docs/dark`
- Scalar Classic：`/docs/classic`
