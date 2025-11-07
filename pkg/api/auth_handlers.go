package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/pkg/auth"
	"myblog_last_new/pkg/models"
	"net/http"
	"time"
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
		response := models.ErrorResponse(400, "Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查是否是博客主人登录
	isOwner := creds.Account == "harrio" && creds.Password == "525300@ycr"
	
	var user models.User
	if isOwner {
		// 博客主人登录，使用特殊处理
		user = models.User{
			ID:      0,
			Name:    "harrio",
			Account: "harrio",
		}
		
		// 记录博客主人访问
		go recordOwnerVisit(db)
	} else {
		// 普通用户登录验证
		err = db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", creds.Account).Scan(&user.ID, &user.Name, &user.Account)
		if err != nil {
			if err == sql.ErrNoRows {
				response := models.ErrorResponse(401, "User not found")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(response)
				return
			}
			response := models.ErrorResponse(500, "Query failed: "+err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	tokenString, err := auth.GenerateJWT(user.Account)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to generate token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	responseData := map[string]interface{}{
		"token": tokenString,
		"user":  user,
	}
	if isOwner {
		responseData["is_owner"] = true
	}

	response := models.SuccessResponse(responseData)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// recordOwnerVisit records owner visit to database
func recordOwnerVisit(db *sql.DB) {
	today := time.Now().Format("2006-01-02")
	
	// 使用 INSERT ... ON DUPLICATE KEY UPDATE 来更新或插入访问记录
	query := `
		INSERT INTO owner_visit_logs (visit_date, visit_count) 
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE 
		visit_count = visit_count + 1,
		last_visit_time = CURRENT_TIMESTAMP
	`
	
	_, err := db.Exec(query, today)
	if err != nil {
		fmt.Printf("Failed to record owner visit: %v\n", err)
	}
}
