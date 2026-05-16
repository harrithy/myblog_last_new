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
		SELECT id, name, COALESCE(email, ''), COALESCE(account, email), COALESCE(nickname, ''), COALESCE(DATE_FORMAT(birthday, '%Y-%m-%d'), ''), COALESCE(github_id, 0), COALESCE(avatar_url, ''), COALESCE(github_url, ''), COALESCE(bio, '')
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
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Account, &u.Nickname, &u.Birthday, &u.GitHubID, &u.AvatarURL, &u.GitHubURL, &u.Bio); err != nil {
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
		SELECT id, name, COALESCE(email, ''), COALESCE(account, email), COALESCE(nickname, ''), COALESCE(DATE_FORMAT(birthday, '%Y-%m-%d'), ''), COALESCE(password, ''), COALESCE(github_id, 0), COALESCE(avatar_url, ''), COALESCE(github_url, ''), COALESCE(bio, '')
		FROM users
		WHERE email = ? OR account = ?
		LIMIT 1
	`, login, login).
		Scan(&user.ID, &user.Name, &user.Email, &user.Account, &user.Nickname, &user.Birthday, &user.Password, &user.GitHubID, &user.AvatarURL, &user.GitHubURL, &user.Bio)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByID returns a user by numeric ID.
func (r *UserRepository) GetByID(id int) (*models.User, error) {
	var user models.User
	err := r.db.QueryRow(`
		SELECT id, name, COALESCE(email, ''), COALESCE(account, email), COALESCE(nickname, ''), COALESCE(DATE_FORMAT(birthday, '%Y-%m-%d'), ''), COALESCE(github_id, 0), COALESCE(avatar_url, ''), COALESCE(github_url, ''), COALESCE(bio, '')
		FROM users
		WHERE id = ?
		LIMIT 1
	`, id).Scan(&user.ID, &user.Name, &user.Email, &user.Account, &user.Nickname, &user.Birthday, &user.GitHubID, &user.AvatarURL, &user.GitHubURL, &user.Bio)
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
	email := strings.ToLower(strings.TrimSpace(u.Email))
	account := strings.TrimSpace(u.Account)
	if email == "" && strings.Contains(account, "@") {
		email = strings.ToLower(account)
	}

	name := strings.TrimSpace(u.Name)
	if name == "" {
		name = account
	}

	nickname := strings.TrimSpace(u.Nickname)
	if nickname == "" {
		nickname = name
	}

	var birthday interface{}
	if strings.TrimSpace(u.Birthday) != "" {
		birthday = u.Birthday
	}

	_, err := r.db.Exec(
		"INSERT INTO users(name, email, account, password, nickname, birthday) VALUES(?, ?, ?, ?, ?, ?)",
		name, email, account, u.Password, nickname, birthday,
	)
	return err
}

// ExistsByEmail returns true when a user already exists with the given email.
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false, nil
	}

	return r.existsByField("email", email)
}

// ExistsByAccount returns true when a user already exists with the given account name.
func (r *UserRepository) ExistsByAccount(account string) (bool, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return false, nil
	}

	return r.existsByField("account", account)
}

// ExistsByEmailExcludingID returns true when another user already uses the email.
func (r *UserRepository) ExistsByEmailExcludingID(email string, excludeID int) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false, nil
	}

	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ? AND id <> ?", email, excludeID).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *UserRepository) existsByField(field, value string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE " + field + " = ?"
	if err := r.db.QueryRow(query, value).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetByGitHubID 根据 GitHub ID 查找用户
func (r *UserRepository) GetByGitHubID(githubID int64) (*models.User, error) {
	var user models.User
	var avatarURL, githubURL sql.NullString
err := r.db.QueryRow(
		"SELECT id, name, COALESCE(email, ''), COALESCE(account, email), COALESCE(nickname, ''), COALESCE(birthday, ''), COALESCE(github_id, 0), COALESCE(avatar_url, ''), COALESCE(github_url, ''), COALESCE(bio, '') FROM users WHERE github_id = ?",
		githubID,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Account, &user.Nickname, &user.Birthday, &user.GitHubID, &avatarURL, &githubURL, &user.Bio)
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
		Email:     email,
		Account:   email,
		Nickname:  githubUser.Login,
		GitHubID:  githubUser.ID,
		AvatarURL: githubUser.AvatarURL,
		GitHubURL: githubUser.HTMLURL,
	}, nil
}

// UpdateByID updates editable profile fields for a user.
func (r *UserRepository) UpdateByID(u *models.User) (int64, error) {
	result, err := r.db.Exec(
		"UPDATE users SET name = ?, email = ?, nickname = ?, birthday = ?, avatar_url = ?, bio = ? WHERE id = ?",
		u.Name, u.Email, u.Nickname, nullableString(u.Birthday), nullableString(u.AvatarURL), nullableString(u.Bio), u.ID,
	)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// DeleteByID deletes a user by numeric ID.
func (r *UserRepository) DeleteByID(id int) (int64, error) {
	result, err := r.db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func nullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}
