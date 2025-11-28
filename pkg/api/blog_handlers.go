package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"myblog_last_new/pkg/models"
)

// GetBlogs godoc
// @Summary 获取博客列表
// @Description 根据分类ID分页获取博客列表。
// @Tags blogs
// @Produce  json
// @Param   category_id query    int     false  "分类ID"
// @Param   page        query    int     false  "页码，默认1"
// @Param   pagesize    query    int     false  "每页数量，默认10"
// @Success 200 {object} models.APIResponse{data=[]models.Blog}
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /blogs [get]
func GetBlogs(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pagesize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	categoryIDStr := r.URL.Query().Get("category_id")

	offset := (page - 1) * pageSize

	queryArgs := make([]interface{}, 0)
	whereClause := ""
	if categoryIDStr != "" {
		categoryID, err := strconv.Atoi(categoryIDStr)
		if err == nil {
			whereClause = "WHERE b.category_id = ?"
			queryArgs = append(queryArgs, categoryID)
		}
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM blogs b " + whereClause
	err = db.QueryRow(countQuery, queryArgs...).Scan(&total)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to get total count: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	dataQuery := `
		SELECT b.id, b.title, b.url, b.category_id, IFNULL(c.name, ''), IFNULL(b.description, ''), b.created_at, b.updated_at 
		FROM blogs b 
		LEFT JOIN categories c ON b.category_id = c.id 
		` + whereClause + ` ORDER BY b.created_at DESC LIMIT ? OFFSET ?`
	dataArgs := append(queryArgs, pageSize, offset)
	rows, err := db.Query(dataQuery, dataArgs...)
	if err != nil {
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var blogs []models.Blog
	for rows.Next() {
		var blog models.Blog
		if err := rows.Scan(&blog.ID, &blog.Title, &blog.URL, &blog.CategoryID, &blog.CategoryName, &blog.Description, &blog.CreatedAt, &blog.UpdatedAt); err != nil {
			response := models.ErrorResponse(500, "Data parse failed: "+err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		blogs = append(blogs, blog)
	}

	response := models.SuccessResponse(blogs)
	response.Total = total
	response.Page = page

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetBlogDetail godoc
// @Summary 获取博客详情
// @Description 根据ID获取单个博客的详细信息。
// @Tags blogs
// @Produce  json
// @Param   id   path    int     true  "博客ID"
// @Success 200 {object} models.APIResponse{data=models.Blog}
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 404 {object} models.APIResponse "博客不存在"
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /blogs/{id} [get]
func GetBlogDetail(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	idStr := strings.TrimPrefix(r.URL.Path, "/blogs/")
	if idStr == "" {
		response := models.ErrorResponse(400, "Blog ID is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	blogID, err := strconv.Atoi(idStr)
	if err != nil {
		response := models.ErrorResponse(400, "Invalid blog ID")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var blog models.Blog
	query := `
		SELECT b.id, b.title, b.url, b.category_id, IFNULL(c.name, ''), IFNULL(b.description, ''), b.created_at, b.updated_at 
		FROM blogs b 
		LEFT JOIN categories c ON b.category_id = c.id 
		WHERE b.id = ?`
	err = db.QueryRow(query, blogID).Scan(&blog.ID, &blog.Title, &blog.URL, &blog.CategoryID, &blog.CategoryName, &blog.Description, &blog.CreatedAt, &blog.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			response := models.ErrorResponse(404, "Blog not found")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			return
		}
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.SuccessResponse(blog)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
