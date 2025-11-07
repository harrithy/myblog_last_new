package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"time"
)

// LogVisit handles logging a user visit
// LogVisit godoc
// @Summary 记录用户访问
// @Description 记录一次新的用户访问。用户昵称是可选的，访问时间是必传的，内容是可选的。
// @Tags visits
// @Accept  json
// @Produce  json
// @Param   visit   body    models.VisitLog   true  "访问信息"
// @Success 201 {object} models.APIResponse{data=models.VisitLog}
// @Router /visits [post]
func LogVisit(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var visit models.VisitLog
	if err := json.NewDecoder(r.Body).Decode(&visit); err != nil {
		response := models.ErrorResponse(400, "Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查访问时间是否必传
	if visit.VisitTime.IsZero() {
		response := models.ErrorResponse(400, "Visit time is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 如果没有提供content，使用默认值
	if visit.Content == "" {
		visit.Content = "普通访问记录"
	}

	// The UnmarshalJSON for CustomTime already handles parsing, so we can directly use the time.
	// 转换为MySQL DATETIME格式
	mysqlTimeFormat := visit.VisitTime.Format("2006-01-02 15:04:05")

	// 准备插入语句，用户昵称是可选的，content现在也支持
	stmt, err := db.Prepare("INSERT INTO visit_logs (user_nickname, visit_time, content) VALUES (?, ?, ?)")
	if err != nil {
		response := models.ErrorResponse(500, "Failed to prepare statement: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(visit.UserNickname, mysqlTimeFormat, visit.Content)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to execute statement: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 清理旧数据，只保留最新的20条
	go cleanupOldVisitLogs(db, 20)

	id, _ := result.LastInsertId()
	visit.ID = int(id)
	// visit.VisitTime is already set from the request body
	// 如果没有提供CreatedAt，让数据库自动生成
	if visit.CreatedAt.IsZero() {
		visit.CreatedAt = models.CustomTime{Time: time.Now()}
	}

	response := models.SuccessResponse(visit)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetVisitLogs handles retrieving all visit logs
// GetVisitLogs godoc
// @Summary 获取所有访问日志
// @Description 检索所有用户访问日志的列表，按访问时间降序排列。
// @Tags visits
// @Produce  json
// @Success 200 {object} models.APIResponse{data=[]models.VisitLog}
// @Failure 400 {object} models.APIResponse
// @Router /visits [get]
func GetVisitLogs(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pagesize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var total int64
	err = db.QueryRow("SELECT COUNT(*) FROM visit_logs").Scan(&total)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to get total count: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	query := "SELECT id, user_nickname, visit_time, IFNULL(content, '普通访问记录'), created_at FROM visit_logs ORDER BY visit_time DESC, id DESC LIMIT ? OFFSET ?"
	rows, err := db.Query(query, pageSize, offset)
	if err != nil {
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var visits []models.VisitLog
	for rows.Next() {
		var v models.VisitLog
		if err := rows.Scan(&v.ID, &v.UserNickname, &v.VisitTime, &v.Content, &v.CreatedAt); err != nil {
			response := models.ErrorResponse(500, "Data parse failed: "+err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		visits = append(visits, v)
	}

	response := models.SuccessResponse(visits)
	response.Total = total
	response.Page = page
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// LogGuestRecord handles logging a guest record with entry time and content
// LogGuestRecord godoc
// @Summary 记录访客进入信息
// @Description 记录访客进入网站的时间和内容信息
// @Tags guest
// @Accept  json
// @Produce  json
// @Param   record   body    models.GuestRecord   true  "访客记录信息"
// @Success 201 {object} models.APIResponse{data=models.GuestRecord}
// @Router /guest [post]
func LogGuestRecord(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var record models.GuestRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		response := models.ErrorResponse(400, "Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 检查必填字段
	if record.EntryTime.IsZero() {
		response := models.ErrorResponse(400, "Entry time is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if record.Content == "" {
		response := models.ErrorResponse(400, "Content is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 转换为MySQL DATETIME格式
	mysqlTimeFormat := record.EntryTime.Format("2006-01-02 15:04:05")

	// 准备插入语句
	stmt, err := db.Prepare("INSERT INTO guest_records (entry_time, content) VALUES (?, ?)")
	if err != nil {
		response := models.ErrorResponse(500, "Failed to prepare statement: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(mysqlTimeFormat, record.Content)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to execute statement: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	id, _ := result.LastInsertId()
	record.ID = int(id)
	
	// 设置创建时间
	if record.CreatedAt.IsZero() {
		record.CreatedAt = models.CustomTime{Time: time.Now()}
	}

	response := models.SuccessResponse(record)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// cleanupOldVisitLogs 清理旧的访问日志，只保留指定数量的最新记录
func cleanupOldVisitLogs(db *sql.DB, keepCount int) {
	// 获取当前记录总数
	var totalCount int
	err := db.QueryRow("SELECT COUNT(*) FROM visit_logs").Scan(&totalCount)
	if err != nil {
		fmt.Printf("Failed to count visit logs: %v\n", err)
		return
	}

	// 如果记录数超过要保留的数量，删除旧记录
	if totalCount > keepCount {
		// 使用子查询获取要保留的最新记录的最小ID
		deleteQuery := `
			DELETE FROM visit_logs 
			WHERE id < (
				SELECT min_id FROM (
					SELECT MIN(id) as min_id FROM (
						SELECT id FROM visit_logs 
						ORDER BY created_at DESC 
						LIMIT ?
					) as keep_records
				) as temp
			)
		`
		
		result, err := db.Exec(deleteQuery, keepCount)
		if err != nil {
			fmt.Printf("Failed to delete old visit logs: %v\n", err)
			return
		}
		
		affected, _ := result.RowsAffected()
		fmt.Printf("Cleaned up %d old visit log records, keeping latest %d\n", affected, keepCount)
	}
}
