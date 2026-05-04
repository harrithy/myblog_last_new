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

// CategoryHandler handles category-related requests.
type CategoryHandler struct {
	repo *repository.CategoryRepository
}

// NewCategoryHandler creates a new CategoryHandler.
func NewCategoryHandler(repo *repository.CategoryRepository) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

// CreateCategory godoc
// @Summary Create category
// @Description Create a new category or nested child category.
// @Tags categories
// @Accept json
// @Produce json
// @Param category body models.Category true "Category payload"
// @Success 201 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "Invalid request"
// @Failure 500 {object} response.APIResponse "Create failed"
// @Security ApiKeyAuth
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
		category.Type = models.CategoryTypeFolder
	} else if !models.IsValidCategoryType(category.Type) {
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
		response.InternalError(w, "Failed to create category: "+err.Error())
		return
	}

	category.ID = int(id)
	response.Created(w, category)
}

// GetCategories godoc
// @Summary List categories
// @Description List categories with optional tree mode and pagination.
// @Tags categories
// @Produce json
// @Param tree query bool false "Return tree structure, default true"
// @Param parent_id query int false "Parent category ID"
// @Param type query string false "Category type filter: folder or article"
// @Param keyword query string false "Category name keyword"
// @Param page query int false "Page number starting from 1"
// @Param page_size query int false "Page size"
// @Success 200 {object} response.APIResponse{data=[]models.Category}
// @Failure 500 {object} response.APIResponse "Query failed"
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

	if page > 0 {
		response.SuccessWithPage(w, categories, total, page)
		return
	}

	var result interface{}
	if treeMode && parentIDStr == "" && filter.Keyword == "" {
		result = repository.BuildCategoryTree(categories)
	} else {
		result = categories
	}

	response.Success(w, result)
}

// GetCategoryByID godoc
// @Summary Get category detail
// @Description Get a category and its nested children by ID.
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "Invalid ID"
// @Failure 404 {object} response.APIResponse "Category not found"
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

	category, err := h.repo.GetSubtree(categoryID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "Category not found")
			return
		}
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, category)
}

// UpdateCategory godoc
// @Summary Update category
// @Description Update category fields.
// @Tags categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Param category body models.Category true "Category payload"
// @Success 200 {object} response.APIResponse{data=models.Category}
// @Failure 400 {object} response.APIResponse "Invalid request"
// @Failure 404 {object} response.APIResponse "Category not found"
// @Security ApiKeyAuth
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

	if category.Type != "" && !models.IsValidCategoryType(category.Type) {
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
// @Summary Delete category
// @Description Delete a category and its descendants.
// @Tags categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} response.APIResponse
// @Failure 400 {object} response.APIResponse "Invalid request"
// @Failure 404 {object} response.APIResponse "Category not found"
// @Security ApiKeyAuth
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

func (h *CategoryHandler) extractID(path, prefix string) string {
	path = strings.TrimPrefix(path, "/api")
	return strings.TrimPrefix(path, prefix)
}

// GetHotTags godoc
// @Summary Get hot tags
// @Description Get the most frequently used tags.
// @Tags categories
// @Produce json
// @Success 200 {object} response.APIResponse{data=[]repository.HotTag}
// @Failure 500 {object} response.APIResponse "Query failed"
// @Router /categories/hot-tags [get]
func (h *CategoryHandler) GetHotTags(w http.ResponseWriter, r *http.Request) {
	hotTags, err := h.repo.GetHotTags(6)
	if err != nil {
		response.InternalError(w, "Failed to get hot tags: "+err.Error())
		return
	}

	if hotTags == nil {
		hotTags = []repository.HotTag{}
	}

	response.Success(w, hotTags)
}
