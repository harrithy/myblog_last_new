package app

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	_ "myblog_last_new/docs"
	"myblog_last_new/internal/config"
	"myblog_last_new/internal/router"
	"myblog_last_new/pkg/storage"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Run starts the HTTP server.
func Run() error {
	db, err := storage.ConnectDB()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if strings.EqualFold(config.GetEnv("AUTO_MIGRATE", "true"), "true") {
		if err := storage.InitDB(db); err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
	}

	mux := http.NewServeMux()
	router.RegisterRoutes(mux, db)

	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	port := config.GetEnv("PORT", "8080")

	fmt.Printf("Server starting on port %s...\n", port)
	fmt.Printf("Swagger UI: http://localhost:%s/swagger/index.html\n", port)
	fmt.Printf("Category admin page: http://localhost:%s/static/category.html\n", port)
	fmt.Printf("Static API page: http://localhost:%s/static/api.html\n", port)
	fmt.Printf("WebSocket endpoint: ws://localhost:%s/ws\n", port)
	fmt.Printf("WebSocket online count: http://localhost:%s/ws/online\n", port)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}
