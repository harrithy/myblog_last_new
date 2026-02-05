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
	"strings"

	"github.com/golang-jwt/jwt/v5"
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

// VerifyToken godoc
// @Summary 验证 Token 有效性
// @Description 验证 JWT Token 是否有效，返回当前用户信息
// @Tags auth
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} response.APIResponse{data=object} "Token 有效，返回用户信息"
// @Failure 401 {object} response.APIResponse "Token 无效或已过期"
// @Router /auth/verify [get]
func (h *AuthHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	// 获取 Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		response.Unauthorized(w, "缺少 Token")
		return
	}

	// 解析 token
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims := &middleware.Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("my_secret_key"), nil
	})

	if err != nil || !token.Valid {
		response.Unauthorized(w, "Token 无效或已过期")
		return
	}

	// 检查是否是博主（支持普通登录和 GitHub 登录）
	// 普通登录：account == "harrio"
	// GitHub 登录：account == "github_156180607"（博主的 GitHub ID）
	// 邮箱登录：account == "harrithy@github.com"
	isOwner := claims.Username == "harrio" || claims.Username == "github_156180607" || claims.Username == "harrithy@github.com"

	var userData map[string]interface{}
	if isOwner {
		userData = map[string]interface{}{
			"id":       0,
			"name":     "harrio",
			"account":  "harrio",
			"is_owner": true,
		}
	} else {
		// 查询用户信息
		user, err := h.userRepo.GetByEmail(claims.Username)
		if err != nil {
			if err == sql.ErrNoRows {
				response.Unauthorized(w, "用户不存在")
				return
			}
			response.InternalError(w, "查询用户失败")
			return
		}
		userData = map[string]interface{}{
			"id":       user.ID,
			"name":     user.Name,
			"account":  user.Account,
			"nickname": user.Nickname,
			"is_owner": false,
		}
	}

	// 返回过期时间
	userData["expires_at"] = claims.ExpiresAt.Time

	response.Success(w, userData)
}
