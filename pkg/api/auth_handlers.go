package api

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/auth"
	"myblog_last_new/pkg/models"
	"net/http"
)

// Login godoc
// @Summary 用户登录
// @Description 验证用户身份并返回一个 JWT 令牌。
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   credentials   body    models.User   true  "用户凭证"
// @Success 200 {object} map[string]string "{"token": "..."}"
// @Failure 400 {string} string "无效的请求体"
// @Failure 401 {string} string "未找到用户"
// @Failure 500 {string} string "查询失败或生成令牌失败"
// @Router /login [post]
func Login(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var creds models.User
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", creds.Email).Scan(&user.ID, &user.Name, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Query failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tokenString, err := auth.GenerateJWT(user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}
