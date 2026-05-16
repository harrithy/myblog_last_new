package models

const (
	// CategoryTypeFolder marks a navigational folder category.
	CategoryTypeFolder = "folder"
	// CategoryTypeArticle marks an article node stored in categories.
	CategoryTypeArticle = "article"
)

// IsValidCategoryType reports whether the given category type is supported.
func IsValidCategoryType(value string) bool {
	switch value {
	case CategoryTypeFolder, CategoryTypeArticle:
		return true
	default:
		return false
	}
}

// User defines a registered user account.
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	Account   string `json:"account"`
	Nickname  string `json:"nickname"`
	Birthday  string `json:"birthday"`
	Password  string `json:"password,omitempty"`
	GitHubID  int64  `json:"github_id,omitempty"`  // GitHub user ID.
	AvatarURL string `json:"avatar_url,omitempty"` // GitHub avatar URL.
	GitHubURL string `json:"github_url,omitempty"` // GitHub profile URL.
	Bio       string `json:"bio,omitempty"`        // Optional profile bio.
}

// AuthCredentials defines the request body for password login.
type AuthCredentials struct {
	Account  string `json:"account,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password"`
}

// RegisterRequest defines the request body for public registration.
type RegisterRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email"`
	Account  string `json:"account"`
	Nickname string `json:"nickname,omitempty"`
	Birthday string `json:"birthday,omitempty"`
	Password string `json:"password"`
}

// AuthResponse defines the response body for login and registration actions.
type AuthResponse struct {
	Token   string `json:"token"`
	User    User   `json:"user"`
	IsOwner bool   `json:"is_owner,omitempty"`
}

// GitHubUser defines GitHub user info returned by OAuth APIs.
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`      // GitHub login name.
	Name      string `json:"name"`       // Display name.
	Email     string `json:"email"`      // Public email, if available.
	AvatarURL string `json:"avatar_url"` // Avatar image URL.
	HTMLURL   string `json:"html_url"`   // GitHub profile URL.
}

// APIResponse defines the standard API response envelope.
type APIResponse struct {
	Code  int         `json:"code"`
	Data  interface{} `json:"data"`
	Msg   string      `json:"msg"`
	Total int64       `json:"total,omitempty"`
	Page  int         `json:"page,omitempty"`
}

// SuccessResponse creates a successful API response.
func SuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Code: 200,
		Data: data,
		Msg:  "success",
	}
}

// ErrorResponse creates an error API response.
func ErrorResponse(code int, msg string) APIResponse {
	return APIResponse{
		Code: code,
		Data: nil,
		Msg:  msg,
	}
}

// VisitLog defines a visit log entry.
type VisitLog struct {
	ID           int        `json:"id"`
	UserNickname string     `json:"user_nickname,omitempty"`
	VisitTime    CustomTime `json:"visit_time" validate:"required"`
	Content      string     `json:"content"`
	CreatedAt    CustomTime `json:"created_at"`
}

// GuestRecord defines a guest entry record for the site.
type GuestRecord struct {
	ID        int        `json:"id"`
	EntryTime CustomTime `json:"entry_time" validate:"required"`
	Content   string     `json:"content" validate:"required"`
	CreatedAt CustomTime `json:"created_at"`
}

// OwnerVisitLog defines aggregated visit data for the site owner.
type OwnerVisitLog struct {
	ID            int        `json:"id"`
	VisitDate     CustomDate `json:"visit_date" validate:"required"`
	VisitCount    int        `json:"visit_count"`
	LastVisitTime CustomTime `json:"last_visit_time"`
	CreatedAt     CustomTime `json:"created_at"`
}

// Blog defines the response shape for article-backed blog endpoints.
type Blog struct {
	ID           int        `json:"id"`
	Title        string     `json:"title"`
	URL          string     `json:"url"`
	CategoryID   int        `json:"category_id"`
	CategoryName string     `json:"category_name,omitempty"` // Parent category name, populated on reads.
	Description  string     `json:"description"`
	CreatedAt    CustomTime `json:"created_at"`
	UpdatedAt    CustomTime `json:"updated_at"`
}

// Category defines a hierarchical category or article node.
type Category struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`                  // folder for navigation, article for content nodes.
	Description string     `json:"description,omitempty"` // Optional summary text.
	Tags        []string   `json:"tags,omitempty"`        // Optional tag list.
	URL         string     `json:"url,omitempty"`         // Article URL when Type is article.
	ImgURL      string     `json:"img_url,omitempty"`     // Cover image URL.
	ParentID    *int       `json:"parent_id,omitempty"`   // Nil means a top-level category.
	SortOrder   int        `json:"sort_order"`
	CreatedAt   CustomTime `json:"created_at"`
	UpdatedAt   CustomTime `json:"updated_at"`
	Children    []Category `json:"children,omitempty"` // Nested child categories, populated on reads.
}

// Comment defines a comment with optional nested replies.
type Comment struct {
	ID        int        `json:"id"`
	ArticleID int        `json:"article_id"`           // Article node ID from categories(type=article).
	ParentID  *int       `json:"parent_id,omitempty"`  // Nil means a root comment.
	Nickname  string     `json:"nickname"`             // Comment author nickname.
	Email     string     `json:"email,omitempty"`      // Optional contact email.
	AvatarURL string     `json:"avatar_url,omitempty"` // Optional avatar URL.
	Content   string     `json:"content"`              // Comment body.
	CreatedAt CustomTime `json:"created_at"`
	Children  []Comment  `json:"children,omitempty"` // Nested replies, populated on reads.
}

// CreateCommentRequest defines the request body for creating a comment.
type CreateCommentRequest struct {
	ArticleID int    `json:"article_id" example:"11"`                                       // Article node ID.
	ParentID  *int   `json:"parent_id,omitempty" example:"101"`                             // Optional parent comment ID.
	Nickname  string `json:"nickname" example:"Alex"`                                       // Comment author nickname.
	Email     string `json:"email,omitempty" example:"alex@example.com"`                    // Optional contact email.
	AvatarURL string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"` // Optional avatar URL.
	Content   string `json:"content" example:"Great article!"`                              // Comment body.
}
