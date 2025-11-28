package api

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/models"
	"net/http"
	"strconv"
	"strings"
)

// GetComments 获取文章评论列表（树形结构）
// @Summary 获取文章评论列表
// @Description 根据文章ID获取评论列表，返回树形结构（父评论包含子评论）
// @Tags comments
// @Accept json
// @Produce json
// @Param article_id query int true "文章ID"
// @Success 200 {object} models.APIResponse
// @Router /api/comments [get]
func GetComments(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// 获取文章ID参数
	articleIDStr := r.URL.Query().Get("article_id")
	if articleIDStr == "" {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "article_id is required"))
		return
	}

	articleID, err := strconv.Atoi(articleIDStr)
	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "invalid article_id"))
		return
	}

	// 查询所有评论
	query := `
		SELECT id, article_id, parent_id, nickname, email, content, created_at
		FROM comments
		WHERE article_id = ?
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, articleID)
	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(500, "Failed to query comments: "+err.Error()))
		return
	}
	defer rows.Close()

	// 用于存储所有评论
	commentMap := make(map[int]*models.Comment)
	var rootComments []*models.Comment

	for rows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		var email sql.NullString

		err := rows.Scan(
			&comment.ID,
			&comment.ArticleID,
			&parentID,
			&comment.Nickname,
			&email,
			&comment.Content,
			&comment.CreatedAt,
		)
		if err != nil {
			json.NewEncoder(w).Encode(models.ErrorResponse(500, "Failed to scan comment: "+err.Error()))
			return
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		}
		if email.Valid {
			comment.Email = email.String
		}

		comment.Children = []models.Comment{}
		commentMap[comment.ID] = &comment
	}

	// 构建树形结构
	for _, comment := range commentMap {
		if comment.ParentID == nil {
			rootComments = append(rootComments, comment)
		} else {
			if parent, ok := commentMap[*comment.ParentID]; ok {
				parent.Children = append(parent.Children, *comment)
			}
		}
	}

	// 转换为值类型切片
	result := make([]models.Comment, 0, len(rootComments))
	for _, c := range rootComments {
		result = append(result, *c)
	}

	json.NewEncoder(w).Encode(models.APIResponse{
		Code:  200,
		Data:  result,
		Msg:   "success",
		Total: int64(len(result)),
	})
}

// CreateComment 创建评论
// @Summary 创建评论
// @Description 为文章创建评论，支持回复其他评论
// @Tags comments
// @Accept json
// @Produce json
// @Param comment body object true "评论信息" example({"article_id": 1, "nickname": "访客", "content": "评论内容", "parent_id": null, "email": "test@example.com"})
// @Success 200 {object} models.APIResponse
// @Router /api/comments [post]
func CreateComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	var input struct {
		ArticleID int    `json:"article_id"`
		ParentID  *int   `json:"parent_id"`
		Nickname  string `json:"nickname"`
		Email     string `json:"email"`
		Content   string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "Invalid request body"))
		return
	}

	// 验证必填字段
	if input.ArticleID == 0 {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "article_id is required"))
		return
	}
	if strings.TrimSpace(input.Nickname) == "" {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "nickname is required"))
		return
	}
	if strings.TrimSpace(input.Content) == "" {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "content is required"))
		return
	}

	// 验证文章是否存在
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ? AND type = 'article')", input.ArticleID).Scan(&exists)
	if err != nil || !exists {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "article not found"))
		return
	}

	// 如果有父评论ID，验证父评论是否存在
	if input.ParentID != nil {
		var parentExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = ? AND article_id = ?)", *input.ParentID, input.ArticleID).Scan(&parentExists)
		if err != nil || !parentExists {
			json.NewEncoder(w).Encode(models.ErrorResponse(400, "parent comment not found"))
			return
		}
	}

	// 插入评论
	var result sql.Result
	if input.ParentID != nil {
		result, err = db.Exec(
			"INSERT INTO comments (article_id, parent_id, nickname, email, content) VALUES (?, ?, ?, ?, ?)",
			input.ArticleID, *input.ParentID, input.Nickname, input.Email, input.Content,
		)
	} else {
		result, err = db.Exec(
			"INSERT INTO comments (article_id, nickname, email, content) VALUES (?, ?, ?, ?)",
			input.ArticleID, input.Nickname, input.Email, input.Content,
		)
	}

	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(500, "Failed to create comment: "+err.Error()))
		return
	}

	id, _ := result.LastInsertId()

	// 查询刚插入的评论
	var comment models.Comment
	var parentID sql.NullInt64
	var email sql.NullString
	err = db.QueryRow(
		"SELECT id, article_id, parent_id, nickname, email, content, created_at FROM comments WHERE id = ?",
		id,
	).Scan(&comment.ID, &comment.ArticleID, &parentID, &comment.Nickname, &email, &comment.Content, &comment.CreatedAt)

	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(500, "Failed to fetch created comment"))
		return
	}

	if parentID.Valid {
		pid := int(parentID.Int64)
		comment.ParentID = &pid
	}
	if email.Valid {
		comment.Email = email.String
	}

	json.NewEncoder(w).Encode(models.SuccessResponse(comment))
}

// DeleteComment 删除评论
// @Summary 删除评论
// @Description 根据ID删除评论（会级联删除子评论）
// @Tags comments
// @Accept json
// @Produce json
// @Param id path int true "评论ID"
// @Success 200 {object} models.APIResponse
// @Router /api/comments/{id} [delete]
func DeleteComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	w.Header().Set("Content-Type", "application/json")

	// 从URL路径获取评论ID
	path := strings.TrimPrefix(r.URL.Path, "/api/comments/")
	path = strings.TrimPrefix(path, "/comments/")
	idStr := strings.TrimSuffix(path, "/")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(400, "invalid comment id"))
		return
	}

	// 删除评论（外键级联会自动删除子评论）
	result, err := db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		json.NewEncoder(w).Encode(models.ErrorResponse(500, "Failed to delete comment: "+err.Error()))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		json.NewEncoder(w).Encode(models.ErrorResponse(404, "comment not found"))
		return
	}

	json.NewEncoder(w).Encode(models.SuccessResponse(map[string]interface{}{
		"deleted_id": id,
	}))
}
