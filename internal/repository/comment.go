package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
)

// CommentRepository 处理评论数据访问
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository 创建新的 CommentRepository
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// GetByArticleID 返回文章的所有评论
func (r *CommentRepository) GetByArticleID(articleID int) ([]models.Comment, error) {
	query := `
		SELECT id, article_id, parent_id, nickname, email, content, created_at
		FROM comments
		WHERE article_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
			return nil, err
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

	// Build tree structure
	for _, comment := range commentMap {
		if comment.ParentID == nil {
			rootComments = append(rootComments, comment)
		} else {
			if parent, ok := commentMap[*comment.ParentID]; ok {
				parent.Children = append(parent.Children, *comment)
			}
		}
	}

	result := make([]models.Comment, 0, len(rootComments))
	for _, c := range rootComments {
		result = append(result, *c)
	}

	return result, nil
}

// ArticleExists 检查文章是否存在
func (r *CommentRepository) ArticleExists(articleID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ? AND type = 'article')", articleID).Scan(&exists)
	return exists, err
}

// ParentCommentExists 检查父评论是否存在
func (r *CommentRepository) ParentCommentExists(parentID, articleID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = ? AND article_id = ?)", parentID, articleID).Scan(&exists)
	return exists, err
}

// Create 创建新评论
func (r *CommentRepository) Create(articleID int, parentID *int, nickname, email, content string) (*models.Comment, error) {
	var result sql.Result
	var err error

	if parentID != nil {
		result, err = r.db.Exec(
			"INSERT INTO comments (article_id, parent_id, nickname, email, content) VALUES (?, ?, ?, ?, ?)",
			articleID, *parentID, nickname, email, content,
		)
	} else {
		result, err = r.db.Exec(
			"INSERT INTO comments (article_id, nickname, email, content) VALUES (?, ?, ?, ?)",
			articleID, nickname, email, content,
		)
	}

	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	// Fetch the created comment
	var comment models.Comment
	var pID sql.NullInt64
	var em sql.NullString

	err = r.db.QueryRow(
		"SELECT id, article_id, parent_id, nickname, email, content, created_at FROM comments WHERE id = ?",
		id,
	).Scan(&comment.ID, &comment.ArticleID, &pID, &comment.Nickname, &em, &comment.Content, &comment.CreatedAt)

	if err != nil {
		return nil, err
	}

	if pID.Valid {
		pid := int(pID.Int64)
		comment.ParentID = &pid
	}
	if em.Valid {
		comment.Email = em.String
	}

	return &comment, nil
}

// Delete 根据 ID 删除评论
func (r *CommentRepository) Delete(id int) (int64, error) {
	result, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
