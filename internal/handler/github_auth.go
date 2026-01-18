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

// OwnerGitHubID 博主的 GitHub ID
const OwnerGitHubID int64 = 156180607

// GitHubAuthHandler 处理 GitHub OAuth 认证
type GitHubAuthHandler struct {
	userRepo  *repository.UserRepository
	ownerRepo *repository.OwnerVisitRepository
	config    GitHubConfig
}

// NewGitHubAuthHandler 创建新的 GitHubAuthHandler
// 需要设置环境变量: GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, GITHUB_REDIRECT_URI
func NewGitHubAuthHandler(userRepo *repository.UserRepository, ownerRepo *repository.OwnerVisitRepository) *GitHubAuthHandler {
	return &GitHubAuthHandler{
		userRepo:  userRepo,
		ownerRepo: ownerRepo,
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

	// 检查是否是博主登录，记录访问统计
	isOwner := githubUser.ID == OwnerGitHubID
	if isOwner {
		go func() {
			if err := h.ownerRepo.RecordVisit(); err != nil {
				fmt.Printf("Failed to record owner visit: %v\n", err)
			}
		}()
	}

	// 生成 JWT token
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

	// 检查是否是博主登录，记录访问统计
	isOwner := githubUser.ID == OwnerGitHubID
	if isOwner {
		go func() {
			if err := h.ownerRepo.RecordVisit(); err != nil {
				fmt.Printf("Failed to record owner visit: %v\n", err)
			}
		}()
	}

	// 生成 JWT token
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

// OwnerGitHubUsername 博主的 GitHub 用户名
const OwnerGitHubUsername = "harrithy"

// GitHubRepo GitHub 仓库信息
type GitHubRepo struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	HTMLURL     string   `json:"html_url"`
	Homepage    string   `json:"homepage"`
	Language    string   `json:"language"`
	Stars       int      `json:"stargazers_count"`
	Forks       int      `json:"forks_count"`
	Watchers    int      `json:"watchers_count"`
	OpenIssues  int      `json:"open_issues_count"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	PushedAt    string   `json:"pushed_at"`
	Fork        bool     `json:"fork"`
	Private     bool     `json:"private"`
	Topics      []string `json:"topics"`
}

// GetOwnerRepos godoc
// @Summary 获取博主的 GitHub 开源项目
// @Description 获取博主 GitHub 账号下的所有公开仓库
// @Tags github
// @Produce json
// @Param sort query string false "排序方式: created, updated, pushed, full_name" default(updated)
// @Param per_page query int false "每页数量" default(30)
// @Success 200 {object} response.APIResponse{data=[]GitHubRepo}
// @Router /github/repos [get]
func (h *GitHubAuthHandler) GetOwnerRepos(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "updated"
	}

	perPage := r.URL.Query().Get("per_page")
	if perPage == "" {
		perPage = "30"
	}

	apiURL := fmt.Sprintf(
		"https://api.github.com/users/%s/repos?sort=%s&per_page=%s&type=owner",
		OwnerGitHubUsername, sort, perPage,
	)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		response.InternalError(w, "Failed to create request: "+err.Error())
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "MyBlog-App")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		response.InternalError(w, "Failed to fetch repos: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		response.InternalError(w, "Failed to read response: "+err.Error())
		return
	}

	if resp.StatusCode != http.StatusOK {
		response.Error(w, resp.StatusCode, resp.StatusCode, "GitHub API error: "+string(body))
		return
	}

	var repos []GitHubRepo
	if err := json.Unmarshal(body, &repos); err != nil {
		response.InternalError(w, "Failed to parse repos: "+err.Error())
		return
	}

	// 过滤掉 fork 的仓库，只返回原创项目
	var originalRepos []GitHubRepo
	for _, repo := range repos {
		if !repo.Fork && !repo.Private {
			originalRepos = append(originalRepos, repo)
		}
	}

	response.Success(w, originalRepos)
}
