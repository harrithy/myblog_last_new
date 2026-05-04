package handler

import (
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/internal/security"
	"myblog_last_new/pkg/models"
	"net/http"
	"strings"
)

// UserHandler handles user-related requests.
type UserHandler struct {
	repo *repository.UserRepository
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
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
