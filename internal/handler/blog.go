package handler

import (
	"database/sql"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"net/http"
	"strconv"
	"strings"
)

// BlogHandler 处理博客相关请求
type BlogHandler struct {
	repo *repository.BlogRepository
}

// NewBlogHandler 创建新的 BlogHandler
func NewBlogHandler(repo *repository.BlogRepository) *BlogHandler {
	return &BlogHandler{repo: repo}
}

// GetBlogs godoc
// @Summary 获取博客列表
// @Description 根据分类ID分页获取博客列表，支持关键词搜索。
// @Tags blogs
// @Produce  json
// @Param   category_id query    int     false  "分类ID"
// @Param   keyword     query    string  false  "搜索关键词"
// @Param   page        query    int     false  "页码，默认1"
// @Param   pagesize    query    int     false  "每页数量，默认10"
// @Success 200 {object} response.APIResponse{data=[]models.Blog}
// @Failure 500 {object} response.APIResponse "查询失败"
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
// @Summary 获取博客详情
// @Description 根据ID获取单个博客的详细信息。
// @Tags blogs
// @Produce  json
// @Param   id   path    int     true  "博客ID"
// @Success 200 {object} response.APIResponse{data=models.Blog}
// @Failure 400 {object} response.APIResponse "参数错误"
// @Failure 404 {object} response.APIResponse "博客不存在"
// @Failure 500 {object} response.APIResponse "查询失败"
// @Router /blogs/{id} [get]
func (h *BlogHandler) GetBlogDetail(w http.ResponseWriter, r *http.Request) {
	// 从路径提取 ID（支持 /blogs/{id} 和 /api/blogs/{id}）
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
