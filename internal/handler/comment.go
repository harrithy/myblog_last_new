package handler

import (
	"encoding/json"
	"myblog_last_new/internal/repository"
	"myblog_last_new/internal/response"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// CommentHandler handles comment-related requests.
type CommentHandler struct {
	repo *repository.CommentRepository
}

// NewCommentHandler creates a new CommentHandler.
func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

// GetComments godoc
// @Summary List article comments
// @Description Get paginated root comments and nested replies for an article.
// @Tags comments
// @Produce json
// @Param article_id query int true "Article ID"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
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

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 10
	}

	comments, total, err := h.repo.GetByArticleIDWithPagination(articleID, page, pageSize)
	if err != nil {
		response.InternalError(w, "Failed to query comments: "+err.Error())
		return
	}

	response.SuccessWithPage(w, comments, total, page)
}

// CreateComment godoc
// @Summary Create comment
// @Description Create a comment or reply for an article.
// @Tags comments
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param comment body models.CreateCommentRequest true "Comment payload"
// @Success 200 {object} response.APIResponse
// @Router /comments [post]
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var input models.CreateCommentRequest
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

	exists, err := h.repo.ArticleExists(input.ArticleID)
	if err != nil || !exists {
		response.BadRequest(w, "article not found")
		return
	}

	if input.ParentID != nil {
		exists, err := h.repo.ParentCommentExists(*input.ParentID, input.ArticleID)
		if err != nil || !exists {
			response.BadRequest(w, "parent comment not found")
			return
		}
	}

	comment, err := h.repo.Create(input.ArticleID, input.ParentID, input.Nickname, input.Email, input.AvatarURL, input.Content)
	if err != nil {
		response.InternalError(w, "Failed to create comment: "+err.Error())
		return
	}

	response.Success(w, comment)
}

// DeleteComment godoc
// @Summary Delete comment
// @Description Delete a comment by ID.
// @Tags comments
// @Produce json
// @Security ApiKeyAuth
// @Param id path int true "Comment ID"
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
