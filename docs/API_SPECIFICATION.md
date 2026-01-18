# API 接口文档

> 最后更新时间：2025-01-06

## 目录

- [基本信息](#基本信息)
- [认证接口](#认证接口)
- [用户接口](#用户接口)
- [博客接口](#博客接口)
- [分类接口](#分类接口)
- [评论接口](#评论接口)
- [访问记录接口](#访问记录接口)
- [访客接口](#访客接口)
- [博主统计接口](#博主统计接口)
- [GitHub OAuth 接口](#github-oauth-接口)
- [接口规范说明](#接口规范说明)

---

## 基本信息

- **Base URL**: `http://localhost:8080` 或 `http://localhost:8080/api`
- **Swagger 文档**: `http://localhost:8080/swagger/index.html`
- **所有接口同时支持 `/path` 和 `/api/path` 两种路径格式**

### 统一响应格式

```json
{
  "code": 200,           // 状态码
  "msg": "success",      // 消息描述
  "data": {},            // 实际数据
  "total": 100,          // 总数（分页时可选）
  "page": 1              // 当前页（分页时可选）
}
```

### HTTP 状态码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 认证接口

### 用户登录

**POST** `/login`

验证用户身份并返回 JWT 令牌。

**请求体：**

```json
{
  "account": "user@example.com",
  "password": "password123"
}
```

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "name": "用户名",
      "account": "user@example.com"
    },
    "is_owner": true  // 仅博主登录时返回
  }
}
```

**错误响应：**

- `400` - 无效的请求体
- `401` - 未找到用户

---

## 用户接口

### 获取所有用户

**GET** `/users`

获取所有已注册用户的列表。

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "name": "用户名",
      "account": "user@example.com",
      "nickname": "昵称",
      "birthday": "2000-01-01",
      "avatar_url": "https://..."
    }
  ]
}
```

### 添加新用户

**POST** `/users`

创建一个新用户。**需要身份验证（JWT Token）**。

**请求头：**

```
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "name": "用户名",
  "account": "user@example.com",
  "password": "password123",
  "nickname": "昵称",
  "birthday": "2000-01-01"
}
```

**成功响应：** `201 Created`

---

## 博客接口

### 获取博客列表

**GET** `/blogs`

根据分类ID分页获取博客列表，支持关键词搜索。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| category_id | int | 否 | 分类ID |
| keyword | string | 否 | 搜索关键词 |
| page | int | 否 | 页码，默认1 |
| pagesize | int | 否 | 每页数量，默认10 |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "title": "博客标题",
      "url": "/path/to/blog",
      "category_id": 1,
      "category_name": "技术",
      "description": "博客描述",
      "created_at": "2025-01-01 12:00:00",
      "updated_at": "2025-01-01 12:00:00"
    }
  ],
  "total": 100,
  "page": 1
}
```

### 获取博客详情

**GET** `/blogs/{id}`

根据ID获取单个博客的详细信息。

**路径参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| id | int | 是 | 博客ID |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": 1,
    "title": "博客标题",
    "url": "/path/to/blog",
    "category_id": 1,
    "category_name": "技术",
    "description": "博客描述",
    "created_at": "2025-01-01 12:00:00",
    "updated_at": "2025-01-01 12:00:00"
  }
}
```

**错误响应：**

- `400` - 参数错误
- `404` - 博客不存在

---

## 分类接口

### 创建分类

**POST** `/categories`

创建一个新的分类，可以是顶级分类或子分类。

**请求体：**

```json
{
  "name": "技术",           // 必传，分类名称
  "type": "folder",         // 可选，folder(文件夹)或article(文章)，默认folder
  "url": "/path/to/article", // 可选，文章类型时的URL
  "parent_id": 1,           // 可选，父分类ID
  "sort_order": 0           // 可选，排序顺序
}
```

**成功响应：** `201 Created`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": 1,
    "name": "技术",
    "type": "folder",
    "parent_id": null,
    "sort_order": 0
  }
}
```

**错误响应：**

- `400` - 参数错误（名称为空、类型无效、父分类不存在）

### 获取分类列表

**GET** `/categories`

获取所有分类，支持树形结构返回。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| tree | bool | 否 | 是否返回树形结构，默认true |
| parent_id | int | 否 | 父分类ID |
| type | string | 否 | 类型筛选：folder或article |
| keyword | string | 否 | 标题模糊搜索关键词 |

**响应示例（树形结构）：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "name": "Vue",
      "type": "folder",
      "parent_id": null,
      "sort_order": 1,
      "children": [
        {
          "id": 6,
          "name": "Vue3基础",
          "type": "article",
          "url": "/vue/basics",
          "parent_id": 1,
          "sort_order": 1,
          "children": []
        }
      ]
    }
  ]
}
```

### 获取单个分类

**GET** `/categories/{id}`

根据ID获取分类详情，包含子分类。

**路径参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类ID |

**错误响应：**

- `400` - 参数错误
- `404` - 分类不存在

### 更新分类

**PUT** `/categories/{id}`

更新分类信息。

**路径参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类ID |

**请求体：**

```json
{
  "name": "新名称",         // 必传
  "type": "folder",         // 可选
  "parent_id": 2,           // 可选
  "sort_order": 1           // 可选
}
```

**错误响应：**

- `400` - 参数错误（名称为空、不能设置自己为父分类）
- `404` - 分类不存在

### 删除分类

**DELETE** `/categories/{id}`

删除分类，子分类会一并删除。

**路径参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| id | int | 是 | 分类ID |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "message": "Category deleted successfully"
  }
}
```

> ⚠️ 注意：删除分类会级联删除所有子分类

---

## 评论接口

### 获取文章评论列表

**GET** `/comments`

根据文章ID获取评论列表，返回树形结构。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| article_id | int | 是 | 文章ID |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "article_id": 10,
      "parent_id": null,
      "nickname": "评论者",
      "email": "user@example.com",
      "content": "评论内容",
      "created_at": "2025-01-01 12:00:00",
      "children": [
        {
          "id": 2,
          "article_id": 10,
          "parent_id": 1,
          "nickname": "回复者",
          "content": "回复内容",
          "created_at": "2025-01-01 13:00:00",
          "children": []
        }
      ]
    }
  ],
  "total": 2,
  "page": 1
}
```

### 创建评论

**POST** `/comments`

为文章创建评论，支持回复其他评论。

**请求体：**

```json
{
  "article_id": 10,         // 必传，文章ID
  "parent_id": 1,           // 可选，父评论ID（回复时使用）
  "nickname": "评论者",      // 必传，昵称
  "email": "user@example.com", // 可选，邮箱
  "content": "评论内容"      // 必传，评论内容
}
```

**错误响应：**

- `400` - 参数错误（缺少必填字段、文章不存在、父评论不存在）

### 删除评论

**DELETE** `/comments/{id}`

根据ID删除评论。

**路径参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| id | int | 是 | 评论ID |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "deleted_id": 1
  }
}
```

---

## 访问记录接口

### 记录用户访问

**POST** `/visits`

记录一次新的用户访问。

**请求体：**

```json
{
  "user_nickname": "访客",           // 可选
  "visit_time": "2025-01-01 12:00:00", // 必传
  "content": "访问内容描述"          // 可选，默认"普通访问记录"
}
```

**成功响应：** `201 Created`

### 获取访问日志

**GET** `/visits`

检索所有用户访问日志的列表。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认1 |
| pagesize | int | 否 | 每页数量，默认10 |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "user_nickname": "访客",
      "visit_time": "2025-01-01 12:00:00",
      "content": "普通访问记录",
      "created_at": "2025-01-01 12:00:00"
    }
  ],
  "total": 100,
  "page": 1
}
```

