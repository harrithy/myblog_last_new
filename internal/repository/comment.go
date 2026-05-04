package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
)

// CommentRepository handles comment data access.
type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new CommentRepository.
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// GetByArticleID returns all comments for an article.
func (r *CommentRepository) GetByArticleID(articleID int) ([]models.Comment, error) {
	rows, err := r.db.Query(`
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
		FROM comments
		WHERE article_id = ?
		ORDER BY created_at ASC
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments, rootOrder, err := scanComments(rows)
	if err != nil {
		return nil, err
	}

	return buildCommentTree(comments, rootOrder), nil
}

// GetByArticleIDWithPagination returns paginated root comments plus nested replies.
func (r *CommentRepository) GetByArticleIDWithPagination(articleID, page, pageSize int) ([]models.Comment, int64, error) {
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE article_id = ? AND parent_id IS NULL", articleID).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Comment{}, 0, nil
	}

	offset := (page - 1) * pageSize

	rootRows, err := r.db.Query(`
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
		FROM comments
		WHERE article_id = ? AND parent_id IS NULL
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, articleID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rootRows.Close()

	rootComments, rootOrder, err := scanComments(rootRows)
	if err != nil {
		return nil, 0, err
	}

	childRows, err := r.db.Query(`
		SELECT id, article_id, parent_id, nickname, email, avatar_url, content, created_at
		FROM comments
		WHERE article_id = ? AND parent_id IS NOT NULL
		ORDER BY created_at ASC
	`, articleID)
	if err != nil {
		return nil, 0, err
	}
	defer childRows.Close()

	childComments, _, err := scanComments(childRows)
	if err != nil {
		return nil, 0, err
	}

	comments := append(rootComments, childComments...)
	return buildCommentTree(comments, rootOrder), total, nil
}

// ArticleExists checks whether the target article exists.
func (r *CommentRepository) ArticleExists(articleID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM categories WHERE id = ? AND type = ?)",
		articleID,
		models.CategoryTypeArticle,
	).Scan(&exists)
	return exists, err
}

// ParentCommentExists checks whether the parent comment exists for the article.
func (r *CommentRepository) ParentCommentExists(parentID, articleID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM comments WHERE id = ? AND article_id = ?)", parentID, articleID).Scan(&exists)
	return exists, err
}

// Create creates a comment.
func (r *CommentRepository) Create(articleID int, parentID *int, nickname, email, avatarURL, content string) (*models.Comment, error) {
	var (
		result sql.Result
		err    error
	)

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

// Delete deletes a comment by ID.
func (r *CommentRepository) Delete(id int) (int64, error) {
	result, err := r.db.Exec("DELETE FROM comments WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func scanComments(rows *sql.Rows) ([]models.Comment, []int, error) {
	var comments []models.Comment
	var rootOrder []int

	for rows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		var email sql.NullString
		var avatarURL sql.NullString

		if err := rows.Scan(
			&comment.ID,
			&comment.ArticleID,
			&parentID,
			&comment.Nickname,
			&email,
			&avatarURL,
			&comment.Content,
			&comment.CreatedAt,
		); err != nil {
			return nil, nil, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		} else {
			rootOrder = append(rootOrder, comment.ID)
		}
		if email.Valid {
			comment.Email = email.String
		}
		if avatarURL.Valid {
			comment.AvatarURL = avatarURL.String
		}

		comment.Children = []models.Comment{}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return comments, rootOrder, nil
}

func buildCommentTree(comments []models.Comment, rootOrder []int) []models.Comment {
	baseComments := make(map[int]models.Comment, len(comments))
	childIDs := make(map[int][]int, len(comments))

	for _, comment := range comments {
		comment.Children = []models.Comment{}
		baseComments[comment.ID] = comment
	}

	if len(rootOrder) == 0 {
		for _, comment := range comments {
			if comment.ParentID == nil {
				rootOrder = append(rootOrder, comment.ID)
			}
		}
	}

	for _, comment := range comments {
		if comment.ParentID != nil {
			childIDs[*comment.ParentID] = append(childIDs[*comment.ParentID], comment.ID)
		}
	}

	nodes := make(map[int]*models.Comment, len(baseComments))
	var build func(int) *models.Comment
	build = func(id int) *models.Comment {
		if node, ok := nodes[id]; ok {
			return node
		}

		base, ok := baseComments[id]
		if !ok {
			return nil
		}

		node := base
		node.Children = make([]models.Comment, 0, len(childIDs[id]))
		nodes[id] = &node

		for _, childID := range childIDs[id] {
			child := build(childID)
			if child != nil {
				node.Children = append(node.Children, *child)
			}
		}

		nodes[id] = &node
		return &node
	}

	result := make([]models.Comment, 0, len(rootOrder))
	for _, rootID := range rootOrder {
		if root := build(rootID); root != nil {
			result = append(result, *root)
		}
	}

	return result
}
