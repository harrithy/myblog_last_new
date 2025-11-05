package storage

import (
	"database/sql"
	"fmt"

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

	fmt.Println("数据库表初始化成功!")
	return nil
}
func ConnectDB() (*sql.DB, error) {
	dsn := "root:525300@tcp(localhost:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	fmt.Println("成功连接到 MySQL 数据库 'blog'!")
	return db, nil
}
