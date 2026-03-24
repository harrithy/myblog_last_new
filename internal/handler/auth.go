package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/internal/security"
	"myblog_last_new/pkg/models"
	"net/http"
	"strings"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	userRepo  *repository.UserRepository
	ownerRepo *repository.OwnerVisitRepository
	owner     config.OwnerSettings
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userRepo *repository.UserRepository, ownerRepo *repository.OwnerVisitRepository) *AuthHandler {
	return &AuthHandler{
		userRepo:  userRepo,
		ownerRepo: ownerRepo,
		owner:     config.LoadOwnerSettings(),
	}
}

// Login godoc
// @Summary 用户登录
// @Description 使用用户名或邮箱和密码登录，返回 JWT 令牌和用户信息
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   credentials   body    models.AuthCredentials   true  "登录凭证"
// @Success 200 {object} models.AuthResponse
// @Failure 400 {string} string "无效的请求体"
// @Failure 401 {string} string "账号或密码错误"
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var creds models.AuthCredentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	login := normalizeLoginIdentifier(creds)
	password := strings.TrimSpace(creds.Password)
	if login == "" || password == "" {
		response.BadRequest(w, "Account/email and password are required")
		return
	}

	isOwner := h.owner.MatchesPasswordLogin(login, password)

	var user models.User
	if isOwner {
		user = models.User{
			ID:      0,
			Name:    h.owner.Name,
			Email:   h.owner.Email,
			Account: h.owner.Account,
		}

		go func() {
			if err := h.ownerRepo.RecordVisit(); err != nil {
				fmt.Printf("Failed to record owner visit: %v\n", err)
			}
		}()
	} else {
		storedUser, err := h.userRepo.GetByLogin(login)
		if err != nil {
			if err == sql.ErrNoRows {
				response.Unauthorized(w, "Invalid credentials")
				return
			}
			response.InternalError(w, "Query failed: "+err.Error())
			return
		}

		if !security.CheckPassword(storedUser.Password, password) {
			response.Unauthorized(w, "Invalid credentials")
			return
		}

		storedUser.Password = ""
		user = *storedUser
	}

	tokenString, err := middleware.GenerateJWT(user.Account, isOwner)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	response.Success(w, buildAuthResponse(user, tokenString, isOwner))
}

// Register godoc
// @Summary 用户注册
// @Description 使用邮箱、用户名和密码注册，注册成功后直接返回 JWT 令牌
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   user   body    models.RegisterRequest   true  "注册信息"
// @Success 201 {object} models.AuthResponse
// @Failure 400 {string} string "无效的请求体"
// @Failure 409 {string} string "邮箱或用户名已存在"
// @Router /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Account:  req.Account,
		Nickname: req.Nickname,
		Birthday: req.Birthday,
		Password: req.Password,
	}
	normalizeUserForCreate(&user)

	if msg := validateUserForCreate(user); msg != "" {
		response.BadRequest(w, msg)
		return
	}

	conflictMsg, err := getUserCreateConflict(h.userRepo, user)
	if err != nil {
		response.InternalError(w, "Failed to validate user: "+err.Error())
		return
	}
	if conflictMsg != "" {
		response.Conflict(w, conflictMsg)
		return
	}

	hashedPassword, err := security.HashPassword(user.Password)
	if err != nil {
		response.InternalError(w, "Failed to hash password")
		return
	}
	user.Password = hashedPassword

	if err := h.userRepo.Create(&user); err != nil {
		response.InternalError(w, "Failed to create user: "+err.Error())
		return
	}

	storedUser, err := h.userRepo.GetByLogin(user.Account)
	if err != nil {
		response.InternalError(w, "Failed to load created user")
		return
	}
	storedUser.Password = ""

	tokenString, err := middleware.GenerateJWT(storedUser.Account, false)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	response.Created(w, buildAuthResponse(*storedUser, tokenString, false))
}

// VerifyToken godoc
// @Summary 验证 Token
// @Description 验证 JWT Token 是否有效，并返回当前用户信息
// @Tags auth
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} response.APIResponse{data=object}
// @Failure 401 {object} response.APIResponse
// @Router /auth/verify [get]
func (h *AuthHandler) VerifyToken(w http.ResponseWriter, r *http.Request) {
	claims, err := middleware.ParseRequestToken(r)
	if err != nil {
		response.Unauthorized(w, "Token 无效或已过期")
		return
	}

	isOwner := claims.IsOwner || h.owner.IsOwnerIdentity(claims.Username)
	if isOwner {
		go h.ownerRepo.RecordVisit()
	}

	var userData map[string]interface{}
	if isOwner {
		userData = map[string]interface{}{
			"id":       0,
			"name":     h.owner.Name,
			"email":    h.owner.Email,
			"account":  h.owner.Account,
			"is_owner": true,
		}
	} else {
		user, err := h.userRepo.GetByLogin(claims.Username)
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
			"email":    user.Email,
			"account":  user.Account,
			"nickname": user.Nickname,
			"is_owner": false,
		}
	}

	userData["expires_at"] = claims.ExpiresAt.Time
	response.Success(w, userData)
}

func normalizeLoginIdentifier(creds models.AuthCredentials) string {
	if account := strings.TrimSpace(creds.Account); account != "" {
		return account
	}

	return strings.ToLower(strings.TrimSpace(creds.Email))
}

func buildAuthResponse(user models.User, token string, isOwner bool) models.AuthResponse {
	return models.AuthResponse{
		Token:   token,
		User:    user,
		IsOwner: isOwner,
	}
}
