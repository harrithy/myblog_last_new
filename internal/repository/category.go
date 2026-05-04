package repository

import (
	"database/sql"
	"encoding/json"
	"errors"
	"myblog_last_new/pkg/models"
)

// CategoryRepository handles category data access.
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository creates a new CategoryRepository.
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// CategoryFilter represents category query options.
type CategoryFilter struct {
	ParentID *int
	Type     string
	Keyword  string
	Page     int
	PageSize int
}

func parseTags(tagsJSON string) []string {
	if tagsJSON == "" {
		return nil
	}

	var tags []string
	if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
		return nil
	}

	return tags
}

func tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return ""
	}

	data, err := json.Marshal(tags)
	if err != nil {
		return ""
	}

	return string(data)
}

// HotTag represents a frequently used tag.
type HotTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetHotTags returns the most frequently used tags.
func (r *CategoryRepository) GetHotTags(limit int) ([]HotTag, error) {
	rows, err := r.db.Query("SELECT IFNULL(tags, '') FROM categories WHERE tags IS NOT NULL AND tags != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagCount := make(map[string]int)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			continue
		}

		for _, tag := range parseTags(tagsJSON) {
			tagCount[tag]++
		}
	}

	var hotTags []HotTag
	for name, count := range tagCount {
		hotTags = append(hotTags, HotTag{Name: name, Count: count})
	}

	for i := 0; i < len(hotTags)-1; i++ {
		for j := i + 1; j < len(hotTags); j++ {
			if hotTags[j].Count > hotTags[i].Count {
				hotTags[i], hotTags[j] = hotTags[j], hotTags[i]
			}
		}
	}

	if len(hotTags) > limit {
		hotTags = hotTags[:limit]
	}

	return hotTags, nil
}

