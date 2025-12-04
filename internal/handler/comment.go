package handler

import (
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"net/http"
	"strconv"
	"strings"
)

// CommentHandler 处理评论相关请求
type CommentHandler struct {
	repo *repository.CommentRepository
}

// NewCommentHandler 创建新的 CommentHandler
func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

// GetComments godoc
// @Summary 获取文章评论列表
// @Description 根据文章ID获取评论列表，返回树形结构
// @Tags comments
// @Produce json
// @Param article_id query int true "文章ID"
// @Success 200 {object} response.APIResponse
// @Router /comments [get]
func (h *CommentHandler) GetComments(w http.ResponseWriter, r *http.Request) {
	articleIDStr := r.URL.Query().Get("article_id")
	if articleIDStr == "" {
		response.BadRequest(w, "article_id is required")
		return
	}

	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		response.BadRequest(w, "invalid article_id")
		return
	}

	comments, err := h.repo.GetByArticleID(articleID)
	if err != nil {
		response.InternalError(w, "Failed to query comments: "+err.Error())
		return
	}

	response.SuccessWithPage(w, comments, int64(len(comments)), 1)
}

// CreateComment godoc
// @Summary 创建评论
// @Description 为文章创建评论，支持回复其他评论
// @Tags comments
// @Accept json
// @Produce json
// @Param comment body object true "评论信息"
// @Success 200 {object} response.APIResponse
// @Router /comments [post]
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ArticleID int    `json:"article_id"`
		ParentID  *int   `json:"parent_id"`
		Nickname  string `json:"nickname"`
		Email     string `json:"email"`
		Content   string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	if input.ArticleID == 0 {
		response.BadRequest(w, "article_id is required")
		return
	}
	if strings.TrimSpace(input.Nickname) == "" {
		response.BadRequest(w, "nickname is required")
		return
	}
	if strings.TrimSpace(input.Content) == "" {
		response.BadRequest(w, "content is required")
		return
	}

	// 验证文章是否存在
	exists, err := h.repo.ArticleExists(input.ArticleID)
	if err != nil || !exists {
		response.BadRequest(w, "article not found")
		return
	}

	// 如果提供了父评论则验证
	if input.ParentID != nil {
		exists, err := h.repo.ParentCommentExists(*input.ParentID, input.ArticleID)
		if err != nil || !exists {
			response.BadRequest(w, "parent comment not found")
			return
		}
	}

	comment, err := h.repo.Create(input.ArticleID, input.ParentID, input.Nickname, input.Email, input.Content)
	if err != nil {
		response.InternalError(w, "Failed to create comment: "+err.Error())
		return
	}

	response.Success(w, comment)
}

// DeleteComment godoc
// @Summary 删除评论
// @Description 根据ID删除评论
// @Tags comments
// @Produce json
// @Param id path int true "评论ID"
// @Success 200 {object} response.APIResponse
// @Router /comments/{id} [delete]
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/comments/")
	path = strings.TrimPrefix(path, "/comments/")
	idStr := strings.TrimSuffix(path, "/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.BadRequest(w, "invalid comment id")
		return
	}

	rowsAffected, err := h.repo.Delete(id)
	if err != nil {
		response.InternalError(w, "Failed to delete comment: "+err.Error())
		return
	}

	if rowsAffected == 0 {
		response.NotFound(w, "comment not found")
		return
	}

	response.Success(w, map[string]interface{}{"deleted_id": id})
}