---

## 访客接口

### 记录访客进入

**POST** `/guest`

记录访客进入网站的时间和内容信息。

**请求体：**

```json
{
  "entry_time": "2025-01-01 12:00:00", // 必传
  "content": "访客进入信息"            // 必传
}
```

**成功响应：** `201 Created`

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "id": 1,
    "entry_time": "2025-01-01 12:00:00",
    "content": "访客进入信息",
    "created_at": "2025-01-01 12:00:00"
  }
}
```

---

## 博主统计接口

### 获取博主访问统计

**GET** `/owner/visits`

获取博客主人指定天数内每天访问次数的统计信息。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| days | int | 否 | 获取最近多少天的数据，默认7天 |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "visit_stats": [
      {
        "date": "2025-01-01",
        "count": 5
      }
    ],
    "total_visits": 35,
    "days": "7"
  }
}
```

### 获取博主今日访问次数

**GET** `/owner/today-visits`

获取博客主人今天的访问次数统计。

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "date": "2025-01-06",
    "today_visits": 3
  }
}
```

---

## GitHub OAuth 接口

### 获取 GitHub 登录 URL

**GET** `/auth/github`

返回 GitHub OAuth 授权页面的 URL。

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "url": "https://github.com/login/oauth/authorize?client_id=xxx&redirect_uri=xxx&scope=user:email"
  }
}
```

