package handler

import (
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
)

// UserHandler 处理用户相关请求
type UserHandler struct {
	repo *repository.UserRepository
}

// NewUserHandler 创建新的 UserHandler
func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

// GetUsers godoc
// @Summary 获取所有用户
// @Description 获取所有已注册用户的列表。
// @Tags users
// @Produce  json
// @Success 200 {array} models.User
// @Failure 500 {string} string "查询失败或数据解析失败"
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
// @Summary 添加一个新用户
// @Description 在系统中创建一个新用户。此操作需要身份验证。
// @Tags users
// @Accept  json
// @Produce  json
// @Param   user     body    models.User   true  "要添加的用户"
// @Success 201 {object} models.User
// @Failure 400 {string} string "无效的请求体"
// @Failure 500 {string} string "准备或执行语句失败"
// @Security ApiKeyAuth
// @Router /users [post]
func (h *UserHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if err := h.repo.Create(&u); err != nil {
		response.InternalError(w, "Failed to create user: "+err.Error())
		return
	}

	response.Created(w, u)
}
