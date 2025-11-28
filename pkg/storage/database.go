package storage

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// InitDB initializes the database tables
func InitDB(db *sql.DB) error {
	// Create users table if it does not exist
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE,
		nickname VARCHAR(255),
		birthday DATE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	// Create visit_logs table if it does not exist
	visitQuery := `
	CREATE TABLE IF NOT EXISTS visit_logs (
		id INT AUTO_INCREMENT PRIMARY KEY,
		user_nickname VARCHAR(255) NOT NULL,
		visit_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		content TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(visitQuery)
	if err != nil {
		return err
	}

	// 为现有的visit_logs表添加content字段（如果不存在）
	// 先检查字段是否已存在
	var columnExists bool
	checkQuery := `
	SELECT COUNT(*) 
	FROM INFORMATION_SCHEMA.COLUMNS 
	WHERE TABLE_SCHEMA = 'blog' 
	AND TABLE_NAME = 'visit_logs' 
	AND COLUMN_NAME = 'content';`

	err = db.QueryRow(checkQuery).Scan(&columnExists)
	if err != nil {
		fmt.Printf("Warning: Failed to check column: %v\n", err)
	} else if !columnExists {
		// 字段不存在，添加字段
		alterQuery := "ALTER TABLE visit_logs ADD COLUMN content TEXT AFTER visit_time;"
		_, err = db.Exec(alterQuery)
		if err != nil {
			fmt.Printf("Warning: Failed to add column: %v\n", err)
		} else {
			fmt.Println("成功添加 content 字段到 visit_logs 表")
		}
	}

	// Create guest_records table if it does not exist
	guestQuery := `
	CREATE TABLE IF NOT EXISTS guest_records (
		id INT AUTO_INCREMENT PRIMARY KEY,
		entry_time DATETIME NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(guestQuery)
	if err != nil {
		return err
	}

	// Create owner_visit_logs table if it does not exist
	ownerVisitQuery := `
	CREATE TABLE IF NOT EXISTS owner_visit_logs (
		id INT AUTO_INCREMENT PRIMARY KEY,
		visit_date DATE NOT NULL,
		visit_count INT DEFAULT 1,
		last_visit_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY unique_date (visit_date)
	);`

	_, err = db.Exec(ownerVisitQuery)
	if err != nil {
		return err
	}

	// Create categories table first (blogs depends on it)
	categoryQuery := `
	CREATE TABLE IF NOT EXISTS categories (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		type VARCHAR(20) NOT NULL DEFAULT 'folder',
		url VARCHAR(500) DEFAULT NULL,
		parent_id INT DEFAULT NULL,
		sort_order INT DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE CASCADE,
		INDEX idx_parent_id (parent_id),
		INDEX idx_sort_order (sort_order),
		INDEX idx_type (type)
	);`

	_, err = db.Exec(categoryQuery)
	if err != nil {
		return err
	}

	// Create blogs table if it does not exist
	blogQuery := `
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
	);`

	_, err = db.Exec(blogQuery)
	if err != nil {
		return err
	}

	// 为现有的categories表添加type字段（如果不存在）
	var typeColumnExists bool
	checkTypeQuery := `
	SELECT COUNT(*) 
	FROM INFORMATION_SCHEMA.COLUMNS 
	WHERE TABLE_SCHEMA = 'blog' 
	AND TABLE_NAME = 'categories' 
	AND COLUMN_NAME = 'type';`

	err = db.QueryRow(checkTypeQuery).Scan(&typeColumnExists)
	if err != nil {
		fmt.Printf("Warning: Failed to check type column: %v\n", err)
	} else if !typeColumnExists {
		alterQuery := "ALTER TABLE categories ADD COLUMN type VARCHAR(20) NOT NULL DEFAULT 'folder' AFTER name;"
		_, err = db.Exec(alterQuery)
		if err != nil {
			fmt.Printf("Warning: Failed to add type column: %v\n", err)
		} else {
			fmt.Println("成功添加 type 字段到 categories 表")
		}
	}

	// Create comments table if it does not exist
	commentQuery := `
	CREATE TABLE IF NOT EXISTS comments (
		id INT AUTO_INCREMENT PRIMARY KEY,
		article_id INT NOT NULL,
		parent_id INT DEFAULT NULL,
		nickname VARCHAR(255) NOT NULL,
		email VARCHAR(255) DEFAULT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (article_id) REFERENCES categories(id) ON DELETE CASCADE,
		FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE,
		INDEX idx_article_id (article_id),
		INDEX idx_parent_id (parent_id)
	);`

	_, err = db.Exec(commentQuery)
	if err != nil {
		return err
	}

	fmt.Println("数据库表初始化成功!")
	return nil
}
func ConnectDB() (*sql.DB, error) {
	// 从环境变量读取数据库配置，如果没有则使用默认值
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
	fmt.Printf("成功连接到 MySQL 数据库 '%s'!\n", dbName)
	return db, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