### GitHub OAuth 回调

**GET** `/auth/github/callback`

处理 GitHub OAuth 回调，获取用户信息并登录/注册。

**查询参数：**

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| code | string | 是 | GitHub 授权码 |

**成功响应：**

```json
{
  "code": 200,
  "msg": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "name": "GitHub用户名",
      "account": "github_123456",
      "avatar_url": "https://avatars.githubusercontent.com/...",
      "github_url": "https://github.com/username"
    }
  }
}
```

### 使用授权码进行 GitHub 登录

**POST** `/auth/github/login`

前端传递 GitHub 授权码，后端处理登录/注册。

**请求体：**

```json
{
  "code": "github_authorization_code"
}
```

**成功响应：** 同上

---

## 接口规范说明

### 1. 基本原则

### 1.1 RESTful 设计

- 使用标准 HTTP 方法：GET（查询）、POST（创建）、PUT（更新）、DELETE（删除）
- 使用名词作为资源路径，避免动词
- 使用复数形式：`/users`、`/visits`、`/guests`

### 1.2 统一响应格式

所有接口必须使用统一的 `APIResponse` 格式：

```json
{
  "code": 200,           // 状态码
  "msg": "success",      // 消息描述
  "data": {},            // 实际数据
  "total": 100,          // 总数（分页时可选）
  "page": 1              // 当前页（分页时可选）
}
```

### 1.3 HTTP 状态码

- `200` - 成功
- `201` - 创建成功
- `400` - 请求参数错误
- `401` - 未授权
- `403` - 禁止访问
- `404` - 资源不存在
- `500` - 服务器内部错误

## 2. 接口注释规范

### 2.1 Swagger 注释模板

```go
// FunctionName godoc
// @Summary 接口简要描述
// @Description 接口详细描述，说明功能和使用场景
// @Tags 标签分组
// @Accept  json
// @Produce  json
// @Param   param_name   query    string     true  "参数描述" 
// @Param   request_body body    models.RequestModel true "请求体描述"
// @Success 200 {object} models.APIResponse "成功响应"
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 401 {object} models.APIResponse "未授权"
// @Failure 500 {object} models.APIResponse "服务器错误"
// @Security ApiKeyAuth
// @Router /api/path [method]
```

### 2.2 注释字段说明

| 字段 | 说明 | 示例 |
|------|------|------|
| @Summary | 接口简要描述（一句话） | "用户登录" |
| @Description | 详细描述（可多行） | "验证用户身份并返回JWT令牌" |
| @Tags | 接口分组 | "auth"、"users"、"visits" |
| @Accept | 接受的请求类型 | "json" |
| @Produce | 返回的响应类型 | "json" |
| @Param | 参数说明 | 详见下方参数规范 |
| @Success | 成功响应 | `{object} models.APIResponse` |
| @Failure | 失败响应 | `{object} models.APIResponse` |
| @Security | 安全认证 | "ApiKeyAuth"（可选） |
| @Router | 路由路径 | "/login [post]" |

