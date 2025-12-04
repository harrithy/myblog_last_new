package repository

import (
	"database/sql"
	"fmt"
	"myblog_last_new/pkg/models"
	"time"
)

// VisitRepository 处理访问日志数据访问
type VisitRepository struct {
	db *sql.DB
}

// NewVisitRepository 创建新的 VisitRepository
func NewVisitRepository(db *sql.DB) *VisitRepository {
	return &VisitRepository{db: db}
}

// Create 创建新的访问日志
func (r *VisitRepository) Create(visit *models.VisitLog) (int64, error) {
	mysqlTimeFormat := visit.VisitTime.Format("2006-01-02 15:04:05")

	result, err := r.db.Exec(
		"INSERT INTO visit_logs (user_nickname, visit_time, content) VALUES (?, ?, ?)",
		visit.UserNickname, mysqlTimeFormat, visit.Content,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetAll 返回带分页的访问日志
func (r *VisitRepository) GetAll(page, pageSize int) ([]models.VisitLog, int64, error) {
	var total int64
	if err := r.db.QueryRow("SELECT COUNT(*) FROM visit_logs").Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := "SELECT id, user_nickname, visit_time, IFNULL(content, '普通访问记录'), created_at FROM visit_logs ORDER BY visit_time DESC, id DESC LIMIT ? OFFSET ?"

	rows, err := r.db.Query(query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var visits []models.VisitLog
	for rows.Next() {
		var v models.VisitLog
		if err := rows.Scan(&v.ID, &v.UserNickname, &v.VisitTime, &v.Content, &v.CreatedAt); err != nil {
			return nil, 0, err
		}
		visits = append(visits, v)
	}

	return visits, total, nil
}

// CleanupOld 清理旧的访问日志，只保留指定数量
func (r *VisitRepository) CleanupOld(keepCount int) {
	var totalCount int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM visit_logs").Scan(&totalCount); err != nil {
		fmt.Printf("Failed to count visit logs: %v\n", err)
		return
	}

	if totalCount > keepCount {
		deleteQuery := `
			DELETE FROM visit_logs 
			WHERE id < (
				SELECT min_id FROM (
					SELECT MIN(id) as min_id FROM (
						SELECT id FROM visit_logs 
						ORDER BY created_at DESC 
						LIMIT ?
					) as keep_records
				) as temp
			)
		`

		result, err := r.db.Exec(deleteQuery, keepCount)
		if err != nil {
			fmt.Printf("Failed to delete old visit logs: %v\n", err)
			return
		}

		affected, _ := result.RowsAffected()
		fmt.Printf("Cleaned up %d old visit log records, keeping latest %d\n", affected, keepCount)
	}
}

// GuestRepository 处理访客记录数据访问
type GuestRepository struct {
	db *sql.DB
}

// NewGuestRepository 创建新的 GuestRepository
func NewGuestRepository(db *sql.DB) *GuestRepository {
	return &GuestRepository{db: db}
}

// Create 创建新的访客记录
func (r *GuestRepository) Create(record *models.GuestRecord) (int64, error) {
	mysqlTimeFormat := record.EntryTime.Format("2006-01-02 15:04:05")

	result, err := r.db.Exec(
		"INSERT INTO guest_records (entry_time, content) VALUES (?, ?)",
		mysqlTimeFormat, record.Content,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// OwnerVisitRepository 处理博主访问日志数据访问
type OwnerVisitRepository struct {
	db *sql.DB
}

// NewOwnerVisitRepository 创建新的 OwnerVisitRepository
func NewOwnerVisitRepository(db *sql.DB) *OwnerVisitRepository {
	return &OwnerVisitRepository{db: db}
}

// RecordVisit 记录博主访问
func (r *OwnerVisitRepository) RecordVisit() error {
	today := time.Now().Format("2006-01-02")

	query := `
		INSERT INTO owner_visit_logs (visit_date, visit_count) 
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE 
		visit_count = visit_count + 1,
		last_visit_time = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query, today)
	return err
}

// GetStats 返回指定天数内的博主访问统计
func (r *OwnerVisitRepository) GetStats(days string) ([]models.OwnerVisitLog, int, error) {
	query := `
		SELECT visit_date, visit_count, last_visit_time 
		FROM owner_visit_logs 
		WHERE visit_date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		ORDER BY visit_date DESC
	`

	rows, err := r.db.Query(query, days)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var visitStats []models.OwnerVisitLog
	for rows.Next() {
		var stat models.OwnerVisitLog
		if err := rows.Scan(&stat.VisitDate, &stat.VisitCount, &stat.LastVisitTime); err != nil {
			return nil, 0, err
		}
		visitStats = append(visitStats, stat)
	}

	// Get total
	totalQuery := `
		SELECT COALESCE(SUM(visit_count), 0) 
		FROM owner_visit_logs 
		WHERE visit_date >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
	`

	var totalVisits int
	if err := r.db.QueryRow(totalQuery, days).Scan(&totalVisits); err != nil {
		return nil, 0, err
	}

	return visitStats, totalVisits, nil
}

// GetTodayVisits 返回今日访问次数
func (r *OwnerVisitRepository) GetTodayVisits() (int, error) {
	today := time.Now().Format("2006-01-02")

	var todayVisits int
	err := r.db.QueryRow(`
		SELECT COALESCE(visit_count, 0) 
		FROM owner_visit_logs 
		WHERE visit_date = ?
	`, today).Scan(&todayVisits)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return todayVisits, err
}
