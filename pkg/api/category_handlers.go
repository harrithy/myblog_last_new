package api

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// CreateCategory godoc
// @Summary 创建分类
// @Description 创建一个新的分类，可以是顶级分类或子分类。parent_id为空表示顶级分类，传入父分类ID则创建子分类。type可选folder(文件夹)或article(文章)，默认folder。
// @Tags categories
// @Accept  json
// @Produce  json
// @Param   category   body    models.Category   true  "分类信息"
// @Success 201 {object} models.APIResponse{data=models.Category}
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 500 {object} models.APIResponse "创建失败"
// @Router /categories [post]
func CreateCategory(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		response := models.ErrorResponse(400, "Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证分类名称
	if category.Name == "" {
		response := models.ErrorResponse(400, "Category name is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证并设置默认type
	if category.Type == "" {
		category.Type = "folder"
	} else if category.Type != "folder" && category.Type != "article" {
		response := models.ErrorResponse(400, "Type must be 'folder' or 'article'")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 如果有父分类ID，验证父分类是否存在
	if category.ParentID != nil {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", *category.ParentID).Scan(&exists)
		if err != nil || !exists {
			response := models.ErrorResponse(400, "Parent category not found")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// 插入分类
	var result sql.Result
	var err error
	if category.ParentID != nil {
		result, err = db.Exec(
			"INSERT INTO categories (name, type, url, parent_id, sort_order) VALUES (?, ?, ?, ?, ?)",
			category.Name, category.Type, category.URL, *category.ParentID, category.SortOrder,
		)
	} else {
		result, err = db.Exec(
			"INSERT INTO categories (name, type, url, sort_order) VALUES (?, ?, ?, ?)",
			category.Name, category.Type, category.URL, category.SortOrder,
		)
	}

	if err != nil {
		response := models.ErrorResponse(500, "Failed to create category: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	id, _ := result.LastInsertId()
	category.ID = int(id)

	response := models.SuccessResponse(category)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetCategories godoc
// @Summary 获取分类列表
// @Description 获取所有分类，支持树形结构返回，支持按父分类和类型筛选
// @Tags categories
// @Produce  json
// @Param   tree      query    bool    false  "是否返回树形结构，默认true"
// @Param   parent_id query    int     false  "父分类ID，查询指定父分类下的子分类"
// @Param   type      query    string  false  "类型筛选：folder(文件夹)或article(文章)"
// @Success 200 {object} models.APIResponse{data=[]models.Category}
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /categories [get]
func GetCategories(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	treeMode := r.URL.Query().Get("tree") != "false"
	parentIDStr := r.URL.Query().Get("parent_id")
	categoryType := r.URL.Query().Get("type")

	// 构建查询条件
	query := "SELECT id, name, type, IFNULL(url, ''), parent_id, sort_order, created_at, updated_at FROM categories WHERE 1=1"
	var args []interface{}

	if parentIDStr != "" {
		parentID, err := strconv.Atoi(parentIDStr)
		if err == nil {
			query += " AND parent_id = ?"
			args = append(args, parentID)
		}
	}

	if categoryType != "" && (categoryType == "folder" || categoryType == "article") {
		query += " AND type = ?"
		args = append(args, categoryType)
	}

	query += " ORDER BY sort_order ASC, id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		response := models.ErrorResponse(500, "Query failed: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var parentID sql.NullInt64
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.URL, &parentID, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			response := models.ErrorResponse(500, "Data parse failed: "+err.Error())
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}
		if parentID.Valid {
			pid := int(parentID.Int64)
			cat.ParentID = &pid
		}
		categories = append(categories, cat)
	}

	var result interface{}
	if treeMode && parentIDStr == "" {
		result = buildCategoryTree(categories)
	} else {
		result = categories
	}

	response := models.SuccessResponse(result)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetCategoryByID godoc
// @Summary 获取单个分类详情
// @Description 根据ID获取分类详情，包含子分类
// @Tags categories
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Success 200 {object} models.APIResponse{data=models.Category}
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 404 {object} models.APIResponse "分类不存在"
// @Failure 500 {object} models.APIResponse "查询失败"
// @Router /categories/{id} [get]
func GetCategoryByID(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	if idStr == "" {
		response := models.ErrorResponse(400, "Category ID is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response := models.ErrorResponse(400, "Invalid category ID")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var category models.Category
	var parentID sql.NullInt64
	err = db.QueryRow(`
		SELECT id, name, type, IFNULL(url, ''), parent_id, sort_order, created_at, updated_at 
		FROM categories WHERE id = ?
	`, categoryID).Scan(&category.ID, &category.Name, &category.Type, &category.URL, &parentID, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			response := models.ErrorResponse(404, "Category not found")
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

	if parentID.Valid {
		pid := int(parentID.Int64)
		category.ParentID = &pid
	}

	// 获取子分类
	children, err := getChildCategories(db, categoryID)
	if err == nil && len(children) > 0 {
		category.Children = children
	}

	response := models.SuccessResponse(category)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateCategory godoc
// @Summary 更新分类
// @Description 更新分类信息，可修改名称、父分类和排序
// @Tags categories
// @Accept  json
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Param   category   body    models.Category   true  "分类信息"
// @Success 200 {object} models.APIResponse{data=models.Category}
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 404 {object} models.APIResponse "分类不存在"
// @Failure 500 {object} models.APIResponse "更新失败"
// @Router /categories/{id} [put]
func UpdateCategory(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	if idStr == "" {
		response := models.ErrorResponse(400, "Category ID is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response := models.ErrorResponse(400, "Invalid category ID")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		response := models.ErrorResponse(400, "Invalid request body")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证分类名称
	if category.Name == "" {
		response := models.ErrorResponse(400, "Category name is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 防止将分类设置为自己的子分类
	if category.ParentID != nil && *category.ParentID == categoryID {
		response := models.ErrorResponse(400, "Category cannot be its own parent")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 验证type
	if category.Type != "" && category.Type != "folder" && category.Type != "article" {
		response := models.ErrorResponse(400, "Type must be 'folder' or 'article'")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// 更新分类
	var result sql.Result
	if category.ParentID != nil {
		result, err = db.Exec(
			"UPDATE categories SET name = ?, type = ?, url = ?, parent_id = ?, sort_order = ? WHERE id = ?",
			category.Name, category.Type, category.URL, *category.ParentID, category.SortOrder, categoryID,
		)
	} else {
		result, err = db.Exec(
			"UPDATE categories SET name = ?, type = ?, url = ?, parent_id = NULL, sort_order = ? WHERE id = ?",
			category.Name, category.Type, category.URL, category.SortOrder, categoryID,
		)
	}

	if err != nil {
		response := models.ErrorResponse(500, "Failed to update category: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		response := models.ErrorResponse(404, "Category not found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	category.ID = categoryID
	response := models.SuccessResponse(category)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteCategory godoc
// @Summary 删除分类
// @Description 删除分类，子分类会一并删除
// @Tags categories
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Success 200 {object} models.APIResponse "删除成功"
// @Failure 400 {object} models.APIResponse "参数错误"
// @Failure 404 {object} models.APIResponse "分类不存在"
// @Failure 500 {object} models.APIResponse "删除失败"
// @Router /categories/{id} [delete]
func DeleteCategory(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	idStr := strings.TrimPrefix(r.URL.Path, "/categories/")
	if idStr == "" {
		response := models.ErrorResponse(400, "Category ID is required")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response := models.ErrorResponse(400, "Invalid category ID")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	result, err := db.Exec("DELETE FROM categories WHERE id = ?", categoryID)
	if err != nil {
		response := models.ErrorResponse(500, "Failed to delete category: "+err.Error())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		response := models.ErrorResponse(404, "Category not found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.SuccessResponse(map[string]string{"message": "Category deleted successfully"})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// buildCategoryTree 将扁平的分类列表构建成树形结构
func buildCategoryTree(categories []models.Category) []models.Category {
	categoryMap := make(map[int]*models.Category)
	var roots []models.Category

	// 先将所有分类放入map
	for i := range categories {
		categories[i].Children = []models.Category{}
		categoryMap[categories[i].ID] = &categories[i]
	}

	// 构建树形结构
	for i := range categories {
		cat := &categories[i]
		if cat.ParentID == nil {
			roots = append(roots, *cat)
		} else {
			if parent, ok := categoryMap[*cat.ParentID]; ok {
				parent.Children = append(parent.Children, *cat)
			}
		}
	}

	// 更新roots中的children（因为上面修改的是map中的指针）
	for i := range roots {
		if mapped, ok := categoryMap[roots[i].ID]; ok {
			roots[i].Children = mapped.Children
		}
	}

	return roots
}

// getChildCategories 获取指定分类的子分类
func getChildCategories(db *sql.DB, parentID int) ([]models.Category, error) {
	rows, err := db.Query(`
		SELECT id, name, type, IFNULL(url, ''), parent_id, sort_order, created_at, updated_at 
		FROM categories 
		WHERE parent_id = ?
		ORDER BY sort_order ASC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []models.Category
	for rows.Next() {
		var cat models.Category
		var pid sql.NullInt64
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.URL, &pid, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, err
		}
		if pid.Valid {
			p := int(pid.Int64)
			cat.ParentID = &p
		}
		// 递归获取子分类的子分类
		subChildren, _ := getChildCategories(db, cat.ID)
		if len(subChildren) > 0 {
			cat.Children = subChildren
		}
		children = append(children, cat)
	}

	return children, nil
}
