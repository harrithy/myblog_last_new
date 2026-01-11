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
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
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
		var avatarURL sql.NullString

		err := rows.Scan(
			&comment.ID,
			&comment.ArticleID,
			&parentID,
			&comment.Nickname,
			&email,
			&avatarURL,
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
		if avatarURL.Valid {
			comment.AvatarURL = avatarURL.String
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

// GetByArticleIDWithPagination 支持分页获取文章评论
func (r *CommentRepository) GetByArticleIDWithPagination(articleID, page, pageSize int) ([]models.Comment, int64, error) {
	// 1. Count root comments
	var total int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE article_id = ? AND parent_id IS NULL", articleID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []models.Comment{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// 2. Fetch root comments
	query := `
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
		FROM comments
		WHERE article_id = ? AND parent_id IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	rows, err := r.db.Query(query, articleID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	commentMap := make(map[int]*models.Comment)
	var rootComments []*models.Comment

	for rows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		var email sql.NullString
		var avatarURL sql.NullString

		err := rows.Scan(
			&comment.ID,
			&comment.ArticleID,
			&parentID,
			&comment.Nickname,
			&email,
			&avatarURL,
			&comment.Content,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		}
		if email.Valid {
			comment.Email = email.String
		}
		if avatarURL.Valid {
			comment.AvatarURL = avatarURL.String
		}

		comment.Children = []models.Comment{}
		// 注意：这里需要复制一份，否则循环变量地址问题
		c := comment
		commentMap[comment.ID] = &c
		rootComments = append(rootComments, &c)
	}

	// 3. Fetch ALL child comments for the article to build the tree
	childQuery := `
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
		FROM comments
		WHERE article_id = ? AND parent_id IS NOT NULL
		ORDER BY created_at ASC
	`
	childRows, err := r.db.Query(childQuery, articleID)
	if err != nil {
		return nil, 0, err
	}
	defer childRows.Close()

	for childRows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		var email sql.NullString
		var avatarURL sql.NullString

		err := childRows.Scan(
			&comment.ID,
			&comment.ArticleID,
			&parentID,
			&comment.Nickname,
			&email,
			&avatarURL,
			&comment.Content,
			&comment.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		}
		if email.Valid {
			comment.Email = email.String
		}
		if avatarURL.Valid {
			comment.AvatarURL = avatarURL.String
		}

		comment.Children = []models.Comment{}
		c := comment
		commentMap[comment.ID] = &c
	}

	// 4. Build Tree
	for _, comment := range commentMap {
		if comment.ParentID != nil {
			if parent, ok := commentMap[*comment.ParentID]; ok {
				parent.Children = append(parent.Children, *comment)
			}
		}
	}

	// 5. Construct result
	result := make([]models.Comment, 0, len(rootComments))
	for _, c := range rootComments {
		result = append(result, *c)
	}

	return result, total, nil
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
func (r *CommentRepository) Create(articleID int, parentID *int, nickname, email, avatarURL, content string) (*models.Comment, error) {
	var result sql.Result
	var err error

	if parentID != nil {
		result, err = r.db.Exec(
			"INSERT INTO comments (article_id, parent_id, nickname, email, avatar_url, content) VALUES (?, ?, ?, ?, ?, ?)",
			articleID, *parentID, nickname, email, avatarURL, content,
		)
	} else {
		result, err = r.db.Exec(
			"INSERT INTO comments (article_id, nickname, email, avatar_url, content) VALUES (?, ?, ?, ?, ?)",
			articleID, nickname, email, avatarURL, content,
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
	var av sql.NullString

	err = r.db.QueryRow(
		"SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at FROM comments WHERE id = ?",
		id,
	).Scan(&comment.ID, &comment.ArticleID, &pID, &comment.Nickname, &em, &av, &comment.Content, &comment.CreatedAt)

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
	if av.Valid {
		comment.AvatarURL = av.String
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
