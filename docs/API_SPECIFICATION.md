# API 接口文档规范

## 1. 基本原则

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

## 10. 分类接口

### 10.1 分类模型

```json
{
  "id": 1,
  "name": "技术",           // 必传
  "type": "folder",         // 可选，folder=文件夹，article=文章，默认folder
  "parent_id": null,        // 可选，父分类ID，null表示顶级分类
  "sort_order": 0,          // 可选，排序顺序，默认0
  "created_at": "2025-01-01 00:00:00",
  "updated_at": "2025-01-01 00:00:00",
  "children": []            // 子分类列表，查询时返回
}
```

### 10.2 创建分类

**POST** `/categories`

**请求体：**
```json
{
  "name": "技术",           // 必传，分类名称
  "type": "folder",         // 可选，folder(文件夹)或article(文章)，默认folder
  "parent_id": 1,           // 可选，父分类ID
  "sort_order": 0           // 可选，排序顺序
}
```

**响应示例：**
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

### 10.3 获取分类列表

**GET** `/categories`

| 参数 | 类型 | 必传 | 说明 |
|------|------|------|------|
| tree | bool | 否 | 是否返回树形结构，默认true |
| parent_id | int | 否 | 父分类ID，查询指定父分类下的子分类 |
| type | string | 否 | 类型筛选：folder(文件夹)或article(文章) |

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
          "parent_id": 1,
          "sort_order": 1,
          "children": []
        }
      ]
    }
  ]
}
```

### 10.4 获取单个分类

**GET** `/categories/{id}`

### 10.5 更新分类

**PUT** `/categories/{id}`

**请求体：**
```json
{
  "name": "新名称",         // 必传
  "parent_id": 2,           // 可选
  "sort_order": 1           // 可选
}
```

### 10.6 删除分类

**DELETE** `/categories/{id}`

> 注意：删除分类会级联删除所有子分类

---

**注意：所有新开发的接口都必须严格按照此规范执行，确保代码质量和文档的一致性。**
