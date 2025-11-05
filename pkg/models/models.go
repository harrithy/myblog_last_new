package models

// User defines user model
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Birthday string `json:"birthday"`
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
	ID         int        `json:"id"`
	EntryTime  CustomTime `json:"entry_time" validate:"required"`
	Content    string     `json:"content" validate:"required"`
	CreatedAt  CustomTime `json:"created_at"`
}
