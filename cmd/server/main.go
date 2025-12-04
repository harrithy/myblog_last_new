package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "myblog_last_new/docs"
	"myblog_last_new/internal/router"
	"myblog_last_new/pkg/storage"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title My Blog API
// @version 1.0
// @description This is a sample server for a blog application.
// @host localhost:8080
// @BasePath /
func main() {
	// Initialize database connection
	db, err := storage.ConnectDB()
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	defer db.Close()

	// Initialize database tables
	if err := storage.InitDB(db); err != nil {
		log.Fatalf("数据库表初始化失败: %v", err)
	}

	// Setup router
	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	// Swagger UI
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// Static file server
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start server
	fmt.Printf("服务器正在端口 %s 启动...\n", port)
	fmt.Printf("API 文档地址: http://localhost:%s/swagger/index.html\n", port)
	fmt.Printf("分类管理页面: http://localhost:%s/static/category.html\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("无法启动服务器: %v", err)
	}
}
