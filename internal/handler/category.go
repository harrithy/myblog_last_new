package handler

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// CategoryHandler 处理分类相关请求
type CategoryHandler struct {
	repo *repository.CategoryRepository
}

// NewCategoryHandler 创建新的 CategoryHandler
func NewCategoryHandler(repo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

// CreateCategory godoc
// @Summary 创建分类
// @Description 创建一个新的分类，可以是顶级分类或子分类。
// @Tags categories
// @Accept  json
// @Produce  json
// @Param   category   body    models.Category   true  "分类信息"
// @Success 201 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 500 {object} response.APIResponse "创建失败"
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if category.Name == "" {
		response.BadRequest(w, "Category name is required")
		return
	}

	if category.Type == "" {
		category.Type = "folder"
	} else if category.Type != "folder" && category.Type != "article" {
		response.BadRequest(w, "Type must be 'folder' or 'article'")
		return
	}

	if category.ParentID != nil {
		exists, err := h.repo.Exists(*category.ParentID)
		if err != nil || !exists {
			response.BadRequest(w, "Parent category not found")
			return
		}
	}

	id, err := h.repo.Create(&category)
	if err != nil {
		response.InternalError(w, "创建分类失败: "+err.Error())
		return
	}

	category.ID = int(id)
	response.Created(w, category)
}

// GetCategories godoc
// @Summary 获取分类列表
// @Description 获取所有分类，支持树形结构返回和分页查询
// @Tags categories
// @Produce  json
// @Param   tree      query    bool    false  "是否返回树形结构，默认true"
// @Param   parent_id query    int     false  "父分类ID"
// @Param   type      query    string  false  "类型筛选：folder或article"
// @Param   keyword   query    string  false  "标题模糊搜索关键词"
// @Param   page      query    int     false  "页码，从1开始"
// @Param   page_size query    int     false  "每页数量，默认10"
// @Success 200 {object} response.APIResponse{data=[]models.Category}
// @Failure 500 {object} response.APIResponse "查询失败"
// @Router /categories [get]
func (h *CategoryHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	treeMode := r.URL.Query().Get("tree") != "false"
	parentIDStr := r.URL.Query().Get("parent_id")
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	filter := repository.CategoryFilter{
		Type:    r.URL.Query().Get("type"),
		Keyword: r.URL.Query().Get("keyword"),
	}

	if parentIDStr != "" {
		if parentID, err := strconv.Atoi(parentIDStr); err == nil {
			filter.ParentID = &parentID
		}
	}

	// 解析分页参数
	var page, pageSize int
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}
	// 如果指定了page但没有page_size，默认10
	if page > 0 && pageSize == 0 {
		pageSize = 10
	}
	filter.Page = page
	filter.PageSize = pageSize

	categories, total, err := h.repo.GetAll(filter)
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	// 如果启用分页，返回带分页信息的响应
	if page > 0 {
		response.SuccessWithPage(w, categories, total, page)
		return
	}

	// 非分页模式
	var result interface{}
	// 使用关键词搜索时，直接返回扁平列表
	if treeMode && parentIDStr == "" && filter.Keyword == "" {
		result = repository.BuildCategoryTree(categories)
	} else {
		result = categories
	}

	response.Success(w, result)
}

// GetCategoryByID godoc
// @Summary 获取单个分类详情
// @Description 根据ID获取分类详情，包含子分类
// @Tags categories
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Success 200 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 404 {object} response.APIResponse "分类不存在"
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	idStr := h.extractID(r.URL.Path, "/categories/")
	if idStr == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid category ID")
		return
	}

	category, err := h.repo.GetByID(categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "Category not found")
			return
		}
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	children, err := h.repo.GetChildren(categoryID)
	if err == nil && len(children) > 0 {
		category.Children = children
	}

	response.Success(w, category)
}

// UpdateCategory godoc
// @Summary 更新分类
// @Description 更新分类信息
// @Tags categories
// @Accept  json
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Param   category   body    models.Category   true  "分类信息"
// @Success 200 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 404 {object} response.APIResponse "分类不存在"
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := h.extractID(r.URL.Path, "/categories/")
	if idStr == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid category ID")
		return
	}

	var category models.Category
	if err := json.NewDecoder(r.Body).Decode(&category); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if category.Name == "" {
		response.BadRequest(w, "Category name is required")
		return
	}

	if category.ParentID != nil && *category.ParentID == categoryID {
		response.BadRequest(w, "Category cannot be its own parent")
		return
	}

	if category.Type != "" && category.Type != "folder" && category.Type != "article" {
		response.BadRequest(w, "Type must be 'folder' or 'article'")
		return
	}

	rowsAffected, err := h.repo.Update(categoryID, &category)
	if err != nil {
		response.InternalError(w, "Failed to update category: "+err.Error())
		return
	}

	if rowsAffected == 0 {
		response.NotFound(w, "Category not found")
		return
	}

	category.ID = categoryID
	response.Success(w, category)
}

// DeleteCategory godoc
// @Summary 删除分类
// @Description 删除分类，子分类会一并删除
// @Tags categories
// @Produce  json
// @Param   id   path    int     true  "分类ID"
// @Success 200 {object} response.APIResponse "删除成功"
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 404 {object} response.APIResponse "分类不存在"
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := h.extractID(r.URL.Path, "/categories/")
	if idStr == "" {
		response.BadRequest(w, "Category ID is required")
		return
	}

	categoryID, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid category ID")
		return
	}

	rowsAffected, err := h.repo.Delete(categoryID)
	if err != nil {
		response.InternalError(w, "Failed to delete category: "+err.Error())
		return
	}

	if rowsAffected == 0 {
		response.NotFound(w, "Category not found")
		return
	}

	response.Success(w, map[string]string{"message": "Category deleted successfully"})
}

// extractID 从 URL 路径提取 ID
func (h *CategoryHandler) extractID(path, prefix string) string {
	path = strings.TrimPrefix(path, "/api")
	return strings.TrimPrefix(path, prefix)
}

// GetHotTags godoc
// @Summary 获取热门标签
// @Description 获取使用次数前6的标签
// @Tags categories
// @Produce  json
// @Success 200 {object} response.APIResponse{data=[]repository.HotTag}
// @Failure 500 {object} response.APIResponse "查询失败"
// @Router /categories/hot-tags [get]
func (h *CategoryHandler) GetHotTags(w http.ResponseWriter, r *http.Request) {
	hotTags, err := h.repo.GetHotTags(6)
	if err != nil {
		response.InternalError(w, "获取热门标签失败: "+err.Error())
		return
	}

	// 确保返回空数组而不是 null
	if hotTags == nil {
		hotTags = []repository.HotTag{}
	}

	response.Success(w, hotTags)
}
