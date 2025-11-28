package models

// User defines user model
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Account  string `json:"account"`
	Nickname string `json:"nickname"`
	Birthday string `json:"birthday"`
	Password string `json:"password,omitempty"`
}

// APIResponse defines the standard API response format
type APIResponse struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Msg   string      `json:"msg"`
	Total int64       `json:"total,omitempty"`
	Page  int         `json:"page,omitempty"`
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Code: 200,
		Data: data,
		Msg:  "success",
	}
}

// ErrorResponse creates an error API response
func ErrorResponse(code int, msg string) APIResponse {
	return APIResponse{
		Code: code,
		Data: nil,
		Msg:  msg,
	}
}

// VisitLog defines visit log model
type VisitLog struct {
	ID           int        `json:"id"`
	UserNickname string     `json:"user_nickname,omitempty"`
	VisitTime    CustomTime `json:"visit_time" validate:"required"`
	Content      string     `json:"content"`
	CreatedAt    CustomTime `json:"created_at"`
}

// GuestRecord defines guest record model for tracking website entries
type GuestRecord struct {
	ID        int        `json:"id"`
	EntryTime CustomTime `json:"entry_time" validate:"required"`
	Content   string     `json:"content" validate:"required"`
	CreatedAt CustomTime `json:"created_at"`
}

// OwnerVisitLog defines owner visit log model
type OwnerVisitLog struct {
	ID            int        `json:"id"`
	VisitDate     CustomDate `json:"visit_date" validate:"required"`
	VisitCount    int        `json:"visit_count"`
	LastVisitTime CustomTime `json:"last_visit_time"`
	CreatedAt     CustomTime `json:"created_at"`
}

// Blog defines blog metadata
type Blog struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	CategoryID   int        `json:"category_id"`
	CategoryName string     `json:"category_name,omitempty"` // 查询时填充
	Description  string     `json:"description"`
	CreatedAt    CustomTime `json:"created_at"`
	UpdatedAt    CustomTime `json:"updated_at"`
}

// Category defines category model with parent-child relationship
type Category struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`                // folder=文件夹, article=文章
	URL       string     `json:"url,omitempty"`       // 文章类型时存储文章地址
	ParentID  *int       `json:"parent_id,omitempty"` // nil表示顶级分类
	SortOrder int        `json:"sort_order"`
	CreatedAt CustomTime `json:"created_at"`
	UpdatedAt CustomTime `json:"updated_at"`
	Children  []Category `json:"children,omitempty"` // 子分类列表，查询时填充
}

// Comment defines comment model with parent-child relationship
type Comment struct {
	ID        int        `json:"id"`
	ArticleID int        `json:"article_id"`          // 关联的文章ID（分类ID，type=article）
	ParentID  *int       `json:"parent_id,omitempty"` // 父评论ID，nil表示顶级评论
	Nickname  string     `json:"nickname"`            // 评论者昵称
	Email     string     `json:"email,omitempty"`     // 评论者邮箱（可选）
	Content   string     `json:"content"`             // 评论内容
	CreatedAt CustomTime `json:"created_at"`
	Children  []Comment  `json:"children,omitempty"` // 子评论列表，查询时填充
}
