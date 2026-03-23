package repository

import (
	"database/sql"
	"myblog_last_new/pkg/models"
	"strings"
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
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(account, email), COALESCE(nickname, ''), COALESCE(DATE_FORMAT(birthday, '%Y-%m-%d'), '')
		FROM users
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Account, &u.Nickname, &u.Birthday); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetByLogin returns a user by email or account.
func (r *UserRepository) GetByLogin(login string) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, name, COALESCE(account, email), COALESCE(nickname, ''), COALESCE(DATE_FORMAT(birthday, '%Y-%m-%d'), ''), COALESCE(password, '')
		FROM users
		WHERE email = ? OR account = ?
		LIMIT 1
	`, login, login).
		Scan(&user.ID, &user.Name, &user.Account, &user.Nickname, &user.Birthday, &user.Password)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail keeps compatibility with older call sites.
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	return r.GetByLogin(email)
}

// Create 创建新用户
func (r *UserRepository) Create(u *models.User) error {
	account := strings.TrimSpace(u.Account)
	var birthday interface{}
	if strings.TrimSpace(u.Birthday) != "" {
		birthday = u.Birthday
	}

	_, err := r.db.Exec(
		"INSERT INTO users(name, email, account, password, nickname, birthday) VALUES(?, ?, ?, ?, ?, ?)",
		u.Name, account, account, u.Password, u.Nickname, birthday,
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
		"INSERT INTO users(name, email, account, nickname, github_id, avatar_url, github_url) VALUES(?, ?, ?, ?, ?, ?, ?)",
		name, email, email, githubUser.Login, githubUser.ID, githubUser.AvatarURL, githubUser.HTMLURL,
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
