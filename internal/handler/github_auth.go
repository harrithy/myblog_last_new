package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// GitHubConfig 存储 GitHub OAuth 配置
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// GitHubAuthHandler 处理 GitHub OAuth 认证
type GitHubAuthHandler struct {
	userRepo *repository.UserRepository
	config   GitHubConfig
}

// NewGitHubAuthHandler 创建新的 GitHubAuthHandler
// 需要设置环境变量: GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, GITHUB_REDIRECT_URI
func NewGitHubAuthHandler(userRepo *repository.UserRepository) *GitHubAuthHandler {
	return &GitHubAuthHandler{
		userRepo: userRepo,
		config: GitHubConfig{
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			RedirectURI:  getEnvOrDefault("GITHUB_REDIRECT_URI", "http://localhost:5173/callback"),
		},
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetGitHubLoginURL godoc
// @Summary 获取 GitHub 登录 URL
// @Description 返回 GitHub OAuth 授权页面的 URL
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string "{"url": "..."}"
// @Router /auth/github [get]
func (h *GitHubAuthHandler) GetGitHubLoginURL(w http.ResponseWriter, r *http.Request) {
	if h.config.ClientID == "" {
		response.InternalError(w, "GitHub OAuth not configured")
		return
	}

	authURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=user:email",
		h.config.ClientID,
		url.QueryEscape(h.config.RedirectURI),
	)

	response.Success(w, map[string]string{"url": authURL})
}

// GitHubCallback godoc
// @Summary GitHub OAuth 回调
// @Description 处理 GitHub OAuth 回调，获取用户信息并登录/注册
// @Tags auth
// @Produce json
// @Param code query string true "GitHub 授权码"
// @Success 200 {object} map[string]interface{} "{"token": "...", "user": {...}}"
// @Failure 400 {string} string "缺少授权码"
// @Failure 500 {string} string "获取 token 失败"
// @Router /auth/github/callback [get]
func (h *GitHubAuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		response.BadRequest(w, "Missing authorization code")
		return
	}

	// 用授权码换取 access token
	accessToken, err := h.exchangeCodeForToken(code)
	if err != nil {
		response.InternalError(w, "Failed to exchange code for token: "+err.Error())
		return
	}

	// 获取 GitHub 用户信息
	githubUser, err := h.getGitHubUser(accessToken)
	if err != nil {
		response.InternalError(w, "Failed to get GitHub user info: "+err.Error())
		return
	}

	// 查找或创建用户
	user, err := h.userRepo.FindOrCreateByGitHub(githubUser)
	if err != nil {
		response.InternalError(w, "Failed to find or create user: "+err.Error())
		return
	}

	// 生成 JWT token
	tokenString, err := middleware.GenerateJWT(user.Account)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	response.Success(w, map[string]interface{}{
		"token": tokenString,
		"user":  user,
	})
}

// GitHubCallbackWithCode godoc
// @Summary 使用授权码进行 GitHub 登录
// @Description 前端传递 GitHub 授权码，后端处理登录/注册
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "包含 code 的请求体"
// @Success 200 {object} map[string]interface{} "{"token": "...", "user": {...}}"
// @Failure 400 {string} string "无效的请求体"
// @Failure 500 {string} string "获取 token 失败"
// @Router /auth/github/login [post]
func (h *GitHubAuthHandler) GitHubCallbackWithCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if req.Code == "" {
		response.BadRequest(w, "Missing authorization code")
		return
	}

	// 用授权码换取 access token
	accessToken, err := h.exchangeCodeForToken(req.Code)
	if err != nil {
		response.InternalError(w, "Failed to exchange code for token: "+err.Error())
		return
	}

	// 获取 GitHub 用户信息
	githubUser, err := h.getGitHubUser(accessToken)
	if err != nil {
		response.InternalError(w, "Failed to get GitHub user info: "+err.Error())
		return
	}

	// 查找或创建用户
	user, err := h.userRepo.FindOrCreateByGitHub(githubUser)
	if err != nil {
		response.InternalError(w, "Failed to find or create user: "+err.Error())
		return
	}

	// 生成 JWT token
	tokenString, err := middleware.GenerateJWT(user.Account)
	if err != nil {
		response.InternalError(w, "Failed to generate token")
		return
	}

	response.Success(w, map[string]interface{}{
		"token": tokenString,
		"user":  user,
	})
}

// exchangeCodeForToken 用授权码换取 access token
func (h *GitHubAuthHandler) exchangeCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", h.config.ClientID)
	data.Set("client_secret", h.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", h.config.RedirectURI)

	req, err := http.NewRequest("POST", "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("GitHub OAuth response status: %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return tokenResp.AccessToken, nil
}

// getGitHubUser 获取 GitHub 用户信息
func (h *GitHubAuthHandler) getGitHubUser(accessToken string) (*models.GitHubUser, error) {
	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var user models.GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