## 3. 参数规范

### 3.1 路径参数

```go
// @Param   id   path    int     true  "用户ID"
```

### 3.2 查询参数

```go
// @Param   page   query    int     false  "页码，默认1"
// @Param   size   query    int     false  "每页数量，默认10"
// @Param   keyword query   string  false  "搜索关键词"
```

### 3.3 请求体参数

```go
// @Param   user   body    models.User   true  "用户信息"
```

### 3.4 Header 参数

```go
// @Param   Authorization   header    string  true  "Bearer token"
```

## 4. 响应规范

### 4.1 成功响应

```go
// @Success 200 {object} models.APIResponse "操作成功"
```

### 4.2 错误响应

```go
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 401 {object} models.APIResponse "未授权"
// @Failure 500 {object} models.APIResponse "服务器错误"
```

### 4.3 分页响应

```go
// @Success 200 {object} models.APIResponse{data=[]models.User,total=int,page=int} "用户列表"
```

## 5. 代码实现规范

### 5.1 错误处理

```go
func Handler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // 参数验证
    var req models.RequestModel
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        response := models.ErrorResponse(400, "Invalid request body")
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusBadRequest)
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // 业务逻辑处理
    data, err := processBusinessLogic(req, db)
    if err != nil {
        response := models.ErrorResponse(500, "Business logic failed: "+err.Error())
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusInternalServerError)
        json.NewEncoder(w).Encode(response)
        return
    }
    
    // 成功响应
    response := models.SuccessResponse(data)
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 5.2 参数验证

```go
// 必填参数验证
if req.RequiredField == "" {
    response := models.ErrorResponse(400, "Required field is missing")
    // ...
}

// 参数格式验证
if !isValidEmail(req.Email) {
    response := models.ErrorResponse(400, "Invalid email format")
    // ...
}
```

## 6. 命名规范

### 6.1 接口命名

- 使用动词+名词：`GetUsers`、`CreateUser`、`UpdateUser`
- 遵循 Go 命名规范：首字母大写（公开）

### 6.2 路由命名

- 使用小写+连字符：`/user-profile`、`/visit-logs`
- 资源名使用复数：`/users`、`/visits`

### 6.3 模型命名

- 使用大驼峰：`User`、`VisitLog`、`GuestRecord`
- 响应模型添加后缀：`UserResponse`、`UserRequest`

## 7. 示例模板

### 7.1 CRUD 接口模板

```go
// GetUsers godoc
// @Summary 获取用户列表
// @Description 分页获取用户列表，支持搜索
// @Tags users
// @Accept  json
// @Produce  json
// @Param   page    query    int     false  "页码，默认1"
// @Param   size    query    int     false  "每页数量，默认10"
// @Param   keyword query    string  false  "搜索关键词"
// @Success 200 {object} models.APIResponse{data=[]models.User,total=int,page=int} "获取成功"
// @Failure 500 {object} models.APIResponse "服务器错误"
// @Router /users [get]
func GetUsers(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // 实现代码...
}

