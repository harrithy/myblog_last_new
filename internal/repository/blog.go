package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
	"strings"
)

// BlogRepository 处理博客数据访问
type BlogRepository struct {
	db *sql.DB
}

// NewBlogRepository 创建新的 BlogRepository
func NewBlogRepository(db *sql.DB) *BlogRepository {
	return &BlogRepository{db: db}
}

// BlogFilter 表示博客查询的过滤选项
type BlogFilter struct {
	CategoryID *int
	Keyword    string
	Page       int
	PageSize   int
}

// GetAll 返回带过滤和分页的博客列表
func (r *BlogRepository) GetAll(filter BlogFilter) ([]models.Blog, int64, error) {
	queryArgs := make([]interface{}, 0)
	var whereClauses []string

	if filter.CategoryID != nil {
		whereClauses = append(whereClauses, "b.category_id = ?")
		queryArgs = append(queryArgs, *filter.CategoryID)
	}

	if filter.Keyword != "" {
		whereClauses = append(whereClauses, "b.title LIKE ?")
		queryArgs = append(queryArgs, "%"+filter.Keyword+"%")
	}

	whereStr := ""
	if len(whereClauses) > 0 {
		whereStr = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM blogs b " + whereStr
	if err := r.db.QueryRow(countQuery, queryArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get data
	offset := (filter.Page - 1) * filter.PageSize
	dataQuery := `
		SELECT b.id, b.title, b.url, b.category_id, IFNULL(c.name, ''), IFNULL(b.description, ''), b.created_at, b.updated_at 
		FROM blogs b 
		LEFT JOIN categories c ON b.category_id = c.id 
		` + whereStr + ` ORDER BY b.created_at DESC LIMIT ? OFFSET ?`

	dataArgs := append(queryArgs, filter.PageSize, offset)
	rows, err := r.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var blogs []models.Blog
	for rows.Next() {
		var blog models.Blog
		if err := rows.Scan(&blog.ID, &blog.Title, &blog.URL, &blog.CategoryID, &blog.CategoryName, &blog.Description, &blog.CreatedAt, &blog.UpdatedAt); err != nil {
			return nil, 0, err
		}
		blogs = append(blogs, blog)
	}

	return blogs, total, nil
}

// GetByID 根据 ID 返回博客
func (r *BlogRepository) GetByID(id int) (*models.Blog, error) {
	var blog models.Blog
	query := `
		SELECT b.id, b.title, b.url, b.category_id, IFNULL(c.name, ''), IFNULL(b.description, ''), b.created_at, b.updated_at 
		FROM blogs b 
		LEFT JOIN categories c ON b.category_id = c.id 
		WHERE b.id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&blog.ID, &blog.Title, &blog.URL, &blog.CategoryID,
		&blog.CategoryName, &blog.Description, &blog.CreatedAt, &blog.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &blog, nil
}
