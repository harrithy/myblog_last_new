package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
	"strings"
)

// BlogRepository handles read access for article-like blog data.
// The current source of truth is categories(type='article').
type BlogRepository struct {
	db *sql.DB
}

// NewBlogRepository creates a new BlogRepository.
func NewBlogRepository(db *sql.DB) *BlogRepository {
	return &BlogRepository{db: db}
}

// BlogFilter represents blog query filters.
type BlogFilter struct {
	CategoryID *int
	Keyword    string
	Page       int
	PageSize   int
}

// GetAll returns paginated article nodes from categories while preserving the blog response shape.
func (r *BlogRepository) GetAll(filter BlogFilter) ([]models.Blog, int64, error) {
	queryArgs := make([]interface{}, 0)
	whereClauses := []string{"c.type = 'article'"}

	if filter.CategoryID != nil {
		whereClauses = append(whereClauses, "c.parent_id = ?")
		queryArgs = append(queryArgs, *filter.CategoryID)
	}

	if filter.Keyword != "" {
		whereClauses = append(whereClauses, "c.name LIKE ?")
		queryArgs = append(queryArgs, "%"+filter.Keyword+"%")
	}

	whereStr := "WHERE " + strings.Join(whereClauses, " AND ")

	var total int64
	countQuery := "SELECT COUNT(*) FROM categories c " + whereStr
	if err := r.db.QueryRow(countQuery, queryArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := `
		SELECT
			c.id,
			c.name,
			IFNULL(c.url, ''),
			COALESCE(c.parent_id, 0),
			IFNULL(parent.name, ''),
			IFNULL(c.description, ''),
			c.created_at,
			c.updated_at
		FROM categories c
		LEFT JOIN categories parent ON c.parent_id = parent.id
		` + whereStr + `
		ORDER BY c.created_at DESC
		LIMIT ? OFFSET ?`

	dataArgs := append(queryArgs, filter.PageSize, offset)
	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blogs []models.Blog
	for rows.Next() {
		var blog models.Blog
		if err := rows.Scan(
			&blog.ID,
			&blog.Title,
			&blog.URL,
			&blog.CategoryID,
			&blog.CategoryName,
			&blog.Description,
			&blog.CreatedAt,
			&blog.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		blogs = append(blogs, blog)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return blogs, total, nil
}

// GetByID returns an article node by category ID while preserving the blog response shape.
func (r *BlogRepository) GetByID(id int) (*models.Blog, error) {
	var blog models.Blog

	query := `
		SELECT
			c.id,
			c.name,
			IFNULL(c.url, ''),
			COALESCE(c.parent_id, 0),
			IFNULL(parent.name, ''),
			IFNULL(c.description, ''),
			c.created_at,
			c.updated_at
		FROM categories c
		LEFT JOIN categories parent ON c.parent_id = parent.id
		WHERE c.id = ? AND c.type = 'article'`

	err := r.db.QueryRow(query, id).Scan(
		&blog.ID,
		&blog.Title,
		&blog.URL,
		&blog.CategoryID,
		&blog.CategoryName,
		&blog.Description,
		&blog.CreatedAt,
		&blog.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &blog, nil
}
