package handler

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/middleware"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/internal/security"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// UserHandler handles user-related requests.
type UserHandler struct {
	repo  *repository.UserRepository
	owner config.OwnerSettings
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		repo:  repo,
		owner: config.LoadOwnerSettings(),
	}
}

// GetUsers godoc
// @Summary List users
// @Description Returns all registered users.
// @Tags users
// @Produce  json
// @Success 200 {array} models.User
// @Failure 500 {string} string "Query failed"
// @Router /users [get]
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.GetAll()
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, users)
}

// AddUser godoc
// @Summary Add a user
// @Description Creates a new user record. This endpoint requires owner authentication.
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user     body    models.User   true  "User payload"
// @Success 201 {object} models.User
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Create failed"
// @Security ApiKeyAuth
// @Router /users [post]
func (h *UserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	normalizeUserForCreate(&u)

	if msg := validateUserForCreate(u); msg != "" {
		response.BadRequest(w, msg)
		return
	}

	conflictMsg, err := getUserCreateConflict(h.repo, u)
	if err != nil {
		response.InternalError(w, "Failed to validate user: "+err.Error())
		return
	}
	if conflictMsg != "" {
		response.Conflict(w, conflictMsg)
		return
	}

	hashedPassword, err := security.HashPassword(u.Password)
	if err != nil {
		response.InternalError(w, "Failed to hash password")
		return
	}
	u.Password = hashedPassword

	if err := h.repo.Create(&u); err != nil {
		response.InternalError(w, "Failed to create user: "+err.Error())
		return
	}

	u.Password = ""
	response.Created(w, u)
}

// GetUserByID returns one user by ID for the current authenticated user or owner.
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(r.URL.Path)
	if !ok {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	if !h.authorizeUserAccess(w, r, userID) {
		return
	}

	user, err := h.repo.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, user)
}

// UpdateUser updates one user profile by ID for the current authenticated user or owner.
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(r.URL.Path)
	if !ok {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	if !h.authorizeUserAccess(w, r, userID) {
		return
	}

	currentUser, err := h.repo.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "User not found")
			return
		}
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	type userUpdateRequest struct {
		Name      *string `json:"name"`
		Username  *string `json:"username"`
		Email     *string `json:"email"`
		Nickname  *string `json:"nickname"`
		Avatar    *string `json:"avatar"`
		AvatarURL *string `json:"avatar_url"`
		Bio       *string `json:"bio"`
		Birthday  *string `json:"birthday"`
	}

	var req userUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	updatedUser := *currentUser

	if req.Name != nil {
		updatedUser.Name = strings.TrimSpace(*req.Name)
	}
	if req.Username != nil && updatedUser.Name == "" {
		updatedUser.Name = strings.TrimSpace(*req.Username)
	}
	if req.Email != nil {
		updatedUser.Email = strings.ToLower(strings.TrimSpace(*req.Email))
	}
	if req.Nickname != nil {
		updatedUser.Nickname = strings.TrimSpace(*req.Nickname)
	}
	if req.AvatarURL != nil {
		updatedUser.AvatarURL = strings.TrimSpace(*req.AvatarURL)
	}
	if req.Avatar != nil && updatedUser.AvatarURL == currentUser.AvatarURL {
		updatedUser.AvatarURL = strings.TrimSpace(*req.Avatar)
	}
	if req.Bio != nil {
		updatedUser.Bio = strings.TrimSpace(*req.Bio)
	}
	if req.Birthday != nil {
		updatedUser.Birthday = strings.TrimSpace(*req.Birthday)
	}

	if updatedUser.Name == "" {
		updatedUser.Name = updatedUser.Account
	}
	if updatedUser.Nickname == "" {
		updatedUser.Nickname = updatedUser.Name
	}
	if updatedUser.Email == "" {
		response.BadRequest(w, "Email is required")
		return
	}

	emailExists, err := h.repo.ExistsByEmailExcludingID(updatedUser.Email, updatedUser.ID)
	if err != nil {
		response.InternalError(w, "Failed to validate email uniqueness")
		return
	}
	if emailExists {
		response.Conflict(w, "Email already exists")
		return
	}

	rowsAffected, err := h.repo.UpdateByID(&updatedUser)
	if err != nil {
		response.InternalError(w, "Failed to update user: "+err.Error())
		return
	}
	if rowsAffected == 0 {
		response.NotFound(w, "User not found")
		return
	}

	response.Success(w, updatedUser)
}

// DeleteUser deletes one user by ID for the current authenticated user or owner.
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := extractUserID(r.URL.Path)
	if !ok {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	if !h.authorizeUserAccess(w, r, userID) {
		return
	}

	rowsAffected, err := h.repo.DeleteByID(userID)
	if err != nil {
		response.InternalError(w, "Failed to delete user: "+err.Error())
		return
	}
	if rowsAffected == 0 {
		response.NotFound(w, "User not found")
		return
	}

	response.Success(w, map[string]int{"deleted_id": userID})
}

func normalizeUserForCreate(u *models.User) {
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Account = strings.TrimSpace(u.Account)
	u.Nickname = strings.TrimSpace(u.Nickname)
	u.Birthday = strings.TrimSpace(u.Birthday)
	u.Password = strings.TrimSpace(u.Password)

	if u.Email == "" && strings.Contains(u.Account, "@") {
		u.Email = strings.ToLower(u.Account)
	}
	if u.Name == "" {
		u.Name = u.Account
	}
	if u.Nickname == "" {
		u.Nickname = u.Name
	}
}

func validateUserForCreate(u models.User) string {
	if u.Email == "" {
		return "Email is required"
	}
	if u.Account == "" {
		return "Account is required"
	}
	if u.Password == "" {
		return "Password is required"
	}
	if u.Name == "" {
		return "Name is required"
	}

	return ""
}

func getUserCreateConflict(repo *repository.UserRepository, u models.User) (string, error) {
	emailExists, err := repo.ExistsByEmail(u.Email)
	if err != nil {
		return "", err
	}
	if emailExists {
		return "Email already exists", nil
	}

	accountExists, err := repo.ExistsByAccount(u.Account)
	if err != nil {
		return "", err
	}
	if accountExists {
		return "Account already exists", nil
	}

	return "", nil
}

func extractUserID(path string) (int, bool) {
	trimmedPath := strings.TrimPrefix(path, "/api")
	idStr := strings.TrimPrefix(trimmedPath, "/users/")
	idStr = strings.TrimSuffix(idStr, "/")
	if idStr == "" {
		return 0, false
	}

	userID, err := strconv.Atoi(idStr)
	if err != nil || userID < 1 {
		return 0, false
	}

	return userID, true
}

func (h *UserHandler) authorizeUserAccess(w http.ResponseWriter, r *http.Request, userID int) bool {
	claims, err := middleware.ParseRequestToken(r)
	if err != nil {
		response.Unauthorized(w, "Invalid token")
		return false
	}

	if claims.IsOwner || h.owner.IsOwnerIdentity(claims.Username) {
		return true
	}

	user, err := h.repo.GetByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "User not found")
			return false
		}
		response.InternalError(w, "Failed to load user")
		return false
	}

	if claims.Username != user.Account && claims.Username != user.Email {
		response.Forbidden(w, "You can only access your own profile")
		return false
	}

	return true
}
