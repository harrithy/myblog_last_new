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

// GetByGitHubID 根据 GitHub ID 查找用户
func (r *UserRepository) GetByGitHubID(githubID int64) (*models.User, error) {
	var user models.User
	var avatarURL, githubURL sql.NullString
	err := r.db.QueryRow(
		"SELECT id, name, email, nickname, COALESCE(birthday, ''), COALESCE(github_id, 0), COALESCE(avatar_url, ''), COALESCE(github_url, '') FROM users WHERE github_id = ?",
		githubID,
	).Scan(&user.ID, &user.Name, &user.Account, &user.Nickname, &user.Birthday, &user.GitHubID, &avatarURL, &githubURL)
	if err != nil {
		return nil, err
	}
	user.AvatarURL = avatarURL.String
	user.GitHubURL = githubURL.String
	return &user, nil
}

// FindOrCreateByGitHub 根据 GitHub 用户信息查找或创建用户
func (r *UserRepository) FindOrCreateByGitHub(githubUser *models.GitHubUser) (*models.User, error) {
	// 先尝试根据 GitHub ID 查找用户
	user, err := r.GetByGitHubID(githubUser.ID)
	if err == nil {
		// 用户已存在，更新信息
		_, updateErr := r.db.Exec(
			"UPDATE users SET name = ?, avatar_url = ?, github_url = ? WHERE github_id = ?",
			githubUser.Name, githubUser.AvatarURL, githubUser.HTMLURL, githubUser.ID,
		)
		if updateErr != nil {
			return nil, updateErr
		}
		user.Name = githubUser.Name
		user.AvatarURL = githubUser.AvatarURL
		user.GitHubURL = githubUser.HTMLURL
		return user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// 用户不存在，创建新用户
	email := githubUser.Email
	if email == "" {
		email = githubUser.Login + "@github.com" // 如果没有公开邮箱，使用 GitHub 用户名生成
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	result, err := r.db.Exec(
		"INSERT INTO users(name, email, nickname, github_id, avatar_url, github_url) VALUES(?, ?, ?, ?, ?, ?)",
		name, email, githubUser.Login, githubUser.ID, githubUser.AvatarURL, githubUser.HTMLURL,
	)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &models.User{
		ID:        int(id),
		Name:      name,
		Account:   email,
		Nickname:  githubUser.Login,
		GitHubID:  githubUser.ID,
		AvatarURL: githubUser.AvatarURL,
		GitHubURL: githubUser.HTMLURL,
	}, nil
}
