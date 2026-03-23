package storage

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// InitDB initializes required tables and ensures legacy columns exist.
func InitDB(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			nickname VARCHAR(255),
			birthday DATE,
			github_id BIGINT DEFAULT NULL,
			avatar_url VARCHAR(500) DEFAULT NULL,
			github_url VARCHAR(500) DEFAULT NULL,
			account VARCHAR(255) DEFAULT NULL,
			password VARCHAR(255) DEFAULT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY unique_github_id (github_id),
			UNIQUE KEY unique_account (account)
		);
	`); err != nil {
		return err
	}

	if err := ensureColumnExists(db, "users", "github_id", "ALTER TABLE users ADD COLUMN github_id BIGINT DEFAULT NULL, ADD UNIQUE KEY unique_github_id (github_id)"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "users", "avatar_url", "ALTER TABLE users ADD COLUMN avatar_url VARCHAR(500) DEFAULT NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "users", "github_url", "ALTER TABLE users ADD COLUMN github_url VARCHAR(500) DEFAULT NULL"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "users", "account", "ALTER TABLE users ADD COLUMN account VARCHAR(255) DEFAULT NULL, ADD UNIQUE KEY unique_account (account)"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "users", "password", "ALTER TABLE users ADD COLUMN password VARCHAR(255) DEFAULT NULL"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS visit_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_nickname VARCHAR(255) NOT NULL,
			visit_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			content TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "visit_logs", "content", "ALTER TABLE visit_logs ADD COLUMN content TEXT AFTER visit_time"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS guest_records (
			id INT AUTO_INCREMENT PRIMARY KEY,
			entry_time DATETIME NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS owner_visit_logs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			visit_date DATE NOT NULL,
			visit_count INT DEFAULT 1,
			last_visit_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY unique_date (visit_date)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(20) NOT NULL DEFAULT 'folder',
			description TEXT DEFAULT NULL,
			tags TEXT DEFAULT NULL,
			url VARCHAR(500) DEFAULT NULL,
			img_url VARCHAR(500) DEFAULT NULL,
			parent_id INT DEFAULT NULL,
			sort_order INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE,
			INDEX idx_parent_id (parent_id),
			INDEX idx_sort_order (sort_order),
			INDEX idx_type (type)
		);
	`); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "categories", "type", "ALTER TABLE categories ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'folder' AFTER name"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "categories", "description", "ALTER TABLE categories ADD COLUMN description TEXT DEFAULT NULL AFTER type"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "categories", "tags", "ALTER TABLE categories ADD COLUMN tags TEXT DEFAULT NULL AFTER description"); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "categories", "img_url", "ALTER TABLE categories ADD COLUMN img_url VARCHAR(500) DEFAULT NULL AFTER url"); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS blogs (
			id INT AUTO_INCREMENT PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			url VARCHAR(500) NOT NULL UNIQUE,
			category_id INT NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
			INDEX idx_category_id (category_id)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id INT AUTO_INCREMENT PRIMARY KEY,
			article_id INT NOT NULL,
			parent_id INT DEFAULT NULL,
			nickname VARCHAR(255) NOT NULL,
			email VARCHAR(255) DEFAULT NULL,
			avatar_url VARCHAR(500) DEFAULT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (article_id) REFERENCES categories(id) ON DELETE CASCADE,
			FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE,
			INDEX idx_article_id (article_id),
			INDEX idx_parent_id (parent_id)
		);
	`); err != nil {
		return err
	}
	if err := ensureColumnExists(db, "comments", "avatar_url", "ALTER TABLE comments ADD COLUMN avatar_url VARCHAR(500) DEFAULT NULL AFTER email"); err != nil {
		return err
	}

	fmt.Println("数据库表初始化成功!")
	return nil
}

func ensureColumnExists(db *sql.DB, tableName, columnName, alterSQL string) error {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?
		  AND COLUMN_NAME = ?
	`, tableName, columnName).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check %s.%s column: %w", tableName, columnName, err)
	}

	if count > 0 {
		return nil
	}

	if _, err := db.Exec(alterSQL); err != nil {
		return fmt.Errorf("failed to add %s.%s column: %w", tableName, columnName, err)
	}

	return nil
}

// ConnectDB opens and verifies the database connection.
func ConnectDB() (*sql.DB, error) {
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "525300")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbName := getEnv("DB_NAME", "blog")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(30 * time.Minute)

	fmt.Printf("成功连接到 MySQL 数据库 '%s'!\n", dbName)
	return db, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
