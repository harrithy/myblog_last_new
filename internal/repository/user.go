package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
)

// UserRepository 处理用户数据访问
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建新的 UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetAll 返回所有用户
func (r *UserRepository) GetAll() ([]models.User, error) {
	rows, err := r.db.Query("SELECT id, name, email FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Account); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetByEmail 根据邮箱返回用户
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow("SELECT id, name, email FROM users WHERE email = ?", email).
		Scan(&user.ID, &user.Name, &user.Account)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create 创建新用户
func (r *UserRepository) Create(u *models.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users(name, email, nickname, birthday) VALUES(?, ?, ?, ?)",
		u.Name, u.Account, u.Nickname, u.Birthday,
	)
	return err
}
