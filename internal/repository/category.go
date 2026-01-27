package repository

import (
	"database/sql"
	"encoding/json"
	"myblog_last_new/pkg/models"
)

// CategoryRepository 处理分类数据访问
type CategoryRepository struct {
	db *sql.DB
}

// NewCategoryRepository 创建新的 CategoryRepository
func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// CategoryFilter 表示分类查询的过滤选项
type CategoryFilter struct {
	ParentID *int
	Type     string
	Keyword  string // 标题模糊搜索关键词
}

// parseTags 解析 JSON 格式的标签
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

// tagsToJSON 将标签转换为 JSON 格式
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

// HotTag 表示热门标签
type HotTag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// GetHotTags 获取热门标签（使用次数前 N 个）
func (r *CategoryRepository) GetHotTags(limit int) ([]HotTag, error) {
	// 查询所有分类的标签
	rows, err := r.db.Query("SELECT IFNULL(tags, '') FROM categories WHERE tags IS NOT NULL AND tags != ''")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 统计标签使用次数
	tagCount := make(map[string]int)
	for rows.Next() {
		var tagsJSON string
		if err := rows.Scan(&tagsJSON); err != nil {
			continue
		}
		tags := parseTags(tagsJSON)
		for _, tag := range tags {
			tagCount[tag]++
		}
	}

	// 转换为切片并排序
	var hotTags []HotTag
	for name, count := range tagCount {
		hotTags = append(hotTags, HotTag{Name: name, Count: count})
	}

	// 按使用次数降序排序
	for i := 0; i < len(hotTags)-1; i++ {
		for j := i + 1; j < len(hotTags); j++ {
			if hotTags[j].Count > hotTags[i].Count {
				hotTags[i], hotTags[j] = hotTags[j], hotTags[i]
			}
		}
	}

	// 取前 limit 个
	if len(hotTags) > limit {
		hotTags = hotTags[:limit]
	}

	return hotTags, nil
}

// GetAll 返回所有分类，支持可选过滤
func (r *CategoryRepository) GetAll(filter CategoryFilter) ([]models.Category, error) {
	query := "SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at FROM categories WHERE 1=1"
	var args []interface{}

	if filter.ParentID != nil {
		query += " AND parent_id = ?"
		args = append(args, *filter.ParentID)
	}

	if filter.Type != "" && (filter.Type == "folder" || filter.Type == "article") {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}

	if filter.Keyword != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+filter.Keyword+"%")
	}

	query += " ORDER BY sort_order ASC, id ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
		categories = append(categories, cat)
	}

	return categories, nil
}

// GetByID 根据 ID 返回分类
func (r *CategoryRepository) GetByID(id int) (*models.Category, error) {
	var category models.Category
	var parentID sql.NullInt64
	var tagsJSON string

	err := r.db.QueryRow(`
		SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at 
		FROM categories WHERE id = ?
	`, id).Scan(&category.ID, &category.Name, &category.Type, &category.Description, &tagsJSON, &category.URL, &category.ImgURL, &parentID, &category.SortOrder, &category.CreatedAt, &category.UpdatedAt)

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

// Exists 检查分类是否存在
func (r *CategoryRepository) Exists(id int) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM categories WHERE id = ?)", id).Scan(&exists)
	return exists, err
}

// Create 创建新分类
func (r *CategoryRepository) Create(cat *models.Category) (int64, error) {
	var result sql.Result
	var err error

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

// Update 更新分类
func (r *CategoryRepository) Update(id int, cat *models.Category) (int64, error) {
	var result sql.Result
	var err error

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

// Delete 删除分类
func (r *CategoryRepository) Delete(id int) (int64, error) {
	result, err := r.db.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetChildren 返回父分类的子分类
func (r *CategoryRepository) GetChildren(parentID int) ([]models.Category, error) {
	rows, err := r.db.Query(`
		SELECT id, name, type, IFNULL(description, ''), IFNULL(tags, ''), IFNULL(url, ''), IFNULL(img_url, ''), parent_id, sort_order, created_at, updated_at 
		FROM categories 
		WHERE parent_id = ?
		ORDER BY sort_order ASC, id ASC
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var children []models.Category
	for rows.Next() {
		var cat models.Category
		var pid sql.NullInt64
		var tagsJSON string
		if err := rows.Scan(&cat.ID, &cat.Name, &cat.Type, &cat.Description, &tagsJSON, &cat.URL, &cat.ImgURL, &pid, &cat.SortOrder, &cat.CreatedAt, &cat.UpdatedAt); err != nil {
			return nil, err
		}
		if pid.Valid {
			p := int(pid.Int64)
			cat.ParentID = &p
		}
		cat.Tags = parseTags(tagsJSON)
		// Recursively get children
		subChildren, _ := r.GetChildren(cat.ID)
		if len(subChildren) > 0 {
			cat.Children = subChildren
		}
		children = append(children, cat)
	}

	return children, nil
}

// BuildCategoryTree 将扁平分类列表构建成树形结构
func BuildCategoryTree(categories []models.Category) []models.Category {
	categoryMap := make(map[int]*models.Category)
	var roots []models.Category

	for i := range categories {
		categories[i].Children = []models.Category{}
		categoryMap[categories[i].ID] = &categories[i]
	}

	for i := range categories {
		cat := &categories[i]
		if cat.ParentID == nil {
			roots = append(roots, *cat)
		} else {
			if parent, ok := categoryMap[*cat.ParentID]; ok {
				parent.Children = append(parent.Children, *cat)
			}
		}
	}

	for i := range roots {
		if mapped, ok := categoryMap[roots[i].ID]; ok {
			roots[i].Children = mapped.Children
		}
	}

	return roots
}
