package api

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/models"
	"net/http"
)

// GetUsers handles user list requests
// GetUsers godoc
// @Summary 获取所有用户
// @Description 获取所有已注册用户的列表。
// @Tags users
// @Produce  json
// @Success 200 {array} models.User
// @Failure 500 {string} string "查询失败或数据解析失败"
// @Router /users [get]
func GetUsers(w http.ResponseWriter, _ *http.Request, db *sql.DB) {
	// Query the database
	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		http.Error(w, "Query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Account); err != nil {
			http.Error(w, "Data parse failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// AddUser handles the creation of a new user
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
func AddUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	stmt, err := db.Prepare("INSERT INTO users(name, email, nickname, birthday) VALUES(?, ?, ?, ?)")
	if err != nil {
		http.Error(w, "Failed to prepare statement: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(u.Name, u.Account, u.Nickname, u.Birthday)
	if err != nil {
		http.Error(w, "Failed to execute statement: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}
