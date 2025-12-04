package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
)

// AuthHandler 处理认证相关请求
type AuthHandler struct {
	userRepo  *repository.UserRepository
	ownerRepo *repository.OwnerVisitRepository
}

// NewAuthHandler 创建新的 AuthHandler
func NewAuthHandler(userRepo *repository.UserRepository, ownerRepo *repository.OwnerVisitRepository) *AuthHandler {
	return &AuthHandler{
		userRepo:  userRepo,
		ownerRepo: ownerRepo,
	}
}

// Login godoc
// @Summary 用户登录
// @Description 验证用户身份并返回一个 JWT 令牌
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   credentials   body    models.User   true  "用户凭证"
// @Success 200 {object} map[string]string "{"token": "..."}"
// @Failure 400 {string} string "无效的请求体"
// @Failure 401 {string} string "未找到用户"
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// 检查是否是博主登录
	isOwner := creds.Account == "harrio" && creds.Password == "525300@ycr"

	var user models.User
	if isOwner {
		user = models.User{
			ID:      0,
			Name:    "harrio",
			Account: "harrio",
		}

		// 异步记录博主访问
		go func() {
			if err := h.ownerRepo.RecordVisit(); err != nil {
				fmt.Printf("Failed to record owner visit: %v\n", err)
			}
		}()
	} else {
		u, err := h.userRepo.GetByEmail(creds.Account)
		if err != nil {
			if err == sql.ErrNoRows {
				response.Unauthorized(w, "User not found")
				return
			}
			response.InternalError(w, "Query failed: "+err.Error())
			return
		}
		user = *u
	}

	tokenString, err := middleware.GenerateJWT(user.Account)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	responseData := map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}
	if isOwner {
		responseData["is_owner"] = true
	}

	response.Success(w, responseData)
}