// GetAll returns categories with optional filtering and pagination.
func (r *CategoryRepository) GetAll(filter CategoryFilter) ([]models.Category, int64, error) {
	whereClauses := "WHERE 1=1"
	var args []interface{}

	if filter.ParentID != nil {
		whereClauses += " AND parent_id = ?"
		args = append(args, *filter.ParentID)
	}

	if models.IsValidCategoryType(filter.Type) {
		whereClauses += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.Keyword != "" {
		whereClauses += " AND name LIKE ?"
		args = append(args, "%"+filter.Keyword+"%")
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM categories " + whereClauses
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at
		FROM categories ` + whereClauses + `
		ORDER BY sort_order ASC, id ASC`

	if filter.Page > 0 && filter.PageSize > 0 {
		offset := (filter.Page - 1) * filter.PageSize
		query += " LIMIT ? OFFSET ?"
		args = append(args, filter.PageSize, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	categories, err := scanCategories(rows)
	if err != nil {
		return nil, 0, err
	}

	return categories, total, nil
}

// GetByID returns a category by ID.
func (r *CategoryRepository) GetByID(id int) (*models.Category, error) {
	var category models.Category
	var parentID sql.NullInt64
	var tagsJSON string

	err := r.db.QueryRow(`
		SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at
		FROM categories
		WHERE id = ?
	`, id).Scan(
		&category.ID,
		&category.Name,
		&category.Type,
		&category.Description,
		&tagsJSON,
		&category.URL,
		&category.ImgURL,
		&parentID,
		&category.SortOrder,
		&category.CreatedAt,
		&category.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if parentID.Valid {
		pid := int(parentID.Int64)
		category.ParentID = &pid
	}
	category.Tags = parseTags(tagsJSON)

	return &category, nil
}

// GetSubtree returns a category and all nested descendants using a single query.
func (r *CategoryRepository) GetSubtree(id int) (*models.Category, error) {
	rows, err := r.db.Query(`
		SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at
		FROM categories
		ORDER BY sort_order ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories, err := scanCategories(rows)
	if err != nil {
		return nil, err
	}

	categoryMap, roots := buildCategoryTree(categories)
	category, ok := categoryMap[id]
	if !ok {
		return nil, sql.ErrNoRows
	}

	if category.ParentID == nil {
		for _, root := range roots {
			if root.ID == id {
				return &root, nil
			}
		}
	}

	return category, nil
}

// Exists checks whether a category exists.
func (r *CategoryRepository) Exists(id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

// Create creates a category.
func (r *CategoryRepository) Create(cat *models.Category) (int64, error) {
	var (
		result sql.Result
		err    error
	)

	tagsJSON := tagsToJSON(cat.Tags)

	if cat.ParentID != nil {
		result, err = r.db.Exec(
			"INSERT INTO categories (name, type, description, tags, url, img_url, parent_id, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			cat.Name, cat.Type, cat.Description, tagsJSON, cat.URL, cat.ImgURL, *cat.ParentID, cat.SortOrder,
		)
	} else {
		result, err = r.db.Exec(
			"INSERT INTO categories (name, type, description, tags, url, img_url, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)",
			cat.Name, cat.Type, cat.Description, tagsJSON, cat.URL, cat.ImgURL, cat.SortOrder,
		)
	}
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// Update updates a category.
func (r *CategoryRepository) Update(id int, cat *models.Category) (int64, error) {
	var (
		result sql.Result
		err    error
	)

	tagsJSON := tagsToJSON(cat.Tags)

	if cat.ParentID != nil {
		result, err = r.db.Exec(
			"UPDATE categories SET name = ?, type = ?, description = ?, tags = ?, url = ?, img_url = ?, parent_id = ?, sort_order = ? WHERE id = ?",
			cat.Name, cat.Type, cat.Description, tagsJSON, cat.URL, cat.ImgURL, *cat.ParentID, cat.SortOrder, id,
		)
	} else {
		result, err = r.db.Exec(
			"UPDATE categories SET name = ?, type = ?, description = ?, tags = ?, url = ?, img_url = ?, parent_id = NULL, sort_order = ? WHERE id = ?",
			cat.Name, cat.Type, cat.Description, tagsJSON, cat.URL, cat.ImgURL, cat.SortOrder, id,
		)
	}
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Delete deletes a category.
func (r *CategoryRepository) Delete(id int) (int64, error) {
	result, err := r.db.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetChildren returns all nested children for a category.
func (r *CategoryRepository) GetChildren(parentID int) ([]models.Category, error) {
	subtree, err := r.GetSubtree(parentID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []models.Category{}, nil
		}
		return nil, err
	}

	if subtree.Children == nil {
		return []models.Category{}, nil
	}

	return subtree.Children, nil
}

// BuildCategoryTree builds a category tree from a flat list.
func BuildCategoryTree(categories []models.Category) []models.Category {
	_, roots := buildCategoryTree(categories)
	return roots
}

func scanCategories(rows *sql.Rows) ([]models.Category, error) {
	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		var parentID sql.NullInt64
		var tagsJSON string
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Description, &tagsJSON, &cat.URL, &cat.ImgURL, &parentID, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			cat.ParentID = &pid
		}
		cat.Tags = parseTags(tagsJSON)
		cat.Children = []models.Category{}
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func buildCategoryTree(categories []models.Category) (map[int]*models.Category, []models.Category) {
	baseCategories := make(map[int]models.Category, len(categories))
	childIDs := make(map[int][]int, len(categories))
	rootIDs := make([]int, 0)

	for i := range categories {
		cat := categories[i]
		cat.Children = []models.Category{}
		baseCategories[cat.ID] = cat
	}

	for i := range categories {
		cat := categories[i]
		if cat.ParentID == nil {
			rootIDs = append(rootIDs, cat.ID)
			continue
		}

		if _, ok := baseCategories[*cat.ParentID]; !ok {
			rootIDs = append(rootIDs, cat.ID)
			continue
		}

		childIDs[*cat.ParentID] = append(childIDs[*cat.ParentID], cat.ID)
	}

	categoryMap := make(map[int]*models.Category, len(baseCategories))
	var build func(int) *models.Category
	build = func(id int) *models.Category {
		if category, ok := categoryMap[id]; ok {
			return category
		}

		base, ok := baseCategories[id]
		if !ok {
			return nil
		}

		node := base
		node.Children = make([]models.Category, 0, len(childIDs[id]))
		categoryMap[id] = &node

		for _, childID := range childIDs[id] {
			child := build(childID)
			if child != nil {
				node.Children = append(node.Children, *child)
			}
		}

		categoryMap[id] = &node
		return &node
	}

	for id := range baseCategories {
		build(id)
	}

	roots := make([]models.Category, 0, len(rootIDs))
	for _, rootID := range rootIDs {
		if root := build(rootID); root != nil {
			roots = append(roots, *root)
		}
	}

	return categoryMap, roots
}
