package api

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/models"
	"net/http"
	"time"
)

// GetOwnerVisitStats godoc
// @Summary 获取博客主人访问统计
// @Description 获取博客主人指定天数内每天访问次数的统计信息，包含访问日期、访问次数和最后访问时间
// @Tags owner
// @Accept  json
// @Produce  json
// @Param   days   query    int     false  "获取最近多少天的数据，默认7天，最大365天" 
// @Success 200 {object} models.APIResponse{data=object} "获取成功"
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /owner/visits [get]
func GetOwnerVisitStats(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// 获取查询参数
	days := r.URL.Query().Get("days")
	if days == "" {
		days = "7"
	}

	query := `
		SELECT visit_date, visit_count, last_visit_time 
		FROM owner_visit_logs 
		WHERE visit_date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		ORDER BY visit_date DESC
	`

	rows, err := db.Query(query, days)
	if err != nil {
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var visitStats []models.OwnerVisitLog
	for rows.Next() {
		var stat models.OwnerVisitLog
		err := rows.Scan(&stat.VisitDate, &stat.VisitCount, &stat.LastVisitTime)
		if err != nil {
			response := models.ErrorResponse(500, "Data scan failed: "+err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		visitStats = append(visitStats, stat)
	}

	// 计算总访问次数
	totalQuery := `
		SELECT COALESCE(SUM(visit_count), 0) 
		FROM owner_visit_logs 
		WHERE visit_date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
	`
	
	var totalVisits int
	err = db.QueryRow(totalQuery, days).Scan(&totalVisits)
	if err != nil {
		response := models.ErrorResponse(500, "Total query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"visit_stats":  visitStats,
		"total_visits": totalVisits,
		"days":         days,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SuccessResponse(response))
}

// GetOwnerTodayVisits godoc
// @Summary 获取博客主人今日访问次数
// @Description 获取博客主人今天的访问次数统计
// @Tags owner
// @Accept  json
// @Produce  json
// @Success 200 {object} models.APIResponse{data=object} "获取成功"
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /owner/today-visits [get]
func GetOwnerTodayVisits(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	today := time.Now().Format("2006-01-02")
	
	query := `
		SELECT COALESCE(visit_count, 0) 
		FROM owner_visit_logs 
		WHERE visit_date = ?
	`
	
	var todayVisits int
	err := db.QueryRow(query, today).Scan(&todayVisits)
	if err != nil && err != sql.ErrNoRows {
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"date":         today,
		"today_visits": todayVisits,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.SuccessResponse(response))
}
