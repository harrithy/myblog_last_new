package handler

import (
	"database/sql"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"net/http"
	"strconv"
	"strings"
)

// BlogHandler handles article-backed blog endpoints.
type BlogHandler struct {
	repo *repository.BlogRepository
}

// NewBlogHandler creates a new BlogHandler.
func NewBlogHandler(repo *repository.BlogRepository) *BlogHandler {
	return &BlogHandler{repo: repo}
}

// GetBlogs godoc
// @Summary List article-backed blogs
// @Description Returns paginated article nodes from categories(type=article), with optional parent category and keyword filters.
// @Tags blogs
// @Produce  json
// @Param   category_id query    int     false  "Parent category ID used to filter article nodes"
// @Param   keyword     query    string  false  "Keyword matched against article titles"
// @Param   page        query    int     false  "Page number, default 1"
// @Param   pagesize    query    int     false  "Page size, default 10"
// @Success 200 {object} response.APIResponse{data=[]models.Blog}
// @Failure 500 {object} response.APIResponse "Query failed"
// @Router /blogs [get]
func (h *BlogHandler) GetBlogs(w http.ResponseWriter, r *http.Request) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(r.URL.Query().Get("pagesize"))
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	filter := repository.BlogFilter{
		Keyword:  r.URL.Query().Get("keyword"),
		Page:     page,
		PageSize: pageSize,
	}

	if categoryIDStr := r.URL.Query().Get("category_id"); categoryIDStr != "" {
		if categoryID, err := strconv.Atoi(categoryIDStr); err == nil {
			filter.CategoryID = &categoryID
		}
	}

	blogs, total, err := h.repo.GetAll(filter)
	if err != nil {
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.SuccessWithPage(w, blogs, total, page)
}

// GetBlogDetail godoc
// @Summary Get article-backed blog detail
// @Description Returns a single article node from categories(type=article) while preserving the blog response shape.
// @Tags blogs
// @Produce  json
// @Param   id   path    int     true  "Article node ID"
// @Success 200 {object} response.APIResponse{data=models.Blog}
// @Failure 400 {object} response.APIResponse "Invalid ID"
// @Failure 404 {object} response.APIResponse "Blog not found"
// @Failure 500 {object} response.APIResponse "Query failed"
// @Router /blogs/{id} [get]
func (h *BlogHandler) GetBlogDetail(w http.ResponseWriter, r *http.Request) {
	// Support both /blogs/{id} and /api/blogs/{id}.
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api")
	idStr := strings.TrimPrefix(path, "/blogs/")

	if idStr == "" {
		response.BadRequest(w, "Blog ID is required")
		return
	}

	blogID, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(w, "Invalid blog ID")
		return
	}

	blog, err := h.repo.GetByID(blogID)
	if err != nil {
		if err == sql.ErrNoRows {
			response.NotFound(w, "Blog not found")
			return
		}
		response.InternalError(w, "Query failed: "+err.Error())
		return
	}

	response.Success(w, blog)
}
