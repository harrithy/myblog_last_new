package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHubConfig stores GitHub OAuth settings.
type GitHubConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// GitHubAuthHandler handles GitHub OAuth requests.
type GitHubAuthHandler struct {
	userRepo  *repository.UserRepository
	ownerRepo *repository.OwnerVisitRepository
	config    GitHubConfig
	owner     config.OwnerSettings
}

// NewGitHubAuthHandler creates a new GitHubAuthHandler.
func NewGitHubAuthHandler(userRepo *repository.UserRepository, ownerRepo *repository.OwnerVisitRepository) *GitHubAuthHandler {
	return &GitHubAuthHandler{
		userRepo:  userRepo,
		ownerRepo: ownerRepo,
		owner:     config.LoadOwnerSettings(),
		config: GitHubConfig{
			ClientID:     config.GetEnv("GITHUB_CLIENT_ID", ""),
			ClientSecret: config.GetEnv("GITHUB_CLIENT_SECRET", ""),
			RedirectURI:  config.GetEnv("GITHUB_REDIRECT_URI", "http://localhost:5173/callback"),
		},
	}
}

// GetGitHubLoginURL godoc
// @Summary Get GitHub login URL
// @Description Returns the GitHub OAuth authorization URL.
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /auth/github [get]
func (h *GitHubAuthHandler) GetGitHubLoginURL(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.config.ClientID) == "" {
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
// @Summary Handle GitHub OAuth callback
// @Description Exchanges the GitHub authorization code, loads the GitHub user, and signs the user in.
// @Tags auth
// @Produce json
// @Param code query string true "GitHub authorization code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Missing authorization code"
// @Failure 500 {string} string "GitHub login failed"
// @Router /auth/github/callback [get]
func (h *GitHubAuthHandler) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		response.BadRequest(w, "Missing authorization code")
		return
	}

	h.loginWithGitHubCode(w, code)
}

// GitHubCallbackWithCode godoc
// @Summary Sign in with a GitHub code
// @Description Accepts a GitHub authorization code from the frontend and completes the login flow.
// @Tags auth
// @Accept json
// @Produce json
// @Param body body object true "JSON payload containing a GitHub authorization code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "GitHub login failed"
// @Router /auth/github/login [post]
func (h *GitHubAuthHandler) GitHubCallbackWithCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		response.BadRequest(w, "Missing authorization code")
		return
	}

	h.loginWithGitHubCode(w, code)
}

func (h *GitHubAuthHandler) loginWithGitHubCode(w http.ResponseWriter, code string) {
	accessToken, err := h.exchangeCodeForToken(code)
	if err != nil {
		response.InternalError(w, "Failed to exchange code for token: "+err.Error())
		return
	}

	githubUser, err := h.getGitHubUser(accessToken)
	if err != nil {
		response.InternalError(w, "Failed to get GitHub user info: "+err.Error())
		return
	}

	user, err := h.userRepo.FindOrCreateByGitHub(githubUser)
	if err != nil {
		response.InternalError(w, "Failed to find or create user: "+err.Error())
		return
	}

	isOwner := h.isOwnerGitHubUser(githubUser)
	if isOwner {
		go func() {
			if err := h.ownerRepo.RecordVisit(); err != nil {
				fmt.Printf("Failed to record owner visit: %v\n", err)
			}
		}()
	}

	tokenString, err := middleware.GenerateJWT(user.Account, isOwner)
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

func (h *GitHubAuthHandler) isOwnerGitHubUser(user *models.GitHubUser) bool {
	return h.owner.GitHubID > 0 && user != nil && user.ID == h.owner.GitHubID
}

// exchangeCodeForToken exchanges an authorization code for an access token.
func (h *GitHubAuthHandler) exchangeCodeForToken(code string) (string, error) {
	data := url.Values{}
	data.Set("client_id", h.config.ClientID)
	data.Set("client_secret", h.config.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", h.config.RedirectURI)

	req, err := http.NewRequest(http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		if tokenResp.Error != "" {
			return "", fmt.Errorf("github token exchange failed: %s: %s", tokenResp.Error, tokenResp.ErrorDesc)
		}
		return "", fmt.Errorf("github token exchange failed with status %d", resp.StatusCode)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("github token exchange returned an empty access token")
	}

	return tokenResp.AccessToken, nil
}

// getGitHubUser fetches GitHub user info by access token.
func (h *GitHubAuthHandler) getGitHubUser(accessToken string) (*models.GitHubUser, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var user models.GitHubUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	if user.ID == 0 || strings.TrimSpace(user.Login) == "" {
		return nil, fmt.Errorf("github user response missing required identity fields")
	}

	return &user, nil
}

// GitHubRepo defines repository data returned from GitHub.
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
// @Summary List the owner's public GitHub repositories
// @Description Returns public, non-fork repositories owned by the configured GitHub account.
// @Tags github
// @Produce json
// @Param sort query string false "Sort mode: created, updated, pushed, full_name" default(updated)
// @Param per_page query int false "Page size" default(30)
// @Success 200 {object} response.APIResponse{data=[]GitHubRepo}
// @Router /github/repos [get]
func (h *GitHubAuthHandler) GetOwnerRepos(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.owner.GitHubUsername) == "" {
		response.InternalError(w, "Owner GitHub username is not configured")
		return
	}

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
		h.owner.GitHubUsername, sort, perPage,
	)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
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

	originalRepos := make([]GitHubRepo, 0, len(repos))
	for _, repo := range repos {
		if !repo.Fork && !repo.Private {
			originalRepos = append(originalRepos, repo)
		}
	}

	response.Success(w, originalRepos)
}