// CreateUser godoc
// @Summary 创建用户
// @Description 创建新用户
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user   body    models.CreateUserRequest  true  "用户信息"
// @Success 201 {object} models.APIResponse{data=models.User} "创建成功"
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 500 {object} models.APIResponse "服务器错误"
// @Router /users [post]
func CreateUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // 实现代码...
}
```

### 7.2 认证接口模板

```go
// Login godoc
// @Summary 用户登录
// @Description 验证用户身份并返回JWT令牌
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   credentials   body    models.LoginRequest  true  "登录凭证"
// @Success 200 {object} models.APIResponse{data=models.LoginResponse} "登录成功"
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 401 {object} models.APIResponse "用户名或密码错误"
// @Router /login [post]
func Login(w http.ResponseWriter, r *http.Request, db *sql.DB) {
    // 实现代码...
}
```

## 8. 测试规范

### 8.1 单元测试

- 每个接口函数都要有对应的单元测试
- 测试文件命名：`handler_test.go`
- 测试函数命名：`TestFunctionName`

### 8.2 集成测试

- 测试完整的请求-响应流程
- 使用测试数据库
- 测试边界条件和错误场景

## 9. 版本控制

### 9.1 API 版本

- 在路由中添加版本号：`/api/v1/users`
- 向后兼容原则
- 废弃接口要提前通知

### 9.2 文档版本

- 每次接口变更都要更新文档
- 记录变更日志
- 标注废弃和新增的接口

---

## 数据模型

### User 用户模型

```json
{
  "id": 1,
  "name": "用户名",
  "account": "user@example.com",
  "nickname": "昵称",
  "birthday": "2000-01-01",
  "github_id": 123456,
  "avatar_url": "https://avatars.githubusercontent.com/...",
  "github_url": "https://github.com/username"
}
```

### Blog 博客模型

```json
{
  "id": 1,
  "title": "博客标题",
  "url": "/path/to/blog",
  "category_id": 1,
  "category_name": "技术",
  "description": "博客描述",
  "created_at": "2025-01-01 12:00:00",
  "updated_at": "2025-01-01 12:00:00"
}
```

### Category 分类模型

```json
{
  "id": 1,
  "name": "技术",
  "type": "folder",         // folder=文件夹，article=文章
  "url": "/path/to/article", // 文章类型时的URL
  "parent_id": null,        // null表示顶级分类
  "sort_order": 0,
  "created_at": "2025-01-01 00:00:00",
  "updated_at": "2025-01-01 00:00:00",
  "children": []            // 子分类列表
}
```

### Comment 评论模型

```json
{
  "id": 1,
  "article_id": 10,
  "parent_id": null,        // null表示顶级评论
  "nickname": "评论者",
  "email": "user@example.com",
  "content": "评论内容",
  "created_at": "2025-01-01 12:00:00",
  "children": []            // 子评论列表
}
```

### VisitLog 访问日志模型

```json
{
  "id": 1,
  "user_nickname": "访客",
  "visit_time": "2025-01-01 12:00:00",
  "content": "访问内容",
  "created_at": "2025-01-01 12:00:00"
}
```

### GuestRecord 访客记录模型

```json
{
  "id": 1,
  "entry_time": "2025-01-01 12:00:00",
  "content": "访客进入信息",
  "created_at": "2025-01-01 12:00:00"
}
```

---

## 接口汇总表

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/login` | 用户登录 | 否 |
| GET | `/users` | 获取所有用户 | 否 |
| POST | `/users` | 添加新用户 | 是 |
| GET | `/blogs` | 获取博客列表 | 否 |
| GET | `/blogs/{id}` | 获取博客详情 | 否 |
| GET | `/categories` | 获取分类列表 | 否 |
| POST | `/categories` | 创建分类 | 否 |
| GET | `/categories/{id}` | 获取单个分类 | 否 |
| PUT | `/categories/{id}` | 更新分类 | 否 |
| DELETE | `/categories/{id}` | 删除分类 | 否 |
| GET | `/comments` | 获取评论列表 | 否 |
| POST | `/comments` | 创建评论 | 否 |
| DELETE | `/comments/{id}` | 删除评论 | 否 |
| GET | `/visits` | 获取访问日志 | 否 |
| POST | `/visits` | 记录用户访问 | 否 |
| POST | `/guest` | 记录访客进入 | 否 |
| GET | `/owner/visits` | 获取博主访问统计 | 否 |
| GET | `/owner/today-visits` | 获取博主今日访问 | 否 |
| GET | `/auth/github` | 获取GitHub登录URL | 否 |
| GET | `/auth/github/callback` | GitHub OAuth回调 | 否 |
| POST | `/auth/github/login` | GitHub授权码登录 | 否 |

---

**注意：所有新开发的接口都必须严格按照此规范执行，确保代码质量和文档的一致性。**
